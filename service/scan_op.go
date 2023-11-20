package service

import (
	"errors"
	mapset "github.com/deckarep/golang-set"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"img_process/tools"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	//exif "github.com/dsoprea/go-exif/v3"
	"github.com/panjf2000/ants/v2"
)

const startPath = "/Users/ld/Desktop/pic-new" //统计的起始目录，必须包含pic-new
//const startPath = "/Volumes/ld_ssd/pic-new"

// const startPath = "/Volumes/ld_hardone/pic-new"
//const startPath = "/Volumes/ld_hardraid/old-pic/pic-new"

const poolSize = 8                //并行处理的线程
const md5Retry = 3                //文件md5计算重试次数
const md5CountLength = 1024 * 128 //md5计算的长度

const monthFilter = "xx" //月份过滤
const dayFilter = "xx"   //日期过滤

var sl = tools.InitLogger()

var basePath = startPath[0 : strings.Index(startPath, "pic-new")+7] //指向pic-new的目录

var suffixMap = map[string]int{} //后缀统计
var yearMap = map[string]int{}   //年份统计
var monthMap = map[string]int{}  //月份统计
var dayMap = map[string]int{}    //日期统计

var fileTotalCnt = 0 //文件总量
var dirTotalCnt = 0  //目录总量

var fileDateFileList = mapset.NewSet() //文件名带日期的照片

var deleteFileList = mapset.NewSet()     //需要删除的文件
var moveFileList = mapset.NewSet()       //目录与最小日期不匹配，需要移动
var modifyDateFileList = mapset.NewSet() //修改时间与最小日期不匹配，需要修改
var shootDateFileList = mapset.NewSet()  //拍摄时间与最小日期不匹配，需要修改

var shouldDeleteMd5Files []string //统计需要删除的文件

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
	FileTotal          int            //文件总数
	DirTotal           int            //目录总数
	CreateDate         time.Time      //记录时间
	UseTime            int            //用时
	BasePath           string         //基础目录
	SuffixMap          map[string]int //后缀统计
	YearMap            map[string]int //年份统计
	FileDateCnt        int            //有时间文件统计
	DeleteFileCnt      int            //需要删除文件数
	ModifyDateFileCnt  int            //需要修改修改日期文件数
	MoveFileCnt        int            //需要移动文件数
	ShootDateFileCnt   int            //需要修改拍摄日期文件数
	EmptyDirCnt        int            //空文件数
	DumpFileCnt        int            //重复md5数
	DumpFileDeleteList []string       //需要删除文件数
	ExifErr1Cnt        int            //exif错误1数
	ExifErr2Cnt        int            //exif错误2数
	ExifErr3Cnt        int            //exif错误3数
	ExifErr1Map        string         //exif错误1统计
	ExifErr2Map        string         //exif错误2统计
	ExifErr3Map        string         //exif错误3统计
}

func (ps *photoStruct) psPrint() { //打印照片相关信息
	if ps.dirDate != ps.minDate {
		sl.Info("dirDate : ", tools.StrWithColor(ps.dirDate, "red"))
	} else {
		sl.Info("dirDate : ", tools.StrWithColor(ps.dirDate, "green"))
	}
	if ps.modifyDate != ps.minDate {
		sl.Info("modifyDate : ", tools.StrWithColor(ps.modifyDate, "red"))
	} else {
		sl.Info("modifyDate : ", tools.StrWithColor(ps.modifyDate, "green"))
	}
	if ps.shootDate != ps.minDate {
		sl.Info("shootDate : ", tools.StrWithColor(ps.shootDate, "red"))
	} else {
		sl.Info("shootDate : ", tools.StrWithColor(ps.shootDate, "green"))
	}
	sl.Info("minDate : ", tools.StrWithColor(ps.minDate, "green"))
}

var deleteDirList []dirStruct     //需要处理的目录结构体列表（空目录）
var processFileList []photoStruct //需要处理的文件结构体列表（非法格式删除、移动、修改时间、重复文件删除）
var processFileListMu sync.Mutex

var md5Map = make(map[string][]string) //以md5为key存储文件
var md5MapMu sync.Mutex

