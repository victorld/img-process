package service

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestBuildScanExecutionSnapshotIncludesMaskedSystemConfig(t *testing.T) {
	oldDbUsername := cons.DbUsername
	oldDbPassword := cons.DbPassword
	oldDbHost := cons.DbHost
	oldDbPort := cons.DbPort
	oldDbName := cons.DbName
	oldDbConfig := cons.DbConfig
	oldHttpPort := cons.HttpPort
	oldHttpUsername := cons.HttpUsername
	oldHttpPassword := cons.HttpPassword
	oldStartPathBak := cons.StartPathBak
	oldImgCache := cons.ImgCache
	oldSyncTable := cons.SyncTable
	oldTruncateTable := cons.TruncateTable
	oldSqlDebug := cons.SqlDebug
	oldPoolSize := cons.PoolSize
	oldMd5Retry := cons.Md5Retry
	oldMd5CountLength := cons.Md5CountLength
	oldGisKey := cons.GisKey
	oldIDInsertBatchSize := cons.IDInsertBatchSize
	oldIDDeleteBatchSize := cons.IDDeleteBatchSize
	oldGDUpdateBatchSize := cons.GDUpdateBatchSize
	oldAppConfig := cons.AppConfig
	t.Cleanup(func() {
		cons.DbUsername = oldDbUsername
		cons.DbPassword = oldDbPassword
		cons.DbHost = oldDbHost
		cons.DbPort = oldDbPort
		cons.DbName = oldDbName
		cons.DbConfig = oldDbConfig
		cons.HttpPort = oldHttpPort
		cons.HttpUsername = oldHttpUsername
		cons.HttpPassword = oldHttpPassword
		cons.StartPathBak = oldStartPathBak
		cons.ImgCache = oldImgCache
		cons.SyncTable = oldSyncTable
		cons.TruncateTable = oldTruncateTable
		cons.SqlDebug = oldSqlDebug
		cons.PoolSize = oldPoolSize
		cons.Md5Retry = oldMd5Retry
		cons.Md5CountLength = oldMd5CountLength
		cons.GisKey = oldGisKey
		cons.IDInsertBatchSize = oldIDInsertBatchSize
		cons.IDDeleteBatchSize = oldIDDeleteBatchSize
		cons.GDUpdateBatchSize = oldGDUpdateBatchSize
		cons.AppConfig = oldAppConfig
	})

	cons.DbUsername = "root"
	cons.DbPassword = "secret"
	cons.DbHost = "db-host"
	cons.DbPort = "3306"
	cons.DbName = "img_process"
	cons.DbConfig = "charset=utf8mb4"
	cons.HttpPort = "8081"
	cons.HttpUsername = "admin"
	cons.HttpPassword = "adminpass"
	cons.StartPathBak = "/backup"
	cons.ImgCache = true
	cons.SyncTable = true
	cons.TruncateTable = false
	cons.SqlDebug = true
	cons.PoolSize = 6
	cons.Md5Retry = 3
	cons.Md5CountLength = 4096
	cons.GisKey = "gis-secret"
	cons.IDInsertBatchSize = 1000
	cons.IDDeleteBatchSize = 300
	cons.GDUpdateBatchSize = 500
	cons.AppConfig.Basic.ColorOutput = true

	startPath := "/photos"
	deleteShow := true
	snapshot := buildScanExecutionSnapshot(model.DoScanImgArg{
		StartPath:    &startPath,
		StartPathBak: &cons.StartPathBak,
		DeleteShow:   &deleteShow,
	})

	if snapshot["startPath"] != "/photos" {
		t.Fatalf("startPath = %v, want /photos", snapshot["startPath"])
	}
	if snapshot["startPathBak"] != "/backup" {
		t.Fatalf("startPathBak = %v, want /backup", snapshot["startPathBak"])
	}
	if snapshot["DbHost"] != "db-host" {
		t.Fatalf("DbHost = %v, want db-host", snapshot["DbHost"])
	}
	if snapshot["DbPassword"] != "s*****" {
		t.Fatalf("DbPassword = %v, want s*****", snapshot["DbPassword"])
	}
	if snapshot["HttpPassword"] != "a********" {
		t.Fatalf("HttpPassword = %v, want a********", snapshot["HttpPassword"])
	}
	if snapshot["key"] != "g*********" {
		t.Fatalf("key = %v, want g*********", snapshot["key"])
	}
	if snapshot["PoolSize"] != 6 {
		t.Fatalf("PoolSize = %v, want 6", snapshot["PoolSize"])
	}
	if snapshot["ColorOutput"] != true {
		t.Fatalf("ColorOutput = %v, want true", snapshot["ColorOutput"])
	}
}

func TestScanExecutionSnapshotCanDecodeAsScanArgs(t *testing.T) {
	startPath := "/photos"
	md5Show := true
	snapshot := buildScanExecutionSnapshot(model.DoScanImgArg{
		StartPath: &startPath,
		Md5Show:   &md5Show,
	})

	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	var decoded model.DoScanImgArg
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal scan args: %v", err)
	}
	if decoded.StartPath == nil || *decoded.StartPath != "/photos" {
		t.Fatalf("StartPath = %v, want /photos", decoded.StartPath)
	}
	if decoded.Md5Show == nil || !*decoded.Md5Show {
		t.Fatalf("Md5Show = %v, want true", decoded.Md5Show)
	}
}

func TestBuildCronExpr(t *testing.T) {
	expr, err := buildCronExpr(model.ScheduleModeHourly, scheduleConfig{Minute: 10})
	if err != nil {
		t.Fatalf("buildCronExpr hourly error: %v", err)
	}
	if expr != "10 * * * *" {
		t.Fatalf("hourly expr = %s", expr)
	}

	expr, err = buildCronExpr(model.ScheduleModeDaily, scheduleConfig{Hour: 2, Minute: 30})
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

func TestBuildScheduleModelUsesEnvironmentTimezone(t *testing.T) {
	t.Setenv("TZ", "Asia/Shanghai")
	root := t.TempDir()
	cons.StartPath = root
	cons.StartPathBak = ""
	req := model.UpsertScheduleReq{
		Name:           "env timezone",
		Enabled:        true,
		Mode:           model.ScheduleModeHourly,
		ScheduleConfig: `{"minute":5}`,
		ScanArgs: model.DoScanImgArg{
			StartPath: &root,
		},
	}

	schedule, err := buildScheduleModel(req)
	if err != nil {
		t.Fatalf("buildScheduleModel error: %v", err)
	}
	if schedule.Timezone != "Asia/Shanghai" {
		t.Fatalf("Timezone = %s, want Asia/Shanghai", schedule.Timezone)
	}
}

func TestValidateScanArgsRejectsMissingStartPath(t *testing.T) {
	startPath := filepath.Join(t.TempDir(), "missing")
	err := validateScanArgs(model.DoScanImgArg{StartPath: &startPath})
	if err == nil {
		t.Fatal("validateScanArgs should reject missing path")
	}
}

func TestValidateScanArgsAcceptsExistingDirectories(t *testing.T) {
	root := t.TempDir()
	backup := filepath.Join(root, "backup")
	if err := os.MkdirAll(backup, 0o755); err != nil {
		t.Fatalf("mkdir backup: %v", err)
	}

	err := validateScanArgs(model.DoScanImgArg{
		StartPath:    &root,
		StartPathBak: &backup,
	})
	if err != nil {
		t.Fatalf("validateScanArgs returned error: %v", err)
	}
}
