package model

import (
	"time"
)

// imgRecord表 结构体  ImgRecord
type ImgRecordDB struct {
	COMMON_MODEL
	ScanArgs           string     `json:"scanArgs" form:"scanArgs" gorm:"column:scan_args;comment:扫描参数;size:255;"`
	FileTotal          *int       `json:"fileTotal" form:"fileTotal" gorm:"type:int(10);column:file_total;comment:文件总数;size:10;"`                                  //文件总数
	DirTotal           *int       `json:"dirTotal" form:"dirTotal" gorm:"type:int(10);column:dir_total;comment:目录总数;size:10;"`                                     //目录总数
	StartDate          *time.Time `json:"startDate" form:"startDate" gorm:"column:start_date;comment:开始时间;"`                                                       //记录时间
	UseTime            *int       `json:"useTime" form:"useTime" gorm:"type:int(10);column:use_time;comment:用时;size:10;"`                                          //用时
	BasePath           string     `json:"basePath" form:"basePath" gorm:"column:base_path;comment:基础目录;size:255;"`                                                 //基础目录
	SuffixMap          string     `json:"suffixMap" form:"suffixMap" gorm:"column:suffix_map;comment:后缀统计;size:255;"`                                              //后缀统计
	YearMap            string     `json:"yearMap" form:"yearMap" gorm:"column:year_map;comment:年份统计;size:255;"`                                                    //年份统计
	FileDateCnt        *int       `json:"fileDateCnt" form:"fileDateCnt" gorm:"type:int(10);column:file_date_cnt;comment:有时间文件统计;size:10;"`                        //有时间文件统计
	DeleteFileCnt      *int       `json:"deleteFileCnt" form:"deleteFileCnt" gorm:"type:int(10);column:delete_file_cnt;comment:需要删除文件数;size:10;"`                  //需要删除文件数
	ModifyDateFileCnt  *int       `json:"modifyDateFileCnt" form:"modifyDateFileCnt" gorm:"type:int(10);column:modify_date_file_cnt;comment:需要修改修改日期文件数;size:10;"` //需要修改修改日期文件数
	MoveFileCnt        *int       `json:"moveFileCnt" form:"moveFileCnt" gorm:"type:int(10);column:move_file_cnt;comment:需要移动文件数;size:10;"`                        //需要移动文件数
	ShootDateFileCnt   *int       `json:"shootDateFileCnt" form:"shootDateFileCnt" gorm:"type:int(10);column:shoot_date_file_cnt;comment:需要修改拍摄日期文件数;size:10;"`    //需要修改拍摄日期文件数
	EmptyDirCnt        *int       `json:"emptyDirCnt" form:"emptyDirCnt" gorm:"type:int(10);column:empty_dir_cnt;comment:空文件数;size:10;"`                           //空文件数
	DumpFileCnt        *int       `json:"dumpFileCnt" form:"dumpFileCnt" gorm:"type:int(10);column:dump_file_cnt;comment:重复md5数;size:10;"`                         //重复md5数
	DumpFileDeleteList string     `json:"dumpFileDeleteList" form:"dumpFileDeleteList" gorm:"column:dump_file_delete_list;comment:需要删除文件数;size:255;"`              //需要删除文件数
	ExifErr1Cnt        *int       `json:"exifErr1Cnt" form:"exifErr1Cnt" gorm:"type:int(10);column:exif_err1_cnt;comment:exif错误1数;size:10;"`                       //exif错误1数
	ExifErr2Cnt        *int       `json:"exifErr2Cnt" form:"exifErr2Cnt" gorm:"type:int(10);column:exif_err2_cnt;comment:exif错误2数;size:10;"`                       //exif错误2数
	ExifErr3Cnt        *int       `json:"exifErr3Cnt" form:"exifErr3Cnt" gorm:"type:int(10);column:exif_err3_cnt;comment:exif错误3数;size:10;"`                       //exif错误3数
	ExifErr1Map        string     `json:"exifErr1Map" form:"exifErr1Map" gorm:"column:exif_err1_map;comment:exif错误1统计;size:255;"`                                  //exif错误1统计
	ExifErr2Map        string     `json:"exifErr2Map" form:"exifErr2Map" gorm:"column:exif_err2_map;comment:exif错误2统计;size:255;"`                                  //exif错误2统计
	ExifErr3Map        string     `json:"exifErr3Map" form:"exifErr3Map" gorm:"column:exif_err3_map;comment:exif错误3统计;size:255;"`                                  //exif错误3统计
	IsComplete         *int       `json:"isComplete" form:"isComplete" gorm:"type:int(10);column:is_complete;comment:是否完整;size:10;"`
	Remark             string     `json:"remark" form:"remark" gorm:"column:remark;comment:备注;size:255;"`
}

// TableName imgRecord表 ImgRecord自定义表名 img_record
func (ImgRecordDB) TableName() string {
	return "img_record"
}

type ImgRecordSearch struct {
	ImgRecordDB
	StartCreatedAt *time.Time `json:"startCreatedAt" form:"startCreatedAt"`
	EndCreatedAt   *time.Time `json:"endCreatedAt" form:"endCreatedAt"`
	PageInfo
}
