package main

import (
	"fmt"
	"img_process/cons"
	"img_process/service"
	"img_process/tools"
)

func main() {
	tools.InitLogger()
	tools.InitViper()
	cons.InitConst()

	const scanUuidFinal = "2024-02-21-11-20-19_23ceef18d06811ee96e7acde48001122"

	fmt.Println()
	filePath := cons.WorkDir + "/log/dump_delete_file/" + scanUuidFinal + "/dump_delete_list"
	fmt.Println("file path : ", filePath)

	//service.DeleteMD5DupFilesByJson(filePath)
	service.DeleteMD5DupFilesByLine(filePath)

}
