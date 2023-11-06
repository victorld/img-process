package main

import (
	"errors"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"img_process/tools"
	"log"
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
// const startPath = "/Volumes/ld_hardone/pic-new"
//const startPath = "/Volumes/ld_hardraid/old-pic/pic-new"

const poolSize = 8 //并行处理的线程
const md5Retry = 3 //文件md5计算重试次数

const deleteShow = true     //是否统计并显示非法文件和空目录
const dirDateShow = true    //是否统计并显示需要移动目录的文件
const modifyDateShow = true //是否统计并显示需要修改日期的文件
const md5Show = true        //是否统计并显示重复文件

const deleteAction = false     //是否操作删除非法文件和空目录
const dirDateAction = false    //是否操作需要移动目录的文件
const modifyDateAction = false //是否操作修改日期的文件

var basePath = startPath[0 : strings.Index(startPath, "pic-new")+7] //指向pic-new的目录

var suffixMap = map[string]int{} //后缀统计
var yearMap = map[string]int{}   //年份统计

var fileTotalCnt = 0 //文件总量
var dirTotalCnt = 0  //目录总量

var fileDateFileList = mapset.NewSet() //文件名带日期的照片

var deleteFileList = mapset.NewSet()     //需要删除的文件
var dirDateFileList = mapset.NewSet()    //目录与最小日期不匹配，需要移动
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

func (ps *photoStruct) psPrint() { //打印照片相关信息
	if ps.dirDate != ps.minDate {
		fmt.Println("dirDate : ", tools.StrWithColor(ps.dirDate, "red"))
	} else {
		fmt.Println("dirDate : ", tools.StrWithColor(ps.dirDate, "green"))
	}
	if ps.modifyDate != ps.minDate {
		fmt.Println("modifyDate : ", tools.StrWithColor(ps.modifyDate, "red"))
	} else {
		fmt.Println("modifyDate : ", tools.StrWithColor(ps.modifyDate, "green"))
	}
	if ps.shootDate != ps.minDate {
		fmt.Println("shootDate : ", tools.StrWithColor(ps.shootDate, "red"))
	} else {
		fmt.Println("shootDate : ", tools.StrWithColor(ps.shootDate, "green"))
	}
	fmt.Println("minDate : ", tools.StrWithColor(ps.minDate, "green"))
}

var processDirList []dirStruct    //需要处理的目录结构体列表（空目录）
var processFileList []photoStruct //需要处理的文件结构体列表（非法格式删除、移动、修改时间、重复文件删除）
var processFileListMu sync.Mutex

var md5Map = make(map[string][]string) //以md5为key存储文件
var md5MapMu sync.Mutex

var nost1FileSuffixMap = map[string]int{} //shoot time error1后缀
var nost1FileSet = mapset.NewSet()        //shoot time error1照片
var nost2FileSuffixMap = map[string]int{} //shoot time error2后缀
var nost2FileSet = mapset.NewSet()        //shoot time error2照片
var nost3FileSuffixMap = map[string]int{} //shoot time error3后缀
var nost3FileSet = mapset.NewSet()        //shoot time error3照片
var nost1FileMu sync.Mutex
var nost2FileMu sync.Mutex
var nost3FileMu sync.Mutex

var md5EmptyFileList []string //获取md5为空的文件
var md5EmptyFileListMu sync.Mutex

var wg sync.WaitGroup //异步照片处理等待

func main() {

	start := time.Now() // 获取当前时间

	fmt.Println("startPath : ", startPath)
	fmt.Println("basePath : ", basePath)

	println()

	fmt.Println(tools.StrWithColor("==========ROUND 1: SCAN FILE==========", "red"))
	fmt.Println()

	p, _ := ants.NewPool(poolSize) //新建一个pool对象
	defer p.Release()

	_ = filepath.Walk(startPath, func(file string, info os.FileInfo, err error) error {
		if info.IsDir() { //遍历目录
			if flag, err := tools.IsEmpty(file); err == nil && flag { //空目录加入待处理列表
				ds := dirStruct{isEmptyDir: true, dir: file}
				processDirList = append(processDirList, ds)

			}
			dirTotalCnt = dirTotalCnt + 1
		} else { //遍历文件
			//fmt.Println(file)
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
					processOneFile(file) //单个文件协程处理
				})

				if value, ok := suffixMap[fileSuffix]; ok { //统计文件的后缀
					suffixMap[fileSuffix] = value + 1
				} else {
					suffixMap[fileSuffix] = 1
				}

				dirDate := tools.GetDirDate(file)
				year := dirDate[0:4]

				if value, ok := yearMap[year]; ok { //统计照片年份
					yearMap[year] = value + 1
				} else {
					yearMap[year] = 1
				}

				fileTotalCnt = fileTotalCnt + 1
				if fileTotalCnt%100 == 0 { //每隔100行打印一次
					println("processed ", tools.StrWithColor(strconv.Itoa(fileTotalCnt), "red"))
					println("pool running size : ", p.Running())
				}
			}
		}
		return nil
	})
	fmt.Println("processed(end)", tools.StrWithColor(strconv.Itoa(fileTotalCnt), "red"))

	wg.Wait()

	elapsed := time.Since(start)

	start2 := time.Now() // 获取当前时间

	fmt.Println()
	fmt.Println(tools.StrWithColor("==========ROUND 2: PROCESS FILE==========", "red"))
	fmt.Println()
	fmt.Println(tools.StrWithColor("PRINT DETAIL TYPE1(delete file,modify date,move file): ", "red"))
	for _, ps := range processFileList { //第一个参数是下标

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
	fmt.Println()

	fmt.Println(tools.StrWithColor("PRINT DETAIL TYPE3(dump file): ", "red"))
	dumpMap := dumpFileProcess() //5、重复文件处理处理

	fmt.Println()
	fmt.Println(tools.StrWithColor("PRINT STAT TYPE0(comman info): ", "red"))
	fmt.Println("suffixMap : ", tools.MarshalPrint(suffixMap))
	fmt.Println("yearMap : ", tools.MarshalPrint(yearMap))
	fmt.Println("file total : ", tools.StrWithColor(strconv.Itoa(fileTotalCnt), "red"))
	fmt.Println("dir total : ", tools.StrWithColor(strconv.Itoa(dirTotalCnt), "red"))
	fmt.Println("file contain date(just for print) : ", tools.StrWithColor(strconv.Itoa(fileDateFileList.Cardinality()), "red"))
	fmt.Println("exif parse error 1 : ", tools.StrWithColor(tools.MarshalPrint(nost1FileSuffixMap), "red"))
	fmt.Println("exif parse error 1 : ", tools.StrWithColor(strconv.Itoa(nost1FileSet.Cardinality()), "red"))
	//fmt.Println("exif parse error 1 list : ", nost1FileSet)
	fmt.Println("exif parse error 2 : ", tools.StrWithColor(tools.MarshalPrint(nost2FileSuffixMap), "red"))
	fmt.Println("exif parse error 2 : ", tools.StrWithColor(strconv.Itoa(nost2FileSet.Cardinality()), "red"))
	//fmt.Println("exif parse error 2 list : ", nost2FileSet)
	fmt.Println("exif parse error 3 : ", tools.StrWithColor(tools.MarshalPrint(nost3FileSuffixMap), "red"))
	fmt.Println("exif parse error 3 : ", tools.StrWithColor(strconv.Itoa(nost3FileSet.Cardinality()), "red"))
	//fmt.Println("exif parse error 3 list : ", nost3FileSet)

	fmt.Println()
	fmt.Println(tools.StrWithColor("PRINT STAT TYPE1(delete file,modify date,move file): ", "red"))
	fmt.Println("delete file total : ", tools.StrWithColor(strconv.Itoa(deleteFileList.Cardinality()), "red"))
	fmt.Println("modify date total : ", tools.StrWithColor(strconv.Itoa(modifyDateFileList.Cardinality()), "red"))
	fmt.Println("move file total : ", tools.StrWithColor(strconv.Itoa(dirDateFileList.Cardinality()), "red"))
	fmt.Println("shoot date total : ", tools.StrWithColor(strconv.Itoa(shootDateFileList.Cardinality()), "red"))

	fmt.Println()
	fmt.Println(tools.StrWithColor("PRINT STAT TYPE2(empty dir) : ", "red"))
	fmt.Println("empty dir total : ", tools.StrWithColor(strconv.Itoa(len(processDirList)), "red"))

	fmt.Println()
	fmt.Println(tools.StrWithColor("PRINT STAT TYPE3(dump file) : ", "red"))
	fmt.Println("dump file total : ", tools.StrWithColor(strconv.Itoa(len(dumpMap)), "red"))

	fmt.Println("shouldDeleteMd5Files length : ", tools.StrWithColor(strconv.Itoa(len(shouldDeleteMd5Files)), "red"))
	if len(shouldDeleteMd5Files) != 0 {
		sm3 := tools.MarshalPrint(shouldDeleteMd5Files)
		fmt.Println("shouldDeleteMd5Files print origin : ", sm3)
		fileUuid, err := tools.WriteStringToFile(sm3)
		if err != nil {
			return
		}
		filePath := "/tmp/" + fileUuid
		//fmt.Println("file path : ", filePath)
		fileContent2, err := tools.ReadFileString(filePath)
		if err != nil {
			return
		}
		fmt.Println("shouldDeleteMd5Files print reread : ", fileContent2)
		fmt.Println("tmp file md5 : ", tools.StrWithColor(fileUuid, "red"))
	}
	fmt.Println("md5 get error length : ", tools.StrWithColor(strconv.Itoa(len(md5EmptyFileList)), "red"))
	if len(md5EmptyFileList) != 0 {
		fmt.Println("md5EmptyFileList : ", tools.MarshalPrint(md5EmptyFileList))
	}

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
			ps.psPrint()
			*printDateFlag = true
		}
		fmt.Println(tools.StrWithColor("should modify file ", "yellow"), ps.photo, "modifyDate to", ps.minDate)
	}
	if modifyDateAction {
		tm, _ := time.Parse("2006-01-02", ps.minDate)
		tools.ChangeModifyDate(ps.photo, tm)
		fmt.Println(tools.StrWithColor("modify file ", "yellow"), ps.photo, "modifyDate to", ps.minDate, "get realdate", tools.GetModifyDate(ps.photo))
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
			ps.psPrint()
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

func dumpFileProcess() map[string][]string {
	var dumpMap = make(map[string][]string) //md5Map里筛选出有重复文件的Map

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

				fmt.Println("file : ", tools.StrWithColor(md5, "blue"))
				for _, photo := range files {
					if photo != minPhoto {
						shouldDeleteMd5Files = append(shouldDeleteMd5Files, photo)
						fmt.Println("choose : ", photo, tools.StrWithColor("DELETE", "red"))
					} else {
						fmt.Println("choose : ", photo, tools.StrWithColor("SAVE", "green"))
					}
				}
				fmt.Println()

			}
		}

	}
	return dumpMap
}

