package services

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"go-demo/database"
	"go-demo/models"
)

type SystemSettingInput struct {
	SettingKey   string
	SettingValue string
	GroupName    string
	ValueType    string
	Description  string
	IsSecret     bool
}

type AIModelCallLogInput struct {
	Provider         string
	APIFormat        string
	Model            string
	Operation        string
	Status           string
	LatencyMS        int64
	PromptTokens     int
	CompletionTokens int
	RequestChars     int
	ResponseChars    int
	ErrorMessage     string
}

func (s *AdminDataService) ListSystemSettings(ctx context.Context) ([]models.SystemSetting, error) {
	if err := s.ensureDefaultSystemSettings(ctx); err != nil {
		return nil, err
	}
	rows, err := database.QueryCtx(ctx, s.db, `SELECT id,setting_key,setting_value,group_name,value_type,description,is_secret,updated_at FROM system_settings ORDER BY group_name ASC,setting_key ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.SystemSetting, 0)
	for rows.Next() {
		item, err := scanSystemSetting(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *AdminDataService) SaveSystemSettings(ctx context.Context, inputs []SystemSettingInput) ([]models.SystemSetting, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for _, input := range inputs {
		key := strings.TrimSpace(input.SettingKey)
		if key == "" {
			continue
		}
		groupName := strings.TrimSpace(input.GroupName)
		if groupName == "" {
			groupName = "general"
		}
		valueType := strings.TrimSpace(input.ValueType)
		if valueType == "" {
			valueType = "text"
		}
		var id int64
		err := database.QueryRowTxCtx(ctx, tx, `SELECT id FROM system_settings WHERE setting_key=$1`, key).Scan(&id)
		if err == sql.ErrNoRows {
			_, err = database.ExecTxCtx(ctx, tx, `INSERT INTO system_settings(setting_key,setting_value,group_name,value_type,description,is_secret) VALUES($1,$2,$3,$4,$5,$6)`,
				key, input.SettingValue, groupName, valueType, strings.TrimSpace(input.Description), input.IsSecret)
		} else if err == nil {
			_, err = database.ExecTxCtx(ctx, tx, `UPDATE system_settings SET setting_value=$1,group_name=$2,value_type=$3,description=$4,is_secret=$5,updated_at=`+database.Now()+` WHERE id=$6`,
				input.SettingValue, groupName, valueType, strings.TrimSpace(input.Description), input.IsSecret, id)
		}
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.ListSystemSettings(ctx)
}

func (s *AdminDataService) ensureDefaultSystemSettings(ctx context.Context) error {
	defaults := []SystemSettingInput{
		{SettingKey: "site.name", SettingValue: "个人官网", GroupName: "site", ValueType: "text", Description: "官网名称"},
		{SettingKey: "site.description", SettingValue: "作品、文章与知识库展示", GroupName: "site", ValueType: "textarea", Description: "官网描述"},
		{SettingKey: "site.maintenance", SettingValue: "false", GroupName: "site", ValueType: "boolean", Description: "官网维护模式"},
		{SettingKey: "rag.public_enabled", SettingValue: "true", GroupName: "rag", ValueType: "boolean", Description: "允许官网知识库公开问答"},
		{SettingKey: "rag.default_visibility", SettingValue: "internal", GroupName: "rag", ValueType: "select", Description: "上传文档默认可见性"},
		{SettingKey: "ai.log_retention_days", SettingValue: "30", GroupName: "ai", ValueType: "number", Description: "AI 调用日志保留天数"},
		{SettingKey: "ops.alert_email", SettingValue: "", GroupName: "ops", ValueType: "text", Description: "运营告警接收邮箱"},
	}
	for _, item := range defaults {
		var count int64
		if err := database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM system_settings WHERE setting_key=$1`, item.SettingKey).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if _, err := database.ExecCtx(ctx, s.db, `INSERT INTO system_settings(setting_key,setting_value,group_name,value_type,description,is_secret) VALUES($1,$2,$3,$4,$5,$6)`,
			item.SettingKey, item.SettingValue, item.GroupName, item.ValueType, item.Description, item.IsSecret); err != nil {
			return err
		}
	}
	return nil
}

