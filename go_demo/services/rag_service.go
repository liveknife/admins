package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-demo/database"
	"go-demo/models"
)

const (
	knowledgeSourceResource     = "site_resource"
	knowledgeSourceProject      = "site_project"
	knowledgeSourceTech         = "site_tech_stack"
	knowledgeSourceTimeline     = "site_timeline"
	knowledgeSourceDocument     = "uploaded_document"
	knowledgeVisibilityPublic   = "public"
	knowledgeVisibilityInternal = "internal"
	defaultRAGTopK              = 6
	defaultRAGMinScore          = 0.08
)

type RAGService struct {
	db  *sql.DB
	ai  AIClient
	top int
}

type knowledgeChunkInput struct {
	SourceType string
	SourceID   int64
	Visibility string
	Title      string
	Summary    string
	Content    string
	Metadata   map[string]any
}

type knowledgeChunk struct {
	ID           int64
	SourceType   string
	SourceID     int64
	Visibility   string
	Title        string
	Summary      string
	Content      string
	Metadata     map[string]any
	Embedding    []float64
	Score        float64
	VectorScore  float64
	BM25Score    float64
	KeywordScore float64
	SourceWeight float64
	RerankScore  float64
}

func NewRAGService(db *sql.DB) *RAGService {
	return &RAGService{db: db, ai: NewAIClient(db), top: envInt("RAG_TOP_K", defaultRAGTopK)}
}

func (r *RAGService) SyncSiteResource(ctx context.Context, item *models.SiteResource) error {
	if r == nil || item == nil {
		return nil
	}
	if item.Status != "published" {
		return r.DeleteSource(ctx, knowledgeSourceResource, item.ID)
	}
	chunks := buildSiteResourceChunks(*item)
	return r.replaceChunks(ctx, knowledgeSourceResource, item.ID, chunks)
}

func (r *RAGService) SyncSiteProject(ctx context.Context, item *models.SiteProject) error {
	if r == nil || item == nil {
		return nil
	}
	if item.Status != "published" {
		return r.DeleteSource(ctx, knowledgeSourceProject, item.ID)
	}
	return r.replaceChunks(ctx, knowledgeSourceProject, item.ID, buildSiteProjectChunks(*item))
}

func (r *RAGService) SyncSiteTechStack(ctx context.Context, item *models.SiteTechStack) error {
	if r == nil || item == nil {
		return nil
	}
	if !item.IsActive {
		return r.DeleteSource(ctx, knowledgeSourceTech, item.ID)
	}
	return r.replaceChunks(ctx, knowledgeSourceTech, item.ID, buildSiteTechStackChunks(*item))
}

func (r *RAGService) SyncSiteTimelineEvent(ctx context.Context, item *models.SiteTimelineEvent) error {
	if r == nil || item == nil {
		return nil
	}
	if item.Status != "published" {
		return r.DeleteSource(ctx, knowledgeSourceTimeline, item.ID)
	}
	return r.replaceChunks(ctx, knowledgeSourceTimeline, item.ID, buildSiteTimelineChunks(*item))
}

func (r *RAGService) SyncUploadedDocument(ctx context.Context, item *models.UploadedDocument) error {
	if r == nil || item == nil {
		return nil
	}
	if item.Status != "active" {
		return r.DeleteSource(ctx, knowledgeSourceDocument, item.ID)
	}
	return r.replaceChunks(ctx, knowledgeSourceDocument, item.ID, buildUploadedDocumentChunks(*item))
}

func (r *RAGService) DeleteSource(ctx context.Context, sourceType string, sourceID int64) error {
	if r == nil {
		return nil
	}
	_, err := database.ExecCtx(ctx, r.db, `DELETE FROM knowledge_chunks WHERE source_type=$1 AND source_id=$2`, sourceType, sourceID)
	return err
}

func (r *RAGService) Stats(ctx context.Context) (*models.RAGIndexStats, error) {
	stats := &models.RAGIndexStats{
		BySource:          map[string]int64{},
		ByVisibility:      map[string]int64{},
		TopK:              r.top,
		MinScore:          r.minScore(),
		RerankTopN:        r.rerankTopN(),
		SourceWeights:     r.sourceWeights(),
		ChatEnabled:       r.chatEnabled(ctx),
		StreamingEnabled:  r.streamingEnabled(ctx),
		VectorBackend:     r.vectorBackend(),
		PGVectorAvailable: r.pgVectorAvailable(ctx),
	}
	rows, err := database.QueryCtx(ctx, r.db, `SELECT source_type,visibility,COUNT(*) FROM knowledge_chunks WHERE status='active' GROUP BY source_type,visibility`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sourceType string
		var visibility string
		var count int64
		if err := rows.Scan(&sourceType, &visibility, &count); err != nil {
			return nil, err
		}
		stats.BySource[sourceType] += count
		stats.ByVisibility[normalizeVisibility(visibility)] += count
		stats.TotalChunks += count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var updatedAt sql.NullTime
	if err := database.QueryRowCtx(ctx, r.db, `SELECT MAX(updated_at) FROM knowledge_chunks WHERE status='active'`).Scan(&updatedAt); err != nil {
		return nil, err
	}
	if updatedAt.Valid {
		t := updatedAt.Time
		stats.UpdatedAt = &t
	}
	if job, err := r.LatestJob(ctx); err == nil {
		stats.LatestJob = job
	}
	_ = database.QueryRowCtx(ctx, r.db, `SELECT COUNT(*),COALESCE(SUM(CASE WHEN matched THEN 1 ELSE 0 END),0),COALESCE(AVG(latency_ms),0),COALESCE(AVG(source_count),0) FROM rag_query_logs`).Scan(
		&stats.QueryCount, &stats.HitCount, &stats.AverageLatencyMs, &stats.AverageSourceCount,
	)
	_ = database.QueryRowCtx(ctx, r.db, `SELECT COUNT(*),COALESCE(SUM(CASE WHEN rating='up' THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN rating='down' THEN 1 ELSE 0 END),0) FROM rag_feedback`).Scan(
		&stats.FeedbackCount, &stats.PositiveFeedback, &stats.NegativeFeedback,
	)
	return stats, nil
}

func (r *RAGService) Rebuild(ctx context.Context) (*models.RAGIndexStats, error) {
	if _, err := database.ExecCtx(ctx, r.db, `DELETE FROM knowledge_chunks`); err != nil {
		return nil, err
	}
	if err := r.ensureInitialIndex(ctx); err != nil {
		return nil, err
	}
	return r.Stats(ctx)
}

func (r *RAGService) EnqueueRebuild(ctx context.Context) (*models.RAGIndexJob, error) {
	return r.createJob(ctx, "rebuild")
}

func (r *RAGService) RetryJob(ctx context.Context, id int64) (*models.RAGIndexJob, error) {
	if _, err := database.ExecCtx(ctx, r.db, `UPDATE rag_index_jobs SET status='pending',error_message='',updated_at=`+database.Now()+` WHERE id=$1`, id); err != nil {
		return nil, err
	}
	return r.GetJob(ctx, id)
}

func (r *RAGService) StartWorker(ctx context.Context) {
	if r == nil || r.db == nil {
		return
	}
	_, _ = database.ExecCtx(ctx, r.db, `UPDATE rag_index_jobs SET status='pending',updated_at=`+database.Now()+` WHERE status IN ('running','retrying') AND finished_at IS NULL`)
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.processNextJob(ctx)
			}
		}
	}()
}

