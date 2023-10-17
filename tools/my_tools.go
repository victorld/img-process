package tools

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

func StrWithColor(str string, color string) string {
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

func GetFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func MoveFile(src string, dst string) {

	// 移动文件
	os.Rename(src, dst)
}

func DeleteFile(filePath string) {

	// 删除文件
	os.Remove(filePath)

}

func DeleteEmptyDir(filePath string) {

	for {
		if flag, err := IsEmpty(filePath); err == nil && flag {
			os.Remove(filePath)
			fmt.Println("remove dir : ", filePath)
			parentDir := filepath.Dir(filePath)
			DeleteEmptyDir(parentDir)
		} else {
			break
		}
	}
	return

}

func IsEmpty(filePath string) (bool, error) {

	dir, err := os.ReadDir(filePath)
	if err != nil {
		return false, err
	}
	if len(dir) == 0 {
		return true, nil
	} else {
		return false, nil
	}

}

func GetSyncMapLens(sm sync.Map) int {
	len := 0
	sm.Range(func(k, v interface{}) bool {
		len++
		return true
	})
	return len
}
