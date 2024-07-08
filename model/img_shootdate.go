package model

import (
	"time"
)

// ImgShootDateDB  imgShootdate表 结构体  ImgShootdate
type ImgShootDateDB struct {
	CommonModel
	ImgKey    string `json:"imgKey" form:"imgKey" gorm:"column:img_key;comment:照片;size:255;index:img_key,unique"`
	ShootDate string `json:"shootDate" form:"shootDate" gorm:"column:shoot_date;comment:拍摄时间;size:255;"`
	State     *int   `json:"state" form:"state" gorm:"type:int(10);column:state;comment:状态;size:10;"`
	Remark    string `json:"remark" form:"remark" gorm:"column:remark;comment:备注;size:255;"`
}

// TableName imgShootDate表 ImgShootDate自定义表名 img_shoot_date
func (ImgShootDateDB) TableName() string {
	return "img_shoot_date"
}

type ImgShootDateSearch struct {
	ImgShootDateDB
	StartCreatedAt *time.Time `json:"startCreatedAt" form:"startCreatedAt"`
	EndCreatedAt   *time.Time `json:"endCreatedAt" form:"endCreatedAt"`
	PageInfo
}
