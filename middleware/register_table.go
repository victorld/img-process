package middleware

import (
	"img_process/dao"
	"img_process/model"
	"img_process/tools"
)

var imgDatabaseService = dao.ImgDatabaseService{}
var imgRecordService = dao.ImgRecordService{}
var gisDatabaseService = dao.GisDatabaseService{}

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
}
