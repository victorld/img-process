package main

import (
	"fmt"
	"img_process/service"
)

func main() {

	const deleteShow = true     //是否统计并显示非法文件和空目录
	const moveFileShow = true   //是否统计并显示需要移动目录的文件
	const modifyDateShow = true //是否统计并显示需要修改日期的文件
	const md5Show = true        //是否统计并显示重复文件

	const deleteAction = false     //是否操作删除非法文件和空目录
	const moveFileAction = false   //是否操作需要移动目录的文件
	const modifyDateAction = false //是否操作修改日期的文件

	var scanResult = service.DoScan(deleteShow, moveFileShow, modifyDateShow, md5Show, deleteAction, moveFileAction, modifyDateAction)
	fmt.Println("scan result : ", scanResult)

}
