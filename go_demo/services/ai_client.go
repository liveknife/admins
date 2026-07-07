package services

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"go-demo/models"
)

type AIClient interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
	Chat(ctx context.Context, req ChatRequest) (string, error)
	ChatEnabled() bool
}

type ChatRequest struct {
	System string
	User   string
}

type ChatTokenHandler func(token string) error

type ChatStreamingClient interface {
	ChatStream(ctx context.Context, req ChatRequest, onToken ChatTokenHandler) error
}

func supportsChatStream(client AIClient) bool {
	if client == nil {
		return false
	}
	_, ok := client.(ChatStreamingClient)
	return ok
}

type AIAPIError struct {
	Provider   string
	Path       string
	StatusCode int
	Message    string
}

func (e *AIAPIError) Error() string {
	if e == nil {
		return ""
	}
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("ai api %s returned %d: %s", e.Path, e.StatusCode, message)
	}
	return message
}

func NewAIClientFromEnv() AIClient {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("AI_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = defaultAIBaseURL(os.Getenv("AI_PROVIDER"))
	}
	chatModel := strings.TrimSpace(os.Getenv("AI_CHAT_MODEL"))
	embeddingModel := strings.TrimSpace(os.Getenv("AI_EMBEDDING_MODEL"))
	apiKey := strings.TrimSpace(os.Getenv("AI_API_KEY"))
	if baseURL != "" && (chatModel != "" || embeddingModel != "") {
		return &openAICompatibleClient{
			provider:       strings.TrimSpace(os.Getenv("AI_PROVIDER")),
			baseURL:        baseURL,
			apiKey:         apiKey,
			chatModel:      chatModel,
			embeddingModel: embeddingModel,
			temperature:    0.2,
			timeoutSeconds: 45,
			httpClient:     &http.Client{Timeout: 45 * time.Second},
			fallback:       localAIClient{},
		}
	}
	return localAIClient{}
}

func NewAIClient(db *sql.DB) AIClient {
	if db == nil {
		return NewAIClientFromEnv()
	}
	return &dbBackedAIClient{db: db, fallback: NewAIClientFromEnv()}
}

func defaultAIBaseURL(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openai":
		return "https://api.openai.com/v1"
	case "deepseek":
		return "https://api.deepseek.com/v1"
	case "ollama":
		return "http://localhost:11434/v1"
	case "anthropic", "claude":
		return "https://api.anthropic.com/v1"
	default:
		return ""
	}
}

type dbBackedAIClient struct {
	db       *sql.DB
	fallback AIClient
}

func (c *dbBackedAIClient) ChatEnabled() bool {
	return c.fallback != nil && c.fallback.ChatEnabled()
}

func (c *dbBackedAIClient) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	client, err := c.activeClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Embed(ctx, texts)
}

func (c *dbBackedAIClient) Chat(ctx context.Context, req ChatRequest) (string, error) {
	client, err := c.activeClient(ctx)
	if err != nil {
		return "", err
	}
	return client.Chat(ctx, req)
}

func (c *dbBackedAIClient) ChatStream(ctx context.Context, req ChatRequest, onToken ChatTokenHandler) error {
	client, err := c.activeClient(ctx)
	if err != nil {
		return err
	}
	streamer, ok := client.(ChatStreamingClient)
	if !ok {
		answer, err := client.Chat(ctx, req)
		if err != nil {
			return err
		}
		if onToken != nil {
			return onToken(answer)
		}
		return nil
	}
	return streamer.ChatStream(ctx, req, onToken)
}

func (c *dbBackedAIClient) activeClient(ctx context.Context) (AIClient, error) {
	cfg, err := (&AdminDataService{db: c.db}).ActiveAIModelConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return c.fallback, nil
	}
	return aiClientFromConfig(*cfg), nil
}

