package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"img_process/tools"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	//exif "github.com/dsoprea/go-exif/v3"
	"github.com/panjf2000/ants/v2"
)

var startPath = "/Users/ld/Desktop/pic-new" //统计的起始目录，必须包含pic-new
var poolSize = 8                            //并行处理的线程

//var startPath = "/Volumes/ld_hardone/pic-new"

var basePath = startPath[0 : strings.Index(startPath, "pic-new")+7] //指向pic-new的目录

var deleteShow = true
var dirDateShow = true
var modifyDateShow = true
var md5Show = true

var deleteAction = false
var dirDateAction = false
var modifyDateAction = false
var md5Action = false

var suffixMap = map[string]int{} //后缀统计
var nost1FileSuffixMap sync.Map  //shoot time没有的照片
var nost2FileSuffixMap sync.Map  //shoot time没有的照片

var md5Map sync.Map                     //以md5为key存储文件
var dumpMap = make(map[string][]string) //md5Map里筛选出有重复文件的Map

var totalCnt = 0 //照片总量

var fileDateFileList = mapset.NewSet() //文件名带日期的照片

var deleteFileList = mapset.NewSet()     //需要删除的文件
var dirDateFileList = mapset.NewSet()    //目录与最小日期不匹配，需要移动
var modifyDateFileList = mapset.NewSet() //修改时间与最小日期不匹配，需要修改
var shootDateFileList = mapset.NewSet()  //拍摄时间与最小日期不匹配，需要修改

var processDirList = []dirStruct{}    //需要处理的目录结构体列表（空目录）
var processFileList = []photoStruct{} //需要处理的文件结构体列表（非法格式删除、移动、修改时间、重复文件删除）
var shouldDeleteFiles = []string{}    //统计需要删除的文件

var date1Pattern = regexp.MustCompile("^.*(20[012]\\d}(0[1-9]|1[0-2])(0[1-9]|[1-2]\\d|3[01])).*$")
var data1Template = "20060102"
var date2Pattern = regexp.MustCompile("^.*((0[1-9]|[1-2]\\d|3[01])-(0[1-9]|1[0-2])-[012]\\d).*$")
var data2Template = "02-01-06" // 31-12-19
var date3Pattern = regexp.MustCompile("^.*(20[012]\\d:(0[1-9]|1[0-2]):(0[1-9]|[1-2]\\d|3[01])).*$")
var data3Template = "2006:01:02" //
var date4Pattern = regexp.MustCompile("^.*(20[012]\\d-(0[1-9]|1[0-2])-(0[1-9]|[1-2]\\d|3[01])).*$")
var data4Template = "2006-01-02" //
var datetimePattern *regexp.Regexp = regexp.MustCompile("^.*(20[012]\\d:(0[1-9]|1[0-2]):(0[1-9]|[1-2]\\d|3[01]) (\\d{2}:\\d{2}:\\d{2})).*$")
var datetimeTemplate = "2006:01:02 15:04:05"

var timePatternArray = []*regexp.Regexp{date1Pattern, date2Pattern, date3Pattern, date4Pattern, datetimePattern}
var timeTemplateArray = []string{data1Template, data2Template, data3Template, data4Template, datetimeTemplate}

var wg sync.WaitGroup

type dirStruct struct { //目录打印需要的结构体
	dir string

	isEmptyDir bool
}

type photoStruct struct { //照片打印需要的结构体
	photo      string
	dirDate    string
	modifyDate string
	shootDate  string
	fileDate   string
	minDate    string

	isDeleteFile bool

	isMoveFile  bool
	targetPhoto string

	isModifyDateFile bool
}

var processFileListMu sync.Mutex
var md5MapMu sync.Mutex

