package cons

import (
	"fmt"
	"img_process/tools"
	"os"
	"strconv"
)

var (
	DbUsername        string
	DbPassword        string
	DbHost            string
	DbPort            string
	DbName            string
	DbConfig          string
	HttpPort          string
	HttpUsername      string
	HttpPassword      string
	StartPath         string
	StartPathBak      string
	GisKey            string
	DeleteShow        bool
	MoveFileShow      bool
	ModifyDateShow    bool
	Md5Show           bool
	DeleteAction      bool
	MoveFileAction    bool
	ModifyDateAction  bool
	ImgCache          bool
	SyncTable         bool
	TruncateTable     bool
	BakStatEnable     bool
	WorkDir           string
	PoolSize          int
	Md5Retry          int
	Md5CountLength    int64
	IDInsertBatchSize int
	IDDeleteBatchSize int
)

func InitConst() {
	//server
	DbUsername = tools.GetConfigString("database.DbUsername")
	DbPassword = tools.GetConfigString("database.DbPassword")
	DbHost = tools.GetConfigString("database.DbHost")
	DbPort = tools.GetConfigString("database.DbPort")
	DbName = tools.GetConfigString("database.DbName")
	DbConfig = tools.GetConfigString("database.DbConfig")

	HttpPort = tools.GetConfigString("server.HttpPort")
	HttpUsername = tools.GetConfigString("server.HttpUsername")
	HttpPassword = tools.GetConfigString("server.HttpPassword")

	StartPath = tools.GetConfigString("scanArgs.StartPath")
	DeleteShow, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.DeleteShow"))
	MoveFileShow, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.MoveFileShow"))
	ModifyDateShow, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.ModifyDateShow"))
	Md5Show, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.Md5Show"))
	DeleteAction, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.DeleteAction"))
	MoveFileAction, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.MoveFileAction"))
	ModifyDateAction, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.ModifyDateAction"))

	ImgCache, _ = strconv.ParseBool(tools.GetConfigString("cache.ImgCache"))
	SyncTable, _ = strconv.ParseBool(tools.GetConfigString("cache.SyncTable"))
	TruncateTable, _ = strconv.ParseBool(tools.GetConfigString("cache.TruncateTable"))

	PoolSize, _ = strconv.Atoi(tools.GetConfigString("dump.PoolSize"))
	Md5Retry, _ = strconv.Atoi(tools.GetConfigString("dump.Md5Retry"))
	Md5CountLength, _ = strconv.ParseInt(tools.GetConfigString("dump.Md5CountLength"), 10, 64)

	StartPathBak = tools.GetConfigString("bak.StartPathBak")
	BakStatEnable, _ = strconv.ParseBool(tools.GetConfigString("bak.BakStatEnable"))

	GisKey = tools.GetConfigString("gis.key")

	IDInsertBatchSize, _ = strconv.Atoi(tools.GetConfigString("batch.IDInsertBatchSize"))
	IDDeleteBatchSize, _ = strconv.Atoi(tools.GetConfigString("batch.IDDeleteBatchSize"))

	fmt.Println("DbUsername :", DbUsername)
	fmt.Println("DbPassword :", DbPassword)
	fmt.Println("DbHost :", DbHost)
	fmt.Println("DbPort :", DbPort)
	fmt.Println("DbName :", DbName)
	fmt.Println("DbConfig :", DbConfig)

	fmt.Println("HttpPort :", HttpPort)
	fmt.Println("HttpUsername :", HttpUsername)
	fmt.Println("HttpPassword :", HttpPassword)

	fmt.Println("StartPath :", StartPath)
	fmt.Println("DeleteShow :", DeleteShow)
	fmt.Println("MoveFileShow :", MoveFileShow)
	fmt.Println("ModifyDateShow :", ModifyDateShow)
	fmt.Println("DeleteAction :", DeleteAction)
	fmt.Println("MoveFileAction :", MoveFileAction)
	fmt.Println("ModifyDateAction :", ModifyDateAction)

	fmt.Println("ImgCache :", ImgCache)
	fmt.Println("TruncateTable :", TruncateTable)
	fmt.Println("SyncTable :", SyncTable)

	fmt.Println("StartPathBak :", StartPathBak)
	fmt.Println("BakStatEnable :", BakStatEnable)

	fmt.Println("PoolSize :", PoolSize)
	fmt.Println("Md5Retry :", Md5Retry)
	fmt.Println("Md5CountLength :", Md5CountLength)

	fmt.Println("GisKey: ", GisKey)

	fmt.Println("IDInsertBatchSize: ", IDInsertBatchSize)
	fmt.Println("IDDeleteBatchSize: ", IDDeleteBatchSize)

	WorkDir, _ = os.Getwd() // 项目工作目录
	fmt.Println("工作目录: " + WorkDir)

	fmt.Println()

}
