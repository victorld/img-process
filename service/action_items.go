package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"img_process/cons"
	"img_process/model"
	"img_process/tools"
)

type actionItemMetadata struct {
	FileName               string                    `json:"fileName"`
	CurrentPath            string                    `json:"currentPath"`
	TargetPath             string                    `json:"targetPath"`
	TargetFileName         string                    `json:"targetFileName"`
	DirDate                string                    `json:"dirDate"`
	ModifyDate             string                    `json:"modifyDate"`
	FileNameDate           string                    `json:"fileNameDate"`
	ShootDate              string                    `json:"shootDate"`
	ShootDateRaw           string                    `json:"shootDateRaw"`
	MinDate                string                    `json:"minDate"`
	TargetDate             string                    `json:"targetDate"`
	ExecutedShootDate      string                    `json:"executedShootDate"`
	KeepPath               string                    `json:"keepPath"`
	KeepFileName           string                    `json:"keepFileName"`
	SizeMatch              bool                      `json:"sizeMatch"`
	DeleteEligible         *bool                     `json:"deleteEligible"`
	DeleteIneligibleReason string                    `json:"deleteIneligibleReason"`
	DuplicatePhotos        []model.ScanActionPreview `json:"duplicatePhotos"`
	ExecutedDeleteSide     string                    `json:"executedDeleteSide"`
	ExecutedDeletePath     string                    `json:"executedDeletePath"`
	RecommendedDeletePath  string                    `json:"recommendedDeletePath"`
	MatchKey               string                    `json:"matchKey"`
	MatchType              string                    `json:"matchType"`
}

func (r *AppRuntime) ListActionItems(search model.ScanActionItemSearch) ([]model.ScanActionItemView, model.ScanActionCounts, model.ScanActionGroupedCounts, int64, error) {
	groupedCounts, err := r.actionItemService.CountGroupedByJob(search.JobID)
	if err != nil {
		return nil, model.ScanActionCounts{}, model.ScanActionGroupedCounts{}, 0, err
	}
	groupedCounts, err = r.refineDuplicateGroupedCounts(search.JobID, groupedCounts)
	if err != nil {
		return nil, model.ScanActionCounts{}, model.ScanActionGroupedCounts{}, 0, err
	}

	if isDuplicateActionType(search.ActionType) {
		views, total, groupedCounts, err := r.listDuplicateActionItemViews(search, groupedCounts)
		if err != nil {
			return nil, model.ScanActionCounts{}, model.ScanActionGroupedCounts{}, 0, err
		}
		return views, countsForActionTab(groupedCounts, search.Tab), groupedCounts, total, nil
	}

	list, total, err := r.actionItemService.List(search)
	if err != nil {
		return nil, model.ScanActionCounts{}, model.ScanActionGroupedCounts{}, 0, err
	}

	views := make([]model.ScanActionItemView, 0, len(list))
	for _, item := range list {
		views = append(views, buildActionItemView(item))
	}

	return views, countsForActionTab(groupedCounts, search.Tab), groupedCounts, total, nil
}

func (r *AppRuntime) listDuplicateActionItemViews(search model.ScanActionItemSearch, groupedCounts model.ScanActionGroupedCounts) ([]model.ScanActionItemView, int64, model.ScanActionGroupedCounts, error) {
	allSearch := search
	allSearch.Page = 0
	allSearch.PageSize = 0
	list, total, err := r.actionItemService.List(allSearch)
	if err != nil {
		return nil, 0, groupedCounts, err
	}

	var excludedPaths map[string]struct{}
	if search.Tab == "pending" {
		excludedPaths, err = r.executedDuplicateDeletePaths(search.JobID, search.ActionType)
		if err != nil {
			return nil, 0, groupedCounts, err
		}
	}
	views := buildDuplicateGroupViewsWithExcludedPaths(list, excludedPaths)
	total = int64(len(views))
	if shouldUseLegacyDuplicateCompare(search, total, groupedCounts) {
		legacyViews, err := r.legacyDuplicateCompareViews(search.JobID)
		if err != nil {
			return nil, 0, groupedCounts, err
		}
		views = legacyViews
		total = int64(len(legacyViews))
	}

	viewTotal := countDuplicateViews(views)
	switch search.Tab {
	case "pending":
		replaceDuplicateCountForType(&groupedCounts.Pending, search.ActionType, viewTotal)
	case "executed":
		replaceDuplicateCountForType(&groupedCounts.Executed, search.ActionType, viewTotal)
	case "error":
		replaceDuplicateCountForType(&groupedCounts.Error, search.ActionType, viewTotal)
	}

	pagedViews, pagedTotal := paginateActionViews(views, search.Page, search.PageSize)
	if isDuplicateActionType(search.ActionType) {
		pagedTotal = viewTotal
	}
	return pagedViews, pagedTotal, groupedCounts, nil
}

