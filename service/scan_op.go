package service

import (
	"encoding/json"
	"errors"
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
	"strconv"
	"strings"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
)

const monthFilter = "xx" //月份过滤参数，打印使用
const dayFilter = "xx"   //日期过滤参数，打印使用

var basePath string

var startPath string
var startPathBak string
var deleteShow bool
var moveFileShow bool
var modifyDateShow bool
var renameFileShow bool
var md5Show bool
var deleteAction bool
var moveFileAction bool
var modifyDateAction bool
var renameFileAction bool

var scanUuidFinal string

var deleteDirList []dirStruct //需要处理的目录结构体列表（空目录）

var processFileList []photoStruct //需要处理的文件结构体列表（非法格式删除、移动、修改时间、重复文件删除）
var processFileListMu sync.Mutex  //processFileList锁

var md5DumpMap = make(map[string][]string) //以md5为key存储的重复文件
var md5DumpMapMu sync.Mutex                //md5Map锁

var getExifInfoErrorSuffixMap = map[string]int{} //exif获取出问题的后缀统计
var getExifInfoErrorSet = mapset.NewSet()        //exif获取出问题的照片集合统计
var getExifInfoErrorSuffixMapMu sync.Mutex       //getExifInfoErrorSuffixMap锁

var shootDateCacheMapBakMu sync.Mutex //待删除的img_database map删除key使用的锁

var shouldDeleteMd5Files []string //统计需要删除的文件

var fileDateFileList = mapset.NewSet()          //文件名带日期的照片
var deleteFileList = mapset.NewSet()            //需要删除的文件
var moveFileList = mapset.NewSet()              //目录与最小日期不匹配，需要移动
var renameFileList = mapset.NewSet()            //文件名不一致，需要改名
var modifyDateFileList = mapset.NewSet()        //修改时间与最小日期不匹配，需要修改
var shootDateNullFileList = mapset.NewSet()     //没有拍摄时间
var shootDateMismatchFileList = mapset.NewSet() //拍摄时间与最小日期不匹配，需要修改
var shootDateEarlierFileList = mapset.NewSet()  //拍摄时间与最小日期不匹配，且拍摄日期比目录时间小，需要修改

var imgDatabaseDBList []*model.ImgDatabaseDB //img_database 待插入list

var imgDatabaseService = dao.ImgDatabaseService{}
var imgRecordService = dao.ImgRecordService{}
var gisDatabaseService = dao.GisDatabaseService{}

var wg sync.WaitGroup //异步照片处理等待

type dirStruct struct { //目录打印需要的结构体
	dir        string
	isEmptyDir bool
}

type photoStruct struct { //照片打印需要的结构体
	photo            string
	dirDate          string
	modifyDate       string
	shootDate        string
	fileDate         string
	minDate          string
	isDeleteFile     bool
	isMoveFile       bool
	targetPhoto      string
	isModifyDateFile bool
	isRenameFile     bool
}

