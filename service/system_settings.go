package service

import (
	"encoding/json"
	"fmt"

	"img_process/cons"
	"img_process/dao"
	"img_process/model"
	"img_process/plugin/orm"

	"gorm.io/gorm/logger"
)

var systemSettingService = dao.SystemSettingService{}

type SystemSettingSnapshot map[string]map[string]any

var readonlySystemSettingSections = []string{"database", "server"}

var editableSystemSettingSectionSet = map[string]bool{
	"scanArgs": true,
	"basic":    true,
	"cache":    true,
	"dump":     true,
	"bak":      true,
	"gis":      true,
	"batch":    true,
}

var readonlySystemSettingSectionSet = map[string]bool{
	"database": true,
	"server":   true,
}

func InitSystemSettings() error {
	if err := systemSettingService.DeleteSections(readonlySystemSettingSections); err != nil {
		return err
	}
	config, err := loadConfigFromSettings(cons.GetConfig())
	if err != nil {
		return err
	}
	if err := saveConfigToSettings(config); err != nil {
		return err
	}
	cons.ApplyConfig(config)
	applyMutableRuntimeSettings()
	return nil
}

func GetSystemSettingSnapshot(maskSecrets bool) SystemSettingSnapshot {
	return configToSnapshot(currentRuntimeConfig(), maskSecrets)
}

func ReadonlySystemSettingSections() []string {
	return append([]string(nil), readonlySystemSettingSections...)
}

func UpdateSystemSettings(snapshot SystemSettingSnapshot) (SystemSettingSnapshot, error) {
	if err := validateEditableSnapshot(snapshot); err != nil {
		return nil, err
	}
	current := currentRuntimeConfig()
	next, err := mergeSnapshotIntoConfig(current, snapshot)
	if err != nil {
		return nil, err
	}
	if err := saveConfigToSettings(next); err != nil {
		return nil, err
	}
	cons.ApplyConfig(next)
	applyMutableRuntimeSettings()
	return configToSnapshot(currentRuntimeConfig(), true), nil
}

func saveConfigToSettings(config cons.Config) error {
	settings, err := configToSettings(config)
	if err != nil {
		return err
	}
	return systemSettingService.UpsertMany(settings)
}

func loadConfigFromSettings(base cons.Config) (cons.Config, error) {
	settings, err := systemSettingService.List()
	if err != nil {
		return base, err
	}
	snapshot := SystemSettingSnapshot{}
	for _, setting := range settings {
		if snapshot[setting.Section] == nil {
			snapshot[setting.Section] = map[string]any{}
		}
		var value any
		if err := json.Unmarshal([]byte(setting.ValueJSON), &value); err != nil {
			return base, fmt.Errorf("parse system setting %s.%s: %w", setting.Section, setting.ItemKey, err)
		}
		snapshot[setting.Section][setting.ItemKey] = value
	}
	return mergeSnapshotIntoConfig(base, snapshot)
}

func configToSettings(config cons.Config) ([]model.SystemSettingDB, error) {
	snapshot := configToSnapshot(config, false)
	settings := make([]model.SystemSettingDB, 0)
	for section, fields := range snapshot {
		if !editableSystemSettingSectionSet[section] {
			continue
		}
		for key, value := range fields {
			raw, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			settings = append(settings, model.SystemSettingDB{
				Section:   section,
				ItemKey:   key,
				ValueJSON: string(raw),
				ValueType: settingValueType(value),
			})
		}
	}
	return settings, nil
}

func validateEditableSnapshot(snapshot SystemSettingSnapshot) error {
	for section := range snapshot {
		if readonlySystemSettingSectionSet[section] {
			return fmt.Errorf("%s section is read-only and must be changed in config.yaml", section)
		}
		if !editableSystemSettingSectionSet[section] {
			return fmt.Errorf("%s section is not supported", section)
		}
	}
	return nil
}

