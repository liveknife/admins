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
	// 系统资源监控
	SystemResources *SystemResource `json:"system_resources,omitempty"`
	// 访问来源分布（近7天）
	SourceStats []SourceStat `json:"source_stats,omitempty"`
	// 消息类型统计
	MessageTypeStats []MessageTypeStat `json:"message_type_stats,omitempty"`
}

// SystemResource 系统资源使用率
type SystemResource struct {
	CPUUsage    float64 `json:"cpu_usage"`    // CPU 使用率 (0-100)
	MemoryUsage float64 `json:"memory_usage"` // 内存使用率 (0-100)
	DiskUsage   float64 `json:"disk_usage"`   // 磁盘使用率 (0-100)
	// 24小时趋势数据（每小时一个点）
	CPUTrend    []float64 `json:"cpu_trend"`
	MemoryTrend []float64 `json:"memory_trend"`
}

// SourceStat 访问来源统计
type SourceStat struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// MessageTypeStat 消息类型统计
type MessageTypeStat struct {
	Name  string `json:"name"` // 系统消息/聊天消息/通知公告/其他消息
	Value int64  `json:"value"`
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
	Question string            `json:"question"`
	Answer   string            `json:"answer"`
	Insights []string          `json:"insights"`
	Rows     []map[string]any  `json:"rows"`
	Metrics  map[string]int64  `json:"metrics"`
	Sources  []KnowledgeSource `json:"sources,omitempty"`
}

