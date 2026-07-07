package routes

import (
	"database/sql"
	"net/http"
	"time"

	"go-demo/config"
	"go-demo/controllers"
	"go-demo/docs"
	"go-demo/middlewares"
	"go-demo/models"
	"go-demo/services"

	"github.com/gin-gonic/gin"
)

// Setup 初始化所有路由并返回 Gin Engine
func Setup(db *sql.DB) *gin.Engine {
	r := gin.Default()
	r.Use(middlewares.CORS())
	r.Static("/uploads", "./uploads")

	authService := services.NewAuthService(db)
	adminData := services.NewAdminDataService(db)
	monitorService := services.NewMonitorService(db)
	captchaService := services.NewCaptchaService()
	authCtrl := controllers.NewAuthController(authService, captchaService)
	adminCtrl := controllers.NewAdminController(authService, adminData, monitorService)
	chatCtrl := controllers.NewChatController(db, authService)
	monitorService.SetRuntimeStatsProvider(chatCtrl)
	r.Use(middlewares.RequestMonitor(monitorService))

	r.GET("/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Welcome to Go Demo API"}) })
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	specs := buildRouteSpecs(authService, authCtrl, adminCtrl, chatCtrl)
	mount(r, specs, authService)

	// Swagger UI: /swagger/index.html (仅开发环境)
	if !config.IsProduction() {
		docs.Register(r, "/swagger", docs.Info{
			Title:       "Admins Platform API",
			Version:     "1.0.0",
			Description: "后台管理平台 API 文档。所有以 `/api/v1/admin/*` 开头的路由需要 Bearer JWT，并按权限码授权。",
			Schemas: []any{
				models.User{}, models.TokenPair{}, models.SiteHome{}, models.SiteResource{},
				models.SiteMessage{}, models.SiteProject{}, models.SiteTechStack{},
				models.SiteAnnouncement{}, models.SiteBanner{}, models.SiteTimelineEvent{},
				models.Notification{}, models.OperationLog{},
			},
		}, docRoutes(specs))
	}

	return r
}

// ──────────────────────────────────────────────
// 路由声明：一处描述，同时喂给 Gin 与 Swagger
// ──────────────────────────────────────────────

type routeSpec struct {
	Method     string
	Path       string // Gin path，含 :id
	Handler    gin.HandlerFunc
	Middleware []gin.HandlerFunc // 额外中间件（如限流）
	Auth       bool              // 需要 Bearer JWT
	Permission string            // 需要的权限码；空表示不校验
	Doc        docs.Op
}

func mount(r *gin.Engine, specs []routeSpec, authService *services.AuthService) {
	for _, s := range specs {
		handlers := []gin.HandlerFunc{}
		handlers = append(handlers, s.Middleware...)
		if s.Auth {
			handlers = append(handlers, middlewares.AuthMiddleware(authService))
		}
		if s.Permission != "" {
			handlers = append(handlers, middlewares.RequirePermission(authService, s.Permission))
		}
		handlers = append(handlers, s.Handler)
		r.Handle(s.Method, s.Path, handlers...)
	}
}

func docRoutes(specs []routeSpec) []docs.Route {
	out := make([]docs.Route, 0, len(specs))
	for _, s := range specs {
		op := s.Doc
		op.Security = s.Auth
		if s.Permission != "" {
			op.Permission = s.Permission
		}
		out = append(out, docs.Route{Method: s.Method, Path: s.Path, Op: op})
	}
	return out
}

// ──────────────────────────────────────────────
// 具体路由清单
// ──────────────────────────────────────────────