var exifErr1FileSuffixMap = map[string]int{} //shoot time error1后缀
var exifErr1FileSet = mapset.NewSet()        //shoot time error1照片
var exifErr2FileSuffixMap = map[string]int{} //shoot time error2后缀
var exifErr2FileSet = mapset.NewSet()        //shoot time error2照片
var exifErr3FileSuffixMap = map[string]int{} //shoot time error3后缀
var exifErr3FileSet = mapset.NewSet()        //shoot time error3照片
var exifErr1FileMu sync.Mutex
var exifErr2FileMu sync.Mutex
var exifErr3FileMu sync.Mutex

var md5EmptyFileList []string //获取md5为空的文件
var md5EmptyFileListMu sync.Mutex

var wg sync.WaitGroup //异步照片处理等待

func DoScan(
	deleteShow bool,
	moveFileShow bool,
	modifyDateShow bool,
	md5Show bool,
	deleteAction bool,
	moveFileAction bool,
	modifyDateAction bool) string {

	defer sl.Sync()

	start := time.Now() // 获取当前时间

	sl.Info()
	sl.Info("————————————————————————————————————————————————————————")
	sl.Info("time : ", start.Format(tools.DatetimeTemplate))
	sl.Info("startPath : ", startPath)
	sl.Info("basePath : ", basePath)

	sl.Info()

	sl.Info(tools.StrWithColor("==========ROUND 1: SCAN FILE==========", "red"))
	sl.Info()

	p, _ := ants.NewPool(poolSize) //新建一个pool对象
	defer p.Release()

	// 计时器
	//ticker := time.NewTicker(time.Second * 2)
	ticker := time.NewTicker(time.Minute * 5)
	tickerSize := 0
	go func() {
		for t := range ticker.C {
			sl.Info(tools.StrWithColor("Tick at "+t.Format(tools.DatetimeTemplate), "red") + tools.StrWithColor(" , tick range processed "+strconv.Itoa(fileTotalCnt-tickerSize), "red"))
			tickerSize = fileTotalCnt
			sl.Info()
		}
	}()

	_ = filepath.Walk(startPath, func(file string, info os.FileInfo, err error) error {
		if info.IsDir() { //遍历目录
			if flag, err := tools.IsEmpty(file); err == nil && flag { //空目录加入待处理列表
				ds := dirStruct{isEmptyDir: true, dir: file}
				deleteDirList = append(deleteDirList, ds)

			}
			dirTotalCnt = dirTotalCnt + 1
		} else { //遍历文件
			//sl.Info(file)
			fileName := path.Base(file)
			fileSuffix := strings.ToLower(path.Ext(file))

			flag := true
			if strings.HasPrefix(fileName, ".") || strings.HasSuffix(fileName, "nas_downloading") { //非法文件加入待处理列表
				ps := photoStruct{isDeleteFile: true, photo: file}
				processFileListMu.Lock()
				processFileList = append(processFileList, ps)
				processFileListMu.Unlock()
				deleteFileList.Add(file)

				flag = false

			}

			if flag {

				wg.Add(1)

				_ = p.Submit(func() {
					processOneFile(file, md5Show) //单个文件协程处理
				})

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

				fileTotalCnt = fileTotalCnt + 1
				if fileTotalCnt%1000 == 0 { //每隔1000行打印一次
					sl.Info("processed ", tools.StrWithColor(strconv.Itoa(fileTotalCnt), "red"))
					sl.Info("pool running size : ", p.Running())
				}
			}
		}
		return nil
	})
	sl.Info("processed(end) ", tools.StrWithColor(strconv.Itoa(fileTotalCnt), "red"))

	wg.Wait()

	sl.Info(tools.StrWithColor("Tick at "+time.Now().Format(tools.DatetimeTemplate), "red") + tools.StrWithColor(" , tick range processed "+strconv.Itoa(fileTotalCnt-tickerSize), "red"))

	ticker.Stop() //计时终止

	elapsed := time.Since(start)

	start2 := time.Now() // 获取当前时间

	sl.Info()
	sl.Info(tools.StrWithColor("==========ROUND 2: PROCESS FILE==========", "red"))
	sl.Info()
	sl.Info(tools.StrWithColor("PRINT DETAIL TYPE1(delete file,modify date,move file): ", "red"))
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
	sl.Info()
	sl.Info(tools.StrWithColor("PRINT DETAIL TYPE2(empty dir): ", "red"))
	emptyDirProcess(deleteShow, deleteAction) //4、空目录处理
	sl.Info()

	sl.Info(tools.StrWithColor("PRINT DETAIL TYPE3(dump file): ", "red"))
	dumpMap := dumpFileProcess(md5Show) //5、重复文件处理处理

	sl.Info(tools.StrWithColor("PRINT STAT TYPE0(comman info): ", "red"))
	sl.Info("suffixMap : ", tools.MarshalPrint(suffixMap))
	sl.Info("yearMap : ", tools.MarshalPrint(yearMap))
	sl.Info("month count: ")
	tools.MapPrintWithFilter(monthMap, monthFilter)
	sl.Info("day count: ")
	tools.MapPrintWithFilter(dayMap, dayFilter)
	sl.Info("file total : ", tools.StrWithColor(strconv.Itoa(fileTotalCnt), "red"))
	sl.Info("dir total : ", tools.StrWithColor(strconv.Itoa(dirTotalCnt), "red"))
	sl.Info("file contain date(just for print) : ", tools.StrWithColor(strconv.Itoa(fileDateFileList.Cardinality()), "red"))
	sl.Info("exif parse error 1 : ", tools.StrWithColor(tools.MarshalPrint(exifErr1FileSuffixMap), "red"))
	sl.Info("exif parse error 1 : ", tools.StrWithColor(strconv.Itoa(exifErr1FileSet.Cardinality()), "red"))
	//sl.Info("exif parse error 1 list : ", exifErr1FileSet)
	sl.Info("exif parse error 2 : ", tools.StrWithColor(tools.MarshalPrint(exifErr2FileSuffixMap), "red"))
	sl.Info("exif parse error 2 : ", tools.StrWithColor(strconv.Itoa(exifErr2FileSet.Cardinality()), "red"))
	//sl.Info("exif parse error 2 list : ", exifErr2FileSet)
	sl.Info("exif parse error 3 : ", tools.StrWithColor(tools.MarshalPrint(exifErr3FileSuffixMap), "red"))
	sl.Info("exif parse error 3 : ", tools.StrWithColor(strconv.Itoa(exifErr3FileSet.Cardinality()), "red"))
	//sl.Info("exif parse error 3 list : ", exifErr3FileSet)

	sl.Info()
	sl.Info(tools.StrWithColor("PRINT STAT TYPE1(delete file,modify date,move file): ", "red"))
	pr := "delete file total : " + tools.StrWithColor(strconv.Itoa(deleteFileList.Cardinality()), "red")
	if deleteFileList.Cardinality() > 0 && deleteAction {
		pr = pr + tools.StrWithColor("   actioned", "red")
	}
	sl.Info(pr)
	pr = "modify date total : " + tools.StrWithColor(strconv.Itoa(modifyDateFileList.Cardinality()), "red")
	if modifyDateFileList.Cardinality() > 0 && modifyDateAction {
		pr = pr + tools.StrWithColor("   actioned", "red")
	}
	sl.Info(pr)
	pr = "move file total : " + tools.StrWithColor(strconv.Itoa(moveFileList.Cardinality()), "red")
	if moveFileList.Cardinality() > 0 && moveFileAction {
		pr = pr + tools.StrWithColor("   actioned", "red")
	}
	sl.Info(pr)
	sl.Info("shoot date total : ", tools.StrWithColor(strconv.Itoa(shootDateFileList.Cardinality()), "red"))

	sl.Info()
	sl.Info(tools.StrWithColor("PRINT STAT TYPE2(empty dir) : ", "red"))
	sl.Info("empty dir total : ", tools.StrWithColor(strconv.Itoa(len(deleteDirList)), "red"))

	sl.Info()
	sl.Info(tools.StrWithColor("PRINT STAT TYPE3(dump file) : ", "red"))
	sl.Info("dump file total : ", tools.StrWithColor(strconv.Itoa(len(dumpMap)), "red"))

	sl.Info("shouldDeleteMd5Files length : ", tools.StrWithColor(strconv.Itoa(len(shouldDeleteMd5Files)), "red"))
	if len(shouldDeleteMd5Files) != 0 {
		sm3 := tools.MarshalPrint(shouldDeleteMd5Files)
		sl.Info("shouldDeleteMd5Files print origin : ", sm3)
		fileUuid, _ := tools.WriteStringToUuidFile(sm3)

		filePath := "/tmp/" + fileUuid
		//sl.Info("file path : ", filePath)
		fileContent2, _ := tools.ReadFileString(filePath)

		sl.Info("shouldDeleteMd5Files print reread : ", fileContent2)
		sl.Info("tmp file md5 : ", tools.StrWithColor(fileUuid, "red"))
	}
	sl.Info("md5 get error length : ", tools.StrWithColor(strconv.Itoa(len(md5EmptyFileList)), "red"))
	if len(md5EmptyFileList) != 0 {
		sl.Info("md5EmptyFileList : ", tools.MarshalPrint(md5EmptyFileList))
	}

	sl.Info()
	sl.Info(tools.StrWithColor("==========ROUND 3: PROCESS COST==========", "red"))
	sl.Info()
	elapsed2 := time.Since(start2)
	sl.Info("执行扫描完成耗时 : ", elapsed)
	sl.Info("执行数据处理完成耗时 : ", elapsed2)
	sl.Info()

	imgRecord := ImgRecord{}
	imgRecord.FileTotal = fileTotalCnt
	imgRecord.DirTotal = dirTotalCnt
	imgRecord.CreateDate = time.Now()
	imgRecord.UseTime = int(math.Ceil(elapsed.Seconds()))
	imgRecord.BasePath = basePath
	imgRecord.SuffixMap = suffixMap
	imgRecord.YearMap = yearMap
	imgRecord.FileDateCnt = fileDateFileList.Cardinality()
	imgRecord.DeleteFileCnt = deleteFileList.Cardinality()
	imgRecord.ModifyDateFileCnt = modifyDateFileList.Cardinality()
	imgRecord.MoveFileCnt = moveFileList.Cardinality()
	imgRecord.ShootDateFileCnt = shootDateFileList.Cardinality()
	imgRecord.EmptyDirCnt = len(deleteDirList)
	imgRecord.DumpFileCnt = len(dumpMap)
	imgRecord.DumpFileDeleteList = shouldDeleteMd5Files
	imgRecord.ExifErr1Cnt = exifErr1FileSet.Cardinality()
	imgRecord.ExifErr2Cnt = exifErr2FileSet.Cardinality()
	imgRecord.ExifErr3Cnt = exifErr3FileSet.Cardinality()
	//imgRecord.ExifErr1Map = exifErr1FileSet.String()
	//imgRecord.ExifErr2Map = exifErr2FileSet.String()
	//imgRecord.ExifErr3Map = exifErr3FileSet.String()

	return tools.MarshalPrint(imgRecord)

}

func deleteFileProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool, deleteShow bool, deleteAction bool) {
	if deleteShow || deleteAction {
		sl.Info()
		sl.Info("file : ", tools.StrWithColor(ps.photo, "blue"))
		*printFileFlag = true
		sl.Info(tools.StrWithColor("should delete file :", "yellow"), ps.photo)
	}

	if deleteAction {
		err := os.Remove(ps.photo)
		if err != nil {
			sl.Info(tools.StrWithColor("delete file failed:", "yellow"), ps.photo, err)
		} else {
			sl.Info(tools.StrWithColor("delete file sucessed:", "green"), ps.photo)
		}
	}
}

func modifyDateProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool, modifyDateShow bool, modifyDateAction bool) {
	if modifyDateShow || modifyDateAction {
		if !*printFileFlag {
			sl.Info()
			sl.Info("file : ", tools.StrWithColor(ps.photo, "blue"))
			*printFileFlag = true
		}
		if !*printDateFlag {
			ps.psPrint()
			*printDateFlag = true
		}
		sl.Info(tools.StrWithColor("should modify file ", "yellow"), ps.photo, "modifyDate to", ps.minDate)
	}
	if modifyDateAction {
		tm, _ := time.Parse("2006-01-02", ps.minDate)
		tools.ChangeModifyDate(ps.photo, tm)
		sl.Info(tools.StrWithColor("modify file ", "yellow"), ps.photo, "modifyDate to", ps.minDate, "get realdate", tools.GetModifyDate(ps.photo))
	}
}

func moveFileProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool, moveFileShow bool, moveFileAction bool) {
	if moveFileShow || moveFileAction {
		if !*printFileFlag {
			sl.Info()
			sl.Info("file : ", tools.StrWithColor(ps.photo, "blue"))
			*printFileFlag = true
		}
		if !*printDateFlag {
			ps.psPrint()
			*printDateFlag = true
		}
		sl.Info(tools.StrWithColor("should move file ", "yellow"), ps.photo, "to", ps.targetPhoto)
	}
	if moveFileAction {
		tools.MoveFile(ps.photo, ps.targetPhoto)
		sl.Info(tools.StrWithColor("move file ", "yellow"), ps.photo, "to", ps.targetPhoto)
	}
}

func emptyDirProcess(deleteShow bool, deleteAction bool) {
	for _, ds := range deleteDirList {
		if ds.isEmptyDir {
			if deleteShow || deleteAction {
				sl.Info("dir : ", tools.StrWithColor(ds.dir, "blue"))
				sl.Info(tools.StrWithColor("should delete empty dir :", "yellow"), ds.dir)
			}

			if deleteAction {
				err := os.Remove(ds.dir)
				if err != nil {
					sl.Info(tools.StrWithColor("delete empty dir failed:", "yellow"), ds.dir, err)
				} else {
					sl.Info(tools.StrWithColor("delete empty dir sucessed:", "green"), ds.dir)
				}
			}
		}
		sl.Info()

	}
}

