package service

import (
	"encoding/json"
	"fmt"
	"time"

	"img_process/cons"
	"img_process/dao"
	"img_process/model"
	"img_process/tools"
)

type ScanRecorder interface {
	SetScanUUID(scanUUID string)
	SetPhase(phase string, payload map[string]any)
	SetTotalCount(total int64)
	Heartbeat(phase string, processed int64, payload map[string]any)
	Log(level string, phase string, message string, payload map[string]any)
	RecordCandidateAction(item model.ScanActionItemDB) uint
	RecordActionResult(id uint, success bool, err error, payload map[string]any)
	RecordError(phase string, relatedPath string, err error, payload map[string]any)
	RecordArtifact(title string, relatedPath string, payload map[string]any)
	RecordLifecycle(title string, message string, payload map[string]any)
	Finish(summary ImgRecord, artifactPath string)
	FinishWithError(err error)
}

type noopScanRecorder struct{}

func (noopScanRecorder) SetScanUUID(string)                                   {}
func (noopScanRecorder) SetPhase(string, map[string]any)                      {}
func (noopScanRecorder) SetTotalCount(int64)                                  {}
func (noopScanRecorder) Heartbeat(string, int64, map[string]any)              {}
func (noopScanRecorder) Log(string, string, string, map[string]any)           {}
func (noopScanRecorder) RecordCandidateAction(model.ScanActionItemDB) uint    { return 0 }
func (noopScanRecorder) RecordActionResult(uint, bool, error, map[string]any) {}
func (noopScanRecorder) RecordError(string, string, error, map[string]any)    {}
func (noopScanRecorder) RecordArtifact(string, string, map[string]any)        {}
func (noopScanRecorder) RecordLifecycle(string, string, map[string]any)       {}
func (noopScanRecorder) Finish(ImgRecord, string)                             {}
func (noopScanRecorder) FinishWithError(error)                                {}

type DBScanRecorder struct {
	jobID             uint
	jobService        dao.ScanJobService
	actionItemService dao.ScanActionItemService
	eventService      dao.ScanEventService
	jobLogService     dao.ScanJobLogService
}

func NewDBScanRecorder(jobID uint) *DBScanRecorder {
	return &DBScanRecorder{jobID: jobID}
}

func (r *DBScanRecorder) SetScanUUID(scanUUID string) {
	job, err := r.jobService.GetByID(r.jobID)
	if err != nil {
		return
	}
	job.ScanUUID = scanUUID
	job.ArtifactPath = cons.WorkDir + "/log/dump_delete_file/" + scanUUID
	_ = r.jobService.Update(&job)
	r.Log("info", "initializing", "已分配扫描 UUID", ginH("scanUUID", scanUUID))
}

func (r *DBScanRecorder) SetPhase(phase string, payload map[string]any) {
	now := time.Now()
	job, err := r.jobService.GetByID(r.jobID)
	if err == nil {
		job.CurrentPhase = phase
		job.LastHeartbeatAt = &now
		_ = r.jobService.Update(&job)
	}
	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:       r.jobID,
		EventType:   model.EventTypePhase,
		Phase:       phase,
		Level:       "info",
		Title:       "阶段变更",
		Message:     phase,
		PayloadJSON: marshalPayload(payload),
	})
	r.Log("info", phase, "阶段变更", payload)
}

func (r *DBScanRecorder) SetTotalCount(total int64) {
	job, err := r.jobService.GetByID(r.jobID)
	if err != nil {
		return
	}
	job.TotalCount = total
	_ = r.jobService.Update(&job)
}

func (r *DBScanRecorder) Heartbeat(phase string, processed int64, payload map[string]any) {
	now := time.Now()
	job, err := r.jobService.GetByID(r.jobID)
	if err == nil {
		job.CurrentPhase = phase
		job.LastHeartbeatAt = &now
		job.ProcessedCount = processed
		_ = r.jobService.Update(&job)
	}
	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:       r.jobID,
		EventType:   model.EventTypeProgress,
		Phase:       phase,
		Level:       "info",
		Title:       "进度更新",
		Message:     fmt.Sprintf("已处理 %d 项", processed),
		PayloadJSON: marshalPayload(payload),
	})
	logPayload := ginH("processed", processed)
	for key, value := range payload {
		logPayload[key] = value
	}
	r.Log("info", phase, fmt.Sprintf("已处理 %d 项", processed), logPayload)
}

func (r *DBScanRecorder) Log(level string, phase string, message string, payload map[string]any) {
	_ = r.jobLogService.Create(&model.ScanJobLogDB{
		JobID:       r.jobID,
		Level:       level,
		Phase:       phase,
		Message:     message,
		PayloadJSON: marshalPayload(payload),
	})
}

func (r *DBScanRecorder) RecordCandidateAction(item model.ScanActionItemDB) uint {
	item.JobID = r.jobID
	if item.Stage == "" {
		item.Stage = model.ActionStageCandidate
	}
	if item.Status == "" {
		item.Status = model.ActionStatusPending
	}
	now := time.Now()
	item.DiscoveredAt = &now
	if err := r.actionItemService.Create(&item); err != nil {
		tools.Logger.Error("create action item error : ", err)
		return 0
	}
	r.Log("info", item.Stage, "记录候选动作", ginH(
		"actionType", item.ActionType,
		"sourcePath", item.SourcePath,
		"targetPath", item.TargetPath,
		"reasonText", item.ReasonText,
	))
	return item.ID
}

