package main

import (
	"github.com/gin-gonic/gin"
	"img_process/route"
	"img_process/tools"
)

func main() {

	tools.InitLogger()
	tools.InitViper()
	tools.InitMysql()

	if tools.ImgMysqlDB != nil {
		db, _ := tools.ImgMysqlDB.DB()
		defer db.Close()
	}

	r := gin.Default()
	r = route.InitRouter(r)
	port := tools.GetConfigString("server.port")
	if port != "" {
		panic(r.Run(":" + port))
	}
	panic(r.Run()) // listen and serve on 0.0.0.0
}
