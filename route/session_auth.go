package route

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"img_process/service"
	"img_process/tools"
)

func SessionAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("img_process_session")
		if err != nil {
			tools.FailWithStatus(c, http.StatusUnauthorized, "未登录", gin.H{"authenticated": false})
			c.Abort()
			return
		}
		username, ok := service.Runtime.ValidateSession(token)
		if !ok {
			tools.FailWithStatus(c, http.StatusUnauthorized, "登录已失效", gin.H{"authenticated": false})
			c.Abort()
			return
		}
		c.Set("username", username)
		c.Next()
	}
}
