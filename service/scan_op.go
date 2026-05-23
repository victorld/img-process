package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"img_process/cons"
	"img_process/dao"
	"img_process/middleware"
	"img_process/model"
	"img_process/tools"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
)

const monthFilter = "xx" //月份过滤参数，打印使用
const dayFilter = "xx"   //日期过滤参数，打印使用
const backupDiffSummarySampleLimit = 20

var imgDatabaseService = dao.ImgDatabaseService{}
var imgRecordService = dao.ImgRecordService{}
var gisDatabaseService = dao.GisDatabaseService{}

type dirStruct struct { //目录打印需要的结构体
	dir        string
	isEmptyDir bool
	actionID   uint
}

type photoStruct struct { //照片打印需要的结构体
	photo            string
	dirDate          string
	modifyDate       string
	shootDate        string
	shootDateRaw     string
	fileDate         string
	minDate          string
	isDeleteFile     bool
	isMoveFile       bool
	moveTargetPath   string
	isModifyDateFile bool
	isRenameFile     bool
	renameTargetPath string
	deleteActionID   uint
	moveActionID     uint
	modifyActionID   uint
	renameActionID   uint
}

type ImgRecord struct {
	ScanArgs                 string         //扫描参数
	FileTotal                int            //文件总数
	FileTotalBak             *int           //文件总数
	DirTotal                 int            //目录总数
	DirTotalBak              *int           //目录总数
	StartDate                time.Time      //记录时间
	UseTime                  int            //用时
	BakNewFileCnt            int            //用时
	BakDeleteFileCnt         int            //用时
	BasePath                 string         //基础目录
	BasePathBak              string         //基础目录
	BakNewFile               string         //基础目录
	BakDeleteFile            string         //基础目录
	SuffixMap                map[string]int //后缀统计
	SuffixMapBak             map[string]int //后缀统计
	YearMap                  map[string]int //年份统计
	YearMapBak               map[string]int //年份统计
	FileDateCnt              int            //有时间文件统计
	DeleteFileCnt            int            //需要删除文件数
	ModifyDateFileCnt        int            //需要修改修改日期文件数
	MoveFileCnt              int            //需要移动文件数
	RenameFileCnt            int            //需要改名文件数
	ShootDateMismatchFileCnt int            //需要修改拍摄日期文件数
	ShootDateNullFileCnt     int            //没有拍摄时间
	ShootDateEarlierFileCnt  int            //需要修改拍摄日期文件数，更早
	EmptyDirCnt              int            //空文件数
	DumpFileCnt              int            //重复md5数
	PathDuplicateFileCnt     int            //文件路径重复项数
	ExifDateNameSet          string         //需要删除文件数
	ExifErrCnt               int            //exif错误数
	IsComplete               int            //是否完整
	Remark                   string         //备注
}

type backupDiffSummary struct {
	Count        int      `json:"count"`
	Sample       []string `json:"sample"`
	SampleLimit  int      `json:"sampleLimit"`
	Truncated    bool     `json:"truncated"`
	ArtifactPath string   `json:"artifactPath"`
}

type Scanner struct {
	scanArgs model.DoScanImgArg

	startPath        string
	startPathBak     string
	basePath         string
	deleteShow       bool
	moveFileShow     bool
	modifyDateShow   bool
	renameFileShow   bool
	md5Show          bool
	deleteAction     bool
	moveFileAction   bool
	modifyDateAction bool
	renameFileAction bool
	scanUUID         string

	deleteDirList   []dirStruct
	processFileList []photoStruct
	processFileMu   sync.Mutex
	stateMu         sync.Mutex

	md5DumpMap   map[string][]string
	md5DumpMapMu sync.Mutex

	getExifInfoErrorSuffixMap map[string]int
	getExifInfoErrorSet       mapset.Set
	errorStatsMu              sync.Mutex

	shouldDeleteMd5Files []string

	fileDateFileList          mapset.Set
	deleteFileList            mapset.Set
	moveFileList              mapset.Set
	renameFileList            mapset.Set
	modifyDateFileList        mapset.Set
	shootDateNullFileList     mapset.Set
	shootDateMismatchFileList mapset.Set
	shootDateEarlierFileList  mapset.Set
	exifDateNameSet           mapset.Set

	suffixMap      map[string]int
	suffixMapBak   map[string]int
	yearMap        map[string]int
	yearMapBak     map[string]int
	monthMap       map[string]int
	monthMapBak    map[string]int
	dayMap         map[string]int
	dayMapBak      map[string]int
	imageNumMap    map[string][]string
	imageNumRevMap map[string][]string
	pathDupMap     map[string][]string
	diffMap        map[string]int

	imgDatabaseDBList   []*model.ImgDatabaseDB
	imgDatabaseDBListMu sync.Mutex

	imgCache        map[string]middleware.ImgCacheData
	staleImgCache   map[string]middleware.ImgCacheData
	staleImgCacheMu sync.Mutex

	gisCache   map[string]middleware.GisData
	gisCacheMu sync.RWMutex

	fileTotalCnt    atomic.Int64
	fileTotalCntBak atomic.Int64
	dirTotalCnt     int
	dirTotalCntBak  int

	firstErr   error
	isComplete int
	metrics    scanMetrics
	recorder   ScanRecorder
	phase      string

	wg sync.WaitGroup

	loadImgCache             func() (map[string]middleware.ImgCacheData, error)
	loadGisCache             func() (map[string]middleware.GisData, error)
	getExifInfo              func(string) (string, string, int, string, []string, error)
	getLocationAddressOnline func(string) (string, error)
	createGisDatabase        func(*model.GisDatabaseDB) error
}

func (ps *photoStruct) psDatePrint() { //打印照片日期块信息
	if ps.dirDate != ps.minDate {
		tools.Logger.Info("dirDate : ", tools.StrWithColor(ps.dirDate, "red"))
	} else {
		tools.Logger.Info("dirDate : ", tools.StrWithColor(ps.dirDate, "green"))
	}
	if ps.modifyDate != ps.minDate {
		tools.Logger.Info("modifyDate : ", tools.StrWithColor(ps.modifyDate, "red"))
	} else {
		tools.Logger.Info("modifyDate : ", tools.StrWithColor(ps.modifyDate, "green"))
	}
	if ps.shootDate != ps.minDate {
		tools.Logger.Info("shootDate : ", tools.StrWithColor(ps.shootDate, "red"))
	} else {
		tools.Logger.Info("shootDate : ", tools.StrWithColor(ps.shootDate, "green"))
	}
	tools.Logger.Info("minDate : ", tools.StrWithColor(ps.minDate, "green"))
}

func newScanner(scanArgs model.DoScanImgArg) *Scanner {
	return newScannerWithRecorder(scanArgs, noopScanRecorder{})
}

func newScannerWithRecorder(scanArgs model.DoScanImgArg, recorder ScanRecorder) *Scanner {
	if recorder == nil {
		recorder = noopScanRecorder{}
	}
	return &Scanner{
		scanArgs:                  scanArgs,
		md5DumpMap:                make(map[string][]string),
		getExifInfoErrorSuffixMap: map[string]int{},
		getExifInfoErrorSet:       mapset.NewSet(),
		fileDateFileList:          mapset.NewSet(),
		deleteFileList:            mapset.NewSet(),
		moveFileList:              mapset.NewSet(),
		renameFileList:            mapset.NewSet(),
		modifyDateFileList:        mapset.NewSet(),
		shootDateNullFileList:     mapset.NewSet(),
		shootDateMismatchFileList: mapset.NewSet(),
		shootDateEarlierFileList:  mapset.NewSet(),
		exifDateNameSet:           mapset.NewSet(),
		suffixMap:                 map[string]int{},
		suffixMapBak:              map[string]int{},
		yearMap:                   map[string]int{},
		yearMapBak:                map[string]int{},
		monthMap:                  map[string]int{},
		monthMapBak:               map[string]int{},
		dayMap:                    map[string]int{},
		dayMapBak:                 map[string]int{},
		imageNumMap:               map[string][]string{},
		imageNumRevMap:            map[string][]string{},
		pathDupMap:                map[string][]string{},
		diffMap:                   map[string]int{},
		imgCache:                  map[string]middleware.ImgCacheData{},
		staleImgCache:             map[string]middleware.ImgCacheData{},
		gisCache:                  map[string]middleware.GisData{},
		isComplete:                1,
		loadImgCache:              middleware.LoadImgCache,
		loadGisCache:              middleware.LoadGisCache,
		getExifInfo:               middleware.GetExifInfo,
		getLocationAddressOnline:  middleware.GetLocationAddressOnline,
		createGisDatabase:         gisDatabaseService.CreateGisDatabase,
		recorder:                  recorder,
	}
}

