package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"img_process/cons"
	"img_process/middleware"
	"img_process/model"
	"img_process/tools"

	mapset "github.com/deckarep/golang-set"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

type actionResultRecorder struct {
	noopScanRecorder
	results []recordedActionResult
}

type artifactRecorder struct {
	noopScanRecorder
	artifacts []recordedArtifact
	errors    []recordedError
}

type recordedActionResult struct {
	id      uint
	success bool
	payload map[string]any
}

type recordedArtifact struct {
	title       string
	relatedPath string
	payload     map[string]any
}

type recordedError struct {
	phase       string
	relatedPath string
	err         error
	payload     map[string]any
}

func (r *actionResultRecorder) RecordActionResult(id uint, success bool, err error, payload map[string]any) {
	copied := map[string]any{}
	for key, value := range payload {
		copied[key] = value
	}
	r.results = append(r.results, recordedActionResult{
		id:      id,
		success: success,
		payload: copied,
	})
}

func (r *artifactRecorder) RecordArtifact(title string, relatedPath string, payload map[string]any) {
	copied := map[string]any{}
	for key, value := range payload {
		copied[key] = value
	}
	r.artifacts = append(r.artifacts, recordedArtifact{
		title:       title,
		relatedPath: relatedPath,
		payload:     copied,
	})
}

func (r *artifactRecorder) RecordError(phase string, relatedPath string, err error, payload map[string]any) {
	copied := map[string]any{}
	for key, value := range payload {
		copied[key] = value
	}
	r.errors = append(r.errors, recordedError{
		phase:       phase,
		relatedPath: relatedPath,
		err:         err,
		payload:     copied,
	})
}

func ensureTestLogger() {
	if tools.Logger == nil {
		tools.Logger = zap.NewNop().Sugar()
	}
}

func TestWriteBackupDiffArtifactsStoresFullDetailsAndReturnsSmallSummary(t *testing.T) {
	ensureTestLogger()
	oldWorkDir := cons.WorkDir
	cons.WorkDir = t.TempDir()
	t.Cleanup(func() {
		cons.WorkDir = oldWorkDir
	})

	recorder := &artifactRecorder{}
	scanner := newScannerWithRecorder(model.DoScanImgArg{}, recorder)
	scanner.scanUUID = "2026-05-20-12-00-00_testscan"

	bakNewFile := make([]string, 0, backupDiffSummarySampleLimit+5)
	for i := backupDiffSummarySampleLimit + 4; i >= 0; i-- {
		bakNewFile = append(bakNewFile, filepath.Join("new", "file_"+string(rune('A'+i))))
	}
	bakDeleteFile := []string{"delete/file_C", "delete/file_A", "delete/file_B"}
	sortStringsForTest(bakNewFile)
	sortStringsForTest(bakDeleteFile)

	newSummary, deleteSummary := scanner.writeBackupDiffArtifacts(bakNewFile, bakDeleteFile)

	if len(recorder.errors) != 0 {
		t.Fatalf("RecordError calls = %d, want 0", len(recorder.errors))
	}
	if len(recorder.artifacts) != 2 {
		t.Fatalf("RecordArtifact calls = %d, want 2", len(recorder.artifacts))
	}

	assertBackupSummary(t, newSummary, len(bakNewFile), true, bakNewFile[:backupDiffSummarySampleLimit])
	assertBackupSummary(t, deleteSummary, len(bakDeleteFile), false, bakDeleteFile)

	if got := tools.MarshalJsonToString(newSummary); strings.Contains(got, bakNewFile[len(bakNewFile)-1]) {
		t.Fatalf("summary should not contain full tail detail, got %s", got)
	}

	assertFileLines(t, newSummary.ArtifactPath, bakNewFile)
	assertFileLines(t, deleteSummary.ArtifactPath, bakDeleteFile)

	if recorder.artifacts[0].payload["count"] != len(bakNewFile) {
		t.Fatalf("new artifact count payload = %v", recorder.artifacts[0].payload["count"])
	}
	if recorder.artifacts[0].payload["sampleLimit"] != backupDiffSummarySampleLimit {
		t.Fatalf("new artifact sampleLimit payload = %v", recorder.artifacts[0].payload["sampleLimit"])
	}
	if recorder.artifacts[0].payload["truncated"] != true {
		t.Fatalf("new artifact truncated payload = %v", recorder.artifacts[0].payload["truncated"])
	}

	var decoded backupDiffSummary
	if err := json.Unmarshal([]byte(tools.MarshalJsonToString(newSummary)), &decoded); err != nil {
		t.Fatalf("summary should marshal as JSON: %v", err)
	}
	if decoded.Count != len(bakNewFile) {
		t.Fatalf("decoded count = %d, want %d", decoded.Count, len(bakNewFile))
	}
}

