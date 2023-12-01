package tools

import (
	"github.com/spf13/viper"
	"os"
)

var VP *viper.Viper

func InitViper() {
	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	VP = viper.New()

	VP.AddConfigPath(path)     //设置读取的文件路径
	VP.SetConfigName("config") //设置读取的文件名
	VP.SetConfigType("yaml")   //设置文件的类型
	//尝试进行配置读取
	if err := VP.ReadInConfig(); err != nil {
		panic(err)
	}
}

func GetConfigString(key string) string {
	ret := VP.GetString(key)
	return ret
}