func buildDuplicateGroupViews(items []model.ScanActionItemDB) []model.ScanActionItemView {
	return buildDuplicateGroupViewsWithExcludedPaths(items, nil)
}

func buildDuplicateGroupViewsWithExcludedPaths(items []model.ScanActionItemDB, excludedPaths map[string]struct{}) []model.ScanActionItemView {
	views := make([]model.ScanActionItemView, 0, len(items))
	groupIndex := map[string]int{}
	for _, item := range items {
		if item.Stage != model.ActionStageExecuted {
			if _, ok := excludedPaths[cleanPathKey(item.SourcePath)]; ok {
				continue
			}
		}
		view := buildActionItemView(item)
		if item.Stage != model.ActionStageExecuted {
			view = filterDuplicateViewPhotos(view, excludedPaths)
			if len(view.DuplicatePhotos) == 0 {
				continue
			}
		}
		if item.Stage == model.ActionStageExecuted {
			views = append(views, view)
			continue
		}
		key := duplicateGroupViewKey(item)
		index, ok := groupIndex[key]
		if !ok {
			groupIndex[key] = len(views)
			views = append(views, view)
			continue
		}
		views[index] = mergeDuplicateGroupView(views[index], view)
	}
	return views
}

func filterDuplicateViewPhotos(view model.ScanActionItemView, excludedPaths map[string]struct{}) model.ScanActionItemView {
	if len(excludedPaths) == 0 || len(view.DuplicatePhotos) == 0 {
		return view
	}
	photos := make([]model.ScanActionPreview, 0, len(view.DuplicatePhotos))
	for _, photo := range view.DuplicatePhotos {
		if _, ok := excludedPaths[cleanPathKey(photo.Path)]; ok {
			continue
		}
		photos = append(photos, photo)
	}
	view.DuplicatePhotos = photos
	photoA, photoB := duplicatePairFromPhotos(photos)
	view.Pair = &model.ScanActionPair{PhotoA: photoA, PhotoB: photoB}
	return view
}

func (r *AppRuntime) refineDuplicateGroupedCounts(jobID uint, groupedCounts model.ScanActionGroupedCounts) (model.ScanActionGroupedCounts, error) {
	list, err := r.allDuplicateActionItems(jobID)
	if err != nil {
		return groupedCounts, err
	}
	if len(list) == 0 {
		return groupedCounts, nil
	}

	for _, actionType := range []string{model.ActionTypeDeleteDup, model.ActionTypeDeletePathDup} {
		typeItems := filterActionItemsByType(list, actionType)
		excludedPaths := duplicateExecutedDeletePaths(typeItems)
		pendingItems := make([]model.ScanActionItemDB, 0, len(typeItems))
		executedItems := make([]model.ScanActionItemDB, 0, len(typeItems))
		errorItems := make([]model.ScanActionItemDB, 0, len(typeItems))
		for _, item := range typeItems {
			if item.Stage == model.ActionStageCandidate && item.Status == model.ActionStatusPending {
				pendingItems = append(pendingItems, item)
			}
			if item.Stage == model.ActionStageExecuted {
				executedItems = append(executedItems, item)
			}
			if item.Status == model.ActionStatusFailed {
				errorItems = append(errorItems, item)
			}
		}

		replaceDuplicateCountForType(&groupedCounts.Pending, actionType, countDuplicateViews(buildDuplicateGroupViewsWithExcludedPaths(pendingItems, excludedPaths)))
		replaceDuplicateCountForType(&groupedCounts.Executed, actionType, countDuplicateViews(buildDuplicateGroupViews(executedItems)))
		replaceDuplicateCountForType(&groupedCounts.Error, actionType, countDuplicateViews(buildDuplicateGroupViews(errorItems)))
	}
	return groupedCounts, nil
}

