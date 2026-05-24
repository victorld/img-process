package api

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"img_process/cons"
	"img_process/dao"
	"img_process/model"
	"img_process/service"
	"img_process/tools"
)

type WebAPI struct{}

var previewThumbMu sync.Mutex
var backupDiffDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

var listJobsFunc = service.Runtime.ListJobs
var getJobFunc = service.Runtime.GetJob
var countGroupedActionItemsByJobsFunc = service.Runtime.CountActionItemsGroupedByJobs
var listActionItemsFunc = func(search model.ScanActionItemSearch) ([]model.ScanActionItemView, model.ScanActionCounts, model.ScanActionGroupedCounts, int64, error) {
	return service.Runtime.ListActionItems(search)
}
var deleteJobFunc = service.Runtime.DeleteJob
var executeModifyShootTimeActionItemFunc = service.Runtime.ExecuteModifyShootTimeActionItem
var executeMoveActionItemFunc = service.Runtime.ExecuteMoveActionItem
var executeRenameActionItemFunc = service.Runtime.ExecuteRenameActionItem
var listFileAnalysisFunc = new(dao.ImgDatabaseService).GetFileAnalysis
var selectSystemDirectoryFunc = selectSystemDirectory
var runSystemDirectoryPickerFunc = runSystemDirectoryPicker
var runSystemDirectoryPickerScriptFunc = runSystemDirectoryPickerScript
var transparentPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

func (api *WebAPI) Login(c *gin.Context) {
	var req model.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	if req.Username != cons.HttpUsername || req.Password != cons.HttpPassword {
		tools.FailWithStatus(c, http.StatusUnauthorized, "用户名或密码错误", gin.H{"authenticated": false})
		return
	}
	token := service.Runtime.Login(req.Username)
	c.SetCookie("img_process_session", token, 86400, "/", "", false, true)
	tools.Success(c, gin.H{"username": req.Username, "authenticated": true}, "登录成功")
}

func (api *WebAPI) Logout(c *gin.Context) {
	token, _ := c.Cookie("img_process_session")
	if token != "" {
		service.Runtime.Logout(token)
	}
	c.SetCookie("img_process_session", "", -1, "/", "", false, true)
	tools.Success(c, gin.H{"authenticated": false}, "退出成功")
}

func (api *WebAPI) Me(c *gin.Context) {
	username, _ := c.Get("username")
	tools.Success(c, gin.H{"username": username, "authenticated": true}, "ok")
}

func (api *WebAPI) ListJobs(c *gin.Context) {
	var search model.ScanJobSearch
	bindPageQuery(c, &search.PageInfo)
	search.Status = c.Query("status")
	search.Source = c.Query("source")
	search.Keyword = c.Query("keyword")
	if raw := c.Query("hasAction"); raw != "" {
		val := raw == "true"
		search.HasAction = &val
	}
	if start, end := parseTimeRange(c.Query("startCreated"), c.Query("endCreated")); start != nil && end != nil {
		search.StartCreated = start
		search.EndCreated = end
	}

	list, total, err := listJobsFunc(search)
	if err != nil {
		tools.Fail(c, "查询任务失败", gin.H{"error": err.Error()})
		return
	}
	jobIDs := make([]uint, 0, len(list))
	for _, item := range list {
		jobIDs = append(jobIDs, item.ID)
	}
	groupedCountsByJob, err := countGroupedActionItemsByJobsFunc(jobIDs)
	if err != nil {
		tools.Fail(c, "查询任务动作统计失败", gin.H{"error": err.Error()})
		return
	}
	ret := make([]gin.H, 0, len(list))
	for _, item := range list {
		ret = append(ret, serializeJob(item, groupedCountsByJob[item.ID]))
	}
	tools.Success(c, gin.H{"list": ret, "total": total}, "ok")
}

func (api *WebAPI) CreateJob(c *gin.Context) {
	var req model.CreateJobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	job, err := service.Runtime.CreateJob(req.Source, req.ScheduleID, req.ScanArgs)
	if err != nil {
		if isBadRequestError(err) {
			tools.FailWithStatus(c, http.StatusBadRequest, "创建任务失败", gin.H{"error": err.Error()})
			return
		}
		tools.Fail(c, "创建任务失败", gin.H{"error": err.Error()})
		return
	}
	tools.SuccessWithStatus(c, http.StatusCreated, gin.H{"job": serializeJob(job, model.ScanActionGroupedCounts{})}, "任务已创建")
}

func (api *WebAPI) GetJob(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	job, err := getJobFunc(jobID)
	if err != nil {
		tools.FailWithStatus(c, http.StatusNotFound, "任务不存在", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"job": serializeJob(job, model.ScanActionGroupedCounts{})}, "ok")
}

