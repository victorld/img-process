package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"img_process/middleware"
	"img_process/model"
	"img_process/tools"
)

const shootTimeRecalculatedMessage = "拍摄时间已手动变更，候选已重算"
const renameRecalculatedMessage = "文件名已手动变更，候选已重算"
const moveRecalculatedMessage = "位置已手动变更，候选已重算"

var modifyShootDateFunc = middleware.ModifyShootDate
var moveActionMoveFile = tools.MoveFile
var renameActionMoveFile = tools.MoveFile

func (r *AppRuntime) ExecuteDeleteActionItem(jobID uint, itemID uint) error {
	item, err := r.actionItemService.GetByID(itemID)
	if err != nil {
		return err
	}
	if item.JobID != jobID {
		return errors.New("action item does not belong to job")
	}
	if !isPendingDeleteAction(item) {
		return errors.New("action item is not a pending delete action")
	}

	return r.executeDeleteActionItem(&item)
}

func (r *AppRuntime) ExecuteDuplicateActionItem(jobID uint, itemID uint, side string) error {
	item, err := r.actionItemService.GetByID(itemID)
	if err != nil {
		return err
	}
	if item.JobID != jobID {
		return errors.New("action item does not belong to job")
	}
	if !isPendingDuplicateDeleteAction(item) {
		return errors.New("action item is not a pending duplicate delete action")
	}

	return r.executeDuplicateDeleteActionItem(&item, side)
}

func (r *AppRuntime) ExecuteModifyShootTimeActionItem(jobID uint, itemID uint) error {
	item, err := r.actionItemService.GetByID(itemID)
	if err != nil {
		return err
	}
	if item.JobID != jobID {
		return errors.New("action item does not belong to job")
	}
	if !isPendingModifyShootTimeAction(item) {
		return errors.New("action item is not a pending modify_time action")
	}

	targetDate, executedShootDate, err := modifyShootTimeTarget(item)
	if err != nil {
		return err
	}

	err = modifyShootDateFunc(item.SourcePath, executedShootDate)
	if err != nil {
		if saveErr := r.completeModifyShootTimeAction(&item, targetDate, executedShootDate, err); saveErr != nil {
			return saveErr
		}
		r.recordModifyShootTimeEvent(item, "error", "拍摄时间变更失败", err.Error(), ginH(
			"targetDate", targetDate,
			"executedShootDate", executedShootDate,
		))
		return err
	}

	if err := r.completeModifyShootTimeAction(&item, targetDate, executedShootDate, nil); err != nil {
		return err
	}
	if err := r.refreshPendingActionsAfterShootTimeUpdate(item); err != nil {
		r.recordModifyShootTimeEvent(item, "error", "拍摄时间已变更但候选重算失败", err.Error(), ginH(
			"targetDate", targetDate,
			"executedShootDate", executedShootDate,
		))
		return err
	}

	r.recordModifyShootTimeEvent(item, "info", "拍摄时间已变更", fmt.Sprintf("已写入拍摄时间 %s，并重算当前文件候选动作", executedShootDate), ginH(
		"targetDate", targetDate,
		"executedShootDate", executedShootDate,
	))
	return nil
}

func (r *AppRuntime) ExecuteMoveActionItem(jobID uint, itemID uint) error {
	item, err := r.actionItemService.GetByID(itemID)
	if err != nil {
		return err
	}
	if item.JobID != jobID {
		return errors.New("action item does not belong to job")
	}
	if !isPendingMoveAction(item) {
		return errors.New("action item is not a pending move action")
	}

	targetPath, err := moveActionTarget(item)
	if err != nil {
		return err
	}

	err = moveActionMoveFile(item.SourcePath, targetPath)
	if err != nil {
		if saveErr := r.completeMoveAction(&item, targetPath, err); saveErr != nil {
			return saveErr
		}
		r.recordMoveActionEvent(item, targetPath, "error", "位置变更失败", err.Error(), ginH(
			"targetPath", targetPath,
		))
		return err
	}

	if err := r.completeMoveAction(&item, targetPath, nil); err != nil {
		return err
	}
	if err := r.refreshPendingActionsAfterMove(item, targetPath); err != nil {
		r.recordMoveActionEvent(item, targetPath, "error", "位置已变更但候选重算失败", err.Error(), ginH(
			"targetPath", targetPath,
		))
		return err
	}

	r.recordMoveActionEvent(item, targetPath, "info", "位置已变更", fmt.Sprintf("已移动到 %s，并重算当前文件候选动作", targetPath), ginH(
		"targetPath", targetPath,
	))
	return nil
}

