package routes

import (
	"database/sql"
	"net/http"

	"go-demo/controllers"
	"go-demo/middlewares"
	"go-demo/services"

	"github.com/gin-gonic/gin"
)

// Setup 初始化所有路由并返回 Gin Engine
func Setup(db *sql.DB) *gin.Engine {
	r := gin.Default()
	r.Static("/uploads", "./uploads")

	// 创建服务与控制器
	authService := services.NewAuthService(db)
	adminData := services.NewAdminDataService(db)
	monitorService := services.NewMonitorService(db)
	authCtrl := controllers.NewAuthController(authService)
	adminCtrl := controllers.NewAdminController(authService, adminData, monitorService)
	chatCtrl := controllers.NewChatController(db, authService)
	monitorService.SetRuntimeStatsProvider(chatCtrl)
	r.Use(middlewares.RequestMonitor(monitorService))

	// 公开路由
	r.GET("/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Welcome to Go Demo API"}) })
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	api := r.Group("/api/v1")
	{
		api.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "pong"}) })
		api.GET("/password-public-key", authCtrl.PasswordPublicKey)
		api.GET("/chat/ws", chatCtrl.WebSocket)
		api.POST("/register", authCtrl.Register)
		api.POST("/login", authCtrl.Login)
		api.POST("/refresh-token", authCtrl.RefreshToken)
		api.POST("/forgot-password", authCtrl.ForgotPassword)
		api.POST("/reset-password", authCtrl.ResetPassword)

		// 认证路由
		protected := api.Group("")
		protected.Use(middlewares.AuthMiddleware(authService))
		{
			protected.GET("/me", authCtrl.Me)

			// 聊天路由
			protected.GET("/chat/users", middlewares.RequirePermission(authService, "messages:chat"), chatCtrl.ListUsers)
			protected.GET("/chat/history/:id", middlewares.RequirePermission(authService, "messages:chat"), chatCtrl.History)
			protected.POST("/chat/upload", middlewares.RequirePermission(authService, "messages:chat"), chatCtrl.Upload)
			protected.POST("/chat/translate", middlewares.RequirePermission(authService, "messages:chat"), chatCtrl.Translate)
			protected.POST("/chat/read/:id", middlewares.RequirePermission(authService, "messages:chat"), chatCtrl.MarkRead)

			// 管理员路由
			admin := protected.Group("/admin")
			admin.Use(middlewares.RequirePermission(authService, "admin:access"))
			{
				admin.GET("/dashboard", adminCtrl.Dashboard)

				// 用户管理
				admin.GET("/users", middlewares.RequirePermission(authService, "users:read"), adminCtrl.ListUsers)
				admin.POST("/users", middlewares.RequirePermission(authService, "users:write"), adminCtrl.CreateUser)
				admin.PUT("/users/:id", middlewares.RequirePermission(authService, "users:write"), adminCtrl.UpdateUser)
				admin.PUT("/users/:id/roles", middlewares.RequirePermission(authService, "users:write"), adminCtrl.SetUserRoles)
				admin.GET("/users/:id/password", middlewares.RequirePermission(authService, "users:password:read"), adminCtrl.GetUserPassword)
				admin.PUT("/users/:id/password", middlewares.RequirePermission(authService, "users:write"), adminCtrl.ResetUserPassword)
				admin.DELETE("/users/:id", middlewares.RequirePermission(authService, "users:write"), adminCtrl.DeactivateUser)

				// 角色管理
				admin.GET("/roles", middlewares.RequirePermission(authService, "roles:read"), adminCtrl.ListRoles)
				admin.POST("/roles", middlewares.RequirePermission(authService, "roles:write"), adminCtrl.CreateRole)
				admin.PUT("/roles/:id", middlewares.RequirePermission(authService, "roles:write"), adminCtrl.UpdateRole)
				admin.DELETE("/roles/:id", middlewares.RequirePermission(authService, "roles:write"), adminCtrl.DeleteRole)

				// 权限列表
				admin.GET("/permissions", middlewares.RequirePermission(authService, "permissions:read"), adminCtrl.ListPermissions)
				admin.GET("/permissions/tree", middlewares.RequirePermission(authService, "permissions:read"), adminCtrl.PermissionTree)
				admin.GET("/roles/:id/preview", middlewares.RequirePermission(authService, "roles:read"), adminCtrl.RolePreview)

				// 操作日志
				admin.GET("/operation-logs", middlewares.RequirePermission(authService, "logs:read"), adminCtrl.ListOperationLogs)

				// 通知中心
				admin.GET("/notifications", middlewares.RequirePermission(authService, "notifications:read"), adminCtrl.ListNotifications)
				admin.GET("/notifications/unread-count", middlewares.RequirePermission(authService, "notifications:read"), adminCtrl.UnreadNotificationCount)
				admin.PUT("/notifications/:id/read", middlewares.RequirePermission(authService, "notifications:write"), adminCtrl.MarkNotificationRead)
				admin.PUT("/notifications/read-all", middlewares.RequirePermission(authService, "notifications:write"), adminCtrl.MarkAllNotificationsRead)

				// AI 助手与系统健康
				admin.POST("/ai/ask", middlewares.RequirePermission(authService, "ai:assistant"), adminCtrl.AskAssistant)
				admin.GET("/health", middlewares.RequirePermission(authService, "health:read"), adminCtrl.SystemHealth)
			}
		}
	}

	return r
}
