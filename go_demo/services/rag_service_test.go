package services

import (
	"context"
	"strings"
	"testing"

	"go-demo/models"
)

func TestSplitTextUsesOverlap(t *testing.T) {
	got := splitText("abcdefghijklmnopqrstuvwxyz", 10, 3)
	if len(got) != 4 {
		t.Fatalf("expected 4 chunks, got %d: %#v", len(got), got)
	}
	if got[0] != "abcdefghij" {
		t.Fatalf("unexpected first chunk: %q", got[0])
	}
	if got[1] != "hijklmnopq" {
		t.Fatalf("unexpected overlapped second chunk: %q", got[1])
	}
}

func TestCosine(t *testing.T) {
	if score := cosine([]float64{1, 0}, []float64{1, 0}); score != 1 {
		t.Fatalf("expected identical vectors to score 1, got %f", score)
	}
	if score := cosine([]float64{1, 0}, []float64{0, 1}); score != 0 {
		t.Fatalf("expected orthogonal vectors to score 0, got %f", score)
	}
}

func TestBuildSiteProjectChunks(t *testing.T) {
	chunks := buildSiteProjectChunks(models.SiteProject{
		ID:          7,
		Name:        "Admin Platform",
		Summary:     "RBAC admin system",
		Description: "Manages users, roles, website content, and chat.",
		StackTags:   "Go,Vue,PostgreSQL",
		Status:      "published",
	})
	if len(chunks) != 1 {
		t.Fatalf("expected one project chunk, got %d", len(chunks))
	}
	if chunks[0].SourceType != knowledgeSourceProject || chunks[0].SourceID != 7 {
		t.Fatalf("unexpected project source: %#v", chunks[0])
	}
	if chunks[0].Visibility != knowledgeVisibilityPublic {
		t.Fatalf("expected public project chunk, got %q", chunks[0].Visibility)
	}
	if !containsAll(chunks[0].Content, "Admin Platform", "Go,Vue,PostgreSQL") {
		t.Fatalf("project chunk content missing expected text: %q", chunks[0].Content)
	}
}

func TestBuildUploadedDocumentChunksVisibility(t *testing.T) {
	chunks := buildUploadedDocumentChunks(models.UploadedDocument{
		ID:           12,
		OriginalName: "Runbook.md",
		MimeType:     "text/markdown",
		FileSize:     128,
		Visibility:   knowledgeVisibilityInternal,
		TextContent:  "private deployment notes",
		Status:       "active",
	})
	if len(chunks) != 1 {
		t.Fatalf("expected one document chunk, got %d", len(chunks))
	}
	if chunks[0].Visibility != knowledgeVisibilityInternal {
		t.Fatalf("expected internal document chunk, got %q", chunks[0].Visibility)
	}
	if chunks[0].Metadata["visibility"] != knowledgeVisibilityInternal {
		t.Fatalf("expected internal metadata, got %#v", chunks[0].Metadata)
	}
}

func TestNormalizeVisibilityDefaultsToInternal(t *testing.T) {
	if got := normalizeVisibility(""); got != knowledgeVisibilityInternal {
		t.Fatalf("empty visibility should default to internal, got %q", got)
	}
	if got := normalizeVisibility("public"); got != knowledgeVisibilityPublic {
		t.Fatalf("public visibility changed to %q", got)
	}
}

func TestBuildSiteTechStackChunks(t *testing.T) {
	chunks := buildSiteTechStackChunks(models.SiteTechStack{
		ID:          3,
		Name:        "Gin",
		Category:    "backend",
		Level:       80,
		Description: "HTTP API framework",
		IsActive:    true,
	})
	if len(chunks) != 1 {
		t.Fatalf("expected one tech stack chunk, got %d", len(chunks))
	}
	if chunks[0].SourceType != knowledgeSourceTech || chunks[0].SourceID != 3 {
		t.Fatalf("unexpected tech source: %#v", chunks[0])
	}
	if !containsAll(chunks[0].Content, "Gin", "backend", "HTTP API framework") {
		t.Fatalf("tech stack chunk content missing expected text: %q", chunks[0].Content)
	}
}

func TestLocalEmbeddingDeterministic(t *testing.T) {
	client := localAIClient{}
	a, err := client.Embed(context.Background(), []string{"React Go database"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := client.Embed(context.Background(), []string{"React Go database"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 1 || len(b) != 1 || len(a[0]) != len(b[0]) {
		t.Fatalf("unexpected embedding dimensions: %v %v", len(a), len(b))
	}
	for i := range a[0] {
		if a[0][i] != b[0][i] {
			t.Fatalf("embedding differs at index %d", i)
		}
	}
}

func containsAll(value string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}