func (api *WebAPI) DeleteJob(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	counts, err := deleteJobFunc(jobID)
	if err != nil {
		if strings.Contains(err.Error(), "pending or running jobs cannot be deleted") {
			tools.FailWithStatus(c, http.StatusBadRequest, "删除任务失败", gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tools.FailWithStatus(c, http.StatusNotFound, "任务不存在", gin.H{"id": jobID})
			return
		}
		tools.Fail(c, "删除任务失败", gin.H{"error": err.Error()})
		return
	}
	if counts.Jobs == 0 {
		tools.FailWithStatus(c, http.StatusNotFound, "任务不存在", gin.H{"id": jobID})
		return
	}
	tools.Success(c, gin.H{
		"id":          jobID,
		"actionItems": counts.ActionItems,
		"events":      counts.Events,
		"logs":        counts.Logs,
		"schedules":   counts.Schedules,
	}, "任务已删除")
}

func (api *WebAPI) ListJobEvents(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	var search model.ScanEventSearch
	search.JobID = jobID
	bindPageQuery(c, &search.PageInfo)
	list, total, err := service.Runtime.ListEvents(search)
	if err != nil {
		tools.Fail(c, "查询事件失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"list": list, "total": total}, "ok")
}

func (api *WebAPI) ListJobLogs(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	var search model.ScanJobLogSearch
	search.JobID = jobID
	bindPageQuery(c, &search.PageInfo)
	if search.PageSize == 20 {
		search.PageSize = 200
	}
	list, total, err := service.Runtime.ListLogs(search)
	if err != nil {
		tools.Fail(c, "查询日志失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"list": list, "total": total}, "ok")
}

func (api *WebAPI) ListJobActionItems(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	var search model.ScanActionItemSearch
	search.JobID = jobID
	search.Tab = c.DefaultQuery("tab", "pending")
	search.ActionType = c.Query("type")
	search.Status = c.Query("status")
	search.Keyword = c.Query("keyword")
	bindPageQuery(c, &search.PageInfo)
	list, counts, groupedCounts, total, err := listActionItemsFunc(search)
	if err != nil {
		tools.Fail(c, "查询动作明细失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"list": list, "total": total, "counts": counts, "groupedCounts": groupedCounts}, "ok")
}

func (api *WebAPI) GetJobBackupDiff(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	job, err := getJobFunc(jobID)
	if err != nil {
		tools.FailWithStatus(c, http.StatusNotFound, "任务不存在", gin.H{"error": err.Error()})
		return
	}
	summary, _ := parseJSON(job.SummaryJSON).(map[string]any)
	newFiles := buildBackupDiffGroup(summary, job.ScanUUID, "BakNewFile", "bakNewFile", "bak_new_file_list", "主目录新增，备份缺少", "这些文件已在主目录出现，但备份目录还没有匹配记录，后续需要补齐备份。", "备份目录未找到同名目录标识和文件名")
	deletedFiles := buildBackupDiffGroup(summary, job.ScanUUID, "BakDeleteFile", "bakDeleteFile", "bak_delete_file_list", "备份多余，主目录缺少", "这些文件仍在备份目录，但主目录已经没有匹配记录，建议人工核对后再处理。", "主目录未匹配到同名目录标识和文件名")
	tools.Success(c, gin.H{"newFiles": newFiles, "deletedFiles": deletedFiles}, "ok")
}

func (api *WebAPI) GetFileAnalysis(c *gin.Context) {
	var search model.FileAnalysisSearch
	bindPageQuery(c, &search.PageInfo)
	search.FileKey = c.Query("fileKey")
	search.ShootDateStatus = c.Query("shootDateStatus")
	search.GeoStatus = c.Query("geoStatus")
	search.LocAddrKeyword = c.Query("locAddrKeyword")
	search.ShootDateStart = c.Query("shootDateStart")
	search.ShootDateEnd = c.Query("shootDateEnd")

	result, err := listFileAnalysisFunc(search)
	if err != nil {
		tools.Fail(c, "查询文件分析失败", gin.H{"error": err.Error()})
		return
	}
	for i := range result.List {
		result.List[i].PreviewURL = fileAnalysisPreviewURL(result.List[i].ImgKey, "thumb")
	}
	tools.Success(c, gin.H{
		"summary":     result.Summary,
		"yearStats":   result.YearStats,
		"suffixStats": result.SuffixStats,
		"list":        result.List,
		"total":       result.Total,
	}, "ok")
}

func (api *WebAPI) PreviewFileAnalysis(c *gin.Context) {
	imgKey := strings.TrimSpace(c.Query("imgKey"))
	if imgKey == "" {
		serveFileAnalysisPlaceholder(c)
		return
	}
	path, err := resolveFileAnalysisPreviewPath(imgKey)
	if err != nil {
		serveFileAnalysisPlaceholder(c)
		return
	}
	c.Header("Cache-Control", "private, max-age=60")
	previewPath := path
	if !isBrowserImagePreview(path) {
		thumbPath, err := ensurePreviewImage(path, c.Query("quality"))
		if err != nil {
			serveFileAnalysisPlaceholder(c)
			return
		}
		previewPath = thumbPath
	}
	c.Header("Content-Disposition", "inline; filename=\""+filepath.Base(previewPath)+"\"")
	c.File(previewPath)
}

func (api *WebAPI) PreviewJobAction(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	itemID, err := parseUintQuery(c, "itemId")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "动作ID错误", gin.H{"error": err.Error()})
		return
	}
	slot := c.DefaultQuery("slot", "source")
	path, err := service.Runtime.ResolveActionPreview(jobID, itemID, slot)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, fs.ErrNotExist) {
			status = http.StatusNotFound
		}
		tools.FailWithStatus(c, status, "图片预览失败", gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "private, max-age=60")
	previewPath := path
	if !isBrowserImagePreview(path) {
		thumbPath, err := ensurePreviewImage(path, c.Query("quality"))
		if err != nil {
			tools.FailWithStatus(c, http.StatusBadRequest, "图片预览失败", gin.H{"error": err.Error()})
			return
		}
		previewPath = thumbPath
	}
	c.Header("Content-Disposition", "inline; filename=\""+filepath.Base(previewPath)+"\"")
	c.File(previewPath)
}

func isBrowserImagePreview(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return true
	default:
		return false
	}
}

type previewImageQuality struct {
	suffix   string
	maxWidth int
}

func previewQualityFromQuery(raw string) previewImageQuality {
	if raw == "full" {
		return previewImageQuality{suffix: "full", maxWidth: 2048}
	}
	return previewImageQuality{suffix: "thumb", maxWidth: 320}
}

func ensurePreviewImage(sourcePath string, qualityRaw string) (string, error) {
	previewThumbMu.Lock()
	defer previewThumbMu.Unlock()

	stat, err := os.Stat(sourcePath)
	if err != nil {
		return "", err
	}
	quality := previewQualityFromQuery(qualityRaw)
	hash := sha1.Sum([]byte(sourcePath + "|" + stat.ModTime().Format(time.RFC3339Nano) + "|" + strconv.FormatInt(stat.Size(), 10) + "|" + quality.suffix))
	dir := filepath.Join(cons.WorkDir, "log", "preview_cache")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	thumbPath := filepath.Join(dir, hex.EncodeToString(hash[:])+"_"+quality.suffix+".jpg")
	if _, err := os.Stat(thumbPath); err == nil {
		return thumbPath, nil
	}

	if isHEICPreview(sourcePath) {
		if output, err := runPreviewSips("-s", "format", "jpeg", "-Z", strconv.Itoa(quality.maxWidth), sourcePath, "--out", thumbPath); err == nil {
			return thumbPath, nil
		} else {
			tools.Logger.Warn("sips 生成预览失败，改用 ffmpeg：", strings.TrimSpace(string(output)))
		}
	}

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-i", sourcePath,
		"-frames:v", "1",
		"-vf", "scale='min(" + strconv.Itoa(quality.maxWidth) + ",iw)':-1",
		thumbPath,
	}
	output, err := runPreviewFFmpeg(args...)
	if err != nil {
		return "", errors.New("生成预览缩略图失败：" + strings.TrimSpace(string(output)))
	}
	return thumbPath, nil
}

func isHEICPreview(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".heic", ".heif":
		return true
	default:
		return false
	}
}

var runPreviewSips = func(args ...string) ([]byte, error) {
	return execCommandCombinedOutput("sips", args...)
}

var runPreviewFFmpeg = func(args ...string) ([]byte, error) {
	return execCommandCombinedOutput("ffmpeg", args...)
}

func execCommandCombinedOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func runSystemDirectoryPickerScript(script string) ([]byte, []byte, error) {
	cmd := exec.Command("osascript", "-e", script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	return stdout, stderr.Bytes(), err
}

func fileAnalysisPreviewURL(imgKey string, quality string) string {
	params := url.Values{}
	params.Set("imgKey", imgKey)
	if quality != "" {
		params.Set("quality", quality)
	}
	return "/api/files/analysis/preview?" + params.Encode()
}

func resolveFileAnalysisPreviewPath(imgKey string) (string, error) {
	dirDate, fileName := splitFileAnalysisKey(imgKey)
	if dirDate == "" || fileName == "" || strings.Contains(fileName, string(filepath.Separator)) {
		return "", fs.ErrNotExist
	}
	if len(dirDate) < len("2006-01-02") {
		return "", fs.ErrNotExist
	}
	year := dirDate[:4]
	month := dirDate[:7]
	candidates := []string{
		filepath.Join(cons.StartPath, year, month, dirDate, fileName),
		filepath.Join(cons.StartPathBak, year, month, dirDate, fileName),
	}
	roots := []string{cons.StartPath, cons.StartPathBak}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if !isPathInRoots(candidate, roots) {
			continue
		}
		stat, err := os.Stat(candidate)
		if err == nil && !stat.IsDir() {
			return candidate, nil
		}
	}
	if fallback := findFileAnalysisPreviewFallback(roots, dirDate, fileName); fallback != "" {
		return fallback, nil
	}
	return "", fs.ErrNotExist
}

func findFileAnalysisPreviewFallback(roots []string, dirDate string, fileName string) string {
	type match struct {
		path  string
		score int
	}
	matches := make([]match, 0, 4)
	seen := map[string]struct{}{}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		for _, candidate := range fileAnalysisPreviewFallbackCandidates(root, dirDate, fileName) {
			if !isPathInRoots(candidate, roots) {
				continue
			}
			stat, err := os.Stat(candidate)
			if err != nil || stat.IsDir() {
				continue
			}
			absPath, err := filepath.Abs(candidate)
			if err != nil {
				continue
			}
			if _, ok := seen[absPath]; ok {
				continue
			}
			seen[absPath] = struct{}{}
			matches = append(matches, match{path: candidate, score: fileAnalysisPreviewPathScore(candidate, dirDate)})
		}
	}
	if len(matches) == 0 {
		return ""
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score == matches[j].score {
			return matches[i].path < matches[j].path
		}
		return matches[i].score > matches[j].score
	})
	return matches[0].path
}

