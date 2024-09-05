package main

import (
	"img_process/cons"
	"img_process/middleware"
	"img_process/model"
	"img_process/plugin/orm"
	"img_process/service"
	"img_process/tools"
)

func main() {

	tools.InitLogger()
	tools.InitViper()
	cons.InitConst()
	orm.InitMysql()

	var startPath = ""
	var startPathBak = ""

	var deleteShow = true      //是否统计并显示非法文件和空目录
	var moveFileShow = true    //是否统计并显示需要移动目录的文件
	var modifyDateShow = false //是否统计并显示需要修改日期的文件
	var md5Show = true         //是否统计并显示重复文件

	var deleteAction = false     //是否操作删除非法文件和空目录
	var moveFileAction = false   //是否操作需要移动目录的文件
	var modifyDateAction = false //是否操作修改日期的文件

	scanArgs := model.DoScanImgArg{DeleteShow: &deleteShow, MoveFileShow: &moveFileShow, ModifyDateShow: &modifyDateShow, Md5Show: &md5Show, DeleteAction: &deleteAction, MoveFileAction: &moveFileAction, ModifyDateAction: &modifyDateAction, StartPath: &startPath, StartPathBak: &startPathBak}
	tools.Logger.Info("DoScanImg main args : " + tools.MarshalJsonToString(scanArgs))

	middleware.RegisterTable()

	service.ScanAndSave(scanArgs)

}
