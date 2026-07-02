package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

type SiteKnowledgeRequest struct {
	Question string `json:"question" form:"question"`
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
	Status      string     `json:"status" form:"status"`
	IsFeatured  bool       `json:"is_featured" form:"is_featured"`
	SortOrder   int        `json:"sort_order" form:"sort_order"`
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
		Status:      strings.TrimSpace(req.Status),
		IsFeatured:  req.IsFeatured,
		SortOrder:   req.SortOrder,
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
	home, err := c.adminData.PublicSiteHome(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load site home"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"home": home})
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
	var req SiteKnowledgeRequest
	if err := bindRequest(g, &req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	answer, err := c.adminData.AskSiteKnowledge(g.Request.Context(), req.Question)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search knowledge base"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"answer": answer})
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
