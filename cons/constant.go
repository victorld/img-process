package cons

import (
	"fmt"
	"img_process/tools"
	"os"
)

type Config struct {
	Database struct {
		DbUsername string
		DbPassword string
		DbHost     string
		DbPort     string
		DbName     string
		DbConfig   string
	}
	Server struct {
		HttpPort     string
		HttpUsername string
		HttpPassword string
	}
	ScanArgs struct {
		StartPath        string
		DeleteShow       bool
		MoveFileShow     bool
		ModifyDateShow   bool
		RenameFileShow   bool
		Md5Show          bool
		DeleteAction     bool
		MoveFileAction   bool
		ModifyDateAction bool
		RenameFileAction bool
	}
	Cache struct {
		ImgCache      bool
		SyncTable     bool
		TruncateTable bool
	}
	Bak struct {
		StartPathBak  string
		BakStatEnable bool
	}
	Gis struct {
		Key string
	}
	Basic struct {
		SqlDebug    bool
		ColorOutput bool
	}
	Dump struct {
		PoolSize       int
		Md5Retry       int
		Md5CountLength int64
	}
	Batch struct {
		IDInsertBatchSize int
		IDDeleteBatchSize int
		GDUpdateBatchSize int
	}
}

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
	AppConfig         Config
)

func InitConst() {
	AppConfig = Config{}
	if err := tools.UnmarshalConfig(&AppConfig); err != nil {
		logConfig("config unmarshal error", err)
	}

	DbUsername = AppConfig.Database.DbUsername
	DbPassword = AppConfig.Database.DbPassword
	DbHost = AppConfig.Database.DbHost
	DbPort = AppConfig.Database.DbPort
	DbName = AppConfig.Database.DbName
	DbConfig = AppConfig.Database.DbConfig

	HttpPort = AppConfig.Server.HttpPort
	HttpUsername = AppConfig.Server.HttpUsername
	HttpPassword = AppConfig.Server.HttpPassword

	StartPath = AppConfig.ScanArgs.StartPath
	DeleteShow = AppConfig.ScanArgs.DeleteShow
	MoveFileShow = AppConfig.ScanArgs.MoveFileShow
	ModifyDateShow = AppConfig.ScanArgs.ModifyDateShow
	RenameFileShow = AppConfig.ScanArgs.RenameFileShow
	Md5Show = AppConfig.ScanArgs.Md5Show
	DeleteAction = AppConfig.ScanArgs.DeleteAction
	MoveFileAction = AppConfig.ScanArgs.MoveFileAction
	ModifyDateAction = AppConfig.ScanArgs.ModifyDateAction
	RenameFileAction = AppConfig.ScanArgs.RenameFileAction

	ImgCache = AppConfig.Cache.ImgCache
	SyncTable = AppConfig.Cache.SyncTable
	TruncateTable = AppConfig.Cache.TruncateTable

	PoolSize = AppConfig.Dump.PoolSize
	Md5Retry = AppConfig.Dump.Md5Retry
	Md5CountLength = AppConfig.Dump.Md5CountLength

	StartPathBak = AppConfig.Bak.StartPathBak
	BakStatEnable = AppConfig.Bak.BakStatEnable

	GisKey = AppConfig.Gis.Key
	SqlDebug = AppConfig.Basic.SqlDebug

	IDInsertBatchSize = AppConfig.Batch.IDInsertBatchSize
	IDDeleteBatchSize = AppConfig.Batch.IDDeleteBatchSize
	GDUpdateBatchSize = AppConfig.Batch.GDUpdateBatchSize
	tools.SetColorOutput(AppConfig.Basic.ColorOutput)

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

func GetConfig() Config {
	return AppConfig
}

func logConfig(key string, value any) {
	if tools.Logger != nil {
		tools.Logger.Info(key, " : ", value)
		return
	}
	fmt.Println(key, ":", value)
}