func fileAnalysisPreviewFallbackCandidates(root string, dirDate string, fileName string) []string {
	dirs := []string{root}
	if len(dirDate) >= 4 {
		yearDir := filepath.Join(root, dirDate[:4])
		dirs = append(dirs, yearDir)
		if len(dirDate) >= 7 {
			monthDir := filepath.Join(yearDir, dirDate[:7])
			dirs = append(dirs, monthDir)
			if len(dirDate) >= 10 {
				dirs = append(dirs, filepath.Join(monthDir, dirDate[:10]))
			}
		}
	}
	for _, dir := range append([]string{}, dirs...) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				dirs = append(dirs, filepath.Join(dir, entry.Name()))
			}
		}
	}
	candidates := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		candidates = append(candidates, filepath.Join(dir, fileName))
	}
	return candidates
}

func fileAnalysisPreviewPathScore(path string, dirDate string) int {
	normalized := filepath.ToSlash(path)
	score := 0
	if strings.Contains(normalized, dirDate) {
		score += 3
	}
	if len(dirDate) >= 7 && strings.Contains(normalized, dirDate[:7]) {
		score += 2
	}
	if len(dirDate) >= 4 && strings.Contains(normalized, dirDate[:4]) {
		score++
	}
	return score
}

func splitFileAnalysisKey(imgKey string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(imgKey), "|", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

type backupDiffSummaryView struct {
	Count        int      `json:"count"`
	Sample       []string `json:"sample"`
	SampleLimit  int      `json:"sampleLimit"`
	Truncated    bool     `json:"truncated"`
	ArtifactPath string   `json:"artifactPath"`
}

type backupDiffItemView struct {
	Key            string `json:"key"`
	RawLine        string `json:"rawLine"`
	DirectoryLabel string `json:"directoryLabel"`
	FileName       string `json:"fileName"`
	Date           string `json:"date"`
	Reason         string `json:"reason"`
}

type backupDiffGroupView struct {
	Label        string               `json:"label"`
	Field        string               `json:"field"`
	Count        int                  `json:"count"`
	ArtifactPath string               `json:"artifactPath"`
	Complete     bool                 `json:"complete"`
	Note         string               `json:"note"`
	Items        []backupDiffItemView `json:"items"`
}

func buildBackupDiffGroup(summary map[string]any, scanUUID string, primaryKey string, aliasKey string, fileName string, label string, note string, reason string) backupDiffGroupView {
	group := backupDiffGroupView{
		Label:    label,
		Field:    aliasKey,
		Note:     note,
		Complete: true,
		Items:    []backupDiffItemView{},
	}
	summaryValue := valueFromMapKeys(summary, primaryKey, aliasKey)
	diffSummary := parseBackupDiffSummary(summaryValue)
	group.Count = diffSummary.Count
	group.ArtifactPath = diffSummary.ArtifactPath

	lines, err := readBackupDiffArtifactLines(scanUUID, diffSummary.ArtifactPath, fileName)
	if err != nil {
		group.Complete = false
		lines = diffSummary.Sample
	} else if len(lines) > 0 {
		group.Complete = true
	}
	if len(lines) == 0 && len(diffSummary.Sample) > 0 {
		group.Complete = false
		lines = diffSummary.Sample
	}
	group.Items = buildBackupDiffItems(lines, reason)
	if group.Count == 0 {
		group.Count = len(group.Items)
	}
	return group
}

func parseBackupDiffSummary(value any) backupDiffSummaryView {
	switch current := value.(type) {
	case nil:
		return backupDiffSummaryView{Sample: []string{}}
	case backupDiffSummaryView:
		return current
	case map[string]any:
		return backupDiffSummaryView{
			Count:        int(numberFromAny(current["count"])),
			Sample:       stringsFromAny(current["sample"]),
			SampleLimit:  int(numberFromAny(current["sampleLimit"])),
			Truncated:    boolFromAny(current["truncated"]),
			ArtifactPath: stringFromAny(current["artifactPath"]),
		}
	case string:
		trimmed := strings.TrimSpace(current)
		if trimmed == "" || trimmed == "null" {
			return backupDiffSummaryView{Sample: []string{}}
		}
		var decoded backupDiffSummaryView
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			if decoded.Sample == nil {
				decoded.Sample = []string{}
			}
			return decoded
		}
		var legacy []string
		if err := json.Unmarshal([]byte(trimmed), &legacy); err == nil {
			return backupDiffSummaryView{Count: len(legacy), Sample: legacy}
		}
		return backupDiffSummaryView{Count: 1, Sample: []string{trimmed}}
	case []any:
		sample := stringsFromAny(current)
		return backupDiffSummaryView{Count: len(sample), Sample: sample}
	case []string:
		return backupDiffSummaryView{Count: len(current), Sample: current}
	default:
		return backupDiffSummaryView{Sample: []string{}}
	}
}

