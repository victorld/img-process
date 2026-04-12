package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"img_process/tools"
	"net/http"
	"runtime/debug"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				tools.Logger.Error("panic recovered path=", ctx.Request.URL.Path, " method=", ctx.Request.Method, " err=", err, " stack=", string(debug.Stack()))
				tools.FailWithStatus(ctx, http.StatusInternalServerError, fmt.Sprint(err), nil)
			}
		}()

		ctx.Next()
	}
}