// 扫描并将结果写入数据库
func ScanAndSave(scanArgs model.DoScanImgArg) (string, error) {
	return ScanAndSaveWithRecorder(scanArgs, noopScanRecorder{})
}

func ScanAndSaveWithRecorder(scanArgs model.DoScanImgArg, recorder ScanRecorder) (string, error) {
	imgRecordString, err := DoScanWithRecorder(scanArgs, recorder)
	if err != nil {
		tools.Logger.Error("scan result error : ", err)
		return "", err
	}

	var imgRecord ImgRecord
	if err := json.Unmarshal([]byte(imgRecordString), &imgRecord); err != nil {
		return "", err
	}

	imgRecordDB := model.ImgRecordDB{
		ScanArgs:                 imgRecord.ScanArgs,
		FileTotal:                intPtr(imgRecord.FileTotal),
		FileTotalBak:             imgRecord.FileTotalBak,
		DirTotal:                 intPtr(imgRecord.DirTotal),
		DirTotalBak:              imgRecord.DirTotalBak,
		StartDate:                timePtr(imgRecord.StartDate),
		UseTime:                  intPtr(imgRecord.UseTime),
		BasePath:                 imgRecord.BasePath,
		BasePathBak:              imgRecord.BasePathBak,
		SuffixMap:                tools.MarshalJsonToString(imgRecord.SuffixMap),
		SuffixMapBak:             tools.MarshalJsonToString(imgRecord.SuffixMapBak),
		YearMap:                  tools.MarshalJsonToString(imgRecord.YearMap),
		YearMapBak:               tools.MarshalJsonToString(imgRecord.YearMapBak),
		BakNewFileCnt:            intPtr(imgRecord.BakNewFileCnt),
		BakDeleteFileCnt:         intPtr(imgRecord.BakDeleteFileCnt),
		BakNewFile:               imgRecord.BakNewFile,
		BakDeleteFile:            imgRecord.BakDeleteFile,
		FileDateCnt:              intPtr(imgRecord.FileDateCnt),
		DeleteFileCnt:            intPtr(imgRecord.DeleteFileCnt),
		ModifyDateFileCnt:        intPtr(imgRecord.ModifyDateFileCnt),
		MoveFileCnt:              intPtr(imgRecord.MoveFileCnt),
		RenameFileCnt:            intPtr(imgRecord.RenameFileCnt),
		ShootDateMismatchFileCnt: intPtr(imgRecord.ShootDateMismatchFileCnt),
		ShootDateNullFileCnt:     intPtr(imgRecord.ShootDateNullFileCnt),
		ShootDateEarlierFileCnt:  intPtr(imgRecord.ShootDateEarlierFileCnt),
		EmptyDirCnt:              intPtr(imgRecord.EmptyDirCnt),
		DumpFileCnt:              intPtr(imgRecord.DumpFileCnt),
		PathDuplicateFileCnt:     intPtr(imgRecord.PathDuplicateFileCnt),
		ExifErrCnt:               intPtr(imgRecord.ExifErrCnt),
		ExifDateNameSet:          imgRecord.ExifDateNameSet,
		IsComplete:               intPtr(imgRecord.IsComplete),
		Remark:                   imgRecord.Remark,
	}

	if err = imgRecordService.CreateImgRecord(&imgRecordDB); err != nil {
		tools.Logger.Error("create error : ", err)
		return "", err
	}

	tools.Logger.Info("写入数据库成功")
	return imgRecordString, nil
}

func intPtr(v int) *int {
	return &v
}

func backupStatEnabled(startPathBak string) bool {
	return strings.TrimSpace(startPathBak) != ""
}

func buildBackupDiffSummary(items []string, artifactPath string) backupDiffSummary {
	summary := backupDiffSummary{
		Count:       len(items),
		SampleLimit: backupDiffSummarySampleLimit,
		Truncated:   len(items) > backupDiffSummarySampleLimit,
	}
	if len(items) > 0 {
		limit := backupDiffSummarySampleLimit
		if len(items) < limit {
			limit = len(items)
		}
		summary.Sample = append([]string(nil), items[:limit]...)
		summary.ArtifactPath = artifactPath
	} else {
		summary.Sample = []string{}
	}
	return summary
}

func backupDiffArtifactPath(scanUUID string, fileName string) string {
	return filepath.Join(cons.WorkDir, "log", "dump_delete_file", scanUUID, fileName)
}

func timePtr(v time.Time) *time.Time {
	return &v
}

// 扫描主体程序
func DoScan(scanArgs model.DoScanImgArg) (string, error) {
	return DoScanWithRecorder(scanArgs, noopScanRecorder{})
}

func DoScanWithRecorder(scanArgs model.DoScanImgArg, recorder ScanRecorder) (string, error) {
	scanner := newScannerWithRecorder(scanArgs, recorder)
	return scanner.Run()
}

func (s *Scanner) normalizeArgs() {
	if s.scanArgs.StartPath == nil || *s.scanArgs.StartPath == "" {
		s.scanArgs.StartPath = &cons.StartPath
	}
	if s.scanArgs.StartPathBak == nil || *s.scanArgs.StartPathBak == "" {
		s.scanArgs.StartPathBak = &cons.StartPathBak
	}
	if s.scanArgs.DeleteShow == nil {
		s.scanArgs.DeleteShow = &cons.DeleteShow
	}
	if s.scanArgs.MoveFileShow == nil {
		s.scanArgs.MoveFileShow = &cons.MoveFileShow
	}
	if s.scanArgs.ModifyDateShow == nil {
		s.scanArgs.ModifyDateShow = &cons.ModifyDateShow
	}
	if s.scanArgs.RenameFileShow == nil {
		s.scanArgs.RenameFileShow = &cons.RenameFileShow
	}
	if s.scanArgs.Md5Show == nil {
		s.scanArgs.Md5Show = &cons.Md5Show
	}
	if s.scanArgs.DeleteAction == nil {
		s.scanArgs.DeleteAction = &cons.DeleteAction
	}
	if s.scanArgs.MoveFileAction == nil {
		s.scanArgs.MoveFileAction = &cons.MoveFileAction
	}
	if s.scanArgs.ModifyDateAction == nil {
		s.scanArgs.ModifyDateAction = &cons.ModifyDateAction
	}
	if s.scanArgs.RenameFileAction == nil {
		s.scanArgs.RenameFileAction = &cons.RenameFileAction
	}

	s.startPath = *s.scanArgs.StartPath
	s.startPathBak = *s.scanArgs.StartPathBak
	s.deleteShow = *s.scanArgs.DeleteShow
	s.moveFileShow = *s.scanArgs.MoveFileShow
	s.modifyDateShow = *s.scanArgs.ModifyDateShow
	s.renameFileShow = *s.scanArgs.RenameFileShow
	s.md5Show = *s.scanArgs.Md5Show
	s.deleteAction = *s.scanArgs.DeleteAction
	s.moveFileAction = *s.scanArgs.MoveFileAction
	s.modifyDateAction = *s.scanArgs.ModifyDateAction
	s.renameFileAction = *s.scanArgs.RenameFileAction
}

