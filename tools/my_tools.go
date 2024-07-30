package tools

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	goexif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
const DatetimeDirTemplate = "2006-01-02-15-04-05"

var timePatternArray = []*regexp.Regexp{date1Pattern, date2Pattern, date3Pattern, date4Pattern, datetimePattern}
var timeTemplateArray = []string{Data1Template, Data2Template, Data3Template, Data4Template, DatetimeTemplate}

func StrWithColor(str string, color string) string {

	ColorOutput, _ := strconv.ParseBool(GetConfigString("basic.ColorOutput"))
	if !ColorOutput {
		return str
	}
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

func GetFileMD5(filePath string, length int64) (string, error) {
	file, err := os.Open(filePath)
	defer file.Close()
	if err != nil {
		return "", err
	}
	hash := md5.New()
	if length <= 0 {
		_, err = io.Copy(hash, file)
	} else {
		_, err = io.CopyN(hash, file, length)
	}
	if err != nil && err.Error() != "EOF" {
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

func GetRealPath(file string) string {
	realpath := file
	flag := Exists(file)
	if flag {

	} else {
		parentDir := filepath.Dir(file)
		fileName := path.Base(file)
		files, err := os.ReadDir(parentDir)
		if err != nil {
			fmt.Println("read file path error", err)
			return ""
		}
		// 忽略以 . 开头的文件
		for i := 0; i < len(files); i++ {
			fileItem := files[i].Name()
			if strings.HasPrefix(fileItem, fileName) {
				fileName = fileItem
			}
		}
		realpath = parentDir + string(os.PathSeparator) + fileName
	}
	return realpath
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

func CopyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}
	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()
	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func GetFileSize(filePath string) *int64 {
	fi, err := os.Stat(filePath)
	var fileSize *int64
	if err == nil {
		size := fi.Size()
		fileSize = &size
	}
	return fileSize

}
func DeleteFile(filePath string) error {

	// 删除文件
	err := os.Remove(filePath)
	return err

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

func WriteStringToFile(content string, filepath string) {
	contentBytes := []byte(content)
	os.WriteFile(filepath, contentBytes, 0666)
}

func ReadFileString(fileName string) (string, error) {
	f, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	return string(f), nil
}

// ReadLines reads all lines of the file.
func ReadFileLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func GetFileMD5WithRetry(photo string, retry int, length int64) (string, error) {
	var md5 string
	var err error
	for i := 0; i < retry; i++ {
		md5, err = GetFileMD5(photo, length)
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
	dirDate = dirDate[0:10]
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

func MarshalJsonToString(v any) string {
	sm, _ := json.Marshal(v)
	return string(sm)
}

func MapPrintWithFilter[T any](m map[string]T, filter string) {

	keys := []string{}

	for key := range m {
		keys = append(keys, key)
	}
	// 给key排序，从小到大
	sort.Strings(keys)

	for _, key := range keys {
		if strings.Contains(key, filter) {
			fmt.Printf("%v --- %v\n", key, m[key])
		}
	}

}

func ImageNumMapWriteToFile(m map[string][]string, filepath string) {
	keys := []string{}

	for key := range m {
		keys = append(keys, key)
	}
	// 给key排序，从小到大
	sort.Strings(keys)

	var buffer bytes.Buffer

	for _, key := range keys {
		buffer.WriteString(key + "---")
		for _, key2 := range m[key] {
			buffer.WriteString(key2 + ",")
		}
		buffer.WriteString("\n")
	}

	os.WriteFile(filepath, buffer.Bytes(), 0666)

}

func ImageNumRevMapWriteToFile(m map[string][]string, filepath string) {
	whiteList, _ := ReadFileLines("orderWrongAccept.txt")
	var whiteListKey []string
	for _, key := range whiteList {
		whiteListKey = append(whiteListKey, strings.TrimSpace(strings.Split(key, ",")[0]))
	}

	var orderWrongImage []string

	keys := []string{}

	for key := range m {
		keys = append(keys, key)
	}
	// 给key排序，从小到大
	sort.Strings(keys)

	var buffer bytes.Buffer

	for _, key := range keys {
		buffer.WriteString(key + "---")
		oldKey := ""
		for _, key2 := range m[key] {
			if key2 > oldKey {
				buffer.WriteString(key2 + ",")
			} else if Find(whiteList, key2) {
				buffer.WriteString("**" + key2 + ",")
			} else {
				buffer.WriteString("##" + key2 + ",")
				orderWrongImage = append(orderWrongImage, key2)

			}
			oldKey = key2

		}
		buffer.WriteString("\n")
	}

	os.WriteFile(filepath, buffer.Bytes(), 0666)

	fmt.Println("orderWrongImage : ", orderWrongImage)

}

// Find获取一个切片并在其中查找元素。如果找到它，它将返回它的密钥，否则它将返回-1和一个错误的bool。
func Find(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

//func TestGetShootDate(path string) (string, error) {
//
//	f, err := os.Open(path)
//
//	defer func() {
//		f.Close()
//		if r := recover(); r != nil {
//			fmt.Println("exifErr3 Recovered. Error : ", r, " path : ", path)
//		}
//	}()
//
//	if err != nil {
//		fmt.Println(err)
//		return "", err
//	}
//
//	x, err := exif.Decode(f)
//	if err != nil {
//		fmt.Println("exifErr1 exif decode error :  path : ", path)
//		return "", errors.New("exif decode error")
//	}
//
//	shootTime, err := x.DateTime()
//
//	if err != nil {
//		fmt.Println("exifErr2 exif DateTime error :  path : ", path)
//		return "", errors.New("no shoot time")
//	} else {
//		shootTimeStr := shootTime.Format("2006-01-02")
//		//shootTimeStr := shootTime.Format("2006-01-02 15:04:05")
//		return shootTimeStr, nil
//	}
//
//}

// DateTime    DateTimeOriginal    DateTimeDigitized
func GetExifValue(updatedExifIfd *goexif.Ifd, key string) (string, error) {

	results, err := updatedExifIfd.FindTagWithName(key)
	if err != nil {
		//fmt.Println(err)
		return "", err
	}

	ite := results[0]

	phrase, err := ite.FormatFirst()
	if err != nil {
		//fmt.Println(err)
		return "", err
	}

	return phrase, nil
}

func GetExifDateTime(path string) (time.Time, error) {

	//opt := goexif.ScanOptions{}
	//dt, err := goexif.SearchFileAndExtractExif(path)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//ets, _, err := goexif.GetFlatExifData(dt, &opt)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//for _, et := range ets {
	//	fmt.Println(et.TagId, et.TagName, et.TagTypeName, et.Value)
	//}

	rawExif, err := goexif.SearchFileAndExtractExif(path)
	if err != nil {
		//fmt.Println(err)
		return time.Time{}, err
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		//fmt.Println(err)
		return time.Time{}, err
	}

	ti := goexif.NewTagIndex()

	_, index, err := goexif.Collect(im, ti, rawExif)
	if err != nil {
		//fmt.Println(err)
		return time.Time{}, err
	}

	updatedRootIfd := index.RootIfd

	updatedExifIfd, err := updatedRootIfd.ChildWithIfdPath(exifcommon.IfdExifStandardIfdIdentity)
	if err != nil {
		//fmt.Println(err)
		return time.Time{}, err
	}

	value, err := GetExifValue(updatedExifIfd, "DateTimeOriginal")
	if err != nil {
		value, err = GetExifValue(updatedExifIfd, "DateTime")
		if err != nil {
			return time.Time{}, err
		}
	}

	exifTimeLayout := "2006:01:02 15:04:05"

	t, err := time.Parse(exifTimeLayout, value)
	if err != nil {
		//fmt.Println(err)
		return time.Time{}, err
	}
	return t, nil
}

func PrintExifData(path string) {

	opt := goexif.ScanOptions{}
	dt, err := goexif.SearchFileAndExtractExif(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	ets, _, err := goexif.GetFlatExifData(dt, &opt)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, et := range ets {
		fmt.Println(et.TagId, et.TagName, et.TagTypeName, et.Value)
	}
}
