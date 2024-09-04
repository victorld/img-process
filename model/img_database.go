package model

import (
	"time"
)

// ImgDatabaseDB  imgShootdate表 结构体  ImgShootdate
type ImgDatabaseDB struct {
	CommonModel
	ImgKey    string `json:"imgKey" form:"imgKey" gorm:"column:img_key;comment:照片;size:255;index:img_key,unique"`
	ShootDate string `json:"shootDate" form:"shootDate" gorm:"column:shoot_date;comment:拍摄时间;size:255;"`
	LocNum    string `json:"locNum" form:"locNum" gorm:"column:loc_num;comment:经纬度;size:255;"`
	LocAddr   string `json:"locAddr" form:"locAddr" gorm:"column:loc_addr;comment:位置信息;type:text;"`
	State     *int   `json:"state" form:"state" gorm:"type:int(10);column:state;comment:状态(1：启用，当前都为1);size:10;"`
	Remark    string `json:"remark" form:"remark" gorm:"column:remark;comment:备注;type:text;"`
}

// TableName imgDatabase表 ImgDatabase自定义表名 img_database
func (ImgDatabaseDB) TableName() string {
	return "img_database"
}

type ImgDatabaseSearch struct {
	ImgDatabaseDB
	StartCreatedAt *time.Time `json:"startCreatedAt" form:"startCreatedAt"`
	EndCreatedAt   *time.Time `json:"endCreatedAt" form:"endCreatedAt"`
	PageInfo
}