type ImgRecord struct {
	ScanArgs                 string         //扫描参数
	FileTotal                int            //文件总数
	FileTotalBak             int            //文件总数
	DirTotal                 int            //目录总数
	DirTotalBak              int            //目录总数
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
	ShootDateNullFileCnt     int            //没有拍摄日期文件数
	ShootDateEarlierFileCnt  int            //需要修改拍摄日期文件数，更早
	EmptyDirCnt              int            //空文件数
	DumpFileCnt              int            //重复md5数
	ExifDateNameSet          string         //需要删除文件数
	ExifErrCnt               int            //exif错误数
	IsComplete               int            //是否完整
	Remark                   string         //备注
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

// 扫描并将结果写入数据库
func ScanAndSave(scanArgs model.DoScanImgArg) (string, error) {

	imgRecordString, err := DoScan(scanArgs)

	if err != nil {
		tools.Logger.Error("scan result error : ", err)
		return "", err
	}

	var imgRecord ImgRecord
	json.Unmarshal([]byte(imgRecordString), &imgRecord)
	var imgRecordDB model.ImgRecordDB
	json.Unmarshal([]byte(imgRecordString), &imgRecordDB)

	imgRecordDB.SuffixMap = tools.MarshalJsonToString(imgRecord.SuffixMap)
	imgRecordDB.SuffixMapBak = tools.MarshalJsonToString(imgRecord.SuffixMapBak)
	imgRecordDB.YearMap = tools.MarshalJsonToString(imgRecord.YearMap)
	imgRecordDB.YearMapBak = tools.MarshalJsonToString(imgRecord.YearMapBak)
	//imgRecordDB.DumpFileDeleteList = tools.MarshalJsonToString(imgRecord.DumpFileDeleteList)

	if err = imgRecordService.CreateImgRecord(&imgRecordDB); err != nil {
		tools.Logger.Error("create error : ", err)
		return "", err
	} else {
		tools.Logger.Info("写入数据库成功")
	}
	return imgRecordString, nil
}

// 扫描主体程序
func DoScan(scanArgs model.DoScanImgArg) (string, error) {

	deleteDirList = []dirStruct{}                //需要处理的目录结构体列表（空目录）
	processFileList = []photoStruct{}            //需要处理的文件结构体列表（非法格式删除、移动、修改时间、重复文件删除）
	md5DumpMap = make(map[string][]string)       //以md5为key存储的重复文件
	getExifInfoErrorSuffixMap = map[string]int{} //exif获取出问题的后缀统计
	getExifInfoErrorSet = mapset.NewSet()        //exif获取出问题的照片集合统计
	shouldDeleteMd5Files = []string{}            //统计需要删除的文件
	fileDateFileList = mapset.NewSet()           //文件名带日期的照片
	deleteFileList = mapset.NewSet()             //需要删除的文件
	moveFileList = mapset.NewSet()               //目录与最小日期不匹配，需要移动
	renameFileList = mapset.NewSet()             //文件名不一致，需要改名
	modifyDateFileList = mapset.NewSet()         //修改时间与最小日期不匹配，需要修改
	shootDateMismatchFileList = mapset.NewSet()  //拍摄时间与最小日期不匹配，需要修改
	shootDateNullFileList = mapset.NewSet()      //拍摄时间没有
	shootDateEarlierFileList = mapset.NewSet()   //拍摄时间与最小日期不匹配（拍摄时间小），需要修改
	imgDatabaseDBList = []*model.ImgDatabaseDB{} //img_database 待插入list

	var bakNewFile []string    //主备目录对比后，主目录新增文件
	var bakDeleteFile []string //主备目录对比后，主目录删除文件

	var suffixMap = map[string]int{}    //后缀统计
	var suffixMapBak = map[string]int{} //后缀统计（备份目录）
	var yearMap = map[string]int{}      //年份统计
	var yearMapBak = map[string]int{}   //年份统计（备份目录）
	var monthMap = map[string]int{}     //月份统计
	var monthMapBak = map[string]int{}  //月份统计（备份目录）
	var dayMap = map[string]int{}       //日期统计
	var dayMapBak = map[string]int{}    //日期统计（备份目录）

	var imageNumMap = map[string][]string{}    //照片名数字顺序统计-照片key
	var imageNumRevMap = map[string][]string{} //照片名数字顺序统计-月份key

	var diffMap = map[string]int{} //统计主备目录一致性的map记录   0表示备库里没有这个文件；2表示主库里没有这个文件；1表示两边都有

	var fileTotalCnt = 0    //文件总量
	var dirTotalCnt = 0     //目录总量
	var fileTotalCntBak = 0 //文件总量（备份目录）
	var dirTotalCntBak = 0  //目录总量（备份目录）

	start1 := time.Now() // 获取当前时间

	if cons.TruncateTable { //清理img cache表
		err := imgDatabaseService.TruncateImgDatabase()
		if err != nil {
			panic("TruncateImgDatabase ERROR ! ")
		} else {
			tools.Logger.Info("TruncateImgDatabase success!")
		}
	}
	if cons.ImgCache { //判断是否需要构建cache
		middleware.CreateImgCache()
	}
	middleware.CreateGisDatabaseCache() //构建gis cache

	elapsed1 := time.Since(start1)
	start2 := time.Now() // 获取当前时间

	if scanArgs.StartPath == nil || *scanArgs.StartPath == "" {
		scanArgs.StartPath = &cons.StartPath
	}
	if scanArgs.StartPathBak == nil || *scanArgs.StartPathBak == "" {
		scanArgs.StartPathBak = &cons.StartPathBak
	}
	if scanArgs.DeleteShow == nil {
		scanArgs.DeleteShow = &cons.DeleteShow
	}
	if scanArgs.MoveFileShow == nil {
		scanArgs.MoveFileShow = &cons.MoveFileShow
	}
	if scanArgs.ModifyDateShow == nil {
		scanArgs.ModifyDateShow = &cons.ModifyDateShow
	}
	if scanArgs.RenameFileShow == nil {
		scanArgs.RenameFileShow = &cons.RenameFileShow
	}
	if scanArgs.Md5Show == nil {
		scanArgs.Md5Show = &cons.Md5Show
	}
	if scanArgs.DeleteAction == nil {
		scanArgs.DeleteAction = &cons.DeleteAction
	}
	if scanArgs.MoveFileAction == nil {
		scanArgs.MoveFileAction = &cons.MoveFileAction
	}
	if scanArgs.ModifyDateAction == nil {
		scanArgs.ModifyDateAction = &cons.ModifyDateAction
	}
	if scanArgs.RenameFileAction == nil {
		scanArgs.RenameFileAction = &cons.RenameFileAction
	}

	startPath = *scanArgs.StartPath
	startPathBak = *scanArgs.StartPathBak
	deleteShow = *scanArgs.DeleteShow
	moveFileShow = *scanArgs.MoveFileShow
	modifyDateShow = *scanArgs.ModifyDateShow
	renameFileShow = *scanArgs.RenameFileShow
	md5Show = *scanArgs.Md5Show
	deleteAction = *scanArgs.DeleteAction
	moveFileAction = *scanArgs.MoveFileAction
	modifyDateAction = *scanArgs.ModifyDateAction
	renameFileAction = *scanArgs.RenameFileAction

	tools.Logger.Info("DoScan args final: ")
	tools.Logger.Info("startPath : ", startPath)
	tools.Logger.Info("startPathBak : ", startPathBak)
	tools.Logger.Info("deleteShow : ", deleteShow)
	tools.Logger.Info("moveFileShow : ", moveFileShow)
	tools.Logger.Info("modifyDateShow : ", modifyDateShow)
	tools.Logger.Info("renameFileShow : ", renameFileShow)
	tools.Logger.Info("md5Show : ", md5Show)
	tools.Logger.Info("deleteAction : ", deleteAction)
	tools.Logger.Info("moveFileAction : ", moveFileAction)
	tools.Logger.Info("modifyDateAction : ", modifyDateAction)
	tools.Logger.Info("renameFileAction : ", renameFileAction)

	scanUuid, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	timeStr := time.Now().Format(tools.DatetimeDirTemplate)
	scanUuidFinal = timeStr + "_" + strings.ReplaceAll(scanUuid.String(), "-", "")
	tools.Logger.Info("SCAN JOBID : ", tools.StrWithColor(scanUuidFinal, "red"))

	if !strings.Contains(startPath, "pic-new") {
		return "", errors.New("startPath error ")
	}

	basePath = startPath[0 : strings.Index(startPath, "pic-new")+7] //指向pic-new的目录

	defer tools.Logger.Sync()

	tools.Logger.Info()
	tools.Logger.Info("————————————————————————————————————————————————————————")
	tools.Logger.Info("time : ", start1.Format(tools.DatetimeTemplate))
	tools.Logger.Info("startPath : ", startPath)
	tools.Logger.Info("basePath : ", basePath)
	tools.Logger.Info("startPathBak : ", startPathBak)

	tools.Logger.Info()

	tools.Logger.Info(tools.StrWithColor("==========ROUND 1: SCAN FILE==========", "red"))
	tools.Logger.Info()

	p, _ := ants.NewPool(cons.PoolSize) //新建一个pool对象
	defer p.Release()

	// 计时器
	//ticker := time.NewTicker(time.Second * 2)
	ticker2 := time.NewTicker(time.Minute * 1)
	tickerSize2 := 0
	go func() {
		for t := range ticker2.C {
			tools.Logger.Info(tools.StrWithColor("Tick at "+t.Format(tools.DatetimeTemplate), "red") + tools.StrWithColor(" , tick range processed "+strconv.Itoa(fileTotalCnt-tickerSize2), "red"))
			tickerSize2 = fileTotalCnt
		}
	}()

	// Optionally register camera makenote data parsing - currently Nikon and Canon are supported.
	//exif.RegisterParsers(mknote.All...)

	IsComplete := 1
	_ = filepath.Walk(startPath, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			tools.Logger.Error("WALK ERROR : ", err)
			panic("WALK ERROR ! ")
			IsComplete = 0
			return err
		}
		if info.IsDir() { //遍历目录
			if flag, err := tools.IsEmpty(file); err == nil && flag { //空目录加入待处理列表
				ds := dirStruct{isEmptyDir: true, dir: file}
				deleteDirList = append(deleteDirList, ds)

			}
			dirTotalCnt = dirTotalCnt + 1
		} else { //遍历文件
			//tools.Logger.Info(file)
			fileName := filepath.Base(file)
			fileSuffix := strings.ToLower(path.Ext(file))

			if strings.HasSuffix(fileName, "_.pic.jpg") || strings.HasPrefix(fileName, ".") || strings.HasPrefix(fileName, "IMG_E") || strings.HasSuffix(fileName, "nas_downloading") || tools.GetFileSize(file) == 0 { //非法文件加入待处理列表
				ps := photoStruct{isDeleteFile: true, photo: file}
				processFileListMu.Lock()
				processFileList = append(processFileList, ps)
				processFileListMu.Unlock()
				deleteFileList.Add(file)

			} else {

				parentDir := path.Base(filepath.Dir(file))
				dumpCompareKey := parentDir + "|" + fileName

				if value, ok := suffixMap[fileSuffix]; ok { //统计文件的后缀
					suffixMap[fileSuffix] = value + 1
				} else {
					suffixMap[fileSuffix] = 1
				}

				day := tools.GetDirDate(file)
				year := day[0:4]
				month := day[0:7]

				if value, ok := yearMap[year]; ok { //统计照片年份
					yearMap[year] = value + 1
				} else {
					yearMap[year] = 1
				}

				if value, ok := monthMap[month]; ok { //统计照片年份
					monthMap[month] = value + 1
				} else {
					monthMap[month] = 1
				}

				if value, ok := dayMap[day]; ok { //统计照片年份
					dayMap[day] = value + 1
				} else {
					dayMap[day] = 1
				}

				if strings.HasPrefix(fileName, "IMG_") {
					head := fileName[4:5]
					if value, ok := imageNumMap[fileName]; ok { //返回值ok表示是否存在这个值
						imageNumMap[fileName] = append(value, day)
					} else {
						imageNumMap[fileName] = []string{day}
					}

					if value, ok := imageNumRevMap[year+"-"+head]; ok { //返回值ok表示是否存在这个值
						imageNumRevMap[year+"-"+head] = append(value, fileName+","+day+"")
					} else {
						imageNumRevMap[year+"-"+head] = []string{fileName + "," + day + ""}
					}
				}

				fileTotalCnt = fileTotalCnt + 1
				diffMap[dumpCompareKey] = 0
				if fileTotalCnt%1000 == 0 { //每隔1000行打印一次
					tools.Logger.Info("processed ", tools.StrWithColor(strconv.Itoa(fileTotalCnt), "red"))
					tools.Logger.Info("pool running size : ", p.Running())
				}

				wg.Add(1)

				_ = p.Submit(func() {
					processOneFile(file) //单个文件协程处理
				})

			}
		}
		return nil
	})

	tools.Logger.Info("processed(end) ", tools.StrWithColor(strconv.Itoa(fileTotalCnt), "red"))

	wg.Wait()

	tools.Logger.Info(tools.StrWithColor("Tick at "+time.Now().Format(tools.DatetimeTemplate), "red") + tools.StrWithColor(" , tick range processed "+strconv.Itoa(fileTotalCnt-tickerSize2), "red"))

	ticker2.Stop() //计时终止

	elapsed2 := time.Since(start2)
	start3 := time.Now() // 获取当前时间

	if cons.ImgCache { //指定使用cache时，才更新库
		if err = imgDatabaseService.CreateImgDatabaseBatch(imgDatabaseDBList); err != nil {
			tools.Logger.Error("CreateImgDatabase error : ", err)
		}
	}

	elapsed3 := time.Since(start3)
	start4 := time.Now() // 获取当前时间
	var basePathBak = ""

	if cons.BakStatEnable {
		if startPathBak == "" || !strings.Contains(startPathBak, "pic-new") {
			return "", errors.New("StartPathBak error ")
		}
		basePathBak = startPathBak[0 : strings.Index(startPathBak, "pic-new")+7] //指向pic-new的目录
		tools.Logger.Info("basePathBak : ", basePathBak)

		ticker := time.NewTicker(time.Minute * 1)
		tickerSize := 0
		go func() {
			for t := range ticker.C {
				tools.Logger.Info(tools.StrWithColor("Tick at "+t.Format(tools.DatetimeTemplate), "red") + tools.StrWithColor(" , tick range processed "+strconv.Itoa(fileTotalCntBak-tickerSize), "red"))
				tickerSize = fileTotalCntBak
			}
		}()

		_ = filepath.Walk(startPathBak, func(file string, info os.FileInfo, err error) error {
			if err != nil {
				tools.Logger.Error("startPathBak WALK ERROR : ", err)
				panic("startPathBak WALK ERROR ! ")
				return err
			}
			if info.IsDir() { //遍历目录
				dirTotalCntBak = dirTotalCntBak + 1

			} else { //遍历文件
				fileName := filepath.Base(file)
				fileSuffix := strings.ToLower(path.Ext(file))

				if strings.HasPrefix(fileName, ".") || strings.HasPrefix(fileName, "IMG_E") || strings.HasSuffix(fileName, "nas_downloading") || tools.GetFileSize(file) == 0 { //非法文件加入待处理列表

				} else {

					parentDir := path.Base(filepath.Dir(file))
					dumpCompareKey := parentDir + "|" + fileName

					if value, ok := suffixMapBak[fileSuffix]; ok { //统计文件的后缀
						suffixMapBak[fileSuffix] = value + 1
					} else {
						suffixMapBak[fileSuffix] = 1
					}

					day := tools.GetDirDate(file)
					year := day[0:4]
					month := day[0:7]

					if value, ok := yearMapBak[year]; ok { //统计照片年份
						yearMapBak[year] = value + 1
					} else {
						yearMapBak[year] = 1
					}

					if value, ok := monthMapBak[month]; ok { //统计照片年份
						monthMapBak[month] = value + 1
					} else {
						monthMapBak[month] = 1
					}

					if value, ok := dayMapBak[day]; ok { //统计照片年份
						dayMapBak[day] = value + 1
					} else {
						dayMapBak[day] = 1
					}

					fileTotalCntBak = fileTotalCntBak + 1
					if _, ok := diffMap[dumpCompareKey]; ok {
						diffMap[dumpCompareKey] = 1
					} else {
						diffMap[dumpCompareKey] = 2
					}

					if fileTotalCntBak%1000 == 0 { //每隔1000行打印一次
						tools.Logger.Info("bak0-dir processed ", tools.StrWithColor(strconv.Itoa(fileTotalCntBak), "red"))
						tools.Logger.Info("pool running size : ", p.Running())
					}
				}

			}
			return nil
		})
		tools.Logger.Info("bak0-dir processed(end) ", tools.StrWithColor(strconv.Itoa(fileTotalCntBak), "red"))

		tools.Logger.Info(tools.StrWithColor("Tick at "+time.Now().Format(tools.DatetimeTemplate), "red") + tools.StrWithColor(" , tick range processed "+strconv.Itoa(fileTotalCntBak-tickerSize), "red"))

		ticker.Stop() //计时终止
	}

	elapsed4 := time.Since(start4)
	start5 := time.Now() // 获取当前时间

	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("==========ROUND 2: PROCESS FILE==========", "red"))
	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("PRINT DETAIL TYPE1(delete file,modify date,move file): ", "red"))

	processFileProcess() //待处理文件处理
	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("PRINT DETAIL TYPE2(empty dir): ", "red"))
	emptyDirProcess() //空目录处理
	tools.Logger.Info()

	tools.Logger.Info(tools.StrWithColor("PRINT DETAIL TYPE3(dump file): ", "red"))
	dumpMap := dumpFileProcess() //5、重复文件处理处理

	tools.Logger.Info(tools.StrWithColor("PRINT STAT TYPE0(comman info): ", "red"))

	tools.Logger.Info("suffixMap（后缀统计） : ", tools.MarshalJsonToString(suffixMap))

	tools.Logger.Info("yearMap（年份统计） : ", tools.MarshalJsonToString(yearMap))
	tools.Logger.Info("month count（月份统计） : ")
	tools.MapPrintWithFilter(monthMap, monthFilter)
	tools.Logger.Info("day count（日期统计） : ")
	tools.MapPrintWithFilter(dayMap, dayFilter)

	tools.Logger.Info("file total（总文件数） : ", tools.StrWithColor(strconv.Itoa(fileTotalCnt), "red"))
	tools.Logger.Info("dir total（总目录数） : ", tools.StrWithColor(strconv.Itoa(dirTotalCnt), "red"))
	tools.Logger.Info("file contain date(just for print)（照片名称带日志的数量） : ", tools.StrWithColor(strconv.Itoa(fileDateFileList.Cardinality()), "red"))
	tools.Logger.Info("exif parse error（exif解析出错的后缀汇总） : ", tools.StrWithColor(tools.MarshalJsonToString(getExifInfoErrorSuffixMap), "red"))
	tools.Logger.Info("ExifNameSet list（exif统计的所有日期key打印） : ", middleware.ExifDateNameSet.String())

	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("PRINT STAT TYPE1: ", "red"))
	pr := "delete file total（删除文件统计） : " + tools.StrWithColor(strconv.Itoa(deleteFileList.Cardinality()), "red")
	if deleteFileList.Cardinality() > 0 && deleteAction {
		pr = pr + tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)
	pr = "modify date total（没有shootdate且修改日期不对文件统计） : " + tools.StrWithColor(strconv.Itoa(modifyDateFileList.Cardinality()), "red")
	if modifyDateFileList.Cardinality() > 0 && modifyDateAction {
		pr = pr + tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)
	pr = "move file total（移动文件统计） : " + tools.StrWithColor(strconv.Itoa(moveFileList.Cardinality()), "red")
	if moveFileList.Cardinality() > 0 && moveFileAction {
		pr = pr + tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)
	pr = "rename file total（改名文件统计） : " + tools.StrWithColor(strconv.Itoa(renameFileList.Cardinality()), "red")
	if renameFileList.Cardinality() > 0 && renameFileAction {
		pr = pr + tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)
	tools.Logger.Info("shoot date total（拍摄日期跟目录不一致统计） : ", tools.StrWithColor(strconv.Itoa(shootDateMismatchFileList.Cardinality()), "red"))
	tools.Logger.Info("shoot date total（拍摄日期没有统计） : ", tools.StrWithColor(strconv.Itoa(shootDateNullFileList.Cardinality()), "red"))
	tools.Logger.Info("shoot date total（拍摄日期跟目录不一致统计，且拍摄日期更小） : ", tools.StrWithColor(strconv.Itoa(shootDateEarlierFileList.Cardinality()), "red"))

	tools.Logger.Info()
	pr = "empty dir total（空目录总数） : " + tools.StrWithColor(strconv.Itoa(len(deleteDirList)), "red")
	if len(deleteDirList) > 0 && deleteAction {
		pr = pr + tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)

	tools.Logger.Info()
	tools.Logger.Info("dump file total（重复文件组数量） : ", tools.StrWithColor(strconv.Itoa(len(dumpMap)), "red"))
	if len(dumpMap) != 0 { //重复数据写文件
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
		filePath := cons.WorkDir + "/log/dump_delete_file/" + scanUuidFinal
		os.MkdirAll(filePath, os.ModePerm)
		tools.WriteStringToFile(builder.String(), filePath+"/dump_compare")
	}

	tools.Logger.Info("shouldDeleteMd5Files length（重复文件应该删除的数量） : ", tools.StrWithColor(strconv.Itoa(len(shouldDeleteMd5Files)), "red"))
	if len(shouldDeleteMd5Files) != 0 { //待删除写文件
		//sm3 := tools.MarshalJsonToString(shouldDeleteMd5Files)
		sm3 := strings.Join(shouldDeleteMd5Files, "\n")
		filePath := cons.WorkDir + "/log/dump_delete_file/" + scanUuidFinal + "/dump_delete_list"
		tools.WriteStringToFile(sm3, filePath)
	}

	tools.Logger.Info("imageNumMap length（照片名数字顺序统计-照片key） : ", tools.StrWithColor(strconv.Itoa(len(imageNumMap)), "red"))
	if len(imageNumMap) != 0 {
		filePath := cons.WorkDir + "/log/img_num_list"
		tools.ImageNumMapWriteToFile(imageNumMap, filePath)
	}
	tools.Logger.Info("imageNumRevMap length（照片名数字顺序统计-月份key） : ", tools.StrWithColor(strconv.Itoa(len(imageNumRevMap)), "red"))
	if len(imageNumRevMap) != 0 {
		filePath := cons.WorkDir + "/log/img_num_rev_list"
		tools.ImageNumRevMapWriteToFile(imageNumRevMap, filePath)
	}

	if cons.BakStatEnable { //对比主目录和备份目录
		for imgKey, flag := range diffMap {
			if flag == 0 { //为0表示备库里没有这个文件
				bakNewFile = append(bakNewFile, imgKey)
			}
			if flag == 2 { //为2表示主库里没有这个文件
				bakDeleteFile = append(bakDeleteFile, imgKey)
			}

		}
		tools.Logger.Info("bakNewFile(新增文件待备份) : ", tools.MarshalJsonToString(bakNewFile))
		tools.Logger.Info("bakDeleteFile(备份里删除文件) : ", tools.MarshalJsonToString(bakDeleteFile))
	}
	tools.Logger.Info("img_database需要新插入的数量: ", len(imgDatabaseDBList))
	tools.Logger.Info("img_database没有匹配上key，应该删除的数量: ", len(middleware.ImgCacheMapBak))

	if cons.SyncTable && len(middleware.ImgCacheMapBak) != 0 { //批量删除多余的img_database
		tools.Logger.Info("正在批量删除多余的img_database。。。 ")
		var imgKeyToDelete []string
		for key, _ := range middleware.ImgCacheMapBak {
			imgKeyToDelete = append(imgKeyToDelete, key)
			if len(imgKeyToDelete) >= cons.IDDeleteBatchSize {
				imgDatabaseService.DeleteImgDatabaseByImgKey(imgKeyToDelete)
				imgKeyToDelete = []string{}
			}
		}
		imgDatabaseService.DeleteImgDatabaseByImgKey(imgKeyToDelete)
	}
	var imgDatabaseSearch model.ImgDatabaseSearch
	imgDatabaseTotal, _ := imgDatabaseService.GetImgDatabaseInfoCount(imgDatabaseSearch)
	tools.Logger.Info("文件总数 : ", fileTotalCnt, " , imgDatabase总数 : ", imgDatabaseTotal)

	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("==========ROUND 3: PROCESS COST==========", "red"))
	tools.Logger.Info()
	elapsed5 := time.Since(start5)
	tools.Logger.Info("加载缓存完成耗时 : ", elapsed1)
	tools.Logger.Info("执行主目录扫描完成耗时 : ", elapsed2)
	tools.Logger.Info("执行img_database批量写入完成耗时 : ", elapsed3)
	tools.Logger.Info("执行备目录扫描完成耗时 : ", elapsed4)
	tools.Logger.Info("执行数据处理完成耗时 : ", elapsed5)
	tools.Logger.Info()

	imgRecord := ImgRecord{}
	imgRecord.FileTotal = fileTotalCnt
	imgRecord.FileTotalBak = fileTotalCntBak
	imgRecord.DirTotal = dirTotalCnt
	imgRecord.DirTotalBak = dirTotalCntBak
	imgRecord.StartDate = start1
	imgRecord.UseTime = int(math.Ceil(elapsed1.Seconds() + elapsed2.Seconds() + elapsed3.Seconds() + elapsed4.Seconds() + elapsed5.Seconds()))
	imgRecord.BasePath = basePath
	imgRecord.BasePathBak = basePathBak
	imgRecord.BakNewFileCnt = len(bakNewFile)
	imgRecord.BakDeleteFileCnt = len(bakDeleteFile)
	imgRecord.BakNewFile = tools.MarshalJsonToString(bakNewFile)
	imgRecord.BakDeleteFile = tools.MarshalJsonToString(bakDeleteFile)
	imgRecord.SuffixMap = suffixMap
	imgRecord.SuffixMapBak = suffixMapBak
	imgRecord.YearMap = yearMap
	imgRecord.YearMapBak = yearMapBak
	imgRecord.FileDateCnt = fileDateFileList.Cardinality()
	imgRecord.DeleteFileCnt = deleteFileList.Cardinality()
	imgRecord.ModifyDateFileCnt = modifyDateFileList.Cardinality()
	imgRecord.MoveFileCnt = moveFileList.Cardinality()
	imgRecord.RenameFileCnt = renameFileList.Cardinality()
	imgRecord.ShootDateMismatchFileCnt = shootDateMismatchFileList.Cardinality()
	imgRecord.ShootDateNullFileCnt = shootDateNullFileList.Cardinality()
	imgRecord.ShootDateEarlierFileCnt = shootDateEarlierFileList.Cardinality()
	imgRecord.EmptyDirCnt = len(deleteDirList)
	imgRecord.DumpFileCnt = len(dumpMap)
	imgRecord.ExifDateNameSet = middleware.ExifDateNameSet.String()
	imgRecord.ExifErrCnt = getExifInfoErrorSet.Cardinality()
	imgRecord.ScanArgs = tools.MarshalJsonToString(scanArgs)
	imgRecord.IsComplete = IsComplete
	imgRecord.Remark = ""

	ret := tools.MarshalJsonToString(imgRecord)
	tools.Logger.Info("scan result : ", ret)
	return ret, nil

}