func (r *RAGService) processNextJob(ctx context.Context) {
	var id int64
	err := database.QueryRowCtx(ctx, r.db, `SELECT id FROM rag_index_jobs WHERE status IN ('pending','retrying') ORDER BY id ASC LIMIT 1`).Scan(&id)
	if err != nil {
		return
	}
	r.processJob(ctx, id)
}

func (r *RAGService) ListJobs(ctx context.Context, limit int) ([]models.RAGIndexJob, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := database.QueryCtx(ctx, r.db, `SELECT id,job_type,status,retry_count,max_retries,error_message,started_at,finished_at,created_at,updated_at FROM rag_index_jobs ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.RAGIndexJob, 0)
	for rows.Next() {
		job, err := scanRAGIndexJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

func (r *RAGService) LatestJob(ctx context.Context) (*models.RAGIndexJob, error) {
	job, err := scanRAGIndexJob(database.QueryRowCtx(ctx, r.db, `SELECT id,job_type,status,retry_count,max_retries,error_message,started_at,finished_at,created_at,updated_at FROM rag_index_jobs ORDER BY id DESC LIMIT 1`))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *RAGService) GetJob(ctx context.Context, id int64) (*models.RAGIndexJob, error) {
	job, err := scanRAGIndexJob(database.QueryRowCtx(ctx, r.db, `SELECT id,job_type,status,retry_count,max_retries,error_message,started_at,finished_at,created_at,updated_at FROM rag_index_jobs WHERE id=$1`, id))
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *RAGService) ListQueryLogs(ctx context.Context, limit int) ([]models.RAGQueryLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := database.QueryCtx(ctx, r.db, `SELECT id,question,answer,matched,source_count,top_score,latency_ms,used_chat_model,source_json,created_at FROM rag_query_logs ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.RAGQueryLog, 0)
	for rows.Next() {
		var item models.RAGQueryLog
		if err := rows.Scan(&item.ID, &item.Question, &item.Answer, &item.Matched, &item.SourceCount, &item.TopScore, &item.LatencyMs, &item.UsedChatModel, &item.SourceJSON, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *RAGService) ListFeedback(ctx context.Context, limit int, rating string) ([]models.RAGFeedback, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rating = strings.ToLower(strings.TrimSpace(rating))
	query := `SELECT id,query_log_id,question,rating,comment,ip_address,user_agent,created_at FROM rag_feedback`
	args := []any{}
	if rating == "up" || rating == "down" || rating == "neutral" {
		query += ` WHERE rating=$1`
		args = append(args, rating)
	}
	query += ` ORDER BY id DESC LIMIT $` + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := database.QueryCtx(ctx, r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.RAGFeedback, 0)
	for rows.Next() {
		var item models.RAGFeedback
		if err := rows.Scan(&item.ID, &item.QueryLogID, &item.Question, &item.Rating, &item.Comment, &item.IPAddress, &item.UserAgent, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *RAGService) ListChunks(ctx context.Context, sourceType string, sourceID int64, limit int) ([]models.KnowledgeChunkPreview, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	where := "WHERE status='active'"
	args := []any{}
	if strings.TrimSpace(sourceType) != "" {
		where += fmt.Sprintf(" AND source_type=$%d", len(args)+1)
		args = append(args, strings.TrimSpace(sourceType))
	}
	if sourceID > 0 {
		where += fmt.Sprintf(" AND source_id=$%d", len(args)+1)
		args = append(args, sourceID)
	}
	args = append(args, limit)
	rows, err := database.QueryCtx(ctx, r.db, `SELECT id,source_type,source_id,visibility,title,summary,content,metadata_json,token_count,status,created_at,updated_at FROM knowledge_chunks `+where+` ORDER BY source_type,source_id,id ASC LIMIT $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.KnowledgeChunkPreview, 0)
	for rows.Next() {
		var item models.KnowledgeChunkPreview
		var rawMetadata string
		if err := rows.Scan(&item.ID, &item.SourceType, &item.SourceID, &item.Visibility, &item.Title, &item.Summary, &item.Content, &rawMetadata, &item.TokenCount, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Visibility = normalizeVisibility(item.Visibility)
		_ = json.Unmarshal([]byte(rawMetadata), &item.Metadata)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *RAGService) SearchDiagnostics(ctx context.Context, question string, includeInternal bool, topK int) ([]models.KnowledgeSource, error) {
	_ = r.ensureInitialIndex(ctx)
	chunks, err := r.search(ctx, question, topK, includeInternal)
	if err != nil {
		return nil, err
	}
	return chunksToSources(chunks, question), nil
}

func (r *RAGService) RunEval(ctx context.Context, includeInternal bool) (*models.RAGEvalRun, error) {
	cases := loadRAGEvalCases()
	run := &models.RAGEvalRun{
		Total:     len(cases),
		Results:   make([]models.RAGEvalCaseResult, 0, len(cases)),
		CreatedAt: time.Now(),
	}
	for _, item := range cases {
		start := time.Now()
		chunks, err := r.search(ctx, item.Question, r.top, includeInternal)
		if err != nil {
			return nil, err
		}
		sources := chunksToSources(chunks, item.Question)
		answer := r.extractiveAnswer(item.Question, chunks)
		if r.chatEnabled(ctx) {
			if generated, err := r.generateAnswer(ctx, item.Question, chunks); err == nil && strings.TrimSpace(generated) != "" {
				answer = generated
			}
		}
		result := models.RAGEvalCaseResult{
			Case:          item,
			Matched:       len(sources) > 0,
			RecallHit:     evalRecallHit(item, sources),
			AnswerQuality: evalAnswerQuality(item, answer),
			LatencyMs:     time.Since(start).Milliseconds(),
			Sources:       sources,
			Answer:        answer,
		}
		if len(sources) > 0 {
			result.TopScore = sources[0].Score
			run.Matched++
		}
		if result.RecallHit {
			run.RecallHits++
		}
		run.AverageTopScore += result.TopScore
		run.AverageQuality += result.AnswerQuality
		run.AverageLatencyMs += float64(result.LatencyMs)
		run.Results = append(run.Results, result)
	}
	if run.Total > 0 {
		run.AverageTopScore = math.Round(run.AverageTopScore/float64(run.Total)*10000) / 10000
		run.AverageQuality = math.Round(run.AverageQuality/float64(run.Total)*10000) / 10000
		run.AverageLatencyMs = math.Round(run.AverageLatencyMs/float64(run.Total)*100) / 100
	}
	return run, nil
}

func (r *RAGService) AskSiteKnowledge(ctx context.Context, question string, fallback func(context.Context, string) (*models.SiteKnowledgeAnswer, error)) (*models.SiteKnowledgeAnswer, error) {
	start := time.Now()
	question = strings.TrimSpace(question)
	if question == "" {
		return &models.SiteKnowledgeAnswer{Question: question, Answer: "可以问我关于 React、Go、数据库、项目经验或学习笔记的问题。"}, nil
	}

	_ = r.ensureInitialIndex(ctx)
	matches, err := r.search(ctx, question, r.top, false)
	if err != nil || len(matches) == 0 {
		if fallback != nil {
			answer, fallbackErr := fallback(ctx, question)
			if answer != nil {
				answer.QueryLogID = r.logQuery(ctx, question, answer.Answer, nil, false, time.Since(start), false)
				answer.Suggestions = r.suggestQuestions(question, nil)
			}
			return answer, fallbackErr
		}
		return nil, err
	}

	answer := r.extractiveAnswer(question, matches)
	usedChat := false
	if r.chatEnabled(ctx) {
		if generated, err := r.generateAnswer(ctx, question, matches); err == nil && strings.TrimSpace(generated) != "" {
			answer = generated
			usedChat = true
		}
	}

	resources, projects := r.sourcesToLegacyMatches(ctx, matches)
	sources := chunksToSources(matches, question)
	queryLogID := r.logQuery(ctx, question, answer, sources, true, time.Since(start), usedChat)
	return &models.SiteKnowledgeAnswer{
		Question:    question,
		Answer:      answer,
		Sources:     sources,
		Matches:     resources,
		Projects:    projects,
		Suggestions: r.suggestQuestions(question, matches),
		QueryLogID:  queryLogID,
	}, nil
}

func (r *RAGService) AskSiteKnowledgeStream(ctx context.Context, question string, fallback func(context.Context, string) (*models.SiteKnowledgeAnswer, error), onToken ChatTokenHandler) (*models.SiteKnowledgeAnswer, error) {
	start := time.Now()
	question = strings.TrimSpace(question)
	if question == "" {
		answer := "可以问我关于 React、Go、数据库、项目经验或学习笔记的问题。"
		if onToken != nil {
			_ = onToken(answer)
		}
		return &models.SiteKnowledgeAnswer{Question: question, Answer: answer}, nil
	}

	_ = r.ensureInitialIndex(ctx)
	matches, err := r.search(ctx, question, r.top, false)
	if err != nil || len(matches) == 0 {
		if fallback != nil {
			answer, fallbackErr := fallback(ctx, question)
			if answer != nil {
				answer.QueryLogID = r.logQuery(ctx, question, answer.Answer, nil, false, time.Since(start), false)
				answer.Suggestions = r.suggestQuestions(question, nil)
				if onToken != nil {
					for _, token := range streamTextChunks(answer.Answer, 18) {
						if err := onToken(token); err != nil {
							return answer, err
						}
					}
				}
			}
			return answer, fallbackErr
		}
		return nil, err
	}

	resources, projects := r.sourcesToLegacyMatches(ctx, matches)
	sources := chunksToSources(matches, question)
	answer := ""
	usedChat := false
	if r.chatEnabled(ctx) {
		req := r.answerChatRequest(question, matches)
		if streamer, ok := r.ai.(ChatStreamingClient); ok {
			var builder strings.Builder
			if err := streamer.ChatStream(ctx, req, func(token string) error {
				builder.WriteString(token)
				if onToken != nil {
					return onToken(token)
				}
				return nil
			}); err == nil && strings.TrimSpace(builder.String()) != "" {
				answer = strings.TrimSpace(builder.String())
				usedChat = true
			}
		}
		if answer == "" {
			if generated, err := r.generateAnswer(ctx, question, matches); err == nil && strings.TrimSpace(generated) != "" {
				answer = generated
				usedChat = true
			}
		}
	}
	if answer == "" {
		answer = r.extractiveAnswer(question, matches)
	}
	if !usedChat && onToken != nil {
		for _, token := range streamTextChunks(answer, 18) {
			if err := onToken(token); err != nil {
				return nil, err
			}
		}
	}
	queryLogID := r.logQuery(ctx, question, answer, sources, true, time.Since(start), usedChat)
	return &models.SiteKnowledgeAnswer{
		Question:    question,
		Answer:      answer,
		Sources:     sources,
		Matches:     resources,
		Projects:    projects,
		Suggestions: r.suggestQuestions(question, matches),
		QueryLogID:  queryLogID,
	}, nil
}

func (r *RAGService) AskAdminKnowledge(ctx context.Context, question string) (*models.AIAssistantResult, bool, error) {
	start := time.Now()
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, false, nil
	}
	_ = r.ensureInitialIndex(ctx)
	matches, err := r.search(ctx, question, r.top, true)
	if err != nil {
		return nil, false, err
	}
	if len(matches) == 0 {
		r.logQuery(ctx, question, "", nil, false, time.Since(start), false)
		return nil, false, nil
	}
	answer := r.extractiveAnswer(question, matches)
	usedChat := false
	if r.chatEnabled(ctx) {
		if generated, err := r.generateAnswer(ctx, question, matches); err == nil && strings.TrimSpace(generated) != "" {
			answer = generated
			usedChat = true
		}
	}
	sources := chunksToSources(matches, question)
	result := &models.AIAssistantResult{
		Question: question,
		Answer:   answer,
		Insights: []string{
			fmt.Sprintf("已从知识库命中 %d 个相关片段。", len(matches)),
			"答案基于官网资源、项目和技术栈内容生成。",
		},
		Metrics: map[string]int64{"sources": int64(len(matches))},
		Sources: sources,
	}
	result.Insights = []string{
		fmt.Sprintf("已从知识库命中 %d 个相关片段。", len(matches)),
		"答案基于官网资源、项目、技术栈、时间线和后台上传文档生成。",
	}
	r.logQuery(ctx, question, answer, sources, true, time.Since(start), usedChat)
	for _, source := range result.Sources {
		result.Rows = append(result.Rows, map[string]any{
			"type":  source.SourceType,
			"id":    source.SourceID,
			"title": source.Title,
			"score": source.Score,
		})
	}
	return result, true, nil
}

func (r *RAGService) replaceChunks(ctx context.Context, sourceType string, sourceID int64, chunks []knowledgeChunkInput) error {
	_, err := database.ExecCtx(ctx, r.db, `DELETE FROM knowledge_chunks WHERE source_type=$1 AND source_id=$2`, sourceType, sourceID)
	if err != nil {
		return err
	}
	if len(chunks) == 0 {
		return nil
	}
	texts := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		texts = append(texts, chunk.Content)
	}
	embeddings, err := r.ai.Embed(ctx, texts)
	if err != nil {
		return err
	}
	if len(embeddings) != len(chunks) {
		return fmt.Errorf("embedding count mismatch: got %d want %d", len(embeddings), len(chunks))
	}
	for i, chunk := range chunks {
		metadata, _ := json.Marshal(chunk.Metadata)
		embedding, _ := json.Marshal(embeddings[i])
		contentHash := hashText(chunk.Content)
		visibility := normalizeVisibility(chunk.Visibility)
		if r.canUsePGVector(ctx, embeddings[i]) {
			_, err = database.ExecCtx(ctx, r.db,
				`INSERT INTO knowledge_chunks(source_type,source_id,visibility,title,summary,content,metadata_json,embedding_json,embedding_vector,content_hash,token_count,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::vector,$10,$11,'active')`,
				chunk.SourceType, chunk.SourceID, visibility, chunk.Title, chunk.Summary, chunk.Content, string(metadata), string(embedding), vectorLiteral(embeddings[i]), contentHash, len([]rune(chunk.Content)))
		} else {
			_, err = database.ExecCtx(ctx, r.db,
				`INSERT INTO knowledge_chunks(source_type,source_id,visibility,title,summary,content,metadata_json,embedding_json,content_hash,token_count,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'active')`,
				chunk.SourceType, chunk.SourceID, visibility, chunk.Title, chunk.Summary, chunk.Content, string(metadata), string(embedding), contentHash, len([]rune(chunk.Content)))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RAGService) ensureInitialIndex(ctx context.Context) error {
	if err := r.ensureResourceIndex(ctx); err != nil {
		return err
	}
	if err := r.ensureProjectIndex(ctx); err != nil {
		return err
	}
	if err := r.ensureTechStackIndex(ctx); err != nil {
		return err
	}
	if err := r.ensureTimelineIndex(ctx); err != nil {
		return err
	}
	return r.ensureDocumentIndex(ctx)
}

func (r *RAGService) ensureResourceIndex(ctx context.Context) error {
	if r.hasActiveChunks(ctx, knowledgeSourceResource) {
		return nil
	}
	rows, err := database.QueryCtx(ctx, r.db, siteResourceSelect()+` FROM site_resources WHERE status='published' ORDER BY is_featured DESC,sort_order ASC,id DESC LIMIT 80`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanSiteResource(rows)
		if err != nil {
			return err
		}
		if err := r.SyncSiteResource(ctx, &item); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func (r *RAGService) ensureProjectIndex(ctx context.Context) error {
	if r.hasActiveChunks(ctx, knowledgeSourceProject) {
		return nil
	}
	projects, _, err := (&AdminDataService{db: r.db}).ListSiteProjects(ctx, 1, 80, "published")
	if err != nil {
		return err
	}
	for i := range projects {
		if err := r.SyncSiteProject(ctx, &projects[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *RAGService) ensureTechStackIndex(ctx context.Context) error {
	if r.hasActiveChunks(ctx, knowledgeSourceTech) {
		return nil
	}
	techStacks, _, err := (&AdminDataService{db: r.db}).ListSiteTechStacks(ctx, 1, 80, "active")
	if err != nil {
		return err
	}
	for i := range techStacks {
		if err := r.SyncSiteTechStack(ctx, &techStacks[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *RAGService) ensureTimelineIndex(ctx context.Context) error {
	if r.hasActiveChunks(ctx, knowledgeSourceTimeline) {
		return nil
	}
	timeline, _, err := (&AdminDataService{db: r.db}).ListSiteTimelineEvents(ctx, 1, 120, "published")
	if err != nil {
		return err
	}
	for i := range timeline {
		if err := r.SyncSiteTimelineEvent(ctx, &timeline[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *RAGService) ensureDocumentIndex(ctx context.Context) error {
	if r.hasActiveChunks(ctx, knowledgeSourceDocument) {
		return nil
	}
	documentService := NewDocumentService(r.db)
	documents, _, err := documentService.List(ctx, 1, 100)
	if err != nil {
		return err
	}
	for i := range documents {
		if documents[i].Status != "active" {
			continue
		}
		full, err := documentService.Get(ctx, documents[i].ID, true)
		if err != nil {
			return err
		}
		if err := r.SyncUploadedDocument(ctx, full); err != nil {
			return err
		}
	}
	return nil
}

func (r *RAGService) hasActiveChunks(ctx context.Context, sourceType string) bool {
	var count int64
	err := database.QueryRowCtx(ctx, r.db, `SELECT COUNT(*) FROM knowledge_chunks WHERE status='active' AND source_type=$1`, sourceType).Scan(&count)
	return err == nil && count > 0
}

func (r *RAGService) search(ctx context.Context, question string, topK int, includeInternal bool) ([]knowledgeChunk, error) {
	queryEmbedding, err := r.ai.Embed(ctx, []string{question})
	if err != nil {
		return nil, err
	}
	if len(queryEmbedding) == 0 {
		return nil, nil
	}
	if r.canUsePGVector(ctx, queryEmbedding[0]) {
		if items, err := r.searchPGVector(ctx, question, queryEmbedding[0], topK, includeInternal); err == nil {
			return items, nil
		}
	}
	visibilitySQL := `visibility='public'`
	if includeInternal {
		visibilitySQL = `visibility IN ('public','internal')`
	}
	rows, err := database.QueryCtx(ctx, r.db, `SELECT id,source_type,source_id,visibility,title,summary,content,metadata_json,embedding_json FROM knowledge_chunks WHERE status='active' AND `+visibilitySQL+` ORDER BY updated_at DESC LIMIT 300`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	terms := tokenizeQuery(question)
	minScore := r.minScore()
	items := make([]knowledgeChunk, 0)
	for rows.Next() {
		var item knowledgeChunk
		var rawMetadata, rawEmbedding string
		if err := rows.Scan(&item.ID, &item.SourceType, &item.SourceID, &item.Visibility, &item.Title, &item.Summary, &item.Content, &rawMetadata, &rawEmbedding); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(rawMetadata), &item.Metadata)
		if json.Unmarshal([]byte(rawEmbedding), &item.Embedding) != nil || len(item.Embedding) == 0 {
			continue
		}
		item.VectorScore = cosine(queryEmbedding[0], item.Embedding)
		item.KeywordScore = keywordScore(item, terms)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items = r.rerank(question, items)
	filtered := items[:0]
	for _, item := range items {
		if item.Score >= minScore {
			filtered = append(filtered, item)
		}
	}
	items = filtered
	if topK <= 0 || topK > defaultRAGTopK*2 {
		topK = defaultRAGTopK
	}
	if len(items) > topK {
		items = items[:topK]
	}
	return items, nil
}

func (r *RAGService) searchPGVector(ctx context.Context, question string, embedding []float64, topK int, includeInternal bool) ([]knowledgeChunk, error) {
	if topK <= 0 || topK > defaultRAGTopK*2 {
		topK = defaultRAGTopK
	}
	visibilitySQL := `visibility='public'`
	if includeInternal {
		visibilitySQL = `visibility IN ('public','internal')`
	}
	rows, err := database.QueryCtx(ctx, r.db, `SELECT id,source_type,source_id,visibility,title,summary,content,metadata_json,embedding_json,(embedding_vector <=> $1::vector) AS distance FROM knowledge_chunks WHERE status='active' AND `+visibilitySQL+` AND embedding_vector IS NOT NULL ORDER BY embedding_vector <=> $1::vector LIMIT $2`, vectorLiteral(embedding), topK*6)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	terms := tokenizeQuery(question)
	minScore := r.minScore()
	items := make([]knowledgeChunk, 0)
	for rows.Next() {
		var item knowledgeChunk
		var rawMetadata, rawEmbedding string
		var distance float64
		if err := rows.Scan(&item.ID, &item.SourceType, &item.SourceID, &item.Visibility, &item.Title, &item.Summary, &item.Content, &rawMetadata, &rawEmbedding, &distance); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(rawMetadata), &item.Metadata)
		_ = json.Unmarshal([]byte(rawEmbedding), &item.Embedding)
		item.VectorScore = math.Max(0, 1-distance)
		item.KeywordScore = keywordScore(item, terms)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items = r.rerank(question, items)
	filtered := items[:0]
	for _, item := range items {
		if item.Score >= minScore {
			filtered = append(filtered, item)
		}
	}
	items = filtered
	if len(items) > topK {
		items = items[:topK]
	}
	return items, nil
}

func (r *RAGService) rerank(question string, items []knowledgeChunk) []knowledgeChunk {
	if len(items) == 0 {
		return items
	}
	terms := tokenizeQuery(question)
	bm25Scores := bm25Scores(terms, items)
	weights := r.sourceWeights()
	for i := range items {
		items[i].BM25Score = bm25Scores[i]
		items[i].SourceWeight = weights[items[i].SourceType]
		if items[i].SourceWeight <= 0 {
			items[i].SourceWeight = 1
		}
		base := items[i].VectorScore*0.58 + items[i].BM25Score*0.32 + items[i].KeywordScore*0.10
		titleBoost := 0.0
		if scoreText(strings.ToLower(items[i].Title), terms) > 0 {
			titleBoost = 0.04
		}
		items[i].RerankScore = math.Min((base+titleBoost)*items[i].SourceWeight, 1.5)
		items[i].Score = items[i].RerankScore
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score == items[j].Score {
			return items[i].UpdatedTitle() < items[j].UpdatedTitle()
		}
		return items[i].Score > items[j].Score
	})
	if n := r.rerankTopN(); n > 0 && len(items) > n {
		items = items[:n]
	}
	return items
}

func (c knowledgeChunk) UpdatedTitle() string {
	return strings.ToLower(c.SourceType + ":" + c.Title)
}

func keywordScore(item knowledgeChunk, terms []string) float64 {
	text := strings.ToLower(item.Title + " " + item.Summary + " " + item.Content)
	return math.Min(float64(scoreText(text, terms))/12, 1)
}

func bm25Scores(terms []string, items []knowledgeChunk) []float64 {
	out := make([]float64, len(items))
	if len(terms) == 0 || len(items) == 0 {
		return out
	}
	type docStats struct {
		tokens []string
		counts map[string]int
		length int
	}
	docs := make([]docStats, len(items))
	df := map[string]int{}
	totalLength := 0
	uniqueTerms := uniqueStrings(terms)
	for i, item := range items {
		tokens := tokenizeQuery(item.Title + " " + item.Summary + " " + item.Content)
		counts := map[string]int{}
		for _, token := range tokens {
			counts[token]++
		}
		docs[i] = docStats{tokens: tokens, counts: counts, length: len(tokens)}
		totalLength += len(tokens)
		for _, term := range uniqueTerms {
			if counts[term] > 0 {
				df[term]++
			}
		}
	}
	avgLength := float64(totalLength) / float64(len(items))
	if avgLength <= 0 {
		avgLength = 1
	}
	const k1 = 1.2
	const b = 0.75
	maxScore := 0.0
	for i, doc := range docs {
		score := 0.0
		for _, term := range uniqueTerms {
			tf := float64(doc.counts[term])
			if tf == 0 {
				continue
			}
			idf := math.Log(1 + (float64(len(items))-float64(df[term])+0.5)/(float64(df[term])+0.5))
			denom := tf + k1*(1-b+b*float64(doc.length)/avgLength)
			score += idf * (tf * (k1 + 1)) / denom
		}
		out[i] = score
		if score > maxScore {
			maxScore = score
		}
	}
	if maxScore > 0 {
		for i := range out {
			out[i] = out[i] / maxScore
		}
	}
	return out
}

func (r *RAGService) generateAnswer(ctx context.Context, question string, chunks []knowledgeChunk) (string, error) {
	return r.ai.Chat(ctx, r.answerChatRequest(question, chunks))
}

func (r *RAGService) answerChatRequest(question string, chunks []knowledgeChunk) ChatRequest {
	var contextText strings.Builder
	for i, chunk := range chunks {
		fmt.Fprintf(&contextText, "[%d] %s\n%s\n\n", i+1, chunk.Title, limitRunes(chunk.Content, 1200))
	}
	return ChatRequest{
		System: "你是这个站点的知识库助手。只能基于提供的资料回答；资料不足时要明确说明。回答使用中文，简洁、有条理，并在关键结论后标注来源编号，例如 [1]。",
		User:   "用户问题：\n" + question + "\n\n可用资料：\n" + contextText.String(),
	}
}

func (r *RAGService) extractiveAnswer(question string, chunks []knowledgeChunk) string {
	if len(chunks) == 0 {
		return "暂时没有在可检索内容里找到强相关资料。你可以换一个关键词，例如 React、Go、数据库或项目复盘。"
	}
	titles := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.Title != "" {
			titles = append(titles, chunk.Title)
		}
	}
	answer := "我从知识库里找到了这些相关资料：" + strings.Join(uniqueStrings(titles), "、") + "。"
	if chunks[0].Summary != "" {
		answer += " 最相关的是《" + chunks[0].Title + "》：" + limitRunes(chunks[0].Summary, 180)
	}
	if question != "" && !r.chatEnabled(context.Background()) {
		answer += " 配置大模型后，我可以基于这些资料生成更完整的自然语言答案。"
	}
	return answer
}

func (r *RAGService) chatEnabled(ctx context.Context) bool {
	if r.ai.ChatEnabled() {
		return true
	}
	cfg, err := (&AdminDataService{db: r.db}).ActiveAIModelConfig(ctx)
	return err == nil && cfg != nil && strings.TrimSpace(cfg.ChatModel) != ""
}

func (r *RAGService) sourcesToLegacyMatches(ctx context.Context, chunks []knowledgeChunk) ([]models.SiteResource, []models.SiteProject) {
	resourceIDs := make([]int64, 0)
	projectIDs := make([]int64, 0)
	seenResources := map[int64]bool{}
	seenProjects := map[int64]bool{}
	for _, chunk := range chunks {
		if chunk.SourceType == knowledgeSourceResource && !seenResources[chunk.SourceID] {
			seenResources[chunk.SourceID] = true
			resourceIDs = append(resourceIDs, chunk.SourceID)
		}
		if chunk.SourceType == knowledgeSourceProject && !seenProjects[chunk.SourceID] {
			seenProjects[chunk.SourceID] = true
			projectIDs = append(projectIDs, chunk.SourceID)
		}
	}
	resources := make([]models.SiteResource, 0, len(resourceIDs))
	service := &AdminDataService{db: r.db}
	for _, id := range resourceIDs {
		if item, err := service.GetSiteResource(ctx, id); err == nil && item != nil {
			resources = append(resources, *item)
		}
	}
	projects := make([]models.SiteProject, 0, len(projectIDs))
	for _, id := range projectIDs {
		if item, err := service.GetSiteProject(ctx, id); err == nil && item != nil {
			projects = append(projects, *item)
		}
	}
	return resources, projects
}

func buildSiteResourceChunks(item models.SiteResource) []knowledgeChunkInput {
	text := strings.TrimSpace(strings.Join([]string{item.Summary, item.Content, item.MarkdownContent}, "\n\n"))
	if text == "" {
		text = item.Title
	}
	parts := splitText(text, 900, 180)
	out := make([]knowledgeChunkInput, 0, len(parts))
	for i, part := range parts {
		title := item.Title
		if len(parts) > 1 {
			title = fmt.Sprintf("%s #%d", item.Title, i+1)
		}
		out = append(out, knowledgeChunkInput{
			SourceType: knowledgeSourceResource,
			SourceID:   item.ID,
			Visibility: knowledgeVisibilityPublic,
			Title:      title,
			Summary:    item.Summary,
			Content:    strings.TrimSpace(strings.Join([]string{item.Title, item.Category, item.Tags, part}, "\n")),
			Metadata: map[string]any{
				"slug":     item.Slug,
				"category": item.Category,
				"tags":     item.Tags,
			},
		})
	}
	return out
}

func buildSiteProjectChunks(item models.SiteProject) []knowledgeChunkInput {
	text := strings.TrimSpace(strings.Join([]string{item.Summary, item.Description, item.StackTags}, "\n\n"))
	if text == "" {
		text = item.Name
	}
	return []knowledgeChunkInput{{
		SourceType: knowledgeSourceProject,
		SourceID:   item.ID,
		Visibility: knowledgeVisibilityPublic,
		Title:      item.Name,
		Summary:    item.Summary,
		Content: strings.TrimSpace(strings.Join([]string{
			item.Name,
			item.Summary,
			item.Description,
			"Tech stacks: " + item.StackTags,
			"Demo: " + item.DemoURL,
			"Repo: " + item.RepoURL,
		}, "\n")),
		Metadata: map[string]any{
			"stack_tags": item.StackTags,
			"demo_url":   item.DemoURL,
			"repo_url":   item.RepoURL,
		},
	}}
}

func buildSiteTechStackChunks(item models.SiteTechStack) []knowledgeChunkInput {
	return []knowledgeChunkInput{{
		SourceType: knowledgeSourceTech,
		SourceID:   item.ID,
		Visibility: knowledgeVisibilityPublic,
		Title:      item.Name,
		Summary:    item.Description,
		Content: strings.TrimSpace(strings.Join([]string{
			item.Name,
			"Category: " + item.Category,
			fmt.Sprintf("Level: %d", item.Level),
			item.Description,
		}, "\n")),
		Metadata: map[string]any{
			"category": item.Category,
			"level":    item.Level,
		},
	}}
}

func buildSiteTimelineChunks(item models.SiteTimelineEvent) []knowledgeChunkInput {
	text := strings.TrimSpace(strings.Join([]string{item.Summary, item.Content, item.Phase, item.EventType, item.Tags}, "\n\n"))
	if text == "" {
		text = item.Title
	}
	parts := splitText(text, 900, 180)
	out := make([]knowledgeChunkInput, 0, len(parts))
	for i, part := range parts {
		title := item.Title
		if len(parts) > 1 {
			title = fmt.Sprintf("%s #%d", item.Title, i+1)
		}
		out = append(out, knowledgeChunkInput{
			SourceType: knowledgeSourceTimeline,
			SourceID:   item.ID,
			Visibility: knowledgeVisibilityPublic,
			Title:      title,
			Summary:    item.Summary,
			Content: strings.TrimSpace(strings.Join([]string{
				item.Title,
				item.Summary,
				item.Content,
				"Phase: " + item.Phase,
				"Type: " + item.EventType,
				"Tags: " + item.Tags,
				part,
			}, "\n")),
			Metadata: map[string]any{
				"phase":      item.Phase,
				"event_type": item.EventType,
				"tags":       item.Tags,
				"link_url":   item.LinkURL,
			},
		})
	}
	return out
}

func buildUploadedDocumentChunks(item models.UploadedDocument) []knowledgeChunkInput {
	text := strings.TrimSpace(item.TextContent)
	if text == "" {
		text = item.OriginalName
	}
	parts := splitText(text, 900, 180)
	out := make([]knowledgeChunkInput, 0, len(parts))
	visibility := normalizeVisibility(item.Visibility)
	for i, part := range parts {
		title := item.OriginalName
		if len(parts) > 1 {
			title = fmt.Sprintf("%s #%d", item.OriginalName, i+1)
		}
		out = append(out, knowledgeChunkInput{
			SourceType: knowledgeSourceDocument,
			SourceID:   item.ID,
			Visibility: visibility,
			Title:      title,
			Summary:    fmt.Sprintf("%s · %d bytes", item.MimeType, item.FileSize),
			Content: strings.TrimSpace(strings.Join([]string{
				item.OriginalName,
				item.MimeType,
				part,
			}, "\n")),
			Metadata: map[string]any{
				"file_path":   item.FilePath,
				"mime_type":   item.MimeType,
				"visibility":  visibility,
				"chunk_index": i + 1,
				"chunk_count": len(parts),
			},
		})
	}
	return out
}

func splitText(text string, size, overlap int) []string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return nil
	}
	if size <= 0 {
		size = 900
	}
	if overlap < 0 || overlap >= size {
		overlap = 0
	}
	out := make([]string, 0, len(runes)/size+1)
	for start := 0; start < len(runes); {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[start:end]))
		if end == len(runes) {
			break
		}
		start = end - overlap
	}
	return out
}

func chunksToSources(chunks []knowledgeChunk, question string) []models.KnowledgeSource {
	out := make([]models.KnowledgeSource, 0, len(chunks))
	terms := tokenizeQuery(question)
	minScore := envFloat("RAG_MIN_SCORE", defaultRAGMinScore)
	for index, chunk := range chunks {
		out = append(out, models.KnowledgeSource{
			ChunkID:         chunk.ID,
			CitationID:      index + 1,
			SourceType:      chunk.SourceType,
			SourceID:        chunk.SourceID,
			Visibility:      normalizeVisibility(chunk.Visibility),
			Title:           chunk.Title,
			Summary:         chunk.Summary,
			Score:           math.Round(chunk.Score*10000) / 10000,
			VectorScore:     math.Round(chunk.VectorScore*10000) / 10000,
			BM25Score:       math.Round(chunk.BM25Score*10000) / 10000,
			KeywordScore:    math.Round(chunk.KeywordScore*10000) / 10000,
			SourceWeight:    math.Round(chunk.SourceWeight*10000) / 10000,
			RerankScore:     math.Round(chunk.RerankScore*10000) / 10000,
			Threshold:       math.Round(minScore*10000) / 10000,
			URL:             sourceURL(chunk),
			Snippet:         buildSnippet(chunk.Content, terms, 180),
			HighlightedText: highlightSnippet(buildSnippet(chunk.Content, terms, 180), terms),
		})
	}
	return out
}

func (r *RAGService) createJob(ctx context.Context, jobType string) (*models.RAGIndexJob, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	id, err := database.InsertID(tx, `INSERT INTO rag_index_jobs(job_type,status,max_retries) VALUES($1,'pending',3) RETURNING id`, jobType)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetJob(ctx, id)
}

func (r *RAGService) processJob(ctx context.Context, id int64) {
	job, err := r.GetJob(ctx, id)
	if err != nil || job == nil {
		return
	}
	for attempt := job.RetryCount; attempt <= job.MaxRetries; attempt++ {
		_, _ = database.ExecCtx(ctx, r.db, `UPDATE rag_index_jobs SET status='running',retry_count=$1,error_message='',started_at=`+database.Now()+`,updated_at=`+database.Now()+` WHERE id=$2`, attempt, id)
		_, err = r.Rebuild(ctx)
		if err == nil {
			_, _ = database.ExecCtx(ctx, r.db, `UPDATE rag_index_jobs SET status='success',error_message='',finished_at=`+database.Now()+`,updated_at=`+database.Now()+` WHERE id=$1`, id)
			return
		}
		if attempt >= job.MaxRetries {
			break
		}
		_, _ = database.ExecCtx(ctx, r.db, `UPDATE rag_index_jobs SET status='retrying',error_message=$1,updated_at=`+database.Now()+` WHERE id=$2`, err.Error(), id)
		time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
	}
	message := ""
	if err != nil {
		message = err.Error()
	}
	_, _ = database.ExecCtx(ctx, r.db, `UPDATE rag_index_jobs SET status='failed',error_message=$1,finished_at=`+database.Now()+`,updated_at=`+database.Now()+` WHERE id=$2`, limitRunes(message, 1000), id)
}

type ragJobScanner interface {
	Scan(dest ...any) error
}

func scanRAGIndexJob(scanner ragJobScanner) (models.RAGIndexJob, error) {
	var job models.RAGIndexJob
	err := scanner.Scan(&job.ID, &job.JobType, &job.Status, &job.RetryCount, &job.MaxRetries, &job.ErrorMessage, &job.StartedAt, &job.FinishedAt, &job.CreatedAt, &job.UpdatedAt)
	return job, err
}

func (r *RAGService) logQuery(ctx context.Context, question, answer string, sources []models.KnowledgeSource, matched bool, latency time.Duration, usedChat bool) int64 {
	topScore := 0.0
	if len(sources) > 0 {
		topScore = sources[0].Score
	}
	rawSources, _ := json.Marshal(sources)
	query := `INSERT INTO rag_query_logs(question,answer,matched,source_count,top_score,latency_ms,used_chat_model,source_json) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`
	args := []any{limitRunes(question, 2000), limitRunes(answer, 4000), matched, len(sources), topScore, latency.Milliseconds(), usedChat, string(rawSources)}
	if database.CurrentDialect != nil && database.CurrentDialect.SupportsReturning() {
		var id int64
		if err := database.QueryRowCtx(ctx, r.db, query, args...).Scan(&id); err == nil {
			return id
		}
		return 0
	}
	result, err := database.ExecCtx(ctx, r.db, strings.Replace(query, " RETURNING id", "", 1), args...)
	if err != nil {
		return 0
	}
	id, _ := result.LastInsertId()
	return id
}

func (r *RAGService) suggestQuestions(question string, chunks []knowledgeChunk) []string {
	base := []string{}
	seen := map[string]bool{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		base = append(base, value)
	}
	if len(chunks) > 0 {
		title := strings.TrimSpace(strings.TrimSuffix(chunks[0].Title, " #1"))
		add(fmt.Sprintf("能展开讲讲「%s」的关键点吗？", title))
		if chunks[0].SourceType == knowledgeSourceResource {
			add(fmt.Sprintf("「%s」里有没有代码示例？", title))
		}
		if chunks[0].SourceType == knowledgeSourceTimeline {
			add(fmt.Sprintf("这条时间线之后还发生了什么？"))
		}
	}
	terms := uniqueStrings(tokenizeQuery(question))
	if len(terms) > 0 {
		add(fmt.Sprintf("%s 在项目中有哪些实践经验？", terms[0]))
	}
	add("这些内容可以按学习路线总结一下吗？")
	add("有没有相关项目或文章可以继续看？")
	if len(base) > 4 {
		return base[:4]
	}
	return base
}

func loadRAGEvalCases() []models.RAGEvalCase {
	raw := strings.TrimSpace(os.Getenv("RAG_EVAL_CASES_JSON"))
	if raw != "" {
		var cases []models.RAGEvalCase
		if json.Unmarshal([]byte(raw), &cases) == nil && len(cases) > 0 {
			return cases
		}
	}
	return []models.RAGEvalCase{
		{ID: "site-stack", Question: "这个项目使用了哪些前后端技术？", ExpectedSources: []string{knowledgeSourceTech, knowledgeSourceProject}, ExpectedTerms: []string{"Go", "Vue"}},
		{ID: "site-projects", Question: "有哪些可以展示的项目经验？", ExpectedSources: []string{knowledgeSourceProject}, ExpectedTerms: []string{"项目"}},
		{ID: "learning-notes", Question: "有没有学习笔记或文章可以继续看？", ExpectedSources: []string{knowledgeSourceResource}, ExpectedTerms: []string{"学习", "文章"}},
		{ID: "timeline", Question: "最近的成长时间线有哪些关键节点？", ExpectedSources: []string{knowledgeSourceTimeline}, ExpectedTerms: []string{"时间线", "阶段"}},
	}
}

func evalRecallHit(item models.RAGEvalCase, sources []models.KnowledgeSource) bool {
	if len(item.ExpectedSources) == 0 {
		return len(sources) > 0
	}
	expected := map[string]bool{}
	for _, source := range item.ExpectedSources {
		expected[strings.TrimSpace(source)] = true
	}
	for _, source := range sources {
		key := source.SourceType
		if expected[key] || expected[fmt.Sprintf("%s:%d", source.SourceType, source.SourceID)] {
			return true
		}
	}
	return false
}

func evalAnswerQuality(item models.RAGEvalCase, answer string) float64 {
	answer = strings.ToLower(answer)
	terms := uniqueStrings(item.ExpectedTerms)
	if len(terms) == 0 {
		if strings.TrimSpace(answer) == "" {
			return 0
		}
		return 1
	}
	hits := 0
	for _, term := range terms {
		if strings.Contains(answer, strings.ToLower(term)) {
			hits++
		}
	}
	return math.Round(float64(hits)/float64(len(terms))*10000) / 10000
}

func sourceURL(chunk knowledgeChunk) string {
	switch chunk.SourceType {
	case knowledgeSourceResource:
		if slug, ok := chunk.Metadata["slug"].(string); ok && strings.TrimSpace(slug) != "" {
			return "/#/articles/" + strings.TrimSpace(slug)
		}
		return "/#resources"
	case knowledgeSourceProject:
		if demoURL, ok := chunk.Metadata["demo_url"].(string); ok && strings.TrimSpace(demoURL) != "" {
			return strings.TrimSpace(demoURL)
		}
		if repoURL, ok := chunk.Metadata["repo_url"].(string); ok && strings.TrimSpace(repoURL) != "" {
			return strings.TrimSpace(repoURL)
		}
		return "/#demos"
	case knowledgeSourceTech:
		return "/#stack"
	case knowledgeSourceTimeline:
		if linkURL, ok := chunk.Metadata["link_url"].(string); ok && strings.TrimSpace(linkURL) != "" {
			return strings.TrimSpace(linkURL)
		}
		return "/#timeline"
	case knowledgeSourceDocument:
		return ""
	default:
		return ""
	}
}

func buildSnippet(content string, terms []string, max int) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	lower := strings.ToLower(content)
	pos := -1
	for _, term := range terms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" {
			continue
		}
		if idx := strings.Index(lower, term); idx >= 0 && (pos == -1 || idx < pos) {
			pos = idx
		}
	}
	runes := []rune(content)
	if pos < 0 {
		return limitRunes(content, max)
	}
	startRune := len([]rune(content[:pos])) - max/3
	if startRune < 0 {
		startRune = 0
	}
	endRune := startRune + max
	if endRune > len(runes) {
		endRune = len(runes)
	}
	prefix := ""
	if startRune > 0 {
		prefix = "..."
	}
	suffix := ""
	if endRune < len(runes) {
		suffix = "..."
	}
	return prefix + string(runes[startRune:endRune]) + suffix
}

func highlightSnippet(snippet string, terms []string) string {
	out := html.EscapeString(snippet)
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" || len([]rune(term)) < 2 {
			continue
		}
		escapedTerm := html.EscapeString(term)
		out = replaceFold(out, escapedTerm, "<mark>"+escapedTerm+"</mark>")
	}
	return out
}

func replaceFold(value, needle, replacement string) string {
	lowerValue := strings.ToLower(value)
	lowerNeedle := strings.ToLower(needle)
	idx := strings.Index(lowerValue, lowerNeedle)
	if idx < 0 {
		return value
	}
	return value[:idx] + replacement + value[idx+len(needle):]
}

func (r *RAGService) vectorBackend() string {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("RAG_VECTOR_BACKEND")))
	if value == "" {
		if r.pgVectorAvailable(context.Background()) {
			return "pgvector"
		}
		return "json"
	}
	return value
}

func (r *RAGService) pgVectorAvailable(ctx context.Context) bool {
	if database.CurrentDialect == nil || database.CurrentDialect.Type != database.DBTypePostgres {
		return false
	}
	var count int
	err := database.QueryRowCtx(ctx, r.db, `SELECT COUNT(*) FROM pg_extension WHERE extname='vector'`).Scan(&count)
	return err == nil && count > 0
}

func (r *RAGService) canUsePGVector(ctx context.Context, embedding []float64) bool {
	if len(embedding) == 0 || len(embedding) != r.vectorDim() {
		return false
	}
	backend := strings.ToLower(strings.TrimSpace(os.Getenv("RAG_VECTOR_BACKEND")))
	if backend != "" && backend != "pgvector" {
		return false
	}
	return r.pgVectorAvailable(ctx)
}

func (r *RAGService) vectorDim() int {
	return envInt("RAG_VECTOR_DIM", 256)
}

func (r *RAGService) minScore() float64 {
	return envFloat("RAG_MIN_SCORE", defaultRAGMinScore)
}

func (r *RAGService) rerankTopN() int {
	value := envInt("RAG_RERANK_TOP_N", r.top*6)
	if value < r.top {
		return r.top
	}
	return value
}

func (r *RAGService) sourceWeights() map[string]float64 {
	return map[string]float64{
		knowledgeSourceResource: envFloat("RAG_WEIGHT_RESOURCE", 1),
		knowledgeSourceProject:  envFloat("RAG_WEIGHT_PROJECT", 1),
		knowledgeSourceTech:     envFloat("RAG_WEIGHT_TECH", 1),
		knowledgeSourceTimeline: envFloat("RAG_WEIGHT_TIMELINE", 1),
		knowledgeSourceDocument: envFloat("RAG_WEIGHT_DOCUMENT", 1),
	}
}

func (r *RAGService) streamingEnabled(ctx context.Context) bool {
	client := r.ai
	if dbClient, ok := client.(*dbBackedAIClient); ok {
		active, err := dbClient.activeClient(ctx)
		return err == nil && supportsChatStream(active)
	}
	return supportsChatStream(client)
}

func normalizeVisibility(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case knowledgeVisibilityPublic, knowledgeVisibilityInternal:
		return value
	default:
		return knowledgeVisibilityInternal
	}
}

func vectorLiteral(values []float64) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.FormatFloat(value, 'f', -1, 64))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, an, bn float64
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		an += a[i] * a[i]
		bn += b[i] * b[i]
	}
	if an == 0 || bn == 0 {
		return 0
	}
	return dot / (math.Sqrt(an) * math.Sqrt(bn))
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(text)))
	return hex.EncodeToString(sum[:])
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func limitRunes(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if max <= 0 || len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "..."
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

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func envFloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}
