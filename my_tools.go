package main

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

func strWithColor(str string, color string) string {
	if color == "red" {
		str = "\033[31m" + str + "\033[0m"
	} else if color == "green" {
		str = "\033[32m" + str + "\033[0m"
	} else if color == "yellow" {
		str = "\033[33m" + str + "\033[0m"
	} else if color == "blue" {
		str = "\033[34m" + str + "\033[0m"
	} else {

	}
	return str
}

func getFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	hash := md5.New()
	_, _ = io.Copy(hash, file)
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func moveFile(src string, dst string) {

	// 移动文件
	os.Rename(src, dst)
}

func deleteFile(filePath string) {

	// 删除文件
	os.Remove(filePath)

}

func isEmpty(filePath string) bool {

	dir, _ := os.ReadDir(filePath)
	if len(dir) == 0 {
		return true
	} else {
		return false
	}

}