func filterActionItemsByType(items []model.ScanActionItemDB, actionType string) []model.ScanActionItemDB {
	filtered := make([]model.ScanActionItemDB, 0, len(items))
	for _, item := range items {
		if item.ActionType == actionType {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (r *AppRuntime) executedDuplicateDeletePaths(jobID uint, actionType string) (map[string]struct{}, error) {
	list, err := r.allDuplicateActionItems(jobID)
	if err != nil {
		return nil, err
	}
	return duplicateExecutedDeletePaths(filterActionItemsByType(list, actionType)), nil
}

func (r *AppRuntime) allDuplicateActionItems(jobID uint) ([]model.ScanActionItemDB, error) {
	search := model.ScanActionItemSearch{
		JobID: jobID,
		Tab:   "duplicate",
	}
	list, _, err := r.actionItemService.List(search)
	return list, err
}

func duplicateExecutedDeletePaths(items []model.ScanActionItemDB) map[string]struct{} {
	paths := map[string]struct{}{}
	for _, item := range items {
		if item.Stage != model.ActionStageExecuted {
			continue
		}
		metadata := parseActionItemMetadata(item)
		deletePath := firstNonEmpty(metadata.ExecutedDeletePath, metadata.RecommendedDeletePath, item.SourcePath)
		if key := cleanPathKey(deletePath); key != "" {
			paths[key] = struct{}{}
		}
	}
	return paths
}

func countDuplicateViews(views []model.ScanActionItemView) int64 {
	return int64(len(views))
}

func cleanPathKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return filepath.Clean(value)
}

func duplicateGroupViewKey(item model.ScanActionItemDB) string {
	if strings.TrimSpace(item.DuplicateGroup) != "" {
		return item.DuplicateGroup
	}
	return strconv.FormatUint(uint64(item.ID), 10)
}

func mergeDuplicateGroupView(base model.ScanActionItemView, next model.ScanActionItemView) model.ScanActionItemView {
	if len(base.DuplicatePhotos) == 0 && len(next.DuplicatePhotos) > 0 {
		base.DuplicatePhotos = next.DuplicatePhotos
		base.Pair = next.Pair
		base.MetadataJSON = next.MetadataJSON
		base.SourcePath = firstNonEmpty(base.SourcePath, next.SourcePath)
		base.TargetPath = firstNonEmpty(base.TargetPath, next.TargetPath)
	}
	if base.DuplicateMeta == nil && next.DuplicateMeta != nil {
		base.DuplicateMeta = next.DuplicateMeta
	}
	if base.DiscoveredAt == nil || (next.DiscoveredAt != nil && next.DiscoveredAt.Before(*base.DiscoveredAt)) {
		base.DiscoveredAt = next.DiscoveredAt
	}
	return base
}

func replaceDeleteDuplicateCount(counts *model.ScanActionCounts, groupTotal int64) {
	counts.Total = counts.Total - counts.DeleteDuplicate + groupTotal
	counts.DeleteDuplicate = groupTotal
}

func replaceDeletePathDuplicateCount(counts *model.ScanActionCounts, groupTotal int64) {
	counts.Total = counts.Total - counts.DeletePathDup + groupTotal
	counts.DeletePathDup = groupTotal
}

func replaceDuplicateCountForType(counts *model.ScanActionCounts, actionType string, groupTotal int64) {
	switch actionType {
	case model.ActionTypeDeletePathDup:
		replaceDeletePathDuplicateCount(counts, groupTotal)
	default:
		replaceDeleteDuplicateCount(counts, groupTotal)
	}
}

func isDuplicateActionType(actionType string) bool {
	return actionType == model.ActionTypeDeleteDup || actionType == model.ActionTypeDeletePathDup
}

func countsForActionTab(groupedCounts model.ScanActionGroupedCounts, tab string) model.ScanActionCounts {
	switch tab {
	case "executed":
		return groupedCounts.Executed
	case "error":
		return groupedCounts.Error
	default:
		return groupedCounts.Pending
	}
}

func (r *AppRuntime) ResolveActionPreview(jobID uint, itemID uint, slot string) (string, error) {
	view, err := r.getActionItemViewForPreview(jobID, itemID)
	if err != nil {
		return "", err
	}
	path := previewPathFromView(view, slot)
	if path == "" {
		return "", errors.New("preview path not found")
	}

	job, err := r.jobService.GetByID(jobID)
	if err != nil {
		return "", err
	}

	allowedRoots := collectAllowedPreviewRoots(job)
	if !isPathInRoots(path, allowedRoots) {
		return "", errors.New("preview path is outside allowed roots")
	}

	if stat, err := os.Stat(path); err != nil || stat.IsDir() {
		if err != nil {
			return "", err
		}
		return "", errors.New("preview path is not a file")
	}

	return path, nil
}

func (r *AppRuntime) getActionItemViewForPreview(jobID uint, itemID uint) (model.ScanActionItemView, error) {
	item, err := r.actionItemService.GetByID(itemID)
	if err == nil {
		if item.JobID != jobID {
			return model.ScanActionItemView{}, errors.New("action item does not belong to job")
		}
		return buildActionItemView(item), nil
	}

	legacyViews, legacyErr := r.legacyDuplicateCompareViews(jobID)
	if legacyErr != nil {
		return model.ScanActionItemView{}, legacyErr
	}
	for _, view := range legacyViews {
		if view.ID == itemID {
			return view, nil
		}
	}
	return model.ScanActionItemView{}, err
}

func buildActionItemView(item model.ScanActionItemDB) model.ScanActionItemView {
	metadata := parseActionItemMetadata(item)
	sourcePath := strings.TrimSpace(item.SourcePath)
	targetPath := strings.TrimSpace(item.TargetPath)
	view := model.ScanActionItemView{
		ID:             item.ID,
		ActionType:     item.ActionType,
		ObjectType:     item.ObjectType,
		SourcePath:     sourcePath,
		TargetPath:     targetPath,
		ReasonCode:     item.ReasonCode,
		ReasonText:     item.ReasonText,
		Stage:          item.Stage,
		Status:         item.Status,
		DiscoveredAt:   item.DiscoveredAt,
		ExecutedAt:     item.ExecutedAt,
		ErrorMessage:   item.ErrorMessage,
		MetadataJSON:   item.MetadataJSON,
		DuplicateGroup: item.DuplicateGroup,
	}

	switch item.ActionType {
	case model.ActionTypeDeleteDup, model.ActionTypeDeletePathDup:
		keepPath := metadata.KeepPath
		if keepPath == "" {
			keepPath = item.TargetPath
		}
		duplicatePhotos := metadata.DuplicatePhotos
		if len(duplicatePhotos) == 0 {
			duplicatePhotos = legacyPairPhotosAsList(item, metadata, sourcePath, keepPath)
		}
		duplicatePhotos = duplicatePhotosForActionState(item, metadata, duplicatePhotos)
		duplicatePhotos = enrichDuplicatePhotos(duplicatePhotos, item.DuplicateGroup, true)
		if item.ActionType == model.ActionTypeDeletePathDup {
			duplicatePhotos = markPathDuplicatePhotos(duplicatePhotos, firstNonEmpty(metadata.MatchKey, item.DuplicateGroup))
			duplicatePhotos = applyPathDuplicateRecommendation(duplicatePhotos)
		}
		view.Pair = &model.ScanActionPair{
			PhotoA: model.ScanActionPreview{
				FileName:    firstNonEmpty(metadata.FileName, filepath.Base(sourcePath)),
				Path:        firstNonEmpty(metadata.CurrentPath, sourcePath),
				PreviewSlot: "pair_a",
			},
			PhotoB: model.ScanActionPreview{
				FileName:    firstNonEmpty(metadata.KeepFileName, filepath.Base(keepPath)),
				Path:        keepPath,
				PreviewSlot: "pair_b",
			},
		}
		deleteEligible := true
		if metadata.DeleteEligible != nil {
			deleteEligible = *metadata.DeleteEligible
		}
		view.DuplicateMeta = &model.ScanActionDuplicateMeta{
			SizeMatch:              metadata.SizeMatch,
			DeleteEligible:         deleteEligible,
			DeleteIneligibleReason: metadata.DeleteIneligibleReason,
		}
		view.DuplicatePhotos = duplicatePhotos
	default:
		view.Detail = &model.ScanActionDetail{
			FileName:       firstNonEmpty(metadata.FileName, filepath.Base(sourcePath)),
			CurrentPath:    firstNonEmpty(metadata.CurrentPath, sourcePath),
			TargetPath:     firstNonEmpty(metadata.TargetPath, targetPath),
			TargetFileName: firstNonEmpty(metadata.TargetFileName, fileNameOrEmpty(metadata.TargetPath), fileNameOrEmpty(targetPath)),
			DirDate:        metadata.DirDate,
			ModifyDate:     metadata.ModifyDate,
			FileNameDate:   metadata.FileNameDate,
			ShootDate:      metadata.ShootDate,
			ShootDateRaw:   metadata.ShootDateRaw,
			MinDate:        metadata.MinDate,
			PreviewSlot:    "source",
		}
	}

	return view
}

func shouldUseLegacyDuplicateCompare(search model.ScanActionItemSearch, total int64, groupedCounts model.ScanActionGroupedCounts) bool {
	return search.Tab == "pending" &&
		search.ActionType == model.ActionTypeDeleteDup &&
		total == 0 &&
		groupedCounts.Pending.DeleteDuplicate == 0 &&
		groupedCounts.Executed.DeleteDuplicate == 0 &&
		groupedCounts.Error.DeleteDuplicate == 0
}

func (r *AppRuntime) legacyDuplicateCompareViews(jobID uint) ([]model.ScanActionItemView, error) {
	job, err := r.jobService.GetByID(jobID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(job.ScanUUID) == "" {
		return nil, nil
	}
	comparePath := filepath.Join(cons.WorkDir, "log", "dump_delete_file", job.ScanUUID, "dump_compare")
	lines, err := tools.ReadFileLines(comparePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	views := make([]model.ScanActionItemView, 0, len(lines))
	pathResolver := newLegacyDuplicatePathResolver(job)
	for index, line := range lines {
		view, ok := legacyDuplicateCompareLineView(jobID, job.ScanUUID, index, line, job.StartAt, pathResolver)
		if ok {
			views = append(views, view)
		}
	}
	return views, nil
}

func legacyDuplicateCompareLineView(jobID uint, scanUUID string, index int, line string, discoveredAt *time.Time, resolver func(string) []string) (model.ScanActionItemView, bool) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return model.ScanActionItemView{}, false
	}
	group := strings.TrimSpace(parts[0])
	rawNames := strings.Split(parts[1], "|")
	names := make([]string, 0, len(rawNames))
	for _, rawName := range rawNames {
		name := strings.TrimSpace(rawName)
		if name != "" {
			names = append(names, name)
		}
	}
	if group == "" || len(names) == 0 {
		return model.ScanActionItemView{}, false
	}
	photos := make([]model.ScanActionPreview, 0, len(names))
	for _, name := range names {
		matches := []string{}
		if resolver != nil {
			matches = resolver(name)
		}
		if len(matches) == 0 {
			photos = append(photos, model.ScanActionPreview{
				FileName:          name,
				Path:              "",
				PathSource:        "未找到文件",
				MD5Matched:        true,
				DeleteEligible:    false,
				RecommendedDelete: false,
			})
			continue
		}
		for matchIndex, match := range matches {
			photos = append(photos, duplicatePhotoPreview(name, match, false, "文件系统搜索", true, matchIndex, len(matches)))
		}
	}
	photos = uniqueDuplicatePhotoPreviews(photos)
	photos = enrichDuplicatePhotos(photos, group, false)
	photoA, photoB := duplicatePairFromPhotos(photos)
	return model.ScanActionItemView{
		ID:             legacyDuplicateViewID(index),
		ActionType:     model.ActionTypeDeleteDup,
		ObjectType:     model.ActionObjectFile,
		SourcePath:     firstPhotoPath(photos),
		TargetPath:     "",
		ReasonCode:     "legacy_dump_compare",
		ReasonText:     "历史扫描发现重复项",
		Stage:          model.ActionStageDiscovery,
		Status:         model.ActionStatusSkipped,
		DiscoveredAt:   discoveredAt,
		DuplicateGroup: group,
		Pair: &model.ScanActionPair{
			PhotoA: photoA,
			PhotoB: photoB,
		},
		DuplicatePhotos: photos,
		DuplicateMeta: &model.ScanActionDuplicateMeta{
			SizeMatch:              true,
			DeleteEligible:         true,
			DeleteIneligibleReason: "",
		},
		MetadataJSON: tools.MarshalJsonToString(map[string]any{
			"scanUuid":                   scanUUID,
			"deleteEligible":             true,
			"legacyDumpCompare":          true,
			"legacyDumpCompareFileNames": names,
			"duplicatePhotos":            photos,
		}),
	}, true
}

func legacyDuplicateViewID(index int) uint {
	return uint(900000000 + index + 1)
}

func paginateActionViews(views []model.ScanActionItemView, page int, pageSize int) ([]model.ScanActionItemView, int64) {
	total := int64(len(views))
	if pageSize <= 0 {
		return views, total
	}
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start >= len(views) {
		return []model.ScanActionItemView{}, total
	}
	end := start + pageSize
	if end > len(views) {
		end = len(views)
	}
	return views[start:end], total
}

func parseActionItemMetadata(item model.ScanActionItemDB) actionItemMetadata {
	metadata := actionItemMetadata{}
	sourcePath := strings.TrimSpace(item.SourcePath)
	targetPath := strings.TrimSpace(item.TargetPath)
	if item.MetadataJSON != "" {
		_ = json.Unmarshal([]byte(item.MetadataJSON), &metadata)
	}
	if metadata.FileName == "" {
		metadata.FileName = filepath.Base(sourcePath)
	}
	if metadata.CurrentPath == "" {
		metadata.CurrentPath = sourcePath
	}
	if metadata.TargetPath == "" {
		metadata.TargetPath = targetPath
	}
	if metadata.TargetFileName == "" {
		metadata.TargetFileName = fileNameOrEmpty(metadata.TargetPath)
	}
	if metadata.DirDate == "" {
		metadata.DirDate = firstNonEmpty(metadata.DirDate, toolsDirDate(sourcePath))
	}
	if metadata.FileNameDate == "" {
		metadata.FileNameDate = firstNonEmpty(metadata.FileNameDate, toolsFileDate(sourcePath))
	}
	if metadata.ModifyDate == "" {
		metadata.ModifyDate = tools.GetModifyDate(sourcePath)
	}
	if metadata.KeepFileName == "" && metadata.KeepPath != "" {
		metadata.KeepFileName = filepath.Base(metadata.KeepPath)
	}
	return metadata
}

func legacyPairPhotosAsList(item model.ScanActionItemDB, metadata actionItemMetadata, sourcePath string, keepPath string) []model.ScanActionPreview {
	deleteEligible := true
	if metadata.DeleteEligible != nil {
		deleteEligible = *metadata.DeleteEligible
	}
	return []model.ScanActionPreview{
		duplicatePhotoPreview(
			firstNonEmpty(metadata.FileName, filepath.Base(sourcePath)),
			firstNonEmpty(metadata.CurrentPath, sourcePath),
			true,
			"扫描记录",
			deleteEligible,
			0,
			1,
		),
		duplicatePhotoPreview(
			firstNonEmpty(metadata.KeepFileName, filepath.Base(keepPath)),
			keepPath,
			false,
			"扫描记录",
			deleteEligible,
			0,
			1,
		),
	}
}

func duplicatePhotosForActionState(item model.ScanActionItemDB, metadata actionItemMetadata, photos []model.ScanActionPreview) []model.ScanActionPreview {
	if item.Stage != model.ActionStageExecuted {
		return photos
	}

	deletePath := strings.TrimSpace(metadata.ExecutedDeletePath)
	if deletePath == "" {
		deletePath = strings.TrimSpace(firstNonEmpty(metadata.RecommendedDeletePath, metadata.CurrentPath, item.SourcePath))
	}
	if deletePath == "" {
		return photos
	}

	if len(photos) > 0 {
		marked := make([]model.ScanActionPreview, 0, len(photos))
		found := false
		for _, photo := range photos {
			isExecuted := sameCleanPath(photo.Path, deletePath)
			photo.ExecutedAction = isExecuted
			if isExecuted {
				photo.DeleteEligible = true
				found = true
			}
			marked = append(marked, photo)
		}
		if found {
			return marked
		}
	}

	sizeBytes := fileSizeOrZero(deletePath)
	return []model.ScanActionPreview{
		{
			FileName:          filepath.Base(deletePath),
			Path:              strings.TrimSpace(deletePath),
			SizeBytes:         sizeBytes,
			SizeText:          formatFileSize(sizeBytes),
			MD5Matched:        true,
			PathSource:        "执行记录",
			RecommendedDelete: true,
			ExecutedAction:    true,
			DeleteEligible:    true,
			CandidateIndex:    0,
			MatchCount:        1,
		},
	}
}

func duplicatePhotoPreview(fileName string, path string, recommendedDelete bool, pathSource string, deleteEligible bool, candidateIndex int, matchCount int) model.ScanActionPreview {
	sizeBytes := fileSizeOrZero(path)
	return model.ScanActionPreview{
		FileName:          firstNonEmpty(fileName, filepath.Base(path)),
		Path:              strings.TrimSpace(path),
		SizeBytes:         sizeBytes,
		SizeText:          formatFileSize(sizeBytes),
		MD5Matched:        true,
		PathSource:        pathSource,
		RecommendedDelete: recommendedDelete,
		DeleteEligible:    deleteEligible && strings.TrimSpace(path) != "",
		CandidateIndex:    candidateIndex,
		MatchCount:        matchCount,
	}
}

func uniqueDuplicatePhotoPreviews(photos []model.ScanActionPreview) []model.ScanActionPreview {
	unique := make([]model.ScanActionPreview, 0, len(photos))
	seen := map[string]struct{}{}
	for _, photo := range photos {
		key := strings.TrimSpace(photo.Path)
		if key != "" {
			key = filepath.Clean(key)
		} else {
			key = "missing:" + strings.TrimSpace(photo.FileName)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, photo)
	}
	return unique
}

func enrichDuplicatePhotos(photos []model.ScanActionPreview, group string, defaultDeleteEligible bool) []model.ScanActionPreview {
	enriched := make([]model.ScanActionPreview, 0, len(photos))
	for index, photo := range photos {
		if photo.FileName == "" {
			photo.FileName = filepath.Base(photo.Path)
		}
		if photo.SizeBytes == 0 {
			photo.SizeBytes = fileSizeOrZero(photo.Path)
		}
		if photo.SizeText == "" {
			photo.SizeText = formatFileSize(photo.SizeBytes)
		}
		if photo.PathSource == "" {
			photo.PathSource = "扫描记录"
		}
		photo.MD5Matched = true
		if !photo.DeleteEligible && defaultDeleteEligible && strings.TrimSpace(photo.Path) != "" {
			photo.DeleteEligible = true
		}
		photo.PreviewSlot = "duplicate_" + strconv.Itoa(index)
		if photo.MatchKey == "" {
			photo.MatchKey = group
		}
		enriched = append(enriched, photo)
	}
	return enriched
}

func markPathDuplicatePhotos(photos []model.ScanActionPreview, matchKey string) []model.ScanActionPreview {
	ret := make([]model.ScanActionPreview, 0, len(photos))
	for _, photo := range photos {
		photo.MD5Matched = false
		photo.MatchType = "img_key"
		if photo.MatchKey == "" {
			photo.MatchKey = matchKey
		}
		if photo.PathSource == "" || photo.PathSource == "扫描记录" {
			photo.PathSource = "日期+文件名"
		}
		ret = append(ret, photo)
	}
	return ret
}

func applyPathDuplicateRecommendation(photos []model.ScanActionPreview) []model.ScanActionPreview {
	paths := make([]string, 0, len(photos))
	for _, photo := range photos {
		if strings.TrimSpace(photo.Path) != "" {
			paths = append(paths, photo.Path)
		}
	}
	keepPath := choosePathDuplicateKeepPhoto(paths)
	if keepPath == "" {
		return photos
	}

	ret := make([]model.ScanActionPreview, 0, len(photos))
	for _, photo := range photos {
		photo.RecommendedDelete = !sameCleanPath(photo.Path, keepPath)
		photo.DeleteEligible = strings.TrimSpace(photo.Path) != ""
		ret = append(ret, photo)
	}
	return ret
}

func duplicatePairFromPhotos(photos []model.ScanActionPreview) (model.ScanActionPreview, model.ScanActionPreview) {
	if len(photos) == 0 {
		return model.ScanActionPreview{}, model.ScanActionPreview{}
	}
	if len(photos) == 1 {
		return photos[0], model.ScanActionPreview{}
	}
	return photos[0], photos[1]
}

func firstPhotoPath(photos []model.ScanActionPreview) string {
	for _, photo := range photos {
		if strings.TrimSpace(photo.Path) != "" {
			return photo.Path
		}
	}
	return ""
}

func newLegacyDuplicatePathResolver(job model.ScanJobDB) func(string) []string {
	roots := collectLegacyDuplicateSearchRoots(job)
	index := map[string][]string{}
	for _, root := range roots {
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			for _, name := range legacyDuplicateLookupNames(info.Name()) {
				index[name] = append(index[name], path)
			}
			return nil
		})
	}
	for key := range index {
		sort.Strings(index[key])
		index[key] = uniqueCleanPaths(index[key])
	}
	return func(name string) []string {
		for _, lookupName := range legacyDuplicateLookupNames(name) {
			if matches := index[lookupName]; len(matches) > 0 {
				return matches
			}
		}
		return nil
	}
}

