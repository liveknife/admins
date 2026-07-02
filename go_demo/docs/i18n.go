package docs

// i18n 里的字典用于把 routes/routes.go 里写的中文 summary / description / 参数说明
// 映射成英文。原则：
//   - 找不到映射时直接返回原文（保底不丢字）
//   - 只翻译"可读文案"，不动路径、字段名、schema 名等技术性字符串
//   - 简单的短语一次映射；长句子按整句映射，比正则拆词更可控

// zhToEnDict 中文 → 英文字典。routes.go 里新增中文文案时，来这里补一行即可。
var zhToEnDict = map[string]string{
	// ── OpenAPI 顶层信息 ──
	"后台管理平台 API 文档。所有以 `/api/v1/admin/*` 开头的路由需要 Bearer JWT，并按权限码授权。": "Admin platform API. All routes prefixed with `/api/v1/admin/*` require a Bearer JWT and a matching permission code.",

	// ── Tags ──
	"Public":              "Public",
	"Auth":                "Auth",
	"Site (Public)":       "Site (Public)",
	"Chat":                "Chat",
	"Profile":             "Profile",
	"Admin · Dashboard":   "Admin · Dashboard",
	"Admin · Users":       "Admin · Users",
	"Admin · Roles":       "Admin · Roles",
	"Admin · Logs":        "Admin · Logs",
	"Admin · Notifications": "Admin · Notifications",
	"Admin · System":      "Admin · System",
	"Admin · Site":        "Admin · Site",

	// ── 公开接口 summary ──
	"健康探针 (Ping)":           "Health probe (ping)",
	"获取 RSA 密码传输公钥":         "Fetch RSA public key for password transport",
	"获取图形验证码":               "Fetch an image captcha",

	// ── Auth ──
	"注册用户":         "Register user",
	"登录":           "Login",
	"刷新访问令牌":       "Refresh access token",
	"创建密码重置令牌":     "Create a password reset token",
	"使用令牌重置密码":     "Reset password with a token",
	"获取当前登录用户":     "Get current logged-in user",

	// ── Profile ──
	"更新当前用户资料": "Update current user profile",
	"修改当前用户密码": "Change current user password",
	"上传头像":     "Upload avatar",

	// ── Site (Public) ──
	"官网首页聚合数据":        "Aggregated site home data",
	"按 slug 或 ID 获取文章详情": "Fetch article detail by slug or numeric ID",
	"全文搜索已发布文章":      "Full-text search across published articles",
	"官网知识库问答":         "Ask the site knowledge base",
	"提交访客留言":          "Submit a visitor message",
	"上报访问统计":          "Report a page visit",

	// ── Chat ──
	"聊天 WebSocket 入口":     "Chat WebSocket entry",
	"聊天用户列表":            "List chat users",
	"查询与指定用户的聊天历史":       "Fetch chat history with a user",
	"上传聊天附件":            "Upload a chat attachment",
	"翻译聊天消息":            "Translate a chat message",
	"标记与该用户的消息为已读":       "Mark all messages with this user as read",

	// ── Admin · Dashboard ──
	"管理端仪表盘概览": "Admin dashboard overview",

	// ── Admin · Users ──
	"用户列表":                "List users",
	"创建用户":                "Create user",
	"编辑用户":                "Update user",
	"分配用户角色":              "Assign roles to user",
	"获取用户明文密码（密码保险箱）":     "Retrieve user plaintext password (vault)",
	"重置用户密码":              "Reset user password",
	"停用用户":                "Deactivate user",

	// ── Admin · Roles ──
	"角色列表":         "List roles",
	"创建角色":         "Create role",
	"编辑角色":         "Update role",
	"删除角色":         "Delete role",
	"权限列表":         "List permissions",
	"权限树（菜单 + 按钮）": "Permission tree (menu + button)",
	"角色权限预览":       "Preview role permissions",

	// ── Admin · Logs / Notifications ──
	"操作日志分页":    "Paginated operation logs",
	"通知分页":      "Paginated notifications",
	"未读通知数":     "Unread notification count",
	"标记单条已读":    "Mark one notification as read",
	"全部标记已读":    "Mark all notifications as read",

	// ── Admin · System ──
	"AI 助手提问":         "Ask the AI assistant",
	"系统健康监控":         "System health metrics",
	"数据库元信息 (catalog)": "Database catalog metadata",
	"数据库表列表":         "List database tables",
	"数据库表字段":         "List columns of a table",

	// ── Admin · Site（复用同一批"列表/创建/编辑/删除 + 上传"套路） ──
	"公告列表":          "List announcements",
	"创建公告":          "Create announcement",
	"编辑公告":          "Update announcement",
	"删除公告":          "Delete announcement",
	"轮播列表":          "List banners",
	"创建轮播":          "Create banner",
	"编辑轮播":          "Update banner",
	"删除轮播":          "Delete banner",
	"资源列表":          "List articles",
	"创建资源":          "Create article",
	"编辑资源":          "Update article",
	"删除资源":          "Delete article",
	"技术栈列表":         "List tech stacks",
	"创建技术栈":         "Create tech stack",
	"编辑技术栈":         "Update tech stack",
	"删除技术栈":         "Delete tech stack",
	"项目列表":          "List projects",
	"创建项目":          "Create project",
	"编辑项目":          "Update project",
	"删除项目":          "Delete project",
	"时间轴列表":         "List timeline events",
	"创建时间轴事件":       "Create timeline event",
	"编辑时间轴事件":       "Update timeline event",
	"删除时间轴事件":       "Delete timeline event",
	"留言列表":          "List messages",
	"回复/审核留言":       "Reply / moderate message",
	"删除留言":          "Delete message",
	"官网访问数据分析":      "Site visit analytics",
	"上传官网素材":        "Upload a site asset",

	// ── 描述（比较长的一次映射） ──
	"支持用户名 / 邮箱 / 手机号登录；密码必须先用 `/api/v1/password-public-key` 返回的公钥加密。": "Login by username / email / phone. Password must be encrypted with the public key from `/api/v1/password-public-key`.",
	"客户端使用返回的公钥对明文密码进行 RSA-OAEP-SHA256 加密后传输。":                     "Client-side encrypts the plaintext password with RSA-OAEP-SHA256 using the returned public key.",
	"返回 base64 编码的 PNG 图形验证码和对应的 captcha_id，登录时随请求一起提交。":              "Returns a base64-encoded PNG captcha and its captcha_id, both sent alongside the login request.",
	"按标题 / 摘要 / 正文 / 标签 / 分类进行大小写不敏感的模糊匹配，命中标题权重最高。":                  "Case-insensitive fuzzy match on title / summary / body / tags / category. Title matches carry the highest rank.",
	"通过 URL 参数 `token=<access_token>` 传递 JWT。协议升级到 WebSocket 后不再走 HTTP。": "Pass the JWT via URL parameter `token=<access_token>`. After the protocol upgrade, subsequent traffic is WebSocket only.",
	"旧密码与新密码都必须先经 RSA-OAEP-SHA256 加密。":                              "Both old and new passwords must be RSA-OAEP-SHA256 encrypted first.",
	"`multipart/form-data`，字段名 `file`，允许 jpg/png/webp/gif，最大 5 MB。":  "`multipart/form-data`, field name `file`. Accepts jpg/png/webp/gif, up to 5 MB.",
	"multipart/form-data 字段 `file`":                                  "multipart/form-data field `file`",

	// ── 参数说明 ──
	"页码，从 1 开始，默认 1":       "Page number, 1-based. Default 1.",
	"每页数量，默认 10，最大 100":    "Page size. Default 10, max 100.",
	"用户名 / 邮箱 / 手机号，任选其一":  "Username / email / phone — pick any.",
	"图形验证码，字符值":           "Captcha value (characters).",
	"图形验证码 ID，来自 /captcha": "Captcha id issued by /captcha.",
	"RSA-OAEP-SHA256 加密后的 base64 密码": "Password encrypted with RSA-OAEP-SHA256, base64 encoded.",
	"头像 URL，可以通过 /me/avatar 上传后填入":  "Avatar URL. Fill in the value returned by /me/avatar upload.",
	"文章 slug 或数字 ID":            "Article slug or numeric ID.",
	"搜索关键词":                   "Search keyword.",
	"限定分类":                    "Filter by category.",
	"限定单个标签":                  "Filter by a single tag.",
	"Bearer 访问令牌":              "Bearer access token.",
	"true/false 过滤已读状态":        "true/false — filter by read state.",
	"对端用户 ID":                 "Peer user ID.",
	"错误消息":                    "Error message",
	"错误响应":                    "Error response",
	"注册成功":                    "Created",
	"账号或密码错误":                 "Invalid account or password",
	"登录尝试过于频繁":                "Too many login attempts",
	"用户已存在":                   "User already exists",
	"文章不存在":                   "Article not found",
	"已记录":                     "Recorded",

	// ── 权限提示（前缀） ──
	"需要权限：": "Required permission: ",
}

// translate 返回 s 的英文映射；找不到时原样返回。
func translate(s string) string {
	if s == "" {
		return s
	}
	if v, ok := zhToEnDict[s]; ok {
		return v
	}
	return s
}