func aiClientFromConfig(cfg models.AIModelConfig) AIClient {
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 45
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultAIBaseURL(cfg.Provider)
	}
	if cfg.APIFormat == apiFormatAnthropic {
		return &anthropicClient{
			provider:       strings.TrimSpace(cfg.Provider),
			baseURL:        baseURL,
			apiKey:         strings.TrimSpace(cfg.APIKey),
			chatModel:      strings.TrimSpace(cfg.ChatModel),
			temperature:    cfg.Temperature,
			maxTokens:      cfg.MaxTokens,
			httpClient:     &http.Client{Timeout: time.Duration(timeout) * time.Second},
			embeddingModel: strings.TrimSpace(cfg.EmbeddingModel),
			fallback:       localAIClient{},
		}
	}
	return &openAICompatibleClient{
		provider:       strings.TrimSpace(cfg.Provider),
		baseURL:        baseURL,
		apiKey:         strings.TrimSpace(cfg.APIKey),
		chatModel:      strings.TrimSpace(cfg.ChatModel),
		embeddingModel: strings.TrimSpace(cfg.EmbeddingModel),
		temperature:    cfg.Temperature,
		maxTokens:      cfg.MaxTokens,
		timeoutSeconds: timeout,
		httpClient:     &http.Client{Timeout: time.Duration(timeout) * time.Second},
		fallback:       localAIClient{},
	}
}

type localAIClient struct{}

func (localAIClient) ChatEnabled() bool { return false }

func (localAIClient) Embed(_ context.Context, texts []string) ([][]float64, error) {
	out := make([][]float64, 0, len(texts))
	for _, text := range texts {
		out = append(out, hashedEmbedding(text, 256))
	}
	return out, nil
}

func (localAIClient) Chat(_ context.Context, _ ChatRequest) (string, error) {
	return "", errors.New("chat model is not configured")
}

type openAICompatibleClient struct {
	provider       string
	baseURL        string
	apiKey         string
	chatModel      string
	embeddingModel string
	temperature    float64
	maxTokens      int
	timeoutSeconds int
	httpClient     *http.Client
	fallback       localAIClient
}

func (c *openAICompatibleClient) ChatEnabled() bool { return c.chatModel != "" }

func (c *openAICompatibleClient) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if c.embeddingModel == "" {
		return c.fallback.Embed(ctx, texts)
	}
	body := map[string]any{"model": c.embeddingModel, "input": texts}
	var res struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := c.postJSON(ctx, "/embeddings", body, &res); err != nil {
		return nil, err
	}
	if res.Error != nil && res.Error.Message != "" {
		return nil, errors.New(res.Error.Message)
	}
	out := make([][]float64, 0, len(res.Data))
	for _, item := range res.Data {
		out = append(out, item.Embedding)
	}
	if len(out) != len(texts) {
		return nil, fmt.Errorf("embedding response count mismatch: got %d want %d", len(out), len(texts))
	}
	return out, nil
}

func (c *openAICompatibleClient) Chat(ctx context.Context, req ChatRequest) (string, error) {
	if c.chatModel == "" {
		return "", errors.New("chat model is not configured")
	}
	messages := []map[string]string{}
	if strings.TrimSpace(req.System) != "" {
		messages = append(messages, map[string]string{"role": "system", "content": req.System})
	}
	messages = append(messages, map[string]string{"role": "user", "content": req.User})
	body := map[string]any{
		"model":       c.chatModel,
		"messages":    messages,
		"temperature": c.temperature,
	}
	if c.maxTokens > 0 {
		body["max_tokens"] = c.maxTokens
	}
	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := c.postJSON(ctx, "/chat/completions", body, &res); err != nil {
		return "", err
	}
	if res.Error != nil && res.Error.Message != "" {
		return "", errors.New(res.Error.Message)
	}
	if len(res.Choices) == 0 {
		return "", errors.New("chat response has no choices")
	}
	answer := strings.TrimSpace(res.Choices[0].Message.Content)
	if answer == "" {
		return "", errors.New("chat response is empty")
	}
	return answer, nil
}

func (c *openAICompatibleClient) ChatStream(ctx context.Context, req ChatRequest, onToken ChatTokenHandler) error {
	if c.chatModel == "" {
		return errors.New("chat model is not configured")
	}
	messages := []map[string]string{}
	if strings.TrimSpace(req.System) != "" {
		messages = append(messages, map[string]string{"role": "system", "content": req.System})
	}
	messages = append(messages, map[string]string{"role": "user", "content": req.User})
	body := map[string]any{
		"model":       c.chatModel,
		"messages":    messages,
		"temperature": c.temperature,
		"stream":      true,
	}
	if c.maxTokens > 0 {
		body["max_tokens"] = c.maxTokens
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiResponseError(c.provider, "/chat/completions", resp)
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error,omitempty"`
		}
		if json.Unmarshal([]byte(data), &chunk) != nil {
			continue
		}
		if chunk.Error != nil && chunk.Error.Message != "" {
			return errors.New(chunk.Error.Message)
		}
		for _, choice := range chunk.Choices {
			token := choice.Delta.Content
			if token == "" {
				token = choice.Message.Content
			}
			if token != "" && onToken != nil {
				if err := onToken(token); err != nil {
					return err
				}
			}
		}
	}
	return scanner.Err()
}

