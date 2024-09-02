package middleware

import "img_process/tools"

func GetExifInfo(path string) (string, string, int, string, error) {

	flag := -1
	var output string
	shootTime, locNum, output, err := GetExifInfoCommand(path)
	if err != nil {
		shootTime, locNum, err = GetExifInfoGo(path)
		if err != nil {
			tools.Logger.Info("no exif : ", path)
		} else {
			flag = 2
			tools.Logger.Info("command get  exif : ", path)
		}
	} else {
		flag = 1
	}

	return shootTime, locNum, flag, output, err
}
