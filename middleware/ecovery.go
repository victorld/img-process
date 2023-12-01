package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"img_process/tools"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				tools.Fail(ctx, fmt.Sprint(err), nil)
			}
		}()

		ctx.Next()
	}
}
