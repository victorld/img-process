package dao

import (
	"img_process/model"
	"img_process/plugin/orm"
)

type ScanScheduleService struct{}

func (s *ScanScheduleService) RegisterScanSchedule(schedule *model.ScanScheduleDB) error {
	return orm.ImgMysqlDB.AutoMigrate(&schedule)
}

func (s *ScanScheduleService) Create(schedule *model.ScanScheduleDB) error {
	return orm.ImgMysqlDB.Create(schedule).Error
}

func (s *ScanScheduleService) Update(schedule *model.ScanScheduleDB) error {
	return orm.ImgMysqlDB.Save(schedule).Error
}

func (s *ScanScheduleService) Delete(id uint) error {
	return orm.ImgMysqlDB.Delete(&model.ScanScheduleDB{}, id).Error
}

func (s *ScanScheduleService) GetByID(id uint) (model.ScanScheduleDB, error) {
	var schedule model.ScanScheduleDB
	err := orm.ImgMysqlDB.First(&schedule, id).Error
	return schedule, err
}

func (s *ScanScheduleService) List(search model.ScanScheduleSearch) ([]model.ScanScheduleDB, int64, error) {
	var (
		list  []model.ScanScheduleDB
		total int64
	)
	db := orm.ImgMysqlDB.Model(&model.ScanScheduleDB{}).Order("id desc")
	if search.Enabled != nil {
		db = db.Where("enabled = ?", *search.Enabled)
	}
	if search.Keyword != "" {
		like := "%" + search.Keyword + "%"
		db = db.Where("name LIKE ?", like)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if search.PageSize > 0 {
		db = db.Limit(search.PageSize).Offset(search.PageSize * (search.Page - 1))
	}
	err := db.Find(&list).Error
	return list, total, err
}

func (s *ScanScheduleService) ListEnabled() ([]model.ScanScheduleDB, error) {
	var list []model.ScanScheduleDB
	err := orm.ImgMysqlDB.Where("enabled = ?", true).Find(&list).Error
	return list, err
}
