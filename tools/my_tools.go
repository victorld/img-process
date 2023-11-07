package tools

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var date1Pattern = regexp.MustCompile("^.*(20[012]\\d}(0[1-9]|1[0-2])(0[1-9]|[1-2]\\d|3[01])).*$")

const Data1Template = "20060102"

var date2Pattern = regexp.MustCompile("^.*((0[1-9]|[1-2]\\d|3[01])-(0[1-9]|1[0-2])-[012]\\d).*$")

const Data2Template = "02-01-06" // 31-12-19
var date3Pattern = regexp.MustCompile("^.*(20[012]\\d:(0[1-9]|1[0-2]):(0[1-9]|[1-2]\\d|3[01])).*$")

const Data3Template = "2006:01:02" //
var date4Pattern = regexp.MustCompile("^.*(20[012]\\d-(0[1-9]|1[0-2])-(0[1-9]|[1-2]\\d|3[01])).*$")

const Data4Template = "2006-01-02" //
var datetimePattern *regexp.Regexp = regexp.MustCompile("^.*(20[012]\\d:(0[1-9]|1[0-2]):(0[1-9]|[1-2]\\d|3[01]) (\\d{2}:\\d{2}:\\d{2})).*$")

const DatetimeTemplate = "2006:01:02 15:04:05"

var timePatternArray = []*regexp.Regexp{date1Pattern, date2Pattern, date3Pattern, date4Pattern, datetimePattern}
var timeTemplateArray = []string{Data1Template, Data2Template, Data3Template, Data4Template, DatetimeTemplate}

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
	defer file.Close()
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

func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func MoveFile(src string, dst string) {

	parentDir := filepath.Dir(dst)
	if !Exists(parentDir) {
		err := os.MkdirAll(parentDir, os.ModePerm)
		if err != nil {
			fmt.Println(err)
			return
		} else {
			fmt.Println("创建父目录：", parentDir)
		}
	}
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

func WriteStringToFile(content string) (string, error) {
	contentBytes := []byte(content)
	var uuid1, err = uuid.NewUUID()
	if err != nil {
		return "", err
	}
	fileUuid := strings.ReplaceAll(uuid1.String(), "-", "")
	filepath := "/tmp/" + fileUuid
	os.WriteFile(filepath, contentBytes, 0666)
	return fileUuid, nil
}

func ReadFileString(fileName string) (string, error) {
	f, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	return string(f), nil
}

func GetFileMD5WithRetry(photo string, retry int) (string, error) {
	var md5 string
	var err error
	for i := 0; i < retry; i++ {
		md5, err = GetFileMD5(photo)
		if err != nil {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	return md5, err
}

func GetDirDate(photo string) string {
	parentDir := filepath.Dir(photo)
	dirDate := path.Base(parentDir)
	return dirDate
}

func GetFileDate(photo string) string {
	filename := path.Base(photo)

	var fileDate string
	for i, v := range timePatternArray {
		if match := v.FindStringSubmatch(filename); match != nil {
			stamp, _ := time.ParseInLocation(timeTemplateArray[i], match[1], time.Local)
			fileDate = stamp.Format("2006-01-02")
		}
	}
	return fileDate

}

func GetModifyDate(photo string) string {
	fileInfo, err := os.Stat(photo)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	modify := fileInfo.ModTime()
	modifyDate := modify.Format("2006-01-02")
	return modifyDate
}

func ChangeModifyDate(photo string, time time.Time) {
	err := os.Chtimes(photo, time, time)
	if err != nil {
		fmt.Print("ChangeModifyDate error : ", photo)
	}
}

func MarshalPrint(v any) string {
	sm, _ := json.Marshal(v)
	return string(sm)
}