func (s *Scanner) Run() (string, error) {
	s.normalizeArgs()

	scanUUID, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}

	s.scanUUID = time.Now().Format(tools.DatetimeDirTemplate) + "_" + strings.ReplaceAll(scanUUID.String(), "-", "")
	s.recorder.SetScanUUID(s.scanUUID)
	s.basePath, err = resolveBasePath(s.startPath)
	if err != nil {
		return "", fmt.Errorf("startPath error: %w", err)
	}

	if cons.TruncateTable {
		if err := imgDatabaseService.TruncateImgDatabase(); err != nil {
			return "", fmt.Errorf("truncate img_database: %w", err)
		}
		tools.Logger.Info("TruncateImgDatabase success!")
	}

	if cons.ImgCache {
		imgCache, err := s.loadImgCache()
		if err != nil {
			return "", fmt.Errorf("load image cache: %w", err)
		}
		s.imgCache = imgCache
		s.staleImgCache = cloneImgCache(imgCache)
	}

	gisCache, err := s.loadGisCache()
	if err != nil {
		return "", fmt.Errorf("load gis cache: %w", err)
	}
	s.gisCache = gisCache

	if !middleware.IsExiftoolAvailable() {
		tools.Logger.Warn("未检测到 exiftool，当前会自动回退到 Go EXIF 解析。建议先安装 exiftool（macOS 可执行：brew install exiftool），安装后照片和视频的拍摄时间、GPS 等信息识别效果会大大增强。")
	}

	defer tools.Logger.Sync()

	tools.Logger.Info("DoScan args final: ")
	tools.Logger.Info("startPath : ", s.startPath)
	tools.Logger.Info("startPathBak : ", s.startPathBak)
	tools.Logger.Info("deleteShow : ", s.deleteShow)
	tools.Logger.Info("moveFileShow : ", s.moveFileShow)
	tools.Logger.Info("modifyDateShow : ", s.modifyDateShow)
	tools.Logger.Info("renameFileShow : ", s.renameFileShow)
	tools.Logger.Info("md5Show : ", s.md5Show)
	tools.Logger.Info("deleteAction : ", s.deleteAction)
	tools.Logger.Info("moveFileAction : ", s.moveFileAction)
	tools.Logger.Info("modifyDateAction : ", s.modifyDateAction)
	tools.Logger.Info("renameFileAction : ", s.renameFileAction)
	tools.Logger.Info("SCAN JOBID : ", tools.StrWithColor(s.scanUUID, "red"))
	s.setPhase("initializing", ginH("scanUUID", s.scanUUID))

	start1 := time.Now()
	tools.Logger.Info()
	tools.Logger.Info("————————————————————————————————————————————————————————")
	tools.Logger.Info("time : ", start1.Format(tools.DatetimeTemplate))
	tools.Logger.Info("startPath : ", s.startPath)
	tools.Logger.Info("basePath : ", s.basePath)
	tools.Logger.Info("startPathBak : ", s.startPathBak)
	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("==========ROUND 1: SCAN FILE==========", "red"))
	tools.Logger.Info()
	s.setPhase("scan_primary", ginH("startPath", s.startPath))

	p, err := ants.NewPool(cons.PoolSize)
	if err != nil {
		return "", fmt.Errorf("create pool: %w", err)
	}
	defer p.Release()

	stopPrimaryTicker := s.startProgressTicker(&s.fileTotalCnt, "scan_primary")
	if err := s.walkPrimaryPath(p); err != nil {
		return "", err
	}
	s.wg.Wait()
	stopPrimaryTicker()
	s.recorder.SetTotalCount(s.fileTotalCnt.Load())

	elapsed2 := time.Since(start1)
	start3 := time.Now()

	if cons.ImgCache {
		if err := imgDatabaseService.CreateImgDatabaseBatch(s.snapshotImgDatabaseDBList()); err != nil {
			tools.Logger.Error("CreateImgDatabase error : ", err)
		}
	}

	elapsed3 := time.Since(start3)
	start4 := time.Now()
	basePathBak := ""

	if backupStatEnabled(s.startPathBak) {
		s.setPhase("scan_backup", ginH("startPathBak", s.startPathBak))
		basePathBak, err = resolveBasePath(s.startPathBak)
		if err != nil {
			return "", fmt.Errorf("StartPathBak error: %w", err)
		}
		tools.Logger.Info("basePathBak : ", basePathBak)
		stopBackupTicker := s.startProgressTicker(&s.fileTotalCntBak, "scan_backup")
		if err := s.walkBackupPath(p); err != nil {
			return "", err
		}
		stopBackupTicker()
	}

	elapsed4 := time.Since(start4)
	start5 := time.Now()

	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("==========ROUND 2: PROCESS FILE==========", "red"))
	tools.Logger.Info()
	s.setPhase("process_actions", nil)
	tools.Logger.Info(tools.StrWithColor("PRINT DETAIL TYPE1(delete file,modify date,move file): ", "red"))
	s.processFileProcess()
	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("PRINT DETAIL TYPE2(empty dir): ", "red"))
	s.emptyDirProcess()
	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("PRINT DETAIL TYPE3(dump file): ", "red"))
	s.setPhase("process_duplicates", nil)
	dumpMap := s.dumpFileProcess()
	pathDuplicateMap := s.pathDuplicateProcess()

	s.setPhase("build_result", nil)
	ret, err := s.buildResult(start1, basePathBak, dumpMap, pathDuplicateMap, elapsed2, elapsed3, elapsed4, start5)
	if err != nil {
		return "", err
	}
	return ret, nil
}

func (s *Scanner) walkPrimaryPath(p *ants.Pool) error {
	tools.Logger.Info()
	return filepath.Walk(s.startPath, func(file string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			s.recordNonFatalError(fmt.Errorf("walk %s: %w", file, walkErr))
			return nil
		}
		if info == nil {
			return nil
		}
		if info.IsDir() {
			if flag, err := tools.IsEmpty(file); err == nil && flag {
				actionID := s.recorder.RecordCandidateAction(model.ScanActionItemDB{
					ActionType: model.ActionTypeDeleteEmptyDir,
					ObjectType: model.ActionObjectDir,
					SourcePath: file,
					ReasonCode: "empty_dir",
					ReasonText: "目录为空，建议删除",
				})
				s.deleteDirList = append(s.deleteDirList, dirStruct{isEmptyDir: true, dir: file, actionID: actionID})
			}
			s.dirTotalCnt++
			return nil
		}

		fileName := filepath.Base(file)
		fileSuffix := strings.ToLower(path.Ext(file))
		if shouldDeleteFileByName(file) {
			ps := newDeletePhotoStruct(file)
			ps.deleteActionID = s.recorder.RecordCandidateAction(model.ScanActionItemDB{
				ActionType: model.ActionTypeDelete,
				ObjectType: model.ActionObjectFile,
				SourcePath: file,
				ReasonCode: "invalid_name",
				ReasonText: "文件名命中删除规则",
				MetadataJSON: tools.MarshalJsonToString(ginH(
					"fileName", filepath.Base(file),
					"currentPath", file,
				)),
			})
			s.processFileMu.Lock()
			s.processFileList = append(s.processFileList, ps)
			s.processFileMu.Unlock()
			s.deleteFileList.Add(file)
			return nil
		}

		parentDir := path.Base(filepath.Dir(file))
		dumpCompareKey := parentDir + "|" + fileName
		imgKey := tools.GetDirDate(file) + "|" + fileName
		day := tools.GetDirDate(file)
		year, month, hasDate := splitDateParts(day)
		s.suffixMap[fileSuffix]++
		if hasDate {
			s.dayMap[day]++
			s.yearMap[year]++
			s.monthMap[month]++
		}
		if hasDate && strings.HasPrefix(fileName, "IMG_") && len(fileName) >= 5 {
			head := fileName[4:5]
			s.imageNumMap[fileName] = append(s.imageNumMap[fileName], day)
			s.imageNumRevMap[year+"-"+head] = append(s.imageNumRevMap[year+"-"+head], fileName+","+day)
		}
		s.diffMap[dumpCompareKey] = 0
		s.pathDupMap[imgKey] = append(s.pathDupMap[imgKey], file)

		if count := s.fileTotalCnt.Add(1); count%1000 == 0 {
			tools.Logger.Info("processed ", tools.StrWithColor(strconv.FormatInt(count, 10), "red"))
			tools.Logger.Info("pool running size : ", p.Running())
		}

		s.wg.Add(1)
		if err := p.Submit(func() {
			defer s.wg.Done()
			s.processOneFile(file)
		}); err != nil {
			s.wg.Done()
			s.recordNonFatalError(fmt.Errorf("submit %s: %w", file, err))
		}

		return nil
	})
}

