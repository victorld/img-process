package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"img_process/model"
)

type captureRecorder struct {
	noopScanRecorder
	items []model.ScanActionItemDB
}

func (r *captureRecorder) RecordCandidateAction(item model.ScanActionItemDB) uint {
	r.items = append(r.items, item)
	return uint(len(r.items))
}

func TestApplyFileDecisionWritesStructuredMetadata(t *testing.T) {
	recorder := &captureRecorder{}
	scanner := newScannerWithRecorder(model.DoScanImgArg{}, recorder)
	decision := fileDecision{
		hasChanges:   true,
		shouldMove:   true,
		shouldRename: true,
		shouldModify: true,
		photo: photoStruct{
			photo:            filepath.Join("/photos", "2024", "2024-01", "2024-01-03", "IMG_0001.JPG"),
			moveTargetPath:   filepath.Join("/photos", "2024", "2024-01", "2024-01-02", "IMG_0001.JPG"),
			renameTargetPath: filepath.Join("/photos", "2024", "2024-01", "2024-01-02", "IMG_0001[2024-01-02_10-11-12^Road].JPG"),
			dirDate:          "2024-01-03",
			modifyDate:       "2024-01-03",
			fileDate:         "2024-01-02",
			shootDate:        "2024-01-02",
			shootDateRaw:     "2024:01:02 10:11:12",
			minDate:          "2024-01-02",
		},
	}

	scanner.applyFileDecision(decision)

	if len(recorder.items) != 3 {
		t.Fatalf("recorded items = %d, want 3", len(recorder.items))
	}

	moveItem := recorder.items[0]
	moveMeta := parseTestMetadata(t, moveItem.MetadataJSON)
	if moveMeta["fileName"] != "IMG_0001.JPG" {
		t.Fatalf("move metadata fileName = %v", moveMeta["fileName"])
	}
	if moveMeta["fileNameDate"] != "2024-01-02" {
		t.Fatalf("move metadata fileNameDate = %v", moveMeta["fileNameDate"])
	}
	if moveMeta["shootDateRaw"] != "2024:01:02 10:11:12" {
		t.Fatalf("move metadata shootDateRaw = %v", moveMeta["shootDateRaw"])
	}
	if moveItem.TargetPath != decision.photo.moveTargetPath {
		t.Fatalf("move target path = %q", moveItem.TargetPath)
	}
	if moveMeta["targetPath"] != decision.photo.moveTargetPath {
		t.Fatalf("move metadata targetPath = %v", moveMeta["targetPath"])
	}

	renameItem := recorder.items[1]
	renameMeta := parseTestMetadata(t, renameItem.MetadataJSON)
	if renameMeta["targetFileName"] == "" {
		t.Fatal("rename metadata targetFileName should not be empty")
	}
	if renameItem.TargetPath != decision.photo.renameTargetPath {
		t.Fatalf("rename target path = %q", renameItem.TargetPath)
	}
	if renameMeta["targetPath"] != decision.photo.renameTargetPath {
		t.Fatalf("rename metadata targetPath = %v", renameMeta["targetPath"])
	}

	modifyItem := recorder.items[2]
	modifyMeta := parseTestMetadata(t, modifyItem.MetadataJSON)
	if modifyMeta["modifyDate"] != "2024-01-03" {
		t.Fatalf("modify metadata modifyDate = %v", modifyMeta["modifyDate"])
	}
	if modifyMeta["minDate"] != "2024-01-02" {
		t.Fatalf("modify metadata minDate = %v", modifyMeta["minDate"])
	}
}

func TestBuildActionItemViewBuildsDuplicatePair(t *testing.T) {
	item := model.ScanActionItemDB{
		CommonModel:    model.CommonModel{ID: 7},
		ActionType:     model.ActionTypeDeleteDup,
		ObjectType:     model.ActionObjectFile,
		SourcePath:     "/photos/a.jpg",
		DuplicateGroup: "dup-1",
		MetadataJSON:   `{"fileName":"a.jpg","currentPath":"/photos/a.jpg","keepPath":"/photos/b.jpg","keepFileName":"b.jpg","sizeMatch":true,"deleteEligible":true}`,
	}

	view := buildActionItemView(item)
	if view.Pair == nil {
		t.Fatal("view.Pair should not be nil")
	}
	if view.Pair.PhotoA.Path != "/photos/a.jpg" {
		t.Fatalf("photoA path = %q", view.Pair.PhotoA.Path)
	}
	if view.Pair.PhotoB.Path != "/photos/b.jpg" {
		t.Fatalf("photoB path = %q", view.Pair.PhotoB.Path)
	}
	if view.DuplicateMeta == nil || !view.DuplicateMeta.DeleteEligible {
		t.Fatalf("duplicate meta = %#v, want delete eligible", view.DuplicateMeta)
	}
}