func uniqueCleanPaths(paths []string) []string {
	unique := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, path := range paths {
		cleanPath := filepath.Clean(path)
		if _, ok := seen[cleanPath]; ok {
			continue
		}
		seen[cleanPath] = struct{}{}
		unique = append(unique, path)
	}
	return unique
}

func collectLegacyDuplicateSearchRoots(job model.ScanJobDB) []string {
	roots := []string{}

	var args model.DoScanImgArg
	if job.ScanArgs != "" {
		_ = json.Unmarshal([]byte(job.ScanArgs), &args)
	}
	if args.StartPath != nil && strings.TrimSpace(*args.StartPath) != "" {
		roots = append(roots, *args.StartPath)
	}
	if len(roots) == 0 && strings.TrimSpace(job.SummaryJSON) != "" {
		var summary ImgRecord
		if err := json.Unmarshal([]byte(job.SummaryJSON), &summary); err == nil && strings.TrimSpace(summary.BasePath) != "" {
			roots = append(roots, summary.BasePath)
		}
	}

	return uniqueExistingDirs(roots)
}

func uniqueExistingDirs(roots []string) []string {
	unique := make([]string, 0, len(roots))
	seen := map[string]struct{}{}
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		stat, err := os.Stat(absRoot)
		if err != nil || !stat.IsDir() {
			continue
		}
		if _, ok := seen[absRoot]; ok {
			continue
		}
		seen[absRoot] = struct{}{}
		unique = append(unique, absRoot)
	}
	return unique
}

