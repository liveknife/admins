package services

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"go-demo/database"
	"go-demo/models"
)

type AdminDataService struct {
	db *sql.DB
}

type OperationLogInput struct {
	UserID    int64
	Username  string
	Action    string
	Resource  string
	Detail    string
	IP        string
	UserAgent string
}

func NewAdminDataService(db *sql.DB) *AdminDataService {
	return &AdminDataService{db: db}
}

func (s *AdminDataService) Dashboard(ctx context.Context, userID int64) (*models.DashboardSummary, error) {
	out := &models.DashboardSummary{}
	database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM users`).Scan(&out.UserCount)
	database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&out.ActiveUserCount)
	database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM roles`).Scan(&out.RoleCount)
	database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM permissions`).Scan(&out.PermissionCount)
	database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM chat_messages`).Scan(&out.MessageCount)
	database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM notifications WHERE (user_id=$1 OR user_id IS NULL) AND is_read=FALSE`, userID).Scan(&out.UnreadNotification)

	logs, _, err := s.ListOperationLogs(ctx, 1, 5)
	if err != nil {
		return nil, err
	}
	notices, _, err := s.ListNotifications(ctx, userID, 1, 5, "")
	if err != nil {
		return nil, err
	}
	trend, err := s.dashboardTrend(ctx)
	if err != nil {
		return nil, err
	}
	out.RecentLogs = logs
	out.RecentNotifications = notices
	out.MetricTrend = trend
	return out, nil
}

func (s *AdminDataService) PermissionTree(ctx context.Context) ([]models.PermissionTreeNode, error) {
	perms, err := s.listPermissionCodes(ctx)
	if err != nil {
		return nil, err
	}
	grouped := map[string][]models.PermissionTreeNode{
		"menu":   {},
		"button": {},
		"system": {},
	}
	for _, p := range perms {
		group := classifyPermission(p.Code)
		grouped[group] = append(grouped[group], models.PermissionTreeNode{
			ID:          p.Code,
			Label:       permissionLabel(p.Code),
			Code:        p.Code,
			Type:        group,
			Description: p.Description,
		})
	}
	for key := range grouped {
		sort.Slice(grouped[key], func(i, j int) bool { return grouped[key][i].Code < grouped[key][j].Code })
	}
	return []models.PermissionTreeNode{
		{ID: "menu", Label: "菜单权限", Type: "group", Children: grouped["menu"]},
		{ID: "button", Label: "按钮权限", Type: "group", Children: grouped["button"]},
		{ID: "system", Label: "系统能力", Type: "group", Children: grouped["system"]},
	}, nil
}

func (s *AdminDataService) RolePreview(ctx context.Context, roleID int64) (*models.RolePermissionPreview, error) {
	var role models.Role
	if err := database.QueryRowCtx(ctx, s.db, `SELECT id,name,description,created_at FROM roles WHERE id=$1`, roleID).
		Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt); err != nil {
		return nil, err
	}
	perms, err := s.permissionsForRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	role.Permissions = perms
	menuNodes, buttonNodes := previewPermissionNodes(perms)
	return &models.RolePermissionPreview{
		Role:        role,
		Menus:       menuNodes,
		Buttons:     buttonNodes,
		Permissions: perms,
	}, nil
}

func (s *AdminDataService) AskAssistant(ctx context.Context, question string) (*models.AIAssistantResult, error) {
	question = strings.TrimSpace(question)
	lower := strings.ToLower(question)
	if question == "" {
		question = "总结最近系统情况"
	}
	result := &models.AIAssistantResult{
		Question: question,
		Metrics:  make(map[string]int64),
	}
	if strings.Contains(question, "登录失败") || strings.Contains(lower, "login failed") {
		rows, err := s.loginFailureRows(ctx)
		if err != nil {
			return nil, err
		}
		result.Rows = rows
		result.Answer = "已按账号统计最近登录失败次数。当前系统没有独立登录失败表，因此基于操作日志中的登录失败关键词进行分析。"
		result.Insights = append(result.Insights, fmt.Sprintf("共找到 %d 个可疑账号。", len(rows)))
		return result, nil
	}
	if strings.Contains(question, "报告") || strings.Contains(question, "用户操作") || strings.Contains(lower, "report") {
		return s.userOperationReport(ctx, question)
	}
	if strings.Contains(question, "异常") || strings.Contains(lower, "error") || strings.Contains(lower, "exception") {
		return s.exceptionAnalysis(ctx, question)
	}
	return s.logSummary(ctx, question)
}

func (s *AdminDataService) RecordOperationLog(ctx context.Context, input OperationLogInput) {
	input.Action = strings.TrimSpace(input.Action)
	if input.Action == "" {
		return
	}
	_, _ = database.ExecCtx(ctx, s.db,
		`INSERT INTO operation_logs(user_id,username,action,resource,detail,ip,user_agent) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		input.UserID,
		strings.TrimSpace(input.Username),
		input.Action,
		strings.TrimSpace(input.Resource),
		strings.TrimSpace(input.Detail),
		strings.TrimSpace(input.IP),
		strings.TrimSpace(input.UserAgent),
	)
}