// 主目录遍历完成后，待处理文件处理
func processFileProcess() {
	for _, ps := range processFileList { //第一个参数是下标

		printFileFlag := false
		printDateFlag := false

		if ps.isDeleteFile {
			deleteFileProcess(ps, &printFileFlag) //1、需要删除的文件处理
		}
		if ps.isModifyDateFile {
			modifyDateProcess(ps, &printFileFlag, &printDateFlag) //2、需要修改时间的文件处理
		}
		if ps.isMoveFile {
			moveFileProcess(ps, &printFileFlag, &printDateFlag) //3、需要移动的文件处理
		}
		if ps.isRenameFile {
			renameFileProcess(ps, &printFileFlag, &printDateFlag) //4、需要改名的文件处理
		}

	}
}

// 待删除文件处理逻辑
func deleteFileProcess(ps photoStruct, printFileFlag *bool) {
	if deleteShow || deleteAction {
		tools.Logger.Info()
		tools.Logger.Info("file : ", tools.StrWithColor(ps.photo, "blue"))
		*printFileFlag = true
		tools.Logger.Info(tools.StrWithColor("should delete file :", "yellow"), ps.photo, " SIZE: ", tools.GetFileSize(ps.photo))
	}

	if deleteAction {
		err := os.Remove(ps.photo)
		if err != nil {
			tools.Logger.Info(tools.StrWithColor("delete file failed:", "yellow"), ps.photo, err)
		} else {
			tools.Logger.Info(tools.StrWithColor("delete file sucessed:", "green"), ps.photo)
		}
	}
}

