package dao

import (
	"testing"

	"img_process/model"
)

func TestCountRowsByTypeSeparatesDeleteEmptyDir(t *testing.T) {
	rows := []actionCountRow{
		{ActionType: model.ActionTypeDelete, Total: 2},
		{ActionType: model.ActionTypeDeleteEmptyDir, Total: 3},
		{ActionType: model.ActionTypeRename, Total: 1},
	}

	counts := countRowsByType(rows)
	if counts.Delete != 2 {
		t.Fatalf("delete counts = %d, want 2", counts.Delete)
	}
	if counts.DeleteEmptyDir != 3 {
		t.Fatalf("delete empty dir counts = %d, want 3", counts.DeleteEmptyDir)
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
		{Stage: model.ActionStageCandidate, Status: model.ActionStatusPending, ActionType: model.ActionTypeDeleteEmptyDir, Total: 4},
		{Stage: model.ActionStageExecuted, Status: model.ActionStatusSucceeded, ActionType: model.ActionTypeDeleteEmptyDir, Total: 5},
		{Stage: model.ActionStageExecuted, Status: model.ActionStatusFailed, ActionType: model.ActionTypeDeleteEmptyDir, Total: 6},
	}

	grouped := model.ScanActionGroupedCounts{}
	for _, row := range rows {
		addGroupedActionCount(&grouped, row)
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
	if grouped.Pending.DeleteEmptyDir != 4 {
		t.Fatalf("pending delete empty dir = %d, want 4", grouped.Pending.DeleteEmptyDir)
	}
	if grouped.Executed.DeleteEmptyDir != 11 {
		t.Fatalf("executed delete empty dir = %d, want 11", grouped.Executed.DeleteEmptyDir)
	}
	if grouped.Error.DeleteEmptyDir != 6 {
		t.Fatalf("error delete empty dir = %d, want 6", grouped.Error.DeleteEmptyDir)
	}
}

func TestGroupedCountsIgnoreDiscoveryDuplicateAsPending(t *testing.T) {
	rows := []actionCountRow{
		{Stage: model.ActionStageDiscovery, Status: model.ActionStatusSkipped, ActionType: model.ActionTypeDeleteDup, Total: 2},
		{Stage: model.ActionStageCandidate, Status: model.ActionStatusPending, ActionType: model.ActionTypeDeleteDup, Total: 1},
	}

	grouped := model.ScanActionGroupedCounts{}
	for _, row := range rows {
		addGroupedActionCount(&grouped, row)
	}

	if grouped.Pending.DeleteDuplicate != 1 {
		t.Fatalf("pending duplicate = %d, want 1", grouped.Pending.DeleteDuplicate)
	}
	if grouped.Pending.Total != 1 {
		t.Fatalf("pending total = %d, want 1", grouped.Pending.Total)
	}
}
