package main

import (
	"fmt"
	"img_process/cons"
	"img_process/service"
	"img_process/tools"
)

// 删除重复文件
func main() {
	tools.InitLogger()
	tools.InitViper()
	cons.InitConst()

	const scanUuidFinal = "2025-01-25-20-07-24_f0530738db1411ef97c02656"

	fmt.Println()
	filePath := cons.WorkDir + "/log/dump_delete_file/" + scanUuidFinal + "/dump_delete_list"
	fmt.Println("file path : ", filePath)

	//service.DeleteMD5DupFilesByJson(filePath)
	service.DeleteMD5DupFilesByLine(filePath)

}