// 待更新修改日期文件处理逻辑
func modifyDateProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool) {
	if modifyDateShow || modifyDateAction {
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
	if modifyDateAction { //修改日期（不启用）
		localLoc, _ := time.LoadLocation("Asia/Shanghai") // 本地时区设置为上海
		tm, _ := time.ParseInLocation("2006-01-02 15:04:05", ps.minDate+" 12:00:00", localLoc)
		tools.ChangeModifyDate(ps.photo, tm)
		//shootDate := strings.ReplaceAll(ps.minDate, "-", ":") + " 12:00:00"
		//middleware.ModifyShootDate(ps.photo, shootDate)
		tools.Logger.Info(tools.StrWithColor("modify file ", "yellow"), ps.photo, "modifyDate to", ps.minDate, "get realdate", tools.GetModifyDate(ps.photo))
	}
}

// 待移动文件处理逻辑
func moveFileProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool) {
	if moveFileShow || moveFileAction {
		if !*printFileFlag {
			tools.Logger.Info()
			tools.Logger.Info("file : ", tools.StrWithColor(ps.photo, "blue"))
			*printFileFlag = true
		}
		if !*printDateFlag {
			ps.psDatePrint()
			*printDateFlag = true
		}
		tools.Logger.Info(tools.StrWithColor("should move file ", "yellow"), ps.photo, " to ", ps.targetPhoto)
	}
	if moveFileAction {
		tools.MoveFile(ps.photo, ps.targetPhoto)
		tools.Logger.Info(tools.StrWithColor("move file ", "yellow"), ps.photo, " to ", ps.targetPhoto)
	}
}

