package model

import "time"

const (
	ScheduleModeDaily   = "daily"
	ScheduleModeWeekly  = "weekly"
	ScheduleModeMonthly = "monthly"
	ScheduleModeCustom  = "custom"
)

type ScanScheduleDB struct {
	CommonModel
	Name           string     `json:"name" gorm:"column:name;size:128;comment:名称"`
	Enabled        bool       `json:"enabled" gorm:"column:enabled;index;comment:是否启用"`
	Timezone       string     `json:"timezone" gorm:"column:timezone;size:64;comment:时区"`
	Mode           string     `json:"mode" gorm:"column:mode;size:32;comment:计划模式"`
	CronExpr       string     `json:"cronExpr" gorm:"column:cron_expr;size:128;comment:cron表达式"`
	ScheduleConfig string     `json:"scheduleConfig" gorm:"column:schedule_config;type:longtext;comment:计划配置"`
	ScanArgs       string     `json:"scanArgs" gorm:"column:scan_args;type:longtext;comment:扫描参数"`
	LastRunAt      *time.Time `json:"lastRunAt" gorm:"column:last_run_at;comment:最近运行时间"`
	NextRunAt      *time.Time `json:"nextRunAt" gorm:"column:next_run_at;comment:下次运行时间"`
	LastJobID      *uint      `json:"lastJobId" gorm:"column:last_job_id;comment:最近任务ID"`
	LastJobStatus  string     `json:"lastJobStatus" gorm:"column:last_job_status;size:32;comment:最近任务状态"`
}

func (ScanScheduleDB) TableName() string {
	return "scan_schedule"
}

type ScanScheduleSearch struct {
	PageInfo
	Enabled *bool `json:"enabled" form:"enabled"`
}
