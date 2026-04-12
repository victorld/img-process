package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"img_process/cons"
	"img_process/model"
	"img_process/service"
	"img_process/tools"
	"net/http"
	"path/filepath"
	"regexp"
	"sync"
)

type ImgRecordOwnApi struct {
}

var (
	scanMu             sync.Mutex //processFileList锁，保证只有一个后台扫描任务执行
	scanAndSaveFunc    = service.ScanAndSave
	deleteDumpFileFunc = service.DeleteMD5DupFilesByLine
	scanUUIDPattern    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}_[A-Za-z0-9]+$`)
)

type deleteReq struct {
	ScanUUID string `json:"scanUuid" form:"scanUuid"`
}

// DoScanImg 执行扫描
func (imgRecordOwnApi *ImgRecordOwnApi) DoScanImg(c *gin.Context) {
	var doScanImgArg model.DoScanImgArg
	if err := bindDoScanImgArg(c, &doScanImgArg); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "绑定参数不对", gin.H{"error": err.Error()})
		return
	}

	tools.Logger.Info("DoScanImg web args : " + tools.MarshalJsonToString(doScanImgArg))

	if scanMu.TryLock() {
		go func() {
			defer scanMu.Unlock()
			tools.Logger.Info("扫描开始")
			imgRecordString, err := scanAndSaveFunc(doScanImgArg)
			if err != nil {
				tools.FancyHandleError(err)
				return
			}
			tools.Logger.Info("扫描结束，结果：", imgRecordString)
		}()

		tools.Logger.Info("DoScanImg ret accepted")
		tools.SuccessWithStatus(c, http.StatusAccepted, gin.H{"ret": "ok"}, "扫描任务下发成功，请稍后检查数据库记录")
		return
	}

	tools.Logger.Info("DoScanImg processing, exit")
	tools.FailWithStatus(c, http.StatusConflict, "扫描进行中，请等待扫描结束", gin.H{"ret": "not ok"})
}

// DeleteMD5DupFiles 删除重复文件
func (imgRecordOwnApi *ImgRecordOwnApi) DeleteMD5DupFiles(c *gin.Context) {
	var req deleteReq
	if err := bindDeleteReq(c, &req); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "scanUuid 不能为空", gin.H{"error": err.Error()})
		return
	}

	filePath, err := resolveDeleteListPath(req.ScanUUID)
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, err.Error(), gin.H{"error": err.Error()})
		return
	}
	tools.Logger.Info("file path : ", filePath)
	deleteDumpFileFunc(filePath)
	tools.Success(c, gin.H{"ret": "ok"}, "删除任务执行完成")
}

func bindDoScanImgArg(c *gin.Context, doScanImgArg *model.DoScanImgArg) error {
	if c.Request.Method == http.MethodPost {
		if c.Request.ContentLength > 0 {
			return c.ShouldBindJSON(doScanImgArg)
		}
	}
	return c.ShouldBindQuery(doScanImgArg)
}

func bindDeleteReq(c *gin.Context, req interface{}) error {
	if c.Request.Method == http.MethodDelete || c.Request.Method == http.MethodPost {
		if c.Request.ContentLength > 0 {
			if err := c.ShouldBindJSON(req); err != nil {
				return err
			}
			return nil
		}
	}
	if err := c.ShouldBindQuery(req); err != nil {
		return err
	}

	switch v := req.(type) {
	case *deleteReq:
		if v.ScanUUID == "" {
			return errors.New("missing scanUuid")
		}
		if !scanUUIDPattern.MatchString(v.ScanUUID) {
			return errors.New("invalid scanUuid")
		}
	}

	return nil
}

func resolveDeleteListPath(scanUUID string) (string, error) {
	root := filepath.Join(cons.WorkDir, "log", "dump_delete_file")
	target := filepath.Join(root, scanUUID, "dump_delete_list")
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
		return "", errors.New("invalid scanUuid path")
	}
	return target, nil
}
