package main

import (
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

func testShootDate() {
	file := "/Users/ld/Desktop/IMG_0112.JPG"
	shootTime, err := tools.GetExifDateTime(file)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(shootTime.Format("2006-01-02"))
	}

	//tools.TestGetShootDate(file)

}

func testPrintExifData() {
	file := "/Users/ld/Desktop/IMG_0112.JPG"
	tools.PrintExifData(file)
}

func main() {
	fmt.Println()

	//testShootDate()
	testPrintExifData()
	//testDate()
	//testGetMD5()
	//testMd5Delete()
	//testModifyDate()
	//testChan()
}