func (s *AdminDataService) RecordAIModelCallLog(ctx context.Context, input AIModelCallLogInput) error {
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "success"
	}
	_, err := database.ExecCtx(ctx, s.db, `INSERT INTO ai_model_call_logs(provider,api_format,model,operation,status,latency_ms,prompt_tokens,completion_tokens,request_chars,response_chars,error_message) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		limitRunes(strings.TrimSpace(input.Provider), 60),
		limitRunes(strings.TrimSpace(input.APIFormat), 40),
		limitRunes(strings.TrimSpace(input.Model), 160),
		limitRunes(strings.TrimSpace(input.Operation), 30),
		status,
		input.LatencyMS,
		input.PromptTokens,
		input.CompletionTokens,
		input.RequestChars,
		input.ResponseChars,
		limitRunes(input.ErrorMessage, 2000),
	)
	return err
}

func (s *AdminDataService) ListAIModelCallLogs(ctx context.Context, page, pageSize int, provider, operation, status string) ([]models.AIModelCallLog, int64, error) {
	where := "WHERE 1=1"
	args := []any{}
	if strings.TrimSpace(provider) != "" {
		args = append(args, provider)
		where += " AND provider=$" + intPlaceholder(len(args))
	}
	if strings.TrimSpace(operation) != "" {
		args = append(args, operation)
		where += " AND operation=$" + intPlaceholder(len(args))
	}
	if strings.TrimSpace(status) != "" {
		args = append(args, status)
		where += " AND status=$" + intPlaceholder(len(args))
	}
	var total int64
	if err := database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM ai_model_call_logs `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limit, offset := normalizePagination(page, pageSize)
	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)
	query := `SELECT id,provider,api_format,model,operation,status,latency_ms,prompt_tokens,completion_tokens,request_chars,response_chars,error_message,created_at FROM ai_model_call_logs ` + where + ` ORDER BY created_at DESC,id DESC LIMIT $` + intPlaceholder(len(listArgs)-1) + ` OFFSET $` + intPlaceholder(len(listArgs))
	rows, err := database.QueryCtx(ctx, s.db, query, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]models.AIModelCallLog, 0)
	for rows.Next() {
		item, err := scanAIModelCallLog(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *AdminDataService) AIModelCallStats(ctx context.Context) (*models.AIModelCallStats, error) {
	out := &models.AIModelCallStats{GeneratedAt: time.Now()}
	var avgLatency sql.NullFloat64
	_ = database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*),COALESCE(AVG(latency_ms),0),COALESCE(SUM(prompt_tokens+completion_tokens),0) FROM ai_model_call_logs`).Scan(&out.TotalCalls, &avgLatency, &out.TotalTokens)
	out.AvgLatencyMS = int64(avgLatency.Float64)
	_ = database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM ai_model_call_logs WHERE status='success'`).Scan(&out.SuccessCalls)
	_ = database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM ai_model_call_logs WHERE status<>'success'`).Scan(&out.ErrorCalls)
	_ = database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM ai_model_call_logs WHERE created_at >= $1`, time.Now().Add(-24*time.Hour)).Scan(&out.TodayCalls)

	dayExpr := "DATE(created_at)"
	if database.CurrentDialect != nil && database.CurrentDialect.Type == database.DBTypePostgres {
		dayExpr = "TO_CHAR(created_at, 'YYYY-MM-DD')"
	}
	rows, err := database.QueryCtx(ctx, s.db, `SELECT `+dayExpr+`,COUNT(*),SUM(CASE WHEN status='success' THEN 0 ELSE 1 END),COALESCE(AVG(latency_ms),0),COALESCE(SUM(prompt_tokens+completion_tokens),0) FROM ai_model_call_logs WHERE created_at >= $1 GROUP BY `+dayExpr+` ORDER BY `+dayExpr+` ASC`, time.Now().AddDate(0, 0, -13))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item models.AIModelCallDailyStat
			var avg sql.NullFloat64
			if rows.Scan(&item.Date, &item.Calls, &item.Errors, &avg, &item.Tokens) == nil {
				item.AvgMS = int64(avg.Float64)
				out.DailyStats = append(out.DailyStats, item)
			}
		}
	}
	modelRows, err := database.QueryCtx(ctx, s.db, `SELECT provider,model,COUNT(*),SUM(CASE WHEN status='success' THEN 0 ELSE 1 END),COALESCE(AVG(latency_ms),0),COALESCE(SUM(prompt_tokens+completion_tokens),0) FROM ai_model_call_logs GROUP BY provider,model ORDER BY COUNT(*) DESC LIMIT 8`)
	if err == nil {
		defer modelRows.Close()
		for modelRows.Next() {
			var item models.AIModelCallModelStat
			var avg sql.NullFloat64
			if modelRows.Scan(&item.Provider, &item.Model, &item.Calls, &item.Errors, &avg, &item.Tokens) == nil {
				item.AvgMS = int64(avg.Float64)
				out.ModelStats = append(out.ModelStats, item)
			}
		}
	}
	recentErrors, _, err := s.ListAIModelCallLogs(ctx, 1, 8, "", "", "error")
	if err == nil {
		out.RecentErrors = recentErrors
	}
	return out, nil
}