func processOneFile(photo string) {

	defer wg.Done()

	suffix := strings.ToLower(path.Ext(photo))

	shootDate := ""
	if suffix != ".heic" && suffix != ".mov" && suffix != ".mp4" && suffix != ".png" { //exif拍摄时间获取
		shootDate, _ = getShootDateMethod2(photo, suffix)
		if shootDate != "" {
			//fmt.Println("shootDate : " + shootDate)
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

	if md5Show { //如果需要计算md5，则把所有照片按照md5整理
		md5, err := tools.GetFileMD5WithRetry(photo, md5Retry)
		if err != nil {
			log.Print("GetFileMD5 err for ", md5Retry, " times : ", err)
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
			//fmt.Println("Recovered. Error:\n", r)
			nost3FileMu.Lock()
			if value, ok := nost3FileSuffixMap[suffix]; ok {
				nost3FileSuffixMap[suffix] = value + 1
			} else {
				nost3FileSuffixMap[suffix] = 1
			}
			nost3FileSet.Add(path)
			nost3FileMu.Unlock()
		}
		f.Close()
	}()

	if err != nil {
		fmt.Print(err)
		return "", err
	}

	// Optionally register camera makenote data parsing - currently Nikon and Canon are supported.
	exif.RegisterParsers(mknote.All...)

	x, err := exif.Decode(f)
	if err != nil {
		//log.Print(err)
		nost1FileMu.Lock()
		if value, ok := nost1FileSuffixMap[suffix]; ok {
			nost1FileSuffixMap[suffix] = value + 1
		} else {
			nost1FileSuffixMap[suffix] = 1
		}
		nost1FileSet.Add(path)
		nost1FileMu.Unlock()

		return "", errors.New("exif decode error")
	}

	shootTime, err := x.DateTime()

	if err != nil {
		nost2FileMu.Lock()
		if value, ok := nost2FileSuffixMap[suffix]; ok {
			nost2FileSuffixMap[suffix] = value + 1
		} else {
			nost2FileSuffixMap[suffix] = 1
		}
		nost2FileSet.Add(path)
		nost2FileMu.Unlock()

		return "", errors.New("no shoot time")
	} else {
		shootTimeStr := shootTime.Format("2006-01-02")
		//shootTimeStr := shootTime.Format("2006-01-02 15:04:05")
		return shootTimeStr, nil
	}

}