func (s *AdminDataService) ListOperationLogs(ctx context.Context, page, pageSize int) ([]models.OperationLog, int64, error) {
	var total int64
	if err := database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM operation_logs`).Scan(&total); err != nil {
		return nil, 0, err
	}
	limit, offset := normalizePagination(page, pageSize)
	rows, err := database.QueryCtx(ctx, s.db,
		`SELECT id,user_id,username,action,resource,detail,ip,user_agent,created_at FROM operation_logs ORDER BY id DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := make([]models.OperationLog, 0)
	for rows.Next() {
		var item models.OperationLog
		if err := rows.Scan(&item.ID, &item.UserID, &item.Username, &item.Action, &item.Resource, &item.Detail, &item.IP, &item.UserAgent, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, item)
	}
	return logs, total, rows.Err()
}

func (s *AdminDataService) ListNotifications(ctx context.Context, userID int64, page, pageSize int, readStatus string) ([]models.Notification, int64, error) {
	where := `WHERE (user_id=$1 OR user_id IS NULL)`
	args := []any{userID}
	if readStatus == "read" {
		where += ` AND is_read=TRUE`
	} else if readStatus == "unread" {
		where += ` AND is_read=FALSE`
	}

	var total int64
	if err := database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM notifications `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limit, offset := normalizePagination(page, pageSize)
	args = append(args, limit, offset)
	limitPlaceholder := database.CurrentDialect.Placeholder(len(args) - 1)
	offsetPlaceholder := database.CurrentDialect.Placeholder(len(args))
	rows, err := database.QueryCtx(ctx, s.db,
		`SELECT id,user_id,title,content,type,is_read,created_at,read_at FROM notifications `+where+` ORDER BY id DESC LIMIT `+limitPlaceholder+` OFFSET `+offsetPlaceholder,
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	notices := make([]models.Notification, 0)
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return nil, 0, err
		}
		notices = append(notices, item)
	}
	return notices, total, rows.Err()
}

func (s *AdminDataService) UnreadNotificationCount(ctx context.Context, userID int64) (int64, error) {
	var total int64
	err := database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM notifications WHERE (user_id=$1 OR user_id IS NULL) AND is_read=FALSE`, userID).Scan(&total)
	return total, err
}

func (s *AdminDataService) MarkNotificationRead(ctx context.Context, userID, noticeID int64) error {
	_, err := database.ExecCtx(ctx, s.db,
		`UPDATE notifications SET is_read=TRUE,read_at=`+database.Now()+` WHERE id=$1 AND (user_id=$2 OR user_id IS NULL)`,
		noticeID, userID,
	)
	return err
}

func (s *AdminDataService) MarkAllNotificationsRead(ctx context.Context, userID int64) error {
	_, err := database.ExecCtx(ctx, s.db,
		`UPDATE notifications SET is_read=TRUE,read_at=`+database.Now()+` WHERE (user_id=$1 OR user_id IS NULL) AND is_read=FALSE`,
		userID,
	)
	return err
}

