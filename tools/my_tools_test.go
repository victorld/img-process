package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestMoveFileReturnsRenameError(t *testing.T) {
	oldRename := renameFile
	renameFile = func(src, dst string) error {
		return errors.New("rename failed")
	}
	t.Cleanup(func() {
		renameFile = oldRename
	})

	src := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := MoveFile(src, filepath.Join(t.TempDir(), "b.txt")); err == nil {
		t.Fatal("expected move error")
	}
}

func TestMoveFileFallsBackToCopyOnEXDEV(t *testing.T) {
	oldRename := renameFile
	renameFile = func(src, dst string) error {
		return syscall.EXDEV
	}
	t.Cleanup(func() {
		renameFile = oldRename
	})

	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "a.txt")
	dst := filepath.Join(tmpDir, "nested", "b.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := MoveFile(src, dst); err != nil {
		t.Fatalf("MoveFile returned error: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("src should be removed, got err=%v", err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("dst should exist: %v", err)
	}
}

func TestDeleteFileMovesFileToTrash(t *testing.T) {
	trashDir := useTempTrashDir(t)

	src := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := DeleteFile(src); err != nil {
		t.Fatalf("DeleteFile returned error: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("src should be removed, got err=%v", err)
	}
	assertFileContent(t, filepath.Join(trashDir, "a.txt"), "hello")
}

func TestDeleteFileKeepsExistingTrashFile(t *testing.T) {
	trashDir := useTempTrashDir(t)
	if err := os.WriteFile(filepath.Join(trashDir, "a.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(src, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := DeleteFile(src); err != nil {
		t.Fatalf("DeleteFile returned error: %v", err)
	}

	assertFileContent(t, filepath.Join(trashDir, "a.txt"), "old")
	entries, err := os.ReadDir(trashDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("trash entries = %d, want 2", len(entries))
	}
	foundNew := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "a-") && strings.HasSuffix(entry.Name(), ".txt") {
			assertFileContent(t, filepath.Join(trashDir, entry.Name()), "new")
			foundNew = true
		}
	}
	if !foundNew {
		t.Fatalf("renamed trash file not found in %#v", entries)
	}
}

func TestDeleteFileReturnsNotExist(t *testing.T) {
	useTempTrashDir(t)

	err := DeleteFile(filepath.Join(t.TempDir(), "missing.txt"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("DeleteFile error = %v, want os.ErrNotExist", err)
	}
}

func TestDeleteEmptyDirMovesDirToTrashAndProcessesParents(t *testing.T) {
	trashDir := useTempTrashDir(t)

	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := DeleteEmptyDir(child); err != nil {
		t.Fatalf("DeleteEmptyDir returned error: %v", err)
	}
	if _, err := os.Stat(child); !os.IsNotExist(err) {
		t.Fatalf("child should be removed, got err=%v", err)
	}
	if _, err := os.Stat(parent); !os.IsNotExist(err) {
		t.Fatalf("parent should be removed, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(trashDir, "child")); err != nil {
		t.Fatalf("trashed child dir should exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(trashDir, "parent")); err != nil {
		t.Fatalf("trashed parent dir should exist: %v", err)
	}
}

func TestGetDirDateReturnsEmptyForInvalidParentDir(t *testing.T) {
	file := filepath.Join("/tmp", "invalid-dir", "a.jpg")
	if got := GetDirDate(file); got != "" {
		t.Fatalf("GetDirDate() = %q, want empty", got)
	}
}

func TestGetDirDateReturnsDateForValidParentDir(t *testing.T) {
	file := filepath.Join("/tmp", "2024-01-02", "a.jpg")
	if got := GetDirDate(file); got != "2024-01-02" {
		t.Fatalf("GetDirDate() = %q, want %q", got, "2024-01-02")
	}
}

func useTempTrashDir(t *testing.T) string {
	t.Helper()
	trashDir := t.TempDir()
	oldTrashDirPath := trashDirPath
	trashDirPath = func() string {
		return trashDir
	}
	t.Cleanup(func() {
		trashDirPath = oldTrashDirPath
	})
	return trashDir
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(content) != want {
		t.Fatalf("%s content = %q, want %q", path, string(content), want)
	}
}
