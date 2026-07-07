-- Enable pgvector and migrate knowledge_chunks embeddings to a native vector column.
-- Run this against PostgreSQL after the vector extension is installed on the server.

CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE knowledge_chunks
  ADD COLUMN IF NOT EXISTS embedding_vector vector(256);

UPDATE knowledge_chunks
SET embedding_vector = embedding_json::vector
WHERE embedding_vector IS NULL
  AND embedding_json IS NOT NULL
  AND embedding_json <> ''
  AND embedding_json <> '[]';

CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_embedding_vector_hnsw
ON knowledge_chunks
USING hnsw (embedding_vector vector_cosine_ops)
WHERE embedding_vector IS NOT NULL;

ANALYZE knowledge_chunks;
