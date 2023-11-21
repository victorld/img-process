package main

import (
	"fmt"
	"img_process/service"
)

func main() {

	const fileMD5 = "96249558882011eeb2e8acde48001122"

	fmt.Println()
	filePath := "/tmp/" + fileMD5
	fmt.Println("file path : ", filePath)

	service.DeleteMD5DupFiles(filePath)

}