// 重命名文件处理逻辑
func renameFileProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool) {
	if renameFileShow || renameFileAction {
		if !*printFileFlag {
			tools.Logger.Info()
			tools.Logger.Info("file : ", tools.StrWithColor(ps.photo, "blue"))
			*printFileFlag = true
		}
		if !*printDateFlag {
			ps.psDatePrint()
			*printDateFlag = true
		}
		tools.Logger.Info(tools.StrWithColor("should rename file ", "yellow"), ps.photo, " to ", ps.targetPhoto)
	}
	if renameFileAction {
		tools.MoveFile(ps.photo, ps.targetPhoto)
		tools.Logger.Info(tools.StrWithColor("rename file ", "yellow"), ps.photo, " to ", ps.targetPhoto)
	}
}

// 空目录处理
func emptyDirProcess() {
	for _, ds := range deleteDirList {
		if ds.isEmptyDir {
			if deleteShow || deleteAction {
				tools.Logger.Info("dir : ", tools.StrWithColor(ds.dir, "blue"))
				tools.Logger.Info(tools.StrWithColor("should delete empty dir :", "yellow"), ds.dir)
			}

			if deleteAction {
				err := os.Remove(ds.dir)
				if err != nil {
					tools.Logger.Info(tools.StrWithColor("delete empty dir failed:", "yellow"), ds.dir, err)
				} else {
					tools.Logger.Info(tools.StrWithColor("delete empty dir sucessed:", "green"), ds.dir)
				}
			}
		}
		tools.Logger.Info()

	}
}