func TestWriteBackupDiffArtifactsKeepsEmptySummaryWithoutFiles(t *testing.T) {
	ensureTestLogger()
	oldWorkDir := cons.WorkDir
	cons.WorkDir = t.TempDir()
	t.Cleanup(func() {
		cons.WorkDir = oldWorkDir
	})

	recorder := &artifactRecorder{}
	scanner := newScannerWithRecorder(model.DoScanImgArg{}, recorder)
	scanner.scanUUID = "2026-05-20-12-00-00_empty"

	newSummary, deleteSummary := scanner.writeBackupDiffArtifacts(nil, nil)

	assertBackupSummary(t, newSummary, 0, false, []string{})
	assertBackupSummary(t, deleteSummary, 0, false, []string{})
	if newSummary.ArtifactPath != "" || deleteSummary.ArtifactPath != "" {
		t.Fatalf("empty summaries should not have artifact paths: %#v %#v", newSummary, deleteSummary)
	}
	if len(recorder.artifacts) != 0 {
		t.Fatalf("RecordArtifact calls = %d, want 0", len(recorder.artifacts))
	}
	entries, err := os.ReadDir(filepath.Join(cons.WorkDir, "log", "dump_delete_file"))
	if err == nil && len(entries) != 0 {
		t.Fatalf("empty diffs should not create artifact files, entries = %d", len(entries))
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read artifact root: %v", err)
	}
}

func TestDumpFileProcessRecordsReviewOnlyDuplicateWhenSizeMismatch(t *testing.T) {
	ensureTestLogger()
	recorder := &captureRecorder{}
	scanner := newScannerWithRecorder(model.DoScanImgArg{}, recorder)
	scanner.md5Show = true

	root := t.TempDir()
	keep := filepath.Join(root, "2024", "2024-01", "2024-01-01", "IMG_0001.JPG")
	duplicate := filepath.Join(root, "2024", "2024-01", "2024-01-02", "IMG_0002.JPG")
	if err := os.MkdirAll(filepath.Dir(keep), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(duplicate), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keep, []byte("same-prefix-a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(duplicate, []byte("same-prefix-but-longer"), 0o644); err != nil {
		t.Fatal(err)
	}
	scanner.md5DumpMap["md5-1"] = []string{keep, duplicate}

	dumpMap := scanner.dumpFileProcess()
	if len(dumpMap) != 1 {
		t.Fatalf("dumpMap len = %d, want 1", len(dumpMap))
	}
	if len(scanner.shouldDeleteMd5Files) != 0 {
		t.Fatalf("shouldDeleteMd5Files = %#v, want empty", scanner.shouldDeleteMd5Files)
	}
	if len(recorder.items) != 1 {
		t.Fatalf("recorded items = %d, want 1", len(recorder.items))
	}
	item := recorder.items[0]
	if item.Stage != model.ActionStageDiscovery {
		t.Fatalf("stage = %q, want discovery", item.Stage)
	}
	if item.Status != model.ActionStatusSkipped {
		t.Fatalf("status = %q, want skipped", item.Status)
	}
	meta := parseTestMetadata(t, item.MetadataJSON)
	if meta["deleteEligible"] != false {
		t.Fatalf("deleteEligible = %v, want false", meta["deleteEligible"])
	}
	if meta["deleteIneligibleReason"] == "" {
		t.Fatal("deleteIneligibleReason should not be empty")
	}
}

func sortStringsForTest(items []string) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j] < items[j-1]; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

