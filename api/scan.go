package api

import (
	"errors"
	"net/http"
	"path/filepath"
	"regexp"

	"github.com/gin-gonic/gin"

	"img_process/cons"
	"img_process/model"
	"img_process/service"
	"img_process/tools"
)

type ImgRecordOwnApi struct{}

var (
	deleteDumpFileFunc = service.DeleteMD5DupFilesByLine
	createJobFunc      = service.Runtime.CreateJob
	scanUUIDPattern    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}_[A-Za-z0-9]+$`)
)

type deleteReq struct {
	ScanUUID string `json:"scanUuid" form:"scanUuid"`
}

// DoScanImg 兼容旧接口，改为创建异步任务
func (imgRecordOwnApi *ImgRecordOwnApi) DoScanImg(c *gin.Context) {
	var doScanImgArg model.DoScanImgArg
	if err := bindDoScanImgArg(c, &doScanImgArg); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "绑定参数不对", gin.H{"error": err.Error()})
		return
	}

	job, err := createJobFunc(model.JobSourceManual, nil, doScanImgArg)
	if err != nil {
		tools.Fail(c, "扫描任务下发失败", gin.H{"error": err.Error()})
		return
	}

	tools.SuccessWithStatus(c, http.StatusAccepted, gin.H{
		"ret":     "ok",
		"jobId":   job.ID,
		"jobUuid": job.JobUUID,
		"status":  job.Status,
	}, "扫描任务下发成功")
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
