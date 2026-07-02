package models

import "time"

// User 用户模型
type User struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	AvatarURL   string     `json:"avatar_url"`
	Roles       []string   `json:"roles"`
	Permissions []string   `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// TokenPair JWT 令牌对
type TokenPair struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresIn  int64  `json:"access_token_expires_in"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
	TokenType             string `json:"token_type"`
}

// Claims JWT 自定义声明
type Claims struct {
	UserID  int64  `json:"user_id"`
	Account string `json:"account"`
	Email   string `json:"email"`
	Type    string `json:"type"`
}

// Role 角色模型
type Role struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

// Permission 权限模型
type Permission struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type DashboardSummary struct {
	UserCount           int64             `json:"user_count"`
	ActiveUserCount     int64             `json:"active_user_count"`
	RoleCount           int64             `json:"role_count"`
	PermissionCount     int64             `json:"permission_count"`
	MessageCount        int64             `json:"message_count"`
	UnreadNotification  int64             `json:"unread_notification"`
	RecentLogs          []OperationLog    `json:"recent_logs"`
	RecentNotifications []Notification    `json:"recent_notifications"`
	MetricTrend         []DashboardMetric `json:"metric_trend"`
}

type DashboardMetric struct {
	Date     string `json:"date"`
	Users    int64  `json:"users"`
	Messages int64  `json:"messages"`
	Logs     int64  `json:"logs"`
}

type OperationLog struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Detail    string    `json:"detail"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

type Notification struct {
	ID        int64      `json:"id"`
	UserID    *int64     `json:"user_id,omitempty"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Type      string     `json:"type"`
	IsRead    bool       `json:"is_read"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
}

type PermissionTreeNode struct {
	ID          string               `json:"id"`
	Label       string               `json:"label"`
	Code        string               `json:"code,omitempty"`
	Type        string               `json:"type"`
	Description string               `json:"description,omitempty"`
	Children    []PermissionTreeNode `json:"children,omitempty"`
}

type RolePermissionPreview struct {
	Role        Role                 `json:"role"`
	Menus       []PermissionTreeNode `json:"menus"`
	Buttons     []PermissionTreeNode `json:"buttons"`
	Permissions []string             `json:"permissions"`
}

type AIAssistantResult struct {
	Question string           `json:"question"`
	Answer   string           `json:"answer"`
	Insights []string         `json:"insights"`
	Rows     []map[string]any `json:"rows"`
	Metrics  map[string]int64 `json:"metrics"`
}

type SystemHealth struct {
	Status    string          `json:"status"`
	CPU       HealthMetric    `json:"cpu"`
	Memory    HealthMetric    `json:"memory"`
	Database  DatabaseHealth  `json:"database"`
	WebSocket WebSocketHealth `json:"websocket"`
	API       APIHealth       `json:"api"`
	Alerts    []string        `json:"alerts"`
	CheckedAt time.Time       `json:"checked_at"`
}

type DatabaseCatalog struct {
	CurrentDatabase string   `json:"current_database"`
	Databases       []string `json:"databases"`
	Engines         []string `json:"engines"`
}

type DatabaseTable struct {
	Name      string     `json:"name"`
	Engine    string     `json:"engine"`
	Collation string     `json:"collation"`
	Rows      int64      `json:"rows"`
	IndexSize string     `json:"index_size"`
	Comment   string     `json:"comment"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

type DatabaseColumn struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	NotNull    bool   `json:"not_null"`
	Default    string `json:"default"`
	Comment    string `json:"comment"`
	PrimaryKey bool   `json:"primary_key"`
}

