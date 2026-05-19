package tools

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
)

var VP *viper.Viper

func InitViper() error {
	path, err := os.Getwd()
	if err != nil {
		return err
	}

	VP = viper.New()
	VP.AutomaticEnv()

	if configFile := os.Getenv("IMG_PROCESS_CONFIG"); configFile != "" {
		VP.SetConfigFile(configFile)
	} else {
		VP.AddConfigPath(path) //设置读取的文件路径
		if configDir := os.Getenv("IMG_PROCESS_CONFIG_DIR"); configDir != "" {
			VP.AddConfigPath(configDir)
		}
		VP.SetConfigName("config") //设置读取的文件名
		VP.SetConfigType("yaml")   //设置文件的类型
	}

	if err := VP.ReadInConfig(); err != nil {
		return fmt.Errorf("read config failed: %w", err)
	}

	return nil
}

func GetConfigString(key string) string {
	if VP == nil {
		return ""
	}
	ret := VP.GetString(key)
	return ret
}

func ConfigFileUsed() string {
	if VP == nil {
		return ""
	}
	return VP.ConfigFileUsed()
}

func UnmarshalConfig(rawVal any) error {
	if VP == nil {
		return fmt.Errorf("viper is not initialized")
	}
	return VP.Unmarshal(rawVal)
}