func (r *AppRuntime) ExecuteRenameActionItem(jobID uint, itemID uint) error {
	item, err := r.actionItemService.GetByID(itemID)
	if err != nil {
		return err
	}
	if item.JobID != jobID {
		return errors.New("action item does not belong to job")
	}
	if !isPendingRenameAction(item) {
		return errors.New("action item is not a pending rename action")
	}

	targetPath, err := renameActionTarget(item)
	if err != nil {
		return err
	}

	err = renameActionMoveFile(item.SourcePath, targetPath)
	if err != nil {
		if saveErr := r.completeRenameAction(&item, targetPath, err); saveErr != nil {
			return saveErr
		}
		r.recordRenameActionEvent(item, "error", "文件名变更失败", err.Error(), ginH(
			"targetPath", targetPath,
			"targetFileName", filepath.Base(targetPath),
		))
		return err
	}

	if err := r.completeRenameAction(&item, targetPath, nil); err != nil {
		return err
	}
	if err := r.refreshPendingActionsAfterRename(item, targetPath); err != nil {
		r.recordRenameActionEvent(item, "error", "文件名已变更但候选重算失败", err.Error(), ginH(
			"targetPath", targetPath,
			"targetFileName", filepath.Base(targetPath),
		))
		return err
	}

	r.recordRenameActionEvent(item, "info", "文件名已变更", fmt.Sprintf("已重命名为 %s，并重算当前文件候选动作", filepath.Base(targetPath)), ginH(
		"targetPath", targetPath,
		"targetFileName", filepath.Base(targetPath),
	))
	return nil
}

func (r *AppRuntime) ExecuteAllDeleteActionItems(jobID uint) (int, error) {
	items, err := r.actionItemService.ListPendingDeleteByJob(jobID)
	if err != nil {
		return 0, err
	}

	successCount := 0
	for idx := range items {
		if err := r.executeDeleteActionItem(&items[idx]); err != nil {
			return successCount, err
		}
		successCount++
	}

	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:     jobID,
		EventType: model.EventTypeLifecycle,
		Phase:     "post_action",
		Level:     "info",
		Title:     "删除类动作已全部执行",
		Message:   "删除类待执行动作已批量执行完成",
	})
	return successCount, nil
}

func (r *AppRuntime) executeDeleteActionItem(item *model.ScanActionItemDB) error {
	now := time.Now()
	var err error
	switch item.ActionType {
	case model.ActionTypeDelete:
		err = tools.DeleteFile(item.SourcePath)
	case model.ActionTypeDeleteEmptyDir:
		err = tools.DeleteEmptyDir(item.SourcePath)
	default:
		return errors.New("unsupported delete action type")
	}

	item.Stage = model.ActionStageExecuted
	item.ExecutedAt = &now
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			item.Status = model.ActionStatusSkipped
			item.ErrorMessage = "目标已不存在，跳过执行"
		} else {
			item.Status = model.ActionStatusFailed
			item.ErrorMessage = err.Error()
		}
	} else {
		item.Status = model.ActionStatusSucceeded
		item.ErrorMessage = ""
	}

	if saveErr := r.actionItemService.Update(item); saveErr != nil {
		return saveErr
	}

	message := "删除动作执行成功"
	level := "info"
	if err != nil {
		message = item.ErrorMessage
		level = "warn"
	}
	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:       item.JobID,
		EventType:   model.EventTypeLifecycle,
		Phase:       "post_action",
		Level:       level,
		Title:       "删除动作已执行",
		Message:     message,
		RelatedPath: item.SourcePath,
	})
	return nil
}