func (s *Scanner) walkBackupPath(p *ants.Pool) error {
	return filepath.Walk(s.startPathBak, func(file string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			s.recordNonFatalError(fmt.Errorf("walk backup %s: %w", file, walkErr))
			return nil
		}
		if info == nil {
			return nil
		}
		if info.IsDir() {
			s.dirTotalCntBak++
			return nil
		}

		fileName := filepath.Base(file)
		fileSuffix := strings.ToLower(path.Ext(file))
		if shouldDeleteFileByName(file) {
			return nil
		}

		count := s.fileTotalCntBak.Add(1)
		parentDir := path.Base(filepath.Dir(file))
		dumpCompareKey := parentDir + "|" + fileName
		day := tools.GetDirDate(file)
		year, month, hasDate := splitDateParts(day)
		s.suffixMapBak[fileSuffix]++
		if hasDate {
			s.yearMapBak[year]++
			s.monthMapBak[month]++
			s.dayMapBak[day]++
		}
		if _, ok := s.diffMap[dumpCompareKey]; ok {
			s.diffMap[dumpCompareKey] = 1
		} else {
			s.diffMap[dumpCompareKey] = 2
		}

		if count%1000 == 0 {
			tools.Logger.Info("bak0-dir processed ", tools.StrWithColor(strconv.FormatInt(count, 10), "red"))
			tools.Logger.Info("pool running size : ", p.Running())
		}
		return nil
	})
}

func (s *Scanner) buildResult(start1 time.Time, basePathBak string, dumpMap map[string][]string, pathDuplicateMap map[string][]string, elapsed2, elapsed3, elapsed4 time.Duration, start5 time.Time) (string, error) {
	var bakNewFile []string
	var bakDeleteFile []string
	if backupStatEnabled(s.startPathBak) {
		for imgKey, flag := range s.diffMap {
			if flag == 0 {
				bakNewFile = append(bakNewFile, imgKey)
			}
			if flag == 2 {
				bakDeleteFile = append(bakDeleteFile, imgKey)
			}
		}
		sort.Strings(bakNewFile)
		sort.Strings(bakDeleteFile)
		tools.Logger.Info("bakNewFile(新增文件待备份) count : ", len(bakNewFile))
		tools.Logger.Info("bakDeleteFile(备份里删除文件) count : ", len(bakDeleteFile))
	}

	tools.Logger.Info(tools.StrWithColor("PRINT STAT TYPE0(comman info): ", "red"))
	tools.Logger.Info("suffixMap（后缀统计） : ", tools.MarshalJsonToString(s.suffixMap))
	tools.Logger.Info("yearMap（年份统计） : ", tools.MarshalJsonToString(s.yearMap))
	tools.Logger.Info("month count（月份统计） : ")
	tools.MapPrintWithFilter(s.monthMap, monthFilter)
	tools.Logger.Info("day count（日期统计） : ")
	tools.MapPrintWithFilter(s.dayMap, dayFilter)
	tools.Logger.Info("file total（总文件数） : ", tools.StrWithColor(strconv.FormatInt(s.fileTotalCnt.Load(), 10), "red"))
	tools.Logger.Info("dir total（总目录数） : ", tools.StrWithColor(strconv.Itoa(s.dirTotalCnt), "red"))
	tools.Logger.Info("file contain date(just for print)（照片名称带日志的数量） : ", tools.StrWithColor(strconv.Itoa(s.fileDateFileList.Cardinality()), "red"))
	tools.Logger.Info("exif parse error（exif解析出错的后缀汇总） : ", tools.StrWithColor(tools.MarshalJsonToString(s.getExifInfoErrorSuffixMap), "red"))
	tools.Logger.Info("ExifNameSet list（exif统计的所有日期key打印） : ", s.exifDateNameSet.String())

	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("PRINT STAT TYPE1: ", "red"))
	pr := "delete file total（删除文件统计） : " + tools.StrWithColor(strconv.Itoa(s.deleteFileList.Cardinality()), "red")
	if s.deleteFileList.Cardinality() > 0 && s.deleteAction {
		pr += tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)

	pr = "modify date total（没有shootdate且修改日期不对文件统计） : " + tools.StrWithColor(strconv.Itoa(s.modifyDateFileList.Cardinality()), "red")
	if s.modifyDateFileList.Cardinality() > 0 && s.modifyDateAction {
		pr += tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)

	pr = "move file total（移动文件统计） : " + tools.StrWithColor(strconv.Itoa(s.moveFileList.Cardinality()), "red")
	if s.moveFileList.Cardinality() > 0 && s.moveFileAction {
		pr += tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)

	pr = "rename file total（改名文件统计） : " + tools.StrWithColor(strconv.Itoa(s.renameFileList.Cardinality()), "red")
	if s.renameFileList.Cardinality() > 0 && s.renameFileAction {
		pr += tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)
	tools.Logger.Info("shoot date total（拍摄日期跟目录不一致统计） : ", tools.StrWithColor(strconv.Itoa(s.shootDateMismatchFileList.Cardinality()), "red"))
	tools.Logger.Info("shoot date total（拍摄日期没有统计） : ", tools.StrWithColor(strconv.Itoa(s.shootDateNullFileList.Cardinality()), "red"))
	tools.Logger.Info("shoot date total（拍摄日期跟目录不一致统计，且拍摄日期更小） : ", tools.StrWithColor(strconv.Itoa(s.shootDateEarlierFileList.Cardinality()), "red"))

	tools.Logger.Info()
	pr = "empty dir total（空目录总数） : " + tools.StrWithColor(strconv.Itoa(len(s.deleteDirList)), "red")
	if len(s.deleteDirList) > 0 && s.deleteAction {
		pr += tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)

	tools.Logger.Info()
	tools.Logger.Info("dump file total（重复文件组数量） : ", tools.StrWithColor(strconv.Itoa(len(dumpMap)), "red"))
	tools.Logger.Info("path duplicate total（文件路径重复项组数量） : ", tools.StrWithColor(strconv.Itoa(len(pathDuplicateMap)), "red"))
	if err := s.writeDumpArtifacts(dumpMap); err != nil {
		tools.Logger.Error("write dump artifacts error : ", err)
	}
	bakNewFileSummary, bakDeleteFileSummary := s.writeBackupDiffArtifacts(bakNewFile, bakDeleteFile)

	tools.Logger.Info("imageNumMap length（照片名数字顺序统计-照片key） : ", tools.StrWithColor(strconv.Itoa(len(s.imageNumMap)), "red"))
	if len(s.imageNumMap) != 0 {
		tools.ImageNumMapWriteToFile(s.imageNumMap, cons.WorkDir+"/log/img_num_list")
	}

	tools.Logger.Info("imageNumRevMap length（照片名数字顺序统计-月份key） : ", tools.StrWithColor(strconv.Itoa(len(s.imageNumRevMap)), "red"))
	if len(s.imageNumRevMap) != 0 {
		tools.ImageNumRevMapWriteToFile(s.imageNumRevMap, cons.WorkDir+"/log/img_num_rev_list")
	}

	staleImgCache := s.snapshotStaleImgCache()
	tools.Logger.Info("img_database需要新插入的数量: ", len(s.snapshotImgDatabaseDBList()))
	tools.Logger.Info("img_database没有匹配上key，应该删除的数量: ", len(staleImgCache))
	if cons.SyncTable && len(staleImgCache) != 0 {
		tools.Logger.Info("正在批量删除多余的img_database。。。 ")
		var imgKeyToDelete []string
		for key := range staleImgCache {
			imgKeyToDelete = append(imgKeyToDelete, key)
			if len(imgKeyToDelete) >= cons.IDDeleteBatchSize {
				_ = imgDatabaseService.DeleteImgDatabaseByImgKey(imgKeyToDelete)
				imgKeyToDelete = []string{}
			}
		}
		_ = imgDatabaseService.DeleteImgDatabaseByImgKey(imgKeyToDelete)
	}

	var imgDatabaseSearch model.ImgDatabaseSearch
	imgDatabaseTotal, _ := imgDatabaseService.GetImgDatabaseInfoCount(imgDatabaseSearch)
	tools.Logger.Info("文件总数 : ", s.fileTotalCnt.Load(), " , imgDatabase总数 : ", imgDatabaseTotal)

	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("==========ROUND 3: PROCESS COST==========", "red"))
	tools.Logger.Info()
	elapsed5 := time.Since(start5)
	tools.Logger.Info("执行主目录扫描完成耗时 : ", elapsed2)
	tools.Logger.Info("执行img_database批量写入完成耗时 : ", elapsed3)
	tools.Logger.Info("执行备目录扫描完成耗时 : ", elapsed4)
	tools.Logger.Info("执行数据处理完成耗时 : ", elapsed5)
	s.logMetrics()
	tools.Logger.Info()

	imgRecord := ImgRecord{
		FileTotal:                int(s.fileTotalCnt.Load()),
		DirTotal:                 s.dirTotalCnt,
		StartDate:                start1,
		UseTime:                  int(math.Ceil(elapsed2.Seconds() + elapsed3.Seconds() + elapsed4.Seconds() + elapsed5.Seconds())),
		BasePath:                 s.basePath,
		BasePathBak:              basePathBak,
		BakNewFileCnt:            len(bakNewFile),
		BakDeleteFileCnt:         len(bakDeleteFile),
		BakNewFile:               tools.MarshalJsonToString(bakNewFileSummary),
		BakDeleteFile:            tools.MarshalJsonToString(bakDeleteFileSummary),
		SuffixMap:                s.suffixMap,
		SuffixMapBak:             s.suffixMapBak,
		YearMap:                  s.yearMap,
		YearMapBak:               s.yearMapBak,
		FileDateCnt:              s.fileDateFileList.Cardinality(),
		DeleteFileCnt:            s.deleteFileList.Cardinality(),
		ModifyDateFileCnt:        s.modifyDateFileList.Cardinality(),
		MoveFileCnt:              s.moveFileList.Cardinality(),
		RenameFileCnt:            s.renameFileList.Cardinality(),
		ShootDateMismatchFileCnt: s.shootDateMismatchFileList.Cardinality(),
		ShootDateNullFileCnt:     s.shootDateNullFileList.Cardinality(),
		ShootDateEarlierFileCnt:  s.shootDateEarlierFileList.Cardinality(),
		EmptyDirCnt:              len(s.deleteDirList),
		DumpFileCnt:              len(dumpMap),
		PathDuplicateFileCnt:     len(pathDuplicateMap),
		ExifDateNameSet:          s.exifDateNameSet.String(),
		ExifErrCnt:               s.getExifInfoErrorSet.Cardinality(),
		ScanArgs:                 tools.MarshalJsonToString(s.scanArgs),
		IsComplete:               s.isComplete,
	}
	if backupStatEnabled(s.startPathBak) {
		imgRecord.FileTotalBak = intPtr(int(s.fileTotalCntBak.Load()))
		imgRecord.DirTotalBak = intPtr(s.dirTotalCntBak)
	}

	if s.firstErr != nil {
		imgRecord.Remark = s.firstErr.Error()
	}

	ret := tools.MarshalJsonToString(imgRecord)
	tools.Logger.Info("scan result : ", ret)
	s.recorder.Finish(imgRecord, cons.WorkDir+"/log/dump_delete_file/"+s.scanUUID)
	return ret, nil
}

