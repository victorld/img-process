package model

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreateJobReq struct {
	DisplayName string       `json:"displayName"`
	Source      string       `json:"source"`
	ScheduleID  *uint        `json:"scheduleId"`
	ScanArgs    DoScanImgArg `json:"scanArgs"`
}

type UpsertScheduleReq struct {
	Name           string       `json:"name" binding:"required"`
	Enabled        bool         `json:"enabled"`
	Timezone       string       `json:"timezone"`
	Mode           string       `json:"mode"`
	CronExpr       string       `json:"cronExpr"`
	ScheduleConfig string       `json:"scheduleConfig"`
	ScanArgs       DoScanImgArg `json:"scanArgs"`
}