func readBackupDiffArtifactLines(scanUUID string, artifactPath string, allowedFileName string) ([]string, error) {
	artifactPath = strings.TrimSpace(artifactPath)
	if artifactPath == "" || scanUUID == "" {
		return nil, fs.ErrNotExist
	}
	if filepath.Base(artifactPath) != allowedFileName {
		return nil, errors.New("invalid backup diff artifact name")
	}
	root := filepath.Join(cons.WorkDir, "log", "dump_delete_file", scanUUID)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	absPath, err := filepath.Abs(artifactPath)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return nil, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil, errors.New("backup diff artifact is outside scan artifact root")
	}
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func buildBackupDiffItems(lines []string, reason string) []backupDiffItemView {
	items := make([]backupDiffItemView, 0, len(lines))
	for _, line := range lines {
		directoryLabel, fileName := splitBackupDiffLine(line)
		items = append(items, backupDiffItemView{
			Key:            line,
			RawLine:        line,
			DirectoryLabel: directoryLabel,
			FileName:       fileName,
			Date:           backupDiffDateFromDirectory(directoryLabel),
			Reason:         reason,
		})
	}
	return items
}

func splitBackupDiffLine(line string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(line), "|", 2)
	if len(parts) != 2 {
		return strings.TrimSpace(line), ""
	}
	return parts[0], parts[1]
}

