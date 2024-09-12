package model

import (
	"time"
)

// ImgRecordDB imgRecord表 结构体  ImgRecord
type GisDatabaseDB struct {
	CommonModel
	LocNum    string `json:"locNum" form:"locNum" gorm:"column:loc_num;comment:经纬度;size:100;index:loc_num_key,unique"`
	LocAddr   string `json:"locAddr" form:"locAddr" gorm:"column:loc_addr;comment:位置信息;type:text;"`
	LocStreet string `json:"locStreet" form:"locStreet" gorm:"column:loc_street;comment:街道信息;type:text;"`
	LocJson   string `json:"locJson" form:"locJson" gorm:"column:loc_json;comment:原始返回位置信息;type:text;"`
}

// TableName imgRecord表 ImgRecord自定义表名 img_record
func (GisDatabaseDB) TableName() string {
	return "gis_database"
}

type GisDatabaseSearch struct {
	ImgRecordDB
	StartCreatedAt *time.Time `json:"startCreatedAt" form:"startCreatedAt"`
	EndCreatedAt   *time.Time `json:"endCreatedAt" form:"endCreatedAt"`
	PageInfo
}
