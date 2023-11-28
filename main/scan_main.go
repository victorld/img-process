package main

import (
	"encoding/json"
	"fmt"
	"img_process/dao"
	"img_process/model"
	"img_process/service"
	"img_process/tools"
)

func main() {

	tools.InitLogger()
	viper := tools.InitViper()
	tools.InitMysql(viper)

	const deleteShow = true     //是否统计并显示非法文件和空目录
	const moveFileShow = true   //是否统计并显示需要移动目录的文件
	const modifyDateShow = true //是否统计并显示需要修改日期的文件
	const md5Show = true        //是否统计并显示重复文件

	const deleteAction = false     //是否操作删除非法文件和空目录
	const moveFileAction = false   //是否操作需要移动目录的文件
	const modifyDateAction = false //是否操作修改日期的文件

	scanArgs := service.ScanArgs{deleteShow, moveFileShow, modifyDateShow, md5Show, deleteAction, moveFileAction, modifyDateAction}

	imgRecordString, err := service.DoScan(scanArgs)
	if err != nil {
		fmt.Println("scan result error : ", err)
	}

	var imgRecord service.ImgRecord
	json.Unmarshal([]byte(imgRecordString), &imgRecord)

	var imgRecordDB model.ImgRecordDB
	json.Unmarshal([]byte(imgRecordString), &imgRecordDB)

	imgRecordDB.SuffixMap = tools.MarshalPrint(imgRecord.SuffixMap)
	imgRecordDB.YearMap = tools.MarshalPrint(imgRecord.YearMap)
	imgRecordDB.DumpFileDeleteList = tools.MarshalPrint(imgRecord.DumpFileDeleteList)
	imgRecordDB.ExifErr1Map = tools.MarshalPrint(imgRecord.ExifErr1Map)
	imgRecordDB.ExifErr2Map = tools.MarshalPrint(imgRecord.ExifErr2Map)
	imgRecordDB.ExifErr3Map = tools.MarshalPrint(imgRecord.ExifErr3Map)

	var imgRecordService = dao.ImgRecordService{}
	/*if err = imgRecordService.RegisterImgRecord(&imgRecordDB); err != nil {
		fmt.Println("register error : ", err)
		return
	}*/

	if err = imgRecordService.CreateImgRecord(&imgRecordDB); err != nil {
		fmt.Println("create error : ", err)
		return
	}

}