// 重复文件处理
func dumpFileProcess() map[string][]string {
	var dumpMap = make(map[string][]string) //md5Map里筛选出有重复文件的Map

	if md5Show {
		for md5, files := range md5DumpMap {
			if len(files) > 1 {
				dumpMap[md5] = files
				minPhoto := ""
				var fileSizeTemp int64
				sizeMatch := true
				for _, photo := range files {
					if fileSizeTemp == 0 {
						fileSizeTemp = tools.GetFileSize(photo)
					} else {
						if fileSizeTemp != tools.GetFileSize(photo) {
							sizeMatch = false
						}
					}

					if minPhoto == "" {
						minPhoto = photo
					} else {
						if tools.GetDirDate(minPhoto) > tools.GetDirDate(photo) { //比minPhoto小则替换minPhoto
							minPhoto = photo
						} else if tools.GetDirDate(minPhoto) < tools.GetDirDate(photo) { //比minPhoto大则不变

						} else if len(tools.GetParentDir(minPhoto)) < len(tools.GetParentDir(photo)) { //如果目录的日期一样，优先保留增加了目录描述的
							minPhoto = photo
						} else {
							if len(path.Base(minPhoto)) > len(path.Base(photo)) { //留文件名短的
								minPhoto = photo
							}
						}
					}
				}

				/*if !sizeMatch { //如果存在文件大小不一样的情况，则不记录
					continue
				}*/

				tools.Logger.Info("file : ", tools.StrWithColor(md5, "blue"))
				for _, photo := range files {
					if photo != minPhoto {
						if sizeMatch {
							shouldDeleteMd5Files = append(shouldDeleteMd5Files, photo)
							tools.Logger.Info("choose : ", photo, tools.StrWithColor(" DELETE", "red"), " SIZE: ", tools.GetFileSize(photo))
						} else {
							tools.Logger.Info("choose : ", photo, tools.StrWithColor(" SAVE(SIZE MISMATCH)", "green"), " SIZE: ", tools.GetFileSize(photo))
						}

					} else {

						if sizeMatch {
							tools.Logger.Info("choose : ", photo, tools.StrWithColor(" SAVE", "green"), " SIZE: ", tools.GetFileSize(photo))
						} else {
							tools.Logger.Info("choose : ", photo, tools.StrWithColor(" SAVE(SIZE MISMATCH)", "green"), " SIZE: ", tools.GetFileSize(photo))
						}

					}

				}
				tools.Logger.Info()

			}
		}

	}
	return dumpMap
}

