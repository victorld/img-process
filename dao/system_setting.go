package dao

import (
	"img_process/model"
	"img_process/plugin/orm"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SystemSettingService struct{}

func (s *SystemSettingService) RegisterSystemSetting(setting *model.SystemSettingDB) error {
	return orm.ImgMysqlDB.AutoMigrate(&setting)
}

func (s *SystemSettingService) List() ([]model.SystemSettingDB, error) {
	var list []model.SystemSettingDB
	err := orm.ImgMysqlDB.Order("section asc, item_key asc").Find(&list).Error
	return list, err
}

func (s *SystemSettingService) Count() (int64, error) {
	var count int64
	err := orm.ImgMysqlDB.Model(&model.SystemSettingDB{}).Count(&count).Error
	return count, err
}

func (s *SystemSettingService) UpsertMany(settings []model.SystemSettingDB) error {
	if len(settings) == 0 {
		return nil
	}
	return orm.ImgMysqlDB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "section"},
			{Name: "item_key"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"value_json", "value_type", "updated_at"}),
	}).Create(&settings).Error
}

func (s *SystemSettingService) DeleteSections(sections []string) error {
	if len(sections) == 0 {
		return nil
	}
	return orm.ImgMysqlDB.Where("section IN ?", sections).Delete(&model.SystemSettingDB{}).Error
}

func (s *SystemSettingService) Transaction(fn func(tx *gorm.DB) error) error {
	return orm.ImgMysqlDB.Transaction(fn)
}
