package orm

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"img_process/cons"
	"img_process/tools"
	"strings"
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

func InitMysql() error {
	mysqlArgs := MysqlArgs{
		cons.DbUsername,
		cons.DbPassword,
		cons.DbHost,
		cons.DbPort,
		cons.DbName,
		cons.DbConfig,
	}
	return GormMysql(mysqlArgs)
}

// GormMysql 初始化Mysql数据库
func GormMysql(mysqlArgs MysqlArgs) error {
	dsn := getDSN(mysqlArgs, true)
	tools.Logger.Info("dsn : ", dsn)

	db, err := openMysql(dsn)
	if err != nil && strings.Contains(err.Error(), "Unknown database") {
		tools.Logger.Warn("database does not exist, creating database : " + mysqlArgs.Dbname)
		if err = createDatabase(mysqlArgs); err == nil {
			db, err = openMysql(dsn)
		}
	}
	if err != nil {
		ImgMysqlDB = nil
		return err
	}

	ImgMysqlDB = db
	ImgMysqlDB.Logger = logger.Default.LogMode(logger.Silent)
	return nil
}

func getDSN(mysqlArgs MysqlArgs, includeDB bool) string {
	dbname := ""
	if includeDB {
		dbname = mysqlArgs.Dbname
	}
	return mysqlArgs.Username + ":" + mysqlArgs.Password + "@tcp(" + mysqlArgs.Host + ":" + mysqlArgs.Port + ")/" + dbname + "?" + mysqlArgs.Config
}

func openMysql(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,   // DSN data source name
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据当前 MySQL 版本自动配置
	}), &gorm.Config{})
}

func createDatabase(mysqlArgs MysqlArgs) error {
	db, err := openMysql(getDSN(mysqlArgs, false))
	if err != nil {
		return err
	}

	return db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", mysqlArgs.Dbname)).Error
}
