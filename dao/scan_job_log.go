package dao

import (
	"img_process/model"
	"img_process/plugin/orm"
)

type ScanJobLogService struct{}

func (s *ScanJobLogService) RegisterScanJobLog(log *model.ScanJobLogDB) error {
	return orm.ImgMysqlDB.AutoMigrate(&log)
}

func (s *ScanJobLogService) Create(log *model.ScanJobLogDB) error {
	return orm.ImgMysqlDB.Create(log).Error
}

func (s *ScanJobLogService) List(search model.ScanJobLogSearch) ([]model.ScanJobLogDB, int64, error) {
	var (
		list  []model.ScanJobLogDB
		total int64
	)

	db := orm.ImgMysqlDB.Model(&model.ScanJobLogDB{}).Where("job_id = ?", search.JobID).Order("id asc")
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if search.PageSize > 0 {
		db = db.Limit(search.PageSize).Offset(search.PageSize * (search.Page - 1))
	}
	err := db.Find(&list).Error
	return list, total, err
}