func cloneImgCache(src map[string]middleware.ImgCacheData) map[string]middleware.ImgCacheData {
	dst := make(map[string]middleware.ImgCacheData, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func (s *Scanner) snapshotImgDatabaseDBList() []*model.ImgDatabaseDB {
	s.imgDatabaseDBListMu.Lock()
	defer s.imgDatabaseDBListMu.Unlock()

	ret := make([]*model.ImgDatabaseDB, len(s.imgDatabaseDBList))
	copy(ret, s.imgDatabaseDBList)
	return ret
}

func (s *Scanner) snapshotStaleImgCache() map[string]middleware.ImgCacheData {
	s.staleImgCacheMu.Lock()
	defer s.staleImgCacheMu.Unlock()

	ret := make(map[string]middleware.ImgCacheData, len(s.staleImgCache))
	for key, value := range s.staleImgCache {
		ret[key] = value
	}
	return ret
}

func (s *Scanner) startProgressTicker(counter *atomic.Int64, phase string) func() {
	ticker := time.NewTicker(time.Minute)
	done := make(chan struct{})
	var lastValue int64

	go func() {
		for {
			select {
			case t := <-ticker.C:
				current := counter.Load()
				tools.Logger.Info(tools.StrWithColor("Tick at "+t.Format(tools.DatetimeTemplate), "red") + tools.StrWithColor(" , tick range processed "+strconv.FormatInt(current-lastValue, 10), "red"))
				s.recorder.Heartbeat(phase, current, ginH("tickRange", current-lastValue))
				lastValue = current
			case <-done:
				return
			}
		}
	}()

	return func() {
		current := counter.Load()
		tools.Logger.Info(tools.StrWithColor("Tick at "+time.Now().Format(tools.DatetimeTemplate), "red") + tools.StrWithColor(" , tick range processed "+strconv.FormatInt(current-lastValue, 10), "red"))
		s.recorder.Heartbeat(phase, current, ginH("tickRange", current-lastValue, "final", true))
		close(done)
		ticker.Stop()
	}
}

func (s *Scanner) recordNonFatalError(err error) {
	if err == nil {
		return
	}

	tools.Logger.Error("scan warning : ", err)
	s.recorder.RecordError(s.phase, "", err, nil)

	s.errorStatsMu.Lock()
	defer s.errorStatsMu.Unlock()
	if s.firstErr == nil {
		s.firstErr = err
	}
	s.isComplete = 0
}

func (s *Scanner) recordExifError(suffix string, photo string) {
	s.errorStatsMu.Lock()
	defer s.errorStatsMu.Unlock()

	s.getExifInfoErrorSuffixMap[suffix]++
	s.getExifInfoErrorSet.Add(photo)
}

func resolveBasePath(scanPath string) (string, error) {
	if scanPath == "" {
		return "", errors.New("startPath is empty")
	}

	fileInfo, err := os.Stat(scanPath)
	if err != nil {
		return "", err
	}
	if !fileInfo.IsDir() {
		return "", errors.New("startPath is not a directory")
	}

	cleanPath := filepath.Clean(scanPath)
	if idx := strings.Index(cleanPath, "pic-new"); idx >= 0 {
		return cleanPath[:idx+7], nil
	}

	return cleanPath, nil
}

// 主目录遍历完成后，待处理文件处理
func (s *Scanner) processFileProcess() {
	s.processFileMu.Lock()
	processFileList := make([]photoStruct, len(s.processFileList))
	copy(processFileList, s.processFileList)
	s.processFileMu.Unlock()

	for _, ps := range processFileList {
		printFileFlag := false
		printDateFlag := false

		if ps.isDeleteFile {
			s.deleteFileProcess(ps, &printFileFlag)
		}
		if ps.isModifyDateFile {
			s.modifyDateProcess(ps, &printFileFlag, &printDateFlag)
		}
		if ps.isMoveFile {
			ps = s.moveFileProcess(ps, &printFileFlag, &printDateFlag)
		}
		if ps.isRenameFile {
			ps = s.renameFileProcess(ps, &printFileFlag, &printDateFlag)
		}
	}
}

func (s *Scanner) setPhase(phase string, payload map[string]any) {
	s.phase = phase
	s.recorder.SetPhase(phase, payload)
}

// 待删除文件处理逻辑
func (s *Scanner) deleteFileProcess(ps photoStruct, printFileFlag *bool) {
	if s.deleteShow || s.deleteAction {
		tools.Logger.Info()
		tools.Logger.Info("file : ", tools.StrWithColor(ps.photo, "blue"))
		*printFileFlag = true
		tools.Logger.Info(tools.StrWithColor("should delete file :", "yellow"), ps.photo, " SIZE: ", tools.GetFileSize(ps.photo))
	}

	if s.deleteAction {
		if err := tools.DeleteFile(ps.photo); err != nil {
			s.recordActionError("delete file", ps.photo, err)
			tools.Logger.Info(tools.StrWithColor("delete file failed:", "yellow"), ps.photo, err)
			s.recorder.RecordActionResult(ps.deleteActionID, false, err, ginH("path", ps.photo))
		} else {
			tools.Logger.Info(tools.StrWithColor("delete file sucessed:", "green"), ps.photo)
			s.recorder.RecordActionResult(ps.deleteActionID, true, nil, ginH("path", ps.photo))
		}
	}
}

// 待更新修改日期文件处理逻辑
func (s *Scanner) modifyDateProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool) {
	if s.modifyDateShow || s.modifyDateAction {
		if !*printFileFlag {
			tools.Logger.Info()
			tools.Logger.Info("file : ", tools.StrWithColor(ps.photo, "blue"))
			*printFileFlag = true
		}
		if !*printDateFlag {
			ps.psDatePrint()
			*printDateFlag = true
		}
		tools.Logger.Info(tools.StrWithColor("should modify file ", "yellow"), ps.photo, " modifyDate to ", ps.minDate)
	}

	if s.modifyDateAction {
		localLoc, _ := time.LoadLocation("Asia/Shanghai")
		tm, _ := time.ParseInLocation("2006-01-02 15:04:05", ps.minDate+" 12:00:00", localLoc)
		err := tools.ChangeModifyDate(ps.photo, tm)
		s.recordActionError("modify file", ps.photo, err)
		if err == nil {
			tools.Logger.Info(tools.StrWithColor("modify file ", "yellow"), ps.photo, "modifyDate to", ps.minDate, "get realdate", tools.GetModifyDate(ps.photo))
		}
		s.recorder.RecordActionResult(ps.modifyActionID, err == nil, err, ginH("path", ps.photo, "targetDate", ps.minDate))
	}
}

// 待移动文件处理逻辑
func (s *Scanner) moveFileProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool) photoStruct {
	if s.moveFileShow || s.moveFileAction {
		if !*printFileFlag {
			tools.Logger.Info()
			tools.Logger.Info("file : ", tools.StrWithColor(ps.photo, "blue"))
			*printFileFlag = true
		}
		if !*printDateFlag {
			ps.psDatePrint()
			*printDateFlag = true
		}
		tools.Logger.Info(tools.StrWithColor("should move file ", "yellow"), ps.photo, " to ", ps.moveTargetPath)
	}

	if s.moveFileAction {
		if err := tools.MoveFile(ps.photo, ps.moveTargetPath); err != nil {
			s.recordActionError("move file", ps.photo, err)
			tools.Logger.Error("move file failed: ", ps.photo, " to ", ps.moveTargetPath, " err: ", err)
			s.recorder.RecordActionResult(ps.moveActionID, false, err, ginH("path", ps.photo, "targetPath", ps.moveTargetPath))
		} else {
			tools.Logger.Info(tools.StrWithColor("move file ", "yellow"), ps.photo, " to ", ps.moveTargetPath)
			s.recorder.RecordActionResult(ps.moveActionID, true, nil, ginH("path", ps.photo, "targetPath", ps.moveTargetPath))
			ps.photo = ps.moveTargetPath
		}
	}
	return ps
}

