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
	knowledgeSourceResource = "site_resource"
	knowledgeSourceProject  = "site_project"
	knowledgeSourceTech     = "site_tech_stack"
	knowledgeSourceTimeline = "site_timeline"
	knowledgeSourceDocument = "uploaded_document"
	defaultRAGTopK          = 6
)

type RAGService struct {
	db  *sql.DB
	ai  AIClient
	top int
}

type knowledgeChunkInput struct {
	SourceType string
	SourceID   int64
	Title      string
	Summary    string
	Content    string
	Metadata   map[string]any
}

type knowledgeChunk struct {
	ID         int64
	SourceType string
	SourceID   int64
	Title      string
	Summary    string
	Content    string
	Metadata   map[string]any
	Embedding  []float64
	Score      float64
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
		TopK:              r.top,
		ChatEnabled:       r.chatEnabled(ctx),
		VectorBackend:     r.vectorBackend(),
		PGVectorAvailable: r.pgVectorAvailable(ctx),
	}
	rows, err := database.QueryCtx(ctx, r.db, `SELECT source_type,COUNT(*) FROM knowledge_chunks WHERE status='active' GROUP BY source_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sourceType string
		var count int64
		if err := rows.Scan(&sourceType, &count); err != nil {
			return nil, err
		}
		stats.BySource[sourceType] = count
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

func (r *RAGService) AskSiteKnowledge(ctx context.Context, question string, fallback func(context.Context, string) (*models.SiteKnowledgeAnswer, error)) (*models.SiteKnowledgeAnswer, error) {
	start := time.Now()
	question = strings.TrimSpace(question)
	if question == "" {
		return &models.SiteKnowledgeAnswer{Question: question, Answer: "可以问我关于 React、Go、数据库、项目经验或学习笔记的问题。"}, nil
	}

	_ = r.ensureInitialIndex(ctx)
	matches, err := r.search(ctx, question, r.top)
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

func (r *RAGService) AskAdminKnowledge(ctx context.Context, question string) (*models.AIAssistantResult, bool, error) {
	start := time.Now()
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, false, nil
	}
	_ = r.ensureInitialIndex(ctx)
	matches, err := r.search(ctx, question, r.top)
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
		if r.canUsePGVector(ctx, embeddings[i]) {
			_, err = database.ExecCtx(ctx, r.db,
				`INSERT INTO knowledge_chunks(source_type,source_id,title,summary,content,metadata_json,embedding_json,embedding_vector,content_hash,token_count,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8::vector,$9,$10,'active')`,
				chunk.SourceType, chunk.SourceID, chunk.Title, chunk.Summary, chunk.Content, string(metadata), string(embedding), vectorLiteral(embeddings[i]), contentHash, len([]rune(chunk.Content)))
		} else {
			_, err = database.ExecCtx(ctx, r.db,
				`INSERT INTO knowledge_chunks(source_type,source_id,title,summary,content,metadata_json,embedding_json,content_hash,token_count,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'active')`,
				chunk.SourceType, chunk.SourceID, chunk.Title, chunk.Summary, chunk.Content, string(metadata), string(embedding), contentHash, len([]rune(chunk.Content)))
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

func (r *RAGService) search(ctx context.Context, question string, topK int) ([]knowledgeChunk, error) {
	queryEmbedding, err := r.ai.Embed(ctx, []string{question})
	if err != nil {
		return nil, err
	}
	if len(queryEmbedding) == 0 {
		return nil, nil
	}
	if r.canUsePGVector(ctx, queryEmbedding[0]) {
		if items, err := r.searchPGVector(ctx, question, queryEmbedding[0], topK); err == nil {
			return items, nil
		}
	}
	rows, err := database.QueryCtx(ctx, r.db, `SELECT id,source_type,source_id,title,summary,content,metadata_json,embedding_json FROM knowledge_chunks WHERE status='active' ORDER BY updated_at DESC LIMIT 300`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	terms := tokenizeQuery(question)
	items := make([]knowledgeChunk, 0)
	for rows.Next() {
		var item knowledgeChunk
		var rawMetadata, rawEmbedding string
		if err := rows.Scan(&item.ID, &item.SourceType, &item.SourceID, &item.Title, &item.Summary, &item.Content, &rawMetadata, &rawEmbedding); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(rawMetadata), &item.Metadata)
		if json.Unmarshal([]byte(rawEmbedding), &item.Embedding) != nil || len(item.Embedding) == 0 {
			continue
		}
		vectorScore := cosine(queryEmbedding[0], item.Embedding)
		keywordScore := math.Min(float64(scoreText(strings.ToLower(item.Title+" "+item.Summary+" "+item.Content), terms))/12, 1)
		item.Score = vectorScore*0.78 + keywordScore*0.22
		if item.Score > 0 {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	if topK <= 0 || topK > defaultRAGTopK*2 {
		topK = defaultRAGTopK
	}
	if len(items) > topK {
		items = items[:topK]
	}
	return items, rows.Err()
}

func (r *RAGService) searchPGVector(ctx context.Context, question string, embedding []float64, topK int) ([]knowledgeChunk, error) {
	if topK <= 0 || topK > defaultRAGTopK*2 {
		topK = defaultRAGTopK
	}
	rows, err := database.QueryCtx(ctx, r.db, `SELECT id,source_type,source_id,title,summary,content,metadata_json,embedding_json,(embedding_vector <=> $1::vector) AS distance FROM knowledge_chunks WHERE status='active' AND embedding_vector IS NOT NULL ORDER BY embedding_vector <=> $1::vector LIMIT $2`, vectorLiteral(embedding), topK*6)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	terms := tokenizeQuery(question)
	items := make([]knowledgeChunk, 0)
	for rows.Next() {
		var item knowledgeChunk
		var rawMetadata, rawEmbedding string
		var distance float64
		if err := rows.Scan(&item.ID, &item.SourceType, &item.SourceID, &item.Title, &item.Summary, &item.Content, &rawMetadata, &rawEmbedding, &distance); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(rawMetadata), &item.Metadata)
		_ = json.Unmarshal([]byte(rawEmbedding), &item.Embedding)
		vectorScore := math.Max(0, 1-distance)
		keywordScore := math.Min(float64(scoreText(strings.ToLower(item.Title+" "+item.Summary+" "+item.Content), terms))/12, 1)
		item.Score = vectorScore*0.82 + keywordScore*0.18
		if item.Score > 0 {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	if len(items) > topK {
		items = items[:topK]
	}
	return items, nil
}

func (r *RAGService) generateAnswer(ctx context.Context, question string, chunks []knowledgeChunk) (string, error) {
	var contextText strings.Builder
	for i, chunk := range chunks {
		fmt.Fprintf(&contextText, "[%d] %s\n%s\n\n", i+1, chunk.Title, limitRunes(chunk.Content, 1200))
	}
	return r.ai.Chat(ctx, ChatRequest{
		System: "你是这个站点的知识库助手。只能基于提供的资料回答；资料不足时要明确说明。回答使用中文，简洁、有条理，并在关键结论后标注来源编号，如 [1]。",
		User:   "用户问题：\n" + question + "\n\n可用资料：\n" + contextText.String(),
	})
}

func (r *RAGService) extractiveAnswer(question string, chunks []knowledgeChunk) string {
	if len(chunks) == 0 {
		return "暂时没有在已发布内容里找到强相关资料。你可以换一个关键词，比如 React、Go、数据库或项目复盘。"
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
	for i, part := range parts {
		title := item.OriginalName
		if len(parts) > 1 {
			title = fmt.Sprintf("%s #%d", item.OriginalName, i+1)
		}
		out = append(out, knowledgeChunkInput{
			SourceType: knowledgeSourceDocument,
			SourceID:   item.ID,
			Title:      title,
			Summary:    fmt.Sprintf("%s · %d bytes", item.MimeType, item.FileSize),
			Content: strings.TrimSpace(strings.Join([]string{
				item.OriginalName,
				item.MimeType,
				part,
			}, "\n")),
			Metadata: map[string]any{
				"file_path": item.FilePath,
				"mime_type": item.MimeType,
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
	for _, chunk := range chunks {
		out = append(out, models.KnowledgeSource{
			SourceType:      chunk.SourceType,
			SourceID:        chunk.SourceID,
			Title:           chunk.Title,
			Summary:         chunk.Summary,
			Score:           math.Round(chunk.Score*10000) / 10000,
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
