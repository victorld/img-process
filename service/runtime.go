package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"img_process/cons"
	"img_process/dao"
	"img_process/model"
	"img_process/tools"
)

type scheduleConfig struct {
	Hour       int   `json:"hour"`
	Minute     int   `json:"minute"`
	Weekdays   []int `json:"weekdays"`
	DayOfMonth int   `json:"dayOfMonth"`
}

type scanArgValidationError struct {
	message string
}

func (e *scanArgValidationError) Error() string {
	return e.message
}

type AppRuntime struct {
	jobService        dao.ScanJobService
	actionItemService dao.ScanActionItemService
	eventService      dao.ScanEventService
	jobLogService     dao.ScanJobLogService
	scheduleService   dao.ScanScheduleService

	notifyCh chan struct{}
	stopCh   chan struct{}

	started atomic.Bool

	sessionMu sync.RWMutex
	sessions  map[string]string

	scheduleMu sync.Mutex
}

var Runtime = NewAppRuntime()

func NewAppRuntime() *AppRuntime {
	return &AppRuntime{
		notifyCh: make(chan struct{}, 1),
		stopCh:   make(chan struct{}),
		sessions: map[string]string{},
	}
}

func (r *AppRuntime) Start() error {
	if r.started.Swap(true) {
		return nil
	}

	if err := r.jobService.MarkInterruptedRunning(); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := r.syncSchedules(time.Now()); err != nil {
		return err
	}
	go r.workerLoop()
	go r.scheduleLoop()
	r.signal()
	return nil
}

func (r *AppRuntime) Stop() {
	select {
	case <-r.stopCh:
	default:
		close(r.stopCh)
	}
}

func (r *AppRuntime) signal() {
	select {
	case r.notifyCh <- struct{}{}:
	default:
	}
}

func (r *AppRuntime) workerLoop() {
	for {
		select {
		case <-r.stopCh:
			return
		case <-r.notifyCh:
			for {
				job, err := r.jobService.GetNextPending()
				if err != nil {
					break
				}
				r.runJob(job)
			}
		}
	}
}

func (r *AppRuntime) runJob(job model.ScanJobDB) {
	now := time.Now()
	job.Status = model.JobStatusRunning
	job.StartAt = &now
	job.CurrentPhase = "starting"
	job.LastHeartbeatAt = &now
	job.ErrorMessage = ""
	if err := r.jobService.Update(&job); err != nil {
		tools.Logger.Error("update running job error : ", err)
		return
	}

	recorder := NewDBScanRecorder(job.ID)
	recorder.RecordLifecycle("任务开始执行", "任务开始执行", map[string]any{
		"jobId": job.ID,
	})

	var scanArgs model.DoScanImgArg
	if err := json.Unmarshal([]byte(job.ScanArgs), &scanArgs); err != nil {
		recorder.RecordError("starting", "", fmt.Errorf("parse scan args: %w", err), nil)
		recorder.FinishWithError(err)
		return
	}

	if _, err := ScanAndSaveWithRecorder(scanArgs, recorder); err != nil {
		recorder.FinishWithError(err)
		return
	}
}

func (r *AppRuntime) CreateJob(source string, scheduleID *uint, scanArgs model.DoScanImgArg) (model.ScanJobDB, error) {
	if source == "" {
		source = model.JobSourceManual
	}
	scanArgs = NormalizeScanArgs(scanArgs)
	if err := validateScanArgs(scanArgs); err != nil {
		return model.ScanJobDB{}, err
	}
	scanArgsJSON := tools.MarshalJsonToString(buildScanExecutionSnapshot(scanArgs))
	hasAction := hasActionEnabled(scanArgs)
	now := time.Now()
	job := model.ScanJobDB{
		JobUUID:    uuid.NewString(),
		Source:     source,
		Status:     model.JobStatusPending,
		ScheduleID: scheduleID,
		QueueAt:    &now,
		HasAction:  hasAction,
		ScanArgs:   scanArgsJSON,
	}
	if err := r.jobService.Create(&job); err != nil {
		return job, err
	}
	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:     job.ID,
		EventType: model.EventTypeLifecycle,
		Phase:     "queued",
		Level:     "info",
		Title:     "任务入队",
		Message:   "任务已创建并进入队列",
		PayloadJSON: tools.MarshalJsonToString(ginH(
			"source", source,
			"hasAction", hasAction,
		)),
	})

	if scheduleID != nil {
		if schedule, err := r.scheduleService.GetByID(*scheduleID); err == nil {
			schedule.LastJobID = &job.ID
			schedule.LastJobStatus = job.Status
			schedule.LastRunAt = &now
			_ = r.scheduleService.Update(&schedule)
		}
	}

	r.signal()
	return job, nil
}