func main() {

	start := time.Now() // 获取当前时间

	fmt.Println("startPath : ", startPath)
	fmt.Println("basePath : ", basePath)

	println()

	fmt.Println(tools.StrWithColor("==========ROUND 1: SCAN FILE==========", "red"))
	fmt.Println()

	p, _ := ants.NewPool(poolSize) //新建一个pool对象，其他同上
	defer p.Release()

	_ = filepath.Walk(startPath, func(file string, info os.FileInfo, err error) error {
		if info.IsDir() { //遍历目录
			if flag, err := tools.IsEmpty(file); err == nil && flag {
				ds := dirStruct{isEmptyDir: true, dir: file}
				processDirList = append(processDirList, ds)

			}
		} else { //遍历文件
			//fmt.Println(file)
			fileName := path.Base(file)
			fileSuffix := strings.ToLower(path.Ext(file))

			flag := true
			if strings.HasPrefix(fileName, ".") || strings.HasSuffix(fileName, "nas_downloading") {
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
					processOneFile(file) //单个文件处理，数据放到不同的归档里
				})

				if value, ok := suffixMap[fileSuffix]; ok {
					suffixMap[fileSuffix] = value + 1
				} else {
					suffixMap[fileSuffix] = 1
				}

				totalCnt = totalCnt + 1
				if totalCnt%100 == 0 {
					println("processed ", tools.StrWithColor(strconv.Itoa(totalCnt), "red"))
				}
			}
		}
		return nil
	})
	fmt.Println("processed(end)", tools.StrWithColor(strconv.Itoa(totalCnt), "red"))

	wg.Wait()

	elapsed := time.Since(start)

	start2 := time.Now() // 获取当前时间

	fmt.Println()
	fmt.Println(tools.StrWithColor("==========ROUND 2: PROCESS FILE==========", "red"))
	fmt.Println()
	fmt.Println(tools.StrWithColor("PRINT DETAIL TYPE1(delete file,modify date,move file): ", "red"))
	for _, ps := range processFileList {

		printFileFlag := false
		printDateFlag := false

		if ps.isDeleteFile {
			deleteFileProcess(ps, &printFileFlag, &printDateFlag) //1、需要删除的文件处理
		}
		if ps.isModifyDateFile {
			modifyDateProcess(ps, &printFileFlag, &printDateFlag) //2、需要修改时间的文件处理
		}
		if ps.isMoveFile {
			dirDateProcess(ps, &printFileFlag, &printDateFlag) //3、需要移动的文件处理
		}

	}
	fmt.Println()
	fmt.Println(tools.StrWithColor("PRINT DETAIL TYPE2(empty dir): ", "red"))
	emptyDirProcess() //4、空目录处理

	fmt.Println(tools.StrWithColor("PRINT DETAIL TYPE3(dump file): ", "red"))
	dumpFileProcess() //5、重复文件处理处理

	fmt.Println()
	fmt.Println(tools.StrWithColor("PRINT STAT TYPE0(comman info): ", "red"))
	sm, _ := json.Marshal(suffixMap)
	fmt.Println("suffixMap : ", string(sm))
	fmt.Println("photo total : ", tools.StrWithColor(strconv.Itoa(totalCnt), "red"))
	fmt.Println("file contain date(just for print) : ", tools.StrWithColor(strconv.Itoa(fileDateFileList.Cardinality()), "red"))

	fmt.Println(tools.StrWithColor("PRINT STAT TYPE1(delete file,modify date,move file): ", "red"))
	fmt.Println("delete file total : ", tools.StrWithColor(strconv.Itoa(deleteFileList.Cardinality()), "red"))
	fmt.Println("modify date total : ", tools.StrWithColor(strconv.Itoa(modifyDateFileList.Cardinality()), "red"))
	fmt.Println("move file total : ", tools.StrWithColor(strconv.Itoa(dirDateFileList.Cardinality()), "red"))
	fmt.Println("shoot date total : ", tools.StrWithColor(strconv.Itoa(shootDateFileList.Cardinality()), "red"))

	fmt.Println("exif parse error 1 : ", tools.StrWithColor(strconv.Itoa(tools.GetSyncMapLens(nost1FileSuffixMap)), "red"))
	//fmt.Println("exif parse error 1 list : ", nost1FileSuffixMap)
	fmt.Println("exif parse error 2 : ", tools.StrWithColor(strconv.Itoa(tools.GetSyncMapLens(nost2FileSuffixMap)), "red"))
	//fmt.Println("exif parse error 2 list : ", nost2FileSuffixMap)

	fmt.Println(tools.StrWithColor("PRINT STAT TYPE2(empty dir) : ", "red"))
	fmt.Println("empty dir total : ", tools.StrWithColor(strconv.Itoa(len(processDirList)), "red"))

	fmt.Println(tools.StrWithColor("PRINT STAT TYPE3(dump file) : ", "red"))
	fmt.Println("dump file total : ", tools.StrWithColor(strconv.Itoa(len(dumpMap)), "red"))
	sm3, _ := json.Marshal(shouldDeleteFiles)
	fmt.Println("shouldDeleteFiles length : ", tools.StrWithColor(strconv.Itoa(len(shouldDeleteFiles)), "red"))
	fmt.Println("shouldDeleteFiles : ", string(sm3))

	fmt.Println()
	fmt.Println(tools.StrWithColor("==========ROUND 3: PROCESS COST==========", "red"))
	fmt.Println()
	elapsed2 := time.Since(start2)
	fmt.Println("执行扫描完成耗时 : ", elapsed)
	fmt.Println("执行数据处理完成耗时 : ", elapsed2)

}

func deleteFileProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool) {
	if deleteShow || deleteAction {
		fmt.Println()
		fmt.Println("file : ", tools.StrWithColor(ps.photo, "blue"))
		*printFileFlag = true
		fmt.Println(tools.StrWithColor("should delete file :", "yellow"), ps.photo)
	}

	if deleteAction {
		err := os.Remove(ps.photo)
		if err != nil {
			println(tools.StrWithColor("delete file failed:", "yellow"), ps.photo, err)
		} else {
			println(tools.StrWithColor("delete file sucessed:", "green"), ps.photo)
		}
	}
}

func modifyDateProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool) {
	if modifyDateShow || modifyDateAction {
		if !*printFileFlag {
			fmt.Println()
			fmt.Println("file : ", tools.StrWithColor(ps.photo, "blue"))
			*printFileFlag = true
		}
		if !*printDateFlag {
			printDate(ps.photo, ps.dirDate, ps.modifyDate, ps.shootDate, ps.fileDate, ps.minDate)
			*printDateFlag = true
		}
		fmt.Println(tools.StrWithColor("should modify file ", "yellow"), ps.photo, "modifyDate to", ps.minDate)
	}
	if modifyDateAction {
		tm, _ := time.Parse("2006-01-02", ps.minDate)
		changeModifyDate(ps.photo, tm)
		fmt.Println(tools.StrWithColor("modify file ", "yellow"), ps.photo, "modifyDate to", ps.minDate, "get realdate", getModifyDate(ps.photo))
	}
}

func dirDateProcess(ps photoStruct, printFileFlag *bool, printDateFlag *bool) {
	if dirDateShow || dirDateAction {
		if !*printFileFlag {
			fmt.Println()
			fmt.Println("file : ", tools.StrWithColor(ps.photo, "blue"))
			*printFileFlag = true
		}
		if !*printDateFlag {
			printDate(ps.photo, ps.dirDate, ps.modifyDate, ps.shootDate, ps.fileDate, ps.minDate)
			*printDateFlag = true
		}
		fmt.Println(tools.StrWithColor("should move file ", "yellow"), ps.photo, "to", ps.targetPhoto)
	}
	if dirDateAction {
		tools.MoveFile(ps.photo, ps.targetPhoto)
		fmt.Println(tools.StrWithColor("move file ", "yellow"), ps.photo, "to", ps.targetPhoto)
	}
}

func emptyDirProcess() {
	for _, ds := range processDirList {
		if ds.isEmptyDir {
			if deleteShow || deleteAction {
				fmt.Println("dir : ", tools.StrWithColor(ds.dir, "blue"))
				fmt.Println(tools.StrWithColor("should delete empty dir :", "yellow"), ds.dir)
			}

			if deleteAction {
				err := os.Remove(ds.dir)
				if err != nil {
					println(tools.StrWithColor("delete empty dir failed:", "yellow"), ds.dir, err)
				} else {
					println(tools.StrWithColor("delete empty dir sucessed:", "green"), ds.dir)
				}
			}
		}
		fmt.Println()

	}
}

func dumpFileProcess() {
	if md5Show || md5Action {
		md5Map.Range(func(key, value interface{}) bool {
			md5 := key.(string)
			files := value.([]string)
			if len(files) > 1 {
				dumpMap[md5] = files
				minPhoto := ""
				for _, photo := range files {
					if minPhoto == "" {
						minPhoto = photo
					} else {
						if getDirDate(minPhoto) > getDirDate(photo) {
							minPhoto = photo
						} else if getDirDate(minPhoto) < getDirDate(photo) {

						} else {
							if path.Base(minPhoto) > path.Base(photo) {
								minPhoto = photo
							}
						}
					}
				}

				fmt.Println("file : ", tools.StrWithColor(md5, "blue"))
				for _, photo := range files {
					if photo != minPhoto {
						shouldDeleteFiles = append(shouldDeleteFiles, photo)
						fmt.Println("choose : ", photo, tools.StrWithColor("DELETE", "red"))
					} else {
						fmt.Println("choose : ", photo, tools.StrWithColor("SAVE", "green"))
					}
				}
				fmt.Println()

			}
			return true
		})

		if md5Action {
			for _, photo := range shouldDeleteFiles {
				tools.DeleteFile(photo)
				fmt.Println(tools.StrWithColor("dump file deleted : ", "red"), photo)
			}
		}
	}
}

