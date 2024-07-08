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

	const scanUuidFinal = "2024-07-02-21-37-34_3cf10ae4387811ef8e5a265653c10cb8"

	fmt.Println()
	filePath := cons.WorkDir + "/log/dump_delete_file/" + scanUuidFinal + "/dump_delete_list"
	fmt.Println("file path : ", filePath)

	//service.DeleteMD5DupFilesByJson(filePath)
	service.DeleteMD5DupFilesByLine(filePath)

}
