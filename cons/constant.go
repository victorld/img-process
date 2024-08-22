package cons

import (
	"fmt"
	"img_process/tools"
	"os"
	"strconv"
)

var (
	DbUsername       string
	DbPassword       string
	DbHost           string
	DbPort           string
	DbName           string
	DbConfig         string
	HttpPort         string
	HttpUsername     string
	HttpPassword     string
	StartPath        string
	StartPathBak     string
	DeleteShow       bool
	MoveFileShow     bool
	ModifyDateShow   bool
	Md5Show          bool
	DeleteAction     bool
	MoveFileAction   bool
	ModifyDateAction bool
	ImgCache         bool
	TruncateTable    bool
	BakStatEnable    bool
	WorkDir          string
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

	StartPath = tools.GetConfigString("scan.StartPath")
	DeleteShow, _ = strconv.ParseBool(tools.GetConfigString("scan.DeleteShow"))
	MoveFileShow, _ = strconv.ParseBool(tools.GetConfigString("scan.MoveFileShow"))
	ModifyDateShow, _ = strconv.ParseBool(tools.GetConfigString("scan.ModifyDateShow"))
	Md5Show, _ = strconv.ParseBool(tools.GetConfigString("scan.Md5Show"))
	DeleteAction, _ = strconv.ParseBool(tools.GetConfigString("scan.DeleteAction"))
	MoveFileAction, _ = strconv.ParseBool(tools.GetConfigString("scan.MoveFileAction"))
	ModifyDateAction, _ = strconv.ParseBool(tools.GetConfigString("scan.ModifyDateAction"))

	ImgCache, _ = strconv.ParseBool(tools.GetConfigString("basic.ImgCache"))
	TruncateTable, _ = strconv.ParseBool(tools.GetConfigString("basic.TruncateTable"))

	StartPathBak = tools.GetConfigString("bak.StartPathBak")
	BakStatEnable, _ = strconv.ParseBool(tools.GetConfigString("bak.BakStatEnable"))

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

	fmt.Println("StartPathBak :", StartPathBak)
	fmt.Println("BakStatEnable :", BakStatEnable)

	WorkDir, _ = os.Getwd() // 项目工作目录
	fmt.Println("工作目录: " + WorkDir)

	fmt.Println()

}