func buildRouteSpecs(
	authService *services.AuthService,
	authCtrl *controllers.AuthController,
	adminCtrl *controllers.AdminController,
	chatCtrl *controllers.ChatController,
) []routeSpec {
	_ = authService // 仅用于类型；实际中间件绑定在 mount 里完成

	// 常用响应结构复用
	respErr := docs.Resp{Description: "错误响应", Schema: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"error": map[string]any{"type": "string", "description": "错误消息"},
		},
	}}
	pageQuery := []docs.Param{
		{Name: "page", In: "query", Type: "integer", Description: "页码，从 1 开始，默认 1"},
		{Name: "page_size", In: "query", Type: "integer", Description: "每页数量，默认 10，最大 100"},
	}

	// 限流：登录 / 注册 / 忘记密码 / 图形验证码 / 留言 / 头像上传等
	// 登录额外维度：account/email —— 单 IP 单账号计一次窗口，避免仅按 IP 时被拨号池绕过
	loginLimit := middlewares.RateLimit("login", 8, time.Minute, extractAccountKey)
	registerLimit := middlewares.RateLimit("register", 5, 10*time.Minute, nil)
	forgotLimit := middlewares.RateLimit("forgot", 5, 10*time.Minute, nil)
	captchaLimit := middlewares.RateLimit("captcha", 30, time.Minute, nil)
	messageLimit := middlewares.RateLimit("site-message", 6, 10*time.Minute, nil)
	avatarLimit := middlewares.RateLimit("avatar-upload", 10, 10*time.Minute, nil)
	passwordLimit := middlewares.RateLimit("change-pw", 6, 10*time.Minute, nil)
	refreshLimit := middlewares.RateLimit("refresh-token", 20, time.Minute, nil)

	specs := []routeSpec{
		// ── 公开：健康、基础 ──
		{Method: "GET", Path: "/api/v1/ping", Handler: pingHandler(), Doc: docs.Op{
			Summary: "健康探针 (Ping)", Tags: []string{"Public"},
			Responses: []docs.Resp{{Schema: map[string]any{"type": "object", "properties": map[string]any{"message": map[string]any{"type": "string", "example": "pong"}}}}},
		}},
		{Method: "GET", Path: "/api/v1/password-public-key", Handler: authCtrl.PasswordPublicKey, Doc: docs.Op{
			Summary: "获取 RSA 密码传输公钥", Tags: []string{"Auth"},
			Description: "客户端使用返回的公钥对明文密码进行 RSA-OAEP-SHA256 加密后传输。",
			Responses: []docs.Resp{{Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"algorithm":  map[string]any{"type": "string", "example": "RSA-OAEP-256"},
					"public_key": map[string]any{"type": "string", "description": "PEM 格式公钥"},
				},
			}}},
		}},
		{Method: "GET", Path: "/api/v1/captcha", Handler: authCtrl.Captcha, Middleware: []gin.HandlerFunc{captchaLimit}, Doc: docs.Op{
			Summary: "获取图形验证码", Tags: []string{"Auth"},
			Description: "返回 base64 编码的 PNG 图形验证码和对应的 captcha_id，登录时随请求一起提交。",
			Responses: []docs.Resp{{Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"captcha_id": map[string]any{"type": "string"},
					"image":      map[string]any{"type": "string", "description": "data:image/png;base64,..."},
					"expires_in": map[string]any{"type": "integer", "description": "有效期(秒)"},
				},
			}}},
		}},

		// ── Auth ──
		{Method: "POST", Path: "/api/v1/register", Handler: authCtrl.Register, Middleware: []gin.HandlerFunc{registerLimit}, Doc: docs.Op{
			Summary: "注册用户", Tags: []string{"Auth"},
			Body: docs.Body{Required: true, Schema: controllers.RegisterRequest{}},
			Responses: []docs.Resp{
				{Status: "201", Description: "注册成功", Schema: map[string]any{"type": "object", "properties": map[string]any{"user": docs.SchemaRef("User")}}},
				{Status: "400", Schema: respErr.Schema}, {Status: "409", Description: "用户已存在", Schema: respErr.Schema},
			},
		}},
		{Method: "POST", Path: "/api/v1/login", Handler: authCtrl.Login, Middleware: []gin.HandlerFunc{loginLimit}, Doc: docs.Op{
			Summary: "登录", Tags: []string{"Auth"},
			Description: "支持用户名 / 邮箱 / 手机号登录；密码必须先用 `/api/v1/password-public-key` 返回的公钥加密。",
			Body:        docs.Body{Required: true, Schema: controllers.LoginRequest{}},
			Responses: []docs.Resp{
				{Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"user":   docs.SchemaRef("User"),
						"tokens": docs.SchemaRef("TokenPair"),
					},
				}},
				{Status: "401", Description: "账号或密码错误", Schema: respErr.Schema},
				{Status: "429", Description: "登录尝试过于频繁", Schema: respErr.Schema},
			},
		}},
		{Method: "POST", Path: "/api/v1/refresh-token", Handler: authCtrl.RefreshToken, Middleware: []gin.HandlerFunc{refreshLimit}, Doc: docs.Op{
			Summary: "刷新访问令牌", Tags: []string{"Auth"},
			Body: docs.Body{Required: true, Schema: controllers.RefreshTokenRequest{}},
		}},
		{Method: "POST", Path: "/api/v1/forgot-password", Handler: authCtrl.ForgotPassword, Middleware: []gin.HandlerFunc{forgotLimit}, Doc: docs.Op{
			Summary: "创建密码重置令牌", Tags: []string{"Auth"},
			Body: docs.Body{Required: true, Schema: controllers.ForgotPasswordRequest{}},
		}},
		{Method: "POST", Path: "/api/v1/reset-password", Handler: authCtrl.ResetPassword, Doc: docs.Op{
			Summary: "使用令牌重置密码", Tags: []string{"Auth"},
			Body: docs.Body{Required: true, Schema: controllers.ResetPasswordRequest{}},
		}},

		// ── 官网公开接口 ──
		{Method: "GET", Path: "/api/v1/site/home", Handler: adminCtrl.PublicSiteHome, Doc: docs.Op{
			Summary: "官网首页聚合数据", Tags: []string{"Site (Public)"},
			Responses: []docs.Resp{{Schema: map[string]any{"type": "object", "properties": map[string]any{"home": docs.SchemaRef("SiteHome")}}}},
		}},
		{Method: "GET", Path: "/api/v1/site/resources/:slug", Handler: adminCtrl.PublicSiteResource, Doc: docs.Op{
			Summary: "按 slug 或 ID 获取文章详情", Tags: []string{"Site (Public)"},
			Params: []docs.Param{{Name: "slug", In: "path", Required: true, Description: "文章 slug 或数字 ID"}},
			Responses: []docs.Resp{
				{Schema: map[string]any{"type": "object", "properties": map[string]any{"resource": docs.SchemaRef("SiteResource")}}},
				{Status: "404", Description: "文章不存在", Schema: respErr.Schema},
			},
		}},
		{Method: "GET", Path: "/api/v1/site/search", Handler: adminCtrl.PublicSiteSearch, Doc: docs.Op{
			Summary: "全文搜索已发布文章", Tags: []string{"Site (Public)"},
			Description: "按标题 / 摘要 / 正文 / 标签 / 分类进行大小写不敏感的模糊匹配，命中标题权重最高。",
			Params: []docs.Param{
				{Name: "q", In: "query", Required: true, Description: "搜索关键词"},
				{Name: "category", In: "query", Description: "限定分类"},
				{Name: "tag", In: "query", Description: "限定单个标签"},
				{Name: "page", In: "query", Type: "integer"},
				{Name: "page_size", In: "query", Type: "integer"},
			},
			Responses: []docs.Resp{{Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"items":     map[string]any{"type": "array", "items": docs.SchemaRef("SiteResource")},
					"total":     map[string]any{"type": "integer"},
					"page":      map[string]any{"type": "integer"},
					"page_size": map[string]any{"type": "integer"},
					"query":     map[string]any{"type": "string"},
				},
			}}},
		}},
		{Method: "POST", Path: "/api/v1/site/knowledge", Handler: adminCtrl.PublicSiteKnowledge, Doc: docs.Op{
			Summary: "官网知识库问答", Tags: []string{"Site (Public)"},
			Body: docs.Body{Required: true, Schema: controllers.SiteKnowledgeRequest{}},
		}},
		{Method: "POST", Path: "/api/v1/site/knowledge/stream", Handler: adminCtrl.PublicSiteKnowledgeStream, Doc: docs.Op{
			Summary: "官网知识库问答 SSE 流", Tags: []string{"Site (Public)"},
			Body: docs.Body{Required: true, Schema: controllers.SiteKnowledgeRequest{}},
		}},
		{Method: "POST", Path: "/api/v1/site/feedback", Handler: adminCtrl.PublicSiteFeedback, Doc: docs.Op{
			Summary: "提交 RAG 问答反馈", Tags: []string{"Site (Public)"},
			Body: docs.Body{Required: true, Schema: controllers.SiteFeedbackRequest{}},
		}},
		{Method: "POST", Path: "/api/v1/site/code-explain", Handler: adminCtrl.PublicSiteCodeExplain, Doc: docs.Op{
			Summary: "AI 解释代码片段", Tags: []string{"Site (Public)"},
			Body: docs.Body{Required: true, Schema: controllers.SiteCodeExplainRequest{}},
		}},
		{Method: "POST", Path: "/api/v1/site/search/summarize", Handler: adminCtrl.PublicSiteSearchSummarize, Doc: docs.Op{
			Summary: "搜索结果 AI 总结", Tags: []string{"Site (Public)"},
			Body: docs.Body{Required: true, Schema: controllers.SiteSearchSummarizeRequest{}},
		}},
		{Method: "POST", Path: "/api/v1/site/messages", Handler: adminCtrl.PublicSiteMessage, Middleware: []gin.HandlerFunc{messageLimit}, Doc: docs.Op{
			Summary: "提交访客留言", Tags: []string{"Site (Public)"},
			Body: docs.Body{Required: true, Schema: controllers.SiteMessageRequest{}},
		}},
		{Method: "POST", Path: "/api/v1/site/visit", Handler: adminCtrl.PublicSiteVisit, Doc: docs.Op{
			Summary: "上报访问统计", Tags: []string{"Site (Public)"},
			Body:      docs.Body{Schema: controllers.SiteVisitRequest{}},
			Responses: []docs.Resp{{Status: "204", Description: "已记录"}},
		}},
		{Method: "GET", Path: "/api/v1/chat/ws", Handler: chatCtrl.WebSocket, Doc: docs.Op{
			Summary: "聊天 WebSocket 入口", Tags: []string{"Chat"},
			Description: "通过 URL 参数 `token=<access_token>` 传递 JWT。协议升级到 WebSocket 后不再走 HTTP。",
			Params:      []docs.Param{{Name: "token", In: "query", Required: true, Description: "Bearer 访问令牌"}},
			Responses:   []docs.Resp{{Status: "101", Description: "Switching Protocols"}},
		}},

		// ── 已登录：Me / 聊天 ──
		{Method: "GET", Path: "/api/v1/me", Handler: authCtrl.Me, Auth: true, Doc: docs.Op{
			Summary: "获取当前登录用户", Tags: []string{"Auth"},
			Responses: []docs.Resp{{Schema: map[string]any{"type": "object", "properties": map[string]any{"user": docs.SchemaRef("User")}}}},
		}},
		{Method: "PUT", Path: "/api/v1/me", Handler: authCtrl.UpdateProfile, Auth: true, Doc: docs.Op{
			Summary: "更新当前用户资料", Tags: []string{"Profile"},
			Body: docs.Body{Required: true, Schema: controllers.UpdateProfileRequest{}},
		}},
		{Method: "PUT", Path: "/api/v1/me/password", Handler: authCtrl.ChangeMyPassword, Auth: true, Middleware: []gin.HandlerFunc{passwordLimit}, Doc: docs.Op{
			Summary: "修改当前用户密码", Tags: []string{"Profile"},
			Description: "旧密码与新密码都必须先经 RSA-OAEP-SHA256 加密。",
			Body:        docs.Body{Required: true, Schema: controllers.ChangePasswordRequest{}},
		}},
		{Method: "POST", Path: "/api/v1/me/avatar", Handler: authCtrl.UploadAvatar, Auth: true, Middleware: []gin.HandlerFunc{avatarLimit}, Doc: docs.Op{
			Summary: "上传头像", Tags: []string{"Profile"},
			Description: "`multipart/form-data`，字段名 `file`，允许 jpg/png/webp/gif，最大 5 MB。",
			Responses:   []docs.Resp{{Schema: map[string]any{"type": "object", "properties": map[string]any{"avatar_url": map[string]any{"type": "string"}}}}},
		}},

		{Method: "GET", Path: "/api/v1/chat/users", Handler: chatCtrl.ListUsers, Auth: true, Permission: "messages:chat", Doc: docs.Op{Summary: "聊天用户列表", Tags: []string{"Chat"}}},
		{Method: "GET", Path: "/api/v1/chat/history/:id", Handler: chatCtrl.History, Auth: true, Permission: "messages:chat", Doc: docs.Op{Summary: "查询与指定用户的聊天历史", Tags: []string{"Chat"}, Params: []docs.Param{{Name: "id", In: "path", Type: "integer", Description: "对端用户 ID"}}}},
		{Method: "POST", Path: "/api/v1/chat/upload", Handler: chatCtrl.Upload, Auth: true, Permission: "messages:chat", Doc: docs.Op{Summary: "上传聊天附件", Tags: []string{"Chat"}, Description: "multipart/form-data 字段 `file`"}},
		{Method: "POST", Path: "/api/v1/chat/translate", Handler: chatCtrl.Translate, Auth: true, Permission: "messages:chat", Doc: docs.Op{Summary: "翻译聊天消息", Tags: []string{"Chat"}}},
		{Method: "POST", Path: "/api/v1/chat/read/:id", Handler: chatCtrl.MarkRead, Auth: true, Permission: "messages:chat", Doc: docs.Op{Summary: "标记与该用户的消息为已读", Tags: []string{"Chat"}, Params: []docs.Param{{Name: "id", In: "path", Type: "integer"}}}},

		// ── Admin：Dashboard ──
		{Method: "GET", Path: "/api/v1/admin/dashboard", Handler: adminCtrl.Dashboard, Auth: true, Permission: "admin:access", Doc: docs.Op{Summary: "管理端仪表盘概览", Tags: []string{"Admin · Dashboard"}}},

		// ── Admin：用户 ──
		{Method: "GET", Path: "/api/v1/admin/users", Handler: adminCtrl.ListUsers, Auth: true, Permission: "users:read", Doc: docs.Op{Summary: "用户列表", Tags: []string{"Admin · Users"}, Params: pageQuery}},
		{Method: "POST", Path: "/api/v1/admin/users", Handler: adminCtrl.CreateUser, Auth: true, Permission: "users:write", Doc: docs.Op{Summary: "创建用户", Tags: []string{"Admin · Users"}, Body: docs.Body{Required: true, Schema: controllers.CreateUserRequest{}}}},
		{Method: "PUT", Path: "/api/v1/admin/users/:id", Handler: adminCtrl.UpdateUser, Auth: true, Permission: "users:write", Doc: docs.Op{Summary: "编辑用户", Tags: []string{"Admin · Users"}, Body: docs.Body{Required: true, Schema: controllers.UpdateUserRequest{}}}},
		{Method: "PUT", Path: "/api/v1/admin/users/:id/roles", Handler: adminCtrl.SetUserRoles, Auth: true, Permission: "users:write", Doc: docs.Op{Summary: "分配用户角色", Tags: []string{"Admin · Users"}, Body: docs.Body{Required: true, Schema: controllers.SetUserRolesRequest{}}}},
		{Method: "GET", Path: "/api/v1/admin/users/:id/password", Handler: adminCtrl.GetUserPassword, Auth: true, Permission: "users:password:read", Doc: docs.Op{Summary: "获取用户明文密码（密码保险箱）", Tags: []string{"Admin · Users"}}},
		{Method: "PUT", Path: "/api/v1/admin/users/:id/password", Handler: adminCtrl.ResetUserPassword, Auth: true, Permission: "users:write", Doc: docs.Op{Summary: "重置用户密码", Tags: []string{"Admin · Users"}, Body: docs.Body{Required: true, Schema: controllers.ResetUserPasswordRequest{}}}},
		{Method: "DELETE", Path: "/api/v1/admin/users/:id", Handler: adminCtrl.DeactivateUser, Auth: true, Permission: "users:write", Doc: docs.Op{Summary: "\u505c\u7528\u7528\u6237\u08f08\u8f6f\u5220\u9664\uff09", Tags: []string{"Admin \u00b7 Users"}}},
		{Method: "PUT", Path: "/api/v1/admin/users/:id/reactivate", Handler: adminCtrl.ReactivateUser, Auth: true, Permission: "users:write", Doc: docs.Op{Summary: "\u6062\u590d\u5df2\u7981\u7528\u7528\u6237\u08f09\u91cd\u7f6e\u5bc6\u7801\u4e3a Admin123\uff09", Tags: []string{"Admin \u00b7 Users"}}},
		{Method: "DELETE", Path: "/api/v1/admin/users/:id/permanent", Handler: adminCtrl.DeleteUser, Auth: true, Permission: "users:write", Doc: docs.Op{Summary: "\u6c38\u4e45\u5220\u9664\u7528\u6237\u08f09\u786c\u5220\u9664\uff0c\u4e0d\u53ef\u6062\u590d\uff09", Tags: []string{"Admin \u00b7 Users"}}},

		// ── Admin：角色 & 权限 ──
		{Method: "GET", Path: "/api/v1/admin/roles", Handler: adminCtrl.ListRoles, Auth: true, Permission: "roles:read", Doc: docs.Op{Summary: "角色列表", Tags: []string{"Admin · Roles"}, Params: pageQuery}},
		{Method: "POST", Path: "/api/v1/admin/roles", Handler: adminCtrl.CreateRole, Auth: true, Permission: "roles:write", Doc: docs.Op{Summary: "创建角色", Tags: []string{"Admin · Roles"}, Body: docs.Body{Required: true, Schema: controllers.RoleRequest{}}}},
		{Method: "PUT", Path: "/api/v1/admin/roles/:id", Handler: adminCtrl.UpdateRole, Auth: true, Permission: "roles:write", Doc: docs.Op{Summary: "编辑角色", Tags: []string{"Admin · Roles"}, Body: docs.Body{Required: true, Schema: controllers.RoleRequest{}}}},
		{Method: "DELETE", Path: "/api/v1/admin/roles/:id", Handler: adminCtrl.DeleteRole, Auth: true, Permission: "roles:write", Doc: docs.Op{Summary: "删除角色", Tags: []string{"Admin · Roles"}}},
		{Method: "GET", Path: "/api/v1/admin/permissions", Handler: adminCtrl.ListPermissions, Auth: true, Permission: "permissions:read", Doc: docs.Op{Summary: "权限列表", Tags: []string{"Admin · Roles"}}},
		{Method: "GET", Path: "/api/v1/admin/permissions/tree", Handler: adminCtrl.PermissionTree, Auth: true, Permission: "permissions:read", Doc: docs.Op{Summary: "权限树（菜单 + 按钮）", Tags: []string{"Admin · Roles"}}},
		{Method: "GET", Path: "/api/v1/admin/roles/:id/preview", Handler: adminCtrl.RolePreview, Auth: true, Permission: "roles:read", Doc: docs.Op{Summary: "角色权限预览", Tags: []string{"Admin · Roles"}}},

		// ── Admin：日志 & 通知 ──
		{Method: "GET", Path: "/api/v1/admin/operation-logs", Handler: adminCtrl.ListOperationLogs, Auth: true, Permission: "logs:read", Doc: docs.Op{Summary: "操作日志分页", Tags: []string{"Admin · Logs"}, Params: pageQuery}},
		{Method: "GET", Path: "/api/v1/admin/notifications", Handler: adminCtrl.ListNotifications, Auth: true, Permission: "notifications:read", Doc: docs.Op{Summary: "通知分页", Tags: []string{"Admin · Notifications"}, Params: append(pageQuery, docs.Param{Name: "read", In: "query", Description: "true/false 过滤已读状态"})}},
		{Method: "GET", Path: "/api/v1/admin/notifications/unread-count", Handler: adminCtrl.UnreadNotificationCount, Auth: true, Permission: "notifications:read", Doc: docs.Op{Summary: "未读通知数", Tags: []string{"Admin · Notifications"}}},
		{Method: "PUT", Path: "/api/v1/admin/notifications/:id/read", Handler: adminCtrl.MarkNotificationRead, Auth: true, Permission: "notifications:write", Doc: docs.Op{Summary: "标记单条已读", Tags: []string{"Admin · Notifications"}}},
		{Method: "PUT", Path: "/api/v1/admin/notifications/read-all", Handler: adminCtrl.MarkAllNotificationsRead, Auth: true, Permission: "notifications:write", Doc: docs.Op{Summary: "全部标记已读", Tags: []string{"Admin · Notifications"}}},
		{Method: "POST", Path: "/api/v1/admin/notifications", Handler: adminCtrl.CreateNotification, Auth: true, Permission: "notifications:write", Doc: docs.Op{Summary: "创建通知（推送）", Tags: []string{"Admin · Notifications"}, Body: docs.Body{Required: true, Schema: controllers.CreateNotificationRequest{}}}},
		{Method: "DELETE", Path: "/api/v1/admin/notifications/:id", Handler: adminCtrl.DeleteNotification, Auth: true, Permission: "notifications:write", Doc: docs.Op{Summary: "删除通知", Tags: []string{"Admin · Notifications"}}},

		// ── Admin：后台公告 ──
		{Method: "GET", Path: "/api/v1/admin/announcements", Handler: adminCtrl.ListAnnouncements, Auth: true, Permission: "announcements:read", Doc: docs.Op{Summary: "公告分页", Tags: []string{"Admin · Announcements"}, Params: pageQuery}},
		{Method: "GET", Path: "/api/v1/admin/announcements/active", Handler: adminCtrl.GetActiveAnnouncement, Auth: false, Permission: "", Doc: docs.Op{Summary: "最新启用的公告（用于布局横幅），无需认证", Tags: []string{"Admin · Announcements"}}},
		{Method: "GET", Path: "/api/v1/admin/announcements/public", Handler: adminCtrl.ListPublicAnnouncements, Auth: true, Permission: "", Doc: docs.Op{Summary: "公开公告列表（所有已登录用户可查看）", Tags: []string{"Admin · Announcements"}}},
		{Method: "POST", Path: "/api/v1/admin/announcements", Handler: adminCtrl.CreateAnnouncement, Auth: true, Permission: "announcements:write", Doc: docs.Op{Summary: "创建公告", Tags: []string{"Admin · Announcements"}, Body: docs.Body{Required: true, Schema: controllers.AnnouncementRequest{}}}},
		{Method: "PUT", Path: "/api/v1/admin/announcements/:id", Handler: adminCtrl.UpdateAnnouncement, Auth: true, Permission: "announcements:write", Doc: docs.Op{Summary: "编辑公告", Tags: []string{"Admin · Announcements"}, Body: docs.Body{Required: true, Schema: controllers.AnnouncementRequest{}}}},
		{Method: "DELETE", Path: "/api/v1/admin/announcements/:id", Handler: adminCtrl.DeleteAnnouncement, Auth: true, Permission: "announcements:write", Doc: docs.Op{Summary: "删除公告", Tags: []string{"Admin · Announcements"}}},

		// ── Admin：AI & 系统健康 ──
		{Method: "POST", Path: "/api/v1/admin/ai/ask", Handler: adminCtrl.AskAssistant, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "AI 助手提问", Tags: []string{"Admin · System"}, Body: docs.Body{Required: true, Schema: controllers.AskAssistantRequest{}}}},
		{Method: "GET", Path: "/api/v1/admin/ai/model-configs", Handler: adminCtrl.ListAIModelConfigs, Auth: true, Permission: "ai:models:read", Doc: docs.Op{Summary: "大模型配置列表", Tags: []string{"Admin · System"}}},
		{Method: "POST", Path: "/api/v1/admin/ai/model-configs", Handler: adminCtrl.SaveAIModelConfig, Auth: true, Permission: "ai:models:write", Doc: docs.Op{Summary: "创建大模型配置", Tags: []string{"Admin · System"}, Body: docs.Body{Required: true, Schema: controllers.AIModelConfigRequest{}}}},
		{Method: "PUT", Path: "/api/v1/admin/ai/model-configs/:id", Handler: adminCtrl.SaveAIModelConfig, Auth: true, Permission: "ai:models:write", Doc: docs.Op{Summary: "更新大模型配置", Tags: []string{"Admin · System"}, Body: docs.Body{Required: true, Schema: controllers.AIModelConfigRequest{}}}},
		{Method: "POST", Path: "/api/v1/admin/ai/model-configs/:id/default", Handler: adminCtrl.SetDefaultAIModelConfig, Auth: true, Permission: "ai:models:write", Doc: docs.Op{Summary: "启用大模型配置", Tags: []string{"Admin · System"}}},
		{Method: "POST", Path: "/api/v1/admin/ai/model-configs/:id/test", Handler: adminCtrl.TestAIModelConfig, Auth: true, Permission: "ai:models:write", Doc: docs.Op{Summary: "测试大模型配置", Tags: []string{"Admin · System"}}},
		{Method: "DELETE", Path: "/api/v1/admin/ai/model-configs/:id", Handler: adminCtrl.DeleteAIModelConfig, Auth: true, Permission: "ai:models:write", Doc: docs.Op{Summary: "删除大模型配置", Tags: []string{"Admin · System"}}},
		{Method: "GET", Path: "/api/v1/admin/ai/rag/stats", Handler: adminCtrl.RAGIndexStats, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "RAG 知识索引状态", Tags: []string{"Admin · System"}}},
		{Method: "POST", Path: "/api/v1/admin/ai/rag/reindex", Handler: adminCtrl.RebuildRAGIndex, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "重建 RAG 知识索引", Tags: []string{"Admin · System"}}},
		{Method: "GET", Path: "/api/v1/admin/ai/rag/jobs", Handler: adminCtrl.ListRAGIndexJobs, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "RAG 索引任务列表", Tags: []string{"Admin · System"}}},
		{Method: "POST", Path: "/api/v1/admin/ai/rag/jobs/:id/retry", Handler: adminCtrl.RetryRAGIndexJob, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "重试 RAG 索引任务", Tags: []string{"Admin · System"}}},
		{Method: "GET", Path: "/api/v1/admin/ai/rag/query-logs", Handler: adminCtrl.ListRAGQueryLogs, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "RAG 问答日志", Tags: []string{"Admin · System"}}},
		{Method: "GET", Path: "/api/v1/admin/ai/rag/feedback", Handler: adminCtrl.ListRAGFeedback, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "RAG 问答反馈列表", Tags: []string{"Admin · System"}, Params: []docs.Param{{Name: "rating", In: "query", Description: "up/down/neutral"}, {Name: "limit", In: "query", Type: "integer"}}}},
		{Method: "POST", Path: "/api/v1/admin/ai/rag/diagnostics", Handler: adminCtrl.SearchRAGDiagnostics, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "RAG 检索诊断", Tags: []string{"Admin · System"}, Body: docs.Body{Required: true, Schema: controllers.RAGDiagnosticsRequest{}}}},
		{Method: "POST", Path: "/api/v1/admin/ai/rag/evals/run", Handler: adminCtrl.RunRAGEval, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "运行 RAG 固定评测集", Tags: []string{"Admin · System"}}},
		{Method: "GET", Path: "/api/v1/admin/ai/rag/chunks", Handler: adminCtrl.ListRAGChunks, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "RAG chunk 预览", Tags: []string{"Admin · System"}}},
		{Method: "POST", Path: "/api/v1/admin/documents/upload", Handler: adminCtrl.UploadDocument, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "上传 RAG 文档", Tags: []string{"Admin · RAG"}, Description: "multipart/form-data：字段 `file` 为文档文件；可选字段 `visibility=internal|public`，默认 internal。public 文档可被官网公开问答检索。"}},
		{Method: "GET", Path: "/api/v1/admin/documents", Handler: adminCtrl.ListDocuments, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "RAG 文档列表", Tags: []string{"Admin · RAG"}, Params: pageQuery}},
		{Method: "GET", Path: "/api/v1/admin/documents/:id/preview", Handler: adminCtrl.PreviewDocument, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "预览 RAG 文档文本", Tags: []string{"Admin · RAG"}}},
		{Method: "GET", Path: "/api/v1/admin/documents/:id/chunks", Handler: adminCtrl.ListDocumentChunks, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "预览 RAG 文档 chunk", Tags: []string{"Admin · RAG"}}},
		{Method: "PUT", Path: "/api/v1/admin/documents/:id/visibility", Handler: adminCtrl.UpdateDocumentVisibility, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "切换 RAG 文档 public/internal", Tags: []string{"Admin · RAG"}, Body: docs.Body{Required: true, Schema: controllers.DocumentVisibilityRequest{}}}},
		{Method: "DELETE", Path: "/api/v1/admin/documents/:id", Handler: adminCtrl.DeleteDocument, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "删除 RAG 文档", Tags: []string{"Admin · RAG"}}},
		{Method: "POST", Path: "/api/v1/admin/documents/:id/rebuild", Handler: adminCtrl.RebuildDocument, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "重建 RAG 文档索引", Tags: []string{"Admin · RAG"}}},
		{Method: "GET", Path: "/api/v1/admin/health", Handler: adminCtrl.SystemHealth, Auth: true, Permission: "health:read", Doc: docs.Op{Summary: "系统健康监控", Tags: []string{"Admin · System"}}},
		{Method: "GET", Path: "/api/v1/admin/database/catalog", Handler: adminCtrl.DatabaseCatalog, Auth: true, Permission: "database:read", Doc: docs.Op{Summary: "数据库元信息 (catalog)", Tags: []string{"Admin · System"}}},
		{Method: "GET", Path: "/api/v1/admin/database/tables", Handler: adminCtrl.ListDatabaseTables, Auth: true, Permission: "database:read", Doc: docs.Op{Summary: "数据库表列表", Tags: []string{"Admin · System"}}},
		{Method: "GET", Path: "/api/v1/admin/database/tables/:table/columns", Handler: adminCtrl.ListDatabaseColumns, Auth: true, Permission: "database:read", Doc: docs.Op{Summary: "数据库表字段", Tags: []string{"Admin · System"}}},

		// ── Admin：官网内容 ──
		{Method: "GET", Path: "/api/v1/admin/site/announcements", Handler: adminCtrl.ListSiteAnnouncements, Auth: true, Permission: "site:read", Doc: docs.Op{Summary: "公告列表", Tags: []string{"Admin · Site"}, Params: pageQuery}},
		{Method: "POST", Path: "/api/v1/admin/site/announcements", Handler: adminCtrl.CreateSiteAnnouncement, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "创建公告", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteAnnouncementRequest{}}}},
		{Method: "PUT", Path: "/api/v1/admin/site/announcements/:id", Handler: adminCtrl.UpdateSiteAnnouncement, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "编辑公告", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteAnnouncementRequest{}}}},
		{Method: "DELETE", Path: "/api/v1/admin/site/announcements/:id", Handler: adminCtrl.DeleteSiteAnnouncement, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "删除公告", Tags: []string{"Admin · Site"}}},

		{Method: "GET", Path: "/api/v1/admin/site/banners", Handler: adminCtrl.ListSiteBanners, Auth: true, Permission: "site:read", Doc: docs.Op{Summary: "轮播列表", Tags: []string{"Admin · Site"}, Params: pageQuery}},
		{Method: "POST", Path: "/api/v1/admin/site/banners", Handler: adminCtrl.CreateSiteBanner, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "创建轮播", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteBannerRequest{}}}},
		{Method: "PUT", Path: "/api/v1/admin/site/banners/:id", Handler: adminCtrl.UpdateSiteBanner, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "编辑轮播", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteBannerRequest{}}}},
		{Method: "DELETE", Path: "/api/v1/admin/site/banners/:id", Handler: adminCtrl.DeleteSiteBanner, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "删除轮播", Tags: []string{"Admin · Site"}}},

		{Method: "GET", Path: "/api/v1/admin/site/resources", Handler: adminCtrl.ListSiteResources, Auth: true, Permission: "site:read", Doc: docs.Op{Summary: "资源列表", Tags: []string{"Admin · Site"}, Params: pageQuery}},
		{Method: "POST", Path: "/api/v1/admin/site/resources", Handler: adminCtrl.SaveSiteResource, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "创建资源", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteResourceRequest{}}}},
		{Method: "PUT", Path: "/api/v1/admin/site/resources/:id", Handler: adminCtrl.SaveSiteResource, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "编辑资源", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteResourceRequest{}}}},
		{Method: "DELETE", Path: "/api/v1/admin/site/resources/:id", Handler: adminCtrl.DeleteSiteResource, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "删除资源", Tags: []string{"Admin · Site"}}},

		{Method: "GET", Path: "/api/v1/admin/site/tech-stacks", Handler: adminCtrl.ListSiteTechStacks, Auth: true, Permission: "site:read", Doc: docs.Op{Summary: "技术栈列表", Tags: []string{"Admin · Site"}, Params: pageQuery}},
		{Method: "POST", Path: "/api/v1/admin/site/tech-stacks", Handler: adminCtrl.SaveSiteTechStack, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "创建技术栈", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteTechStackRequest{}}}},
		{Method: "PUT", Path: "/api/v1/admin/site/tech-stacks/:id", Handler: adminCtrl.SaveSiteTechStack, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "编辑技术栈", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteTechStackRequest{}}}},
		{Method: "DELETE", Path: "/api/v1/admin/site/tech-stacks/:id", Handler: adminCtrl.DeleteSiteTechStack, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "删除技术栈", Tags: []string{"Admin · Site"}}},

		{Method: "GET", Path: "/api/v1/admin/site/projects", Handler: adminCtrl.ListSiteProjects, Auth: true, Permission: "site:read", Doc: docs.Op{Summary: "项目列表", Tags: []string{"Admin · Site"}, Params: pageQuery}},
		{Method: "POST", Path: "/api/v1/admin/site/projects", Handler: adminCtrl.SaveSiteProject, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "创建项目", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteProjectRequest{}}}},
		{Method: "PUT", Path: "/api/v1/admin/site/projects/:id", Handler: adminCtrl.SaveSiteProject, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "编辑项目", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteProjectRequest{}}}},
		{Method: "DELETE", Path: "/api/v1/admin/site/projects/:id", Handler: adminCtrl.DeleteSiteProject, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "删除项目", Tags: []string{"Admin · Site"}}},

		{Method: "GET", Path: "/api/v1/admin/site/timeline", Handler: adminCtrl.ListSiteTimelineEvents, Auth: true, Permission: "site:read", Doc: docs.Op{Summary: "时间轴列表", Tags: []string{"Admin · Site"}, Params: pageQuery}},
		{Method: "POST", Path: "/api/v1/admin/site/timeline", Handler: adminCtrl.SaveSiteTimelineEvent, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "创建时间轴事件", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteTimelineEventRequest{}}}},
		{Method: "PUT", Path: "/api/v1/admin/site/timeline/:id", Handler: adminCtrl.SaveSiteTimelineEvent, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "编辑时间轴事件", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteTimelineEventRequest{}}}},
		{Method: "DELETE", Path: "/api/v1/admin/site/timeline/:id", Handler: adminCtrl.DeleteSiteTimelineEvent, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "删除时间轴事件", Tags: []string{"Admin · Site"}}},

		{Method: "GET", Path: "/api/v1/admin/site/messages", Handler: adminCtrl.ListSiteMessages, Auth: true, Permission: "site:read", Doc: docs.Op{Summary: "留言列表", Tags: []string{"Admin · Site"}, Params: pageQuery}},
		{Method: "PUT", Path: "/api/v1/admin/site/messages/:id", Handler: adminCtrl.SaveSiteMessage, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "回复/审核留言", Tags: []string{"Admin · Site"}, Body: docs.Body{Required: true, Schema: controllers.SiteMessageRequest{}}}},
		{Method: "DELETE", Path: "/api/v1/admin/site/messages/:id", Handler: adminCtrl.DeleteSiteMessage, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "删除留言", Tags: []string{"Admin · Site"}}},

		{Method: "GET", Path: "/api/v1/admin/site/analytics", Handler: adminCtrl.SiteAnalytics, Auth: true, Permission: "site:read", Doc: docs.Op{Summary: "官网访问数据分析", Tags: []string{"Admin · Site"}}},
		{Method: "POST", Path: "/api/v1/admin/site/upload", Handler: adminCtrl.UploadSiteAsset, Auth: true, Permission: "site:write", Doc: docs.Op{Summary: "上传官网素材", Tags: []string{"Admin · Site"}}},
	}

	specs = append(specs,
		routeSpec{Method: "PUT", Path: "/api/v1/admin/ai/rag/feedback/:id/status", Handler: adminCtrl.UpdateRAGFeedbackStatus, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "处理 RAG 反馈状态", Tags: []string{"Admin 路 System"}, Body: docs.Body{Required: true, Schema: controllers.RAGFeedbackStatusRequest{}}}},
		routeSpec{Method: "POST", Path: "/api/v1/admin/ai/rag/feedback/:id/eval-case", Handler: adminCtrl.ConvertRAGFeedbackToEvalCase, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "RAG 反馈转评测用例", Tags: []string{"Admin 路 System"}}},
		routeSpec{Method: "GET", Path: "/api/v1/admin/ai/rag/config", Handler: adminCtrl.RAGConfig, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "RAG 检索调参配置", Tags: []string{"Admin 路 System"}}},
		routeSpec{Method: "PUT", Path: "/api/v1/admin/ai/rag/config", Handler: adminCtrl.SaveRAGConfig, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "保存 RAG 检索调参", Tags: []string{"Admin 路 System"}, Body: docs.Body{Required: true, Schema: models.RAGConfig{}}}},
		routeSpec{Method: "GET", Path: "/api/v1/admin/ai/rag/analytics", Handler: adminCtrl.RAGAnalytics, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "RAG 命中分析", Tags: []string{"Admin 路 System"}}},
		routeSpec{Method: "GET", Path: "/api/v1/admin/ai/rag/eval-cases", Handler: adminCtrl.ListRAGEvalCases, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "RAG 评测用例列表", Tags: []string{"Admin 路 System"}}},
		routeSpec{Method: "POST", Path: "/api/v1/admin/ai/rag/eval-cases", Handler: adminCtrl.SaveRAGEvalCase, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "创建 RAG 评测用例", Tags: []string{"Admin 路 System"}, Body: docs.Body{Required: true, Schema: models.RAGEvalCase{}}}},
		routeSpec{Method: "PUT", Path: "/api/v1/admin/ai/rag/eval-cases/:id", Handler: adminCtrl.SaveRAGEvalCase, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "更新 RAG 评测用例", Tags: []string{"Admin 路 System"}, Body: docs.Body{Required: true, Schema: models.RAGEvalCase{}}}},
		routeSpec{Method: "DELETE", Path: "/api/v1/admin/ai/rag/eval-cases/:id", Handler: adminCtrl.DeleteRAGEvalCase, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "删除 RAG 评测用例", Tags: []string{"Admin 路 System"}}},
		routeSpec{Method: "GET", Path: "/api/v1/admin/ai/rag/evals/runs", Handler: adminCtrl.ListRAGEvalRuns, Auth: true, Permission: "ai:assistant", Doc: docs.Op{Summary: "RAG 评测运行历史", Tags: []string{"Admin 路 System"}}},
	)

	specs = append(specs,
		routeSpec{Method: "GET", Path: "/api/v1/admin/ai/call-logs", Handler: adminCtrl.ListAIModelCallLogs, Auth: true, Permission: "ai:models:read", Doc: docs.Op{Summary: "AI 模型调用日志", Tags: []string{"Admin 路 System"}, Params: pageQuery}},
		routeSpec{Method: "GET", Path: "/api/v1/admin/ai/call-stats", Handler: adminCtrl.AIModelCallStats, Auth: true, Permission: "ai:models:read", Doc: docs.Op{Summary: "AI 模型调用统计", Tags: []string{"Admin 路 System"}}},
		routeSpec{Method: "GET", Path: "/api/v1/admin/system/settings", Handler: adminCtrl.ListSystemSettings, Auth: true, Permission: "admin:access", Doc: docs.Op{Summary: "系统配置中心", Tags: []string{"Admin 路 System"}}},
		routeSpec{Method: "PUT", Path: "/api/v1/admin/system/settings", Handler: adminCtrl.SaveSystemSettings, Auth: true, Permission: "admin:access", Doc: docs.Op{Summary: "保存系统配置", Tags: []string{"Admin 路 System"}, Body: docs.Body{Required: true, Schema: controllers.SystemSettingsRequest{}}}},
		routeSpec{Method: "GET", Path: "/api/v1/admin/site/operations", Handler: adminCtrl.SiteOperationsDashboard, Auth: true, Permission: "site:read", Doc: docs.Op{Summary: "官网运营仪表盘", Tags: []string{"Admin 路 Site"}}},
	)

	return specs
}

func pingHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "pong"}) }
}

// extractAccountKey 从登录请求 body 中读取 account/email 作为限流维度键。
// 使用 ShouldBindBodyWith 让后续 handler 依然能读到 body。
func extractAccountKey(c *gin.Context) string {
	var body struct {
		Account string `json:"account"`
		Email   string `json:"email"`
	}
	// Gin 1.10 提供 ShouldBindBodyWith，缓存原始 body 供后续 ShouldBindJSON 使用
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		return ""
	}
	if body.Account != "" {
		return body.Account
	}
	return body.Email
}
