package main

import (
	"fmt"
	"img_process/service"
)

func main() {

	const fileMD5 = "01d530b4877211ee9a78acde48001122"

	fmt.Println()
	filePath := "/tmp/" + fileMD5
	fmt.Println("file path : ", filePath)

	service.DeleteMD5DupFiles(filePath)

}
