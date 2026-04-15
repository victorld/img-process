package route

import (
	"net/http"
	"os"
	"path/filepath"

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
	scanGroup.POST("scan", imgRecordApi.DoScanImg)
	scanGroup.GET("scan", imgRecordApi.DoScanImg)
	scanGroup.POST("delete", imgRecordApi.DeleteMD5DupFiles)
	scanGroup.DELETE("delete", imgRecordApi.DeleteMD5DupFiles)
	scanGroup.GET("delete", imgRecordApi.DeleteMD5DupFiles)

	webAPI := new(api.WebAPI)
	r.POST("/api/auth/login", webAPI.Login)

	apiGroup := r.Group("/api")
	apiGroup.Use(SessionAuthMiddleware())
	apiGroup.POST("/auth/logout", webAPI.Logout)
	apiGroup.GET("/auth/me", webAPI.Me)
	apiGroup.GET("/jobs", webAPI.ListJobs)
	apiGroup.POST("/jobs", webAPI.CreateJob)
	apiGroup.GET("/jobs/:id", webAPI.GetJob)
	apiGroup.GET("/jobs/:id/events", webAPI.ListJobEvents)
	apiGroup.GET("/jobs/:id/logs", webAPI.ListJobLogs)
	apiGroup.GET("/jobs/:id/action-items", webAPI.ListJobActionItems)
	apiGroup.GET("/jobs/:id/action-preview", webAPI.PreviewJobAction)
	apiGroup.GET("/jobs/:id/stream", webAPI.StreamJob)
	apiGroup.POST("/jobs/:id/actions/delete-duplicates", webAPI.DeleteJobDuplicates)
	apiGroup.POST("/jobs/:id/action-items/:itemId/delete", webAPI.DeleteJobActionItem)
	apiGroup.POST("/jobs/:id/action-items/:itemId/delete-duplicate", webAPI.DeleteDuplicateActionItem)
	apiGroup.POST("/jobs/:id/action-items/:itemId/modify-shoot-time", webAPI.ModifyJobShootTimeActionItem)
	apiGroup.POST("/jobs/:id/action-items/:itemId/move", webAPI.MoveJobActionItem)
	apiGroup.POST("/jobs/:id/action-items/:itemId/rename", webAPI.RenameJobActionItem)
	apiGroup.POST("/jobs/:id/actions/delete-all", webAPI.DeleteAllJobActionItems)
	apiGroup.GET("/schedules", webAPI.ListSchedules)
	apiGroup.POST("/schedules", webAPI.CreateSchedule)
	apiGroup.PUT("/schedules/:id", webAPI.UpdateSchedule)
	apiGroup.DELETE("/schedules/:id", webAPI.DeleteSchedule)
	apiGroup.POST("/schedules/:id/enable", webAPI.EnableSchedule)
	apiGroup.POST("/schedules/:id/disable", webAPI.DisableSchedule)
	apiGroup.POST("/schedules/:id/run", webAPI.RunSchedule)
	apiGroup.GET("/system/status", webAPI.GetSystemStatus)

	registerSPA(r)
	return r
}

func registerSPA(r *gin.Engine) {
	distDir := filepath.Join(cons.WorkDir, "web", "dist")
	indexPath := filepath.Join(distDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return
	}

	r.Static("/assets", filepath.Join(distDir, "assets"))
	r.GET("/", func(c *gin.Context) {
		c.File(indexPath)
	})
	r.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "msg": "not found"})
			return
		}
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/img" {
			c.JSON(http.StatusNotFound, gin.H{"code": http.StatusNotFound, "msg": "not found"})
			return
		}
		c.File(indexPath)
	})
}