func dumpFileProcess(md5Show bool) map[string][]string {
	var dumpMap = make(map[string][]string) //md5Map里筛选出有重复文件的Map

	timeStr := time.Now().Format(tools.DatetimeDirTemplate)
	if md5Show {
		for md5, files := range md5Map {
			if len(files) > 1 {
				dumpMap[md5] = files
				minPhoto := ""
				for _, photo := range files {
					if minPhoto == "" {
						minPhoto = photo
					} else {
						if tools.GetDirDate(minPhoto) > tools.GetDirDate(photo) {
							minPhoto = photo
						} else if tools.GetDirDate(minPhoto) < tools.GetDirDate(photo) {

						} else {
							if path.Base(minPhoto) > path.Base(photo) {
								minPhoto = photo
							}
						}
					}
				}

				sl.Info("file : ", tools.StrWithColor(md5, "blue"))
				for _, photo := range files {
					flag := ""
					if photo != minPhoto {
						shouldDeleteMd5Files = append(shouldDeleteMd5Files, photo)
						sl.Info("choose : ", photo, tools.StrWithColor(" DELETE", "red"))
						flag = "DELETE"
					} else {
						sl.Info("choose : ", photo, tools.StrWithColor(" SAVE", "green"))
						flag = "SAVE"
					}
					targetFile := "/tmp/" + timeStr + "/" + md5 + "/" + flag + "_" + tools.GetDirDate(photo) + "_" + path.Base(photo)
					targetFileDir := filepath.Dir(targetFile)
					os.MkdirAll(targetFileDir, os.ModePerm)
					tools.CopyFile(photo, targetFile)
				}
				sl.Info()

			}
		}

	}
	return dumpMap
}

func processOneFile(photo string, md5Show bool) {

	defer wg.Done()

	suffix := strings.ToLower(path.Ext(photo))

	shootDate := ""
	if suffix != ".heic" && suffix != ".mov" && suffix != ".mp4" && suffix != ".png" { //exif拍摄时间获取
		shootDate, _ = getShootDateMethod2(photo, suffix)
		if shootDate != "" {
			//sl.Info("shootDate : " + shootDate)
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
		targetPhoto := basePath + string(os.PathSeparator) + minDate[0:4] + string(os.PathSeparator) + minDate[0:7] + string(os.PathSeparator) + minDate + string(os.PathSeparator) + path.Base(photo)
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
		md5, err := tools.GetFileMD5WithRetry(photo, md5Retry, md5CountLength)
		if err != nil {
			sl.Info("GetFileMD5 err for ", md5Retry, " times : ", err)
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
		processFileList = append(processFileList, ps)
		processFileListMu.Unlock()
	}

}

func getShootDateMethod2(path string, suffix string) (string, error) {

	f, err := os.Open(path)

	defer func() {
		if r := recover(); r != nil {
			//sl.Info("Recovered. Error:\n", r)
			exifErr3FileMu.Lock()
			if value, ok := exifErr3FileSuffixMap[suffix]; ok {
				exifErr3FileSuffixMap[suffix] = value + 1
			} else {
				exifErr3FileSuffixMap[suffix] = 1
			}
			exifErr3FileSet.Add(path)
			exifErr3FileMu.Unlock()
		}
		f.Close()
	}()

	if err != nil {
		sl.Error(err)
		return "", err
	}

	// Optionally register camera makenote data parsing - currently Nikon and Canon are supported.
	exif.RegisterParsers(mknote.All...)

	x, err := exif.Decode(f)
	if err != nil {
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
	}

}
