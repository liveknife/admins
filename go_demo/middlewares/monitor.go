package middlewares

import (
	"time"

	"go-demo/services"

	"github.com/gin-gonic/gin"
)

func RequestMonitor(monitor *services.MonitorService) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		monitor.RecordRequest(c.Request.Method, path, c.Writer.Status(), time.Since(start))
	}
}