func (r *AppRuntime) GetJob(id uint) (model.ScanJobDB, error) {
	return r.jobService.GetByID(id)
}

func (r *AppRuntime) DeleteJob(id uint) (dao.ScanJobDeleteCounts, error) {
	job, err := r.jobService.GetByID(id)
	if err != nil {
		return dao.ScanJobDeleteCounts{}, err
	}
	if job.Status == model.JobStatusPending || job.Status == model.JobStatusRunning {
		return dao.ScanJobDeleteCounts{}, errors.New("pending or running jobs cannot be deleted")
	}
	return r.jobService.DeleteWithChildren(id)
}

func (r *AppRuntime) ListJobs(search model.ScanJobSearch) ([]model.ScanJobDB, int64, error) {
	return r.jobService.List(search)
}

func (r *AppRuntime) CountActionItemsGroupedByJobs(jobIDs []uint) (map[uint]model.ScanActionGroupedCounts, error) {
	return r.actionItemService.CountGroupedByJobs(jobIDs)
}

func (r *AppRuntime) ListEvents(search model.ScanEventSearch) ([]model.ScanEventDB, int64, error) {
	return r.eventService.List(search)
}

func (r *AppRuntime) ListEventsAfter(jobID uint, afterID uint) ([]model.ScanEventDB, error) {
	return r.eventService.ListAfterID(jobID, afterID)
}

func (r *AppRuntime) ListLogs(search model.ScanJobLogSearch) ([]model.ScanJobLogDB, int64, error) {
	return r.jobLogService.List(search)
}

func (r *AppRuntime) DeleteDuplicateFiles(jobID uint) error {
	job, err := r.jobService.GetByID(jobID)
	if err != nil {
		return err
	}
	if job.Status != model.JobStatusSucceeded {
		return errors.New("job is not completed successfully")
	}

	filePath, err := resolveJobDeleteListPath(job.ScanUUID)
	if err != nil {
		return err
	}

	items, err := r.actionItemService.ListPendingDuplicateByJob(job.ID)
	if err != nil {
		return err
	}
	itemByPath := map[string]model.ScanActionItemDB{}
	for _, item := range items {
		itemByPath[item.SourcePath] = item
	}

	shouldDeleteFiles, err := tools.ReadFileLines(filePath)
	if err != nil {
		return err
	}
	for _, photo := range shouldDeleteFiles {
		item := itemByPath[photo]
		if item.ID == 0 {
			continue
		}
		if err = tools.DeleteFile(photo); err != nil {
			if saveErr := r.completeDuplicateDeleteAction(&item, "A", photo, item.SourcePath, err); saveErr != nil {
				return saveErr
			}
			continue
		}
		if saveErr := r.completeDuplicateDeleteAction(&item, "A", photo, item.SourcePath, nil); saveErr != nil {
			return saveErr
		}
	}

	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:     job.ID,
		EventType: model.EventTypeLifecycle,
		Phase:     "post_action",
		Level:     "info",
		Title:     "重复文件删除已执行",
		Message:   "重复文件删除任务已执行",
	})
	return nil
}

func (r *AppRuntime) DeletePathDuplicateFiles(jobID uint) (int, error) {
	job, err := r.jobService.GetByID(jobID)
	if err != nil {
		return 0, err
	}
	if job.Status != model.JobStatusSucceeded {
		return 0, errors.New("job is not completed successfully")
	}

	items, err := r.actionItemService.ListPendingPathDuplicateByJob(job.ID)
	if err != nil {
		return 0, err
	}

	successCount := 0
	for idx := range items {
		item := items[idx]
		deletePath, ok := recommendedDeletePathFromMetadata(item)
		if !ok {
			continue
		}
		if err := r.executeDuplicateDeleteActionItem(&item, "PATH", deletePath); err != nil {
			return successCount, err
		}
		successCount++
	}

	_ = r.eventService.Create(&model.ScanEventDB{
		JobID:     job.ID,
		EventType: model.EventTypeLifecycle,
		Phase:     "post_action",
		Level:     "info",
		Title:     "文件路径重复项删除已执行",
		Message:   "文件路径重复项已按建议删除",
	})
	return successCount, nil
}