func TestBuildActionItemViewBuildsReviewOnlyDuplicatePair(t *testing.T) {
	item := model.ScanActionItemDB{
		CommonModel:    model.CommonModel{ID: 10},
		ActionType:     model.ActionTypeDeleteDup,
		ObjectType:     model.ActionObjectFile,
		SourcePath:     "/photos/a.jpg",
		TargetPath:     "/photos/b.jpg",
		DuplicateGroup: "dup-review",
		Stage:          model.ActionStageDiscovery,
		Status:         model.ActionStatusSkipped,
		MetadataJSON:   `{"fileName":"a.jpg","currentPath":"/photos/a.jpg","keepPath":"/photos/b.jpg","keepFileName":"b.jpg","sizeMatch":false,"deleteEligible":false,"deleteIneligibleReason":"同 MD5 分组内文件大小不一致，需人工核对"}`,
	}

	view := buildActionItemView(item)
	if view.Pair == nil {
		t.Fatal("view.Pair should not be nil")
	}
	if view.DuplicateMeta == nil {
		t.Fatal("view.DuplicateMeta should not be nil")
	}
	if view.DuplicateMeta.DeleteEligible {
		t.Fatal("review-only duplicate should not be delete eligible")
	}
	if view.DuplicateMeta.DeleteIneligibleReason == "" {
		t.Fatal("review-only duplicate should include reason")
	}
}

func TestLegacyDuplicateCompareLineView(t *testing.T) {
	view, ok := legacyDuplicateCompareLineView(
		2,
		"scan-1",
		0,
		"abc123 : IMG_1.JPG|IMG_2.JPG|IMG_3.JPG",
		nil,
		func(name string) []string {
			return []string{filepath.Join("/photos", name)}
		},
	)
	if !ok {
		t.Fatal("legacy line should build a view")
	}
	if view.Pair == nil {
		t.Fatal("legacy view pair should not be nil")
	}
	if view.Pair.PhotoA.FileName != "IMG_1.JPG" {
		t.Fatalf("photoA file name = %q", view.Pair.PhotoA.FileName)
	}
	if len(view.DuplicatePhotos) != 3 {
		t.Fatalf("duplicate photos len = %d, want 3", len(view.DuplicatePhotos))
	}
	if view.Pair.PhotoB.FileName != "IMG_2.JPG" {
		t.Fatalf("photoB file name = %q", view.Pair.PhotoB.FileName)
	}
	if view.DuplicateMeta == nil || !view.DuplicateMeta.DeleteEligible {
		t.Fatalf("legacy duplicate meta = %#v, want delete eligible", view.DuplicateMeta)
	}
}

