package middleware

import (
	"errors"
	"testing"
)

func TestGetExifInfoFallsBackToGo(t *testing.T) {
	oldRunner := runExiftoolFunc
	oldGo := getExifInfoGoFunc
	runExiftoolFunc = func(args ...string) (string, error) {
		return "", errors.New("exiftool failed")
	}
	getExifInfoGoFunc = func(path string) (string, string, error) {
		return "2024:01:02 03:04:05", "", nil
	}
	t.Cleanup(func() {
		runExiftoolFunc = oldRunner
		getExifInfoGoFunc = oldGo
	})

	_, _, flag, _, _, err := GetExifInfo("/tmp/test.jpg")
	if err != nil {
		t.Fatalf("GetExifInfo returned error: %v", err)
	}
	if flag != 2 {
		t.Fatalf("flag = %d, want 2", flag)
	}
}

func TestGetExifInfoSkipsExiftoolForUnsupportedSuffix(t *testing.T) {
	oldRunner := runExiftoolFunc
	oldGo := getExifInfoGoFunc
	called := false
	runExiftoolFunc = func(args ...string) (string, error) {
		called = true
		return "", nil
	}
	getExifInfoGoFunc = func(path string) (string, string, error) {
		return "", "", errors.New("no exif")
	}
	t.Cleanup(func() {
		runExiftoolFunc = oldRunner
		getExifInfoGoFunc = oldGo
	})

	_, _, _, _, _, _ = GetExifInfo("/tmp/test.txt")
	if called {
		t.Fatal("exiftool should not be called for unsupported suffix")
	}
}