// 重命名文件处理逻辑
func (s *Scanner) renameFileProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool) photoStruct {
	sourcePath := ps.photo
	if s.renameFileShow || s.renameFileAction {
		if !*printFileFlag {
			tools.Logger.Info()
			tools.Logger.Info("file : ", tools.StrWithColor(sourcePath, "blue"))
			*printFileFlag = true
		}
		if !*printDateFlag {
			ps.psDatePrint()
			*printDateFlag = true
		}
		tools.Logger.Info(tools.StrWithColor("should rename file ", "yellow"), sourcePath, " to ", ps.renameTargetPath)
	}

	if s.renameFileAction {
		if err := tools.MoveFile(sourcePath, ps.renameTargetPath); err != nil {
			s.recordActionError("rename file", sourcePath, err)
			tools.Logger.Error("rename file failed: ", sourcePath, " to ", ps.renameTargetPath, " err: ", err)
			s.recorder.RecordActionResult(ps.renameActionID, false, err, ginH("path", sourcePath, "targetPath", ps.renameTargetPath))
		} else {
			tools.Logger.Info(tools.StrWithColor("rename file ", "yellow"), sourcePath, " to ", ps.renameTargetPath)
			s.recorder.RecordActionResult(ps.renameActionID, true, nil, ginH("path", sourcePath, "targetPath", ps.renameTargetPath))
			ps.photo = ps.renameTargetPath
		}
	}
	return ps
}

// 空目录处理
func (s *Scanner) emptyDirProcess() {
	for _, ds := range s.deleteDirList {
		if ds.isEmptyDir {
			if s.deleteShow || s.deleteAction {
				tools.Logger.Info("dir : ", tools.StrWithColor(ds.dir, "blue"))
				tools.Logger.Info(tools.StrWithColor("should delete empty dir :", "yellow"), ds.dir)
			}

			if s.deleteAction {
				if err := tools.DeleteEmptyDir(ds.dir); err != nil {
					s.recordActionError("delete empty dir", ds.dir, err)
					tools.Logger.Info(tools.StrWithColor("delete empty dir failed:", "yellow"), ds.dir, err)
					s.recorder.RecordActionResult(ds.actionID, false, err, ginH("path", ds.dir))
				} else {
					tools.Logger.Info(tools.StrWithColor("delete empty dir sucessed:", "green"), ds.dir)
					s.recorder.RecordActionResult(ds.actionID, true, nil, ginH("path", ds.dir))
				}
			}
		}
		tools.Logger.Info()
	}
}

// 重复文件处理
func (s *Scanner) dumpFileProcess() map[string][]string {
	dumpMap := make(map[string][]string)
	if !s.md5Show {
		return dumpMap
	}

	s.md5DumpMapMu.Lock()
	defer s.md5DumpMapMu.Unlock()

	for md5, files := range s.md5DumpMap {
		if len(files) <= 1 {
			continue
		}

		dumpMap[md5] = append([]string(nil), files...)
		minPhoto := ""
		var fileSizeTemp int64
		sizeMatch := true
		for _, photo := range files {
			if fileSizeTemp == 0 {
				fileSizeTemp = tools.GetFileSize(photo)
			} else if fileSizeTemp != tools.GetFileSize(photo) {
				sizeMatch = false
			}

			if minPhoto == "" {
				minPhoto = photo
				continue
			}
			if tools.GetDirDate(minPhoto) > tools.GetDirDate(photo) {
				minPhoto = photo
			} else if tools.GetDirDate(minPhoto) == tools.GetDirDate(photo) && len(tools.GetParentDir(minPhoto)) < len(tools.GetParentDir(photo)) {
				minPhoto = photo
			} else if tools.GetDirDate(minPhoto) == tools.GetDirDate(photo) && len(tools.GetParentDir(minPhoto)) >= len(tools.GetParentDir(photo)) && len(path.Base(minPhoto)) > len(path.Base(photo)) {
				minPhoto = photo
			}
		}

		tools.Logger.Info("file : ", tools.StrWithColor(md5, "blue"))
		groupPhotos := buildDuplicateGroupPhotoMetadata(files, minPhoto, sizeMatch)
		if !sizeMatch {
			s.recorder.RecordCandidateAction(model.ScanActionItemDB{
				ActionType:     model.ActionTypeDeleteDup,
				ObjectType:     model.ActionObjectFile,
				SourcePath:     firstNonEmptyPath(files),
				TargetPath:     minPhoto,
				ReasonCode:     "duplicate_md5",
				ReasonText:     "重复文件候选，仅展示不建议自动删除",
				Stage:          model.ActionStageDiscovery,
				Status:         model.ActionStatusSkipped,
				DuplicateGroup: md5,
				MetadataJSON: tools.MarshalJsonToString(ginH(
					"fileName", filepath.Base(firstNonEmptyPath(files)),
					"currentPath", firstNonEmptyPath(files),
					"keepPath", minPhoto,
					"keepFileName", filepath.Base(minPhoto),
					"sizeMatch", sizeMatch,
					"deleteEligible", false,
					"deleteIneligibleReason", "同 MD5 分组内文件大小不一致，需人工核对",
					"duplicatePhotos", groupPhotos,
				)),
			})
		}
		for _, photo := range files {
			if photo != minPhoto {
				if sizeMatch {
					s.shouldDeleteMd5Files = append(s.shouldDeleteMd5Files, photo)
					s.recorder.RecordCandidateAction(model.ScanActionItemDB{
						ActionType:     model.ActionTypeDeleteDup,
						ObjectType:     model.ActionObjectFile,
						SourcePath:     photo,
						TargetPath:     minPhoto,
						ReasonCode:     "duplicate_md5",
						ReasonText:     "重复文件候选删除",
						DuplicateGroup: md5,
						MetadataJSON: tools.MarshalJsonToString(ginH(
							"fileName", filepath.Base(photo),
							"currentPath", photo,
							"keepPath", minPhoto,
							"keepFileName", filepath.Base(minPhoto),
							"sizeMatch", sizeMatch,
							"deleteEligible", true,
							"duplicatePhotos", groupPhotos,
						)),
					})
					tools.Logger.Info("choose : ", photo, tools.StrWithColor(" DELETE", "red"), " SIZE: ", tools.GetFileSize(photo))
				} else {
					tools.Logger.Info("choose : ", photo, tools.StrWithColor(" SAVE(SIZE MISMATCH)", "green"), " SIZE: ", tools.GetFileSize(photo))
				}
			} else if sizeMatch {
				tools.Logger.Info("choose : ", photo, tools.StrWithColor(" SAVE", "green"), " SIZE: ", tools.GetFileSize(photo))
			} else {
				tools.Logger.Info("choose : ", photo, tools.StrWithColor(" SAVE(SIZE MISMATCH)", "green"), " SIZE: ", tools.GetFileSize(photo))
			}
		}
		tools.Logger.Info()
	}

	return dumpMap
}

