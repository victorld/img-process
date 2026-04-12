package dao

import (
	"img_process/model"
	"img_process/plugin/orm"
)

type ScanActionItemService struct{}

func (s *ScanActionItemService) RegisterScanActionItem(item *model.ScanActionItemDB) error {
	return orm.ImgMysqlDB.AutoMigrate(&item)
}

func (s *ScanActionItemService) Create(item *model.ScanActionItemDB) error {
	return orm.ImgMysqlDB.Create(item).Error
}

func (s *ScanActionItemService) Update(item *model.ScanActionItemDB) error {
	return orm.ImgMysqlDB.Save(item).Error
}

func (s *ScanActionItemService) GetByID(id uint) (model.ScanActionItemDB, error) {
	var item model.ScanActionItemDB
	err := orm.ImgMysqlDB.First(&item, id).Error
	return item, err
}

func (s *ScanActionItemService) List(search model.ScanActionItemSearch) ([]model.ScanActionItemDB, int64, error) {
	var (
		list  []model.ScanActionItemDB
		total int64
	)

	db := orm.ImgMysqlDB.Model(&model.ScanActionItemDB{}).Where("job_id = ?", search.JobID).Order("id desc")
	switch search.Tab {
	case "pending":
		db = db.Where("stage = ? AND status = ?", model.ActionStageCandidate, model.ActionStatusPending)
	case "executed":
		db = db.Where("stage = ? AND status in ?", model.ActionStageExecuted, []string{model.ActionStatusSucceeded, model.ActionStatusFailed, model.ActionStatusSkipped})
	case "error":
		db = db.Where("status = ?", model.ActionStatusFailed)
	case "duplicate":
		db = db.Where("action_type = ?", model.ActionTypeDeleteDup)
	}
	if search.ActionType != "" {
		db = db.Where("action_type = ?", search.ActionType)
	}
	if search.Status != "" {
		db = db.Where("status = ?", search.Status)
	}
	if search.Keyword != "" {
		like := "%" + search.Keyword + "%"
		db = db.Where("source_path LIKE ? OR target_path LIKE ? OR reason_text LIKE ? OR error_message LIKE ?", like, like, like, like)
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

func (s *ScanActionItemService) ListByJobAndType(jobID uint, actionType string) ([]model.ScanActionItemDB, error) {
	var list []model.ScanActionItemDB
	err := orm.ImgMysqlDB.Where("job_id = ? AND action_type = ?", jobID, actionType).Find(&list).Error
	return list, err
}