func TestLegacyDuplicatePathResolverUsesOnlyPrimaryScanRoot(t *testing.T) {
	primaryRoot := t.TempDir()
	backupRoot := t.TempDir()
	primaryFile := filepath.Join(primaryRoot, "2024", "IMG_0001[2024-01-01].JPG")
	backupFile := filepath.Join(backupRoot, "2024", "IMG_0001.JPG")
	if err := os.MkdirAll(filepath.Dir(primaryFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(backupFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(primaryFile, []byte("primary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte("backup"), 0o644); err != nil {
		t.Fatal(err)
	}

	job := model.ScanJobDB{
		ScanArgs: `{"startPath":"` + primaryRoot + `","startPathBak":"` + backupRoot + `"}`,
	}

	matches := newLegacyDuplicatePathResolver(job)("IMG_0001")
	if len(matches) != 1 {
		t.Fatalf("matches = %#v, want exactly primary match", matches)
	}
	if matches[0] != primaryFile {
		t.Fatalf("match = %q, want %q", matches[0], primaryFile)
	}
}

func TestBuildDuplicateGroupViewsMergesSameMD5Group(t *testing.T) {
	items := []model.ScanActionItemDB{
		{
			CommonModel:    model.CommonModel{ID: 1},
			ActionType:     model.ActionTypeDeleteDup,
			ObjectType:     model.ActionObjectFile,
			SourcePath:     "/photos/b.jpg",
			TargetPath:     "/photos/a.jpg",
			DuplicateGroup: "md5-1",
			MetadataJSON:   `{"deleteEligible":true,"sizeMatch":true,"duplicatePhotos":[{"fileName":"a.jpg","path":"/photos/a.jpg","recommendedDelete":false,"deleteEligible":true},{"fileName":"b.jpg","path":"/photos/b.jpg","recommendedDelete":true,"deleteEligible":true},{"fileName":"c.jpg","path":"/photos/c.jpg","recommendedDelete":true,"deleteEligible":true}]}`,
		},
		{
			CommonModel:    model.CommonModel{ID: 2},
			ActionType:     model.ActionTypeDeleteDup,
			ObjectType:     model.ActionObjectFile,
			SourcePath:     "/photos/c.jpg",
			TargetPath:     "/photos/a.jpg",
			DuplicateGroup: "md5-1",
			MetadataJSON:   `{"deleteEligible":true,"sizeMatch":true,"duplicatePhotos":[{"fileName":"a.jpg","path":"/photos/a.jpg","recommendedDelete":false,"deleteEligible":true},{"fileName":"b.jpg","path":"/photos/b.jpg","recommendedDelete":true,"deleteEligible":true},{"fileName":"c.jpg","path":"/photos/c.jpg","recommendedDelete":true,"deleteEligible":true}]}`,
		},
	}

	views := buildDuplicateGroupViews(items)
	if len(views) != 1 {
		t.Fatalf("views len = %d, want 1", len(views))
	}
	if len(views[0].DuplicatePhotos) != 3 {
		t.Fatalf("duplicatePhotos len = %d, want 3", len(views[0].DuplicatePhotos))
	}
}

func TestBuildDuplicateGroupViewsKeepsExecutedDuplicatePhotosSeparate(t *testing.T) {
	now := time.Now()
	items := []model.ScanActionItemDB{
		{
			CommonModel:    model.CommonModel{ID: 1},
			ActionType:     model.ActionTypeDeleteDup,
			ObjectType:     model.ActionObjectFile,
			SourcePath:     "/photos/b.jpg",
			TargetPath:     "/photos/a.jpg",
			Stage:          model.ActionStageExecuted,
			Status:         model.ActionStatusSucceeded,
			ExecutedAt:     &now,
			DuplicateGroup: "md5-1",
			MetadataJSON:   `{"executedDeletePath":"/photos/b.jpg","duplicatePhotos":[{"fileName":"a.jpg","path":"/photos/a.jpg","recommendedDelete":false,"deleteEligible":true},{"fileName":"b.jpg","path":"/photos/b.jpg","recommendedDelete":true,"deleteEligible":true},{"fileName":"c.jpg","path":"/photos/c.jpg","recommendedDelete":true,"deleteEligible":true}]}`,
		},
		{
			CommonModel:    model.CommonModel{ID: 2},
			ActionType:     model.ActionTypeDeleteDup,
			ObjectType:     model.ActionObjectFile,
			SourcePath:     "/photos/c.jpg",
			TargetPath:     "/photos/a.jpg",
			Stage:          model.ActionStageExecuted,
			Status:         model.ActionStatusSucceeded,
			ExecutedAt:     &now,
			DuplicateGroup: "md5-1",
			MetadataJSON:   `{"executedDeletePath":"/photos/c.jpg","duplicatePhotos":[{"fileName":"a.jpg","path":"/photos/a.jpg","recommendedDelete":false,"deleteEligible":true},{"fileName":"b.jpg","path":"/photos/b.jpg","recommendedDelete":true,"deleteEligible":true},{"fileName":"c.jpg","path":"/photos/c.jpg","recommendedDelete":true,"deleteEligible":true}]}`,
		},
	}

	views := buildDuplicateGroupViews(items)
	if len(views) != 2 {
		t.Fatalf("views len = %d, want 2", len(views))
	}
	for _, view := range views {
		if len(view.DuplicatePhotos) != 3 {
			t.Fatalf("executed duplicate photos len = %d, want 3", len(view.DuplicatePhotos))
		}
		var executedCount int
		for _, photo := range view.DuplicatePhotos {
			if photo.ExecutedAction {
				executedCount++
				if photo.Path != view.SourcePath {
					t.Fatalf("executed photo path = %q, want source %q", photo.Path, view.SourcePath)
				}
			}
		}
		if executedCount != 1 {
			t.Fatalf("executed action marks = %d, want 1", executedCount)
		}
	}
}

func TestBuildDuplicateGroupViewsExcludesExecutedDuplicatePhotoFromPending(t *testing.T) {
	items := []model.ScanActionItemDB{
		{
			CommonModel:    model.CommonModel{ID: 1},
			ActionType:     model.ActionTypeDeleteDup,
			ObjectType:     model.ActionObjectFile,
			SourcePath:     "/photos/b.jpg",
			TargetPath:     "/photos/a.jpg",
			DuplicateGroup: "md5-1",
			MetadataJSON:   `{"deleteEligible":true,"sizeMatch":true,"duplicatePhotos":[{"fileName":"a.jpg","path":"/photos/a.jpg","recommendedDelete":false,"deleteEligible":true},{"fileName":"b.jpg","path":"/photos/b.jpg","recommendedDelete":true,"deleteEligible":true},{"fileName":"c.jpg","path":"/photos/c.jpg","recommendedDelete":true,"deleteEligible":true}]}`,
		},
		{
			CommonModel:    model.CommonModel{ID: 2},
			ActionType:     model.ActionTypeDeleteDup,
			ObjectType:     model.ActionObjectFile,
			SourcePath:     "/photos/c.jpg",
			TargetPath:     "/photos/a.jpg",
			DuplicateGroup: "md5-1",
			MetadataJSON:   `{"deleteEligible":true,"sizeMatch":true,"duplicatePhotos":[{"fileName":"a.jpg","path":"/photos/a.jpg","recommendedDelete":false,"deleteEligible":true},{"fileName":"b.jpg","path":"/photos/b.jpg","recommendedDelete":true,"deleteEligible":true},{"fileName":"c.jpg","path":"/photos/c.jpg","recommendedDelete":true,"deleteEligible":true}]}`,
		},
		{
			CommonModel:    model.CommonModel{ID: 3},
			ActionType:     model.ActionTypeDeleteDup,
			ObjectType:     model.ActionObjectFile,
			SourcePath:     "/photos/b.jpg",
			TargetPath:     "/photos/a.jpg",
			Stage:          model.ActionStageExecuted,
			Status:         model.ActionStatusSucceeded,
			DuplicateGroup: "md5-1",
			MetadataJSON:   `{"executedDeletePath":"/photos/b.jpg","recommendedDeletePath":"/photos/b.jpg","duplicatePhotos":[{"fileName":"a.jpg","path":"/photos/a.jpg","recommendedDelete":false,"deleteEligible":true},{"fileName":"b.jpg","path":"/photos/b.jpg","recommendedDelete":true,"deleteEligible":true},{"fileName":"c.jpg","path":"/photos/c.jpg","recommendedDelete":true,"deleteEligible":true}]}`,
		},
	}

	excludedPaths := duplicateExecutedDeletePaths(items)
	views := buildDuplicateGroupViewsWithExcludedPaths(items[:2], excludedPaths)
	if len(views) != 1 {
		t.Fatalf("views len = %d, want 1", len(views))
	}
	if got := countDuplicateViews(views); got != 1 {
		t.Fatalf("duplicate pending views = %d, want 1", got)
	}
	for _, photo := range views[0].DuplicatePhotos {
		if photo.Path == "/photos/b.jpg" {
			t.Fatalf("executed photo %q should be excluded from pending view", photo.Path)
		}
	}
}

func TestShouldUseLegacyDuplicateCompareOnlyBeforeAnyDuplicateResult(t *testing.T) {
	search := model.ScanActionItemSearch{
		Tab:        "pending",
		ActionType: model.ActionTypeDeleteDup,
	}
	grouped := model.ScanActionGroupedCounts{
		Executed: model.ScanActionCounts{DeleteDuplicate: 1, Total: 1},
	}

	if shouldUseLegacyDuplicateCompare(search, 0, grouped) {
		t.Fatal("legacy duplicate compare should stay disabled after executed duplicate actions exist")
	}
}

func TestBuildActionItemViewIncludesModifyDate(t *testing.T) {
	item := model.ScanActionItemDB{
		CommonModel:  model.CommonModel{ID: 8},
		ActionType:   model.ActionTypeModifyTime,
		ObjectType:   model.ActionObjectFile,
		SourcePath:   "/photos/a.jpg",
		MetadataJSON: `{"fileName":"a.jpg","currentPath":"/photos/a.jpg","dirDate":"2024-01-03","modifyDate":"2024-01-05","minDate":"2024-01-02"}`,
	}

	view := buildActionItemView(item)
	if view.Detail == nil {
		t.Fatal("view.Detail should not be nil")
	}
	if view.Detail.ModifyDate != "2024-01-05" {
		t.Fatalf("detail modifyDate = %q", view.Detail.ModifyDate)
	}
}

func TestBuildActionItemViewFallsBackToSourceModifyDate(t *testing.T) {
	file := filepath.Join(t.TempDir(), "legacy.jpg")
	if err := os.WriteFile(file, []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}

	item := model.ScanActionItemDB{
		CommonModel:  model.CommonModel{ID: 9},
		ActionType:   model.ActionTypeModifyTime,
		ObjectType:   model.ActionObjectFile,
		SourcePath:   file,
		MetadataJSON: `{"fileName":"legacy.jpg","currentPath":"` + file + `"}`,
	}

	view := buildActionItemView(item)
	if view.Detail == nil {
		t.Fatal("view.Detail should not be nil")
	}
	if view.Detail.ModifyDate == "" {
		t.Fatal("detail modifyDate should fall back to source file modify date")
	}
}

func TestIsPathInRoots(t *testing.T) {
	allowed := []string{filepath.Clean("/photos")}
	if !isPathInRoots("/photos/2024/IMG_0001.JPG", allowed) {
		t.Fatal("expected path to be allowed")
	}
	if isPathInRoots("/other/IMG_0001.JPG", allowed) {
		t.Fatal("expected path to be rejected")
	}
}

func parseTestMetadata(t *testing.T, raw string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	return payload
}
