package dao

import (
	"img_process/model"
	"img_process/plugin/orm"
)

type ScanEventService struct{}

func (s *ScanEventService) RegisterScanEvent(event *model.ScanEventDB) error {
	return orm.ImgMysqlDB.AutoMigrate(&event)
}

func (s *ScanEventService) Create(event *model.ScanEventDB) error {
	return orm.ImgMysqlDB.Create(event).Error
}

func (s *ScanEventService) List(search model.ScanEventSearch) ([]model.ScanEventDB, int64, error) {
	var (
		list  []model.ScanEventDB
		total int64
	)

	db := orm.ImgMysqlDB.Model(&model.ScanEventDB{}).Where("job_id = ?", search.JobID).Order("id desc")
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if search.PageSize > 0 {
		db = db.Limit(search.PageSize).Offset(search.PageSize * (search.Page - 1))
	}
	err := db.Find(&list).Error
	return list, total, err
}

func (s *ScanEventService) ListAfterID(jobID uint, afterID uint) ([]model.ScanEventDB, error) {
	var list []model.ScanEventDB
	err := orm.ImgMysqlDB.Where("job_id = ? AND id > ?", jobID, afterID).Order("id asc").Find(&list).Error
	return list, err
}