// 遍历逻辑单文件处理
func processOneFile(photo string) {

	defer wg.Done()

	var shootDateOrigin string
	var shootDate string
	var locStreet string

	/*if strings.Contains(photo, "SIMG_1779") {
		tools.Logger.Info()
	}*/

	shootDateOrigin, locStreet, _ = getImgShootDateAndLoc(photo) //查询照片的拍摄时间，gis信息处理

	if shootDateOrigin != "" {
		t, err := time.Parse("2006:01:02 15:04:05", shootDateOrigin)
		if err == nil {
			shootDate = t.Format("2006-01-02")
		}
	}

	dirDate := tools.GetDirDate(photo)

	fileDate := tools.GetFileDate(photo)
	if fileDate != "" {
		fileDateFileList.Add(photo)
	}

	modifyDate := tools.GetModifyDate(photo)

	minDate := ""

	if dirDate < modifyDate {
		minDate = dirDate
	} else {
		minDate = modifyDate
	}

	if shootDate != "" && shootDate < minDate {
		minDate = shootDate
	}
	/*if fileDate != "" {
		minDate = fileDate
	}*/

	ps := photoStruct{photo: photo, dirDate: dirDate, modifyDate: modifyDate, shootDate: shootDate, fileDate: fileDate, minDate: minDate}
	flag := false

	if dirDate != minDate { //需要移动文件的判断
		moveFileList.Add(photo)
		targetPath := basePath + string(os.PathSeparator) + minDate[0:4] + string(os.PathSeparator) + minDate[0:7] + string(os.PathSeparator) + minDate
		targetPath = tools.GetRealPath(targetPath)
		targetPhoto := targetPath + string(os.PathSeparator) + filepath.Base(photo)
		ps.isMoveFile = true
		ps.targetPhoto = targetPhoto
		flag = true

	}

	//if suffix != ".mov" && suffix != ".mp4" { //exif拍摄时间获取
	if shootDate != "" && shootDate != dirDate {
		shootDateMismatchFileList.Add(photo)
		if shootDate < dirDate {
			shootDateEarlierFileList.Add(photo)
		}
	}
	//}

	if shootDate == "" {
		shootDateNullFileList.Add(photo)
	}

	if shootDate == "" && modifyDate != minDate { //需要修改文件修改时间的判断
		modifyDateFileList.Add(photo)
		ps.isModifyDateFile = true
		flag = true
	}

	/*if strings.Contains(photo, "IMG_5574.JPG") {
		tools.Logger.Info()
	}*/

	targetPhoto := getRenameNewPhoto(photo, shootDateOrigin, locStreet)
	if photo != targetPhoto {
		renameFileList.Add(photo)
		ps.isRenameFile = true
		ps.targetPhoto = targetPhoto
		flag = true
	}

	if md5Show { //如果需要计算md5，则把所有照片按照md5整理
		md5, err := tools.GetFileMD5WithRetry(photo, cons.Md5Retry, cons.Md5CountLength)
		if err != nil {
			tools.Logger.Info("GetFileMD5 err for ", cons.Md5Retry, " times : ", err, " file : ", photo)
		} else {
			md5DumpMapMu.Lock()
			if value, ok := md5DumpMap[md5]; ok { //返回值ok表示是否存在这个值
				md5DumpMap[md5] = append(value, photo)
			} else {
				md5DumpMap[md5] = []string{photo}
			}
			md5DumpMapMu.Unlock()
		}
	}

	if flag { //根据分类统计的结果，判断是否需要放入待处理列表里
		processFileListMu.Lock()
		processFileList = append(processFileList, ps)
		processFileListMu.Unlock()
	}

}