func (r *DBScanRecorder) RecordActionResult(id uint, success bool, err error, payload map[string]any) {
	if id == 0 {
		return
	}
	item, getErr := r.actionItemService.GetByID(id)
	if getErr != nil {
		return
	}
	now := time.Now()
	item.Stage = model.ActionStageExecuted
	item.ExecutedAt = &now
	item.MetadataJSON = mergeJSON(item.MetadataJSON, payload)
	if success {
		item.Status = model.ActionStatusSucceeded
		item.ErrorMessage = ""
	} else {
		item.Status = model.ActionStatusFailed
		if err != nil {
			item.ErrorMessage = err.Error()
		}
	}
	_ = r.actionItemService.Update(&item)
	logPayload := ginH(
		"actionType", item.ActionType,
		"sourcePath", item.SourcePath,
		"targetPath", item.TargetPath,
	)
	for key, value := range payload {
		logPayload[key] = value
	}
	if success {
		r.Log("info", item.Stage, "动作执行成功", logPayload)
		return
	}
	logPayload["error"] = errString(err)
	r.Log("error", item.Stage, "动作执行失败", logPayload)
}

func (r *DBScanRecorder) RecordError(phase string, relatedPath string, err error, payload map[string]any) {
	if err == nil {
		return
	}
	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:       r.jobID,
		EventType:   model.EventTypeError,
		Phase:       phase,
		Level:       "error",
		Title:       "scan error",
		Message:     err.Error(),
		RelatedPath: relatedPath,
		PayloadJSON: marshalPayload(payload),
	})
	logPayload := ginH("relatedPath", relatedPath)
	for key, value := range payload {
		logPayload[key] = value
	}
	logPayload["error"] = err.Error()
	r.Log("error", phase, err.Error(), logPayload)
}

func (r *DBScanRecorder) RecordArtifact(title string, relatedPath string, payload map[string]any) {
	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:       r.jobID,
		EventType:   model.EventTypeArtifact,
		Phase:       "artifact",
		Level:       "info",
		Title:       title,
		Message:     title,
		RelatedPath: relatedPath,
		PayloadJSON: marshalPayload(payload),
	})
	logPayload := ginH("relatedPath", relatedPath)
	for key, value := range payload {
		logPayload[key] = value
	}
	r.Log("info", "artifact", title, logPayload)
}

func (r *DBScanRecorder) RecordLifecycle(title string, message string, payload map[string]any) {
	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:       r.jobID,
		EventType:   model.EventTypeLifecycle,
		Phase:       "lifecycle",
		Level:       "info",
		Title:       title,
		Message:     message,
		PayloadJSON: marshalPayload(payload),
	})
	r.Log("info", "lifecycle", message, payload)
}

func (r *DBScanRecorder) Finish(summary ImgRecord, artifactPath string) {
	job, err := r.jobService.GetByID(r.jobID)
	if err != nil {
		return
	}
	now := time.Now()
	job.Status = model.JobStatusSucceeded
	job.EndAt = &now
	job.LastHeartbeatAt = &now
	job.CurrentPhase = "completed"
	job.ProcessedCount = int64(summary.FileTotal)
	job.TotalCount = int64(summary.FileTotal)
	job.SummaryJSON = tools.MarshalJsonToString(summary)
	job.ArtifactPath = artifactPath
	job.ErrorMessage = summary.Remark
	_ = r.jobService.Update(&job)
	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:     r.jobID,
		EventType: model.EventTypeLifecycle,
		Phase:     "completed",
		Level:     "info",
		Title:     "任务完成",
		Message:   "任务执行完成",
	})
	r.Log("info", "completed", "任务完成", ginH("artifactPath", artifactPath))
	if job.ScheduleID != nil {
		updateScheduleJobStatus(*job.ScheduleID, job.ID, job.Status)
	}
}

func (r *DBScanRecorder) FinishWithError(err error) {
	job, getErr := r.jobService.GetByID(r.jobID)
	if getErr != nil {
		return
	}
	now := time.Now()
	job.Status = model.JobStatusFailed
	job.EndAt = &now
	job.LastHeartbeatAt = &now
	if err != nil {
		job.ErrorMessage = err.Error()
	}
	_ = r.jobService.Update(&job)
	r.RecordError(job.CurrentPhase, "", err, nil)
	r.RecordLifecycle("任务失败", "任务执行失败", ginH("error", errString(err)))
	r.Log("error", job.CurrentPhase, "任务失败", ginH("error", errString(err)))
	if job.ScheduleID != nil {
		updateScheduleJobStatus(*job.ScheduleID, job.ID, job.Status)
	}
}

func marshalPayload(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	return tools.MarshalJsonToString(payload)
}

func mergeJSON(origin string, payload map[string]any) string {
	if len(payload) == 0 {
		return origin
	}
	if origin == "" {
		return marshalPayload(payload)
	}
	var current map[string]any
	if err := json.Unmarshal([]byte(origin), &current); err != nil {
		current = map[string]any{}
	}
	for key, value := range payload {
		current[key] = value
	}
	return tools.MarshalJsonToString(current)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func updateScheduleJobStatus(scheduleID uint, jobID uint, status string) {
	svc := dao.ScanScheduleService{}
	schedule, err := svc.GetByID(scheduleID)
	if err != nil {
		return
	}
	schedule.LastJobID = &jobID
	schedule.LastJobStatus = status
	_ = svc.Update(&schedule)
}