func backupDiffDateFromDirectory(directoryLabel string) string {
	match := backupDiffDatePattern.FindString(strings.TrimSpace(directoryLabel))
	return match
}

func valueFromMapKeys(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if values != nil {
			if value, ok := values[key]; ok {
				return value
			}
		}
	}
	return nil
}

func stringsFromAny(value any) []string {
	switch current := value.(type) {
	case []string:
		return append([]string(nil), current...)
	case []any:
		ret := make([]string, 0, len(current))
		for _, item := range current {
			text := strings.TrimSpace(stringFromAny(item))
			if text != "" {
				ret = append(ret, text)
			}
		}
		return ret
	default:
		return []string{}
	}
}

func stringFromAny(value any) string {
	switch current := value.(type) {
	case string:
		return current
	case nil:
		return ""
	default:
		return strings.TrimSpace(tools.MarshalJsonToString(current))
	}
}

func boolFromAny(value any) bool {
	switch current := value.(type) {
	case bool:
		return current
	case string:
		return current == "true"
	default:
		return false
	}
}

func isPathInRoots(path string, roots []string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(absRoot, absPath)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func serveFileAnalysisPlaceholder(c *gin.Context) {
	c.Header("Content-Type", "image/svg+xml; charset=utf-8")
	c.Header("Cache-Control", "private, max-age=60")
	c.String(http.StatusOK, `<svg xmlns="http://www.w3.org/2000/svg" width="96" height="96"><rect width="96" height="96" fill="#eef2ef"/><text x="48" y="53" text-anchor="middle" fill="#6f7f77" font-size="13">无预览</text></svg>`)
}

func (api *WebAPI) StreamJob(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	var lastEventID uint
	var lastStatus string
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			job, err := service.Runtime.GetJob(jobID)
			if err != nil {
				c.SSEvent("error", gin.H{"message": err.Error()})
				c.Writer.Flush()
				return
			}
			if job.Status != lastStatus {
				lastStatus = job.Status
				c.SSEvent("job", serializeJob(job, model.ScanActionGroupedCounts{}))
			}
			events, err := service.Runtime.ListEventsAfter(jobID, lastEventID)
			if err == nil {
				for _, event := range events {
					lastEventID = event.ID
					c.SSEvent("event", event)
				}
			}
			c.Writer.Flush()
			if isFinishedStatus(job.Status) {
				return
			}
		}
	}
}

func (api *WebAPI) DeleteJobDuplicates(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	if err := service.Runtime.DeleteDuplicateFiles(jobID); err != nil {
		tools.Fail(c, "执行重复文件删除失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"jobId": jobID}, "重复文件删除已执行")
}

func (api *WebAPI) DeleteJobPathDuplicates(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	count, err := service.Runtime.DeletePathDuplicateFiles(jobID)
	if err != nil {
		tools.Fail(c, "执行文件路径重复项删除失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"jobId": jobID, "count": count}, "文件路径重复项删除已执行")
}

func (api *WebAPI) DeleteJobActionItem(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	itemID, err := parseUintParam(c, "itemId")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "动作ID错误", gin.H{"error": err.Error()})
		return
	}
	if err := service.Runtime.ExecuteDeleteActionItem(jobID, itemID); err != nil {
		if isBadRequestError(err) || strings.Contains(err.Error(), "does not belong") || strings.Contains(err.Error(), "not a pending delete action") {
			tools.FailWithStatus(c, http.StatusBadRequest, "执行删除失败", gin.H{"error": err.Error()})
			return
		}
		tools.Fail(c, "执行删除失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"jobId": jobID, "itemId": itemID}, "删除动作已执行")
}

func (api *WebAPI) DeleteDuplicateActionItem(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	itemID, err := parseUintParam(c, "itemId")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "动作ID错误", gin.H{"error": err.Error()})
		return
	}

	var req model.ExecuteDuplicateDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}

	if err := service.Runtime.ExecuteDuplicateActionItem(jobID, itemID, req.Side, req.Path); err != nil {
		if isBadRequestError(err) ||
			strings.Contains(err.Error(), "does not belong") ||
			strings.Contains(err.Error(), "not a pending duplicate delete action") ||
			strings.Contains(err.Error(), "invalid duplicate side") ||
			strings.Contains(err.Error(), "duplicate delete path is empty") ||
			strings.Contains(err.Error(), "duplicate delete path is not in this duplicate group") ||
			strings.Contains(err.Error(), "duplicate delete path is outside allowed roots") {
			tools.FailWithStatus(c, http.StatusBadRequest, "执行重复项删除失败", gin.H{"error": err.Error()})
			return
		}
		tools.Fail(c, "执行重复项删除失败", gin.H{"error": err.Error()})
		return
	}

	tools.Success(c, gin.H{"jobId": jobID, "itemId": itemID, "side": strings.ToUpper(strings.TrimSpace(req.Side)), "path": req.Path}, "重复项删除已执行")
}

func (api *WebAPI) ModifyJobShootTimeActionItem(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	itemID, err := parseUintParam(c, "itemId")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "动作ID错误", gin.H{"error": err.Error()})
		return
	}

	if err := executeModifyShootTimeActionItemFunc(jobID, itemID); err != nil {
		if isBadRequestError(err) ||
			strings.Contains(err.Error(), "does not belong") ||
			strings.Contains(err.Error(), "not a pending modify_time action") ||
			strings.Contains(err.Error(), "target date is empty") ||
			strings.Contains(err.Error(), "target date format is invalid") {
			tools.FailWithStatus(c, http.StatusBadRequest, "执行拍摄时间变更失败", gin.H{"error": err.Error()})
			return
		}
		tools.Fail(c, "执行拍摄时间变更失败", gin.H{"error": err.Error()})
		return
	}

	tools.Success(c, gin.H{"jobId": jobID, "itemId": itemID}, "拍摄时间已变更")
}

