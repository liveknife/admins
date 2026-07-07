package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"

	"go-demo/database"
	"go-demo/models"
)

const (
	apiFormatOpenAI    = "openai"
	apiFormatAnthropic = "anthropic"
	apiFormatCustom    = "custom"
)

type AIModelConfigInput struct {
	Name           string
	Provider       string
	APIFormat      string
	BaseURL        string
	APIKey         string
	ChatModel      string
	EmbeddingModel string
	Temperature    float64
	MaxTokens      int
	TimeoutSeconds int
	ExtraJSON      string
	IsDefault      bool
	Enabled        bool
}

func (s *AdminDataService) ListAIModelConfigs(ctx context.Context) ([]models.AIModelConfig, error) {
	rows, err := database.QueryCtx(ctx, s.db, `SELECT id,name,provider,api_format,base_url,api_key,chat_model,embedding_model,temperature,max_tokens,timeout_seconds,extra_json,is_default,enabled,last_test_status,last_test_message,last_test_at,created_at,updated_at FROM ai_model_configs ORDER BY is_default DESC,enabled DESC,updated_at DESC,id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.AIModelConfig, 0)
	for rows.Next() {
		item, err := scanAIModelConfig(rows, false)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *AdminDataService) GetAIModelConfig(ctx context.Context, id int64) (*models.AIModelConfig, error) {
	return s.getAIModelConfig(ctx, id, false)
}

func (s *AdminDataService) getAIModelConfig(ctx context.Context, id int64, includeSecret bool) (*models.AIModelConfig, error) {
	row := database.QueryRowCtx(ctx, s.db, `SELECT id,name,provider,api_format,base_url,api_key,chat_model,embedding_model,temperature,max_tokens,timeout_seconds,extra_json,is_default,enabled,last_test_status,last_test_message,last_test_at,created_at,updated_at FROM ai_model_configs WHERE id=$1`, id)
	item, err := scanAIModelConfig(row, includeSecret)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *AdminDataService) ActiveAIModelConfig(ctx context.Context) (*models.AIModelConfig, error) {
	row := database.QueryRowCtx(ctx, s.db, `SELECT id,name,provider,api_format,base_url,api_key,chat_model,embedding_model,temperature,max_tokens,timeout_seconds,extra_json,is_default,enabled,last_test_status,last_test_message,last_test_at,created_at,updated_at FROM ai_model_configs WHERE enabled=TRUE ORDER BY is_default DESC,updated_at DESC,id DESC LIMIT 1`)
	item, err := scanAIModelConfig(row, true)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *AdminDataService) SaveAIModelConfig(ctx context.Context, id int64, input AIModelConfigInput) (*models.AIModelConfig, error) {
	input = normalizeAIModelConfigInput(input)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if input.IsDefault {
		if _, err := database.ExecTxCtx(ctx, tx, `UPDATE ai_model_configs SET is_default=FALSE,updated_at=`+database.Now()); err != nil {
			return nil, err
		}
	}

	if id == 0 {
		insert := `INSERT INTO ai_model_configs(name,provider,api_format,base_url,api_key,chat_model,embedding_model,temperature,max_tokens,timeout_seconds,extra_json,is_default,enabled) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`
		id, err = database.InsertID(tx, insert, input.Name, input.Provider, input.APIFormat, input.BaseURL, input.APIKey, input.ChatModel, input.EmbeddingModel, input.Temperature, input.MaxTokens, input.TimeoutSeconds, input.ExtraJSON, input.IsDefault, input.Enabled)
		if err != nil {
			return nil, err
		}
	} else {
		if input.APIKey == "" {
			_, err = database.ExecTxCtx(ctx, tx, `UPDATE ai_model_configs SET name=$1,provider=$2,api_format=$3,base_url=$4,chat_model=$5,embedding_model=$6,temperature=$7,max_tokens=$8,timeout_seconds=$9,extra_json=$10,is_default=$11,enabled=$12,updated_at=`+database.Now()+` WHERE id=$13`,
				input.Name, input.Provider, input.APIFormat, input.BaseURL, input.ChatModel, input.EmbeddingModel, input.Temperature, input.MaxTokens, input.TimeoutSeconds, input.ExtraJSON, input.IsDefault, input.Enabled, id)
		} else {
			_, err = database.ExecTxCtx(ctx, tx, `UPDATE ai_model_configs SET name=$1,provider=$2,api_format=$3,base_url=$4,api_key=$5,chat_model=$6,embedding_model=$7,temperature=$8,max_tokens=$9,timeout_seconds=$10,extra_json=$11,is_default=$12,enabled=$13,updated_at=`+database.Now()+` WHERE id=$14`,
				input.Name, input.Provider, input.APIFormat, input.BaseURL, input.APIKey, input.ChatModel, input.EmbeddingModel, input.Temperature, input.MaxTokens, input.TimeoutSeconds, input.ExtraJSON, input.IsDefault, input.Enabled, id)
		}
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetAIModelConfig(ctx, id)
}

func (s *AdminDataService) SetDefaultAIModelConfig(ctx context.Context, id int64) (*models.AIModelConfig, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := database.ExecTxCtx(ctx, tx, `UPDATE ai_model_configs SET is_default=FALSE,updated_at=`+database.Now()); err != nil {
		return nil, err
	}
	if _, err := database.ExecTxCtx(ctx, tx, `UPDATE ai_model_configs SET is_default=TRUE,enabled=TRUE,updated_at=`+database.Now()+` WHERE id=$1`, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetAIModelConfig(ctx, id)
}

func (s *AdminDataService) DeleteAIModelConfig(ctx context.Context, id int64) error {
	_, err := database.ExecCtx(ctx, s.db, `DELETE FROM ai_model_configs WHERE id=$1`, id)
	return err
}

func (s *AdminDataService) UpdateAIModelConfigTestResult(ctx context.Context, id int64, status, message string) error {
	_, err := database.ExecCtx(ctx, s.db, `UPDATE ai_model_configs SET last_test_status=$1,last_test_message=$2,last_test_at=`+database.Now()+`,updated_at=`+database.Now()+` WHERE id=$3`, status, limitRunes(message, 1000), id)
	return err
}

func (s *AdminDataService) TestAIModelConfig(ctx context.Context, id int64) (*models.AIModelConfig, error) {
	cfg, err := s.getAIModelConfig(ctx, id, true)
	if err != nil {
		return nil, err
	}
	client := aiClientFromConfig(*cfg)
	status, message := "success", "模型配置可用"
	if strings.TrimSpace(cfg.ChatModel) != "" {
		if _, err := client.Chat(ctx, ChatRequest{System: "Reply with OK only.", User: "ping"}); err != nil {
			status, message = "failed", friendlyAIError(cfg.Provider, err)
		}
	} else if strings.TrimSpace(cfg.EmbeddingModel) != "" {
		if _, err := client.Embed(ctx, []string{"ping"}); err != nil {
			status, message = "failed", friendlyAIError(cfg.Provider, err)
		}
	} else {
		status, message = "failed", "请至少配置 chat_model 或 embedding_model"
	}
	if err := s.UpdateAIModelConfigTestResult(ctx, id, status, message); err != nil {
		return nil, err
	}
	return s.GetAIModelConfig(ctx, id)
}

func friendlyAIError(provider string, err error) string {
	if err == nil {
		return ""
	}
	var apiErr *AIAPIError
	if errors.As(err, &apiErr) {
		providerName := strings.TrimSpace(provider)
		if providerName == "" {
			providerName = strings.TrimSpace(apiErr.Provider)
		}
		prefix := ""
		if providerName != "" {
			prefix = strings.ToUpper(providerName[:1]) + providerName[1:] + ": "
		}
		raw := strings.TrimSpace(apiErr.Message)
		lower := strings.ToLower(raw)
		switch apiErr.StatusCode {
		case 401, 403:
			return prefix + "API Key 无效、权限不足，或账号未开通该模型。"
		case 404:
			return prefix + "Base URL、接口路径或模型名称不正确，请检查服务商地址和模型 ID。"
		case 408:
			return prefix + "请求超时，请检查网络、Base URL 或调大超时时间。"
		case 429:
			return prefix + "请求过快、额度不足或触发限流，请检查余额和频率限制。"
		}
		if strings.Contains(lower, "model") && (strings.Contains(lower, "not found") || strings.Contains(lower, "does not exist") || strings.Contains(lower, "invalid")) {
			return prefix + "模型名称可能不正确，服务商返回：" + raw
		}
		if raw != "" {
			return prefix + raw
		}
		return prefix + err.Error()
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "连接大模型服务超时，请检查 Base URL、网络连通性或调大超时时间。"
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "no such host") || strings.Contains(lower, "connection refused") || strings.Contains(lower, "connectex") || strings.Contains(lower, "timeout") {
		return "无法连接大模型服务，请检查 Base URL、代理、防火墙和服务商网络状态。"
	}
	return err.Error()
}

type aiConfigScanner interface {
	Scan(dest ...any) error
}

func scanAIModelConfig(scanner aiConfigScanner, includeSecret bool) (models.AIModelConfig, error) {
	var item models.AIModelConfig
	var apiKey string
	if err := scanner.Scan(&item.ID, &item.Name, &item.Provider, &item.APIFormat, &item.BaseURL, &apiKey, &item.ChatModel, &item.EmbeddingModel, &item.Temperature, &item.MaxTokens, &item.TimeoutSeconds, &item.ExtraJSON, &item.IsDefault, &item.Enabled, &item.LastTestStatus, &item.LastTestMessage, &item.LastTestAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	item.HasAPIKey = strings.TrimSpace(apiKey) != ""
	item.MaskedAPIKey = maskSecret(apiKey)
	if includeSecret {
		item.APIKey = apiKey
	}
	return item, nil
}

func normalizeAIModelConfigInput(input AIModelConfigInput) AIModelConfigInput {
	input.Name = strings.TrimSpace(input.Name)
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.APIFormat = strings.ToLower(strings.TrimSpace(input.APIFormat))
	input.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	input.APIKey = strings.TrimSpace(input.APIKey)
	input.ChatModel = strings.TrimSpace(input.ChatModel)
	input.EmbeddingModel = strings.TrimSpace(input.EmbeddingModel)
	input.ExtraJSON = strings.TrimSpace(input.ExtraJSON)
	if input.Provider == "" {
		input.Provider = "openai"
	}
	if input.APIFormat == "" {
		input.APIFormat = apiFormatOpenAI
	}
	if input.APIFormat != apiFormatOpenAI && input.APIFormat != apiFormatAnthropic && input.APIFormat != apiFormatCustom {
		input.APIFormat = apiFormatCustom
	}
	if input.Name == "" {
		input.Name = fmt.Sprintf("%s %s", strings.ToUpper(input.Provider[:1])+input.Provider[1:], "模型")
	}
	if input.BaseURL == "" && input.APIFormat == apiFormatOpenAI {
		input.BaseURL = defaultAIBaseURL(input.Provider)
	}
	if input.TimeoutSeconds <= 0 {
		input.TimeoutSeconds = 45
	}
	if input.Temperature < 0 {
		input.Temperature = 0
	}
	if input.Temperature > 2 {
		input.Temperature = 2
	}
	if input.MaxTokens < 0 {
		input.MaxTokens = 0
	}
	return input
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 8 {
		return "****"
	}
	return string(runes[:4]) + "..." + string(runes[len(runes)-4:])
}
