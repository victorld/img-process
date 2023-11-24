package main

import (
	"fmt"
	"img_process/service"
)

func main() {

	const fileMD5 = "abcd"

	fmt.Println()
	filePath := "/tmp/" + fileMD5
	fmt.Println("file path : ", filePath)

	service.DeleteMD5DupFilesByJson(filePath)
	//service.DeleteMD5DupFilesByLine(filePath)

}
