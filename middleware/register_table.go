package middleware

import (
	"img_process/dao"
	"img_process/model"
)

var imgDatabaseService = dao.ImgDatabaseService{}
var imgRecordService = dao.ImgRecordService{}
var gisDatabaseService = dao.GisDatabaseService{}
var scanJobService = dao.ScanJobService{}
var scanActionItemService = dao.ScanActionItemService{}
var scanEventService = dao.ScanEventService{}
var scanJobLogService = dao.ScanJobLogService{}
var scanScheduleService = dao.ScanScheduleService{}
var systemSettingService = dao.SystemSettingService{}

// RegisterTable 根据gorm配置同步表结构
func RegisterTable() error {
	var imgRecordDB model.ImgRecordDB
	if err := imgRecordService.RegisterImgRecord(&imgRecordDB); err != nil {
		return err
	}

	var gisDatabaseDB model.GisDatabaseDB
	if err := gisDatabaseService.RegisterGisDatabase(&gisDatabaseDB); err != nil {
		return err
	}

	var imgDatabaseDB model.ImgDatabaseDB
	if err := imgDatabaseService.RegisterImgDatabase(&imgDatabaseDB); err != nil {
		return err
	}

	var scanJobDB model.ScanJobDB
	if err := scanJobService.RegisterScanJob(&scanJobDB); err != nil {
		return err
	}

	var scanActionItemDB model.ScanActionItemDB
	if err := scanActionItemService.RegisterScanActionItem(&scanActionItemDB); err != nil {
		return err
	}

	var scanEventDB model.ScanEventDB
	if err := scanEventService.RegisterScanEvent(&scanEventDB); err != nil {
		return err
	}

	var scanJobLogDB model.ScanJobLogDB
	if err := scanJobLogService.RegisterScanJobLog(&scanJobLogDB); err != nil {
		return err
	}

	var scanScheduleDB model.ScanScheduleDB
	if err := scanScheduleService.RegisterScanSchedule(&scanScheduleDB); err != nil {
		return err
	}

	var systemSettingDB model.SystemSettingDB
	if err := systemSettingService.RegisterSystemSetting(&systemSettingDB); err != nil {
		return err
	}
	return nil
}
