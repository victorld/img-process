package cons

import (
	"fmt"
	"img_process/tools"
	"os"
	"strconv"
)

var (
	Dbusername       string
	Dbpassword       string
	Dbhost           string
	Dbport           string
	Dbname           string
	Dbconfig         string
	HttpPort         string
	HttpUsername     string
	HttpPassword     string
	StartPath        string
	DeleteShow       bool
	MoveFileShow     bool
	ModifyDateShow   bool
	Md5Show          bool
	DeleteAction     bool
	MoveFileAction   bool
	ModifyDateAction bool
	ImgCache         bool
	WorkDir          string
)

func InitConst() {
	//server
	Dbusername = tools.GetConfigString("database.Dbusername")
	Dbpassword = tools.GetConfigString("database.Dbpassword")
	Dbhost = tools.GetConfigString("database.Dbhost")
	Dbport = tools.GetConfigString("database.Dbport")
	Dbname = tools.GetConfigString("database.Dbname")
	Dbconfig = tools.GetConfigString("database.Dbconfig")

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

	fmt.Println("Dbusername :", Dbusername)
	fmt.Println("Dbpassword :", Dbpassword)
	fmt.Println("Dbhost :", Dbhost)
	fmt.Println("Dbport :", Dbport)
	fmt.Println("Dbname :", Dbname)
	fmt.Println("Dbconfig :", Dbconfig)

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

	WorkDir, _ = os.Getwd() // 项目工作目录
	fmt.Println("工作目录: " + WorkDir)

	fmt.Println()

}
