package service

import (
	"errors"
	"os"
	"path/filepath"
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

func ensureTestLogger() {
	if tools.Logger == nil {
		tools.Logger = zap.NewNop().Sugar()
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