func (r *AppRuntime) Login(username string) string {
	token := uuid.NewString()
	r.sessionMu.Lock()
	defer r.sessionMu.Unlock()
	r.sessions[token] = username
	return token
}

func (r *AppRuntime) Logout(token string) {
	r.sessionMu.Lock()
	defer r.sessionMu.Unlock()
	delete(r.sessions, token)
}

func (r *AppRuntime) ValidateSession(token string) (string, bool) {
	r.sessionMu.RLock()
	defer r.sessionMu.RUnlock()
	username, ok := r.sessions[token]
	return username, ok
}

func (r *AppRuntime) ListSchedules(search model.ScanScheduleSearch) ([]model.ScanScheduleDB, int64, error) {
	return r.scheduleService.List(search)
}

func (r *AppRuntime) CreateSchedule(req model.UpsertScheduleReq) (model.ScanScheduleDB, error) {
	schedule, err := buildScheduleModel(req)
	if err != nil {
		return model.ScanScheduleDB{}, err
	}
	if err := r.scheduleService.Create(&schedule); err != nil {
		return schedule, err
	}
	if err := r.syncSchedules(time.Now()); err != nil {
		return schedule, err
	}
	return r.scheduleService.GetByID(schedule.ID)
}

func (r *AppRuntime) UpdateSchedule(id uint, req model.UpsertScheduleReq) (model.ScanScheduleDB, error) {
	current, err := r.scheduleService.GetByID(id)
	if err != nil {
		return current, err
	}
	next, err := buildScheduleModel(req)
	if err != nil {
		return current, err
	}
	current.Name = next.Name
	current.Enabled = next.Enabled
	current.Timezone = next.Timezone
	current.Mode = next.Mode
	current.CronExpr = next.CronExpr
	current.ScheduleConfig = next.ScheduleConfig
	current.ScanArgs = next.ScanArgs
	if err := r.scheduleService.Update(&current); err != nil {
		return current, err
	}
	if err := r.syncSchedules(time.Now()); err != nil {
		return current, err
	}
	return r.scheduleService.GetByID(id)
}

func (r *AppRuntime) DeleteSchedule(id uint) error {
	if err := r.scheduleService.Delete(id); err != nil {
		return err
	}
	return r.syncSchedules(time.Now())
}

func (r *AppRuntime) SetScheduleEnabled(id uint, enabled bool) (model.ScanScheduleDB, error) {
	schedule, err := r.scheduleService.GetByID(id)
	if err != nil {
		return schedule, err
	}
	schedule.Enabled = enabled
	if !enabled {
		schedule.NextRunAt = nil
	}
	if err := r.scheduleService.Update(&schedule); err != nil {
		return schedule, err
	}
	if err := r.syncSchedules(time.Now()); err != nil {
		return schedule, err
	}
	return r.scheduleService.GetByID(id)
}

func (r *AppRuntime) RunSchedule(id uint) (model.ScanJobDB, error) {
	schedule, err := r.scheduleService.GetByID(id)
	if err != nil {
		return model.ScanJobDB{}, err
	}
	var args model.DoScanImgArg
	if err := json.Unmarshal([]byte(schedule.ScanArgs), &args); err != nil {
		return model.ScanJobDB{}, err
	}
	return r.runScheduleJob(&schedule, args, time.Now())
}

func (r *AppRuntime) runScheduleJob(schedule *model.ScanScheduleDB, args model.DoScanImgArg, now time.Time) (model.ScanJobDB, error) {
	job, err := r.CreateJob(model.JobSourceSchedule, &schedule.ID, args)
	if err != nil {
		return job, err
	}
	schedule.LastRunAt = &now
	schedule.LastJobID = &job.ID
	schedule.LastJobStatus = job.Status
	if schedule.Enabled {
		if nextRun, nextErr := nextScheduleTime(*schedule, now); nextErr == nil {
			schedule.NextRunAt = &nextRun
		}
	}
	_ = r.scheduleService.Update(schedule)
	return job, nil
}

func (r *AppRuntime) scheduleLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case now := <-ticker.C:
			if err := r.syncSchedules(now); err != nil {
				tools.Logger.Error("sync schedules error : ", err)
			}
		}
	}
}

