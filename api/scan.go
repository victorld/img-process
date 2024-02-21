package api

import (
	"github.com/gin-gonic/gin"
	"img_process/cons"
	"img_process/model"
	"img_process/service"
	"img_process/tools"
)

type ImgRecordOwnApi struct {
}

// DoScanImg 执行扫描
// @Tags ImgRecord
// @Summary 执行扫描
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data query DoScanImgArg true "扫描参数"
// @Success 200 {string} string "{"success":true,"data":{},"msg":"创建成功"}"
// @Router /imgRecord/doScanImg [get]
func (imgRecordOwnApi *ImgRecordOwnApi) DoScanImg(c *gin.Context) {
	var doScanImgArg model.DoScanImgArg
	err := c.ShouldBindQuery(&doScanImgArg)
	if err != nil {
		tools.Fail(c, "绑定参数不对", nil)
		return
	}

	tools.Logger.Info("DoScanImg args : " + tools.MarshalJsonToString(doScanImgArg))
	if doScanImgArg.StartPath == nil {
		doScanImgArg.StartPath = &cons.StartPath
	}
	if doScanImgArg.DeleteShow == nil {
		doScanImgArg.DeleteShow = &cons.DeleteShow
	}
	if doScanImgArg.MoveFileShow == nil {
		doScanImgArg.MoveFileShow = &cons.MoveFileShow
	}
	if doScanImgArg.ModifyDateShow == nil {
		doScanImgArg.ModifyDateShow = &cons.ModifyDateShow
	}
	if doScanImgArg.Md5Show == nil {
		doScanImgArg.Md5Show = &cons.Md5Show
	}
	if doScanImgArg.DeleteAction == nil {
		doScanImgArg.DeleteAction = &cons.DeleteAction
	}
	if doScanImgArg.MoveFileAction == nil {
		doScanImgArg.MoveFileAction = &cons.MoveFileAction
	}
	if doScanImgArg.ModifyDateAction == nil {
		doScanImgArg.ModifyDateAction = &cons.ModifyDateAction
	}

	tools.Logger.Info("DoScanImg args real: " + tools.MarshalJsonToString(doScanImgArg))

	go func() {

		service.ScanAndSave(doScanImgArg)

	}()

	tools.Logger.Info("DoScanImg ret ok")
	tools.Success(c, gin.H{"ret": "ok"}, "成功")

}

func (imgRecordOwnApi *ImgRecordOwnApi) DeleteMD5DupFiles(c *gin.Context) {

	md5, ok := c.GetQuery("md5")
	if ok {
		filePath := "/tmp/" + md5
		tools.Logger.Info("file path : ", filePath)

		service.DeleteMD5DupFilesByJson(filePath)
	}
}
