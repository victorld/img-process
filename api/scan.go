package api

import (
	"github.com/gin-gonic/gin"
	"img_process/cons"
	"img_process/model"
	"img_process/service"
	"img_process/tools"
	"sync"
)

type ImgRecordOwnApi struct {
}

var scanMu sync.Mutex //processFileList锁，保证只有一个后台扫描任务执行

// DoScanImg 执行扫描
func (imgRecordOwnApi *ImgRecordOwnApi) DoScanImg(c *gin.Context) {
	var doScanImgArg model.DoScanImgArg
	err := c.ShouldBindQuery(&doScanImgArg)
	if err != nil {
		tools.Fail(c, "绑定参数不对", nil)
		return
	}

	tools.Logger.Info("DoScanImg web args : " + tools.MarshalJsonToString(doScanImgArg))

	if scanMu.TryLock() {
		go func() {
			var imgRecordString string
			tools.Logger.Info("扫描开始")
			imgRecordString, err = service.ScanAndSave(doScanImgArg)
			if err != nil {
				tools.FancyHandleError(err)
			} else {
				tools.Logger.Info("扫描结束，结果：", imgRecordString)
			}

			scanMu.Unlock()
		}()

		tools.Logger.Info("DoScanImg ret ok")
		tools.Success(c, gin.H{"ret": "ok"}, "扫描任务下发成功，请稍后检查数据库记录")
	} else {
		tools.Logger.Info("DoScanImg processing, exit")
		tools.Success(c, gin.H{"ret": "not ok"}, "扫描进行中，请等待扫描结束")
	}

}

// DeleteMD5DupFiles 删除重复文件
func (imgRecordOwnApi *ImgRecordOwnApi) DeleteMD5DupFiles(c *gin.Context) {

	scanUuidFinal, ok := c.GetQuery("scanUuid")
	if ok {

		filePath := cons.WorkDir + "/log/dump_delete_file/" + scanUuidFinal + "/dump_delete_list"
		tools.Logger.Info("file path : ", filePath)

		service.DeleteMD5DupFilesByLine(filePath)
	}
}
