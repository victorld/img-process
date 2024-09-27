package route

import (
	"github.com/gin-gonic/gin"
	"img_process/api"
	"img_process/cons"
	"img_process/middleware"
)

func InitRouter(r *gin.Engine) *gin.Engine {
	r.Use(middleware.CORSMiddleware(), middleware.RecoveryMiddleware())

	scanGroup := r.Group("/img", gin.BasicAuth(gin.Accounts{
		cons.HttpUsername: cons.HttpPassword,
	}))

	var imgRecordApi = new(api.ImgRecordOwnApi)
	scanGroup.GET("scan", imgRecordApi.DoScanImg)
	scanGroup.GET("delete", imgRecordApi.DeleteMD5DupFiles)

	return r
}
