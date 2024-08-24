package service

import (
	"encoding/json"
	"errors"
	mapset "github.com/deckarep/golang-set"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"img_process/cons"
	"img_process/dao"
	"img_process/model"
	"img_process/tools"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const monthFilter = "xx" //月份过滤
const dayFilter = "xx"   //日期过滤

var processFileListMu sync.Mutex
var md5MapMu sync.Mutex
var exifErr1FileMu sync.Mutex
var exifErr2FileMu sync.Mutex
var exifErr3FileMu sync.Mutex
var md5EmptyFileListMu sync.Mutex

var imgShootDateService = dao.ImgShootDateService{}
var imgRecordService = dao.ImgRecordService{}

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
}

type ImgRecord struct {
	ScanArgs          string         //扫描参数
	FileTotal         int            //文件总数
	FileTotalBak      int            //文件总数
	DirTotal          int            //目录总数
	DirTotalBak       int            //目录总数
	StartDate         time.Time      //记录时间
	UseTime           int            //用时
	BakNewFileCnt     int            //用时
	BakDeleteFileCnt  int            //用时
	BasePath          string         //基础目录
	BasePathBak       string         //基础目录
	BakNewFile        string         //基础目录
	BakDeleteFile     string         //基础目录
	SuffixMap         map[string]int //后缀统计
	SuffixMapBak      map[string]int //后缀统计
	YearMap           map[string]int //年份统计
	YearMapBak        map[string]int //年份统计
	FileDateCnt       int            //有时间文件统计
	DeleteFileCnt     int            //需要删除文件数
	ModifyDateFileCnt int            //需要修改修改日期文件数
	MoveFileCnt       int            //需要移动文件数
	ShootDateFileCnt  int            //需要修改拍摄日期文件数
	EmptyDirCnt       int            //空文件数
	DumpFileCnt       int            //重复md5数
	//DumpFileDeleteList []string       //需要删除文件数
	ExifErr1Cnt int            //exif错误1数
	ExifErr2Cnt int            //exif错误2数
	ExifErr3Cnt int            //exif错误3数
	ExifErr1Map map[string]int //exif错误1统计
	ExifErr2Map map[string]int //exif错误2统计
	ExifErr3Map map[string]int //exif错误3统计
	IsComplete  int            //是否完整
	Remark      string         //备注
}

