package model

type ScanJobLogDB struct {
	CommonModel
	JobID       uint   `json:"jobId" gorm:"column:job_id;index;comment:任务ID"`
	Level       string `json:"level" gorm:"column:level;size:16;index;comment:日志级别"`
	Phase       string `json:"phase" gorm:"column:phase;size:64;index;comment:阶段"`
	Message     string `json:"message" gorm:"column:message;type:text;comment:日志消息"`
	PayloadJSON string `json:"payloadJson" gorm:"column:payload_json;type:longtext;comment:日志负载"`
}

func (ScanJobLogDB) TableName() string {
	return "scan_job_log"
}

type ScanJobLogSearch struct {
	PageInfo
	JobID uint `json:"jobId" form:"jobId"`
}
