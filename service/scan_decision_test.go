package service

import (
	"path/filepath"
	"testing"
)

func TestEvaluateFileDecisionSkipsMoveOnInvalidDirDate(t *testing.T) {
	meta := fileMetadata{
		photo:           filepath.Join("/tmp", "invalid-dir", "IMG_0001.JPG"),
		dirDate:         "",
		modifyDate:      "2024-01-02",
		shootDate:       "2024-01-01",
		shootDateOrigin: "2024:01:01 10:11:12",
	}

	decision := evaluateFileDecision(meta, filepath.Join("/tmp", "pic-new"))
	if decision.shouldMove {
		t.Fatal("decision.shouldMove should be false when dir date is invalid")
	}
	if !decision.shouldRename {
		t.Fatal("decision.shouldRename should still be true when rename metadata changes")
	}
}

func TestEvaluateFileDecisionBuildsMoveTargetForValidDate(t *testing.T) {
	meta := fileMetadata{
		photo:           filepath.Join("/tmp", "pic-new", "2024", "2024-01", "2024-01-03", "IMG_0001.JPG"),
		dirDate:         "2024-01-03",
		modifyDate:      "2024-01-03",
		shootDate:       "2024-01-02",
		shootDateOrigin: "2024:01:02 10:11:12",
	}

	decision := evaluateFileDecision(meta, filepath.Join("/tmp", "pic-new"))
	if !decision.shouldMove {
		t.Fatal("decision.shouldMove should be true")
	}
	if decision.photo.moveTargetPath == "" {
		t.Fatal("decision.photo.moveTargetPath should not be empty")
	}
	if filepath.Dir(decision.photo.moveTargetPath) != filepath.Join("/tmp", "pic-new", "2024", "2024-01", "2024-01-02") {
		t.Fatalf("move target dir = %q", filepath.Dir(decision.photo.moveTargetPath))
	}
}

func TestEvaluateFileDecisionBuildsRenameTargetFromMoveTarget(t *testing.T) {
	meta := fileMetadata{
		photo:           filepath.Join("/tmp", "pic-new", "2024", "2024-01", "2024-01-03", "IMG_0001.JPG"),
		dirDate:         "2024-01-03",
		modifyDate:      "2024-01-03",
		shootDate:       "2024-01-02",
		shootDateOrigin: "2024:01:02 10:11:12",
		locStreet:       "Road",
	}

	decision := evaluateFileDecision(meta, filepath.Join("/tmp", "pic-new"))
	if !decision.shouldMove {
		t.Fatal("decision.shouldMove should be true")
	}
	if !decision.shouldRename {
		t.Fatal("decision.shouldRename should be true")
	}
	if filepath.Dir(decision.photo.moveTargetPath) != filepath.Join("/tmp", "pic-new", "2024", "2024-01", "2024-01-02") {
		t.Fatalf("move target dir = %q", filepath.Dir(decision.photo.moveTargetPath))
	}
	if filepath.Dir(decision.photo.renameTargetPath) != filepath.Join("/tmp", "pic-new", "2024", "2024-01", "2024-01-02") {
		t.Fatalf("rename target dir = %q", filepath.Dir(decision.photo.renameTargetPath))
	}
	if decision.photo.renameTargetPath == decision.photo.moveTargetPath {
		t.Fatal("rename target path should differ from move target path")
	}
}