func (r *AppRuntime) executeDuplicateDeleteActionItem(item *model.ScanActionItemDB, side string) error {
	normalizedSide, deletePath, recommendedDeletePath, err := duplicateDeleteTarget(*item, side)
	if err != nil {
		return err
	}

	err = tools.DeleteFile(deletePath)
	if saveErr := r.completeDuplicateDeleteAction(item, normalizedSide, deletePath, recommendedDeletePath, err); saveErr != nil {
		return saveErr
	}

	message := "重复项删除执行成功"
	level := "info"
	if err != nil {
		message = item.ErrorMessage
		level = "warn"
	}
	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:       item.JobID,
		EventType:   model.EventTypeLifecycle,
		Phase:       "post_action",
		Level:       level,
		Title:       "重复项删除已执行",
		Message:     message,
		RelatedPath: deletePath,
	})
	return nil
}

func (r *AppRuntime) completeDuplicateDeleteAction(
	item *model.ScanActionItemDB,
	side string,
	deletePath string,
	recommendedDeletePath string,
	err error,
) error {
	now := time.Now()
	item.Stage = model.ActionStageExecuted
	item.ExecutedAt = &now
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			item.Status = model.ActionStatusSkipped
			item.ErrorMessage = "目标已不存在，跳过执行"
		} else {
			item.Status = model.ActionStatusFailed
			item.ErrorMessage = err.Error()
		}
	} else {
		item.Status = model.ActionStatusSucceeded
		item.ErrorMessage = ""
	}
	item.MetadataJSON = appendDuplicateExecutionAudit(item.MetadataJSON, side, deletePath, recommendedDeletePath)
	return r.actionItemService.Update(item)
}

func isPendingDeleteAction(item model.ScanActionItemDB) bool {
	if item.Stage != model.ActionStageCandidate || item.Status != model.ActionStatusPending {
		return false
	}
	return item.ActionType == model.ActionTypeDelete || item.ActionType == model.ActionTypeDeleteEmptyDir
}

func isPendingModifyShootTimeAction(item model.ScanActionItemDB) bool {
	if item.Stage != model.ActionStageCandidate || item.Status != model.ActionStatusPending {
		return false
	}
	return item.ActionType == model.ActionTypeModifyTime
}

func isPendingMoveAction(item model.ScanActionItemDB) bool {
	if item.Stage != model.ActionStageCandidate || item.Status != model.ActionStatusPending {
		return false
	}
	return item.ActionType == model.ActionTypeMove
}

func isPendingRenameAction(item model.ScanActionItemDB) bool {
	if item.Stage != model.ActionStageCandidate || item.Status != model.ActionStatusPending {
		return false
	}
	return item.ActionType == model.ActionTypeRename
}

func isPendingDuplicateDeleteAction(item model.ScanActionItemDB) bool {
	if item.Stage != model.ActionStageCandidate || item.Status != model.ActionStatusPending {
		return false
	}
	return item.ActionType == model.ActionTypeDeleteDup
}

func duplicateDeleteTarget(item model.ScanActionItemDB, side string) (string, string, string, error) {
	normalizedSide := strings.ToUpper(strings.TrimSpace(side))
	if normalizedSide != "A" && normalizedSide != "B" {
		return "", "", "", errors.New("invalid duplicate side")
	}

	metadata := parseActionItemMetadata(item)
	recommendedDeletePath := firstNonEmpty(metadata.CurrentPath, item.SourcePath)
	keepPath := firstNonEmpty(metadata.KeepPath, item.TargetPath)

	deletePath := recommendedDeletePath
	if normalizedSide == "B" {
		deletePath = keepPath
	}
	if strings.TrimSpace(deletePath) == "" {
		return "", "", "", errors.New("duplicate delete path is empty")
	}

	return normalizedSide, deletePath, recommendedDeletePath, nil
}