func (api *WebAPI) MoveJobActionItem(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	itemID, err := parseUintParam(c, "itemId")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "动作ID错误", gin.H{"error": err.Error()})
		return
	}

	if err := executeMoveActionItemFunc(jobID, itemID); err != nil {
		if isBadRequestError(err) ||
			strings.Contains(err.Error(), "does not belong") ||
			strings.Contains(err.Error(), "not a pending move action") ||
			strings.Contains(err.Error(), "move target path is empty") {
			tools.FailWithStatus(c, http.StatusBadRequest, "执行位置变更失败", gin.H{"error": err.Error()})
			return
		}
		tools.Fail(c, "执行位置变更失败", gin.H{"error": err.Error()})
		return
	}

	tools.Success(c, gin.H{"jobId": jobID, "itemId": itemID}, "位置已变更")
}

func (api *WebAPI) RenameJobActionItem(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	itemID, err := parseUintParam(c, "itemId")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "动作ID错误", gin.H{"error": err.Error()})
		return
	}

	if err := executeRenameActionItemFunc(jobID, itemID); err != nil {
		if isBadRequestError(err) ||
			strings.Contains(err.Error(), "does not belong") ||
			strings.Contains(err.Error(), "not a pending rename action") ||
			strings.Contains(err.Error(), "rename target path is empty") {
			tools.FailWithStatus(c, http.StatusBadRequest, "执行重命名失败", gin.H{"error": err.Error()})
			return
		}
		tools.Fail(c, "执行重命名失败", gin.H{"error": err.Error()})
		return
	}

	tools.Success(c, gin.H{"jobId": jobID, "itemId": itemID}, "文件名已变更")
}

func (api *WebAPI) DeleteAllJobActionItems(c *gin.Context) {
	jobID, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "任务ID错误", gin.H{"error": err.Error()})
		return
	}
	count, err := service.Runtime.ExecuteAllDeleteActionItems(jobID)
	if err != nil {
		tools.Fail(c, "批量执行删除失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"jobId": jobID, "count": count}, "删除类动作已全部执行")
}

func (api *WebAPI) ListSchedules(c *gin.Context) {
	var search model.ScanScheduleSearch
	bindPageQuery(c, &search.PageInfo)
	if raw := c.Query("enabled"); raw != "" {
		val := raw == "true"
		search.Enabled = &val
	}
	list, total, err := service.Runtime.ListSchedules(search)
	if err != nil {
		tools.Fail(c, "查询计划失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"list": list, "total": total}, "ok")
}

func (api *WebAPI) CreateSchedule(c *gin.Context) {
	var req model.UpsertScheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	schedule, err := service.Runtime.CreateSchedule(req)
	if err != nil {
		if isBadRequestError(err) {
			tools.FailWithStatus(c, http.StatusBadRequest, "创建计划失败", gin.H{"error": err.Error()})
			return
		}
		tools.Fail(c, "创建计划失败", gin.H{"error": err.Error()})
		return
	}
	tools.SuccessWithStatus(c, http.StatusCreated, gin.H{"schedule": schedule}, "计划已创建")
}

func (api *WebAPI) UpdateSchedule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "计划ID错误", gin.H{"error": err.Error()})
		return
	}
	var req model.UpsertScheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	schedule, err := service.Runtime.UpdateSchedule(id, req)
	if err != nil {
		if isBadRequestError(err) {
			tools.FailWithStatus(c, http.StatusBadRequest, "更新计划失败", gin.H{"error": err.Error()})
			return
		}
		tools.Fail(c, "更新计划失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"schedule": schedule}, "计划已更新")
}

func (api *WebAPI) DeleteSchedule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "计划ID错误", gin.H{"error": err.Error()})
		return
	}
	if err := service.Runtime.DeleteSchedule(id); err != nil {
		tools.Fail(c, "删除计划失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"id": id}, "计划已删除")
}

func (api *WebAPI) EnableSchedule(c *gin.Context) {
	api.toggleSchedule(c, true)
}

func (api *WebAPI) DisableSchedule(c *gin.Context) {
	api.toggleSchedule(c, false)
}

func (api *WebAPI) RunSchedule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "计划ID错误", gin.H{"error": err.Error()})
		return
	}
	job, err := service.Runtime.RunSchedule(id)
	if err != nil {
		tools.Fail(c, "计划执行失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"job": serializeJob(job, model.ScanActionGroupedCounts{})}, "计划已触发")
}

func (api *WebAPI) GetSystemStatus(c *gin.Context) {
	tools.Success(c, gin.H{
		"configSource":     "database",
		"configFile":       tools.ConfigFileUsed(),
		"config":           service.GetSystemSettingSnapshot(true),
		"readonlySections": service.ReadonlySystemSettingSections(),
		"server": gin.H{
			"httpPort":     cons.HttpPort,
			"startPath":    cons.StartPath,
			"startPathBak": cons.StartPathBak,
			"poolSize":     cons.PoolSize,
			"imgCache":     cons.ImgCache,
			"sqlDebug":     cons.SqlDebug,
			"scanDefaults": gin.H{
				"startPath":        cons.StartPath,
				"startPathBak":     cons.StartPathBak,
				"deleteShow":       cons.DeleteShow,
				"moveFileShow":     cons.MoveFileShow,
				"modifyDateShow":   cons.ModifyDateShow,
				"renameFileShow":   cons.RenameFileShow,
				"md5Show":          cons.Md5Show,
				"deleteAction":     cons.DeleteAction,
				"moveFileAction":   cons.MoveFileAction,
				"modifyDateAction": cons.ModifyDateAction,
				"renameFileAction": cons.RenameFileAction,
			},
		},
	}, "ok")
}

func (api *WebAPI) UpdateSystemSettings(c *gin.Context) {
	var req model.UpdateSystemSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	config, err := service.UpdateSystemSettings(service.SystemSettingSnapshot(req.Config))
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "保存设置失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{
		"configSource":     "database",
		"configFile":       tools.ConfigFileUsed(),
		"config":           config,
		"readonlySections": service.ReadonlySystemSettingSections(),
	}, "设置已保存")
}