type SiteAnnouncement struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	LinkURL   string     `json:"link_url"`
	IsActive  bool       `json:"is_active"`
	SortOrder int        `json:"sort_order"`
	StartsAt  *time.Time `json:"starts_at,omitempty"`
	EndsAt    *time.Time `json:"ends_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type SiteBanner struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Subtitle  string    `json:"subtitle"`
	ImageURL  string    `json:"image_url"`
	LinkURL   string    `json:"link_url"`
	IsActive  bool      `json:"is_active"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SiteResource struct {
	ID              int64      `json:"id"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Summary         string     `json:"summary"`
	Content         string     `json:"content"`
	MarkdownContent string     `json:"markdown_content"`
	Category        string     `json:"category"`
	CoverURL        string     `json:"cover_url"`
	LinkURL         string     `json:"link_url"`
	Tags            string     `json:"tags"`
	SEOTitle        string     `json:"seo_title"`
	SEODescription  string     `json:"seo_description"`
	SEOKeywords     string     `json:"seo_keywords"`
	Status          string     `json:"status"`
	IsFeatured      bool       `json:"is_featured"`
	ViewCount       int64      `json:"view_count"`
	SortOrder       int        `json:"sort_order"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type SiteTechStack struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Level       int       `json:"level"`
	IconURL     string    `json:"icon_url"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SiteProject struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Summary     string     `json:"summary"`
	Description string     `json:"description"`
	CoverURL    string     `json:"cover_url"`
	DemoURL     string     `json:"demo_url"`
	RepoURL     string     `json:"repo_url"`
	StackTags   string     `json:"stack_tags"`
	Status      string     `json:"status"`
	IsFeatured  bool       `json:"is_featured"`
	SortOrder   int        `json:"sort_order"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type SiteTimelineEvent struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Summary     string     `json:"summary"`
	Content     string     `json:"content"`
	Phase       string     `json:"phase"`
	EventType   string     `json:"event_type"`
	Tags        string     `json:"tags"`
	LinkURL     string     `json:"link_url"`
	Status      string     `json:"status"`
	IsFeatured  bool       `json:"is_featured"`
	SortOrder   int        `json:"sort_order"`
	HappenedAt  *time.Time `json:"happened_at,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type SiteHome struct {
	Announcements []SiteAnnouncement  `json:"announcements"`
	Banners       []SiteBanner        `json:"banners"`
	Resources     []SiteResource      `json:"resources"`
	TechStacks    []SiteTechStack     `json:"tech_stacks"`
	Projects      []SiteProject       `json:"projects"`
	Timeline      []SiteTimelineEvent `json:"timeline"`
	Messages      []SiteMessage       `json:"messages"`
	Analytics     SitePublicStats     `json:"analytics"`
}

type SiteMessage struct {
	ID          int64     `json:"id"`
	VisitorName string    `json:"visitor_name"`
	Email       string    `json:"email,omitempty"`
	Content     string    `json:"content"`
	Reply       string    `json:"reply"`
	Status      string    `json:"status"`
	IsPublic    bool      `json:"is_public"`
	IPAddress   string    `json:"ip_address,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SitePublicStats struct {
	VisitCount   int64 `json:"visit_count"`
	ArticleCount int64 `json:"article_count"`
	MessageCount int64 `json:"message_count"`
}

type SiteKnowledgeAnswer struct {
	Question string         `json:"question"`
	Answer   string         `json:"answer"`
	Matches  []SiteResource `json:"matches"`
	Projects []SiteProject  `json:"projects"`
}

type SiteVisitBucket struct {
	Date   string `json:"date"`
	Visits int64  `json:"visits"`
}

type SiteVisitTopPage struct {
	Path   string `json:"path"`
	Visits int64  `json:"visits"`
}

type SiteVisitDevice struct {
	Device string `json:"device"`
	Visits int64  `json:"visits"`
}

type SiteAnalytics struct {
	VisitCount      int64              `json:"visit_count"`
	TodayVisits     int64              `json:"today_visits"`
	ArticleCount    int64              `json:"article_count"`
	MessageCount    int64              `json:"message_count"`
	PendingMessages int64              `json:"pending_messages"`
	VisitsByDay     []SiteVisitBucket  `json:"visits_by_day"`
	TopPages        []SiteVisitTopPage `json:"top_pages"`
	DeviceStats     []SiteVisitDevice  `json:"device_stats"`
	TopArticles     []SiteResource     `json:"top_articles"`
}

type HealthMetric struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
	Label string  `json:"label"`
}

type DatabaseHealth struct {
	Status         string `json:"status"`
	OpenConnection int    `json:"open_connection"`
	InUse          int    `json:"in_use"`
	Idle           int    `json:"idle"`
	WaitCount      int64  `json:"wait_count"`
	PingMS         int64  `json:"ping_ms"`
}

type WebSocketHealth struct {
	OnlineUsers int `json:"online_users"`
	Connections int `json:"connections"`
}

type APIHealth struct {
	TotalRequests int64              `json:"total_requests"`
	AverageMS     float64            `json:"average_ms"`
	SlowRequests  int64              `json:"slow_requests"`
	StatusCodes   map[string]int64   `json:"status_codes"`
	TopPaths      []APIMetricSummary `json:"top_paths"`
}

type APIMetricSummary struct {
	Path      string  `json:"path"`
	Method    string  `json:"method"`
	Count     int64   `json:"count"`
	AverageMS float64 `json:"average_ms"`
}

// ──────────────────────────────────────────────
// 聊天相关模型
// ──────────────────────────────────────────────

// ChatUser 聊天用户（含在线状态和未读数）
type ChatUser struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	Online      bool      `json:"online"`
	UnreadCount int64     `json:"unread_count"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	ID          int64     `json:"id"`
	FromUserID  int64     `json:"from_user_id"`
	ToUserID    int64     `json:"to_user_id"`
	MessageType string    `json:"message_type"`
	Content     string    `json:"content"`
	MediaURL    string    `json:"media_url"`
	FileName    string    `json:"file_name"`
	MimeType    string    `json:"mime_type"`
	FileSize    int64     `json:"file_size"`
	Transcript  string    `json:"transcript"`
	Translation string    `json:"translation"`
	IsRead      bool      `json:"is_read"`
	CreatedAt   time.Time `json:"created_at"`
}
