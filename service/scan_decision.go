package service

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"img_process/model"
	"img_process/tools"
)

type scanMetrics struct {
	imgCacheHits    atomic.Int64
	exiftoolHits    atomic.Int64
	exifGoHits      atomic.Int64
	gisRequests     atomic.Int64
	md5TimeNanos    atomic.Int64
	md5ComputeCount atomic.Int64
}

type fileMetadata struct {
	photo           string
	dirDate         string
	fileDate        string
	modifyDate      string
	shootDate       string
	shootDateOrigin string
	locStreet       string
}

type fileDecision struct {
	photo           photoStruct
	hasChanges      bool
	fileDatePresent bool
	shouldMove      bool
	shouldRename    bool
	shouldModify    bool
	shootMismatch   bool
	shootEarlier    bool
	shootDateNull   bool
}

func collectFileMetadata(photo string, shootDateOrigin string, locStreet string) fileMetadata {
	shootDate := ""
	if shootDateOrigin != "" {
		if t, err := time.Parse("2006:01:02 15:04:05", shootDateOrigin); err == nil {
			shootDate = t.Format("2006-01-02")
		}
	}

	return fileMetadata{
		photo:           photo,
		dirDate:         tools.GetDirDate(photo),
		fileDate:        tools.GetFileDate(photo),
		modifyDate:      tools.GetModifyDate(photo),
		shootDate:       shootDate,
		shootDateOrigin: shootDateOrigin,
		locStreet:       locStreet,
	}
}

func evaluateFileDecision(meta fileMetadata, basePath string) fileDecision {
	minDate := meta.modifyDate
	if isEarlierDate(meta.dirDate, minDate) || minDate == "" {
		minDate = meta.dirDate
	}
	if meta.shootDate != "" && (isEarlierDate(meta.shootDate, minDate) || minDate == "") {
		minDate = meta.shootDate
	}

	ps := photoStruct{
		photo:        meta.photo,
		dirDate:      meta.dirDate,
		modifyDate:   meta.modifyDate,
		shootDate:    meta.shootDate,
		shootDateRaw: meta.shootDateOrigin,
		fileDate:     meta.fileDate,
		minDate:      minDate,
	}
	ret := fileDecision{
		photo:           ps,
		fileDatePresent: meta.fileDate != "",
		shootDateNull:   meta.shootDate == "",
	}

	if meta.shootDate != "" && meta.shootDate != meta.dirDate {
		ret.shootMismatch = true
		if isEarlierDate(meta.shootDate, meta.dirDate) {
			ret.shootEarlier = true
		}
	}

	if meta.shootDate == "" && meta.modifyDate != minDate {
		ret.shouldModify = true
		ret.photo.isModifyDateFile = true
		ret.hasChanges = true
	}

	if minDate != "" && meta.dirDate != "" && meta.dirDate != minDate {
		if year, month, ok := splitDateParts(minDate); ok {
			targetPath := filepath.Join(basePath, year, month, minDate)
			if realPath := tools.GetRealPath(targetPath); realPath != "" {
				targetPath = realPath
			}
			ret.photo.isMoveFile = true
			ret.photo.targetPhoto = filepath.Join(targetPath, filepath.Base(meta.photo))
			ret.shouldMove = true
			ret.hasChanges = true
		}
	}

	targetPhoto := getRenameNewPhoto(meta.photo, meta.shootDateOrigin, meta.locStreet)
	if meta.photo != targetPhoto {
		ret.photo.isRenameFile = true
		ret.photo.targetPhoto = targetPhoto
		ret.shouldRename = true
		ret.hasChanges = true
	}

	return ret
}

func splitDateParts(date string) (string, string, bool) {
	if len(date) < len("2006-01-02") {
		return "", "", false
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return "", "", false
	}
	return date[:4], date[:7], true
}

func isEarlierDate(left string, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return left < right
}

func buildImgDatabaseRecord(imgKey string, shootDate string, locNum string, output string, state int, locStreet string, locAddr string) *model.ImgDatabaseDB {
	return &model.ImgDatabaseDB{
		ImgKey:    imgKey,
		ShootDate: shootDate,
		LocNum:    locNum,
		LocStreet: locStreet,
		LocAddr:   locAddr,
		Remark:    output,
		State:     &state,
	}
}

func shouldDeleteFileByName(file string) bool {
	fileName := filepath.Base(file)
	return strings.HasSuffix(fileName, "_.pic.jpg") ||
		strings.HasPrefix(fileName, ".") ||
		strings.HasPrefix(fileName, "IMG_E") ||
		strings.HasSuffix(fileName, "nas_downloading") ||
		tools.GetFileSize(file) == 0
}

func newDeletePhotoStruct(file string) photoStruct {
	return photoStruct{isDeleteFile: true, photo: file}
}

