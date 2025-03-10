package main

import (
	"fmt"
	"img_process/middleware"
	"img_process/tools"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func testGetMD5() {
	v, _ := tools.GetFileMD5("/Users/ld/my-file/temp/desca (2).crt", 0)
	fmt.Println("md5 : ", v)
	//fmt.Println("md5 : ", tools.GetFileMD5("/Users/ld/my-file/temp/Docker.dmg"))

}

func testDate() {
	//获取本地location
	toBeCharge := "2015-01-01 00:00:00"                             //待转化为时间戳的字符串 注意 这里的小时和分钟还要秒必须写 因为是跟着模板走的 修改模板的话也可以不写
	timeLayout := "2006-01-02 15:04:05"                             //转化所需模板
	loc, _ := time.LoadLocation("Local")                            //重要：获取时区
	theTime, _ := time.ParseInLocation(timeLayout, toBeCharge, loc) //使用模板在对应时区转化为time.time类型
	sr := theTime.Unix()                                            //转化为时间戳 类型是int64
	fmt.Println(theTime)                                            //打印输出theTime 2015-01-01 15:15:00 +0800 CST
	fmt.Println(sr)                                                 //打印输出时间戳 1420041600

	//时间戳转日期
	dataTimeStr := time.Unix(sr, 0).Format(timeLayout) //设置时间戳 使用模板格式化为日期字符串
	fmt.Println(dataTimeStr)

}

func testModifyDate() {
	file := "/Users/ld/Desktop/pic-new/2023/2023-09/2023-09-08/11028_1670298127.mp4"
	fmt.Println("file modify date change before : ", tools.GetModifyDate(file))

	tm, _ := time.Parse("2006-01-02", "2023-09-08")
	tools.ChangeModifyDate(file, tm)

	fmt.Println("file modify date change after : ", tools.GetModifyDate(file))

}

func testChan() {

	messages := make(chan string)
	go func() {
		println("waiting fot get message from chan")
		messages <- "ping"
	}()
	println("start")
	time.Sleep(3 * time.Second)
	msg := <-messages
	println("already get message from chan")
	fmt.Println(msg)
}

func testGetExifInfo() {
	file := "/Users/ld/Desktop/IMG_0112.JPG"
	shootTime, locNum, state, output, err := middleware.GetExifInfo(file)
	if err != nil {
		tools.FancyHandleError(err)
	} else {
		fmt.Println("shootTime", shootTime)
		fmt.Println("locNum", locNum)
		fmt.Println("state", state)
		fmt.Println("output", output)
	}

	//tools.TestGetShootDate(file)

}

func getLocationAddress() {
	middleware.CreateGisDatabaseCache()
	//address, err := middleware.GetLocationAddressByCache("116.310454,39.992734")
	address, err := middleware.GetLocationAddressByCache("30.559343,114.279656")
	//address, err := middleware.GetLocationAddressByCache("114.279656,30.559343")
	if err != nil {
		tools.FancyHandleError(err)
	} else {
		fmt.Println("address : ", address)
	}
}

func getExifInfoCommand() {
	file := "/Users/ld/Downloads/save/pic-lib/pic-new/2023/2023-08/2023-08-23/IMG_8197.MOV"
	shootTime, locNum, output, err := middleware.GetExifInfoCommand(file)
	if err != nil {
		tools.FancyHandleError(err)
	} else {
		fmt.Println("shootTime", shootTime)
		fmt.Println("locNum", locNum)
		fmt.Println("output", output)
	}
}

func extractFileInfo() {
	file := "/Users/ld/Downloads/save/pic-lib/pic-new/2023/2023-08/2023-08-23/IMG_8197[dda].MOV"
	fileRegexp := regexp.MustCompile(`^.*\[(.*)\].*$`)
	dateValList := fileRegexp.FindStringSubmatch(file)
	var timeAndLoc string
	if len(dateValList) == 2 {
		timeAndLoc = dateValList[1]
	}
	fmt.Println(timeAndLoc)
}

func changeFileName() {
	photo := "/Users/ld/Downloads/save/pic-lib/pic-new/2023/2023-08/2023-08-23/IMG_8197.pic.MOV"
	var photoNew string
	timeAndLocShould := "2010:04:11-00:00:00|北京市海淀区上地街道"
	fileName := filepath.Base(photo)
	//fileSuffix := strings.ToLower(path.Ext(filePath))
	//// 去除文件扩展名
	//nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(filePath))
	//// 获取文件的父目录
	//parentDir := filepath.Dir(filePath)
	//fmt.Println(fileSuffix, nameWithoutExt, parentDir)

	if strings.Count(photo, "[") == 1 && strings.Count(photo, "]") == 1 {
		re, _ := regexp.Compile(`\[.*\]`)
		photoNew = re.ReplaceAllString(photo, "["+timeAndLocShould+"]")
	} else if strings.Count(photo, "[") == 0 && strings.Count(photo, "]") == 0 {
		if strings.Count(photo, ".") == 1 {
			photoNew = strings.ReplaceAll(photo, ".", "["+timeAndLocShould+"].")
		} else {
			fileSuffix := strings.ToLower(path.Ext(photo))                                                               //文件后缀
			nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(photo))                                          // 去除文件扩展名
			parentDir := filepath.Dir(photo)                                                                             // 获取文件的父目录
			photoNew = parentDir + string(filepath.Separator) + strings.ReplaceAll(nameWithoutExt, ".", "") + fileSuffix //去除文件名里的.
			photoNew = strings.ReplaceAll(photoNew, ".", "["+timeAndLocShould+"].")
			tools.Logger.Info("##################filePath with . , photo : ", photo, " photoNew : ", photoNew)
		}
	} else {
		tools.Logger.Error("##################filePath [] error , photo : ", photo)
	}

	fmt.Println(photoNew)

}

func main() {
	fmt.Println()

	//tools.InitLogger()
	//tools.InitViper()
	//cons.InitConst()
	//orm.InitMysql()

	changeFileName()
}
