package main

import (
	"github.com/gin-gonic/gin"
	"img_process/bootstrap"
	"img_process/cons"
	"img_process/middleware"
	"img_process/route"
	"img_process/service"
	"img_process/tools"
)

// 扫描web服务
func main() {
	closeFn, err := bootstrap.InitApp(true)
	if err != nil {
		tools.Logger.Fatal("bootstrap init error : ", err)
	}
	defer closeFn()
	if err := middleware.RegisterTable(); err != nil {
		tools.Logger.Fatal("register table error : ", err)
	}
	if err := service.InitSystemSettings(); err != nil {
		tools.Logger.Fatal("init system settings error : ", err)
	}
	if err := service.Runtime.Start(); err != nil {
		tools.Logger.Fatal("runtime start error : ", err)
	}
	defer service.Runtime.Stop()

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
