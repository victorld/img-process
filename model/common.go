package model

import (
	"gorm.io/gorm"
	"time"
)

type CommonModel struct {
	ID        uint           `gorm:"primaryKey"` // 主键ID
	CreatedAt time.Time      // 创建时间
	UpdatedAt time.Time      // 更新时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 删除时间
}

type IdsReq struct {
	Ids []int `json:"ids" form:"ids"`
}

// PageInfo Paging common input parameter structure
type PageInfo struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页大小
	Keyword  string `json:"keyword" form:"keyword"`   //关键字
}
