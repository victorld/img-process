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
	if decision.photo.targetPhoto == "" {
		t.Fatal("decision.photo.targetPhoto should not be empty")
	}
}
