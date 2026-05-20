package service

import (
	"encoding/json"
	"testing"
	"time"

	"img_process/model"
)

func TestDuplicateDeleteTarget(t *testing.T) {
	item := model.ScanActionItemDB{
		ActionType:   model.ActionTypeDeleteDup,
		SourcePath:   "/photos/a.jpg",
		TargetPath:   "/photos/b.jpg",
		MetadataJSON: `{"currentPath":"/photos/a.jpg","keepPath":"/photos/b.jpg"}`,
	}

	side, deletePath, recommendedDeletePath, err := duplicateDeleteTarget(item, "A", "")
	if err != nil {
		t.Fatalf("duplicateDeleteTarget A error: %v", err)
	}
	if side != "A" {
		t.Fatalf("side = %q, want A", side)
	}
	if deletePath != "/photos/a.jpg" {
		t.Fatalf("deletePath = %q, want /photos/a.jpg", deletePath)
	}
	if recommendedDeletePath != "/photos/a.jpg" {
		t.Fatalf("recommendedDeletePath = %q, want /photos/a.jpg", recommendedDeletePath)
	}

	side, deletePath, recommendedDeletePath, err = duplicateDeleteTarget(item, "B", "")
	if err != nil {
		t.Fatalf("duplicateDeleteTarget B error: %v", err)
	}
	if side != "B" {
		t.Fatalf("side = %q, want B", side)
	}
	if deletePath != "/photos/b.jpg" {
		t.Fatalf("deletePath = %q, want /photos/b.jpg", deletePath)
	}
	if recommendedDeletePath != "/photos/a.jpg" {
		t.Fatalf("recommendedDeletePath = %q, want /photos/a.jpg", recommendedDeletePath)
	}
}

func TestDuplicateDeleteTargetFallsBackToTargetPath(t *testing.T) {
	item := model.ScanActionItemDB{
		ActionType: model.ActionTypeDeleteDup,
		SourcePath: "/photos/a.jpg",
		TargetPath: "/photos/b.jpg",
	}

	side, deletePath, _, err := duplicateDeleteTarget(item, "B", "")
	if err != nil {
		t.Fatalf("duplicateDeleteTarget fallback error: %v", err)
	}
	if side != "B" {
		t.Fatalf("side = %q, want B", side)
	}
	if deletePath != "/photos/b.jpg" {
		t.Fatalf("deletePath = %q, want /photos/b.jpg", deletePath)
	}
}

func TestAppendDuplicateExecutionAudit(t *testing.T) {
	raw := appendDuplicateExecutionAudit(`{"keepPath":"/photos/b.jpg"}`, "B", "/photos/b.jpg", "/photos/a.jpg")

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if payload["executedDeleteSide"] != "B" {
		t.Fatalf("executedDeleteSide = %v, want B", payload["executedDeleteSide"])
	}
	if payload["executedDeletePath"] != "/photos/b.jpg" {
		t.Fatalf("executedDeletePath = %v, want /photos/b.jpg", payload["executedDeletePath"])
	}
	if payload["recommendedDeletePath"] != "/photos/a.jpg" {
		t.Fatalf("recommendedDeletePath = %v, want /photos/a.jpg", payload["recommendedDeletePath"])
	}
	if payload["manualOverride"] != true {
		t.Fatalf("manualOverride = %v, want true", payload["manualOverride"])
	}
	if payload["keepPath"] != "/photos/b.jpg" {
		t.Fatalf("keepPath = %v, want preserved", payload["keepPath"])
	}
}

func TestFormatShootDateForExif(t *testing.T) {
	got, err := formatShootDateForExif("2024-01-02")
	if err != nil {
		t.Fatalf("formatShootDateForExif error: %v", err)
	}
	if got != "2024:01:02 00:00:00" {
		t.Fatalf("got = %q, want 2024:01:02 00:00:00", got)
	}
}

func TestModifyShootTimeTargetPrefersTargetDate(t *testing.T) {
	item := model.ScanActionItemDB{
		ActionType:   model.ActionTypeModifyTime,
		SourcePath:   "/photos/a.jpg",
		MetadataJSON: `{"targetDate":"2024-01-02","minDate":"2024-01-01"}`,
	}

	targetDate, executedShootDate, err := modifyShootTimeTarget(item)
	if err != nil {
		t.Fatalf("modifyShootTimeTarget error: %v", err)
	}
	if targetDate != "2024-01-02" {
		t.Fatalf("targetDate = %q, want 2024-01-02", targetDate)
	}
	if executedShootDate != "2024:01:02 00:00:00" {
		t.Fatalf("executedShootDate = %q, want 2024:01:02 00:00:00", executedShootDate)
	}
}

func TestMoveActionTargetPrefersMetadataTargetPath(t *testing.T) {
	item := model.ScanActionItemDB{
		ActionType:   model.ActionTypeMove,
		SourcePath:   "/photos/a.jpg",
		TargetPath:   "/photos/fallback.jpg",
		MetadataJSON: `{"targetPath":"/photos/moved.jpg"}`,
	}

	targetPath, err := moveActionTarget(item)
	if err != nil {
		t.Fatalf("moveActionTarget error: %v", err)
	}
	if targetPath != "/photos/moved.jpg" {
		t.Fatalf("targetPath = %q, want /photos/moved.jpg", targetPath)
	}
}

