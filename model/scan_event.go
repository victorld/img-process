package model

const (
	EventTypeLifecycle = "lifecycle"
	EventTypePhase     = "phase"
	EventTypeProgress  = "progress"
	EventTypeError     = "error"
	EventTypeArtifact  = "artifact"
)

type ScanEventDB struct {
	CommonModel
	JobID       uint   `json:"jobId" gorm:"column:job_id;index;comment:任务ID"`
	EventType   string `json:"eventType" gorm:"column:event_type;size:32;index;comment:事件类型"`
	Phase       string `json:"phase" gorm:"column:phase;size:64;index;comment:阶段"`
	Level       string `json:"level" gorm:"column:level;size:16;comment:级别"`
	Title       string `json:"title" gorm:"column:title;size:255;comment:标题"`
	Message     string `json:"message" gorm:"column:message;type:text;comment:消息"`
	RelatedPath string `json:"relatedPath" gorm:"column:related_path;size:1024;comment:关联路径"`
	PayloadJSON string `json:"payloadJson" gorm:"column:payload_json;type:longtext;comment:负载"`
}

func (ScanEventDB) TableName() string {
	return "scan_event"
}

type ScanEventSearch struct {
	PageInfo
	JobID uint `json:"jobId" form:"jobId"`
}
