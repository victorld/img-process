package service

import (
	"testing"

	"img_process/cons"
)

func TestConfigToSettingsOnlyIncludesEditableSections(t *testing.T) {
	config := cons.WithRuntimeDefaults(cons.Config{})
	config.Database.DbUsername = "root"
	config.Server.HttpPort = "8081"

	settings, err := configToSettings(config)
	if err != nil {
		t.Fatalf("configToSettings error: %v", err)
	}
	if len(settings) == 0 {
		t.Fatal("settings is empty")
	}
	for _, setting := range settings {
		if setting.Section == "database" || setting.Section == "server" {
			t.Fatalf("readonly section %s should not be persisted", setting.Section)
		}
		if !editableSystemSettingSectionSet[setting.Section] {
			t.Fatalf("unexpected section persisted: %s", setting.Section)
		}
	}
}

func TestValidateEditableSnapshotRejectsReadonlySections(t *testing.T) {
	err := validateEditableSnapshot(SystemSettingSnapshot{
		"database": {"DbHost": "127.0.0.1"},
	})
	if err == nil {
		t.Fatal("expected readonly section error")
	}
}

func TestMergeSnapshotDoesNotOverrideFileOwnedSections(t *testing.T) {
	config := cons.WithRuntimeDefaults(cons.Config{})
	config.Database.DbHost = "file-db"
	config.Server.HttpPort = "8081"
	config.ScanArgs.StartPath = "/old"

	next, err := mergeSnapshotIntoConfig(config, SystemSettingSnapshot{
		"database": {"DbHost": "db-from-payload"},
		"server":   {"HttpPort": "9090"},
		"scanArgs": {"StartPath": "/new"},
	})
	if err != nil {
		t.Fatalf("mergeSnapshotIntoConfig error: %v", err)
	}
	if next.Database.DbHost != "file-db" {
		t.Fatalf("DbHost = %s, want file-db", next.Database.DbHost)
	}
	if next.Server.HttpPort != "8081" {
		t.Fatalf("HttpPort = %s, want 8081", next.Server.HttpPort)
	}
	if next.ScanArgs.StartPath != "/new" {
		t.Fatalf("StartPath = %s, want /new", next.ScanArgs.StartPath)
	}
}
