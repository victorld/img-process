package model

import "time"

const (
	JobSourceManual   = "manual"
	JobSourceSchedule = "schedule"

	JobStatusPending     = "pending"
	JobStatusRunning     = "running"
	JobStatusSucceeded   = "succeeded"
	JobStatusFailed      = "failed"
	JobStatusInterrupted = "interrupted"
	JobStatusSkipped     = "skipped"
)

type ScanJobDB struct {
	CommonModel
	JobUUID         string     `json:"jobUuid" gorm:"column:job_uuid;size:64;uniqueIndex;comment:任务UUID"`
	ScanUUID        string     `json:"scanUuid" gorm:"column:scan_uuid;size:64;index;comment:扫描UUID"`
	Source          string     `json:"source" gorm:"column:source;size:32;index;comment:来源"`
	Status          string     `json:"status" gorm:"column:status;size:32;index;comment:任务状态"`
	ScheduleID      *uint      `json:"scheduleId" gorm:"column:schedule_id;index;comment:计划ID"`
	QueueAt         *time.Time `json:"queueAt" gorm:"column:queue_at;comment:入队时间"`
	StartAt         *time.Time `json:"startAt" gorm:"column:start_at;comment:开始时间"`
	EndAt           *time.Time `json:"endAt" gorm:"column:end_at;comment:结束时间"`
	CurrentPhase    string     `json:"currentPhase" gorm:"column:current_phase;size:64;comment:当前阶段"`
	LastHeartbeatAt *time.Time `json:"lastHeartbeatAt" gorm:"column:last_heartbeat_at;comment:最近心跳时间"`
	ProcessedCount  int64      `json:"processedCount" gorm:"column:processed_count;comment:已处理数量"`
	TotalCount      int64      `json:"totalCount" gorm:"column:total_count;comment:总数"`
	HasAction       bool       `json:"hasAction" gorm:"column:has_action;comment:是否包含实际执行动作"`
	ScanArgs        string     `json:"scanArgs" gorm:"column:scan_args;type:longtext;comment:扫描参数"`
	SummaryJSON     string     `json:"summaryJson" gorm:"column:summary_json;type:longtext;comment:汇总信息"`
	ArtifactPath    string     `json:"artifactPath" gorm:"column:artifact_path;size:512;comment:产物路径"`
	ErrorMessage    string     `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
}

func (ScanJobDB) TableName() string {
	return "scan_job"
}

type ScanJobSearch struct {
	PageInfo
	Status       string     `json:"status" form:"status"`
	Source       string     `json:"source" form:"source"`
	HasAction    *bool      `json:"hasAction" form:"hasAction"`
	StartCreated *time.Time `json:"startCreated" form:"startCreated"`
	EndCreated   *time.Time `json:"endCreated" form:"endCreated"`
}
