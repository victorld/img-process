package dao

import (
	"testing"

	"img_process/model"
)

func TestCountRowsByTypeTreatsDeleteEmptyDirAsDelete(t *testing.T) {
	rows := []actionCountRow{
		{ActionType: model.ActionTypeDelete, Total: 2},
		{ActionType: model.ActionTypeDeleteEmptyDir, Total: 3},
		{ActionType: model.ActionTypeRename, Total: 1},
	}

	counts := countRowsByType(rows)
	if counts.Delete != 5 {
		t.Fatalf("delete counts = %d, want 5", counts.Delete)
	}
	if counts.Rename != 1 {
		t.Fatalf("rename counts = %d, want 1", counts.Rename)
	}
	if counts.Total != 6 {
		t.Fatalf("total counts = %d, want 6", counts.Total)
	}
}

func TestCountGroupedRowsSeparatesPendingExecutedAndError(t *testing.T) {
	rows := []actionCountRow{
		{Stage: model.ActionStageCandidate, Status: model.ActionStatusPending, ActionType: model.ActionTypeRename, Total: 1},
		{Stage: model.ActionStageExecuted, Status: model.ActionStatusSucceeded, ActionType: model.ActionTypeRename, Total: 2},
		{Stage: model.ActionStageExecuted, Status: model.ActionStatusFailed, ActionType: model.ActionTypeRename, Total: 3},
	}

	grouped := model.ScanActionGroupedCounts{}
	for _, row := range rows {
		if row.Stage == model.ActionStageCandidate && row.Status == model.ActionStatusPending {
			addActionCount(&grouped.Pending, row.ActionType, row.Total)
		}
		if row.Stage == model.ActionStageExecuted {
			addActionCount(&grouped.Executed, row.ActionType, row.Total)
		}
		if row.Status == model.ActionStatusFailed {
			addActionCount(&grouped.Error, row.ActionType, row.Total)
		}
	}

	if grouped.Pending.Rename != 1 {
		t.Fatalf("pending rename = %d, want 1", grouped.Pending.Rename)
	}
	if grouped.Executed.Rename != 5 {
		t.Fatalf("executed rename = %d, want 5", grouped.Executed.Rename)
	}
	if grouped.Error.Rename != 3 {
		t.Fatalf("error rename = %d, want 3", grouped.Error.Rename)
	}
}
