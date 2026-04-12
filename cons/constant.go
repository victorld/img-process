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
	RenameFileShow    bool
	Md5Show           bool
	DeleteAction      bool
	MoveFileAction    bool
	ModifyDateAction  bool
	RenameFileAction  bool
	ImgCache          bool
	SyncTable         bool
	TruncateTable     bool
	BakStatEnable     bool
	SqlDebug          bool
	WorkDir           string
	PoolSize          int
	Md5Retry          int
	Md5CountLength    int64
	IDInsertBatchSize int
	IDDeleteBatchSize int
	GDUpdateBatchSize int
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
	RenameFileShow, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.RenameFileShow"))
	Md5Show, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.Md5Show"))
	DeleteAction, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.DeleteAction"))
	MoveFileAction, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.MoveFileAction"))
	ModifyDateAction, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.ModifyDateAction"))
	RenameFileAction, _ = strconv.ParseBool(tools.GetConfigString("scanArgs.RenameFileAction"))

	ImgCache, _ = strconv.ParseBool(tools.GetConfigString("cache.ImgCache"))
	SyncTable, _ = strconv.ParseBool(tools.GetConfigString("cache.SyncTable"))
	TruncateTable, _ = strconv.ParseBool(tools.GetConfigString("cache.TruncateTable"))

	PoolSize, _ = strconv.Atoi(tools.GetConfigString("dump.PoolSize"))
	Md5Retry, _ = strconv.Atoi(tools.GetConfigString("dump.Md5Retry"))
	Md5CountLength, _ = strconv.ParseInt(tools.GetConfigString("dump.Md5CountLength"), 10, 64)

	StartPathBak = tools.GetConfigString("bak.StartPathBak")
	BakStatEnable, _ = strconv.ParseBool(tools.GetConfigString("bak.BakStatEnable"))

	GisKey = tools.GetConfigString("gis.key")
	SqlDebug, _ = strconv.ParseBool(tools.GetConfigString("basic.SqlDebug"))

	IDInsertBatchSize, _ = strconv.Atoi(tools.GetConfigString("batch.IDInsertBatchSize"))
	IDDeleteBatchSize, _ = strconv.Atoi(tools.GetConfigString("batch.IDDeleteBatchSize"))
	GDUpdateBatchSize, _ = strconv.Atoi(tools.GetConfigString("batch.GDUpdateBatchSize"))

	logConfig("DbUsername", DbUsername)
	logConfig("DbPassword", "[REDACTED]")
	logConfig("DbHost", DbHost)
	logConfig("DbPort", DbPort)
	logConfig("DbName", DbName)
	logConfig("DbConfig", DbConfig)

	logConfig("HttpPort", HttpPort)
	logConfig("HttpUsername", HttpUsername)
	logConfig("HttpPassword", "[REDACTED]")

	logConfig("StartPath", StartPath)
	logConfig("DeleteShow", DeleteShow)
	logConfig("MoveFileShow", MoveFileShow)
	logConfig("ModifyDateShow", ModifyDateShow)
	logConfig("RenameFileShow", RenameFileShow)
	logConfig("DeleteAction", DeleteAction)
	logConfig("MoveFileAction", MoveFileAction)
	logConfig("ModifyDateAction", ModifyDateAction)
	logConfig("RenameFileAction", RenameFileAction)

	logConfig("ImgCache", ImgCache)
	logConfig("TruncateTable", TruncateTable)
	logConfig("SyncTable", SyncTable)
	logConfig("SqlDebug", SqlDebug)

	logConfig("StartPathBak", StartPathBak)
	logConfig("BakStatEnable", BakStatEnable)

	logConfig("PoolSize", PoolSize)
	logConfig("Md5Retry", Md5Retry)
	logConfig("Md5CountLength", Md5CountLength)

	logConfig("GisKey", "[REDACTED]")

	logConfig("IDInsertBatchSize", IDInsertBatchSize)
	logConfig("IDDeleteBatchSize", IDDeleteBatchSize)
	logConfig("GDUpdateBatchSize", GDUpdateBatchSize)

	WorkDir, _ = os.Getwd() // 项目工作目录
	logConfig("工作目录", WorkDir)

	fmt.Println()

}

func logConfig(key string, value any) {
	if tools.Logger != nil {
		tools.Logger.Info(key, " : ", value)
		return
	}
	fmt.Println(key, ":", value)
}
