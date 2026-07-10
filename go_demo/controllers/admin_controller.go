package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-demo/models"
	"go-demo/services"

	"github.com/gin-gonic/gin"
)

type AdminController struct {
	authService *services.AuthService
	adminData   *services.AdminDataService
	monitor     *services.MonitorService
}

func NewAdminController(authService *services.AuthService, adminData *services.AdminDataService, monitor *services.MonitorService) *AdminController {
	return &AdminController{authService: authService, adminData: adminData, monitor: monitor}
}

// ──────────────────────────────────────────────
// 请求体
// ──────────────────────────────────────────────

type RoleRequest struct {
	Name        string   `json:"name" form:"name" binding:"required,min=2,max=50"`
	Description string   `json:"description" form:"description"`
	Permissions []string `json:"permissions" form:"permissions"`
}

type ResetUserPasswordRequest struct {
	PasswordEncrypted string `json:"password_encrypted" form:"password_encrypted" binding:"required"`
}

type SetUserRolesRequest struct {
	Roles []string `json:"roles" form:"roles" binding:"required"`
}

type CreateUserRequest struct {
	Username          string   `json:"username" form:"username" binding:"required,min=2,max=50"`
	Email             string   `json:"email" form:"email" binding:"required,email"`
	Phone             string   `json:"phone" form:"phone"`
	PasswordEncrypted string   `json:"password_encrypted" form:"password_encrypted" binding:"required"`
	Roles             []string `json:"roles" form:"roles"`
}

type UpdateUserRequest struct {
	Username string `json:"username" form:"username" binding:"required,min=2,max=50"`
	Email    string `json:"email" form:"email" binding:"required,email"`
	Phone    string `json:"phone" form:"phone"`
}

type AskAssistantRequest struct {
	Question string `json:"question" form:"question"`
}

type RAGDiagnosticsRequest struct {
	Question        string `json:"question" form:"question" binding:"required"`
	IncludeInternal bool   `json:"include_internal" form:"include_internal"`
	TopK            int    `json:"top_k" form:"top_k"`
}

type DocumentVisibilityRequest struct {
	Visibility string `json:"visibility" form:"visibility" binding:"required"`
}

type RAGFeedbackStatusRequest struct {
	Status    string `json:"status" form:"status"`
	AdminNote string `json:"admin_note" form:"admin_note"`
}

type AIModelConfigRequest struct {
	Name           string  `json:"name" form:"name" binding:"required"`
	Provider       string  `json:"provider" form:"provider"`
	APIFormat      string  `json:"api_format" form:"api_format"`
	BaseURL        string  `json:"base_url" form:"base_url"`
	APIKey         string  `json:"api_key" form:"api_key"`
	ChatModel      string  `json:"chat_model" form:"chat_model"`
	EmbeddingModel string  `json:"embedding_model" form:"embedding_model"`
	Temperature    float64 `json:"temperature" form:"temperature"`
	MaxTokens      int     `json:"max_tokens" form:"max_tokens"`
	TimeoutSeconds int     `json:"timeout_seconds" form:"timeout_seconds"`
	ExtraJSON      string  `json:"extra_json" form:"extra_json"`
	IsDefault      bool    `json:"is_default" form:"is_default"`
	Enabled        bool    `json:"enabled" form:"enabled"`
}

type CreateNotificationRequest struct {
	Title   string `json:"title"   form:"title"   binding:"required,min=1,max=160"`
	Content string `json:"content" form:"content"`
	Type    string `json:"type"    form:"type"    binding:"required,oneof=success warning danger info"`
}

type AnnouncementRequest struct {
	Title    string `json:"title"    form:"title"    binding:"required,min=1,max=160"`
	Content  string `json:"content"  form:"content"`
	Type     string `json:"type"     form:"type"     binding:"required,oneof=info success warning danger"`
	IsActive bool   `json:"is_active"`
}

