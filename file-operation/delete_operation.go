package main

import (
	"encoding/json"
	"fmt"
	"img_process/tools"
	"strconv"
)

var fileMD5 = "f14d431c7fd811eeb4d7265653c10cb8"

func deleteMD5DupFiles(filePath string) {
	fileContent2, err := tools.ReadFileString(filePath)
	if err != nil {
		return
	}
	var shouldDeleteFiles []string
	json.Unmarshal([]byte(fileContent2), &shouldDeleteFiles)
	count := 0
	for _, photo := range shouldDeleteFiles {
		fmt.Println("file : " + photo)
		err := tools.DeleteFile(photo)
		if err != nil {
			fmt.Println(tools.StrWithColor("delete file failed , reason : "+err.Error(), "red"))
		} else {
			fmt.Println(tools.StrWithColor("dump file deleted : ", "green"), photo)
			count++
		}
	}
	fmt.Println()
	fmt.Print(tools.StrWithColor("dump file total : ", "red"), strconv.Itoa(len(shouldDeleteFiles)))
	fmt.Print(tools.StrWithColor(" dump file deleted total : ", "red"), strconv.Itoa(count))
	fmt.Println()

}

func main() {

	fmt.Println()
	filePath := "/tmp/" + fileMD5
	fmt.Println("file path : ", filePath)
	deleteMD5DupFiles(filePath)

}
