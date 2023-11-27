package main

import (
	"encoding/json"
	"fmt"
	"img_process/tools"
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

func testMd5Delete() {
	fileContent := ` ["/Users/ld/Desktop/pic-new/2023/2023-05/2023-05-15/VVFR1089.JPG","/Users/ld/Desktop/pic-new/2023/2023-07/2023-07-22/IMG_7834.JPG","/Users/ld/Desktop/pic-new/2023/2023-05/2023-05-15/VPLK2407.JPG","/Users/ld/Desktop/pic-new/2023/2023-07/2023-07-22/IMG_9801.HEIC","/Users/ld/Desktop/pic-new/2023/2023-07/2023-07-22/IMG_7839.HEIC","/Users/ld/Desktop/pic-new/2008/2008-12/2008-12-01/31-10-08_0835的副本.jpg","/Users/ld/Desktop/pic-new/200808-12/2008-12-01/31-10-08_0835.jpg","/Users/ld/Desktop/pic-new/2023/2023-05/2023-05-15/PRIZ6173.JPG","/Users/ld/Desktop/pic-new/2023/2023-05/2023-05-15/IMG_7334.JPG","/Users/ld/Desktop/pic-new/2023/2023-05/2023-05-15/MTIF4266.JPG","/Users/ld/Desktop/pic-new/2023/2023-05/2023-05-15/IMG_7335.JPG","/Users/ld/Desktop/pic-new/2023/2023-07/2023-07-22/IMG_9801.MOV","/Users/ld/Desktop/pic-new/2023/2023-07/2023-07-22/IMG_7839.MOV","/Users/ld/Desktop/pic-new/2023/2023-05/2023-05-15/IMG_7333.JPG","/Users/ld/Desktop/pic-new/2023/2023-07/2023-07-22/IMG_9800.PNG","/Users/ld/Desktop/pic-new/2023/2023-05/2023-05-15/VNTV3378.JPG","/Users/ld/Desktop/pic-new/2023/2023-07/2023-07-22/IMG_E7839.MOV","/Users/ld/Desktop/pic-new/2023/2023-05/2023-05-15/EVAI2226.JPG","/Users/ld/Desktop/pic-new/2008/2008-11/2008-11-19/19-11-08_1139.jpg","/Users/ld/Desktop/pic-new/2023/2023-05/2023-05-15/QYEE5834.JPG"]`
	fileUuid, err := tools.WriteStringToUuidFile(fileContent)
	if err != nil {
		return
	}
	filePath := "/tmp/" + fileUuid
	fmt.Println("file path : ", filePath)
	fileContent2, err := tools.ReadFileString(filePath)
	if err != nil {
		return
	}
	var shouldDeleteFiles []string
	json.Unmarshal([]byte(fileContent2), &shouldDeleteFiles)
	for _, photo := range shouldDeleteFiles {
		//tools.DeleteFile(photo)
		fmt.Println(tools.StrWithColor("dump file deleted : ", "red"), photo)
	}
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

func testViper() {
	viper := tools.InitViper()
	//打印文件读取出来的内容:
	fmt.Println(viper.Get("database.username"))
	fmt.Println(viper.Get("database.password"))
	fmt.Println(viper.Get("database.host"))
	fmt.Println(viper.Get("database.port"))
	fmt.Println(viper.Get("database.dbname"))
	fmt.Println(viper.Get("database.config"))

}
func main() {
	fmt.Println()

	//testDate()
	//testGetMD5()
	//testMd5Delete()
	//testModifyDate()
	//testChan()
	testViper()
}
