package dao

import (
	"img_process/model"
	"img_process/plugin/orm"
)

type ScanActionItemService struct{}

type actionCountRow struct {
	JobID      uint
	Stage      string
	Status     string
	ActionType string
	Total      int64
}

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
		db = db.Where("action_type in ?", []string{model.ActionTypeDeleteDup, model.ActionTypeDeletePathDup})
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

func (s *ScanActionItemService) ListPendingDeleteByJob(jobID uint) ([]model.ScanActionItemDB, error) {
	var list []model.ScanActionItemDB
	err := orm.ImgMysqlDB.
		Where(
			"job_id = ? AND stage = ? AND status = ? AND action_type in ?",
			jobID,
			model.ActionStageCandidate,
			model.ActionStatusPending,
			[]string{model.ActionTypeDelete, model.ActionTypeDeleteEmptyDir},
		).
		Order("id desc").
		Find(&list).Error
	return list, err
}

func (s *ScanActionItemService) ListPendingDuplicateByJob(jobID uint) ([]model.ScanActionItemDB, error) {
	var list []model.ScanActionItemDB
	err := orm.ImgMysqlDB.
		Where(
			"job_id = ? AND stage = ? AND status = ? AND action_type = ?",
			jobID,
			model.ActionStageCandidate,
			model.ActionStatusPending,
			model.ActionTypeDeleteDup,
		).
		Order("id desc").
		Find(&list).Error
	return list, err
}

func (s *ScanActionItemService) ListPendingPathDuplicateByJob(jobID uint) ([]model.ScanActionItemDB, error) {
	var list []model.ScanActionItemDB
	err := orm.ImgMysqlDB.
		Where(
			"job_id = ? AND stage = ? AND status = ? AND action_type = ?",
			jobID,
			model.ActionStageCandidate,
			model.ActionStatusPending,
			model.ActionTypeDeletePathDup,
		).
		Order("id desc").
		Find(&list).Error
	return list, err
}

func (s *ScanActionItemService) ListPendingByJobAndSourceAndTypes(jobID uint, sourcePath string, actionTypes []string) ([]model.ScanActionItemDB, error) {
	var list []model.ScanActionItemDB
	err := orm.ImgMysqlDB.
		Where(
			"job_id = ? AND source_path = ? AND stage = ? AND status = ? AND action_type in ?",
			jobID,
			sourcePath,
			model.ActionStageCandidate,
			model.ActionStatusPending,
			actionTypes,
		).
		Order("id desc").
		Find(&list).Error
	return list, err
}

func (s *ScanActionItemService) CountPendingByJob(jobID uint) (model.ScanActionCounts, error) {
	var rows []actionCountRow
	err := orm.ImgMysqlDB.Model(&model.ScanActionItemDB{}).
		Select("action_type, count(*) as total").
		Where("job_id = ? AND stage = ? AND status = ?", jobID, model.ActionStageCandidate, model.ActionStatusPending).
		Group("action_type").
		Scan(&rows).Error
	if err != nil {
		return model.ScanActionCounts{}, err
	}

	return countRowsByType(rows), nil
}

func (s *ScanActionItemService) CountGroupedByJob(jobID uint) (model.ScanActionGroupedCounts, error) {
	var rows []actionCountRow
	err := orm.ImgMysqlDB.Model(&model.ScanActionItemDB{}).
		Select("stage, status, action_type, count(*) as total").
		Where("job_id = ?", jobID).
		Group("stage, status, action_type").
		Scan(&rows).Error
	if err != nil {
		return model.ScanActionGroupedCounts{}, err
	}

	grouped := model.ScanActionGroupedCounts{}
	for _, row := range rows {
		addGroupedActionCount(&grouped, row)
	}
	return grouped, nil
}

func (s *ScanActionItemService) CountGroupedByJobs(jobIDs []uint) (map[uint]model.ScanActionGroupedCounts, error) {
	groupedByJob := make(map[uint]model.ScanActionGroupedCounts, len(jobIDs))
	if len(jobIDs) == 0 {
		return groupedByJob, nil
	}

	var rows []actionCountRow
	err := orm.ImgMysqlDB.Model(&model.ScanActionItemDB{}).
		Select("job_id, stage, status, action_type, count(*) as total").
		Where("job_id in ?", jobIDs).
		Group("job_id, stage, status, action_type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, jobID := range jobIDs {
		groupedByJob[jobID] = model.ScanActionGroupedCounts{}
	}
	for _, row := range rows {
		grouped := groupedByJob[row.JobID]
		addGroupedActionCount(&grouped, row)
		groupedByJob[row.JobID] = grouped
	}

	return groupedByJob, nil
}

func addGroupedActionCount(grouped *model.ScanActionGroupedCounts, row actionCountRow) {
	if row.Stage == model.ActionStageCandidate && row.Status == model.ActionStatusPending {
		addActionCount(&grouped.Pending, row.ActionType, row.Total)
	}
	if row.Stage == model.ActionStageExecuted {
		addActionCount(&grouped.Executed, row.ActionType, row.Total)
	}
	if row.Status == model.ActionStatusFailed {
		addActionCount(&grouped.Error, row.ActionType, row.Total)
	}
}

func countRowsByType(rows []actionCountRow) model.ScanActionCounts {
	counts := model.ScanActionCounts{}
	for _, row := range rows {
		addActionCount(&counts, row.ActionType, row.Total)
	}
	return counts
}

func addActionCount(counts *model.ScanActionCounts, actionType string, total int64) {
	switch actionType {
	case model.ActionTypeDelete:
		counts.Delete += total
	case model.ActionTypeDeleteEmptyDir:
		counts.DeleteEmptyDir += total
	case model.ActionTypeMove:
		counts.Move += total
	case model.ActionTypeModifyTime:
		counts.ModifyTime += total
	case model.ActionTypeDeleteDup:
		counts.DeleteDuplicate += total
	case model.ActionTypeDeletePathDup:
		counts.DeletePathDup += total
	case model.ActionTypeRename:
		counts.Rename += total
	default:
		return
	}
	counts.Total = counts.Delete + counts.DeleteEmptyDir + counts.Move + counts.ModifyTime + counts.DeleteDuplicate + counts.DeletePathDup + counts.Rename
}
