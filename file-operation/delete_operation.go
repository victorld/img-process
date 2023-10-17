package main

import (
	"encoding/json"
	"fmt"
	"img_process/tools"
)

var fileMD5 = "2e8c25d66cd311eeaf2ee2b55ff2d813"

func deleteMD5DupFiles(filePath string) {
	fileContent2, err := tools.ReadFileString(filePath)
	if err != nil {
		return
	}
	var shouldDeleteFiles []string
	json.Unmarshal([]byte(fileContent2), &shouldDeleteFiles)
	for _, photo := range shouldDeleteFiles {
		tools.DeleteFile(photo)
		fmt.Println(tools.StrWithColor("dump file deleted : ", "red"), photo)
	}
}

func main() {

	fmt.Println()
	filePath := "/tmp/" + fileMD5
	fmt.Println("file path : ", filePath)
	deleteMD5DupFiles(filePath)

}