func appendDuplicateExecutionAudit(metadataJSON string, side string, deletePath string, recommendedDeletePath string) string {
	payload := map[string]any{}
	if strings.TrimSpace(metadataJSON) != "" {
		_ = json.Unmarshal([]byte(metadataJSON), &payload)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payload["executedDeleteSide"] = side
	payload["executedDeletePath"] = deletePath
	payload["recommendedDeletePath"] = recommendedDeletePath
	payload["manualOverride"] = side == "B"
	return tools.MarshalJsonToString(payload)
}

func moveActionTarget(item model.ScanActionItemDB) (string, error) {
	metadata := parseActionItemMetadata(item)
	targetPath := strings.TrimSpace(firstNonEmpty(metadata.TargetPath, item.TargetPath))
	if targetPath == "" {
		return "", errors.New("move target path is empty")
	}
	return targetPath, nil
}

func renameActionTarget(item model.ScanActionItemDB) (string, error) {
	metadata := parseActionItemMetadata(item)
	targetPath := strings.TrimSpace(firstNonEmpty(metadata.TargetPath, item.TargetPath))
	if targetPath == "" {
		return "", errors.New("rename target path is empty")
	}
	return targetPath, nil
}

func modifyShootTimeTarget(item model.ScanActionItemDB) (string, string, error) {
	metadata := parseActionItemMetadata(item)
	targetDate := strings.TrimSpace(firstNonEmpty(metadata.TargetDate, metadata.MinDate))
	if targetDate == "" {
		return "", "", errors.New("target date is empty")
	}
	executedShootDate, err := formatShootDateForExif(targetDate)
	if err != nil {
		return "", "", err
	}
	return targetDate, executedShootDate, nil
}

func formatShootDateForExif(targetDate string) (string, error) {
	cleanTargetDate := strings.TrimSpace(targetDate)
	if cleanTargetDate == "" {
		return "", errors.New("target date is empty")
	}
	if _, err := time.Parse("2006-01-02", cleanTargetDate); err != nil {
		return "", fmt.Errorf("target date format is invalid: %w", err)
	}
	return strings.ReplaceAll(cleanTargetDate, "-", ":") + " 00:00:00", nil
}

func (r *AppRuntime) completeModifyShootTimeAction(
	item *model.ScanActionItemDB,
	targetDate string,
	executedShootDate string,
	err error,
) error {
	now := time.Now()
	item.Stage = model.ActionStageExecuted
	item.ExecutedAt = &now
	item.MetadataJSON = mergeJSON(item.MetadataJSON, ginH(
		"targetDate", targetDate,
		"executedShootDate", executedShootDate,
	))
	if err != nil {
		item.Status = model.ActionStatusFailed
		item.ErrorMessage = err.Error()
		return r.actionItemService.Update(item)
	}

	item.Status = model.ActionStatusSucceeded
	item.ErrorMessage = ""
	item.MetadataJSON = mergeJSON(item.MetadataJSON, ginH(
		"shootDate", targetDate,
		"shootDateRaw", executedShootDate,
	))
	return r.actionItemService.Update(item)
}

func (r *AppRuntime) completeMoveAction(
	item *model.ScanActionItemDB,
	targetPath string,
	err error,
) error {
	now := time.Now()
	item.Stage = model.ActionStageExecuted
	item.ExecutedAt = &now
	item.MetadataJSON = mergeJSON(item.MetadataJSON, ginH(
		"targetPath", targetPath,
	))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			item.Status = model.ActionStatusSkipped
			item.ErrorMessage = "源文件已不存在，跳过执行"
		} else {
			item.Status = model.ActionStatusFailed
			item.ErrorMessage = err.Error()
		}
		return r.actionItemService.Update(item)
	}

	item.Status = model.ActionStatusSucceeded
	item.ErrorMessage = ""
	return r.actionItemService.Update(item)
}

