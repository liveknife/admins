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
		api.GET("/site/home", adminCtrl.PublicSiteHome)
		api.GET("/site/resources/:slug", adminCtrl.PublicSiteResource)
		api.POST("/site/knowledge", adminCtrl.PublicSiteKnowledge)
		api.POST("/site/messages", adminCtrl.PublicSiteMessage)
		api.POST("/site/visit", adminCtrl.PublicSiteVisit)
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
				admin.GET("/database/catalog", middlewares.RequirePermission(authService, "database:read"), adminCtrl.DatabaseCatalog)
				admin.GET("/database/tables", middlewares.RequirePermission(authService, "database:read"), adminCtrl.ListDatabaseTables)
				admin.GET("/database/tables/:table/columns", middlewares.RequirePermission(authService, "database:read"), adminCtrl.ListDatabaseColumns)

				admin.GET("/site/announcements", middlewares.RequirePermission(authService, "site:read"), adminCtrl.ListSiteAnnouncements)
				admin.POST("/site/announcements", middlewares.RequirePermission(authService, "site:write"), adminCtrl.CreateSiteAnnouncement)
				admin.PUT("/site/announcements/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.UpdateSiteAnnouncement)
				admin.DELETE("/site/announcements/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.DeleteSiteAnnouncement)
				admin.GET("/site/banners", middlewares.RequirePermission(authService, "site:read"), adminCtrl.ListSiteBanners)
				admin.POST("/site/banners", middlewares.RequirePermission(authService, "site:write"), adminCtrl.CreateSiteBanner)
				admin.PUT("/site/banners/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.UpdateSiteBanner)
				admin.DELETE("/site/banners/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.DeleteSiteBanner)
				admin.GET("/site/resources", middlewares.RequirePermission(authService, "site:read"), adminCtrl.ListSiteResources)
				admin.POST("/site/resources", middlewares.RequirePermission(authService, "site:write"), adminCtrl.SaveSiteResource)
				admin.PUT("/site/resources/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.SaveSiteResource)
				admin.DELETE("/site/resources/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.DeleteSiteResource)
				admin.GET("/site/tech-stacks", middlewares.RequirePermission(authService, "site:read"), adminCtrl.ListSiteTechStacks)
				admin.POST("/site/tech-stacks", middlewares.RequirePermission(authService, "site:write"), adminCtrl.SaveSiteTechStack)
				admin.PUT("/site/tech-stacks/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.SaveSiteTechStack)
				admin.DELETE("/site/tech-stacks/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.DeleteSiteTechStack)
				admin.GET("/site/projects", middlewares.RequirePermission(authService, "site:read"), adminCtrl.ListSiteProjects)
				admin.POST("/site/projects", middlewares.RequirePermission(authService, "site:write"), adminCtrl.SaveSiteProject)
				admin.PUT("/site/projects/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.SaveSiteProject)
				admin.DELETE("/site/projects/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.DeleteSiteProject)
				admin.GET("/site/timeline", middlewares.RequirePermission(authService, "site:read"), adminCtrl.ListSiteTimelineEvents)
				admin.POST("/site/timeline", middlewares.RequirePermission(authService, "site:write"), adminCtrl.SaveSiteTimelineEvent)
				admin.PUT("/site/timeline/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.SaveSiteTimelineEvent)
				admin.DELETE("/site/timeline/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.DeleteSiteTimelineEvent)
				admin.GET("/site/messages", middlewares.RequirePermission(authService, "site:read"), adminCtrl.ListSiteMessages)
				admin.PUT("/site/messages/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.SaveSiteMessage)
				admin.DELETE("/site/messages/:id", middlewares.RequirePermission(authService, "site:write"), adminCtrl.DeleteSiteMessage)
				admin.GET("/site/analytics", middlewares.RequirePermission(authService, "site:read"), adminCtrl.SiteAnalytics)
				admin.POST("/site/upload", middlewares.RequirePermission(authService, "site:write"), adminCtrl.UploadSiteAsset)
			}
		}
	}

	return r
}