func (r *AppRuntime) syncSchedules(now time.Time) error {
	r.scheduleMu.Lock()
	defer r.scheduleMu.Unlock()

	schedules, err := r.scheduleService.ListEnabled()
	if err != nil {
		return err
	}
	for _, schedule := range schedules {
		if schedule.NextRunAt != nil && !schedule.NextRunAt.After(now) {
			var args model.DoScanImgArg
			if err := json.Unmarshal([]byte(schedule.ScanArgs), &args); err != nil {
				tools.Logger.Error("parse schedule scan args error : ", err)
				continue
			}
			if _, err := r.runScheduleJob(&schedule, args, now); err != nil {
				tools.Logger.Error("run schedule error : ", err)
			}
			continue
		}

		nextRun, err := nextScheduleTime(schedule, now)
		if err != nil {
			tools.Logger.Error("build next schedule time error : ", err)
			continue
		}
		if schedule.NextRunAt == nil || !schedule.NextRunAt.Equal(nextRun) {
			schedule.NextRunAt = &nextRun
			_ = r.scheduleService.Update(&schedule)
		}
	}
	return nil
}

func buildScheduleModel(req model.UpsertScheduleReq) (model.ScanScheduleDB, error) {
	timezone := req.Timezone
	if timezone == "" {
		timezone = defaultScheduleTimezone()
	}
	scanArgs := NormalizeScanArgs(req.ScanArgs)
	cronExpr := req.CronExpr
	if req.Mode != model.ScheduleModeCustom {
		cfg, err := parseScheduleConfig(req.ScheduleConfig)
		if err != nil {
			return model.ScanScheduleDB{}, err
		}
		cronExpr, err = buildCronExpr(req.Mode, cfg)
		if err != nil {
			return model.ScanScheduleDB{}, err
		}
	}
	if cronExpr == "" {
		return model.ScanScheduleDB{}, errors.New("Cron 表达式不能为空")
	}
	if err := validateScanArgs(scanArgs); err != nil {
		return model.ScanScheduleDB{}, err
	}
	return model.ScanScheduleDB{
		Name:           req.Name,
		Enabled:        req.Enabled,
		Timezone:       timezone,
		Mode:           req.Mode,
		CronExpr:       cronExpr,
		ScheduleConfig: req.ScheduleConfig,
		ScanArgs:       tools.MarshalJsonToString(scanArgs),
	}, nil
}

func defaultScheduleTimezone() string {
	locationName := time.Local.String()
	if locationName != "" && locationName != "Local" {
		if _, err := time.LoadLocation(locationName); err == nil {
			return locationName
		}
	}
	if timezone := os.Getenv("TZ"); timezone != "" {
		if _, err := time.LoadLocation(timezone); err == nil {
			return timezone
		}
	}
	return ""
}

func parseScheduleConfig(raw string) (scheduleConfig, error) {
	if raw == "" {
		return scheduleConfig{}, nil
	}
	var cfg scheduleConfig
	err := json.Unmarshal([]byte(raw), &cfg)
	return cfg, err
}

func buildCronExpr(mode string, cfg scheduleConfig) (string, error) {
	switch mode {
	case model.ScheduleModeHourly:
		return fmt.Sprintf("%d * * * *", cfg.Minute), nil
	case model.ScheduleModeDaily:
		return fmt.Sprintf("%d %d * * *", cfg.Minute, cfg.Hour), nil
	case model.ScheduleModeWeekly:
		if len(cfg.Weekdays) == 0 {
			return "", errors.New("每周模式必须选择星期")
		}
		parts := make([]string, 0, len(cfg.Weekdays))
		for _, weekday := range cfg.Weekdays {
			parts = append(parts, fmt.Sprintf("%d", weekday))
		}
		return fmt.Sprintf("%d %d * * %s", cfg.Minute, cfg.Hour, joinCSV(parts)), nil
	case model.ScheduleModeMonthly:
		if cfg.DayOfMonth <= 0 {
			cfg.DayOfMonth = 1
		}
		return fmt.Sprintf("%d %d %d * *", cfg.Minute, cfg.Hour, cfg.DayOfMonth), nil
	case model.ScheduleModeCustom:
		return "", nil
	default:
		return "", errors.New("不支持的计划模式")
	}
}

