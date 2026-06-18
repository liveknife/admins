package models

import "time"

// User 用户模型
type User struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
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
