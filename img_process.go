package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	//exif "github.com/dsoprea/go-exif/v3"
)

var deleteShow = true

// var shootDateShow = true
var modifyDateShow = true
var dirDateShow = true
var md5Show = true

var deleteAction = false

// var shootDateAction = false
var modifyDateAction = false
var dirDateAction = false
var md5Action = false

var suffixMap = make(map[string]int)

var dumpMap = make(map[string][]string)
var md5Map = make(map[string][]string)

func showMd5Map() interface{} {
	return md5Map
}

var totalCnt = 0

var fileDateFileList = mapset.NewSet()             //文件名带日期的照片
var dirDateFileList = mapset.NewSet(1, 2, 3, 4)    //目录与最小日期不匹配，需要移动
var modifyDateFileList = mapset.NewSet(1, 2, 3, 4) //修改时间与最小日期不匹配，需要修改
var shootDateFileList = mapset.NewSet(1, 2, 3, 4)  //拍摄时间与最小日期不匹配，需要修改
var deleteFileList = mapset.NewSet()               //需要删除的文件
// var tagList = mapset.NewSet()        //
// var nost1FileList = mapset.NewSet() //shoot time没有的照片
// var nost2FileList = mapset.NewSet() //shoot time没有的照片
var nost3FileList = mapset.NewSet() //shoot time没有的照片

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

func main() {

	start := time.Now() // 获取当前时间
	println()
	rootPath := "/Users/ld/Desktop/pic-new/2008"

	var files []string
	var dirs []string

	err := walkDir(rootPath, &dirs, &files)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		//fmt.Println(file)

		fileName := path.Base(file)
		fileSuffix := path.Ext(file)
		//fileprefix := fileName[0 : len(fileName)-len(fileSuffix)]

		//fmt.Println("fileName : " + fileName)
		//fmt.Println("fileSuffix : " + fileSuffix)
		//fmt.Println("fileprefix : " + fileprefix)

		if strings.HasPrefix(fileName, ".") || strings.HasSuffix(fileName, "nas_downloading") {
			if deleteShow {
				fmt.Println(strWithColor("should delete file :", "yellow"), file)
			}

			if deleteAction {
				err := os.Remove(file)
				if err != nil {
					println(strWithColor("delete file failed:", "yellow"), file, err)
				} else {
					println(strWithColor("delete file sucessed:", "green"), file)
				}

				deleteFileList.Add(file)
				continue
			}

		}

		processOneFile(file, fileSuffix)

		if value, ok := suffixMap[fileSuffix]; ok {
			suffixMap[fileSuffix] = value + 1
		} else {
			suffixMap[fileSuffix] = 1
		}

		totalCnt = totalCnt + 1

	}

	sm, _ := json.Marshal(suffixMap)
	fmt.Println("suffixMap : ", string(sm))

	fmt.Println("dirDateFileList : ", strWithColor(strconv.Itoa(dirDateFileList.Cardinality()), "red"))
	fmt.Println("modifyDateFileList : ", strWithColor(strconv.Itoa(modifyDateFileList.Cardinality()), "red"))
	fmt.Println("shootDateFileList : ", strWithColor(strconv.Itoa(shootDateFileList.Cardinality()), "red"))
	fmt.Println("deleteFileList : ", strWithColor(strconv.Itoa(deleteFileList.Cardinality()), "red"))
	//fmt.Println("nost1FileList : ", nost1FileList.Cardinality())
	//fmt.Println("nost2FileList : ", nost2FileList.Cardinality())
	fmt.Println("nost3FileList : ", strWithColor(strconv.Itoa(nost3FileList.Cardinality()), "red"))
	fmt.Println("totalCnt : ", totalCnt)
	//fmt.Println("tagList : ", tagList)

	for k, v := range md5Map {
		if len(v) > 1 {
			if md5Show {
				//for _, file := range v {
				//
				//}

			}
			if md5Action {
				//moveFile
				//moveFile(photo, targetDir)
				//fmt.Println("move file ", photo, "to", targetDir)
			}
			dumpMap[k] = v
		}
	}
	sm2, _ := json.Marshal(dumpMap)
	fmt.Println("dumpMap length: ", strWithColor(strconv.Itoa(len(dumpMap)), "red"))
	fmt.Println("dumpMap : ", string(sm2))

	elapsed := time.Since(start)
	fmt.Println("该函数执行完成耗时：", elapsed)

	//expvar.Publish("a_map", expvar.Func(showMd5Map))
	//http.ListenAndServe(":8080", nil)

}

func walkDir(rootPath string, dirs *[]string, files *[]string) error {

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			*dirs = append(*dirs, path)
		} else {
			*files = append(*files, path)
		}
		return nil
	})
	return err

}

