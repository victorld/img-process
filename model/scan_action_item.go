package model

import "time"

const (
	ActionTypeDelete         = "delete"
	ActionTypeMove           = "move"
	ActionTypeRename         = "rename"
	ActionTypeModifyTime     = "modify_time"
	ActionTypeDeleteEmptyDir = "delete_empty_dir"
	ActionTypeDeleteDup      = "delete_duplicate"

	ActionObjectFile = "file"
	ActionObjectDir  = "dir"

	ActionStageCandidate = "candidate"
	ActionStageExecuted  = "executed"

	ActionStatusPending   = "pending"
	ActionStatusSucceeded = "succeeded"
	ActionStatusFailed    = "failed"
	ActionStatusSkipped   = "skipped"
)

type ScanActionItemDB struct {
	CommonModel
	JobID          uint       `json:"jobId" gorm:"column:job_id;index;comment:任务ID"`
	ActionType     string     `json:"actionType" gorm:"column:action_type;size:64;index;comment:动作类型"`
	ObjectType     string     `json:"objectType" gorm:"column:object_type;size:32;comment:对象类型"`
	SourcePath     string     `json:"sourcePath" gorm:"column:source_path;size:1024;index;comment:源路径"`
	TargetPath     string     `json:"targetPath" gorm:"column:target_path;size:1024;comment:目标路径"`
	ReasonCode     string     `json:"reasonCode" gorm:"column:reason_code;size:64;comment:原因码"`
	ReasonText     string     `json:"reasonText" gorm:"column:reason_text;type:text;comment:原因文本"`
	Stage          string     `json:"stage" gorm:"column:stage;size:32;index;comment:阶段"`
	Status         string     `json:"status" gorm:"column:status;size:32;index;comment:状态"`
	DiscoveredAt   *time.Time `json:"discoveredAt" gorm:"column:discovered_at;comment:发现时间"`
	ExecutedAt     *time.Time `json:"executedAt" gorm:"column:executed_at;comment:执行时间"`
	ErrorMessage   string     `json:"errorMessage" gorm:"column:error_message;type:text;comment:错误信息"`
	MetadataJSON   string     `json:"metadataJson" gorm:"column:metadata_json;type:longtext;comment:附加元数据"`
	DuplicateGroup string     `json:"duplicateGroup" gorm:"column:duplicate_group;size:128;index;comment:重复分组"`
}

func (ScanActionItemDB) TableName() string {
	return "scan_action_item"
}

type ScanActionItemSearch struct {
	PageInfo
	JobID      uint   `json:"jobId" form:"jobId"`
	Tab        string `json:"tab" form:"tab"`
	ActionType string `json:"actionType" form:"actionType"`
	Status     string `json:"status" form:"status"`
	Keyword    string `json:"keyword" form:"keyword"`
}
