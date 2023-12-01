package route

import (
	"github.com/gin-gonic/gin"
	"img_process/api"
	"img_process/middleware"
	"img_process/tools"
)

func InitRouter(r *gin.Engine) *gin.Engine {
	r.Use(middleware.CORSMiddleware(), middleware.RecoveryMiddleware())

	scanGroup := r.Group("/scan", gin.BasicAuth(gin.Accounts{
		tools.GetConfigString("server.username"): tools.GetConfigString("server.password"),
	}))

	var imgRecordApi = new(api.ImgRecordOwnApi)
	scanGroup.GET("doscan", imgRecordApi.DoScanImg)

	return r
}
