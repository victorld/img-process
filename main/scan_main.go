package main

import (
	"img_process/cons"
	"img_process/middleware"
	"img_process/model"
	"img_process/plugin/orm"
	"img_process/service"
	"img_process/tools"
)

// 扫描主入口
func ScanMain() {

	tools.InitLogger()
	tools.InitViper()
	cons.InitConst()
	orm.InitMysql()

	scanArgs := model.DoScanImgArg{DeleteShow: nil, MoveFileShow: nil, ModifyDateShow: nil, RenameFileShow: nil, Md5Show: nil, DeleteAction: nil, MoveFileAction: nil, ModifyDateAction: nil, RenameFileAction: nil, StartPath: nil, StartPathBak: nil}
	tools.Logger.Info("DoScanImg main args : " + tools.MarshalJsonToString(scanArgs))

	middleware.RegisterTable()

	service.ScanAndSave(scanArgs)

}
