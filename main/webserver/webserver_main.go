package main

import (
	"github.com/gin-gonic/gin"
	"img_process/bootstrap"
	"img_process/cons"
	"img_process/route"
	"img_process/tools"
)

// 扫描web服务
func main() {
	closeFn, err := bootstrap.InitApp(true)
	if err != nil {
		tools.Logger.Fatal("bootstrap init error : ", err)
	}
	defer closeFn()

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