func configToSnapshot(config cons.Config, maskSecrets bool) SystemSettingSnapshot {
	secret := func(value string) string {
		if maskSecrets {
			return maskRuntimeSecret(value)
		}
		return value
	}
	return SystemSettingSnapshot{
		"database": {
			"DbUsername": config.Database.DbUsername,
			"DbPassword": secret(config.Database.DbPassword),
			"DbHost":     config.Database.DbHost,
			"DbPort":     config.Database.DbPort,
			"DbName":     config.Database.DbName,
			"DbConfig":   config.Database.DbConfig,
		},
		"server": {
			"HttpPort":     config.Server.HttpPort,
			"HttpUsername": config.Server.HttpUsername,
			"HttpPassword": secret(config.Server.HttpPassword),
		},
		"scanArgs": {
			"StartPath":        config.ScanArgs.StartPath,
			"DeleteShow":       config.ScanArgs.DeleteShow,
			"MoveFileShow":     config.ScanArgs.MoveFileShow,
			"ModifyDateShow":   config.ScanArgs.ModifyDateShow,
			"RenameFileShow":   config.ScanArgs.RenameFileShow,
			"Md5Show":          config.ScanArgs.Md5Show,
			"DeleteAction":     config.ScanArgs.DeleteAction,
			"MoveFileAction":   config.ScanArgs.MoveFileAction,
			"ModifyDateAction": config.ScanArgs.ModifyDateAction,
			"RenameFileAction": config.ScanArgs.RenameFileAction,
		},
		"basic": {
			"ColorOutput": config.Basic.ColorOutput,
			"SqlDebug":    config.Basic.SqlDebug,
		},
		"cache": {
			"ImgCache":      config.Cache.ImgCache,
			"SyncTable":     config.Cache.SyncTable,
			"TruncateTable": config.Cache.TruncateTable,
		},
		"dump": {
			"PoolSize":       config.Dump.PoolSize,
			"Md5Retry":       config.Dump.Md5Retry,
			"Md5CountLength": config.Dump.Md5CountLength,
		},
		"bak": {
			"StartPathBak": config.Bak.StartPathBak,
		},
		"gis": {
			"key": secret(config.Gis.Key),
		},
		"batch": {
			"IDInsertBatchSize": config.Batch.IDInsertBatchSize,
			"IDDeleteBatchSize": config.Batch.IDDeleteBatchSize,
			"GDUpdateBatchSize": config.Batch.GDUpdateBatchSize,
		},
	}
}

func mergeSnapshotIntoConfig(config cons.Config, snapshot SystemSettingSnapshot) (cons.Config, error) {
	var err error
	setString := func(section string, key string, current string) string {
		value, ok := snapshotValue(snapshot, section, key)
		if !ok {
			return current
		}
		next := fmt.Sprint(value)
		if isMaskedSecret(next) {
			return current
		}
		return next
	}
	setBool := func(section string, key string, current bool) bool {
		value, ok := snapshotValue(snapshot, section, key)
		if !ok {
			return current
		}
		next, parseErr := asBool(value)
		if parseErr != nil {
			err = fmt.Errorf("%s.%s must be boolean", section, key)
			return current
		}
		return next
	}
	setInt := func(section string, key string, current int) int {
		value, ok := snapshotValue(snapshot, section, key)
		if !ok {
			return current
		}
		next, parseErr := asInt(value)
		if parseErr != nil {
			err = fmt.Errorf("%s.%s must be integer", section, key)
			return current
		}
		return next
	}
	setInt64 := func(section string, key string, current int64) int64 {
		value, ok := snapshotValue(snapshot, section, key)
		if !ok {
			return current
		}
		next, parseErr := asInt64(value)
		if parseErr != nil {
			err = fmt.Errorf("%s.%s must be integer", section, key)
			return current
		}
		return next
	}

	config.ScanArgs.StartPath = setString("scanArgs", "StartPath", config.ScanArgs.StartPath)
	config.ScanArgs.DeleteShow = setBool("scanArgs", "DeleteShow", config.ScanArgs.DeleteShow)
	config.ScanArgs.MoveFileShow = setBool("scanArgs", "MoveFileShow", config.ScanArgs.MoveFileShow)
	config.ScanArgs.ModifyDateShow = setBool("scanArgs", "ModifyDateShow", config.ScanArgs.ModifyDateShow)
	config.ScanArgs.RenameFileShow = setBool("scanArgs", "RenameFileShow", config.ScanArgs.RenameFileShow)
	config.ScanArgs.Md5Show = setBool("scanArgs", "Md5Show", config.ScanArgs.Md5Show)
	config.ScanArgs.DeleteAction = setBool("scanArgs", "DeleteAction", config.ScanArgs.DeleteAction)
	config.ScanArgs.MoveFileAction = setBool("scanArgs", "MoveFileAction", config.ScanArgs.MoveFileAction)
	config.ScanArgs.ModifyDateAction = setBool("scanArgs", "ModifyDateAction", config.ScanArgs.ModifyDateAction)
	config.ScanArgs.RenameFileAction = setBool("scanArgs", "RenameFileAction", config.ScanArgs.RenameFileAction)

	config.Basic.ColorOutput = setBool("basic", "ColorOutput", config.Basic.ColorOutput)
	config.Basic.SqlDebug = setBool("basic", "SqlDebug", config.Basic.SqlDebug)

	config.Cache.ImgCache = setBool("cache", "ImgCache", config.Cache.ImgCache)
	config.Cache.SyncTable = setBool("cache", "SyncTable", config.Cache.SyncTable)
	config.Cache.TruncateTable = setBool("cache", "TruncateTable", config.Cache.TruncateTable)

	config.Dump.PoolSize = setInt("dump", "PoolSize", config.Dump.PoolSize)
	config.Dump.Md5Retry = setInt("dump", "Md5Retry", config.Dump.Md5Retry)
	config.Dump.Md5CountLength = setInt64("dump", "Md5CountLength", config.Dump.Md5CountLength)

	config.Bak.StartPathBak = setString("bak", "StartPathBak", config.Bak.StartPathBak)
	config.Gis.Key = setString("gis", "key", config.Gis.Key)

	config.Batch.IDInsertBatchSize = setInt("batch", "IDInsertBatchSize", config.Batch.IDInsertBatchSize)
	config.Batch.IDDeleteBatchSize = setInt("batch", "IDDeleteBatchSize", config.Batch.IDDeleteBatchSize)
	config.Batch.GDUpdateBatchSize = setInt("batch", "GDUpdateBatchSize", config.Batch.GDUpdateBatchSize)
	return config, err
}

