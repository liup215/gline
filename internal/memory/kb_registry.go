// kb_registry.go manages the list of knowledge bases.
package memory

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// KBRegistry stores knowledge base metadata.
type KBRegistry struct {
	db *sql.DB
}

// NewKBRegistry opens the registry database.
func NewKBRegistry(path string) (*KBRegistry, error) {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	r := &KBRegistry{db: db}
	if err := r.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return r, nil
}

func (r *KBRegistry) migrate() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS knowledge_bases (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			type TEXT DEFAULT 'hybrid',
			doc_count INTEGER DEFAULT 0,
			chunk_count INTEGER DEFAULT 0,
			fact_count INTEGER DEFAULT 0,
			wiki_page_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
	return err
}

// Close closes the registry.
func (r *KBRegistry) Close() error { return r.db.Close() }

// Create adds a new knowledge base.
func (r *KBRegistry) Create(ctx context.Context, kb *KnowledgeBase) error {
	if kb.ID == "" {
		kb.ID = genID()
	}
	if kb.CreatedAt.IsZero() {
		kb.CreatedAt = time.Now().UTC()
	}
	kb.UpdatedAt = kb.CreatedAt
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO knowledge_bases (id, name, description, type, doc_count, chunk_count, fact_count, wiki_page_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		kb.ID, kb.Name, kb.Description, kb.Type, kb.DocCount, kb.ChunkCount, kb.FactCount, kb.WikiPageCount, kb.CreatedAt, kb.UpdatedAt)
	return err
}

// GetByID returns a knowledge base by ID.
func (r *KBRegistry) GetByID(ctx context.Context, id string) (*KnowledgeBase, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, description, type, doc_count, chunk_count, fact_count, wiki_page_count, created_at, updated_at FROM knowledge_bases WHERE id = ?`, id)
	return r.scanKB(row)
}

// GetByName returns a knowledge base by exact name.
func (r *KBRegistry) GetByName(ctx context.Context, name string) (*KnowledgeBase, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, description, type, doc_count, chunk_count, fact_count, wiki_page_count, created_at, updated_at FROM knowledge_bases WHERE name = ?`, name)
	return r.scanKB(row)
}

// List returns all knowledge bases.
func (r *KBRegistry) List(ctx context.Context) ([]KnowledgeBase, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, description, type, doc_count, chunk_count, fact_count, wiki_page_count, created_at, updated_at FROM knowledge_bases ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []KnowledgeBase
	for rows.Next() {
		kb, err := r.scanKB(rows)
		if err != nil {
			continue
		}
		list = append(list, *kb)
	}
	return list, nil
}

// Delete removes a knowledge base.
func (r *KBRegistry) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM knowledge_bases WHERE id = ?`, id)
	return err
}

// IncrementCounters atomically updates counters.
func (r *KBRegistry) IncrementCounters(ctx context.Context, id string, docs, chunks, facts, pages int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE knowledge_bases SET
			doc_count = doc_count + ?,
			chunk_count = chunk_count + ?,
			fact_count = fact_count + ?,
			wiki_page_count = wiki_page_count + ?,
			updated_at = ?
		WHERE id = ?`,
		docs, chunks, facts, pages, time.Now().UTC(), id)
	return err
}

func (r *KBRegistry) scanKB(scanner interface{ Scan(...interface{}) error }) (*KnowledgeBase, error) {
	var kb KnowledgeBase
	err := scanner.Scan(&kb.ID, &kb.Name, &kb.Description, &kb.Type, &kb.DocCount, &kb.ChunkCount, &kb.FactCount, &kb.WikiPageCount, &kb.CreatedAt, &kb.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("knowledge base not found")
	}
	if err != nil {
		return nil, err
	}
	return &kb, nil
}