func (s *Scanner) pathDuplicateProcess() map[string][]string {
	pathDuplicateMap := make(map[string][]string)
	for imgKey, files := range s.pathDupMap {
		files = uniqueCleanPaths(files)
		if len(files) <= 1 {
			continue
		}
		sort.Strings(files)
		pathDuplicateMap[imgKey] = append([]string(nil), files...)
		keepPhoto := choosePathDuplicateKeepPhoto(files)
		groupPhotos := buildPathDuplicateGroupPhotoMetadata(files, keepPhoto, imgKey)
		tools.Logger.Info("path duplicate : ", tools.StrWithColor(imgKey, "blue"))
		for _, photo := range files {
			if photo == keepPhoto {
				tools.Logger.Info("choose : ", photo, tools.StrWithColor(" SAVE", "green"), " SIZE: ", tools.GetFileSize(photo))
				continue
			}
			s.recorder.RecordCandidateAction(model.ScanActionItemDB{
				ActionType:     model.ActionTypeDeletePathDup,
				ObjectType:     model.ActionObjectFile,
				SourcePath:     photo,
				TargetPath:     keepPhoto,
				ReasonCode:     "duplicate_img_key",
				ReasonText:     "文件路径重复项候选删除",
				DuplicateGroup: imgKey,
				MetadataJSON: tools.MarshalJsonToString(ginH(
					"fileName", filepath.Base(photo),
					"currentPath", photo,
					"keepPath", keepPhoto,
					"keepFileName", filepath.Base(keepPhoto),
					"sizeMatch", true,
					"deleteEligible", true,
					"matchType", "img_key",
					"matchKey", imgKey,
					"duplicatePhotos", groupPhotos,
				)),
			})
			tools.Logger.Info("choose : ", photo, tools.StrWithColor(" DELETE", "red"), " SIZE: ", tools.GetFileSize(photo))
		}
		tools.Logger.Info()
	}
	return pathDuplicateMap
}

func choosePathDuplicateKeepPhoto(files []string) string {
	keepPhoto := ""
	for _, photo := range files {
		if keepPhoto == "" || pathDuplicateKeepLess(photo, keepPhoto) {
			keepPhoto = photo
		}
	}
	return keepPhoto
}

func pathDuplicateKeepLess(left string, right string) bool {
	leftDirDate := tools.GetDirDate(left)
	rightDirDate := tools.GetDirDate(right)
	if leftDirDate != "" && rightDirDate != "" && leftDirDate != rightDirDate {
		return leftDirDate < rightDirDate
	}
	if leftDirDate == "" && rightDirDate != "" {
		return false
	}
	if leftDirDate != "" && rightDirDate == "" {
		return true
	}
	leftHasDescription := pathDuplicateDirHasDescription(left)
	rightHasDescription := pathDuplicateDirHasDescription(right)
	if leftDirDate == rightDirDate && leftHasDescription != rightHasDescription {
		return leftHasDescription
	}
	leftParent := tools.GetParentDir(left)
	rightParent := tools.GetParentDir(right)
	if len(leftParent) != len(rightParent) {
		return len(leftParent) < len(rightParent)
	}
	return left < right
}

func pathDuplicateDirHasDescription(photo string) bool {
	dirName := tools.GetParentDir(photo)
	dirDate := tools.GetDirDate(photo)
	return dirDate != "" && len(dirName) > len(dirDate)
}

func firstNonEmptyPath(paths []string) string {
	for _, candidate := range paths {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return ""
}

func buildDuplicateGroupPhotoMetadata(files []string, recommendedDeletePath string, deleteEligible bool) []model.ScanActionPreview {
	photos := make([]model.ScanActionPreview, 0, len(files))
	for index, file := range files {
		sizeBytes := tools.GetFileSize(file)
		photos = append(photos, model.ScanActionPreview{
			FileName:          filepath.Base(file),
			Path:              file,
			SizeBytes:         sizeBytes,
			SizeText:          formatFileSize(sizeBytes),
			MD5Matched:        true,
			PathSource:        "扫描记录",
			RecommendedDelete: deleteEligible && file != recommendedDeletePath,
			DeleteEligible:    deleteEligible,
			CandidateIndex:    index,
			MatchCount:        1,
		})
	}
	return photos
}

func buildPathDuplicateGroupPhotoMetadata(files []string, keepPath string, imgKey string) []model.ScanActionPreview {
	photos := make([]model.ScanActionPreview, 0, len(files))
	for index, file := range files {
		sizeBytes := tools.GetFileSize(file)
		photos = append(photos, model.ScanActionPreview{
			FileName:          filepath.Base(file),
			Path:              file,
			SizeBytes:         sizeBytes,
			SizeText:          formatFileSize(sizeBytes),
			MD5Matched:        false,
			MatchKey:          imgKey,
			MatchType:         "img_key",
			PathSource:        "日期+文件名",
			RecommendedDelete: file != keepPath,
			DeleteEligible:    true,
			CandidateIndex:    index,
			MatchCount:        1,
		})
	}
	return photos
}

func (s *Scanner) writeDumpArtifacts(dumpMap map[string][]string) error {
	filePath := cons.WorkDir + "/log/dump_delete_file/" + s.scanUUID
	if len(dumpMap) != 0 {
		var builder strings.Builder
		for md5, files := range dumpMap {
			builder.WriteString(md5 + " : ")
			for index, file := range files {
				fileName := filepath.Base(file)
				if index == 0 {
					builder.WriteString(strings.Split(fileName, "[")[0])
				} else {
					builder.WriteString("|" + strings.Split(fileName, "[")[0])
				}
			}
			builder.WriteString("\n")
		}

		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		if err := tools.WriteStringToFile(builder.String(), filePath+"/dump_compare"); err != nil {
			return err
		}
		s.recorder.RecordArtifact("dump compare generated", filePath+"/dump_compare", nil)
	}

	tools.Logger.Info("shouldDeleteMd5Files length（重复文件应该删除的数量） : ", tools.StrWithColor(strconv.Itoa(len(s.shouldDeleteMd5Files)), "red"))
	if len(s.shouldDeleteMd5Files) != 0 {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		if err := tools.WriteStringToFile(strings.Join(s.shouldDeleteMd5Files, "\n"), filePath+"/dump_delete_list"); err != nil {
			return err
		}
		s.recorder.RecordArtifact("dump delete list generated", filePath+"/dump_delete_list", ginH("count", len(s.shouldDeleteMd5Files)))
	}

	return nil
}

func (s *Scanner) writeBackupDiffArtifacts(bakNewFile []string, bakDeleteFile []string) (backupDiffSummary, backupDiffSummary) {
	newPath := s.writeBackupDiffArtifact("backup new file list generated", "bak_new_file_list", bakNewFile)
	deletePath := s.writeBackupDiffArtifact("backup delete file list generated", "bak_delete_file_list", bakDeleteFile)
	return buildBackupDiffSummary(bakNewFile, newPath), buildBackupDiffSummary(bakDeleteFile, deletePath)
}

func (s *Scanner) writeBackupDiffArtifact(title string, fileName string, items []string) string {
	if len(items) == 0 {
		return ""
	}

	filePath := backupDiffArtifactPath(s.scanUUID, fileName)
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		tools.Logger.Error("write backup diff artifact mkdir error : ", err)
		s.recorder.RecordError("artifact", filePath, err, ginH("count", len(items)))
		return ""
	}
	if err := tools.WriteStringToFile(strings.Join(items, "\n"), filePath); err != nil {
		tools.Logger.Error("write backup diff artifact error : ", err)
		s.recorder.RecordError("artifact", filePath, err, ginH("count", len(items)))
		return ""
	}

	payload := ginH(
		"count", len(items),
		"sampleLimit", backupDiffSummarySampleLimit,
		"truncated", len(items) > backupDiffSummarySampleLimit,
	)
	s.recorder.RecordArtifact(title, filePath, payload)
	return filePath
}

