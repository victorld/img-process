package tools

import (
	"errors"
	"os"
	"path/filepath"
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