func processOneFile(photo string, suffix string) {

	shootDate, _ := getShootDateMethod2(photo)
	if shootDate != "" {
		//fmt.Println("shootDate : " + shootDate)
		shootDateFileList.Add(photo)
	}

	dirDate := getDirDate(photo)

	fileDate := getFileDate(photo)
	if fileDate != "" {
		//fmt.Println("fileDate : " + fileDate)
		fileDateFileList.Add(photo)
	}

	modifyDate := getModifyDate(photo)
	if modifyDate != "" {
		//fmt.Println("modifyDate : " + modifyDate)
		modifyDateFileList.Add(photo)
	}

	minDate := ""

	if dirDate < modifyDate {
		minDate = dirDate
	}
	if shootDate != "" && shootDate < minDate {
		minDate = shootDate
	}
	if fileDate != "" {
		minDate = fileDate
	}

	printDateFlag := false
	if dirDate != minDate {
		dirDateFileList.Add(photo)
		targetDir := filepath.Dir(filepath.Dir(photo)) + string(os.PathSeparator) + minDate
		if dirDateShow {
			printDate(photo, dirDate, modifyDate, shootDate, fileDate, minDate)
			printDateFlag = true
			fmt.Println(strWithColor("should move file ", "yellow"), photo, "to", targetDir)
		}
		if dirDateAction {
			//moveFile
			moveFile(photo, targetDir)
			fmt.Println(strWithColor("move file ", "yellow"), photo, "to", targetDir)
		}
	}

	if modifyDate != minDate {
		tm, _ := time.Parse("2006-01-02", minDate)
		if modifyDateShow {
			if !printDateFlag {
				printDate(photo, dirDate, modifyDate, shootDate, fileDate, minDate)
				printDateFlag = true
			}
			fmt.Println(strWithColor("should modify file ", "yellow"), photo, "modifyDate to", minDate)
		}
		if modifyDateAction {
			changeModifyDate(photo, tm)
			fmt.Println(strWithColor("modify file ", "yellow"), photo, "modifyDate to", minDate, "get realdate", getModifyDate(photo))
		}
	}

	if md5Show || md5Action {
		md5, _ := getFileMD5(photo)
		if value, ok := md5Map[md5]; ok {
			md5Map[md5] = append(value, photo)
		} else {
			md5Map[md5] = []string{photo}
		}
	}

}

/*func ReadExifMethod1(path string) (string, error) {
	var dateList []int64
	opt := exif.ScanOptions{}
	dt, err := exif.SearchFileAndExtractExif(path)
	if err != nil {
		//fmt.Println("photo : ", path)
		//fmt.Println("SearchFileAndExtractExif error : ", err)
		nost1FileList.Add(path)
		return "", err
	}
	ets, _, err := exif.GetFlatExifData(dt, &opt)
	if err != nil {
		//fmt.Println("photo : ", path)
		//fmt.Println("GetFlatExifData error : ", err)
		nost2FileList.Add(path)
		return "", err
	}
	for _, et := range ets {
		if strings.Contains(strings.ToLower(et.TagName), "time") || strings.Contains(strings.ToLower(et.TagName), "date") {
			tagList.Add(et.TagName)
		}
		if et.TagName == "DateTimeDigitized" || et.TagName == "DateTime" || et.TagName == "DateTimeOriginal" {
			for _, v := range timePatternArray {
				if match := v.FindStringSubmatch(et.Value.(string)); match != nil {
					//fmt.Println(match[0])
					stamp, _ := time.ParseInLocation(datetimeTemplate, match[0], time.Local)
					//fmt.Println(strconv.FormatInt(stamp.Unix(), 10))
					dateList = append(dateList, stamp.Unix())
				}
			}
		}
		//fmt.Println(et.TagId, et.TagName, et.TagTypeName, et.Value)

	}

	//dateList = append(dateList, 1225382400)
	//dateList = append(dateList, 1325382400)
	//dateList = append(dateList, 1115382400)
	sort.SliceStable(dateList, func(i, j int) bool {
		return dateList[i] < dateList[j]
	})

	//fmt.Println(dateList)

	if len(dateList) > 0 {
		timeEarly := time.Unix(dateList[0], 0)
		return timeEarly.Format("2006-01-02 15:04:05"), nil
	} else {
		nost3FileList.Add(path)
		return "", errors.New("no shoot time")
	}

}*/

func getShootDateMethod2(path string) (string, error) {

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
		return "", err
	}

	shootTime, err := x.DateTime()

	if err != nil {
		nost3FileList.Add(path)
		return "", errors.New("no shoot time")
	} else {
		shootTimeStr := shootTime.Format("2006-01-02")
		//shootTimeStr := shootTime.Format("2006-01-02 15:04:05")
		return shootTimeStr, nil
	}

}

func printDate(photo string, dirDate string, modifyDate string, shootDate string, fileDate string, minDate string) {
	fmt.Println("photo : ", strWithColor(photo, "blue"))
	if dirDate != minDate {
		fmt.Println("dirDate : ", strWithColor(dirDate, "red"))
	} else {
		fmt.Println("dirDate : ", strWithColor(dirDate, "green"))
	}
	if modifyDate != minDate {
		fmt.Println("modifyDate : ", strWithColor(modifyDate, "red"))
	} else {
		fmt.Println("modifyDate : ", strWithColor(modifyDate, "green"))
	}
	if shootDate != minDate {
		fmt.Println("shootDate : ", strWithColor(shootDate, "red"))
	} else {
		fmt.Println("shootDate : ", strWithColor(shootDate, "green"))
	}
	fmt.Println("minDate : ", strWithColor(minDate, "green"))
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
			//fmt.Println(match[0])
			stamp, _ := time.ParseInLocation(timeTemplateArray[i], match[1], time.Local)
			//fmt.Println(strconv.FormatInt(stamp.Unix(), 10))
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
