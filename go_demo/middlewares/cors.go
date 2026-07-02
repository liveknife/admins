package middlewares

import (
	"net/http"
	"os"
	"strings"

	"go-demo/config"

	"github.com/gin-gonic/gin"
)

// CORS 提供跨域资源共享中间件。
// 生产环境仅允许配置的域名（CORS_ALLOWED_ORIGINS），开发环境默认允许 localhost。
// 未匹配的 Origin 请求将被拒绝，返回 403。
func CORS() gin.HandlerFunc {
	allowedOrigins := parseAllowedOrigins()
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		if isOriginAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func parseAllowedOrigins() []string {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if raw == "" {
		if config.IsProduction() {
			return []string{}
		}
		return []string{"http://localhost:*"}
	}
	origins := strings.Split(raw, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}
	return origins
}

func isOriginAllowed(origin string, allowed []string) bool {
	for _, pattern := range allowed {
		if strings.HasSuffix(pattern, ":*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(origin, prefix) {
				return true
			}
		}
		if pattern == origin {
			return true
		}
	}
	return false
}