func (r *AppRuntime) completeRenameAction(
	item *model.ScanActionItemDB,
	targetPath string,
	err error,
) error {
	now := time.Now()
	item.Stage = model.ActionStageExecuted
	item.ExecutedAt = &now
	item.MetadataJSON = mergeJSON(item.MetadataJSON, ginH(
		"targetPath", targetPath,
		"targetFileName", filepath.Base(targetPath),
	))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			item.Status = model.ActionStatusSkipped
			item.ErrorMessage = "源文件已不存在，跳过执行"
		} else {
			item.Status = model.ActionStatusFailed
			item.ErrorMessage = err.Error()
		}
		return r.actionItemService.Update(item)
	}

	item.Status = model.ActionStatusSucceeded
	item.ErrorMessage = ""
	return r.actionItemService.Update(item)
}

func (r *AppRuntime) refreshPendingActionsAfterShootTimeUpdate(item model.ScanActionItemDB) error {
	relatedItems, err := r.actionItemService.ListPendingByJobAndSourceAndTypes(item.JobID, item.SourcePath, []string{
		model.ActionTypeMove,
		model.ActionTypeRename,
		model.ActionTypeModifyTime,
	})
	if err != nil {
		return err
	}

	skippedAt := time.Now()
	for idx := range relatedItems {
		if relatedItems[idx].ID == item.ID {
			continue
		}
		if err := r.markActionSkippedForRecalculation(&relatedItems[idx], skippedAt); err != nil {
			return err
		}
	}

	return r.rebuildPendingActionsForPath(item.JobID, item.SourcePath)
}

func (r *AppRuntime) refreshPendingActionsAfterMove(item model.ScanActionItemDB, targetPath string) error {
	relatedItems, err := r.actionItemService.ListPendingByJobAndSourceAndTypes(item.JobID, item.SourcePath, []string{
		model.ActionTypeMove,
		model.ActionTypeRename,
		model.ActionTypeModifyTime,
	})
	if err != nil {
		return err
	}

	skippedAt := time.Now()
	for idx := range relatedItems {
		if relatedItems[idx].ID == item.ID {
			continue
		}
		if err := r.markActionSkippedForMoveRecalculation(&relatedItems[idx], skippedAt); err != nil {
			return err
		}
	}

	return r.rebuildPendingActionsForPath(item.JobID, targetPath)
}

func (r *AppRuntime) refreshPendingActionsAfterRename(item model.ScanActionItemDB, targetPath string) error {
	relatedItems, err := r.actionItemService.ListPendingByJobAndSourceAndTypes(item.JobID, item.SourcePath, []string{
		model.ActionTypeMove,
		model.ActionTypeRename,
		model.ActionTypeModifyTime,
	})
	if err != nil {
		return err
	}

	skippedAt := time.Now()
	for idx := range relatedItems {
		if relatedItems[idx].ID == item.ID {
			continue
		}
		if err := r.markActionSkippedForRenameRecalculation(&relatedItems[idx], skippedAt); err != nil {
			return err
		}
	}

	return r.rebuildPendingActionsForPath(item.JobID, targetPath)
}

func (r *AppRuntime) rebuildPendingActionsForPath(jobID uint, sourcePath string) error {
	job, err := r.jobService.GetByID(jobID)
	if err != nil {
		return err
	}

	var scanArgs model.DoScanImgArg
	if strings.TrimSpace(job.ScanArgs) != "" {
		if err := json.Unmarshal([]byte(job.ScanArgs), &scanArgs); err != nil {
			return err
		}
	}
	scanArgs = NormalizeScanArgs(scanArgs)
	startPath := ""
	if scanArgs.StartPath != nil {
		startPath = strings.TrimSpace(*scanArgs.StartPath)
	}
	basePath, err := resolveBasePath(startPath)
	if err != nil {
		return err
	}

	recorder := NewDBScanRecorder(jobID)
	scanner := newScannerWithRecorder(scanArgs, recorder)
	scanner.basePath = basePath

	shootDateOrigin, locStreet, _ := scanner.getImgShootDateAndLoc(sourcePath)
	meta := collectFileMetadata(sourcePath, shootDateOrigin, locStreet)
	scanner.applyFileDecision(evaluateFileDecision(meta, scanner.basePath))
	return nil
}