func processOneFile(photo string) {

	suffix := strings.ToLower(path.Ext(photo))

	shootDate := ""
	if suffix != ".heic" && suffix != ".mov" && suffix != ".mp4" && suffix != ".png" {
		shootDate, _ = getShootDateMethod2(photo, suffix)
		if shootDate != "" {
			//fmt.Println("shootDate : " + shootDate)

		}
	}

	dirDate := getDirDate(photo)

	fileDate := getFileDate(photo)
	if fileDate != "" {
		fileDateFileList.Add(photo)
	}

	modifyDate := getModifyDate(photo)

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
		dirDateFileList.Add(photo)
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

	if md5Show || md5Action {
		md5, _ := tools.GetFileMD5(photo)
		md5MapMu.Lock()
		if value, ok := md5Map.Load(md5); ok {
			md5Map.Store(md5, append(value.([]string), photo))
		} else {
			md5Map.Store(md5, []string{photo})
		}
		md5MapMu.Unlock()
	}

	if flag {
		processFileListMu.Lock()
		processFileList = append(processFileList, ps)
		processFileListMu.Unlock()
	}

	wg.Done()

}

func getShootDateMethod2(path string, suffix string) (string, error) {

	defer func() {
		if r := recover(); r != nil {
			//fmt.Println("Recovered. Error:\n", r)
		}
	}()

	f, err := os.Open(path)
	if err != nil {
		log.Print(err)
		return "", err
	}

	// Optionally register camera makenote data parsing - currently Nikon and
	// Canon are supported.
	exif.RegisterParsers(mknote.All...)

	x, err := exif.Decode(f)
	if err != nil {
		//log.Print(err)
		if value, ok := nost1FileSuffixMap.Load(suffix); ok {
			nost1FileSuffixMap.Store(suffix, value.(int)+1)
		} else {
			nost1FileSuffixMap.Store(suffix, 1)
		}
		return "", err
	}

	shootTime, err := x.DateTime()

	if err != nil {
		if value, ok := nost2FileSuffixMap.Load(suffix); ok {
			nost2FileSuffixMap.Store(suffix, value.(int)+1)
		} else {
			nost2FileSuffixMap.Store(suffix, 1)
		}
		return "", errors.New("no shoot time")
	} else {
		shootTimeStr := shootTime.Format("2006-01-02")
		//shootTimeStr := shootTime.Format("2006-01-02 15:04:05")
		return shootTimeStr, nil
	}

}

func printDate(photo string, dirDate string, modifyDate string, shootDate string, fileDate string, minDate string) {
	if dirDate != minDate {
		fmt.Println("dirDate : ", tools.StrWithColor(dirDate, "red"))
	} else {
		fmt.Println("dirDate : ", tools.StrWithColor(dirDate, "green"))
	}
	if modifyDate != minDate {
		fmt.Println("modifyDate : ", tools.StrWithColor(modifyDate, "red"))
	} else {
		fmt.Println("modifyDate : ", tools.StrWithColor(modifyDate, "green"))
	}
	if shootDate != minDate {
		fmt.Println("shootDate : ", tools.StrWithColor(shootDate, "red"))
	} else {
		fmt.Println("shootDate : ", tools.StrWithColor(shootDate, "green"))
	}
	fmt.Println("minDate : ", tools.StrWithColor(minDate, "green"))
}

func getDirDate(photo string) string {
	parentDir := filepath.Dir(photo)
	dirDate := path.Base(parentDir)
	return dirDate
}

func getFileDate(photo string) string {
	filename := path.Base(photo)

	var fileDate string
	for i, v := range timePatternArray {
		if match := v.FindStringSubmatch(filename); match != nil {
			stamp, _ := time.ParseInLocation(timeTemplateArray[i], match[1], time.Local)
			fileDate = stamp.Format("2006-01-02")
		}
	}
	return fileDate

}

func getModifyDate(photo string) string {
	fileInfo, err := os.Stat(photo)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	modify := fileInfo.ModTime()
	modifyDate := modify.Format("2006-01-02")
	return modifyDate
}

func changeModifyDate(photo string, time time.Time) {
	os.Chtimes(photo, time, time)
}