func (s *AdminDataService) dashboardTrend(ctx context.Context) ([]models.DashboardMetric, error) {
	out := make([]models.DashboardMetric, 0, 7)
	start := time.Now().AddDate(0, 0, -6)
	for i := 0; i < 7; i++ {
		dayStart := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location()).AddDate(0, 0, i)
		dayEnd := dayStart.AddDate(0, 0, 1)
		item := models.DashboardMetric{Date: dayStart.Format("2006-01-02")}
		if err := database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM users WHERE created_at >= $1 AND created_at < $2`, dayStart, dayEnd).Scan(&item.Users); err != nil {
			return nil, err
		}
		if err := database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM chat_messages WHERE created_at >= $1 AND created_at < $2`, dayStart, dayEnd).Scan(&item.Messages); err != nil {
			return nil, err
		}
		if err := database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM operation_logs WHERE created_at >= $1 AND created_at < $2`, dayStart, dayEnd).Scan(&item.Logs); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func scanNotification(rows *sql.Rows) (models.Notification, error) {
	var item models.Notification
	var userID sql.NullInt64
	var readAt sql.NullTime
	err := rows.Scan(&item.ID, &userID, &item.Title, &item.Content, &item.Type, &item.IsRead, &item.CreatedAt, &readAt)
	if userID.Valid {
		item.UserID = &userID.Int64
	}
	if readAt.Valid {
		item.ReadAt = &readAt.Time
	}
	return item, err
}

func (s *AdminDataService) listPermissionCodes(ctx context.Context) ([]models.Permission, error) {
	rows, err := database.QueryCtx(ctx, s.db, `SELECT id,code,description,created_at FROM permissions ORDER BY code ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Permission
	for rows.Next() {
		var p models.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *AdminDataService) permissionsForRole(ctx context.Context, roleID int64) ([]string, error) {
	rows, err := database.QueryCtx(ctx, s.db, `SELECT p.code FROM role_permissions rp JOIN permissions p ON p.id=rp.permission_id WHERE rp.role_id=$1 ORDER BY p.code ASC`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		out = append(out, code)
	}
	return out, rows.Err()
}

func classifyPermission(code string) string {
	if strings.HasSuffix(code, ":write") || strings.Contains(code, ":password:") || strings.Contains(code, "notifications:write") {
		return "button"
	}
	if strings.Contains(code, ":read") || code == "messages:chat" || code == "admin:access" || code == "dashboard:read" {
		return "menu"
	}
	return "system"
}

func permissionLabel(code string) string {
	labels := map[string]string{
		"admin:access":        "后台访问",
		"dashboard:read":      "仪表盘首页",
		"users:read":          "用户管理",
		"users:write":         "用户写入操作",
		"users:password:read": "查看用户密码",
		"roles:read":          "角色权限",
		"roles:write":         "角色写入操作",
		"permissions:read":    "权限列表",
		"messages:chat":       "聊天消息",
		"logs:read":           "操作日志",
		"notifications:read":  "通知中心",
		"notifications:write": "通知写入操作",
		"ai:assistant":        "AI 助手",
		"health:read":         "系统健康监控",
	}
	if label, ok := labels[code]; ok {
		return label
	}
	return code
}

func previewPermissionNodes(codes []string) ([]models.PermissionTreeNode, []models.PermissionTreeNode) {
	menuSet := make(map[string]models.PermissionTreeNode)
	buttons := make([]models.PermissionTreeNode, 0)
	for _, code := range codes {
		node := models.PermissionTreeNode{
			ID:    code,
			Label: permissionLabel(code),
			Code:  code,
			Type:  classifyPermission(code),
		}
		if node.Type == "button" {
			buttons = append(buttons, node)
			continue
		}
		if menu := menuForPermission(code); menu.ID != "" {
			menuSet[menu.ID] = menu
		}
	}
	menus := make([]models.PermissionTreeNode, 0, len(menuSet))
	for _, node := range menuSet {
		menus = append(menus, node)
	}
	sort.Slice(menus, func(i, j int) bool { return menus[i].ID < menus[j].ID })
	sort.Slice(buttons, func(i, j int) bool { return buttons[i].Code < buttons[j].Code })
	return menus, buttons
}

func menuForPermission(code string) models.PermissionTreeNode {
	switch code {
	case "dashboard:read":
		return models.PermissionTreeNode{ID: "/welcome", Label: "仪表盘首页", Type: "menu"}
	case "users:read":
		return models.PermissionTreeNode{ID: "/go-admin/users", Label: "用户管理", Type: "menu"}
	case "roles:read", "permissions:read":
		return models.PermissionTreeNode{ID: "/go-admin/roles", Label: "角色权限", Type: "menu"}
	case "logs:read":
		return models.PermissionTreeNode{ID: "/go-admin/operation-logs", Label: "操作日志", Type: "menu"}
	case "notifications:read":
		return models.PermissionTreeNode{ID: "/go-admin/notifications", Label: "通知中心", Type: "menu"}
	case "messages:chat":
		return models.PermissionTreeNode{ID: "/message/chat", Label: "聊天", Type: "menu"}
	case "ai:assistant":
		return models.PermissionTreeNode{ID: "/system-tools/ai-assistant", Label: "AI 助手", Type: "menu"}
	case "health:read":
		return models.PermissionTreeNode{ID: "/system-tools/health", Label: "系统健康监控", Type: "menu"}
	default:
		return models.PermissionTreeNode{}
	}
}

