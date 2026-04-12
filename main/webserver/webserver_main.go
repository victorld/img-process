package main

import (
	"github.com/gin-gonic/gin"
	"img_process/cons"
	"img_process/plugin/orm"
	"img_process/route"
	"img_process/tools"
)

// 扫描web服务
func main() {

	tools.InitLogger()
	if err := tools.InitViper(); err != nil {
		tools.Logger.Fatal("init viper error : ", err)
	}
	cons.InitConst()
	if err := orm.InitMysql(); err != nil {
		tools.Logger.Fatal("init mysql error : ", err)
	}

	if orm.ImgMysqlDB != nil {
		db, _ := orm.ImgMysqlDB.DB()
		defer db.Close()
	}

	r := gin.Default()
	r = route.InitRouter(r)
	port := cons.HttpPort
	if port != "" {
		if err := r.Run(":" + port); err != nil {
			tools.Logger.Error("web server run error : ", err)
		}
		return
	}
	if err := r.Run(); err != nil { // listen and serve on 0.0.0.0
		tools.Logger.Error("web server run error : ", err)
	}
}