func buildCronSpec(schedule model.ScanScheduleDB) (string, error) {
	if schedule.CronExpr == "" {
		return "", errors.New("Cron 表达式不能为空")
	}
	if schedule.Timezone == "" {
		return schedule.CronExpr, nil
	}
	return fmt.Sprintf("CRON_TZ=%s %s", schedule.Timezone, schedule.CronExpr), nil
}

func hasActionEnabled(scanArgs model.DoScanImgArg) bool {
	return boolValue(scanArgs.DeleteAction) ||
		boolValue(scanArgs.MoveFileAction) ||
		boolValue(scanArgs.ModifyDateAction) ||
		boolValue(scanArgs.RenameFileAction)
}

func validateScanArgs(scanArgs model.DoScanImgArg) error {
	startPath := ""
	if scanArgs.StartPath != nil {
		startPath = strings.TrimSpace(*scanArgs.StartPath)
	}
	if startPath == "" {
		return &scanArgValidationError{message: "startPath is empty"}
	}
	info, err := os.Stat(startPath)
	if err != nil {
		return &scanArgValidationError{message: fmt.Sprintf("startPath invalid: %v", err)}
	}
	if !info.IsDir() {
		return &scanArgValidationError{message: "startPath is not a directory"}
	}

	if scanArgs.StartPathBak != nil && strings.TrimSpace(*scanArgs.StartPathBak) != "" {
		backupPath := strings.TrimSpace(*scanArgs.StartPathBak)
		backupInfo, backupErr := os.Stat(backupPath)
		if backupErr != nil {
			return &scanArgValidationError{message: fmt.Sprintf("startPathBak invalid: %v", backupErr)}
		}
		if !backupInfo.IsDir() {
			return &scanArgValidationError{message: "startPathBak is not a directory"}
		}
	}

	return nil
}

func NormalizeScanArgs(scanArgs model.DoScanImgArg) model.DoScanImgArg {
	if scanArgs.StartPath == nil || *scanArgs.StartPath == "" {
		scanArgs.StartPath = stringPtr(cons.StartPath)
	}
	if scanArgs.StartPathBak == nil || *scanArgs.StartPathBak == "" {
		scanArgs.StartPathBak = stringPtr(cons.StartPathBak)
	}
	if scanArgs.DeleteShow == nil {
		scanArgs.DeleteShow = boolPtr(cons.DeleteShow)
	}
	if scanArgs.MoveFileShow == nil {
		scanArgs.MoveFileShow = boolPtr(cons.MoveFileShow)
	}
	if scanArgs.ModifyDateShow == nil {
		scanArgs.ModifyDateShow = boolPtr(cons.ModifyDateShow)
	}
	if scanArgs.RenameFileShow == nil {
		scanArgs.RenameFileShow = boolPtr(cons.RenameFileShow)
	}
	if scanArgs.Md5Show == nil {
		scanArgs.Md5Show = boolPtr(cons.Md5Show)
	}
	if scanArgs.DeleteAction == nil {
		scanArgs.DeleteAction = boolPtr(cons.DeleteAction)
	}
	if scanArgs.MoveFileAction == nil {
		scanArgs.MoveFileAction = boolPtr(cons.MoveFileAction)
	}
	if scanArgs.ModifyDateAction == nil {
		scanArgs.ModifyDateAction = boolPtr(cons.ModifyDateAction)
	}
	if scanArgs.RenameFileAction == nil {
		scanArgs.RenameFileAction = boolPtr(cons.RenameFileAction)
	}
	return scanArgs
}

func buildScanExecutionSnapshot(scanArgs model.DoScanImgArg) map[string]any {
	snapshot := map[string]any{}
	if err := json.Unmarshal([]byte(tools.MarshalJsonToString(scanArgs)), &snapshot); err != nil {
		snapshot = map[string]any{}
	}

	for key, value := range currentSystemConfigSnapshot() {
		snapshot[key] = value
	}
	return snapshot
}