func (s *AdminDataService) logSummary(ctx context.Context, question string) (*models.AIAssistantResult, error) {
	logs, total, err := s.ListOperationLogs(ctx, 1, 20)
	if err != nil {
		return nil, err
	}
	byAction := make(map[string]int64)
	byUser := make(map[string]int64)
	for _, item := range logs {
		byAction[item.Action]++
		user := item.Username
		if user == "" {
			user = "系统"
		}
		byUser[user]++
	}
	result := &models.AIAssistantResult{
		Question: question,
		Answer:   fmt.Sprintf("最近记录中共发现 %d 条操作日志，已抽取最近 20 条做摘要。", total),
		Insights: []string{
			"高频操作：" + joinTopCounts(byAction, 3),
			"活跃管理员：" + joinTopCounts(byUser, 3),
		},
		Metrics: map[string]int64{"total_logs": total},
	}
	for _, item := range logs {
		result.Rows = append(result.Rows, map[string]any{
			"time":     item.CreatedAt,
			"user":     item.Username,
			"action":   item.Action,
			"resource": item.Resource,
			"detail":   item.Detail,
		})
	}
	return result, nil
}

func (s *AdminDataService) exceptionAnalysis(ctx context.Context, question string) (*models.AIAssistantResult, error) {
	rows, err := database.QueryCtx(ctx, s.db, `SELECT action,resource,detail,username,created_at FROM operation_logs WHERE lower(action) LIKE $1 OR lower(detail) LIKE $2 ORDER BY id DESC LIMIT 20`, "%fail%", "%error%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := &models.AIAssistantResult{
		Question: question,
		Answer:   "已扫描操作日志中的失败、错误、异常关键词。",
		Metrics:  map[string]int64{},
	}
	for rows.Next() {
		var action, resource, detail, username string
		var createdAt time.Time
		if err := rows.Scan(&action, &resource, &detail, &username, &createdAt); err != nil {
			return nil, err
		}
		result.Rows = append(result.Rows, map[string]any{"time": createdAt, "user": username, "action": action, "resource": resource, "detail": detail})
	}
	result.Metrics["exception_logs"] = int64(len(result.Rows))
	if len(result.Rows) == 0 {
		result.Insights = append(result.Insights, "未发现明显异常关键词。")
	} else {
		result.Insights = append(result.Insights, "建议优先检查最近失败操作对应的用户、接口和权限配置。")
	}
	return result, nil
}

func (s *AdminDataService) userOperationReport(ctx context.Context, question string) (*models.AIAssistantResult, error) {
	rows, err := database.QueryCtx(ctx, s.db, `SELECT username,COUNT(*) AS total,MAX(created_at) AS last_time FROM operation_logs GROUP BY username ORDER BY total DESC LIMIT 10`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := &models.AIAssistantResult{
		Question: question,
		Answer:   "已生成管理员操作报告，按操作次数排序。",
		Metrics:  map[string]int64{},
	}
	var total int64
	for rows.Next() {
		var username string
		var count int64
		var lastTime time.Time
		if err := rows.Scan(&username, &count, &lastTime); err != nil {
			return nil, err
		}
		if username == "" {
			username = "系统"
		}
		total += count
		result.Rows = append(result.Rows, map[string]any{"user": username, "count": count, "last_time": lastTime})
	}
	result.Metrics["operations"] = total
	result.Insights = append(result.Insights, "报告覆盖最近所有操作日志。")
	return result, nil
}

func (s *AdminDataService) loginFailureRows(ctx context.Context) ([]map[string]any, error) {
	rows, err := database.QueryCtx(ctx, s.db, `SELECT username,COUNT(*) AS total,MAX(created_at) AS last_time FROM operation_logs WHERE lower(action) LIKE $1 OR lower(detail) LIKE $2 GROUP BY username ORDER BY total DESC LIMIT 10`, "%login%", "%登录失败%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		var username string
		var count int64
		var lastTime time.Time
		if err := rows.Scan(&username, &count, &lastTime); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"user": username, "failures": count, "last_time": lastTime})
	}
	return out, rows.Err()
}

func joinTopCounts(values map[string]int64, limit int) string {
	type pair struct {
		key   string
		value int64
	}
	items := make([]pair, 0, len(values))
	for key, value := range values {
		if key == "" {
			key = "-"
		}
		items = append(items, pair{key: key, value: value})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].value > items[j].value })
	if len(items) == 0 {
		return "暂无数据"
	}
	if len(items) > limit {
		items = items[:limit]
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s %d 次", item.key, item.value))
	}
	return strings.Join(parts, "，")
}