func (s *AdminDataService) SiteOperationsDashboard(ctx context.Context) (*models.SiteOperationsDashboard, error) {
	analytics, err := s.SiteAnalytics(ctx)
	if err != nil {
		return nil, err
	}
	out := &models.SiteOperationsDashboard{Analytics: *analytics, GeneratedAt: time.Now()}
	_ = database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM site_projects WHERE status='published'`).Scan(&out.PublishedProjects)
	_ = database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM site_projects WHERE status<>'published'`).Scan(&out.DraftProjects)
	_ = database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM site_projects WHERE is_featured=TRUE`).Scan(&out.FeaturedProjects)
	_ = database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM site_resources WHERE status<>'published'`).Scan(&out.DraftResources)
	if out.Analytics.VisitCount > 0 {
		out.ConversionRate = float64(out.Analytics.MessageCount) / float64(out.Analytics.VisitCount)
	}
	topProjects, _, err := s.ListSiteProjects(ctx, 1, 6, "published")
	if err == nil {
		out.TopProjects = topProjects
	}
	messages, _, err := s.ListSiteMessages(ctx, 1, 6, "")
	if err == nil {
		out.RecentMessages = messages
	}
	out.ContentHealth = []models.SiteContentHealthItem{
		{Label: "待发布文章", Value: out.DraftResources, Tone: healthTone(out.DraftResources, 0, 5)},
		{Label: "待发布项目", Value: out.DraftProjects, Tone: healthTone(out.DraftProjects, 0, 3)},
		{Label: "精选项目", Value: out.FeaturedProjects, Tone: healthTone(out.FeaturedProjects, 1, 999)},
		{Label: "待处理留言", Value: out.Analytics.PendingMessages, Tone: healthTone(out.Analytics.PendingMessages, 0, 0)},
	}
	return out, nil
}

func healthTone(value, minGood, maxGood int64) string {
	if value < minGood || value > maxGood {
		return "warning"
	}
	return "success"
}

func intPlaceholder(value int) string {
	return strconv.Itoa(value)
}

func scanSystemSetting(rows *sql.Rows) (models.SystemSetting, error) {
	var item models.SystemSetting
	err := rows.Scan(&item.ID, &item.SettingKey, &item.SettingValue, &item.GroupName, &item.ValueType, &item.Description, &item.IsSecret, &item.UpdatedAt)
	return item, err
}

func scanAIModelCallLog(rows *sql.Rows) (models.AIModelCallLog, error) {
	var item models.AIModelCallLog
	err := rows.Scan(&item.ID, &item.Provider, &item.APIFormat, &item.Model, &item.Operation, &item.Status, &item.LatencyMS, &item.PromptTokens, &item.CompletionTokens, &item.RequestChars, &item.ResponseChars, &item.ErrorMessage, &item.CreatedAt)
	return item, err
}
