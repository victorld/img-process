package middleware

import (
	"path/filepath"
	"strings"
)

var getExifInfoGoFunc = GetExifInfoGo

var exiftoolSuffixAllowlist = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".heic": {},
	".heif": {},
	".tif":  {},
	".tiff": {},
	".dng":  {},
	".arw":  {},
	".cr2":  {},
	".cr3":  {},
	".nef":  {},
	".orf":  {},
	".raf":  {},
	".rw2":  {},
	".mov":  {},
	".mp4":  {},
	".m4v":  {},
	".avi":  {},
	".mts":  {},
	".3gp":  {},
}

// GetExifInfo 获取拍摄日期和地理位置
func GetExifInfo(path string) (string, string, int, string, []string, error) {

	flag := -1
	var (
		shootTime    string
		locNum       string
		output       string
		dateTagNames []string
		err          error
	)

	useExiftool := shouldUseExiftool(path)
	if useExiftool {
		shootTime, locNum, output, dateTagNames, err = GetExifInfoCommand(path)
	}
	if err != nil || !useExiftool {
		shootTime, locNum, err = getExifInfoGoFunc(path)
		if err != nil {
		} else { //如果go语言获取到了，flag置为2
			flag = 2
		}
	} else { //如果命令行获取到了，flag置为1
		flag = 1
	}

	return shootTime, locNum, flag, output, dateTagNames, err
}

func shouldUseExiftool(path string) bool {
	_, ok := exiftoolSuffixAllowlist[strings.ToLower(filepath.Ext(path))]
	return ok
}