type AIModelConfig struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Provider        string     `json:"provider"`
	APIFormat       string     `json:"api_format"`
	BaseURL         string     `json:"base_url"`
	ChatModel       string     `json:"chat_model"`
	EmbeddingModel  string     `json:"embedding_model"`
	APIKey          string     `json:"api_key,omitempty"`
	HasAPIKey       bool       `json:"has_api_key"`
	MaskedAPIKey    string     `json:"masked_api_key,omitempty"`
	Temperature     float64    `json:"temperature"`
	MaxTokens       int        `json:"max_tokens"`
	TimeoutSeconds  int        `json:"timeout_seconds"`
	ExtraJSON       string     `json:"extra_json"`
	IsDefault       bool       `json:"is_default"`
	Enabled         bool       `json:"enabled"`
	LastTestStatus  string     `json:"last_test_status"`
	LastTestMessage string     `json:"last_test_message"`
	LastTestAt      *time.Time `json:"last_test_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
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

type AdminAnnouncement struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	Question    string            `json:"question"`
	Answer      string            `json:"answer"`
	Sources     []KnowledgeSource `json:"sources"`
	Matches     []SiteResource    `json:"matches"`
	Projects    []SiteProject     `json:"projects"`
	Suggestions []string          `json:"suggestions,omitempty"`
	QueryLogID  int64             `json:"query_log_id,omitempty"`
}

type KnowledgeSource struct {
	ChunkID         int64   `json:"chunk_id,omitempty"`
	CitationID      int     `json:"citation_id,omitempty"`
	SourceType      string  `json:"source_type"`
	SourceID        int64   `json:"source_id"`
	Visibility      string  `json:"visibility,omitempty"`
	Title           string  `json:"title"`
	Summary         string  `json:"summary"`
	Score           float64 `json:"score"`
	VectorScore     float64 `json:"vector_score,omitempty"`
	BM25Score       float64 `json:"bm25_score,omitempty"`
	KeywordScore    float64 `json:"keyword_score,omitempty"`
	SourceWeight    float64 `json:"source_weight,omitempty"`
	RerankScore     float64 `json:"rerank_score,omitempty"`
	Threshold       float64 `json:"threshold,omitempty"`
	URL             string  `json:"url,omitempty"`
	Snippet         string  `json:"snippet,omitempty"`
	HighlightedText string  `json:"highlighted_text,omitempty"`
}

type KnowledgeChunkPreview struct {
	ID         int64          `json:"id"`
	SourceType string         `json:"source_type"`
	SourceID   int64          `json:"source_id"`
	Visibility string         `json:"visibility"`
	Title      string         `json:"title"`
	Summary    string         `json:"summary"`
	Content    string         `json:"content"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	TokenCount int            `json:"token_count"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type RAGIndexStats struct {
	TotalChunks        int64              `json:"total_chunks"`
	BySource           map[string]int64   `json:"by_source"`
	ByVisibility       map[string]int64   `json:"by_visibility"`
	TopK               int                `json:"top_k"`
	MinScore           float64            `json:"min_score"`
	RerankTopN         int                `json:"rerank_top_n"`
	SourceWeights      map[string]float64 `json:"source_weights"`
	ChatEnabled        bool               `json:"chat_enabled"`
	StreamingEnabled   bool               `json:"streaming_enabled"`
	VectorBackend      string             `json:"vector_backend"`
	PGVectorAvailable  bool               `json:"pgvector_available"`
	UpdatedAt          *time.Time         `json:"updated_at,omitempty"`
	LatestJob          *RAGIndexJob       `json:"latest_job,omitempty"`
	QueryCount         int64              `json:"query_count"`
	HitCount           int64              `json:"hit_count"`
	AverageLatencyMs   float64            `json:"average_latency_ms"`
	AverageSourceCount float64            `json:"average_source_count"`
	FeedbackCount      int64              `json:"feedback_count"`
	PositiveFeedback   int64              `json:"positive_feedback"`
	NegativeFeedback   int64              `json:"negative_feedback"`
}

type RAGEvalCase struct {
	ID              string   `json:"id"`
	Question        string   `json:"question"`
	ExpectedSources []string `json:"expected_sources,omitempty"`
	ExpectedTerms   []string `json:"expected_terms,omitempty"`
}

type RAGEvalCaseResult struct {
	Case          RAGEvalCase       `json:"case"`
	Matched       bool              `json:"matched"`
	RecallHit     bool              `json:"recall_hit"`
	AnswerQuality float64           `json:"answer_quality"`
	TopScore      float64           `json:"top_score"`
	LatencyMs     int64             `json:"latency_ms"`
	Sources       []KnowledgeSource `json:"sources"`
	Answer        string            `json:"answer"`
}

type RAGEvalRun struct {
	Total            int                 `json:"total"`
	Matched          int                 `json:"matched"`
	RecallHits       int                 `json:"recall_hits"`
	AverageTopScore  float64             `json:"average_top_score"`
	AverageQuality   float64             `json:"average_quality"`
	AverageLatencyMs float64             `json:"average_latency_ms"`
	Results          []RAGEvalCaseResult `json:"results"`
	CreatedAt        time.Time           `json:"created_at"`
}

type RAGIndexJob struct {
	ID           int64      `json:"id"`
	JobType      string     `json:"job_type"`
	Status       string     `json:"status"`
	RetryCount   int        `json:"retry_count"`
	MaxRetries   int        `json:"max_retries"`
	ErrorMessage string     `json:"error_message"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type RAGQueryLog struct {
	ID            int64     `json:"id"`
	Question      string    `json:"question"`
	Answer        string    `json:"answer"`
	Matched       bool      `json:"matched"`
	SourceCount   int       `json:"source_count"`
	TopScore      float64   `json:"top_score"`
	LatencyMs     int64     `json:"latency_ms"`
	UsedChatModel bool      `json:"used_chat_model"`
	SourceJSON    string    `json:"source_json"`
	CreatedAt     time.Time `json:"created_at"`
}

type RAGFeedback struct {
	ID         int64     `json:"id"`
	QueryLogID int64     `json:"query_log_id"`
	Question   string    `json:"question"`
	Rating     string    `json:"rating"`
	Comment    string    `json:"comment"`
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type UploadedDocument struct {
	ID           int64     `json:"id"`
	OriginalName string    `json:"original_name"`
	FileName     string    `json:"file_name"`
	FilePath     string    `json:"file_path"`
	MimeType     string    `json:"mime_type"`
	FileSize     int64     `json:"file_size"`
	Visibility   string    `json:"visibility"`
	TextContent  string    `json:"text_content,omitempty"`
	ChunkCount   int       `json:"chunk_count"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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
