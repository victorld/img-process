package tools

import (
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var ImgMysqlDB *gorm.DB

type MysqlArgs struct {
	Username string
	Password string
	Host     string
	Port     string
	Dbname   string
	Config   string
}

func InitMysql(viper *viper.Viper) {
	mysqlArgs := MysqlArgs{
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetString("database.host"),
		viper.GetString("database.port"),
		viper.GetString("database.dbname"),
		viper.GetString("database.config"),
	}
	GormMysql(mysqlArgs)
}

// GormMysql 初始化Mysql数据库
func GormMysql(mysqlArgs MysqlArgs) {
	dsn := mysqlArgs.Username + ":" + mysqlArgs.Password + "@tcp(" + mysqlArgs.Host + ":" + mysqlArgs.Port + ")/" + mysqlArgs.Dbname + "?" + mysqlArgs.Config
	Logger.Info("dsn : ", dsn)

	ImgMysqlDB, _ = gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,   // DSN data source name
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
	}), &gorm.Config{})

}