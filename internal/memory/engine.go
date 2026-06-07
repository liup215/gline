// engine.go is the unified memory engine orchestrating all four layers.
package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/liup215/gline/internal/log"
)

// DefaultPaths computes standard paths under ~/.gline/memory/.
func DefaultPaths() (registryPath, factsPath string) {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".gline", "memory")
	return filepath.Join(base, "kb-registry.db"), filepath.Join(base, "facts.db")
}

// UnifiedEngine wires together all four layers.
type UnifiedEngine struct {
	Registry  *KBRegistry
	FactStore FactStore
	// WikiEngine and RAGEngine are concrete implementations added later
	RAGEngine  *RAGManager  // wraps VectorStore + Embedder
	WikiEngine *WikiManager // wraps WikiFS + LLM integration
	Embedder   Embedder
	// Caller is injected by the application layer for LLM-driven tasks (wiki ingest, fact extraction).
	Caller func(ctx context.Context, systemPrompt, userContent string) (string, error)

	mu       sync.RWMutex
	ingestMu sync.Mutex
}

// NewUnifiedEngine builds the engine with the provided embedder.
func NewUnifiedEngine(embedder Embedder) (*UnifiedEngine, error) {
	regPath, factsPath := DefaultPaths()
	reg, err := NewKBRegistry(regPath)
	if err != nil {
		return nil, fmt.Errorf("registry: %w", err)
	}
	fs, err := NewSQLiteFactStore(factsPath)
	if err != nil {
		reg.Close()
		return nil, fmt.Errorf("fact store: %w", err)
	}
	return &UnifiedEngine{
		Registry:   reg,
		FactStore:  fs,
		Embedder:   embedder,
		RAGEngine:  NewRAGManager(),
		WikiEngine: NewWikiManager(),
	}, nil
}