func getRenameNewPhoto(photo string, shootDate string, locStreet string) string {
	photoNew := photo

	if shootDate != "" {
		t, err := time.Parse("2006:01:02 15:04:05", shootDate)
		if err == nil {
			shootDate = t.Format("2006-01-02_15-04-05")
		}
	}
	if shootDate != "" || locStreet != "" {
		fileRegexp := regexp.MustCompile(`^.*\[(.*)\].*$`)
		dateValList := fileRegexp.FindStringSubmatch(photo)
		var timeAndLocFile string
		var timeAndLocShould string
		if len(dateValList) == 2 {
			timeAndLocFile = dateValList[1]
		}

		timeAndLocShould = shootDate + "^" + locStreet
		dirDate := tools.GetDirDate(photo)
		if !strings.Contains(timeAndLocShould, dirDate) {
			timeAndLocShould = "inconsistent^" + timeAndLocShould
		}

		if timeAndLocFile == timeAndLocShould {
			//tools.Logger.Info("timeAndLoc match")
		} else {
			//tools.Logger.Info("timeAndLoc not match")
			if strings.Count(photo, "[") == 1 && strings.Count(photo, "]") == 1 {
				re, _ := regexp.Compile(`\[.*\]`)
				photoNew = re.ReplaceAllString(photo, "["+timeAndLocShould+"]")
			} else if strings.Count(photo, "[") == 0 && strings.Count(photo, "]") == 0 {
				if strings.Count(photo, ".") == 1 {
					photoNew = strings.ReplaceAll(photo, ".", "["+timeAndLocShould+"].")
				} else {
					fileName := filepath.Base(photo)
					fileSuffix := strings.ToLower(path.Ext(photo))                                                                //文件后缀
					nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(photo))                                           // 去除文件扩展名
					parentDir := filepath.Dir(photo)                                                                              // 获取文件的父目录
					photoNew = parentDir + string(filepath.Separator) + strings.ReplaceAll(nameWithoutExt, ".", "_") + fileSuffix //去除文件名里的.
					photoNew = strings.ReplaceAll(photoNew, ".", "["+timeAndLocShould+"].")

					tools.Logger.Info("##################filePath with . , photo : ", photo, " photoNew : ", photoNew)
				}
			} else {
				tools.Logger.Error("##################filePath [] error , photo : ", photo)
			}

		}

		//tools.Logger.Info("filePath change before : ", photo)
		//tools.Logger.Info("filePath change  after : ", photoNew)

	}
	return photoNew
}

// 获取文件的拍摄时间
func getImgShootDateAndLoc(photo string) (string, string, error) {

	suffix := strings.ToLower(path.Ext(photo))

	fileName := filepath.Base(photo)
	dirDate := tools.GetDirDate(photo)
	imgKey := dirDate + "|" + fileName
	shootDate := ""
	locStreet := ""

	if value, ok := middleware.ImgCacheMap[imgKey]; ok {
		shootDate = value.ShootDate
		locStreet = value.LocStreet
		shootDateCacheMapBakMu.Lock()
		delete(middleware.ImgCacheMapBak, imgKey) //查完后删除，方便最后统计没用到的key删除
		shootDateCacheMapBakMu.Unlock()
	} else {
		var locNum string
		var output string
		var err error
		var state int
		shootDate, locNum, state, output, err = middleware.GetExifInfo(photo)

		var imgDatabaseDB model.ImgDatabaseDB
		imgDatabaseDB.ImgKey = imgKey

		imgDatabaseDB.State = &state
		if err != nil {
			getExifInfoErrorSuffixMapMu.Lock()
			if value, ok := getExifInfoErrorSuffixMap[suffix]; ok {
				getExifInfoErrorSuffixMap[suffix] = value + 1
			} else {
				getExifInfoErrorSuffixMap[suffix] = 1
			}
			getExifInfoErrorSet.Add(photo)
			getExifInfoErrorSuffixMapMu.Unlock()
		}
		imgDatabaseDB.ShootDate = shootDate
		imgDatabaseDB.LocNum = locNum
		imgDatabaseDB.Remark = output
		if locNum != "" {
			gisData, err := middleware.GetLocationAddressByCache(locNum)
			if err == nil {
				imgDatabaseDB.LocAddr = gisData.LocAddr
				imgDatabaseDB.LocStreet = gisData.LocStreet
				locStreet = gisData.LocStreet
			}
		}
		if cons.ImgCache {
			imgDatabaseDBList = append(imgDatabaseDBList, &imgDatabaseDB)
		}

	}

	return shootDate, locStreet, nil

}
