package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"img_process/cons"
	"img_process/model"
	"img_process/service"
	"img_process/tools"
)

type WebAPI struct{}

func (api *WebAPI) Login(c *gin.Context) {
	var req model.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	if req.Username != cons.HttpUsername || req.Password != cons.HttpPassword {
		tools.FailWithStatus(c, http.StatusUnauthorized, "用户名或密码错误", gin.H{"authenticated": false})
		return
	}
	token := service.Runtime.Login(req.Username)
	c.SetCookie("img_process_session", token, 86400, "/", "", false, true)
	tools.Success(c, gin.H{"username": req.Username, "authenticated": true}, "登录成功")
}

func (api *WebAPI) Logout(c *gin.Context) {
	token, _ := c.Cookie("img_process_session")
	if token != "" {
		service.Runtime.Logout(token)
	}
	c.SetCookie("img_process_session", "", -1, "/", "", false, true)
	tools.Success(c, gin.H{"authenticated": false}, "退出成功")
}

func (api *WebAPI) Me(c *gin.Context) {
	username, _ := c.Get("username")
	tools.Success(c, gin.H{"username": username, "authenticated": true}, "ok")
}

func (api *WebAPI) ListJobs(c *gin.Context) {
	var search model.ScanJobSearch
	bindPageQuery(c, &search.PageInfo)
	search.Status = c.Query("status")
	search.Source = c.Query("source")
	search.Keyword = c.Query("keyword")
	if raw := c.Query("hasAction"); raw != "" {
		val := raw == "true"
		search.HasAction = &val
	}
	if start, end := parseTimeRange(c.Query("startCreated"), c.Query("endCreated")); start != nil && end != nil {
		search.StartCreated = start
		search.EndCreated = end
	}

	list, total, err := service.Runtime.ListJobs(search)
	if err != nil {
		tools.Fail(c, "查询任务失败", gin.H{"error": err.Error()})
		return
	}
	ret := make([]gin.H, 0, len(list))
	for _, item := range list {
		ret = append(ret, serializeJob(item))
	}
	tools.Success(c, gin.H{"list": ret, "total": total}, "ok")
}

func (api *WebAPI) CreateJob(c *gin.Context) {
	var req model.CreateJobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	job, err := service.Runtime.CreateJob(req.Source, req.ScheduleID, req.ScanArgs)
	if err != nil {
		tools.Fail(c, "创建任务失败", gin.H{"error": err.Error()})
		return
	}
	tools.SuccessWithStatus(c, http.StatusCreated, gin.H{"job": serializeJob(job)}, "任务已创建")
}

func (api *WebAPI) GetJob(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	job, err := service.Runtime.GetJob(jobID)
	if err != nil {
		tools.FailWithStatus(c, http.StatusNotFound, "任务不存在", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"job": serializeJob(job)}, "ok")
}

