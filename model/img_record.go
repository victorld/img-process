package model

import (
	"time"
)

// ImgRecordDB imgRecord表 结构体  ImgRecord
type ImgRecordDB struct {
	CommonModel
	ScanArgs                string     `json:"scanArgs" form:"scanArgs" gorm:"column:scan_args;comment:扫描参数;size:1000;"`
	FileTotal               *int       `json:"fileTotal" form:"fileTotal" gorm:"type:int(10);column:file_total;comment:文件总数;size:10;"`                                                     //文件总数
	FileTotalBak            *int       `json:"fileTotalBak" form:"fileTotalBak" gorm:"type:int(10);column:file_total_bak;comment:文件总数;size:10;"`                                           //文件总数
	DirTotal                *int       `json:"dirTotal" form:"dirTotal" gorm:"type:int(10);column:dir_total;comment:目录总数;size:10;"`                                                        //目录总数
	DirTotalBak             *int       `json:"dirTotalBak" form:"dirTotalBak" gorm:"type:int(10);column:dir_total_bak;comment:目录总数;size:10;"`                                              //目录总数
	StartDate               *time.Time `json:"startDate" form:"startDate" gorm:"column:start_date;comment:开始时间;"`                                                                          //记录时间
	UseTime                 *int       `json:"useTime" form:"useTime" gorm:"type:int(10);column:use_time;comment:用时;size:10;"`                                                             //用时
	BasePath                string     `json:"basePath" form:"basePath" gorm:"column:base_path;comment:基础目录;size:255;"`                                                                    //基础目录
	BasePathBak             string     `json:"basePathBak" form:"basePathBak" gorm:"column:base_path_bak;comment:基础目录;size:255;"`                                                          //基础目录
	SuffixMap               string     `json:"suffixMap" form:"suffixMap" gorm:"column:suffix_map;comment:后缀统计;size:255;"`                                                                 //后缀统计
	SuffixMapBak            string     `json:"suffixMapBak" form:"suffixMapBak" gorm:"column:suffix_map_bak;comment:后缀统计;size:255;"`                                                       //后缀统计
	YearMap                 string     `json:"yearMap" form:"yearMap" gorm:"column:year_map;comment:年份统计;size:255;"`                                                                       //年份统计
	YearMapBak              string     `json:"yearMapBak" form:"yearMapBak" gorm:"column:year_map_bak;comment:年份统计;size:255;"`                                                             //年份统计
	BakNewFileCnt           *int       `json:"bakNewFileCnt" form:"bakNewFileCnt" gorm:"type:int(10);column:bak_new_file_cnt;comment:用时;size:10;"`                                         //
	BakDeleteFileCnt        *int       `json:"bakDeleteFileCnt" form:"bakDeleteFileCnt" gorm:"type:int(10);column:bak_delete_file_cnt;comment:用时;size:10;"`                                //
	BakNewFile              string     `json:"bakNewFile" form:"bakNewFile" gorm:"column:bak_new_file;comment:年份统计;type:text;"`                                                            //
	BakDeleteFile           string     `json:"bakDeleteFile" form:"bakDeleteFile" gorm:"column:bak_delete_file;comment:年份统计;type:text;"`                                                   //
	FileDateCnt             *int       `json:"fileDateCnt" form:"fileDateCnt" gorm:"type:int(10);column:file_date_cnt;comment:有时间文件统计;size:10;"`                                           //有时间文件统计
	DeleteFileCnt           *int       `json:"deleteFileCnt" form:"deleteFileCnt" gorm:"type:int(10);column:delete_file_cnt;comment:需要删除文件数;size:10;"`                                     //需要删除文件数
	ModifyDateFileCnt       *int       `json:"modifyDateFileCnt" form:"modifyDateFileCnt" gorm:"type:int(10);column:modify_date_file_cnt;comment:需要修改修改日期文件数;size:10;"`                    //需要修改修改日期文件数
	MoveFileCnt             *int       `json:"moveFileCnt" form:"moveFileCnt" gorm:"type:int(10);column:move_file_cnt;comment:需要移动文件数;size:10;"`                                           //需要移动文件数
	RenameFileCnt           *int       `json:"renameFileCnt" form:"renameFileCnt" gorm:"type:int(10);column:rename_file_cnt;comment:需要改名文件数;size:10;"`                                     //需要移动文件数
	ShootDateFileCnt        *int       `json:"shootDateFileCnt" form:"shootDateFileCnt" gorm:"type:int(10);column:shoot_date_file_cnt;comment:需要修改拍摄日期文件数;size:10;"`                       //需要修改拍摄日期文件数
	ShootDateEarlierFileCnt *int       `json:"shootDateEarlierFileCnt" form:"shootDateEarlierFileCnt" gorm:"type:int(10);column:shoot_date_earlier_file_cnt;comment:需要修改拍摄日期文件数;size:10;"` //需要修改拍摄日期文件数，拍摄日期更小
	EmptyDirCnt             *int       `json:"emptyDirCnt" form:"emptyDirCnt" gorm:"type:int(10);column:empty_dir_cnt;comment:空文件数;size:10;"`                                              //空文件数
	DumpFileCnt             *int       `json:"dumpFileCnt" form:"dumpFileCnt" gorm:"type:int(10);column:dump_file_cnt;comment:重复md5数;size:10;"`                                            //重复md5数
	//DumpFileDeleteList string     `json:"dumpFileDeleteList" form:"dumpFileDeleteList" gorm:"column:dump_file_delete_list;comment:需要删除文件数;"`                       //需要删除文件数
	ExifErrCnt      *int   `json:"exifErrCnt" form:"exifErrCnt" gorm:"type:int(10);column:exif_err_cnt;comment:exif错误数;size:10;"`        //exif错误数
	ExifDateNameSet string `json:"exifDateNameSet" form:"exifDateNameSet" gorm:"column:exif_date_name_set;comment:exif错误3统计;type:text;"` //exif错误3统计
	IsComplete      *int   `json:"isComplete" form:"isComplete" gorm:"type:int(10);column:is_complete;comment:是否完整;size:10;"`
	Remark          string `json:"remark" form:"remark" gorm:"column:remark;comment:备注;type:text;"`
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
