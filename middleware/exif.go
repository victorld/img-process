package middleware

// GetExifInfo 获取拍摄日期和地理位置
func GetExifInfo(path string) (string, string, int, string, error) {

	flag := -1                                                 //都没获取到，默认置为-1
	shootTime, locNum, output, err := GetExifInfoCommand(path) //优先从命令行获取shoottime
	if err != nil {
		shootTime, locNum, err = GetExifInfoGo(path) //如果命令行没有获取到shoottime，再用go语言获取
		if err != nil {
			//tools.Logger.Info("no exif : ", path)
		} else { //如果go语言获取到了，flag置为2
			flag = 2
			//tools.Logger.Info("command get  exif : ", path)
		}
	} else { //如果命令行获取到了，flag置为1
		flag = 1
	}

	return shootTime, locNum, flag, output, err
}
