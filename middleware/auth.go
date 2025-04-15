package middleware

import (
	"github.com/gin-gonic/gin"
	"path/filepath"
)

func AuthMiddleware(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 白名单路径，不拦截
		whitelist := map[string]bool{
			"/":         true,
			"/status":   true,
			"/versions": true,
			"/metrics":  true,
			"/static":   true, // 前端资源
		}

		// 如果是静态资源路径（以 /static 开头），也跳过验证
		if _, ok := whitelist[c.Request.URL.Path]; ok || filepath.HasPrefix(c.Request.URL.Path, "/static") {
			c.Next()
			return
		}

		if c.GetHeader("X-API-Key") != apiKey {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}

		c.Next()
	}
}