func (s *Scanner) recordActionError(action string, target string, err error) {
	if err == nil {
		return
	}
	s.recordNonFatalError(err)
	tools.Logger.Error(action, " failed for ", target, " : ", err)
}

func (s *Scanner) recordMD5Duration(duration time.Duration) {
	s.metrics.md5ComputeCount.Add(1)
	s.metrics.md5TimeNanos.Add(duration.Nanoseconds())
}

func (s *Scanner) applyFileDecision(decision fileDecision) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if decision.fileDatePresent {
		s.fileDateFileList.Add(decision.photo.photo)
	}
	if decision.shouldMove {
		s.moveFileList.Add(decision.photo.photo)
	}
	if decision.shouldRename {
		s.renameFileList.Add(decision.photo.photo)
	}
	if decision.shouldModify {
		s.modifyDateFileList.Add(decision.photo.photo)
	}
	if decision.shootMismatch {
		s.shootDateMismatchFileList.Add(decision.photo.photo)
	}
	if decision.shootEarlier {
		s.shootDateEarlierFileList.Add(decision.photo.photo)
	}
	if decision.shootDateNull {
		s.shootDateNullFileList.Add(decision.photo.photo)
	}
	if decision.hasChanges {
		if decision.shouldMove {
			decision.photo.moveActionID = s.recorder.RecordCandidateAction(model.ScanActionItemDB{
				ActionType: model.ActionTypeMove,
				ObjectType: model.ActionObjectFile,
				SourcePath: decision.photo.photo,
				TargetPath: decision.photo.targetPhoto,
				ReasonCode: "dir_date_mismatch",
				ReasonText: "目录日期与最小日期不一致，需要移动",
				MetadataJSON: tools.MarshalJsonToString(ginH(
					"fileName", filepath.Base(decision.photo.photo),
					"currentPath", decision.photo.photo,
					"targetPath", decision.photo.targetPhoto,
					"dirDate", decision.photo.dirDate,
					"fileNameDate", decision.photo.fileDate,
					"shootDate", decision.photo.shootDate,
					"shootDateRaw", decision.photo.shootDateRaw,
					"minDate", decision.photo.minDate,
				)),
			})
		}
		if decision.shouldRename {
			decision.photo.renameActionID = s.recorder.RecordCandidateAction(model.ScanActionItemDB{
				ActionType: model.ActionTypeRename,
				ObjectType: model.ActionObjectFile,
				SourcePath: decision.photo.photo,
				TargetPath: decision.photo.targetPhoto,
				ReasonCode: "name_metadata_mismatch",
				ReasonText: "文件名中的时间地点信息与识别结果不一致，需要重命名",
				MetadataJSON: tools.MarshalJsonToString(ginH(
					"fileName", filepath.Base(decision.photo.photo),
					"targetFileName", filepath.Base(decision.photo.targetPhoto),
					"currentPath", decision.photo.photo,
					"targetPath", decision.photo.targetPhoto,
					"shootDate", decision.photo.shootDate,
					"shootDateRaw", decision.photo.shootDateRaw,
				)),
			})
		}
		if decision.shouldModify {
			decision.photo.modifyActionID = s.recorder.RecordCandidateAction(model.ScanActionItemDB{
				ActionType: model.ActionTypeModifyTime,
				ObjectType: model.ActionObjectFile,
				SourcePath: decision.photo.photo,
				ReasonCode: "modify_time_mismatch",
				ReasonText: "修改时间与最小日期不一致，需要修正",
				MetadataJSON: tools.MarshalJsonToString(ginH(
					"fileName", filepath.Base(decision.photo.photo),
					"currentPath", decision.photo.photo,
					"dirDate", decision.photo.dirDate,
					"fileNameDate", decision.photo.fileDate,
					"shootDate", decision.photo.shootDate,
					"shootDateRaw", decision.photo.shootDateRaw,
					"minDate", decision.photo.minDate,
					"targetDate", decision.photo.minDate,
				)),
			})
		}
		s.processFileMu.Lock()
		s.processFileList = append(s.processFileList, decision.photo)
		s.processFileMu.Unlock()
	}
}

func (s *Scanner) logMetrics() {
	md5Count := s.metrics.md5ComputeCount.Load()
	md5Avg := time.Duration(0)
	if md5Count > 0 {
		md5Avg = time.Duration(s.metrics.md5TimeNanos.Load() / md5Count)
	}
	tools.Logger.Info("scan metrics : imgCacheHits=", s.metrics.imgCacheHits.Load(),
		" exiftoolHits=", s.metrics.exiftoolHits.Load(),
		" exifGoHits=", s.metrics.exifGoHits.Load(),
		" gisRequests=", s.metrics.gisRequests.Load(),
		" md5Count=", md5Count,
		" md5Avg=", md5Avg)
}

func currentDirDate(file string) string {
	info, err := os.Stat(file)
	if err != nil || info.IsDir() {
		return ""
	}
	return tools.GetDirDate(file)
}