func legacyDuplicateLookupNames(name string) []string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" {
		return nil
	}
	candidates := []string{name}
	bracketTrimmed := strings.Split(name, "[")[0]
	if bracketTrimmed != "" {
		candidates = append(candidates, bracketTrimmed)
	}
	withoutExt := strings.TrimSuffix(name, filepath.Ext(name))
	if withoutExt != "" {
		candidates = append(candidates, withoutExt)
	}

	unique := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		unique = append(unique, candidate)
	}
	return unique
}

func collectAllowedPreviewRoots(job model.ScanJobDB) []string {
	roots := []string{}

	var args model.DoScanImgArg
	if job.ScanArgs != "" {
		_ = json.Unmarshal([]byte(job.ScanArgs), &args)
	}
	if args.StartPath != nil && strings.TrimSpace(*args.StartPath) != "" {
		roots = append(roots, *args.StartPath)
	}
	if args.StartPathBak != nil && strings.TrimSpace(*args.StartPathBak) != "" {
		roots = append(roots, *args.StartPathBak)
	}
	roots = append(roots, cons.StartPath, cons.StartPathBak)

	unique := make([]string, 0, len(roots))
	seen := map[string]struct{}{}
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		if _, ok := seen[absRoot]; ok {
			continue
		}
		seen[absRoot] = struct{}{}
		unique = append(unique, absRoot)
	}
	return unique
}