func (ps *photoStruct) psPrint() { //打印照片相关信息
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

func ScanAndSave(scanArgs model.DoScanImgArg) {

	imgRecordString, err := DoScan(scanArgs)
	if err != nil {
		tools.Logger.Error("scan result error : ", err)
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
	imgRecordDB.ExifErr1Map = tools.MarshalJsonToString(imgRecord.ExifErr1Map)
	imgRecordDB.ExifErr2Map = tools.MarshalJsonToString(imgRecord.ExifErr2Map)
	imgRecordDB.ExifErr3Map = tools.MarshalJsonToString(imgRecord.ExifErr3Map)

	if err = imgRecordService.RegisterImgRecord(&imgRecordDB); err != nil {
		tools.Logger.Error("register error : ", err)
		return
	}

	var imgShootDateDB model.ImgShootDateDB
	if err = imgShootDateService.RegisterImgShootDate(&imgShootDateDB); err != nil {
		tools.Logger.Error("register error : ", err)
		return
	}

	if err = imgRecordService.CreateImgRecord(&imgRecordDB); err != nil {
		tools.Logger.Error("create error : ", err)
		return
	} else {
		tools.Logger.Info("写入数据库成功")
	}
}

func DoScan(scanArgs model.DoScanImgArg) (string, error) {

	start1 := time.Now() // 获取当前时间
	var shootDateCacheMap = map[string]string{}

	if cons.TruncateTable { //清理img cache表，如果清理表则不用构建cache了
		err := imgShootDateService.TruncateImgShootDate()
		if err != nil {
			panic("TruncateImgShootDate ERROR ! ")
		} else {
			tools.Logger.Info("TruncateImgShootDate success!")
		}
	} else {
		if cons.ImgCache { //不清理表且指定需要cache时才构建
			createShootDateCache(&shootDateCacheMap)
		}
	}

	elapsed1 := time.Since(start1)
	start2 := time.Now() // 获取当前时间

	startPath := *scanArgs.StartPath
	startPathBak := *scanArgs.StartPathBak

	/*	if startPathBak == "" {
		startPathBak = startPath
	}*/

	deleteShow := *scanArgs.DeleteShow
	moveFileShow := *scanArgs.MoveFileShow
	modifyDateShow := *scanArgs.ModifyDateShow
	md5Show := *scanArgs.Md5Show
	deleteAction := *scanArgs.DeleteAction
	moveFileAction := *scanArgs.MoveFileAction
	modifyDateAction := *scanArgs.ModifyDateAction

	scanUuid, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	timeStr := time.Now().Format(tools.DatetimeDirTemplate)
	scanUuidFinal := timeStr + "_" + strings.ReplaceAll(scanUuid.String(), "-", "")
	tools.Logger.Info("SCAN JOBID : ", tools.StrWithColor(scanUuidFinal, "red"))

	if !strings.Contains(startPath, "pic-new") {
		return "", errors.New("startPath error ")
	}

	var basePath = startPath[0 : strings.Index(startPath, "pic-new")+7] //指向pic-new的目录

	tools.Logger.Info("DoScan args : ", deleteShow, moveFileShow, modifyDateShow, md5Show, deleteAction, moveFileAction, modifyDateAction)

	var suffixMap = map[string]int{}           //后缀统计
	var suffixMapBak = map[string]int{}        //后缀统计
	var yearMap = map[string]int{}             //年份统计
	var yearMapBak = map[string]int{}          //年份统计
	var monthMap = map[string]int{}            //月份统计
	var monthMapBak = map[string]int{}         //月份统计
	var dayMap = map[string]int{}              //日期统计
	var dayMapBak = map[string]int{}           //日期统计
	var imageNumMap = map[string][]string{}    //照片数字统计-照片key
	var imageNumRevMap = map[string][]string{} //照片数字统计-日期key
	var diffMap = map[string]int{}             //日期统计

	var fileTotalCnt = 0    //文件总量
	var dirTotalCnt = 0     //目录总量
	var fileTotalCntBak = 0 //文件总量
	var dirTotalCntBak = 0  //目录总量

	var fileDateFileList = mapset.NewSet() //文件名带日期的照片

	var deleteFileList = mapset.NewSet()     //需要删除的文件
	var moveFileList = mapset.NewSet()       //目录与最小日期不匹配，需要移动
	var modifyDateFileList = mapset.NewSet() //修改时间与最小日期不匹配，需要修改
	var shootDateFileList = mapset.NewSet()  //拍摄时间与最小日期不匹配，需要修改

	var shouldDeleteMd5Files []string //统计需要删除的文件

	var deleteDirList []dirStruct     //需要处理的目录结构体列表（空目录）
	var processFileList []photoStruct //需要处理的文件结构体列表（非法格式删除、移动、修改时间、重复文件删除）

	var md5Map = make(map[string][]string) //以md5为key存储文件

	var exifErr1FileSuffixMap = map[string]int{} //shoot time error1后缀
	var exifErr1FileSet = mapset.NewSet()        //shoot time error1照片
	var exifErr2FileSuffixMap = map[string]int{} //shoot time error2后缀
	var exifErr2FileSet = mapset.NewSet()        //shoot time error2照片
	var exifErr3FileSuffixMap = map[string]int{} //shoot time error3后缀
	var exifErr3FileSet = mapset.NewSet()        //shoot time error3照片

	var md5EmptyFileList []string //获取md5为空的文件

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
	ticker := time.NewTicker(time.Minute * 1)
	tickerSize := 0
	go func() {
		for t := range ticker.C {
			tools.Logger.Info(tools.StrWithColor("Tick at "+t.Format(tools.DatetimeTemplate), "red") + tools.StrWithColor(" , tick range processed "+strconv.Itoa(fileTotalCnt-tickerSize), "red"))
			tickerSize = fileTotalCnt
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
			fileName := path.Base(file)
			fileSuffix := strings.ToLower(path.Ext(file))

			if strings.HasPrefix(fileName, ".") || strings.HasPrefix(fileName, "IMG_E") || strings.HasSuffix(fileName, "nas_downloading") || *(tools.GetFileSize(file)) == 0 { //非法文件加入待处理列表
				ps := photoStruct{isDeleteFile: true, photo: file}
				processFileListMu.Lock()
				processFileList = append(processFileList, ps)
				processFileListMu.Unlock()
				deleteFileList.Add(file)

			} else {

				dirDate := tools.GetDirDate(file)
				imgKey := dirDate + "|" + fileName

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
						imageNumRevMap[year+"-"+head] = append(value, fileName+"["+day+"]")
					} else {
						imageNumRevMap[year+"-"+head] = []string{fileName + "[" + day + "]"}
					}
				}

				fileTotalCnt = fileTotalCnt + 1
				diffMap[imgKey] = 0
				if fileTotalCnt%1000 == 0 { //每隔1000行打印一次
					tools.Logger.Info("processed ", tools.StrWithColor(strconv.Itoa(fileTotalCnt), "red"))
					tools.Logger.Info("pool running size : ", p.Running())
				}

				wg.Add(1)

				_ = p.Submit(func() {
					processOneFile(
						basePath,
						file,
						md5Show,
						&processFileList,
						fileDateFileList,
						moveFileList,
						modifyDateFileList,
						shootDateFileList,
						md5EmptyFileList,
						md5Map,
						exifErr1FileSuffixMap,
						exifErr1FileSet,
						exifErr2FileSuffixMap,
						exifErr2FileSet,
						exifErr3FileSuffixMap,
						exifErr3FileSet,
						shootDateCacheMap) //单个文件协程处理
				})

			}
		}
		return nil
	})

	tools.Logger.Info("processed(end) ", tools.StrWithColor(strconv.Itoa(fileTotalCnt), "red"))

	wg.Wait()

	tools.Logger.Info(tools.StrWithColor("Tick at "+time.Now().Format(tools.DatetimeTemplate), "red") + tools.StrWithColor(" , tick range processed "+strconv.Itoa(fileTotalCnt-tickerSize), "red"))

	ticker.Stop() //计时终止

	var basePathBak = ""

	if cons.BakStatEnable {
		if startPathBak == "" || !strings.Contains(startPathBak, "pic-new") {
			return "", errors.New("StartPathBak error ")
		}
		basePathBak = startPathBak[0 : strings.Index(startPathBak, "pic-new")+7] //指向pic-new的目录
		tools.Logger.Info("basePathBak : ", basePathBak)

		ticker.Reset(time.Minute * 1)
		tickerSize = 0
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
				fileName := path.Base(file)
				fileSuffix := strings.ToLower(path.Ext(file))

				if strings.HasPrefix(fileName, ".") || strings.HasPrefix(fileName, "IMG_E") || strings.HasSuffix(fileName, "nas_downloading") || *(tools.GetFileSize(file)) == 0 { //非法文件加入待处理列表

				} else {

					dirDate := tools.GetDirDate(file)
					imgKey := dirDate + "|" + fileName

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
					if _, ok := diffMap[imgKey]; ok {
						diffMap[imgKey] = 1
					} else {
						diffMap[imgKey] = 2
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

	elapsed2 := time.Since(start2)
	start3 := time.Now() // 获取当前时间

	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("==========ROUND 2: PROCESS FILE==========", "red"))
	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("PRINT DETAIL TYPE1(delete file,modify date,move file): ", "red"))
	for _, ps := range processFileList { //第一个参数是下标

		printFileFlag := false
		printDateFlag := false

		if ps.isDeleteFile {
			deleteFileProcess(ps, &printFileFlag, &printDateFlag, deleteShow, deleteAction) //1、需要删除的文件处理
		}
		if ps.isModifyDateFile {
			modifyDateProcess(ps, &printFileFlag, &printDateFlag, modifyDateShow, modifyDateAction) //2、需要修改时间的文件处理
		}
		if ps.isMoveFile {
			moveFileProcess(ps, &printFileFlag, &printDateFlag, moveFileShow, moveFileAction) //3、需要移动的文件处理
		}

	}
	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("PRINT DETAIL TYPE2(empty dir): ", "red"))
	emptyDirProcess(deleteShow, deleteAction, deleteDirList) //4、空目录处理
	tools.Logger.Info()

	tools.Logger.Info(tools.StrWithColor("PRINT DETAIL TYPE3(dump file): ", "red"))
	dumpMap := dumpFileProcess(md5Show, md5Map, &shouldDeleteMd5Files, scanUuidFinal) //5、重复文件处理处理

	tools.Logger.Info(tools.StrWithColor("PRINT STAT TYPE0(comman info): ", "red"))
	tools.Logger.Info("suffixMap : ", tools.MarshalJsonToString(suffixMap))
	tools.Logger.Info("yearMap : ", tools.MarshalJsonToString(yearMap))
	tools.Logger.Info("month count: ")
	tools.MapPrintWithFilter(monthMap, monthFilter)
	tools.Logger.Info("day count: ")
	tools.MapPrintWithFilter(dayMap, dayFilter)
	tools.Logger.Info("file total : ", tools.StrWithColor(strconv.Itoa(fileTotalCnt), "red"))
	tools.Logger.Info("dir total : ", tools.StrWithColor(strconv.Itoa(dirTotalCnt), "red"))
	tools.Logger.Info("file contain date(just for print) : ", tools.StrWithColor(strconv.Itoa(fileDateFileList.Cardinality()), "red"))
	tools.Logger.Info("exif parse error 1 : ", tools.StrWithColor(tools.MarshalJsonToString(exifErr1FileSuffixMap), "red"))
	tools.Logger.Info("exif parse error 1 : ", tools.StrWithColor(strconv.Itoa(exifErr1FileSet.Cardinality()), "red"))
	//tools.Logger.Info("exif parse error 1 list : ", exifErr1FileSet)
	tools.Logger.Info("exif parse error 2 : ", tools.StrWithColor(tools.MarshalJsonToString(exifErr2FileSuffixMap), "red"))
	tools.Logger.Info("exif parse error 2 : ", tools.StrWithColor(strconv.Itoa(exifErr2FileSet.Cardinality()), "red"))
	//tools.Logger.Info("exif parse error 2 list : ", exifErr2FileSet)
	tools.Logger.Info("exif parse error 3 : ", tools.StrWithColor(tools.MarshalJsonToString(exifErr3FileSuffixMap), "red"))
	tools.Logger.Info("exif parse error 3 : ", tools.StrWithColor(strconv.Itoa(exifErr3FileSet.Cardinality()), "red"))
	//tools.Logger.Info("exif parse error 3 list : ", exifErr3FileSet)

	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("PRINT STAT TYPE1(delete file,modify date,move file): ", "red"))
	pr := "delete file total : " + tools.StrWithColor(strconv.Itoa(deleteFileList.Cardinality()), "red")
	if deleteFileList.Cardinality() > 0 && deleteAction {
		pr = pr + tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)
	pr = "modify date total : " + tools.StrWithColor(strconv.Itoa(modifyDateFileList.Cardinality()), "red")
	if modifyDateFileList.Cardinality() > 0 && modifyDateAction {
		pr = pr + tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)
	pr = "move file total : " + tools.StrWithColor(strconv.Itoa(moveFileList.Cardinality()), "red")
	if moveFileList.Cardinality() > 0 && moveFileAction {
		pr = pr + tools.StrWithColor("   actioned", "red")
	}
	tools.Logger.Info(pr)
	tools.Logger.Info("shoot date total : ", tools.StrWithColor(strconv.Itoa(shootDateFileList.Cardinality()), "red"))

	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("PRINT STAT TYPE2(empty dir) : ", "red"))
	tools.Logger.Info("empty dir total : ", tools.StrWithColor(strconv.Itoa(len(deleteDirList)), "red"))

	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("PRINT STAT TYPE3(dump file) : ", "red"))
	tools.Logger.Info("dump file total : ", tools.StrWithColor(strconv.Itoa(len(dumpMap)), "red"))

	tools.Logger.Info("shouldDeleteMd5Files length : ", tools.StrWithColor(strconv.Itoa(len(shouldDeleteMd5Files)), "red"))
	if len(shouldDeleteMd5Files) != 0 {
		//sm3 := tools.MarshalJsonToString(shouldDeleteMd5Files)
		sm3 := strings.Join(shouldDeleteMd5Files, "\n")
		filePath := cons.WorkDir + "/log/dump_delete_file/" + scanUuidFinal + "/dump_delete_list"
		tools.WriteStringToFile(sm3, filePath)
	}
	tools.Logger.Info("md5 get error length : ", tools.StrWithColor(strconv.Itoa(len(md5EmptyFileList)), "red"))
	if len(md5EmptyFileList) != 0 {
		tools.Logger.Info("md5EmptyFileList : ", tools.MarshalJsonToString(md5EmptyFileList))
	}

	tools.Logger.Info("imageNumMap length : ", tools.StrWithColor(strconv.Itoa(len(imageNumMap)), "red"))
	if len(imageNumMap) != 0 {
		filePath := cons.WorkDir + "/log/img_num_list"
		tools.ImageNumMapWriteToFile(imageNumMap, filePath)
	}
	tools.Logger.Info("imageNumRevMap length : ", tools.StrWithColor(strconv.Itoa(len(imageNumRevMap)), "red"))
	if len(imageNumRevMap) != 0 {
		filePath := cons.WorkDir + "/log/img_num_rev_list"
		tools.ImageNumRevMapWriteToFile(imageNumRevMap, filePath)
	}

	var bakNewFile []string
	var bakDeleteFile []string
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

	tools.Logger.Info()
	tools.Logger.Info(tools.StrWithColor("==========ROUND 3: PROCESS COST==========", "red"))
	tools.Logger.Info()
	elapsed3 := time.Since(start3)
	tools.Logger.Info("加载缓存完成耗时 : ", elapsed1)
	tools.Logger.Info("执行扫描完成耗时 : ", elapsed2)
	tools.Logger.Info("执行数据处理完成耗时 : ", elapsed3)
	tools.Logger.Info()

	imgRecord := ImgRecord{}
	imgRecord.FileTotal = fileTotalCnt
	imgRecord.FileTotalBak = fileTotalCntBak
	imgRecord.DirTotal = dirTotalCnt
	imgRecord.DirTotalBak = dirTotalCntBak
	imgRecord.StartDate = start1
	imgRecord.UseTime = int(math.Ceil(elapsed1.Seconds() + elapsed2.Seconds() + elapsed3.Seconds()))
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
	imgRecord.ShootDateFileCnt = shootDateFileList.Cardinality()
	imgRecord.EmptyDirCnt = len(deleteDirList)
	imgRecord.DumpFileCnt = len(dumpMap)
	//imgRecord.DumpFileDeleteList = shouldDeleteMd5Files
	imgRecord.ExifErr1Cnt = exifErr1FileSet.Cardinality()
	imgRecord.ExifErr2Cnt = exifErr2FileSet.Cardinality()
	imgRecord.ExifErr3Cnt = exifErr3FileSet.Cardinality()
	imgRecord.ExifErr1Map = exifErr1FileSuffixMap
	imgRecord.ExifErr2Map = exifErr2FileSuffixMap
	imgRecord.ExifErr3Map = exifErr3FileSuffixMap
	imgRecord.ScanArgs = tools.MarshalJsonToString(scanArgs)
	imgRecord.IsComplete = IsComplete
	imgRecord.Remark = ""

	ret := tools.MarshalJsonToString(imgRecord)
	tools.Logger.Info("scan result : ", ret)
	return ret, nil

}

func deleteFileProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool, deleteShow bool, deleteAction bool) {
	if deleteShow || deleteAction {
		tools.Logger.Info()
		tools.Logger.Info("file : ", tools.StrWithColor(ps.photo, "blue"))
		*printFileFlag = true
		tools.Logger.Info(tools.StrWithColor("should delete file :", "yellow"), ps.photo, " SIZE: ", *tools.GetFileSize(ps.photo))
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

func modifyDateProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool, modifyDateShow bool, modifyDateAction bool) {
	if modifyDateShow || modifyDateAction {
		if !*printFileFlag {
			tools.Logger.Info()
			tools.Logger.Info("file : ", tools.StrWithColor(ps.photo, "blue"))
			*printFileFlag = true
		}
		if !*printDateFlag {
			ps.psPrint()
			*printDateFlag = true
		}
		tools.Logger.Info(tools.StrWithColor("should modify file ", "yellow"), ps.photo, " modifyDate to ", ps.minDate)
	}
	if modifyDateAction {
		tm, _ := time.Parse("2006-01-02", ps.minDate)
		tools.ChangeModifyDate(ps.photo, tm)
		tools.Logger.Info(tools.StrWithColor("modify file ", "yellow"), ps.photo, "modifyDate to", ps.minDate, "get realdate", tools.GetModifyDate(ps.photo))
	}
}

func moveFileProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool, moveFileShow bool, moveFileAction bool) {
	if moveFileShow || moveFileAction {
		if !*printFileFlag {
			tools.Logger.Info()
			tools.Logger.Info("file : ", tools.StrWithColor(ps.photo, "blue"))
			*printFileFlag = true
		}
		if !*printDateFlag {
			ps.psPrint()
			*printDateFlag = true
		}
		tools.Logger.Info(tools.StrWithColor("should move file ", "yellow"), ps.photo, " to ", ps.targetPhoto)
	}
	if moveFileAction {
		tools.MoveFile(ps.photo, ps.targetPhoto)
		tools.Logger.Info(tools.StrWithColor("move file ", "yellow"), ps.photo, " to ", ps.targetPhoto)
	}
}