func (api *WebAPI) SelectSystemDirectory(c *gin.Context) {
	var req model.SelectSystemDirectoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "参数错误", gin.H{"error": err.Error()})
		return
	}
	path, err := selectSystemDirectoryFunc(req.Path, req.Title)
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, err.Error(), gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"path": path}, "ok")
}

func selectSystemDirectory(path string, title string) (string, error) {
	if title == "" {
		title = "选择目录"
	}
	defaultPath := resolveSystemDirectoryPickerDefault(path)
	selected, err := runSystemDirectoryPickerFunc(defaultPath, title)
	if err != nil {
		return "", err
	}
	return normalizeSelectedSystemDirectory(selected)
}

func resolveSystemDirectoryPickerDefault(path string) string {
	candidate := strings.TrimSpace(path)
	if candidate != "" {
		if resolved, ok := existingDirectoryForPicker(candidate); ok {
			return resolved
		}
		for parent := filepath.Dir(candidate); parent != "." && parent != candidate; parent = filepath.Dir(parent) {
			if resolved, ok := existingDirectoryForPicker(parent); ok {
				return resolved
			}
			if filepath.Dir(parent) == parent {
				break
			}
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if resolved, ok := existingDirectoryForPicker(home); ok {
			return resolved
		}
	}
	return string(filepath.Separator)
}

func existingDirectoryForPicker(path string) (string, bool) {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", false
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	return abs, true
}

func runSystemDirectoryPicker(defaultPath string, title string) (string, error) {
	script := `set defaultFolder to POSIX file ` + appleScriptQuote(defaultPath) + `
try
	set selectedFolder to choose folder with prompt ` + appleScriptQuote(title) + ` default location defaultFolder
	return POSIX path of selectedFolder
on error number -128
	return "__CANCELLED__"
end try`
	stdout, stderr, err := runSystemDirectoryPickerScriptFunc(script)
	if err != nil {
		detail := strings.TrimSpace(string(stderr))
		if detail == "" {
			detail = strings.TrimSpace(err.Error())
		}
		return "", errors.New("打开系统目录选择窗口失败：" + detail)
	}
	selected := strings.TrimSpace(string(stdout))
	if selected == "__CANCELLED__" {
		return "", errors.New("已取消选择目录")
	}
	if selected == "" {
		return "", errors.New("未选择目录")
	}
	return selected, nil
}

func normalizeSelectedSystemDirectory(path string) (string, error) {
	selected := strings.TrimSpace(path)
	if selected == "" {
		return "", errors.New("未选择目录")
	}
	abs, err := filepath.Abs(filepath.Clean(selected))
	if err != nil {
		return "", errors.New("请选择有效目录：" + err.Error())
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", errors.New("请选择有效目录：" + err.Error())
	}
	if !info.IsDir() {
		return "", errors.New("请选择有效目录")
	}
	return abs, nil
}

func appleScriptQuote(value string) string {
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

func (api *WebAPI) ListSystemDirectories(c *gin.Context) {
	path := strings.TrimSpace(c.Query("path"))
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			tools.FailWithStatus(c, http.StatusBadRequest, "读取目录失败", gin.H{"error": err.Error()})
			return
		}
		path = home
	}
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			tools.FailWithStatus(c, http.StatusBadRequest, "读取目录失败", gin.H{"error": err.Error()})
			return
		}
		path = abs
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "读取目录失败", gin.H{"error": err.Error()})
		return
	}
	if !info.IsDir() {
		tools.FailWithStatus(c, http.StatusBadRequest, "路径不是目录", gin.H{"error": "path is not a directory"})
		return
	}
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "读取目录失败", gin.H{"error": err.Error()})
		return
	}
	entries := make([]gin.H, 0, len(dirEntries))
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			continue
		}
		entries = append(entries, gin.H{
			"name": entry.Name(),
			"path": filepath.Join(path, entry.Name()),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i]["name"].(string)) < strings.ToLower(entries[j]["name"].(string))
	})
	parent := filepath.Dir(path)
	if parent == path {
		parent = ""
	}
	tools.Success(c, gin.H{
		"path":    path,
		"parent":  parent,
		"entries": entries,
	}, "ok")
}

