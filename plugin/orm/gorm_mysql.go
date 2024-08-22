package orm

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"img_process/cons"
	"img_process/tools"
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

func InitMysql() {
	mysqlArgs := MysqlArgs{
		cons.DbUsername,
		cons.DbPassword,
		cons.DbHost,
		cons.DbPort,
		cons.DbName,
		cons.DbConfig,
	}
	GormMysql(mysqlArgs)
}

// GormMysql 初始化Mysql数据库
func GormMysql(mysqlArgs MysqlArgs) {
	dsn := mysqlArgs.Username + ":" + mysqlArgs.Password + "@tcp(" + mysqlArgs.Host + ":" + mysqlArgs.Port + ")/" + mysqlArgs.Dbname + "?" + mysqlArgs.Config
	tools.Logger.Info("dsn : ", dsn)

	ImgMysqlDB, _ = gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,   // DSN data source name
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
	}), &gorm.Config{})

}