// Close shuts down all layers.
func (e *UnifiedEngine) Close() error {
	var errs []string
	if e.Registry != nil {
		if err := e.Registry.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if e.FactStore != nil {
		if err := e.FactStore.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if e.RAGEngine != nil {
		if err := e.RAGEngine.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if e.WikiEngine != nil {
		if err := e.WikiEngine.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ─── KB Lifecycle ───────────────────────────────────────────────────────────

// InitKB creates a new knowledge base directory and registers it.
func (e *UnifiedEngine) InitKB(ctx context.Context, name, description string, kbType KBType) (*KnowledgeBase, error) {
	kb := &KnowledgeBase{
		ID:          genID(),
		Name:        name,
		Description: description,
		Type:        kbType,
		CreatedAt:   time.Now().UTC(),
	}
	if kbType != KBTypeRAG {
		return nil, fmt.Errorf("unsupported kb type: %s (only 'rag' is supported)", kbType)
	}
	if err := e.Registry.Create(ctx, kb); err != nil {
		return nil, err
	}
	// Create on-disk directories
	dir := KBDir(kb.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	// Create RAG database
	ragPath := filepath.Join(dir, "rag.db")
	_, err := NewVectorStore(ragPath, e.Embedder.Dimension())
	if err != nil {
		return nil, err
	}
	return kb, nil
}

// GetKB resolves by ID or name.
func (e *UnifiedEngine) GetKB(ctx context.Context, idOrName string) (*KnowledgeBase, error) {
	kb, err := e.Registry.GetByID(ctx, idOrName)
	if err == nil {
		return kb, nil
	}
	return e.Registry.GetByName(ctx, idOrName)
}

// RemoveKB deletes a knowledge base and its on-disk data.
func (e *UnifiedEngine) RemoveKB(ctx context.Context, id string) error {
	kb, err := e.Registry.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := e.Registry.Delete(ctx, id); err != nil {
		return err
	}
	dir := KBDir(kb.ID)
	_ = os.RemoveAll(dir)
	return nil
}

// findDocByName looks up an existing document by filename inside a KB.
func (e *UnifiedEngine) findDocByName(ctx context.Context, kbID, name string) string {
	ragPath := filepath.Join(KBDir(kbID), "rag.db")
	vs, err := NewVectorStore(ragPath, 0)
	if err != nil {
		return ""
	}
	defer vs.Close()
	id, _ := vs.GetDocumentIDByName(ctx, kbID, name)
	return id
}

// ListKB returns all knowledge bases.
func (e *UnifiedEngine) ListKB(ctx context.Context) ([]KnowledgeBase, error) {
	return e.Registry.List(ctx)
}

// EnsureDefaultKB creates a default KB named "default" if none exists.
func EnsureDefaultKB(ctx context.Context, e *UnifiedEngine) error {
	list, err := e.ListKB(ctx)
	if err != nil {
		return err
	}
	if len(list) > 0 {
		return nil // already have at least one KB
	}
	_, err = e.InitKB(ctx, "default", "Auto-created default KB", KBTypeRAG)
	return err
}

// ─── High-level Ingest (add document to KB) ──────────────────────────────────

// IngestFile reads a file, parses it, chunks it, embeds it, and stores in RAG.
func (e *UnifiedEngine) IngestFile(ctx context.Context, kbID, filePath string) error {
	e.ingestMu.Lock()
	defer e.ingestMu.Unlock()

	_, err := e.Registry.GetByID(ctx, kbID)
	if err != nil {
		return err
	}
	// Copy raw file into KB directory
	rawDir := filepath.Join(KBDir(kbID), "raw")
	_ = os.MkdirAll(rawDir, 0755)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	base := filepath.Base(filePath)
	rawPath := filepath.Join(rawDir, base)
	if err := os.WriteFile(rawPath, data, 0644); err != nil {
		return err
	}

	// Parse content
	content, err := ParseDocument(rawPath)
	if err != nil {
		return err
	}

	// Deduplication: remove existing document with same name
	oldDocID := e.findDocByName(ctx, kbID, base)
	if oldDocID != "" {
		ragPath := filepath.Join(KBDir(kbID), "rag.db")
		vs, err := NewVectorStore(ragPath, e.Embedder.Dimension())
		if err == nil {
			_ = vs.DeleteDocument(ctx, oldDocID)
			vs.Close()
		}
		_ = e.Registry.IncrementCounters(ctx, kbID, -1, 0, 0, 0) // counters fixed on re-ingest
		// Note: chunk count adjustment happens naturally via StoreDocument below
	}

	// Build document record
	doc := Document{
		ID:        genID(),
		KBID:      kbID,
		Name:      base,
		FileType:  filepath.Ext(base),
		Content:   content,
		CharCount: len(content),
		Status:    "indexing",
		CreatedAt: time.Now().UTC(),
	}

	// Chunk + embed + store in RAG
	chunker := NewChunker()
	chunks := chunker.Chunk(doc.ID, kbID, content)
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Content
	}
	vecs, err := EmbedAndNormalize(ctx, e.Embedder, texts)
	if err != nil {
		return fmt.Errorf("embedding: %w", err)
	}
	for i := range chunks {
		chunks[i].Embedding = vecs[i]
	}

	doc.ChunkCount = len(chunks)
	doc.Status = "indexed"

	ragPath := filepath.Join(KBDir(kbID), "rag.db")
	vs, err := NewVectorStore(ragPath, e.Embedder.Dimension())
	if err != nil {
		return err
	}
	defer vs.Close()

	if err := vs.StoreDocument(ctx, &doc, chunks); err != nil {
		return err
	}

	_ = e.Registry.IncrementCounters(ctx, kbID, 1, len(chunks), 0, 0)
	return nil
}

// WikiIngestFile parses a file and triggers LLM-based wiki generation.
// This is completely separate from RAG ingestion — the caller must manage
// which KB / wiki target the results belong to.
func (e *UnifiedEngine) WikiIngestFile(ctx context.Context, filePath, kbID string) error {
	if e.Caller == nil {
		return fmt.Errorf("wiki ingestion requires e.Caller to be set")
	}
	content, err := ParseDocument(filePath)
	if err != nil {
		return fmt.Errorf("parse %s: %w", filePath, err)
	}
	doc := Document{
		ID:        genID(),
		KBID:      kbID,
		Name:      filepath.Base(filePath),
		Content:   content,
		CharCount: len(content),
		CreatedAt: time.Now().UTC(),
	}
	log.Infof("[Wiki] triggering standalone wiki ingest for %s (kb=%s)", doc.Name, kbID)
	go e.WikiEngine.IngestAsync(kbID, doc, e.Caller)
	return nil
}
