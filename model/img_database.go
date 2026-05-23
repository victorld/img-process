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
	LocStreet string `json:"locStreet" form:"locStreet" gorm:"column:loc_street;comment:街道信息;type:text;"`
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

type FileAnalysisSearch struct {
	PageInfo
	FileKey         string
	ShootDateStatus string
	GeoStatus       string
	LocAddrKeyword  string
	ShootDateStart  string
	ShootDateEnd    string
}

type FileAnalysisSummary struct {
	TotalCount         int64   `json:"totalCount"`
	WithShootDateCount int64   `json:"withShootDateCount"`
	WithLocNumCount    int64   `json:"withLocNumCount"`
	WithLocAddrCount   int64   `json:"withLocAddrCount"`
	ShootDateCoverage  float64 `json:"shootDateCoverage"`
	LocNumCoverage     float64 `json:"locNumCoverage"`
	LocAddrCoverage    float64 `json:"locAddrCoverage"`
}

type FileAnalysisStatItem struct {
	Key     string  `json:"key"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

type FileAnalysisItem struct {
	ID         uint      `json:"id"`
	ImgKey     string    `json:"imgKey"`
	DirDate    string    `json:"dirDate"`
	FileName   string    `json:"fileName"`
	Suffix     string    `json:"suffix"`
	ShootDate  string    `json:"shootDate"`
	LocNum     string    `json:"locNum"`
	LocStreet  string    `json:"locStreet"`
	LocAddr    string    `json:"locAddr"`
	Remark     string    `json:"remark"`
	UpdatedAt  time.Time `json:"updatedAt"`
	PreviewURL string    `json:"previewUrl"`
}

type FileAnalysisResult struct {
	Summary     FileAnalysisSummary    `json:"summary"`
	YearStats   []FileAnalysisStatItem `json:"yearStats"`
	SuffixStats []FileAnalysisStatItem `json:"suffixStats"`
	List        []FileAnalysisItem     `json:"list"`
	Total       int64                  `json:"total"`
}
