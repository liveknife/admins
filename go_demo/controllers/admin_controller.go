package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

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

type roleRequest struct {
	Name        string   `json:"name" form:"name" binding:"required,min=2,max=50"`
	Description string   `json:"description" form:"description"`
	Permissions []string `json:"permissions" form:"permissions"`
}

type resetUserPasswordRequest struct {
	PasswordEncrypted string `json:"password_encrypted" form:"password_encrypted" binding:"required"`
}

type setUserRolesRequest struct {
	Roles []string `json:"roles" form:"roles" binding:"required"`
}

type createUserRequest struct {
	Username          string   `json:"username" form:"username" binding:"required,min=2,max=50"`
	Email             string   `json:"email" form:"email" binding:"required,email"`
	Phone             string   `json:"phone" form:"phone"`
	PasswordEncrypted string   `json:"password_encrypted" form:"password_encrypted" binding:"required"`
	Roles             []string `json:"roles" form:"roles"`
}

type updateUserRequest struct {
	Username string `json:"username" form:"username" binding:"required,min=2,max=50"`
	Email    string `json:"email" form:"email" binding:"required,email"`
	Phone    string `json:"phone" form:"phone"`
}

type askAssistantRequest struct {
	Question string `json:"question" form:"question"`
}

type databaseTablesQuery struct {
	Database string `form:"database"`
	Table    string `form:"table"`
	Engine   string `form:"engine"`
	Comment  string `form:"comment"`
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
	var req setUserRolesRequest
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
	var req resetUserPasswordRequest
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
	var req createUserRequest
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
	var req updateUserRequest
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
	var req roleRequest
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
	var req roleRequest
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
	var req askAssistantRequest
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
	var req databaseTablesQuery
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