func currentSystemConfigSnapshot() map[string]any {
	return map[string]any{
		"DbUsername":        cons.DbUsername,
		"DbPassword":        maskRuntimeSecret(cons.DbPassword),
		"DbHost":            cons.DbHost,
		"DbPort":            cons.DbPort,
		"DbName":            cons.DbName,
		"DbConfig":          cons.DbConfig,
		"HttpPort":          cons.HttpPort,
		"HttpUsername":      cons.HttpUsername,
		"HttpPassword":      maskRuntimeSecret(cons.HttpPassword),
		"ColorOutput":       cons.AppConfig.Basic.ColorOutput,
		"SqlDebug":          cons.SqlDebug,
		"ImgCache":          cons.ImgCache,
		"SyncTable":         cons.SyncTable,
		"TruncateTable":     cons.TruncateTable,
		"PoolSize":          cons.PoolSize,
		"Md5Retry":          cons.Md5Retry,
		"Md5CountLength":    cons.Md5CountLength,
		"key":               maskRuntimeSecret(cons.GisKey),
		"IDInsertBatchSize": cons.IDInsertBatchSize,
		"IDDeleteBatchSize": cons.IDDeleteBatchSize,
		"GDUpdateBatchSize": cons.GDUpdateBatchSize,
	}
}

func maskRuntimeSecret(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) == 1 {
		return string(runes[0])
	}
	return string(runes[0]) + strings.Repeat("*", len(runes)-1)
}

func boolPtr(v bool) *bool {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func boolValue(v *bool) bool {
	return v != nil && *v
}

func joinCSV(parts []string) string {
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += ","
		}
		out += part
	}
	return out
}

func nextScheduleTime(schedule model.ScanScheduleDB, from time.Time) (time.Time, error) {
	spec, err := buildCronSpec(schedule)
	if err != nil {
		return time.Time{}, err
	}
	location := time.Local
	expr := spec
	if strings.HasPrefix(spec, "CRON_TZ=") {
		parts := strings.SplitN(spec, " ", 2)
		if len(parts) != 2 {
			return time.Time{}, errors.New("Cron 表达式不合法")
		}
		locationName := strings.TrimPrefix(parts[0], "CRON_TZ=")
		location, err = time.LoadLocation(locationName)
		if err != nil {
			return time.Time{}, err
		}
		expr = parts[1]
	}

	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return time.Time{}, errors.New("Cron 表达式必须包含 5 段")
	}

	next := from.In(location).Truncate(time.Minute).Add(time.Minute)
	for i := 0; i < 366*24*60; i++ {
		if cronFieldMatches(fields[0], next.Minute()) &&
			cronFieldMatches(fields[1], next.Hour()) &&
			cronFieldMatches(fields[2], next.Day()) &&
			cronFieldMatches(fields[3], int(next.Month())) &&
			cronFieldMatches(fields[4], int(next.Weekday())) {
			return next, nil
		}
		next = next.Add(time.Minute)
	}
	return time.Time{}, errors.New("no matching schedule found in next year")
}

func cronFieldMatches(expr string, value int) bool {
	if expr == "*" {
		return true
	}
	for _, part := range strings.Split(expr, ",") {
		if strings.Contains(part, "/") {
			stepParts := strings.SplitN(part, "/", 2)
			if len(stepParts) != 2 {
				continue
			}
			step, err := strconv.Atoi(stepParts[1])
			if err != nil || step <= 0 {
				continue
			}
			base := stepParts[0]
			if base == "*" {
				if value%step == 0 {
					return true
				}
				continue
			}
			if strings.Contains(base, "-") {
				bounds := strings.SplitN(base, "-", 2)
				if len(bounds) != 2 {
					continue
				}
				start, err1 := strconv.Atoi(bounds[0])
				end, err2 := strconv.Atoi(bounds[1])
				if err1 == nil && err2 == nil && value >= start && value <= end && (value-start)%step == 0 {
					return true
				}
				continue
			}
			start, err := strconv.Atoi(base)
			if err == nil && value >= start && (value-start)%step == 0 {
				return true
			}
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			if len(bounds) != 2 {
				continue
			}
			start, err1 := strconv.Atoi(bounds[0])
			end, err2 := strconv.Atoi(bounds[1])
			if err1 == nil && err2 == nil && value >= start && value <= end {
				return true
			}
			continue
		}
		num, err := strconv.Atoi(part)
		if err == nil && num == value {
			return true
		}
	}
	return false
}

func ginH(kv ...any) map[string]any {
	ret := map[string]any{}
	for i := 0; i+1 < len(kv); i += 2 {
		key, _ := kv[i].(string)
		ret[key] = kv[i+1]
	}
	return ret
}

func resolveJobDeleteListPath(scanUUID string) (string, error) {
	root := filepath.Join(cons.WorkDir, "log", "dump_delete_file")
	target := filepath.Join(root, scanUUID, "dump_delete_list")
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid scanUuid path")
	}
	return target, nil
}