func (r *AppRuntime) markActionSkippedForRecalculation(item *model.ScanActionItemDB, executedAt time.Time) error {
	applySkippedRecalculationState(item, executedAt)
	return r.actionItemService.Update(item)
}

func (r *AppRuntime) markActionSkippedForMoveRecalculation(item *model.ScanActionItemDB, executedAt time.Time) error {
	applySkippedMoveRecalculationState(item, executedAt)
	return r.actionItemService.Update(item)
}

func (r *AppRuntime) markActionSkippedForRenameRecalculation(item *model.ScanActionItemDB, executedAt time.Time) error {
	applySkippedRenameRecalculationState(item, executedAt)
	return r.actionItemService.Update(item)
}

func applySkippedRecalculationState(item *model.ScanActionItemDB, executedAt time.Time) {
	item.Stage = model.ActionStageExecuted
	item.Status = model.ActionStatusSkipped
	item.ExecutedAt = &executedAt
	item.ErrorMessage = shootTimeRecalculatedMessage
	item.MetadataJSON = mergeJSON(item.MetadataJSON, ginH(
		"recalculatedAfterShootTimeUpdate", true,
	))
}

func applySkippedMoveRecalculationState(item *model.ScanActionItemDB, executedAt time.Time) {
	item.Stage = model.ActionStageExecuted
	item.Status = model.ActionStatusSkipped
	item.ExecutedAt = &executedAt
	item.ErrorMessage = moveRecalculatedMessage
	item.MetadataJSON = mergeJSON(item.MetadataJSON, ginH(
		"recalculatedAfterMoveUpdate", true,
	))
}

func applySkippedRenameRecalculationState(item *model.ScanActionItemDB, executedAt time.Time) {
	item.Stage = model.ActionStageExecuted
	item.Status = model.ActionStatusSkipped
	item.ExecutedAt = &executedAt
	item.ErrorMessage = renameRecalculatedMessage
	item.MetadataJSON = mergeJSON(item.MetadataJSON, ginH(
		"recalculatedAfterRenameUpdate", true,
	))
}

func (r *AppRuntime) recordModifyShootTimeEvent(
	item model.ScanActionItemDB,
	level string,
	title string,
	message string,
	payload map[string]any,
) {
	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:       item.JobID,
		EventType:   model.EventTypeLifecycle,
		Phase:       "post_action",
		Level:       level,
		Title:       title,
		Message:     message,
		RelatedPath: item.SourcePath,
		PayloadJSON: marshalPayload(payload),
	})
	_ = r.jobLogService.Create(&model.ScanJobLogDB{
		JobID:       item.JobID,
		Level:       level,
		Phase:       "post_action",
		Message:     title,
		PayloadJSON: marshalPayload(payload),
	})
}

func (r *AppRuntime) recordMoveActionEvent(
	item model.ScanActionItemDB,
	targetPath string,
	level string,
	title string,
	message string,
	payload map[string]any,
) {
	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:       item.JobID,
		EventType:   model.EventTypeLifecycle,
		Phase:       "post_action",
		Level:       level,
		Title:       title,
		Message:     message,
		RelatedPath: firstNonEmpty(targetPath, item.SourcePath),
		PayloadJSON: marshalPayload(payload),
	})
	_ = r.jobLogService.Create(&model.ScanJobLogDB{
		JobID:       item.JobID,
		Level:       level,
		Phase:       "post_action",
		Message:     title,
		PayloadJSON: marshalPayload(payload),
	})
}

func (r *AppRuntime) recordRenameActionEvent(
	item model.ScanActionItemDB,
	level string,
	title string,
	message string,
	payload map[string]any,
) {
	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:       item.JobID,
		EventType:   model.EventTypeLifecycle,
		Phase:       "post_action",
		Level:       level,
		Title:       title,
		Message:     message,
		RelatedPath: item.SourcePath,
		PayloadJSON: marshalPayload(payload),
	})
	_ = r.jobLogService.Create(&model.ScanJobLogDB{
		JobID:       item.JobID,
		Level:       level,
		Phase:       "post_action",
		Message:     title,
		PayloadJSON: marshalPayload(payload),
	})
}