func (c *openAICompatibleClient) postJSON(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiResponseError(c.provider, path, resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type anthropicClient struct {
	provider       string
	baseURL        string
	apiKey         string
	chatModel      string
	embeddingModel string
	temperature    float64
	maxTokens      int
	httpClient     *http.Client
	fallback       localAIClient
}

func (c *anthropicClient) ChatEnabled() bool { return c.chatModel != "" }

func (c *anthropicClient) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	return c.fallback.Embed(ctx, texts)
}

func (c *anthropicClient) Chat(ctx context.Context, req ChatRequest) (string, error) {
	if c.chatModel == "" {
		return "", errors.New("chat model is not configured")
	}
	maxTokens := c.maxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	body := map[string]any{
		"model":       c.chatModel,
		"max_tokens":  maxTokens,
		"temperature": c.temperature,
		"messages": []map[string]string{
			{"role": "user", "content": req.User},
		},
	}
	if strings.TrimSpace(req.System) != "" {
		body["system"] = req.System
	}
	var res struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := c.postJSON(ctx, "/messages", body, &res); err != nil {
		return "", err
	}
	if res.Error != nil && res.Error.Message != "" {
		return "", errors.New(res.Error.Message)
	}
	var parts []string
	for _, item := range res.Content {
		if strings.TrimSpace(item.Text) != "" {
			parts = append(parts, item.Text)
		}
	}
	answer := strings.TrimSpace(strings.Join(parts, "\n"))
	if answer == "" {
		return "", errors.New("chat response is empty")
	}
	return answer, nil
}

func (c *anthropicClient) postJSON(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiResponseError(c.provider, path, resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func apiResponseError(provider, path string, resp *http.Response) error {
	const maxErrorBody = 8192
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	message := apiErrorMessage(raw)
	if message == "" {
		message = resp.Status
	}
	return &AIAPIError{
		Provider:   provider,
		Path:       path,
		StatusCode: resp.StatusCode,
		Message:    message,
	}
}

func apiErrorMessage(raw []byte) string {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return ""
	}
	var parsed struct {
		Error any `json:"error"`
	}
	if json.Unmarshal(raw, &parsed) == nil && parsed.Error != nil {
		switch value := parsed.Error.(type) {
		case string:
			return strings.TrimSpace(value)
		case map[string]any:
			if message, ok := value["message"].(string); ok && strings.TrimSpace(message) != "" {
				return strings.TrimSpace(message)
			}
			if message, ok := value["error"].(string); ok && strings.TrimSpace(message) != "" {
				return strings.TrimSpace(message)
			}
		}
	}
	var generic map[string]any
	if json.Unmarshal(raw, &generic) == nil {
		for _, key := range []string{"message", "error_description", "detail"} {
			if message, ok := generic[key].(string); ok && strings.TrimSpace(message) != "" {
				return strings.TrimSpace(message)
			}
		}
	}
	return text
}

func hashedEmbedding(text string, dims int) []float64 {
	vec := make([]float64, dims)
	for _, token := range embeddingTokens(text) {
		sum := sha256.Sum256([]byte(token))
		idx := int(binary.BigEndian.Uint32(sum[:4]) % uint32(dims))
		sign := 1.0
		if sum[4]&1 == 1 {
			sign = -1
		}
		vec[idx] += sign
	}
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	if norm == 0 {
		return vec
	}
	norm = math.Sqrt(norm)
	for i := range vec {
		vec[i] /= norm
	}
	return vec
}

func embeddingTokens(text string) []string {
	text = strings.ToLower(strings.TrimSpace(text))
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == ',' || r == '，' || r == '.' || r == '。' || r == '?' || r == '？' || r == ';' || r == '；' || r == ':' || r == '：'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if utf8.RuneCountInString(part) <= 12 {
			out = append(out, part)
			continue
		}
		runes := []rune(part)
		for i := 0; i < len(runes); i += 6 {
			end := i + 6
			if end > len(runes) {
				end = len(runes)
			}
			out = append(out, string(runes[i:end]))
		}
	}
	return out
}
