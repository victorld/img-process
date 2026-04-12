package main

import (
	"img_process/bootstrap"
	"img_process/middleware"
	"img_process/model"
	"img_process/service"
	"img_process/tools"
)

// 扫描主入口
func main() {
	closeFn, err := bootstrap.InitApp(true)
	if err != nil {
		tools.Logger.Fatal("bootstrap init error : ", err)
	}
	defer closeFn()

	scanArgs := model.DoScanImgArg{DeleteShow: nil, MoveFileShow: nil, ModifyDateShow: nil, RenameFileShow: nil, Md5Show: nil, DeleteAction: nil, MoveFileAction: nil, ModifyDateAction: nil, RenameFileAction: nil, StartPath: nil, StartPathBak: nil}
	tools.Logger.Info("DoScanImg main args : " + tools.MarshalJsonToString(scanArgs))

	middleware.RegisterTable()

	if _, err := service.ScanAndSave(scanArgs); err != nil {
		tools.Logger.Error("scan main error : ", err)
	}

}
