package main

import (
	"fmt"
	"img_process/cons"
	"img_process/dao"
	"img_process/middleware"
	"img_process/model"
	"img_process/plugin/orm"
	"img_process/tools"
)

var gisDatabaseService = dao.GisDatabaseService{}

// 从img_database库里的json字段提取地址信息单独存储
func main() {
	tools.InitLogger()
	if err := tools.InitViper(); err != nil {
		tools.Logger.Fatal("init viper error : ", err)
	}
	cons.InitConst()
	if err := orm.InitMysql(); err != nil {
		tools.Logger.Fatal("init mysql error : ", err)
	}

	var gisDatabaseDB model.GisDatabaseDB
	if err := gisDatabaseService.RegisterGisDatabase(&gisDatabaseDB); err != nil {
		tools.Logger.Error("register error : ", err)
		return
	}

	var gisDatabaseSearch model.GisDatabaseSearch
	list, _, err := gisDatabaseService.GetGisDatabaseInfoList(gisDatabaseSearch)
	if err != nil {

	}
	for i := range list {

		locJson := list[i].LocJson

		gisData, err := middleware.GetGisDataFromJson(locJson)
		if err != nil {
			tools.Logger.Warn("parse gis json failed : ", err)
			continue
		}

		list[i].LocStreet = gisData.LocStreet

		list[i].LocAddr = gisData.LocAddr

	}

	fmt.Println()

	gisDatabaseService.UpdateGisDatabaseBatch(list, cons.GDUpdateBatchSize)

}
