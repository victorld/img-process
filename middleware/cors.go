package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if origin != "" {
			ctx.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			ctx.Writer.Header().Set("Vary", "Origin")
		} else {
			ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}
		ctx.Writer.Header().Set("Access-Control-Max-Age", "86400")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "*")
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")

		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(200)
		} else {
			ctx.Next()
		}
	}
}
