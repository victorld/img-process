package cons

import (
	"fmt"
	"img_process/tools"
	"os"
	"strings"
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
		StartPathBak string
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
	AppConfig = withConfigFileDefaults(AppConfig)

	ApplyConfig(AppConfig)
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

func WithRuntimeDefaults(config Config) Config {
	if config.ScanArgs.StartPath == "" {
		config.ScanArgs.StartPath = "/Users/ld/my-file/pic-lib/pic-new"
	}
	config.ScanArgs.DeleteShow = true
	config.ScanArgs.MoveFileShow = true
	config.ScanArgs.RenameFileShow = true
	config.ScanArgs.Md5Show = true
	config.Basic.ColorOutput = true
	config.Cache.ImgCache = true
	config.Cache.SyncTable = true
	if config.Dump.PoolSize == 0 {
		config.Dump.PoolSize = 8
	}
	if config.Dump.Md5Retry == 0 {
		config.Dump.Md5Retry = 3
	}
	if config.Dump.Md5CountLength == 0 {
		config.Dump.Md5CountLength = 262144
	}
	if config.Bak.StartPathBak == "" {
		config.Bak.StartPathBak = "/Volumes/mount/personal_folder/pic-lib/pic-new"
	}
	if config.Batch.IDInsertBatchSize == 0 {
		config.Batch.IDInsertBatchSize = 1000
	}
	if config.Batch.IDDeleteBatchSize == 0 {
		config.Batch.IDDeleteBatchSize = 300
	}
	if config.Batch.GDUpdateBatchSize == 0 {
		config.Batch.GDUpdateBatchSize = 1000
	}
	return config
}

func withConfigFileDefaults(config Config) Config {
	defaults := WithRuntimeDefaults(Config{})
	applyStringDefault(&config.ScanArgs.StartPath, defaults.ScanArgs.StartPath, "scanArgs.StartPath")
	applyBoolDefault(&config.ScanArgs.DeleteShow, defaults.ScanArgs.DeleteShow, "scanArgs.DeleteShow")
	applyBoolDefault(&config.ScanArgs.MoveFileShow, defaults.ScanArgs.MoveFileShow, "scanArgs.MoveFileShow")
	applyBoolDefault(&config.ScanArgs.ModifyDateShow, defaults.ScanArgs.ModifyDateShow, "scanArgs.ModifyDateShow")
	applyBoolDefault(&config.ScanArgs.RenameFileShow, defaults.ScanArgs.RenameFileShow, "scanArgs.RenameFileShow")
	applyBoolDefault(&config.ScanArgs.Md5Show, defaults.ScanArgs.Md5Show, "scanArgs.Md5Show")
	applyBoolDefault(&config.ScanArgs.DeleteAction, defaults.ScanArgs.DeleteAction, "scanArgs.DeleteAction")
	applyBoolDefault(&config.ScanArgs.MoveFileAction, defaults.ScanArgs.MoveFileAction, "scanArgs.MoveFileAction")
	applyBoolDefault(&config.ScanArgs.ModifyDateAction, defaults.ScanArgs.ModifyDateAction, "scanArgs.ModifyDateAction")
	applyBoolDefault(&config.ScanArgs.RenameFileAction, defaults.ScanArgs.RenameFileAction, "scanArgs.RenameFileAction")
	applyBoolDefault(&config.Basic.ColorOutput, defaults.Basic.ColorOutput, "basic.ColorOutput")
	applyBoolDefault(&config.Basic.SqlDebug, defaults.Basic.SqlDebug, "basic.SqlDebug")
	applyBoolDefault(&config.Cache.ImgCache, defaults.Cache.ImgCache, "cache.ImgCache")
	applyBoolDefault(&config.Cache.SyncTable, defaults.Cache.SyncTable, "cache.SyncTable")
	applyBoolDefault(&config.Cache.TruncateTable, defaults.Cache.TruncateTable, "cache.TruncateTable")
	applyIntDefault(&config.Dump.PoolSize, defaults.Dump.PoolSize, "dump.PoolSize")
	applyIntDefault(&config.Dump.Md5Retry, defaults.Dump.Md5Retry, "dump.Md5Retry")
	applyInt64Default(&config.Dump.Md5CountLength, defaults.Dump.Md5CountLength, "dump.Md5CountLength")
	applyStringDefault(&config.Bak.StartPathBak, defaults.Bak.StartPathBak, "bak.StartPathBak")
	applyStringDefault(&config.Gis.Key, defaults.Gis.Key, "gis.key")
	applyIntDefault(&config.Batch.IDInsertBatchSize, defaults.Batch.IDInsertBatchSize, "batch.IDInsertBatchSize")
	applyIntDefault(&config.Batch.IDDeleteBatchSize, defaults.Batch.IDDeleteBatchSize, "batch.IDDeleteBatchSize")
	applyIntDefault(&config.Batch.GDUpdateBatchSize, defaults.Batch.GDUpdateBatchSize, "batch.GDUpdateBatchSize")
	return config
}

func applyStringDefault(target *string, value string, key string) {
	if !tools.ConfigKeySet(key) {
		*target = value
	}
}

func applyBoolDefault(target *bool, value bool, key string) {
	if !tools.ConfigKeySet(key) {
		*target = value
	}
}

func applyIntDefault(target *int, value int, key string) {
	if !tools.ConfigKeySet(key) {
		*target = value
	}
}

func applyInt64Default(target *int64, value int64, key string) {
	if !tools.ConfigKeySet(key) {
		*target = value
	}
}

func ApplyConfig(config Config) {
	AppConfig = config

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

	GisKey = strings.TrimSpace(os.Getenv("IMG_PROCESS_GIS_KEY"))
	if GisKey == "" {
		GisKey = AppConfig.Gis.Key
	}
	SqlDebug = AppConfig.Basic.SqlDebug

	IDInsertBatchSize = AppConfig.Batch.IDInsertBatchSize
	IDDeleteBatchSize = AppConfig.Batch.IDDeleteBatchSize
	GDUpdateBatchSize = AppConfig.Batch.GDUpdateBatchSize
	tools.SetColorOutput(AppConfig.Basic.ColorOutput)
}

func logConfig(key string, value any) {
	if tools.Logger != nil {
		tools.Logger.Info(key, " : ", value)
		return
	}
	fmt.Println(key, ":", value)
}
