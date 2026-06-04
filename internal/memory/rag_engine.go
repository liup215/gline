// rag_engine.go provides the high-level RAG layer wrapper.
package memory

import (
	"context"
	"path/filepath"
)

// RAGManager wraps VectorStore with per-KB lifecycle.
type RAGManager struct{}

// NewRAGManager creates a manager.
func NewRAGManager() *RAGManager { return &RAGManager{} }

// Close is a no-op for now (stores are opened per-operation).
func (r *RAGManager) Close() error { return nil }

// Search performs vector+FTS hybrid search for a KB.
func (r *RAGManager) Search(ctx context.Context, kbID string, queryEmbedding []float32, queryText string, topK int, minScore float64) ([]Chunk, error) {
	ragPath := filepath.Join(KBDir(kbID), "rag.db")
	// Dimension is auto-detected from stored embeddings
	vs, err := NewVectorStore(ragPath, 0)
	if err != nil {
		return nil, err
	}
	defer vs.Close()
	return vs.Search(ctx, kbID, queryEmbedding, queryText, topK, minScore)
}