func TestRenameActionTargetPrefersMetadataTargetPath(t *testing.T) {
	item := model.ScanActionItemDB{
		ActionType:   model.ActionTypeRename,
		SourcePath:   "/photos/a.jpg",
		TargetPath:   "/photos/fallback.jpg",
		MetadataJSON: `{"targetPath":"/photos/renamed.jpg"}`,
	}

	targetPath, err := renameActionTarget(item)
	if err != nil {
		t.Fatalf("renameActionTarget error: %v", err)
	}
	if targetPath != "/photos/renamed.jpg" {
		t.Fatalf("targetPath = %q, want /photos/renamed.jpg", targetPath)
	}
}

func TestApplySkippedRecalculationState(t *testing.T) {
	item := model.ScanActionItemDB{
		ActionType:   model.ActionTypeMove,
		MetadataJSON: `{"targetPath":"/photos/b.jpg"}`,
	}
	executedAt := time.Date(2026, 4, 15, 9, 30, 0, 0, time.Local)

	applySkippedRecalculationState(&item, executedAt)

	if item.Stage != model.ActionStageExecuted {
		t.Fatalf("stage = %q, want %q", item.Stage, model.ActionStageExecuted)
	}
	if item.Status != model.ActionStatusSkipped {
		t.Fatalf("status = %q, want %q", item.Status, model.ActionStatusSkipped)
	}
	if item.ErrorMessage != shootTimeRecalculatedMessage {
		t.Fatalf("errorMessage = %q, want %q", item.ErrorMessage, shootTimeRecalculatedMessage)
	}
	if item.ExecutedAt == nil || !item.ExecutedAt.Equal(executedAt) {
		t.Fatalf("executedAt = %v, want %v", item.ExecutedAt, executedAt)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(item.MetadataJSON), &payload); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if payload["recalculatedAfterShootTimeUpdate"] != true {
		t.Fatalf("recalculatedAfterShootTimeUpdate = %v, want true", payload["recalculatedAfterShootTimeUpdate"])
	}
	if payload["targetPath"] != "/photos/b.jpg" {
		t.Fatalf("targetPath = %v, want preserved", payload["targetPath"])
	}
}

func TestApplySkippedMoveRecalculationState(t *testing.T) {
	item := model.ScanActionItemDB{
		ActionType:   model.ActionTypeMove,
		MetadataJSON: `{"targetPath":"/photos/b.jpg"}`,
	}
	executedAt := time.Date(2026, 4, 15, 9, 35, 0, 0, time.Local)

	applySkippedMoveRecalculationState(&item, executedAt)

	if item.Stage != model.ActionStageExecuted {
		t.Fatalf("stage = %q, want %q", item.Stage, model.ActionStageExecuted)
	}
	if item.Status != model.ActionStatusSkipped {
		t.Fatalf("status = %q, want %q", item.Status, model.ActionStatusSkipped)
	}
	if item.ErrorMessage != moveRecalculatedMessage {
		t.Fatalf("errorMessage = %q, want %q", item.ErrorMessage, moveRecalculatedMessage)
	}
	if item.ExecutedAt == nil || !item.ExecutedAt.Equal(executedAt) {
		t.Fatalf("executedAt = %v, want %v", item.ExecutedAt, executedAt)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(item.MetadataJSON), &payload); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if payload["recalculatedAfterMoveUpdate"] != true {
		t.Fatalf("recalculatedAfterMoveUpdate = %v, want true", payload["recalculatedAfterMoveUpdate"])
	}
	if payload["targetPath"] != "/photos/b.jpg" {
		t.Fatalf("targetPath = %v, want preserved", payload["targetPath"])
	}
}

func TestApplySkippedRenameRecalculationState(t *testing.T) {
	item := model.ScanActionItemDB{
		ActionType:   model.ActionTypeMove,
		MetadataJSON: `{"targetPath":"/photos/b.jpg"}`,
	}
	executedAt := time.Date(2026, 4, 15, 9, 45, 0, 0, time.Local)

	applySkippedRenameRecalculationState(&item, executedAt)

	if item.Stage != model.ActionStageExecuted {
		t.Fatalf("stage = %q, want %q", item.Stage, model.ActionStageExecuted)
	}
	if item.Status != model.ActionStatusSkipped {
		t.Fatalf("status = %q, want %q", item.Status, model.ActionStatusSkipped)
	}
	if item.ErrorMessage != renameRecalculatedMessage {
		t.Fatalf("errorMessage = %q, want %q", item.ErrorMessage, renameRecalculatedMessage)
	}
	if item.ExecutedAt == nil || !item.ExecutedAt.Equal(executedAt) {
		t.Fatalf("executedAt = %v, want %v", item.ExecutedAt, executedAt)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(item.MetadataJSON), &payload); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if payload["recalculatedAfterRenameUpdate"] != true {
		t.Fatalf("recalculatedAfterRenameUpdate = %v, want true", payload["recalculatedAfterRenameUpdate"])
	}
	if payload["targetPath"] != "/photos/b.jpg" {
		t.Fatalf("targetPath = %v, want preserved", payload["targetPath"])
	}
}