func snapshotValue(snapshot SystemSettingSnapshot, section string, key string) (any, bool) {
	if snapshot == nil || snapshot[section] == nil {
		return nil, false
	}
	value, ok := snapshot[section][key]
	return value, ok
}

func settingValueType(value any) string {
	switch value.(type) {
	case bool:
		return "bool"
	case int, int64, float64:
		return "number"
	default:
		return "string"
	}
}

func isMaskedSecret(value string) bool {
	if value == "" {
		return false
	}
	runes := []rune(value)
	start := 0
	if len(runes) > 1 && runes[0] != '*' {
		start = 1
	}
	for _, r := range runes[start:] {
		if r != '*' {
			return false
		}
	}
	return true
}

func currentRuntimeConfig() cons.Config {
	config := cons.GetConfig()
	config.Database.DbUsername = cons.DbUsername
	config.Database.DbPassword = cons.DbPassword
	config.Database.DbHost = cons.DbHost
	config.Database.DbPort = cons.DbPort
	config.Database.DbName = cons.DbName
	config.Database.DbConfig = cons.DbConfig
	config.Server.HttpPort = cons.HttpPort
	config.Server.HttpUsername = cons.HttpUsername
	config.Server.HttpPassword = cons.HttpPassword
	config.ScanArgs.StartPath = cons.StartPath
	config.ScanArgs.DeleteShow = cons.DeleteShow
	config.ScanArgs.MoveFileShow = cons.MoveFileShow
	config.ScanArgs.ModifyDateShow = cons.ModifyDateShow
	config.ScanArgs.RenameFileShow = cons.RenameFileShow
	config.ScanArgs.Md5Show = cons.Md5Show
	config.ScanArgs.DeleteAction = cons.DeleteAction
	config.ScanArgs.MoveFileAction = cons.MoveFileAction
	config.ScanArgs.ModifyDateAction = cons.ModifyDateAction
	config.ScanArgs.RenameFileAction = cons.RenameFileAction
	config.Basic.ColorOutput = cons.AppConfig.Basic.ColorOutput
	config.Basic.SqlDebug = cons.SqlDebug
	config.Cache.ImgCache = cons.ImgCache
	config.Cache.SyncTable = cons.SyncTable
	config.Cache.TruncateTable = cons.TruncateTable
	config.Dump.PoolSize = cons.PoolSize
	config.Dump.Md5Retry = cons.Md5Retry
	config.Dump.Md5CountLength = cons.Md5CountLength
	config.Bak.StartPathBak = cons.StartPathBak
	config.Gis.Key = cons.GisKey
	config.Batch.IDInsertBatchSize = cons.IDInsertBatchSize
	config.Batch.IDDeleteBatchSize = cons.IDDeleteBatchSize
	config.Batch.GDUpdateBatchSize = cons.GDUpdateBatchSize
	return config
}

func asBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		if v == "true" {
			return true, nil
		}
		if v == "false" {
			return false, nil
		}
	}
	return false, fmt.Errorf("invalid bool")
}

func asInt(value any) (int, error) {
	next, err := asInt64(value)
	return int(next), err
}

func asInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case json.Number:
		return v.Int64()
	}
	return 0, fmt.Errorf("invalid integer")
}

func applyMutableRuntimeSettings() {
	if orm.ImgMysqlDB != nil {
		logMode := logger.Silent
		if cons.SqlDebug {
			logMode = logger.Info
		}
		orm.ImgMysqlDB.Logger = logger.Default.LogMode(logMode)
	}
}
