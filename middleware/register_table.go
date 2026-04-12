package middleware

import (
	"img_process/dao"
	"img_process/model"
	"img_process/tools"
)

var imgDatabaseService = dao.ImgDatabaseService{}
var imgRecordService = dao.ImgRecordService{}
var gisDatabaseService = dao.GisDatabaseService{}
var scanJobService = dao.ScanJobService{}
var scanActionItemService = dao.ScanActionItemService{}
var scanEventService = dao.ScanEventService{}
var scanScheduleService = dao.ScanScheduleService{}

// RegisterTable 根据gorm配置同步表结构
func RegisterTable() {
	var err error
	var imgRecordDB model.ImgRecordDB
	if err = imgRecordService.RegisterImgRecord(&imgRecordDB); err != nil {
		tools.Logger.Error("register error : ", err)
		return
	}

	var gisDatabaseDB model.GisDatabaseDB
	if err := gisDatabaseService.RegisterGisDatabase(&gisDatabaseDB); err != nil {
		tools.Logger.Error("register error : ", err)
		return
	}

	var imgDatabaseDB model.ImgDatabaseDB
	if err = imgDatabaseService.RegisterImgDatabase(&imgDatabaseDB); err != nil {
		tools.Logger.Error("register error : ", err)
		return
	}

	var scanJobDB model.ScanJobDB
	if err = scanJobService.RegisterScanJob(&scanJobDB); err != nil {
		tools.Logger.Error("register error : ", err)
		return
	}

	var scanActionItemDB model.ScanActionItemDB
	if err = scanActionItemService.RegisterScanActionItem(&scanActionItemDB); err != nil {
		tools.Logger.Error("register error : ", err)
		return
	}

	var scanEventDB model.ScanEventDB
	if err = scanEventService.RegisterScanEvent(&scanEventDB); err != nil {
		tools.Logger.Error("register error : ", err)
		return
	}

	var scanScheduleDB model.ScanScheduleDB
	if err = scanScheduleService.RegisterScanSchedule(&scanScheduleDB); err != nil {
		tools.Logger.Error("register error : ", err)
		return
	}
}
