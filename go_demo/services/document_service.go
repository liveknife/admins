package services

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-demo/database"
	"go-demo/models"
)

type DocumentService struct {
	db  *sql.DB
	rag *RAGService
}

func NewDocumentService(db *sql.DB) *DocumentService {
	return &DocumentService{db: db, rag: NewRAGService(db)}
}

func (s *DocumentService) Upload(ctx context.Context, file *multipart.FileHeader) (*models.UploadedDocument, error) {
	if file == nil {
		return nil, fmt.Errorf("missing file")
	}
	if file.Size <= 0 || file.Size > 20<<20 {
		return nil, fmt.Errorf("file size must be between 1 byte and 20MB")
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".txt" && ext != ".md" && ext != ".markdown" && ext != ".pdf" {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	raw, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	mimeType := file.Header.Get("Content-Type")
	if strings.TrimSpace(mimeType) == "" {
		mimeType = http.DetectContentType(raw)
	}
	dateDir := time.Now().Format("20060102")
	dir := filepath.Join("uploads", "documents", dateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	safeName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	path := filepath.Join(dir, safeName)
	if err := os.WriteFile(path, raw, 0644); err != nil {
		return nil, err
	}

	text, parseErr := extractDocumentText(raw, ext)
	status := "active"
	errorMessage := ""
	if parseErr != nil {
		status = "failed"
		errorMessage = parseErr.Error()
	}
	if strings.TrimSpace(text) == "" && parseErr == nil {
		status = "failed"
		errorMessage = "no text content extracted"
	}

	id, err := s.insertDocument(ctx, file.Filename, safeName, filepath.ToSlash(path), mimeType, file.Size, text, status, errorMessage)
	if err != nil {
		return nil, err
	}
	item, err := s.Get(ctx, id, true)
	if err != nil {
		return nil, err
	}
	if status == "active" {
		if err := s.rag.SyncUploadedDocument(ctx, item); err != nil {
			_ = s.markFailed(ctx, id, err.Error())
			item, _ = s.Get(ctx, id, true)
		} else {
			count := len(buildUploadedDocumentChunks(*item))
			_, _ = database.ExecCtx(ctx, s.db, `UPDATE uploaded_documents SET chunk_count=$1,updated_at=`+database.Now()+` WHERE id=$2`, count, id)
			item.ChunkCount = count
		}
	}
	return item, nil
}

func (s *DocumentService) insertDocument(ctx context.Context, originalName, fileName, filePath, mimeType string, size int64, text, status, errorMessage string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	id, err := database.InsertID(tx, `INSERT INTO uploaded_documents(original_name,file_name,file_path,mime_type,file_size,text_content,status,error_message) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		originalName, fileName, filePath, mimeType, size, text, status, errorMessage)
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (s *DocumentService) List(ctx context.Context, page, pageSize int) ([]models.UploadedDocument, int64, error) {
	limit, offset := normalizePagination(page, pageSize)
	var total int64
	if err := database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM uploaded_documents`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := database.QueryCtx(ctx, s.db, `SELECT id,original_name,file_name,file_path,mime_type,file_size,'' AS text_content,chunk_count,status,error_message,created_at,updated_at FROM uploaded_documents ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]models.UploadedDocument, 0)
	for rows.Next() {
		item, err := scanUploadedDocument(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *DocumentService) Get(ctx context.Context, id int64, includeText bool) (*models.UploadedDocument, error) {
	textExpr := "'' AS text_content"
	if includeText {
		textExpr = "text_content"
	}
	row := database.QueryRowCtx(ctx, s.db, `SELECT id,original_name,file_name,file_path,mime_type,file_size,`+textExpr+`,chunk_count,status,error_message,created_at,updated_at FROM uploaded_documents WHERE id=$1`, id)
	item, err := scanUploadedDocument(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *DocumentService) Preview(ctx context.Context, id int64) (*models.UploadedDocument, error) {
	return s.Get(ctx, id, true)
}

func (s *DocumentService) Delete(ctx context.Context, id int64) (int64, error) {
	item, err := s.Get(ctx, id, false)
	if err != nil {
		return 0, err
	}
	var chunks int64
	_ = database.QueryRowCtx(ctx, s.db, `SELECT COUNT(*) FROM knowledge_chunks WHERE source_type=$1 AND source_id=$2`, knowledgeSourceDocument, id).Scan(&chunks)
	if err := s.rag.DeleteSource(ctx, knowledgeSourceDocument, id); err != nil {
		return 0, err
	}
	if _, err := database.ExecCtx(ctx, s.db, `DELETE FROM uploaded_documents WHERE id=$1`, id); err != nil {
		return 0, err
	}
	if item.FilePath != "" {
		_ = os.Remove(filepath.FromSlash(item.FilePath))
	}
	return chunks, nil
}

func (s *DocumentService) Rebuild(ctx context.Context, id int64) (*models.UploadedDocument, error) {
	item, err := s.Get(ctx, id, true)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(item.TextContent) == "" {
		raw, readErr := os.ReadFile(filepath.FromSlash(item.FilePath))
		if readErr != nil {
			_ = s.markFailed(ctx, id, readErr.Error())
			return s.Get(ctx, id, true)
		}
		text, parseErr := extractDocumentText(raw, strings.ToLower(filepath.Ext(item.OriginalName)))
		if parseErr != nil {
			_ = s.markFailed(ctx, id, parseErr.Error())
			return s.Get(ctx, id, true)
		}
		item.TextContent = text
		_, _ = database.ExecCtx(ctx, s.db, `UPDATE uploaded_documents SET text_content=$1,status='active',error_message='',updated_at=`+database.Now()+` WHERE id=$2`, text, id)
	}
	if err := s.rag.SyncUploadedDocument(ctx, item); err != nil {
		_ = s.markFailed(ctx, id, err.Error())
		return s.Get(ctx, id, true)
	}
	count := len(buildUploadedDocumentChunks(*item))
	_, _ = database.ExecCtx(ctx, s.db, `UPDATE uploaded_documents SET chunk_count=$1,status='active',error_message='',updated_at=`+database.Now()+` WHERE id=$2`, count, id)
	return s.Get(ctx, id, true)
}

func (s *DocumentService) markFailed(ctx context.Context, id int64, message string) error {
	_, err := database.ExecCtx(ctx, s.db, `UPDATE uploaded_documents SET status='failed',error_message=$1,updated_at=`+database.Now()+` WHERE id=$2`, limitRunes(message, 1000), id)
	return err
}

type uploadedDocumentScanner interface {
	Scan(dest ...any) error
}

func scanUploadedDocument(scanner uploadedDocumentScanner) (models.UploadedDocument, error) {
	var item models.UploadedDocument
	err := scanner.Scan(&item.ID, &item.OriginalName, &item.FileName, &item.FilePath, &item.MimeType, &item.FileSize, &item.TextContent, &item.ChunkCount, &item.Status, &item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func extractDocumentText(raw []byte, ext string) (string, error) {
	switch ext {
	case ".txt", ".md", ".markdown":
		return strings.TrimSpace(string(bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF}))), nil
	case ".pdf":
		text := extractPDFTextLoose(raw)
		if strings.TrimSpace(text) == "" {
			return "", fmt.Errorf("PDF text extraction failed; please upload a text-based PDF or MD/TXT version")
		}
		return text, nil
	default:
		return "", fmt.Errorf("unsupported file type")
	}
}

func extractPDFTextLoose(raw []byte) string {
	text := string(raw)
	var out []string
	for {
		start := strings.Index(text, "(")
		if start < 0 {
			break
		}
		text = text[start+1:]
		end := strings.Index(text, ")")
		if end < 0 {
			break
		}
		part := strings.TrimSpace(strings.ReplaceAll(text[:end], `\)`, ")"))
		part = strings.ReplaceAll(part, `\(`, "(")
		if len([]rune(part)) >= 3 && printableRatio(part) > 0.75 {
			out = append(out, part)
		}
		text = text[end+1:]
		if len(strings.Join(out, " ")) > 20000 {
			break
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func printableRatio(value string) float64 {
	if value == "" {
		return 0
	}
	total, printable := 0, 0
	for _, r := range value {
		total++
		if r == '\n' || r == '\t' || (r >= 32 && r != 127) {
			printable++
		}
	}
	return float64(printable) / float64(total)
}