type SiteKnowledgeRequest struct {
	Question string `json:"question" form:"question"`
	Context  []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"context"`
}

type SiteFeedbackRequest struct {
	QueryLogID int64  `json:"query_log_id" form:"query_log_id"`
	Question   string `json:"question" form:"question"`
	Rating     string `json:"rating" form:"rating"`
	Comment    string `json:"comment" form:"comment"`
}

type SiteCodeExplainRequest struct {
	Code     string `json:"code" form:"code" binding:"required"`
	Language string `json:"language" form:"language"`
	Question string `json:"question" form:"question"`
}

type SiteSearchSummarizeRequest struct {
	Query string `json:"query" form:"query" binding:"required"`
}

type SiteVisitRequest struct {
	Path     string `json:"path" form:"path"`
	Referrer string `json:"referrer" form:"referrer"`
	Device   string `json:"device" form:"device"`
}

type DatabaseTablesQuery struct {
	Database string `form:"database"`
	Table    string `form:"table"`
	Engine   string `form:"engine"`
	Comment  string `form:"comment"`
}

type SiteListQuery struct {
	Status string `form:"status"`
}

type SiteAnnouncementRequest struct {
	Title     string     `json:"title" form:"title" binding:"required"`
	Content   string     `json:"content" form:"content"`
	LinkURL   string     `json:"link_url" form:"link_url"`
	IsActive  bool       `json:"is_active" form:"is_active"`
	SortOrder int        `json:"sort_order" form:"sort_order"`
	StartsAt  *time.Time `json:"starts_at" form:"starts_at"`
	EndsAt    *time.Time `json:"ends_at" form:"ends_at"`
}

type SiteBannerRequest struct {
	Title     string `json:"title" form:"title" binding:"required"`
	Subtitle  string `json:"subtitle" form:"subtitle"`
	ImageURL  string `json:"image_url" form:"image_url"`
	LinkURL   string `json:"link_url" form:"link_url"`
	IsActive  bool   `json:"is_active" form:"is_active"`
	SortOrder int    `json:"sort_order" form:"sort_order"`
}

type SiteResourceRequest struct {
	Title           string     `json:"title" form:"title" binding:"required"`
	Slug            string     `json:"slug" form:"slug"`
	Summary         string     `json:"summary" form:"summary"`
	Content         string     `json:"content" form:"content"`
	MarkdownContent string     `json:"markdown_content" form:"markdown_content"`
	Category        string     `json:"category" form:"category"`
	CoverURL        string     `json:"cover_url" form:"cover_url"`
	LinkURL         string     `json:"link_url" form:"link_url"`
	Tags            string     `json:"tags" form:"tags"`
	SEOTitle        string     `json:"seo_title" form:"seo_title"`
	SEODescription  string     `json:"seo_description" form:"seo_description"`
	SEOKeywords     string     `json:"seo_keywords" form:"seo_keywords"`
	Status          string     `json:"status" form:"status"`
	IsFeatured      bool       `json:"is_featured" form:"is_featured"`
	SortOrder       int        `json:"sort_order" form:"sort_order"`
	PublishedAt     *time.Time `json:"published_at" form:"published_at"`
}

type SiteTechStackRequest struct {
	Name        string `json:"name" form:"name" binding:"required"`
	Category    string `json:"category" form:"category"`
	Level       int    `json:"level" form:"level"`
	IconURL     string `json:"icon_url" form:"icon_url"`
	Description string `json:"description" form:"description"`
	IsActive    bool   `json:"is_active" form:"is_active"`
	SortOrder   int    `json:"sort_order" form:"sort_order"`
}

type SiteProjectRequest struct {
	Name        string     `json:"name" form:"name" binding:"required"`
	Summary     string     `json:"summary" form:"summary"`
	Description string     `json:"description" form:"description"`
	CoverURL    string     `json:"cover_url" form:"cover_url"`
	DemoURL     string     `json:"demo_url" form:"demo_url"`
	RepoURL     string     `json:"repo_url" form:"repo_url"`
	StackTags   string     `json:"stack_tags" form:"stack_tags"`
	Role        string     `json:"role" form:"role"`
	Highlights  string     `json:"highlights" form:"highlights"`
	Metrics     string     `json:"metrics" form:"metrics"`
	Challenge   string     `json:"challenge" form:"challenge"`
	Solution    string     `json:"solution" form:"solution"`
	GalleryJSON string     `json:"gallery_json" form:"gallery_json"`
	Status      string     `json:"status" form:"status"`
	IsFeatured  bool       `json:"is_featured" form:"is_featured"`
	SortOrder   int        `json:"sort_order" form:"sort_order"`
	Priority    int        `json:"priority" form:"priority"`
	StartDate   *time.Time `json:"start_date" form:"start_date"`
	EndDate     *time.Time `json:"end_date" form:"end_date"`
	PublishedAt *time.Time `json:"published_at" form:"published_at"`
}

type SiteTimelineEventRequest struct {
	Title       string     `json:"title" form:"title" binding:"required"`
	Summary     string     `json:"summary" form:"summary"`
	Content     string     `json:"content" form:"content"`
	Phase       string     `json:"phase" form:"phase"`
	EventType   string     `json:"event_type" form:"event_type"`
	Tags        string     `json:"tags" form:"tags"`
	LinkURL     string     `json:"link_url" form:"link_url"`
	Status      string     `json:"status" form:"status"`
	IsFeatured  bool       `json:"is_featured" form:"is_featured"`
	SortOrder   int        `json:"sort_order" form:"sort_order"`
	HappenedAt  *time.Time `json:"happened_at" form:"happened_at"`
	PublishedAt *time.Time `json:"published_at" form:"published_at"`
}

type SiteMessageRequest struct {
	VisitorName string `json:"visitor_name" form:"visitor_name"`
	Email       string `json:"email" form:"email"`
	Content     string `json:"content" form:"content" binding:"required"`
	Reply       string `json:"reply" form:"reply"`
	Status      string `json:"status" form:"status"`
	IsPublic    bool   `json:"is_public" form:"is_public"`
}

func siteAnnouncementInput(req SiteAnnouncementRequest) services.SiteAnnouncementInput {
	return services.SiteAnnouncementInput{
		Title:     strings.TrimSpace(req.Title),
		Content:   strings.TrimSpace(req.Content),
		LinkURL:   strings.TrimSpace(req.LinkURL),
		IsActive:  req.IsActive,
		SortOrder: req.SortOrder,
		StartsAt:  req.StartsAt,
		EndsAt:    req.EndsAt,
	}
}

func siteBannerInput(req SiteBannerRequest) services.SiteBannerInput {
	return services.SiteBannerInput{
		Title:     strings.TrimSpace(req.Title),
		Subtitle:  strings.TrimSpace(req.Subtitle),
		ImageURL:  strings.TrimSpace(req.ImageURL),
		LinkURL:   strings.TrimSpace(req.LinkURL),
		IsActive:  req.IsActive,
		SortOrder: req.SortOrder,
	}
}

func siteResourceInput(req SiteResourceRequest) services.SiteResourceInput {
	return services.SiteResourceInput{
		Title:           strings.TrimSpace(req.Title),
		Slug:            strings.TrimSpace(req.Slug),
		Summary:         strings.TrimSpace(req.Summary),
		Content:         strings.TrimSpace(req.Content),
		MarkdownContent: strings.TrimSpace(req.MarkdownContent),
		Category:        strings.TrimSpace(req.Category),
		CoverURL:        strings.TrimSpace(req.CoverURL),
		LinkURL:         strings.TrimSpace(req.LinkURL),
		Tags:            strings.TrimSpace(req.Tags),
		SEOTitle:        strings.TrimSpace(req.SEOTitle),
		SEODescription:  strings.TrimSpace(req.SEODescription),
		SEOKeywords:     strings.TrimSpace(req.SEOKeywords),
		Status:          strings.TrimSpace(req.Status),
		IsFeatured:      req.IsFeatured,
		SortOrder:       req.SortOrder,
		PublishedAt:     req.PublishedAt,
	}
}

func aiModelConfigInput(req AIModelConfigRequest) services.AIModelConfigInput {
	return services.AIModelConfigInput{
		Name:           strings.TrimSpace(req.Name),
		Provider:       strings.TrimSpace(req.Provider),
		APIFormat:      strings.TrimSpace(req.APIFormat),
		BaseURL:        strings.TrimSpace(req.BaseURL),
		APIKey:         strings.TrimSpace(req.APIKey),
		ChatModel:      strings.TrimSpace(req.ChatModel),
		EmbeddingModel: strings.TrimSpace(req.EmbeddingModel),
		Temperature:    req.Temperature,
		MaxTokens:      req.MaxTokens,
		TimeoutSeconds: req.TimeoutSeconds,
		ExtraJSON:      strings.TrimSpace(req.ExtraJSON),
		IsDefault:      req.IsDefault,
		Enabled:        req.Enabled,
	}
}

func siteTechStackInput(req SiteTechStackRequest) services.SiteTechStackInput {
	return services.SiteTechStackInput{
		Name:        strings.TrimSpace(req.Name),
		Category:    strings.TrimSpace(req.Category),
		Level:       req.Level,
		IconURL:     strings.TrimSpace(req.IconURL),
		Description: strings.TrimSpace(req.Description),
		IsActive:    req.IsActive,
		SortOrder:   req.SortOrder,
	}
}

func siteProjectInput(req SiteProjectRequest) services.SiteProjectInput {
	return services.SiteProjectInput{
		Name:        strings.TrimSpace(req.Name),
		Summary:     strings.TrimSpace(req.Summary),
		Description: strings.TrimSpace(req.Description),
		CoverURL:    strings.TrimSpace(req.CoverURL),
		DemoURL:     strings.TrimSpace(req.DemoURL),
		RepoURL:     strings.TrimSpace(req.RepoURL),
		StackTags:   strings.TrimSpace(req.StackTags),
		Role:        strings.TrimSpace(req.Role),
		Highlights:  strings.TrimSpace(req.Highlights),
		Metrics:     strings.TrimSpace(req.Metrics),
		Challenge:   strings.TrimSpace(req.Challenge),
		Solution:    strings.TrimSpace(req.Solution),
		GalleryJSON: strings.TrimSpace(req.GalleryJSON),
		Status:      strings.TrimSpace(req.Status),
		IsFeatured:  req.IsFeatured,
		SortOrder:   req.SortOrder,
		Priority:    req.Priority,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		PublishedAt: req.PublishedAt,
	}
}

func siteTimelineEventInput(req SiteTimelineEventRequest) services.SiteTimelineEventInput {
	return services.SiteTimelineEventInput{
		Title:       strings.TrimSpace(req.Title),
		Summary:     strings.TrimSpace(req.Summary),
		Content:     strings.TrimSpace(req.Content),
		Phase:       strings.TrimSpace(req.Phase),
		EventType:   strings.TrimSpace(req.EventType),
		Tags:        strings.TrimSpace(req.Tags),
		LinkURL:     strings.TrimSpace(req.LinkURL),
		Status:      strings.TrimSpace(req.Status),
		IsFeatured:  req.IsFeatured,
		SortOrder:   req.SortOrder,
		HappenedAt:  req.HappenedAt,
		PublishedAt: req.PublishedAt,
	}
}

func siteMessageInput(req SiteMessageRequest, g *gin.Context) services.SiteMessageInput {
	return services.SiteMessageInput{
		VisitorName: strings.TrimSpace(req.VisitorName),
		Email:       strings.TrimSpace(req.Email),
		Content:     strings.TrimSpace(req.Content),
		Reply:       strings.TrimSpace(req.Reply),
		Status:      strings.TrimSpace(req.Status),
		IsPublic:    req.IsPublic,
		IPAddress:   g.ClientIP(),
		UserAgent:   g.Request.UserAgent(),
	}
}

// ──────────────────────────────────────────────
// 用户管理
// ──────────────────────────────────────────────

func (c *AdminController) ListUsers(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	users, total, err := c.authService.ListUsersPaged(g.Request.Context(), page, pageSize)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"users": users, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) SetUserRoles(g *gin.Context) {
	userID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	var req SetUserRolesRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := c.authService.SetUserRoles(g.Request.Context(), userID, req.Roles)
	if errors.Is(err, services.ErrRoleNotFound) {
		g.JSON(http.StatusBadRequest, gin.H{"error": "role not found"})
		return
	}
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set user roles"})
		return
	}
	c.logAction(g, "\u8bbe\u7f6e\u7528\u6237\u89d2\u8272", "\u7528\u6237\u7ba1\u7406", fmt.Sprintf("\u7528\u6237\uff1a%s\uff0c\u89d2\u8272\uff1a%v", user.Username, req.Roles))
	g.JSON(http.StatusOK, gin.H{"user": user})
}

func (c *AdminController) GetUserPassword(g *gin.Context) {
	userID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	pw, err := c.authService.GetStoredPassword(g.Request.Context(), userID)
	if errors.Is(err, services.ErrPasswordUnavailable) {
		g.JSON(http.StatusNotFound, gin.H{"error": "password is unavailable until the user logs in or an admin resets it"})
		return
	}
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get password"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"password": pw})
}

func (c *AdminController) ResetUserPassword(g *gin.Context) {
	userID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	var req ResetUserPasswordRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pw, err := c.authService.DecryptClientPassword(req.PasswordEncrypted)
	if err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := c.authService.ResetUserPassword(g.Request.Context(), userID, pw); err != nil {
		if errors.Is(err, services.ErrPasswordPolicy) {
			g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}
	c.logAction(g, "\u91cd\u7f6e\u7528\u6237\u5bc6\u7801", "\u7528\u6237\u7ba1\u7406", fmt.Sprintf("\u7528\u6237ID\uff1a%d", userID))
	g.JSON(http.StatusOK, gin.H{"message": "password has been reset"})
}

func (c *AdminController) DeactivateUser(g *gin.Context) {
	userID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	currentUserIDValue, exists := g.Get("user_id")
	if !exists {
		g.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	currentUserID, ok2 := currentUserIDValue.(int64)
	if !ok2 {
		g.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	err := c.authService.DeactivateUser(g.Request.Context(), currentUserID, userID)
	if errors.Is(err, services.ErrCannotDeleteSelf) {
		g.JSON(http.StatusBadRequest, gin.H{"error": "cannot deactivate current user"})
		return
	}
	if errors.Is(err, services.ErrUserNotFound) {
		g.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate user"})
		return
	}
	c.logAction(g, "\u6ce8\u9500\u7528\u6237", "\u7528\u6237\u7ba1\u7406", fmt.Sprintf("\u7528\u6237ID\uff1a%d", userID))
	g.JSON(http.StatusOK, gin.H{"message": "user has been deactivated"})
}

func (c *AdminController) DeleteUser(g *gin.Context) {
	userID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	currentUserIDValue, exists := g.Get("user_id")
	if !exists {
		g.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	currentUserID, ok2 := currentUserIDValue.(int64)
	if !ok2 {
		g.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	err := c.authService.DeleteUser(g.Request.Context(), currentUserID, userID)
	if errors.Is(err, services.ErrCannotDeleteSelf) {
		g.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete current user"})
		return
	}
	if errors.Is(err, services.ErrUserNotFound) {
		g.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}
	c.logAction(g, "\u5220\u9664\u7528\u6237", "\u7528\u6237\u7ba1\u7406", fmt.Sprintf("\u7528\u6237ID\uff1a%d", userID))
	g.JSON(http.StatusOK, gin.H{"message": "user has been deleted permanently"})
}

// ReactivateUser 恢复已禁用用户（重置密码为默认密码 Admin123）
func (c *AdminController) ReactivateUser(g *gin.Context) {
	userID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	currentUserIDValue, exists := g.Get("user_id")
	if !exists {
		g.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	currentUserID, ok2 := currentUserIDValue.(int64)
	if !ok2 {
		g.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	defaultPassword := "Admin123"
	user, err := c.authService.ReactivateUser(g.Request.Context(), currentUserID, userID, defaultPassword)
	if errors.Is(err, services.ErrCannotDeleteSelf) {
		g.JSON(http.StatusBadRequest, gin.H{"error": "cannot reactivate current user"})
		return
	}
	if errors.Is(err, services.ErrUserNotFound) {
		g.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil && err.Error() == "user is not deactivated" {
		g.JSON(http.StatusBadRequest, gin.H{"error": "user is not deactivated, reactivation not needed"})
		return
	}
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reactivate user"})
		return
	}
	c.logAction(g, "\u6062\u590d\u7528\u6237", "\u7528\u6237\u7ba1\u7406", fmt.Sprintf("\u7528\u6237\uff1a%s\uff0c\u5bc6\u7801\u5df2\u91cd\u7f6e\u4e3A\u9ed8\u8ba4\u5bc6\u7801", user.Username))
	g.JSON(http.StatusOK, gin.H{"message": "user has been reactivated with default password", "user": user})
}

func (c *AdminController) CreateUser(g *gin.Context) {
	var req CreateUserRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pw, err := c.authService.DecryptClientPassword(req.PasswordEncrypted)
	if err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := c.authService.CreateUser(g.Request.Context(), req.Username, req.Email, req.Phone, pw, req.Roles)
	if errors.Is(err, services.ErrUserExists) {
		g.JSON(http.StatusConflict, gin.H{"error": "username or email already exists"})
		return
	}
	if errors.Is(err, services.ErrRoleNotFound) {
		g.JSON(http.StatusBadRequest, gin.H{"error": "one or more roles not found"})
		return
	}
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}
	c.logAction(g, "\u521b\u5efa\u7528\u6237", "\u7528\u6237\u7ba1\u7406", fmt.Sprintf("\u7528\u6237\uff1a%s", user.Username))
	g.JSON(http.StatusCreated, gin.H{"user": user})
}

func (c *AdminController) UpdateUser(g *gin.Context) {
	userID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	var req UpdateUserRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := c.authService.UpdateUser(g.Request.Context(), userID, req.Username, req.Email, req.Phone)
	if errors.Is(err, services.ErrUserNotFound) {
		g.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if errors.Is(err, services.ErrUserExists) {
		g.JSON(http.StatusConflict, gin.H{"error": "username or email already exists"})
		return
	}
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}
	c.logAction(g, "\u7f16\u8f91\u7528\u6237", "\u7528\u6237\u7ba1\u7406", fmt.Sprintf("\u7528\u6237\uff1a%s", user.Username))
	g.JSON(http.StatusOK, gin.H{"user": user})
}

// ──────────────────────────────────────────────
// 角色管理
// ──────────────────────────────────────────────

func (c *AdminController) ListRoles(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	roles, total, err := c.authService.ListRolesPaged(g.Request.Context(), page, pageSize)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list roles"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"roles": roles, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) CreateRole(g *gin.Context) {
	var req RoleRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role, err := c.authService.CreateRole(g.Request.Context(), req.Name, req.Description, req.Permissions)
	if errors.Is(err, services.ErrRoleExists) {
		g.JSON(http.StatusConflict, gin.H{"error": "role already exists"})
		return
	}
	if errors.Is(err, services.ErrPermissionNotFound) {
		g.JSON(http.StatusBadRequest, gin.H{"error": "permission not found"})
		return
	}
	if errors.Is(err, services.ErrInvalidRole) {
		g.JSON(http.StatusBadRequest, gin.H{"error": "role name is required"})
		return
	}
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create role"})
		return
	}
	c.logAction(g, "\u521b\u5efa\u89d2\u8272", "\u89d2\u8272\u6743\u9650", fmt.Sprintf("\u89d2\u8272\uff1a%s", role.Name))
	g.JSON(http.StatusCreated, gin.H{"role": role})
}

func (c *AdminController) UpdateRole(g *gin.Context) {
	roleID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	var req RoleRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role, err := c.authService.UpdateRole(g.Request.Context(), roleID, req.Name, req.Description, req.Permissions)
	if errors.Is(err, services.ErrRoleNotFound) {
		g.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	if errors.Is(err, services.ErrRoleExists) {
		g.JSON(http.StatusConflict, gin.H{"error": "role already exists"})
		return
	}
	if errors.Is(err, services.ErrPermissionNotFound) {
		g.JSON(http.StatusBadRequest, gin.H{"error": "permission not found"})
		return
	}
	if errors.Is(err, services.ErrInvalidRole) {
		g.JSON(http.StatusBadRequest, gin.H{"error": "role name is required"})
		return
	}
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
		return
	}
	c.logAction(g, "\u7f16\u8f91\u89d2\u8272", "\u89d2\u8272\u6743\u9650", fmt.Sprintf("\u89d2\u8272\uff1a%s", role.Name))
	g.JSON(http.StatusOK, gin.H{"role": role})
}

func (c *AdminController) DeleteRole(g *gin.Context) {
	roleID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	err := c.authService.DeleteRole(g.Request.Context(), roleID)
	if errors.Is(err, services.ErrRoleNotFound) {
		g.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	if errors.Is(err, services.ErrCannotDeleteRole) {
		g.JSON(http.StatusBadRequest, gin.H{"error": "system role cannot be deleted"})
		return
	}
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete role"})
		return
	}
	c.logAction(g, "\u5220\u9664\u89d2\u8272", "\u89d2\u8272\u6743\u9650", fmt.Sprintf("\u89d2\u8272ID\uff1a%d", roleID))
	g.Status(http.StatusNoContent)
}

func (c *AdminController) ListPermissions(g *gin.Context) {
	perms, err := c.authService.ListPermissions(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list permissions"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"permissions": perms})
}

func (c *AdminController) PermissionTree(g *gin.Context) {
	tree, err := c.adminData.PermissionTree(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load permission tree"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"tree": tree})
}

func (c *AdminController) RolePreview(g *gin.Context) {
	roleID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	preview, err := c.adminData.RolePreview(g.Request.Context(), roleID)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to preview role"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"preview": preview})
}

func (c *AdminController) Dashboard(g *gin.Context) {
	userID := adminCurrentUserID(g)
	data, err := c.adminData.Dashboard(g.Request.Context(), userID)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load dashboard"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"dashboard": data})
}

func (c *AdminController) ListOperationLogs(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	logs, total, err := c.adminData.ListOperationLogs(g.Request.Context(), page, pageSize)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list operation logs"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"logs": logs, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) ListNotifications(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	readStatus := g.Query("read_status")
	notices, total, err := c.adminData.ListNotifications(g.Request.Context(), adminCurrentUserID(g), page, pageSize, readStatus)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notifications"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"notifications": notices, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) UnreadNotificationCount(g *gin.Context) {
	count, err := c.adminData.UnreadNotificationCount(g.Request.Context(), adminCurrentUserID(g))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count notifications"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"count": count})
}

func (c *AdminController) MarkNotificationRead(g *gin.Context) {
	noticeID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	if err := c.adminData.MarkNotificationRead(g.Request.Context(), adminCurrentUserID(g), noticeID); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notification read"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"message": "notification marked as read"})
}

func (c *AdminController) MarkAllNotificationsRead(g *gin.Context) {
	if err := c.adminData.MarkAllNotificationsRead(g.Request.Context(), adminCurrentUserID(g)); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notifications read"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"message": "notifications marked as read"})
}

func (c *AdminController) CreateNotification(g *gin.Context) {
	var req CreateNotificationRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	notice, err := c.adminData.CreateNotification(g.Request.Context(), req.Title, req.Content, req.Type)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create notification"})
		return
	}
	g.JSON(http.StatusCreated, gin.H{"notification": notice})
}

func (c *AdminController) DeleteNotification(g *gin.Context) {
	noticeID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	if err := c.adminData.DeleteNotification(g.Request.Context(), noticeID); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete notification"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"message": "notification deleted"})
}

func (c *AdminController) ListAnnouncements(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	notices, total, err := c.adminData.ListAnnouncements(g.Request.Context(), page, pageSize)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list announcements"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"announcements": notices, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) GetActiveAnnouncement(g *gin.Context) {
	notice, err := c.adminData.GetActiveAnnouncement(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get active announcement"})
		return
	}
	if notice == nil {
		g.JSON(http.StatusOK, gin.H{"announcement": nil})
		return
	}
	g.JSON(http.StatusOK, gin.H{"announcement": notice})
}

// ListPublicAnnouncements 公开公告列表（只需登录，无需特殊权限）
func (c *AdminController) ListPublicAnnouncements(g *gin.Context) {
	notices, err := c.adminData.ListActiveAnnouncements(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list public announcements"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"announcements": notices})
}

func (c *AdminController) CreateAnnouncement(g *gin.Context) {
	var req AnnouncementRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	notice, err := c.adminData.CreateAnnouncement(g.Request.Context(), req.Title, req.Content, req.Type, req.IsActive)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create announcement"})
		return
	}
	g.JSON(http.StatusCreated, gin.H{"announcement": notice})
}

func (c *AdminController) UpdateAnnouncement(g *gin.Context) {
	noticeID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	var req AnnouncementRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	notice, err := c.adminData.UpdateAnnouncement(g.Request.Context(), noticeID, req.Title, req.Content, req.Type, req.IsActive)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update announcement"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"announcement": notice})
}

func (c *AdminController) DeleteAnnouncement(g *gin.Context) {
	noticeID, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	if err := c.adminData.DeleteAnnouncement(g.Request.Context(), noticeID); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete announcement"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"message": "announcement deleted"})
}

func (c *AdminController) AskAssistant(g *gin.Context) {
	var req AskAssistantRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := c.adminData.AskAssistant(g.Request.Context(), req.Question)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze data"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"result": result})
}

func (c *AdminController) ListAIModelConfigs(g *gin.Context) {
	items, err := c.adminData.ListAIModelConfigs(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list ai model configs"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"configs": items})
}

func (c *AdminController) SaveAIModelConfig(g *gin.Context) {
	id := int64(0)
	if raw := g.Param("id"); raw != "" {
		parsed, ok := parseIDParam(g, "id")
		if !ok {
			return
		}
		id = parsed
	}
	var req AIModelConfigRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.SaveAIModelConfig(g.Request.Context(), id, aiModelConfigInput(req))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save ai model config"})
		return
	}
	c.logAction(g, "保存大模型配置", "大模型配置", item.Name)
	g.JSON(http.StatusOK, gin.H{"config": item})
}

func (c *AdminController) SetDefaultAIModelConfig(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	item, err := c.adminData.SetDefaultAIModelConfig(g.Request.Context(), id)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate ai model config"})
		return
	}
	c.logAction(g, "启用大模型配置", "大模型配置", item.Name)
	g.JSON(http.StatusOK, gin.H{"config": item})
}

func (c *AdminController) TestAIModelConfig(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	item, err := c.adminData.TestAIModelConfig(g.Request.Context(), id)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to test ai model config"})
		return
	}
	c.logAction(g, "测试大模型配置", "大模型配置", fmt.Sprintf("%s：%s", item.Name, item.LastTestStatus))
	g.JSON(http.StatusOK, gin.H{"config": item})
}

func (c *AdminController) DeleteAIModelConfig(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	if err := c.adminData.DeleteAIModelConfig(g.Request.Context(), id); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ai model config"})
		return
	}
	c.logAction(g, "删除大模型配置", "大模型配置", fmt.Sprintf("配置ID：%d", id))
	g.Status(http.StatusNoContent)
}

func (c *AdminController) RAGIndexStats(g *gin.Context) {
	stats, err := c.adminData.RAGIndexStats(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load rag index stats"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (c *AdminController) RebuildRAGIndex(g *gin.Context) {
	job, err := c.adminData.EnqueueRAGRebuild(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue rag rebuild"})
		return
	}
	c.logAction(g, "提交 RAG 知识索引重建", "AI 助手", fmt.Sprintf("job=%d", job.ID))
	g.JSON(http.StatusAccepted, gin.H{"job": job})
}

func (c *AdminController) ListRAGIndexJobs(g *gin.Context) {
	limit := parseIntDefault(g.Query("limit"), 20)
	jobs, err := c.adminData.ListRAGIndexJobs(g.Request.Context(), limit)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rag jobs"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

func (c *AdminController) RetryRAGIndexJob(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	job, err := c.adminData.RetryRAGIndexJob(g.Request.Context(), id)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retry rag job"})
		return
	}
	c.logAction(g, "重试 RAG 知识索引任务", "AI 助手", fmt.Sprintf("job=%d", job.ID))
	g.JSON(http.StatusAccepted, gin.H{"job": job})
}

func (c *AdminController) ListRAGQueryLogs(g *gin.Context) {
	limit := parseIntDefault(g.Query("limit"), 30)
	logs, err := c.adminData.ListRAGQueryLogs(g.Request.Context(), limit)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rag query logs"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (c *AdminController) ListRAGFeedback(g *gin.Context) {
	limit := parseIntDefault(g.Query("limit"), 30)
	items, err := c.adminData.ListRAGFeedback(g.Request.Context(), limit, g.Query("rating"), g.Query("status"))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rag feedback"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"feedback": items})
}

func (c *AdminController) RAGConfig(g *gin.Context) {
	cfg, err := c.adminData.RAGConfig(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load rag config"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"config": cfg})
}

func (c *AdminController) SaveRAGConfig(g *gin.Context) {
	var req models.RAGConfig
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg, err := c.adminData.SaveRAGConfig(g.Request.Context(), req)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save rag config"})
		return
	}
	c.logAction(g, "保存 RAG 调参配置", "AI 助手", fmt.Sprintf("topK=%d minScore=%.2f", cfg.TopK, cfg.MinScore))
	g.JSON(http.StatusOK, gin.H{"config": cfg})
}

func (c *AdminController) SearchRAGDiagnostics(g *gin.Context) {
	var req RAGDiagnosticsRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sources, err := c.adminData.SearchRAGDiagnostics(g.Request.Context(), req.Question, req.IncludeInternal, req.TopK)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to run rag diagnostics"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"sources": sources})
}

func (c *AdminController) RunRAGEval(g *gin.Context) {
	includeInternal := strings.EqualFold(g.Query("include_internal"), "true")
	run, err := c.adminData.RunRAGEval(g.Request.Context(), includeInternal)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to run rag eval"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"run": run})
}

func (c *AdminController) ListRAGEvalCases(g *gin.Context) {
	items, err := c.adminData.ListRAGEvalCases(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rag eval cases"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"cases": items})
}

func (c *AdminController) SaveRAGEvalCase(g *gin.Context) {
	var req models.RAGEvalCase
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if raw := strings.TrimSpace(g.Param("id")); raw != "" {
		req.ID = raw
	}
	item, err := c.adminData.SaveRAGEvalCase(g.Request.Context(), req)
	if err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.logAction(g, "保存 RAG 评测用例", "AI 助手", item.ID)
	g.JSON(http.StatusOK, gin.H{"case": item})
}

func (c *AdminController) DeleteRAGEvalCase(g *gin.Context) {
	id := strings.TrimSpace(g.Param("id"))
	if id == "" {
		g.JSON(http.StatusBadRequest, gin.H{"error": "missing eval case id"})
		return
	}
	if err := c.adminData.DeleteRAGEvalCase(g.Request.Context(), id); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete rag eval case"})
		return
	}
	c.logAction(g, "删除 RAG 评测用例", "AI 助手", id)
	g.Status(http.StatusNoContent)
}

func (c *AdminController) ListRAGEvalRuns(g *gin.Context) {
	items, err := c.adminData.ListRAGEvalRuns(g.Request.Context(), parseIntDefault(g.Query("limit"), 20))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rag eval runs"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"runs": items})
}

func (c *AdminController) RAGAnalytics(g *gin.Context) {
	analytics, err := c.adminData.RAGAnalytics(g.Request.Context(), parseIntDefault(g.Query("limit"), 500))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load rag analytics"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

func (c *AdminController) UpdateRAGFeedbackStatus(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	var req RAGFeedbackStatusRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.UpdateRAGFeedbackStatus(g.Request.Context(), id, req.Status, req.AdminNote)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update rag feedback"})
		return
	}
	c.logAction(g, "处理 RAG 反馈", "AI 助手", fmt.Sprintf("feedback=%d status=%s", id, item.Status))
	g.JSON(http.StatusOK, gin.H{"feedback": item})
}

func (c *AdminController) ConvertRAGFeedbackToEvalCase(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	item, err := c.adminData.ConvertRAGFeedbackToEvalCase(g.Request.Context(), id)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to convert rag feedback"})
		return
	}
	c.logAction(g, "RAG 反馈转评测用例", "AI 助手", fmt.Sprintf("feedback=%d case=%s", id, item.ID))
	g.JSON(http.StatusCreated, gin.H{"case": item})
}

func (c *AdminController) ListRAGChunks(g *gin.Context) {
	sourceID := parseIntDefault(g.Query("source_id"), 0)
	limit := parseIntDefault(g.Query("limit"), 100)
	chunks, err := c.adminData.ListRAGChunks(g.Request.Context(), g.Query("source_type"), int64(sourceID), limit)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rag chunks"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"chunks": chunks})
}

func (c *AdminController) UploadDocument(g *gin.Context) {
	fh, err := g.FormFile("file")
	if err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	item, err := c.adminData.UploadDocument(g.Request.Context(), fh, g.PostForm("visibility"))
	if err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.logAction(g, "上传 RAG 文档", "RAG 文档管理", item.OriginalName)
	g.JSON(http.StatusCreated, gin.H{"document": item})
}

func (c *AdminController) ListDocuments(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	items, total, err := c.adminData.ListDocuments(g.Request.Context(), page, pageSize)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list documents"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"documents": items, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) PreviewDocument(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	item, err := c.adminData.PreviewDocument(g.Request.Context(), id)
	if err != nil {
		g.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"document": item})
}

func (c *AdminController) UpdateDocumentVisibility(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	var req DocumentVisibilityRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.UpdateDocumentVisibility(g.Request.Context(), id, req.Visibility)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update document visibility"})
		return
	}
	c.logAction(g, "更新 RAG 文档可见性", "RAG 文档管理", fmt.Sprintf("document=%d visibility=%s", id, item.Visibility))
	g.JSON(http.StatusOK, gin.H{"document": item})
}

func (c *AdminController) ListDocumentChunks(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	chunks, err := c.adminData.ListRAGChunks(g.Request.Context(), "uploaded_document", id, parseIntDefault(g.Query("limit"), 100))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list document chunks"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"chunks": chunks})
}

func (c *AdminController) DeleteDocument(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	chunks, err := c.adminData.DeleteDocument(g.Request.Context(), id)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete document"})
		return
	}
	c.logAction(g, "删除 RAG 文档", "RAG 文档管理", fmt.Sprintf("document=%d chunks=%d", id, chunks))
	g.Status(http.StatusNoContent)
}

func (c *AdminController) RebuildDocument(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	item, err := c.adminData.RebuildDocument(g.Request.Context(), id)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rebuild document"})
		return
	}
	c.logAction(g, "重建 RAG 文档索引", "RAG 文档管理", item.OriginalName)
	g.JSON(http.StatusOK, gin.H{"document": item})
}

func (c *AdminController) SystemHealth(g *gin.Context) {
	g.JSON(http.StatusOK, gin.H{"health": c.monitor.Health()})
}

func (c *AdminController) DatabaseCatalog(g *gin.Context) {
	catalog, err := c.adminData.DatabaseCatalog(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load database catalog"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"catalog": catalog})
}

func (c *AdminController) ListDatabaseTables(g *gin.Context) {
	var req DatabaseTablesQuery
	if err := g.ShouldBindQuery(&req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tables, err := c.adminData.ListDatabaseTables(g.Request.Context(), req.Database, req.Table, req.Engine, req.Comment)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list database tables"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"tables": tables})
}

func (c *AdminController) ListDatabaseColumns(g *gin.Context) {
	databaseName := g.Query("database")
	tableName := g.Param("table")
	columns, err := c.adminData.ListDatabaseColumns(g.Request.Context(), databaseName, tableName)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load table columns"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"columns": columns})
}

func (c *AdminController) PublicSiteHome(g *gin.Context) {
	if c.adminData.PublicSiteMaintenanceEnabled(g.Request.Context()) {
		g.JSON(http.StatusServiceUnavailable, gin.H{
			"maintenance": true,
			"message":     "官网维护中，请稍后再试",
		})
		return
	}
	home, err := c.adminData.PublicSiteHome(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load site home"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"home": home, "maintenance": false})
}

func (c *AdminController) PublicSiteResource(g *gin.Context) {
	item, err := c.adminData.GetSiteResourceBySlug(g.Request.Context(), g.Param("slug"))
	if err != nil {
		g.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"resource": item})
}

// PublicSiteSearch 全文搜索已发布文章
func (c *AdminController) PublicSiteSearch(g *gin.Context) {
	query := strings.TrimSpace(g.Query("q"))
	if query == "" {
		g.JSON(http.StatusOK, gin.H{
			"items": []any{}, "total": 0, "page": 1, "page_size": 10, "query": "",
		})
		return
	}
	page, pageSize := parsePaginationQuery(g)
	items, total, err := c.adminData.SearchSiteResources(
		g.Request.Context(), query, g.Query("category"), g.Query("tag"), page, pageSize,
	)
	if err != nil {
		respondError(g, http.StatusInternalServerError, "search failed")
		return
	}
	g.JSON(http.StatusOK, gin.H{
		"items": items, "total": total, "page": page, "page_size": pageSize, "query": query,
	})
}

func (c *AdminController) PublicSiteKnowledge(g *gin.Context) {
	if !c.adminData.PublicRAGEnabled(g.Request.Context()) {
		g.JSON(http.StatusForbidden, gin.H{"error": "public knowledge base is disabled"})
		return
	}
	var req SiteKnowledgeRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	question := knowledgeQuestionWithContext(req)
	answer, err := c.adminData.AskSiteKnowledge(g.Request.Context(), question)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search knowledge base"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"answer": answer})
}

func (c *AdminController) PublicSiteKnowledgeStream(g *gin.Context) {
	if !c.adminData.PublicRAGEnabled(g.Request.Context()) {
		g.JSON(http.StatusForbidden, gin.H{"error": "public knowledge base is disabled"})
		return
	}
	var req SiteKnowledgeRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g.Header("Content-Type", "text/event-stream; charset=utf-8")
	g.Header("Cache-Control", "no-cache")
	g.Header("Connection", "keep-alive")
	g.Status(http.StatusOK)
	flushSSE(g, "meta", gin.H{"streaming": true})
	answer, err := c.adminData.AskSiteKnowledgeStream(g.Request.Context(), knowledgeQuestionWithContext(req), func(token string) error {
		flushSSE(g, "token", gin.H{"content": token})
		return nil
	})
	if err != nil {
		flushSSE(g, "error", gin.H{"error": "failed to stream knowledge answer"})
		return
	}
	flushSSE(g, "meta", gin.H{"query_log_id": answer.QueryLogID})
	flushSSE(g, "sources", answer.Sources)
	flushSSE(g, "suggestions", answer.Suggestions)
	flushSSE(g, "done", gin.H{"answer": answer})
}

func (c *AdminController) PublicSiteFeedback(g *gin.Context) {
	var req SiteFeedbackRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.SaveRAGFeedback(g.Request.Context(), req.QueryLogID, req.Question, req.Rating, req.Comment, g.ClientIP(), g.Request.UserAgent())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save feedback"})
		return
	}
	g.JSON(http.StatusCreated, gin.H{"feedback": item})
}

func (c *AdminController) PublicSiteCodeExplain(g *gin.Context) {
	var req SiteCodeExplainRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	answer, err := c.adminData.ExplainCode(g.Request.Context(), req.Code, req.Language, req.Question)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to explain code"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"answer": answer})
}

func (c *AdminController) PublicSiteSearchSummarize(g *gin.Context) {
	var req SiteSearchSummarizeRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	answer, err := c.adminData.SummarizeSearch(g.Request.Context(), req.Query)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to summarize search"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"answer": answer})
}

func knowledgeQuestionWithContext(req SiteKnowledgeRequest) string {
	question := strings.TrimSpace(req.Question)
	if len(req.Context) == 0 {
		return question
	}
	var parts []string
	for _, item := range req.Context {
		if len(parts) >= 6 {
			break
		}
		role := strings.TrimSpace(item.Role)
		content := limitControllerRunes(strings.TrimSpace(item.Content), 300)
		if content == "" {
			continue
		}
		if role == "" {
			role = "message"
		}
		parts = append(parts, role+": "+content)
	}
	if len(parts) == 0 {
		return question
	}
	return question + "\n\n最近对话上下文：\n" + strings.Join(parts, "\n")
	return question + "\n\n最近对话上下文：\n" + strings.Join(parts, "\n")
}

func limitControllerRunes(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if max <= 0 || len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "..."
}

func flushSSE(g *gin.Context, event string, payload any) {
	raw, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(g.Writer, "event: %s\ndata: %s\n\n", event, raw)
	if flusher, ok := g.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func streamTextChunks(value string, size int) []string {
	runes := []rune(value)
	if size <= 0 {
		size = 16
	}
	if len(runes) == 0 {
		return []string{""}
	}
	out := make([]string, 0, len(runes)/size+1)
	for start := 0; start < len(runes); start += size {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[start:end]))
	}
	return out
}

func (c *AdminController) PublicSiteMessage(g *gin.Context) {
	var req SiteMessageRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.CreateSiteMessage(g.Request.Context(), siteMessageInput(req, g))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create message"})
		return
	}
	g.JSON(http.StatusCreated, gin.H{"message": item})
}

func (c *AdminController) PublicSiteVisit(g *gin.Context) {
	var req SiteVisitRequest
	if err := bindRequest(g, &req); err != nil {
		g.Status(http.StatusNoContent)
		return
	}
	_ = c.adminData.RecordSiteVisit(g.Request.Context(), services.SiteVisitInput{
		Path:      strings.TrimSpace(req.Path),
		Referrer:  strings.TrimSpace(req.Referrer),
		Device:    normalizeDevice(req.Device, g.Request.UserAgent()),
		IPAddress: g.ClientIP(),
		UserAgent: g.Request.UserAgent(),
	})
	g.Status(http.StatusNoContent)
}

func (c *AdminController) ListSiteAnnouncements(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	items, total, err := c.adminData.ListSiteAnnouncements(g.Request.Context(), page, pageSize, g.Query("status"))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list announcements"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"announcements": items, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) CreateSiteAnnouncement(g *gin.Context) {
	var req SiteAnnouncementRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.CreateSiteAnnouncement(g.Request.Context(), siteAnnouncementInput(req))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create announcement"})
		return
	}
	c.logAction(g, "创建官网公告", "官网管理", item.Title)
	g.JSON(http.StatusCreated, gin.H{"announcement": item})
}

func (c *AdminController) UpdateSiteAnnouncement(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	var req SiteAnnouncementRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.UpdateSiteAnnouncement(g.Request.Context(), id, siteAnnouncementInput(req))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update announcement"})
		return
	}
	c.logAction(g, "编辑官网公告", "官网管理", item.Title)
	g.JSON(http.StatusOK, gin.H{"announcement": item})
}

func (c *AdminController) DeleteSiteAnnouncement(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	if err := c.adminData.DeleteSiteAnnouncement(g.Request.Context(), id); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete announcement"})
		return
	}
	c.logAction(g, "删除官网公告", "官网管理", fmt.Sprintf("公告ID：%d", id))
	g.Status(http.StatusNoContent)
}

func (c *AdminController) ListSiteBanners(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	items, total, err := c.adminData.ListSiteBanners(g.Request.Context(), page, pageSize, g.Query("status"))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list banners"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"banners": items, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) CreateSiteBanner(g *gin.Context) {
	var req SiteBannerRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.CreateSiteBanner(g.Request.Context(), siteBannerInput(req))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create banner"})
		return
	}
	c.logAction(g, "创建官网轮播", "官网管理", item.Title)
	g.JSON(http.StatusCreated, gin.H{"banner": item})
}

func (c *AdminController) UpdateSiteBanner(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	var req SiteBannerRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.UpdateSiteBanner(g.Request.Context(), id, siteBannerInput(req))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update banner"})
		return
	}
	c.logAction(g, "编辑官网轮播", "官网管理", item.Title)
	g.JSON(http.StatusOK, gin.H{"banner": item})
}

func (c *AdminController) DeleteSiteBanner(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	if err := c.adminData.DeleteSiteBanner(g.Request.Context(), id); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete banner"})
		return
	}
	c.logAction(g, "删除官网轮播", "官网管理", fmt.Sprintf("轮播ID：%d", id))
	g.Status(http.StatusNoContent)
}

func (c *AdminController) ListSiteResources(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	items, total, err := c.adminData.ListSiteResources(g.Request.Context(), page, pageSize, g.Query("status"))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list resources"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"resources": items, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) SaveSiteResource(g *gin.Context) {
	id := int64(0)
	if raw := g.Param("id"); raw != "" {
		parsed, ok := parseIDParam(g, "id")
		if !ok {
			return
		}
		id = parsed
	}
	var req SiteResourceRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.SaveSiteResource(g.Request.Context(), id, siteResourceInput(req))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save resource"})
		return
	}
	c.logAction(g, "保存官网资源", "官网管理", item.Title)
	g.JSON(http.StatusOK, gin.H{"resource": item})
}

func (c *AdminController) DeleteSiteResource(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	if err := c.adminData.DeleteSiteResource(g.Request.Context(), id); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete resource"})
		return
	}
	c.logAction(g, "删除官网资源", "官网管理", fmt.Sprintf("资源ID：%d", id))
	g.Status(http.StatusNoContent)
}

func (c *AdminController) ListSiteTechStacks(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	items, total, err := c.adminData.ListSiteTechStacks(g.Request.Context(), page, pageSize, g.Query("status"))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tech stacks"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"tech_stacks": items, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) SaveSiteTechStack(g *gin.Context) {
	id := int64(0)
	if raw := g.Param("id"); raw != "" {
		parsed, ok := parseIDParam(g, "id")
		if !ok {
			return
		}
		id = parsed
	}
	var req SiteTechStackRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.SaveSiteTechStack(g.Request.Context(), id, siteTechStackInput(req))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save tech stack"})
		return
	}
	c.logAction(g, "保存官网技术栈", "官网管理", item.Name)
	g.JSON(http.StatusOK, gin.H{"tech_stack": item})
}

func (c *AdminController) DeleteSiteTechStack(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	if err := c.adminData.DeleteSiteTechStack(g.Request.Context(), id); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete tech stack"})
		return
	}
	c.logAction(g, "删除官网技术栈", "官网管理", fmt.Sprintf("技术栈ID：%d", id))
	g.Status(http.StatusNoContent)
}

func (c *AdminController) ListSiteProjects(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	items, total, err := c.adminData.ListSiteProjects(g.Request.Context(), page, pageSize, g.Query("status"))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list projects"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"projects": items, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) SaveSiteProject(g *gin.Context) {
	id := int64(0)
	if raw := g.Param("id"); raw != "" {
		parsed, ok := parseIDParam(g, "id")
		if !ok {
			return
		}
		id = parsed
	}
	var req SiteProjectRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.SaveSiteProject(g.Request.Context(), id, siteProjectInput(req))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save project"})
		return
	}
	c.logAction(g, "保存官网项目", "官网管理", item.Name)
	g.JSON(http.StatusOK, gin.H{"project": item})
}

func (c *AdminController) DeleteSiteProject(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	if err := c.adminData.DeleteSiteProject(g.Request.Context(), id); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete project"})
		return
	}
	c.logAction(g, "删除官网项目", "官网管理", fmt.Sprintf("项目ID：%d", id))
	g.Status(http.StatusNoContent)
}

func (c *AdminController) ListSiteTimelineEvents(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	items, total, err := c.adminData.ListSiteTimelineEvents(g.Request.Context(), page, pageSize, g.Query("status"))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list timeline events"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"timeline": items, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) SaveSiteTimelineEvent(g *gin.Context) {
	id := int64(0)
	if raw := g.Param("id"); raw != "" {
		parsed, ok := parseIDParam(g, "id")
		if !ok {
			return
		}
		id = parsed
	}
	var req SiteTimelineEventRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.SaveSiteTimelineEvent(g.Request.Context(), id, siteTimelineEventInput(req))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save timeline event"})
		return
	}
	c.logAction(g, "保存官网时间轴", "官网管理", item.Title)
	g.JSON(http.StatusOK, gin.H{"timeline_event": item})
}

func (c *AdminController) DeleteSiteTimelineEvent(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	if err := c.adminData.DeleteSiteTimelineEvent(g.Request.Context(), id); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete timeline event"})
		return
	}
	c.logAction(g, "删除官网时间轴", "官网管理", fmt.Sprintf("时间轴ID：%d", id))
	g.Status(http.StatusNoContent)
}

func (c *AdminController) ListSiteMessages(g *gin.Context) {
	page, pageSize := parsePaginationQuery(g)
	items, total, err := c.adminData.ListSiteMessages(g.Request.Context(), page, pageSize, g.Query("status"))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list messages"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"messages": items, "total": total, "page": page, "page_size": pageSize})
}

func (c *AdminController) SaveSiteMessage(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	var req SiteMessageRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := c.adminData.SaveSiteMessage(g.Request.Context(), id, siteMessageInput(req, g))
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save message"})
		return
	}
	c.logAction(g, "处理官网留言", "官网管理", item.Content)
	g.JSON(http.StatusOK, gin.H{"message": item})
}

func (c *AdminController) DeleteSiteMessage(g *gin.Context) {
	id, ok := parseIDParam(g, "id")
	if !ok {
		return
	}
	if err := c.adminData.DeleteSiteMessage(g.Request.Context(), id); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete message"})
		return
	}
	c.logAction(g, "删除官网留言", "官网管理", fmt.Sprintf("留言ID：%d", id))
	g.Status(http.StatusNoContent)
}

func (c *AdminController) SiteAnalytics(g *gin.Context) {
	analytics, err := c.adminData.SiteAnalytics(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load site analytics"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

func (c *AdminController) UploadSiteAsset(g *gin.Context) {
	fh, err := g.FormFile("file")
	if err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	if fh.Size <= 0 || fh.Size > 10<<20 {
		g.JSON(http.StatusBadRequest, gin.H{"error": "file too large"})
		return
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true, ".svg": true}
	if !allowed[ext] {
		g.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type"})
		return
	}
	name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dateDir := time.Now().Format("20060102")
	dir := filepath.Join("uploads", "site", dateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload dir"})
		return
	}
	path := filepath.Join(dir, name)
	if err := g.SaveUploadedFile(fh, path); err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}
	url := "/" + filepath.ToSlash(path)
	c.logAction(g, "上传官网资源", "官网管理", url)
	g.JSON(http.StatusOK, gin.H{"url": url})
}

func parsePaginationQuery(g *gin.Context) (int, int) {
	page := 1
	pageSize := 10

	if value := g.Query("page"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if value := g.Query("page_size"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func parseIntDefault(value string, fallback int) int {
	if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
		return parsed
	}
	return fallback
}

func normalizeDevice(value, ua string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "mobile" || value == "tablet" || value == "desktop" {
		return value
	}
	lowerUA := strings.ToLower(ua)
	if strings.Contains(lowerUA, "ipad") || strings.Contains(lowerUA, "tablet") {
		return "tablet"
	}
	if strings.Contains(lowerUA, "mobile") || strings.Contains(lowerUA, "android") || strings.Contains(lowerUA, "iphone") {
		return "mobile"
	}
	return "desktop"
}

func adminCurrentUserID(g *gin.Context) int64 {
	value, _ := g.Get("user_id")
	if userID, ok := value.(int64); ok {
		return userID
	}
	return 0
}

func (c *AdminController) logAction(g *gin.Context, action, resource, detail string) {
	if c.adminData == nil {
		return
	}
	userID := adminCurrentUserID(g)
	username := ""
	if userID > 0 {
		if user, err := c.authService.GetUserByID(g.Request.Context(), userID); err == nil {
			username = user.Username
		}
	}
	c.adminData.RecordOperationLog(g.Request.Context(), services.OperationLogInput{
		UserID:    userID,
		Username:  username,
		Action:    action,
		Resource:  resource,
		Detail:    detail,
		IP:        g.ClientIP(),
		UserAgent: g.Request.UserAgent(),
	})
}
