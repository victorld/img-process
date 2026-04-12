package dao

import (
	"img_process/model"
	"img_process/plugin/orm"
)

type ScanJobService struct{}

func (s *ScanJobService) RegisterScanJob(scanJob *model.ScanJobDB) error {
	return orm.ImgMysqlDB.AutoMigrate(&scanJob)
}

func (s *ScanJobService) Create(job *model.ScanJobDB) error {
	return orm.ImgMysqlDB.Create(job).Error
}

func (s *ScanJobService) Update(job *model.ScanJobDB) error {
	return orm.ImgMysqlDB.Save(job).Error
}

func (s *ScanJobService) GetByID(id uint) (model.ScanJobDB, error) {
	var job model.ScanJobDB
	err := orm.ImgMysqlDB.First(&job, id).Error
	return job, err
}

func (s *ScanJobService) List(search model.ScanJobSearch) ([]model.ScanJobDB, int64, error) {
	var (
		list  []model.ScanJobDB
		total int64
	)

	db := orm.ImgMysqlDB.Model(&model.ScanJobDB{}).Order("id desc")
	if search.Status != "" {
		db = db.Where("status = ?", search.Status)
	}
	if search.Source != "" {
		db = db.Where("source = ?", search.Source)
	}
	if search.HasAction != nil {
		db = db.Where("has_action = ?", *search.HasAction)
	}
	if search.StartCreated != nil && search.EndCreated != nil {
		db = db.Where("created_at BETWEEN ? AND ?", search.StartCreated, search.EndCreated)
	}
	if search.Keyword != "" {
		like := "%" + search.Keyword + "%"
		db = db.Where("job_uuid LIKE ? OR scan_uuid LIKE ? OR error_message LIKE ?", like, like, like)
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

func (s *ScanJobService) GetNextPending() (model.ScanJobDB, error) {
	var job model.ScanJobDB
	err := orm.ImgMysqlDB.Where("status = ?", model.JobStatusPending).Order("id asc").First(&job).Error
	return job, err
}

func (s *ScanJobService) MarkInterruptedRunning() error {
	return orm.ImgMysqlDB.Model(&model.ScanJobDB{}).
		Where("status = ?", model.JobStatusRunning).
		Updates(map[string]any{
			"status":        model.JobStatusInterrupted,
			"error_message": "service restarted before job finished",
		}).Error
}
