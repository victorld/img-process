package main

import (
	"github.com/gin-gonic/gin"
	"img_process/cons"
	"img_process/plugin/orm"
	"img_process/route"
	"img_process/tools"
)

// 扫描web服务
func Webserver() {

	tools.InitLogger()
	tools.InitViper()
	cons.InitConst()
	orm.InitMysql()

	if orm.ImgMysqlDB != nil {
		db, _ := orm.ImgMysqlDB.DB()
		defer db.Close()
	}

	r := gin.Default()
	r = route.InitRouter(r)
	port := cons.HttpPort
	if port != "" {
		panic(r.Run(":" + port))
	}
	panic(r.Run()) // listen and serve on 0.0.0.0
}