func assertBackupSummary(t *testing.T, summary backupDiffSummary, wantCount int, wantTruncated bool, wantSample []string) {
	t.Helper()
	if summary.Count != wantCount {
		t.Fatalf("summary.Count = %d, want %d", summary.Count, wantCount)
	}
	if summary.SampleLimit != backupDiffSummarySampleLimit {
		t.Fatalf("summary.SampleLimit = %d, want %d", summary.SampleLimit, backupDiffSummarySampleLimit)
	}
	if summary.Truncated != wantTruncated {
		t.Fatalf("summary.Truncated = %v, want %v", summary.Truncated, wantTruncated)
	}
	if !reflect.DeepEqual(summary.Sample, wantSample) {
		t.Fatalf("summary.Sample = %#v, want %#v", summary.Sample, wantSample)
	}
}

func assertFileLines(t *testing.T, filePath string, want []string) {
	t.Helper()
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read %s: %v", filePath, err)
	}
	got := strings.Split(string(content), "\n")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("file lines = %#v, want %#v", got, want)
	}
}

func TestProcessOneFileConcurrentAppends(t *testing.T) {
	ensureTestLogger()
	oldImgCache := cons.ImgCache
	oldMd5Retry := cons.Md5Retry
	oldMd5CountLength := cons.Md5CountLength
	cons.ImgCache = true
	cons.Md5Retry = 1
	cons.Md5CountLength = 0
	t.Cleanup(func() {
		cons.ImgCache = oldImgCache
		cons.Md5Retry = oldMd5Retry
		cons.Md5CountLength = oldMd5CountLength
	})

	root := filepath.Join(t.TempDir(), "pic-new")
	dayDir := filepath.Join(root, "2024", "2024-01", "2024-01-02")
	if err := os.MkdirAll(dayDir, 0o755); err != nil {
		t.Fatalf("mkdir all: %v", err)
	}

	scanner := newScanner(model.DoScanImgArg{})
	scanner.basePath = root
	scanner.exifDateNameSet = mapset.NewSet()
	scanner.getExifInfo = func(string) (string, string, int, string, []string, error) {
		return "2024:01:02 03:04:05", "", 1, "", []string{"[EXIF]DateTimeOriginal"}, nil
	}

	const fileCount = 16
	files := make([]string, 0, fileCount)
	for i := 0; i < fileCount; i++ {
		file := filepath.Join(dayDir, "IMG_"+string(rune('A'+i))+".JPG")
		if err := os.WriteFile(file, []byte("content"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		files = append(files, file)
	}

	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			scanner.processOneFile(file)
		}(file)
	}
	wg.Wait()

	if got := len(scanner.snapshotImgDatabaseDBList()); got != fileCount {
		t.Fatalf("imgDatabaseDBList length = %d, want %d", got, fileCount)
	}
	if got := scanner.exifDateNameSet.Cardinality(); got != 1 {
		t.Fatalf("exifDateNameSet cardinality = %d, want 1", got)
	}
}

