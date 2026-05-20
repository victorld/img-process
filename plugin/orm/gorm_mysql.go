package orm

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"img_process/cons"
	"img_process/tools"
	"net/url"
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
	tools.Logger.Info("mysql target : ", maskedDSN(mysqlArgs))

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
	logMode := logger.Silent
	if cons.SqlDebug {
		logMode = logger.Info
	}
	ImgMysqlDB.Logger = logger.Default.LogMode(logMode)
	return nil
}

func getDSN(mysqlArgs MysqlArgs, includeDB bool) string {
	dbname := ""
	if includeDB {
		dbname = mysqlArgs.Dbname
	}
	config := withDefaultTimeouts(mysqlArgs.Config)
	return mysqlArgs.Username + ":" + mysqlArgs.Password + "@tcp(" + mysqlArgs.Host + ":" + mysqlArgs.Port + ")/" + dbname + "?" + config
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

func maskedDSN(mysqlArgs MysqlArgs) string {
	return fmt.Sprintf("%s:%s/%s?%s", mysqlArgs.Host, mysqlArgs.Port, mysqlArgs.Dbname, withDefaultTimeouts(mysqlArgs.Config))
}

func withDefaultTimeouts(config string) string {
	values, err := url.ParseQuery(config)
	if err != nil {
		if strings.TrimSpace(config) == "" {
			return "timeout=10s&readTimeout=10s&writeTimeout=10s"
		}
		return config
	}
	if values.Get("timeout") == "" {
		values.Set("timeout", "10s")
	}
	if values.Get("readTimeout") == "" {
		values.Set("readTimeout", "10s")
	}
	if values.Get("writeTimeout") == "" {
		values.Set("writeTimeout", "10s")
	}
	return values.Encode()
}
