package service

import (
	"encoding/json"
	"path/filepath"
	"testing"

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
			photo:        filepath.Join("/photos", "2024", "2024-01", "2024-01-03", "IMG_0001.JPG"),
			targetPhoto:  filepath.Join("/photos", "2024", "2024-01", "2024-01-02", "2024-01-02_10-11-12^Road.JPG"),
			dirDate:      "2024-01-03",
			fileDate:     "2024-01-02",
			shootDate:    "2024-01-02",
			shootDateRaw: "2024:01:02 10:11:12",
			minDate:      "2024-01-02",
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

	renameItem := recorder.items[1]
	renameMeta := parseTestMetadata(t, renameItem.MetadataJSON)
	if renameMeta["targetFileName"] == "" {
		t.Fatal("rename metadata targetFileName should not be empty")
	}

	modifyItem := recorder.items[2]
	modifyMeta := parseTestMetadata(t, modifyItem.MetadataJSON)
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
		MetadataJSON:   `{"fileName":"a.jpg","currentPath":"/photos/a.jpg","keepPath":"/photos/b.jpg","keepFileName":"b.jpg"}`,
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