func TestPathDuplicateProcessRecordsImgKeyDuplicates(t *testing.T) {
	recorder := &captureRecorder{}
	scanner := newScannerWithRecorder(model.DoScanImgArg{}, recorder)
	root := t.TempDir()
	plainDir := filepath.Join(root, "2024", "2024-01", "2024-01-02")
	namedDir := filepath.Join(root, "2024", "2024-01", "2024-01-02-trip")
	if err := os.MkdirAll(plainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(namedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(namedDir, "IMG_0001.JPG")
	duplicate := filepath.Join(plainDir, "IMG_0001.JPG")
	for _, file := range []string{keep, duplicate} {
		if err := os.WriteFile(file, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	imgKey := "2024-01-02|IMG_0001.JPG"
	scanner.pathDupMap[imgKey] = []string{duplicate, keep}
	pathDuplicateMap := scanner.pathDuplicateProcess()

	if len(pathDuplicateMap) != 1 {
		t.Fatalf("pathDuplicateMap len = %d, want 1", len(pathDuplicateMap))
	}
	if len(recorder.items) != 1 {
		t.Fatalf("recorded items = %d, want 1", len(recorder.items))
	}
	item := recorder.items[0]
	if item.ActionType != model.ActionTypeDeletePathDup {
		t.Fatalf("action type = %q, want %q", item.ActionType, model.ActionTypeDeletePathDup)
	}
	if item.SourcePath != duplicate {
		t.Fatalf("source path = %q, want duplicate %q", item.SourcePath, duplicate)
	}
	if item.TargetPath != keep {
		t.Fatalf("target path = %q, want keep %q", item.TargetPath, keep)
	}
	meta := parseTestMetadata(t, item.MetadataJSON)
	if meta["matchKey"] != imgKey {
		t.Fatalf("matchKey = %v, want %q", meta["matchKey"], imgKey)
	}
}

func TestChoosePathDuplicateKeepPhotoPrefersEarlierDirDate(t *testing.T) {
	root := t.TempDir()
	earlier := filepath.Join(root, "2024", "2024-01", "2024-01-02", "IMG_0001.JPG")
	later := filepath.Join(root, "2024", "2024-02", "2024-02-03", "IMG_0001.JPG")

	if got := choosePathDuplicateKeepPhoto([]string{later, earlier}); got != earlier {
		t.Fatalf("keep photo = %q, want earlier dir date %q", got, earlier)
	}
}

func TestChoosePathDuplicateKeepPhotoPrefersDescribedDirForSameDate(t *testing.T) {
	root := t.TempDir()
	plain := filepath.Join(root, "2024", "2024-01", "2024-01-02", "IMG_0001.JPG")
	described := filepath.Join(root, "2024", "2024-01", "2024-01-02-trip", "IMG_0001.JPG")

	if got := choosePathDuplicateKeepPhoto([]string{plain, described}); got != described {
		t.Fatalf("keep photo = %q, want described dir %q", got, described)
	}
}

func TestScannersKeepIndependentState(t *testing.T) {
	ensureTestLogger()
	root := filepath.Join(t.TempDir(), "pic-new")
	dayDir := filepath.Join(root, "2024", "2024-01", "2024-01-02")
	if err := os.MkdirAll(dayDir, 0o755); err != nil {
		t.Fatalf("mkdir all: %v", err)
	}

	fileOne := filepath.Join(dayDir, "IMG_ONE.JPG")
	fileTwo := filepath.Join(dayDir, "IMG_TWO.JPG")
	for _, file := range []string{fileOne, fileTwo} {
		if err := os.WriteFile(file, []byte("content"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	scannerOne := newScanner(model.DoScanImgArg{})
	scannerOne.basePath = root
	scannerOne.imgCache = map[string]middleware.ImgCacheData{"2024-01-02|IMG_ONE.JPG": {ShootDate: "2024:01:02 01:02:03", LocStreet: "StreetOne"}}
	scannerOne.staleImgCache = cloneImgCache(scannerOne.imgCache)

	scannerTwo := newScanner(model.DoScanImgArg{})
	scannerTwo.basePath = root
	scannerTwo.getExifInfo = func(string) (string, string, int, string, []string, error) {
		return "2024:01:02 04:05:06", "", 1, "", []string{"[EXIF]DateTimeOriginalTwo"}, nil
	}

	scannerOne.processOneFile(fileOne)
	scannerTwo.processOneFile(fileTwo)

	if len(scannerOne.snapshotStaleImgCache()) != 0 {
		t.Fatalf("scannerOne staleImgCache should be empty after cache hit")
	}
	if len(scannerTwo.snapshotStaleImgCache()) != 0 {
		t.Fatalf("scannerTwo staleImgCache should stay independent and empty")
	}
	if scannerOne.exifDateNameSet.Cardinality() != 0 {
		t.Fatalf("scannerOne exifDateNameSet should stay empty on cache hit")
	}
	if !scannerTwo.exifDateNameSet.Contains("[EXIF]DateTimeOriginalTwo") {
		t.Fatalf("scannerTwo exifDateNameSet missing expected tag")
	}
}

func TestRunReturnsStartupErrorWhenCacheLoadFails(t *testing.T) {
	ensureTestLogger()
	root := filepath.Join(t.TempDir(), "pic-new")
	dayDir := filepath.Join(root, "2024", "2024-01", "2024-01-02")
	if err := os.MkdirAll(dayDir, 0o755); err != nil {
		t.Fatalf("mkdir all: %v", err)
	}

	startPath := dayDir
	scanner := newScanner(model.DoScanImgArg{StartPath: &startPath})
	scanner.loadGisCache = func() (map[string]middleware.GisData, error) {
		return nil, errors.New("gis unavailable")
	}

	if _, err := scanner.Run(); err == nil {
		t.Fatalf("Run should return startup error")
	}
}

func TestMoveThenRenameUsesMovedPath(t *testing.T) {
	ensureTestLogger()
	root := t.TempDir()
	currentDir := filepath.Join(root, "2024", "2024-01", "2024-01-03")
	targetDir := filepath.Join(root, "2024", "2024-01", "2024-01-02")
	if err := os.MkdirAll(currentDir, 0o755); err != nil {
		t.Fatalf("mkdir currentDir: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir targetDir: %v", err)
	}

	original := filepath.Join(currentDir, "IMG_0001.JPG")
	moved := filepath.Join(targetDir, "IMG_0001.JPG")
	renamed := filepath.Join(targetDir, "IMG_0001[2024-01-02_10-11-12^Road].JPG")
	if err := os.WriteFile(original, []byte("content"), 0o644); err != nil {
		t.Fatalf("write original file: %v", err)
	}

	recorder := &actionResultRecorder{}
	scanner := newScannerWithRecorder(model.DoScanImgArg{}, recorder)
	scanner.moveFileAction = true
	scanner.renameFileAction = true

	ps := photoStruct{
		photo:            original,
		isMoveFile:       true,
		moveTargetPath:   moved,
		isRenameFile:     true,
		renameTargetPath: renamed,
		moveActionID:     1,
		renameActionID:   2,
	}

	printFileFlag := false
	printDateFlag := false
	ps = scanner.moveFileProcess(ps, &printFileFlag, &printDateFlag)
	ps = scanner.renameFileProcess(ps, &printFileFlag, &printDateFlag)

	if _, err := os.Stat(renamed); err != nil {
		t.Fatalf("expected renamed file to exist: %v", err)
	}
	if _, err := os.Stat(original); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected original file to be gone, got: %v", err)
	}
	if _, err := os.Stat(moved); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected intermediate moved file to be gone after rename, got: %v", err)
	}
	if len(recorder.results) != 2 {
		t.Fatalf("recorded results = %d, want 2", len(recorder.results))
	}
	if recorder.results[0].payload["targetPath"] != moved {
		t.Fatalf("move targetPath payload = %v", recorder.results[0].payload["targetPath"])
	}
	if recorder.results[1].payload["path"] != moved {
		t.Fatalf("rename source path payload = %v", recorder.results[1].payload["path"])
	}
	if recorder.results[1].payload["targetPath"] != renamed {
		t.Fatalf("rename targetPath payload = %v", recorder.results[1].payload["targetPath"])
	}
	if ps.photo != renamed {
		t.Fatalf("final photo path = %q", ps.photo)
	}
}

func TestWalkPrimaryPathMarksIncompleteOnWalkError(t *testing.T) {
	ensureTestLogger()
	root := filepath.Join(t.TempDir(), "pic-new")
	okDir := filepath.Join(root, "2024", "2024-01", "2024-01-02")
	badDir := filepath.Join(root, "2024", "2024-01", "2024-01-03")
	if err := os.MkdirAll(okDir, 0o755); err != nil {
		t.Fatalf("mkdir okDir: %v", err)
	}
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatalf("mkdir badDir: %v", err)
	}
	if err := os.Chmod(badDir, 0); err != nil {
		t.Fatalf("chmod badDir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(badDir, 0o755)
	})

	startPath := root
	scanner := newScanner(model.DoScanImgArg{StartPath: &startPath})
	scanner.basePath = root

	pool, err := ants.NewPool(1)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Release()

	if err := scanner.walkPrimaryPath(pool); err != nil {
		t.Fatalf("walkPrimaryPath returned unexpected error: %v", err)
	}
	scanner.wg.Wait()

	if scanner.isComplete != 0 {
		t.Fatalf("scanner.isComplete = %d, want 0 when walk warning happens", scanner.isComplete)
	}
	if scanner.firstErr == nil {
		t.Fatalf("scanner.firstErr should be recorded")
	}
}