func emptyDirProcess(deleteShow bool, deleteAction bool, deleteDirList []dirStruct) {
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

func dumpFileProcess(md5Show bool, md5Map map[string][]string, shouldDeleteMd5Files *[]string, scanUuidFinal string) map[string][]string {
	var dumpMap = make(map[string][]string) //md5Map里筛选出有重复文件的Map

	if md5Show {
		for md5, files := range md5Map {
			if len(files) > 1 {
				dumpMap[md5] = files
				minPhoto := ""
				var fileSizeTemp *int64
				sizeMatch := true
				for _, photo := range files {
					if fileSizeTemp == nil {
						fileSizeTemp = tools.GetFileSize(photo)
					} else {
						if *fileSizeTemp != *tools.GetFileSize(photo) {
							sizeMatch = false
						}
					}

					if minPhoto == "" {
						minPhoto = photo
					} else {
						if tools.GetDirDate(minPhoto) > tools.GetDirDate(photo) { //留目录日期早的
							minPhoto = photo
						} else if tools.GetDirDate(minPhoto) < tools.GetDirDate(photo) {

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
					flag := ""
					if photo != minPhoto {
						if sizeMatch {
							*shouldDeleteMd5Files = append(*shouldDeleteMd5Files, photo)
							tools.Logger.Info("choose : ", photo, tools.StrWithColor(" DELETE", "red"), " SIZE: ", *tools.GetFileSize(photo))
							flag = "DELETE"
						} else {
							tools.Logger.Info("choose : ", photo, tools.StrWithColor(" SAVE(SIZE MISMATCH)", "green"), " SIZE: ", *tools.GetFileSize(photo))
							flag = "SAVE(SIZE MISMATCH) "
						}

					} else {

						if sizeMatch {
							tools.Logger.Info("choose : ", photo, tools.StrWithColor(" SAVE", "green"), " SIZE: ", *tools.GetFileSize(photo))
							flag = "SAVE"
						} else {
							tools.Logger.Info("choose : ", photo, tools.StrWithColor(" SAVE(SIZE MISMATCH)", "green"), " SIZE: ", *tools.GetFileSize(photo))
							flag = "SAVE(SIZE MISMATCH)"
						}

					}
					targetFile := cons.WorkDir + "/log/dump_delete_file/" + scanUuidFinal + "/dump_compare/" + md5 + "/" + flag + "_" + tools.GetDirDate(photo) + "_" + path.Base(photo)
					targetFileDir := filepath.Dir(targetFile)
					os.MkdirAll(targetFileDir, os.ModePerm)
					//tools.CopyFile(photo, targetFile)
				}
				tools.Logger.Info()

			}
		}

	}
	return dumpMap
}

func processOneFile(
	basePath string,
	photo string,
	md5Show bool,
	processFileList *[]photoStruct,
	fileDateFileList mapset.Set,
	moveFileList mapset.Set,
	modifyDateFileList mapset.Set,
	shootDateFileList mapset.Set,
	md5EmptyFileList []string,
	md5Map map[string][]string,
	exifErr1FileSuffixMap map[string]int,
	exifErr1FileSet mapset.Set,
	exifErr2FileSuffixMap map[string]int,
	exifErr2FileSet mapset.Set,
	exifErr3FileSuffixMap map[string]int,
	exifErr3FileSet mapset.Set,
	shootDateCacheMap map[string]string) {

	defer wg.Done()

	suffix := strings.ToLower(path.Ext(photo))

	if strings.HasSuffix(photo, "IMG_5081.HEIC") {
		tools.Logger.Info()
	}

	shootDate := ""
	if suffix != ".mov" && suffix != ".mp4" { //exif拍摄时间获取
		shootDate, _ = getShootDateMethod2(
			photo,
			suffix,
			exifErr1FileSuffixMap,
			exifErr1FileSet,
			shootDateCacheMap)
		if shootDate != "" {
			//tools.Logger.Info("shootDate : " + shootDate)
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
	if fileDate != "" {
		minDate = fileDate
	}

	ps := photoStruct{photo: photo, dirDate: dirDate, modifyDate: modifyDate, shootDate: shootDate, fileDate: fileDate, minDate: minDate}
	flag := false

	if dirDate != minDate {
		moveFileList.Add(photo)
		targetPath := basePath + string(os.PathSeparator) + minDate[0:4] + string(os.PathSeparator) + minDate[0:7] + string(os.PathSeparator) + minDate
		targetPath = tools.GetRealPath(targetPath)
		targetPhoto := targetPath + string(os.PathSeparator) + path.Base(photo)
		ps.isMoveFile = true
		ps.targetPhoto = targetPhoto
		flag = true

	}

	if shootDate != minDate {
		shootDateFileList.Add(photo)
	}

	if modifyDate != minDate {
		modifyDateFileList.Add(photo)
		ps.isModifyDateFile = true
		flag = true
	}

	if md5Show { //如果需要计算md5，则把所有照片按照md5整理
		md5, err := tools.GetFileMD5WithRetry(photo, cons.Md5Retry, cons.Md5CountLength)
		if err != nil {
			tools.Logger.Info("GetFileMD5 err for ", cons.Md5Retry, " times : ", err)
			md5EmptyFileListMu.Lock()
			md5EmptyFileList = append(md5EmptyFileList, photo)
			md5EmptyFileListMu.Unlock()
		} else {
			md5MapMu.Lock()
			if value, ok := md5Map[md5]; ok { //返回值ok表示是否存在这个值
				md5Map[md5] = append(value, photo)
			} else {
				md5Map[md5] = []string{photo}
			}
			md5MapMu.Unlock()
		}
	}

	if flag { //根据分类统计的结果，判断是否需要放入待处理列表里
		processFileListMu.Lock()
		*processFileList = append(*processFileList, ps)
		processFileListMu.Unlock()
	}

}

func createShootDateCache(shootDateCacheMap *map[string]string) {

	var imgShootDateSearch model.ImgShootDateSearch
	list, _, err := imgShootDateService.GetImgShootDateInfoList(imgShootDateSearch)
	if err != nil {

	}
	for _, isd := range list {
		(*shootDateCacheMap)[isd.ImgKey] = isd.ShootDate
	}
	tools.Logger.Info("")

}

func getShootDateMethod2(
	filepath string,
	suffix string,
	exifErr1FileSuffixMap map[string]int,
	exifErr1FileSet mapset.Set,
	shootDateCacheMap map[string]string,
) (string, error) {

	fileName := path.Base(filepath)
	dirDate := tools.GetDirDate(filepath)
	imgKey := dirDate + "|" + fileName
	shootDateRet := ""

	if value, ok := shootDateCacheMap[imgKey]; ok {
		shootDateRet = value
	} else {
		shootTime, err := tools.GetExifDateTime(filepath)
		var imgShootDateDB model.ImgShootDateDB
		imgShootDateDB.ImgKey = imgKey

		state := 1
		imgShootDateDB.State = &state
		if err != nil {
			exifErr1FileMu.Lock()
			if value, ok := exifErr1FileSuffixMap[suffix]; ok {
				exifErr1FileSuffixMap[suffix] = value + 1
			} else {
				exifErr1FileSuffixMap[suffix] = 1
			}
			exifErr1FileSet.Add(filepath)
			exifErr1FileMu.Unlock()
			shootDateRet = ""
			imgShootDateDB.ShootDate = shootDateRet
		} else {
			shootDateRet = shootTime.Format("2006-01-02")
			//shootTimeStr := shootTime.Format("2006-01-02 15:04:05")
			imgShootDateDB.ShootDate = shootDateRet
		}
		if cons.ImgCache { //指定使用cache时，才更新库
			if err = imgShootDateService.CreateImgShootDate(&imgShootDateDB); err != nil {
				tools.Logger.Error("CreateImgShootDate error : ", err)
			}
		}

	}

	return shootDateRet, nil

	/*f, err := os.Open(path)

	defer func() {
		f.Close()
		if r := recover(); r != nil {
			tools.Logger.Error("exifErr3 Recovered. Error : ", r, " path : ", path)
			exifErr3FileMu.Lock()
			if value, ok := exifErr3FileSuffixMap[suffix]; ok {
				exifErr3FileSuffixMap[suffix] = value + 1
			} else {
				exifErr3FileSuffixMap[suffix] = 1
			}
			exifErr3FileSet.Add(path)
			exifErr3FileMu.Unlock()
		}
	}()

	if err != nil {
		tools.Logger.Error(err)
		return "", err
	}

	x, err := exif.Decode(f)
	if err != nil {
		//fmt.Println("exifErr1 Decode Error : ", err, " path : ", path)
		exifErr1FileMu.Lock()
		if value, ok := exifErr1FileSuffixMap[suffix]; ok {
			exifErr1FileSuffixMap[suffix] = value + 1
		} else {
			exifErr1FileSuffixMap[suffix] = 1
		}
		exifErr1FileSet.Add(path)
		exifErr1FileMu.Unlock()

		return "", errors.New("exif decode error")
	}

	shootTime, err := x.DateTime()

	if err != nil {
		exifErr2FileMu.Lock()
		if value, ok := exifErr2FileSuffixMap[suffix]; ok {
			exifErr2FileSuffixMap[suffix] = value + 1
		} else {
			exifErr2FileSuffixMap[suffix] = 1
		}
		exifErr2FileSet.Add(path)
		exifErr2FileMu.Unlock()

		return "", errors.New("no shoot time")
	} else {
		shootTimeStr := shootTime.Format("2006-01-02")
		//shootTimeStr := shootTime.Format("2006-01-02 15:04:05")
		return shootTimeStr, nil
	}*/

}