func (api *WebAPI) ListJobEvents(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	var search model.ScanEventSearch
	search.JobID = jobID
	bindPageQuery(c, &search.PageInfo)
	list, total, err := service.Runtime.ListEvents(search)
	if err != nil {
		tools.Fail(c, "查询事件失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"list": list, "total": total}, "ok")
}

func (api *WebAPI) ListJobActionItems(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	var search model.ScanActionItemSearch
	search.JobID = jobID
	search.Tab = c.DefaultQuery("tab", "pending")
	search.ActionType = c.Query("type")
	search.Status = c.Query("status")
	search.Keyword = c.Query("keyword")
	bindPageQuery(c, &search.PageInfo)
	list, total, err := service.Runtime.ListActionItems(search)
	if err != nil {
		tools.Fail(c, "查询动作明细失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"list": list, "total": total}, "ok")
}

func (api *WebAPI) StreamJob(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	var lastEventID uint
	var lastStatus string
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			job, err := service.Runtime.GetJob(jobID)
			if err != nil {
				c.SSEvent("error", gin.H{"message": err.Error()})
				c.Writer.Flush()
				return
			}
			if job.Status != lastStatus {
				lastStatus = job.Status
				c.SSEvent("job", serializeJob(job))
			}
			events, err := service.Runtime.ListEventsAfter(jobID, lastEventID)
			if err == nil {
				for _, event := range events {
					lastEventID = event.ID
					c.SSEvent("event", event)
				}
			}
			c.Writer.Flush()
			if isFinishedStatus(job.Status) {
				return
			}
		}
	}
}

func (api *WebAPI) DeleteJobDuplicates(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	if err := service.Runtime.DeleteDuplicateFiles(jobID); err != nil {
		tools.Fail(c, "执行重复文件删除失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"jobId": jobID}, "重复文件删除已执行")
}

func (api *WebAPI) ListSchedules(c *gin.Context) {
	var search model.ScanScheduleSearch
	bindPageQuery(c, &search.PageInfo)
	if raw := c.Query("enabled"); raw != "" {
		val := raw == "true"
		search.Enabled = &val
	}
	list, total, err := service.Runtime.ListSchedules(search)
	if err != nil {
		tools.Fail(c, "查询计划失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"list": list, "total": total}, "ok")
}

func (api *WebAPI) CreateSchedule(c *gin.Context) {
	var req model.UpsertScheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	schedule, err := service.Runtime.CreateSchedule(req)
	if err != nil {
		tools.Fail(c, "创建计划失败", gin.H{"error": err.Error()})
		return
	}
	tools.SuccessWithStatus(c, http.StatusCreated, gin.H{"schedule": schedule}, "计划已创建")
}

func (api *WebAPI) UpdateSchedule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "计划ID错误", gin.H{"error": err.Error()})
		return
	}
	var req model.UpsertScheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	schedule, err := service.Runtime.UpdateSchedule(id, req)
	if err != nil {
		tools.Fail(c, "更新计划失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"schedule": schedule}, "计划已更新")
}

func (api *WebAPI) DeleteSchedule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "计划ID错误", gin.H{"error": err.Error()})
		return
	}
	if err := service.Runtime.DeleteSchedule(id); err != nil {
		tools.Fail(c, "删除计划失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"id": id}, "计划已删除")
}

func (api *WebAPI) EnableSchedule(c *gin.Context) {
	api.toggleSchedule(c, true)
}

func (api *WebAPI) DisableSchedule(c *gin.Context) {
	api.toggleSchedule(c, false)
}

func (api *WebAPI) RunSchedule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "计划ID错误", gin.H{"error": err.Error()})
		return
	}
	job, err := service.Runtime.RunSchedule(id)
	if err != nil {
		tools.Fail(c, "计划执行失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"job": serializeJob(job)}, "计划已触发")
}

func (api *WebAPI) GetSystemStatus(c *gin.Context) {
	tools.Success(c, gin.H{
		"server": gin.H{
			"httpPort":      cons.HttpPort,
			"startPath":     cons.StartPath,
			"startPathBak":  cons.StartPathBak,
			"bakStatEnable": cons.BakStatEnable,
			"poolSize":      cons.PoolSize,
			"imgCache":      cons.ImgCache,
			"sqlDebug":      cons.SqlDebug,
			"scanDefaults": gin.H{
				"startPath":        cons.StartPath,
				"startPathBak":     cons.StartPathBak,
				"deleteShow":       cons.DeleteShow,
				"moveFileShow":     cons.MoveFileShow,
				"modifyDateShow":   cons.ModifyDateShow,
				"renameFileShow":   cons.RenameFileShow,
				"md5Show":          cons.Md5Show,
				"deleteAction":     cons.DeleteAction,
				"moveFileAction":   cons.MoveFileAction,
				"modifyDateAction": cons.ModifyDateAction,
				"renameFileAction": cons.RenameFileAction,
			},
		},
	}, "ok")
}

func (api *WebAPI) toggleSchedule(c *gin.Context, enabled bool) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "计划ID错误", gin.H{"error": err.Error()})
		return
	}
	schedule, err := service.Runtime.SetScheduleEnabled(id, enabled)
	if err != nil {
		tools.Fail(c, "更新计划状态失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"schedule": schedule}, "ok")
}

func parseUintParam(c *gin.Context, key string) (uint, error) {
	raw := c.Param(key)
	if raw == "" {
		return 0, errors.New("missing id")
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	return uint(id), err
}

func bindPageQuery(c *gin.Context, pageInfo *model.PageInfo) {
	pageInfo.Page = 1
	pageInfo.PageSize = 20
	if raw := c.Query("page"); raw != "" {
		if page, err := strconv.Atoi(raw); err == nil && page > 0 {
			pageInfo.Page = page
		}
	}
	if raw := c.Query("pageSize"); raw != "" {
		if size, err := strconv.Atoi(raw); err == nil && size > 0 {
			pageInfo.PageSize = size
		}
	}
}

func parseTimeRange(startRaw string, endRaw string) (*time.Time, *time.Time) {
	if startRaw == "" || endRaw == "" {
		return nil, nil
	}
	start, err1 := time.Parse(time.RFC3339, startRaw)
	end, err2 := time.Parse(time.RFC3339, endRaw)
	if err1 != nil || err2 != nil {
		return nil, nil
	}
	return &start, &end
}

func serializeJob(job model.ScanJobDB) gin.H {
	return gin.H{
		"id":              job.ID,
		"jobUuid":         job.JobUUID,
		"scanUuid":        job.ScanUUID,
		"source":          job.Source,
		"status":          job.Status,
		"scheduleId":      job.ScheduleID,
		"queueAt":         job.QueueAt,
		"startAt":         job.StartAt,
		"endAt":           job.EndAt,
		"currentPhase":    job.CurrentPhase,
		"lastHeartbeatAt": job.LastHeartbeatAt,
		"processedCount":  job.ProcessedCount,
		"totalCount":      job.TotalCount,
		"hasAction":       job.HasAction,
		"scanArgs":        parseJSON(job.ScanArgs),
		"summary":         parseJSON(job.SummaryJSON),
		"artifactPath":    job.ArtifactPath,
		"errorMessage":    job.ErrorMessage,
		"createdAt":       job.CreatedAt,
		"updatedAt":       job.UpdatedAt,
	}
}

func parseJSON(raw string) any {
	if raw == "" {
		return nil
	}
	var ret any
	if err := json.Unmarshal([]byte(raw), &ret); err != nil {
		return raw
	}
	return ret
}

func isFinishedStatus(status string) bool {
	return status == model.JobStatusSucceeded || status == model.JobStatusFailed || status == model.JobStatusInterrupted || status == model.JobStatusSkipped
}