// 遍历逻辑单文件处理
func (s *Scanner) processOneFile(photo string) {
	shootDateOrigin, locStreet, _ := s.getImgShootDateAndLoc(photo)
	meta := collectFileMetadata(photo, shootDateOrigin, locStreet)
	s.applyFileDecision(evaluateFileDecision(meta, s.basePath))

	if s.md5Show {
		start := time.Now()
		md5, err := tools.GetFileMD5WithRetry(photo, cons.Md5Retry, cons.Md5CountLength)
		s.recordMD5Duration(time.Since(start))
		if err != nil {
			tools.Logger.Info("GetFileMD5 err for ", cons.Md5Retry, " times : ", err, " file : ", photo)
		} else {
			s.md5DumpMapMu.Lock()
			s.md5DumpMap[md5] = append(s.md5DumpMap[md5], photo)
			s.md5DumpMapMu.Unlock()
		}
	}
}

func getRenameNewPhoto(photo string, shootDate string, locStreet string) string {
	photoNew := photo
	if shootDate != "" {
		if t, err := time.Parse("2006:01:02 15:04:05", shootDate); err == nil {
			shootDate = t.Format("2006-01-02_15-04-05")
		}
	}
	if shootDate == "" && locStreet == "" {
		return photoNew
	}

	fileRegexp := regexp.MustCompile(`^.*\[(.*)\].*$`)
	dateValList := fileRegexp.FindStringSubmatch(photo)
	var timeAndLocFile string
	if len(dateValList) == 2 {
		timeAndLocFile = dateValList[1]
	}

	timeAndLocShould := shootDate + "^" + locStreet
	dirDate := tools.GetDirDate(photo)
	if !strings.Contains(timeAndLocShould, dirDate) {
		timeAndLocShould = "inconsistent^" + timeAndLocShould
	}
	if timeAndLocFile == timeAndLocShould {
		return photoNew
	}

	if strings.Count(photo, "[") == 1 && strings.Count(photo, "]") == 1 {
		re := regexp.MustCompile(`\[.*\]`)
		return re.ReplaceAllString(photo, "["+timeAndLocShould+"]")
	}
	if strings.Count(photo, "[") == 0 && strings.Count(photo, "]") == 0 {
		if strings.Count(photo, ".") == 1 {
			return strings.ReplaceAll(photo, ".", "["+timeAndLocShould+"].")
		}
		fileName := filepath.Base(photo)
		fileSuffix := strings.ToLower(path.Ext(photo))
		nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(photo))
		parentDir := filepath.Dir(photo)
		photoNew = parentDir + string(filepath.Separator) + strings.ReplaceAll(nameWithoutExt, ".", "_") + fileSuffix
		photoNew = strings.ReplaceAll(photoNew, ".", "["+timeAndLocShould+"].")
		tools.Logger.Info("##################filePath with . , photo : ", photo, " photoNew : ", photoNew)
		return photoNew
	}

	tools.Logger.Error("##################filePath [] error , photo : ", photo)
	return photoNew
}

// 获取文件的拍摄时间
func (s *Scanner) getImgShootDateAndLoc(photo string) (string, string, error) {
	suffix := strings.ToLower(path.Ext(photo))
	fileName := filepath.Base(photo)
	dirDate := tools.GetDirDate(photo)
	imgKey := dirDate + "|" + fileName

	if value, ok := s.imgCache[imgKey]; ok {
		s.staleImgCacheMu.Lock()
		delete(s.staleImgCache, imgKey)
		s.staleImgCacheMu.Unlock()
		s.metrics.imgCacheHits.Add(1)
		return value.ShootDate, value.LocStreet, nil
	}

	shootDate, locNum, state, output, dateTagNames, err := s.getExifInfo(photo)
	s.stateMu.Lock()
	for _, dateTagName := range dateTagNames {
		s.exifDateNameSet.Add(dateTagName)
	}
	s.stateMu.Unlock()
	if state == 1 {
		s.metrics.exiftoolHits.Add(1)
	} else if state == 2 {
		s.metrics.exifGoHits.Add(1)
	}

	if err != nil {
		s.recordExifError(suffix, photo)
	}

	locStreet := ""
	locAddr := ""
	if locNum != "" {
		gisData, gisErr := s.getLocationAddressByCache(locNum)
		if gisErr == nil {
			locAddr = gisData.LocAddr
			locStreet = gisData.LocStreet
		}
	}

	if cons.ImgCache {
		s.imgDatabaseDBListMu.Lock()
		s.imgDatabaseDBList = append(s.imgDatabaseDBList, buildImgDatabaseRecord(imgKey, shootDate, locNum, output, state, locStreet, locAddr))
		s.imgDatabaseDBListMu.Unlock()
	}

	return shootDate, locStreet, err
}

func (s *Scanner) getLocationAddressByCache(locNum string) (middleware.GisData, error) {
	s.gisCacheMu.RLock()
	if value, ok := s.gisCache[locNum]; ok {
		s.gisCacheMu.RUnlock()
		return value, nil
	}
	s.gisCacheMu.RUnlock()

	if locNum == "0.000000,0.000000" {
		return middleware.GisData{}, errors.New("not right locNum")
	}

	s.metrics.gisRequests.Add(1)
	locJSON, err := s.getLocationAddressOnline(locNum)
	if err != nil {
		return middleware.GisData{}, err
	}
	gisData, err := middleware.GetGisDataFromJson(locJSON)
	if err != nil {
		return middleware.GisData{}, err
	}

	s.gisCacheMu.Lock()
	if value, ok := s.gisCache[locNum]; ok {
		s.gisCacheMu.Unlock()
		return value, nil
	}
	s.gisCache[locNum] = gisData
	s.gisCacheMu.Unlock()

	gisDatabaseDB := model.GisDatabaseDB{
		LocNum:    locNum,
		LocAddr:   gisData.LocAddr,
		LocStreet: gisData.LocStreet,
		LocJson:   locJSON,
	}
	if err := s.createGisDatabase(&gisDatabaseDB); err != nil {
		tools.Logger.Error("CreateGisDatabase error : ", err)
	}

	return gisData, nil
}
