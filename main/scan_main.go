package main

import (
	"img_process/cons"
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

	startPath := &cons.StartPath       //统计的起始目录，必须包含pic-new
	StartPathBak := &cons.StartPathBak //备份目录

	deleteShow := &cons.DeleteShow         //是否统计并显示非法文件和空目录
	moveFileShow := &cons.MoveFileShow     //是否统计并显示需要移动目录的文件
	modifyDateShow := &cons.ModifyDateShow //是否统计并显示需要修改日期的文件
	md5Show := &cons.Md5Show               //是否统计并显示重复文件

	deleteAction := &cons.DeleteAction         //是否操作删除非法文件和空目录
	moveFileAction := &cons.MoveFileAction     //是否操作需要移动目录的文件
	modifyDateAction := &cons.ModifyDateAction //是否操作修改日期的文件

	scanArgs := model.DoScanImgArg{DeleteShow: deleteShow, MoveFileShow: moveFileShow, ModifyDateShow: modifyDateShow, Md5Show: md5Show, DeleteAction: deleteAction, MoveFileAction: moveFileAction, ModifyDateAction: modifyDateAction, StartPath: startPath, StartPathBak: StartPathBak}

	service.ScanAndSave(scanArgs)

}