func isPathInRoots(target string, roots []string) bool {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	for _, root := range roots {
		rel, err := filepath.Rel(root, absTarget)
		if err != nil {
			continue
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return true
		}
	}
	return false
}

func previewPathFromView(view model.ScanActionItemView, slot string) string {
	if strings.HasPrefix(slot, "duplicate_") {
		indexRaw := strings.TrimPrefix(slot, "duplicate_")
		index, err := strconv.Atoi(indexRaw)
		if err == nil && index >= 0 && index < len(view.DuplicatePhotos) {
			return view.DuplicatePhotos[index].Path
		}
	}
	switch slot {
	case "", "source":
		if view.Detail != nil {
			return view.Detail.CurrentPath
		}
	case "pair_a":
		if view.Pair != nil {
			return view.Pair.PhotoA.Path
		}
	case "pair_b":
		if view.Pair != nil {
			return view.Pair.PhotoB.Path
		}
	}
	return ""
}

func fileSizeOrZero(path string) int64 {
	if strings.TrimSpace(path) == "" {
		return 0
	}
	stat, err := os.Stat(path)
	if err != nil || stat.IsDir() {
		return 0
	}
	return stat.Size()
}

func formatFileSize(size int64) string {
	if size <= 0 {
		return "-"
	}
	const unit = 1024
	if size < unit {
		return strconv.FormatInt(size, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return strconv.FormatFloat(float64(size)/float64(div), 'f', 1, 64) + " " + string("KMGTPE"[exp]) + "B"
}

func toolsDirDate(path string) string {
	if path == "" {
		return ""
	}
	return tools.GetDirDate(path)
}

func toolsFileDate(path string) string {
	if path == "" {
		return ""
	}
	return tools.GetFileDate(path)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func fileNameOrEmpty(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return filepath.Base(path)
}