func maskConfigSecret(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) == 1 {
		return string(runes[0])
	}
	return string(runes[0]) + strings.Repeat("*", len(runes)-1)
}

func (api *WebAPI) toggleSchedule(c *gin.Context, enabled bool) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		tools.FailWithStatus(c, http.StatusBadRequest, "计划ID错误", gin.H{"error": err.Error()})
		return
	}
	schedule, err := service.Runtime.SetScheduleEnabled(id, enabled)
	if err != nil {
		tools.Fail(c, "更新计划状态失败", gin.H{"error": err.Error()})
		return
	}
	tools.Success(c, gin.H{"schedule": schedule}, "ok")
}

func parseUintParam(c *gin.Context, key string) (uint, error) {
	raw := c.Param(key)
	if raw == "" {
		return 0, errors.New("missing id")
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	return uint(id), err
}

func parseUintQuery(c *gin.Context, key string) (uint, error) {
	raw := c.Query(key)
	if raw == "" {
		return 0, errors.New("missing query")
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	return uint(id), err
}

func bindPageQuery(c *gin.Context, pageInfo *model.PageInfo) {
	pageInfo.Page = 1
	pageInfo.PageSize = 20
	if raw := c.Query("page"); raw != "" {
		if page, err := strconv.Atoi(raw); err == nil && page > 0 {
			pageInfo.Page = page
		}
	}
	if raw := c.Query("pageSize"); raw != "" {
		if size, err := strconv.Atoi(raw); err == nil && size > 0 {
			pageInfo.PageSize = size
		}
	}
}

func parseTimeRange(startRaw string, endRaw string) (*time.Time, *time.Time) {
	if startRaw == "" || endRaw == "" {
		return nil, nil
	}
	start, err1 := time.Parse(time.RFC3339, startRaw)
	end, err2 := time.Parse(time.RFC3339, endRaw)
	if err1 != nil || err2 != nil {
		return nil, nil
	}
	return &start, &end
}

func serializeJob(job model.ScanJobDB, groupedCounts model.ScanActionGroupedCounts) gin.H {
	summary := parseJSON(job.SummaryJSON)
	return gin.H{
		"id":                     job.ID,
		"jobUuid":                job.JobUUID,
		"scanUuid":               job.ScanUUID,
		"source":                 job.Source,
		"status":                 job.Status,
		"scheduleId":             job.ScheduleID,
		"queueAt":                job.QueueAt,
		"startAt":                job.StartAt,
		"endAt":                  job.EndAt,
		"currentPhase":           job.CurrentPhase,
		"lastHeartbeatAt":        job.LastHeartbeatAt,
		"processedCount":         job.ProcessedCount,
		"totalCount":             job.TotalCount,
		"totalFolderCount":       summaryInt(summary, "dirTotal", "DirTotal"),
		"totalBackupFileCount":   summaryInt(summary, "fileTotalBak", "FileTotalBak"),
		"totalBackupFolderCount": summaryInt(summary, "dirTotalBak", "DirTotalBak"),
		"hasAction":              job.HasAction,
		"scanArgs":               parseJSON(job.ScanArgs),
		"summary":                summary,
		"pendingActionCount":     groupedCounts.Pending.Total,
		"executedActionCount":    groupedCounts.Executed.Total,
		"artifactPath":           job.ArtifactPath,
		"errorMessage":           job.ErrorMessage,
		"createdAt":              job.CreatedAt,
		"updatedAt":              job.UpdatedAt,
	}
}

func parseJSON(raw string) any {
	if raw == "" {
		return nil
	}
	var ret any
	if err := json.Unmarshal([]byte(raw), &ret); err != nil {
		return raw
	}
	return ret
}

func summaryInt(summary any, keys ...string) int64 {
	obj, ok := summary.(map[string]any)
	if !ok {
		return 0
	}
	for _, key := range keys {
		if value, exists := obj[key]; exists {
			return numberFromAny(value)
		}
	}
	return 0
}

func numberFromAny(value any) int64 {
	switch current := value.(type) {
	case int:
		return int64(current)
	case int8:
		return int64(current)
	case int16:
		return int64(current)
	case int32:
		return int64(current)
	case int64:
		return current
	case uint:
		return int64(current)
	case uint8:
		return int64(current)
	case uint16:
		return int64(current)
	case uint32:
		return int64(current)
	case uint64:
		return int64(current)
	case float32:
		return int64(current)
	case float64:
		return int64(current)
	case string:
		parsed, err := strconv.ParseInt(current, 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func isFinishedStatus(status string) bool {
	return status == model.JobStatusSucceeded || status == model.JobStatusFailed || status == model.JobStatusInterrupted || status == model.JobStatusSkipped
}

func isBadRequestError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// Keep this check resilient to localized error messages.
	return strings.Contains(msg, "startPath") || strings.Contains(strings.ToLower(msg), "cron") || strings.Contains(msg, "Cron")
}
