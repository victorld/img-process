package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"img_process/cons"
	"img_process/model"
	"img_process/tools"
)

type actionItemMetadata struct {
	FileName          string `json:"fileName"`
	CurrentPath       string `json:"currentPath"`
	TargetPath        string `json:"targetPath"`
	TargetFileName    string `json:"targetFileName"`
	DirDate           string `json:"dirDate"`
	ModifyDate        string `json:"modifyDate"`
	FileNameDate      string `json:"fileNameDate"`
	ShootDate         string `json:"shootDate"`
	ShootDateRaw      string `json:"shootDateRaw"`
	MinDate           string `json:"minDate"`
	TargetDate        string `json:"targetDate"`
	ExecutedShootDate string `json:"executedShootDate"`
	KeepPath          string `json:"keepPath"`
	KeepFileName      string `json:"keepFileName"`
}

func (r *AppRuntime) ListActionItems(search model.ScanActionItemSearch) ([]model.ScanActionItemView, model.ScanActionCounts, model.ScanActionGroupedCounts, int64, error) {
	list, total, err := r.actionItemService.List(search)
	if err != nil {
		return nil, model.ScanActionCounts{}, model.ScanActionGroupedCounts{}, 0, err
	}

	groupedCounts, err := r.actionItemService.CountGroupedByJob(search.JobID)
	if err != nil {
		return nil, model.ScanActionCounts{}, model.ScanActionGroupedCounts{}, 0, err
	}

	views := make([]model.ScanActionItemView, 0, len(list))
	for _, item := range list {
		views = append(views, buildActionItemView(item))
	}

	return views, groupedCounts.Pending, groupedCounts, total, nil
}

func (r *AppRuntime) ResolveActionPreview(jobID uint, itemID uint, slot string) (string, error) {
	item, err := r.actionItemService.GetByID(itemID)
	if err != nil {
		return "", err
	}
	if item.JobID != jobID {
		return "", errors.New("action item does not belong to job")
	}

	view := buildActionItemView(item)
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
	case model.ActionTypeDeleteDup:
		keepPath := metadata.KeepPath
		if keepPath == "" {
			keepPath = item.TargetPath
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
