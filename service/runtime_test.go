package service

import (
	"testing"

	"img_process/cons"
	"img_process/model"
)

func TestNormalizeScanArgsUsesConfigDefaults(t *testing.T) {
	cons.StartPath = "/scan"
	cons.StartPathBak = "/scan-bak"
	cons.DeleteShow = true
	cons.MoveFileShow = true
	cons.ModifyDateShow = false
	cons.RenameFileShow = true
	cons.Md5Show = true
	cons.DeleteAction = false
	cons.MoveFileAction = false
	cons.ModifyDateAction = false
	cons.RenameFileAction = false

	args := NormalizeScanArgs(model.DoScanImgArg{})
	if args.StartPath == nil || *args.StartPath != "/scan" {
		t.Fatalf("StartPath = %v, want /scan", args.StartPath)
	}
	if args.StartPathBak == nil || *args.StartPathBak != "/scan-bak" {
		t.Fatalf("StartPathBak = %v, want /scan-bak", args.StartPathBak)
	}
	if args.Md5Show == nil || !*args.Md5Show {
		t.Fatalf("Md5Show = %v, want true", args.Md5Show)
	}
}

func TestBuildCronExpr(t *testing.T) {
	expr, err := buildCronExpr(model.ScheduleModeDaily, scheduleConfig{Hour: 2, Minute: 30})
	if err != nil {
		t.Fatalf("buildCronExpr daily error: %v", err)
	}
	if expr != "30 2 * * *" {
		t.Fatalf("daily expr = %s", expr)
	}

	expr, err = buildCronExpr(model.ScheduleModeWeekly, scheduleConfig{Hour: 3, Minute: 15, Weekdays: []int{1, 5}})
	if err != nil {
		t.Fatalf("buildCronExpr weekly error: %v", err)
	}
	if expr != "15 3 * * 1,5" {
		t.Fatalf("weekly expr = %s", expr)
	}
}
