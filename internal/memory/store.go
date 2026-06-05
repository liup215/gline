// store.go implements a pure-Go vector store backed by SQLite.
// It does NOT require sqlite-vec extension (modernc.org/sqlite can't load C extensions).
// Embeddings are stored as BLOB (float32 array) and similarity is computed in Go.
package memory

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// VectorStore is a SQLite-backed store for documents, chunks and embeddings.
type VectorStore struct {
	db     *sql.DB
	dim    int // embedding dimension
	path   string
}

// NewVectorStore opens (or creates) a RAG database at path.
func NewVectorStore(path string, dim int) (*VectorStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_busy_timeout=5000&_foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	vs := &VectorStore{db: db, dim: dim, path: path}
	if err := vs.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return vs, nil
}

// Close closes the database.
func (s *VectorStore) Close() error { return s.db.Close() }

func (s *VectorStore) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			kb_id TEXT NOT NULL,
			name TEXT NOT NULL,
			source_path TEXT,
			file_type TEXT,
			content TEXT,
			char_count INTEGER,
			chunk_count INTEGER,
			status TEXT DEFAULT 'indexed',
			fact_ids TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS chunks (
			id TEXT PRIMARY KEY,
			doc_id TEXT NOT NULL,
			kb_id TEXT NOT NULL,
			content TEXT NOT NULL,
			start_offset INTEGER,
			end_offset INTEGER,
			embedding BLOB,
			sequence INTEGER,
			wiki_page_ref TEXT,
			fact_ids TEXT,
			FOREIGN KEY (doc_id) REFERENCES documents(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_kb ON chunks(kb_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_doc ON chunks(doc_id)`,
		// FTS5 virtual table for full-text search over chunks
		`CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(content, chunk_id UNINDEXED, kb_id UNINDEXED)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	return nil
}

// GetDocumentIDByName returns the document ID for a given name in a KB, or empty if not found.
func (s *VectorStore) GetDocumentIDByName(ctx context.Context, kbID, name string) (string, error) {
	var id string
	row := s.db.QueryRowContext(ctx, `SELECT id FROM documents WHERE kb_id = ? AND name = ?`, kbID, name)
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return id, nil
}

// StoreDocument inserts a document and its chunks.
func (s *VectorStore) StoreDocument(ctx context.Context, doc *Document, chunks []Chunk) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO documents
		(id, kb_id, name, source_path, file_type, content, char_count, chunk_count, status, fact_ids, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doc.ID, doc.KBID, doc.Name, doc.SourcePath, doc.FileType, doc.Content,
		doc.CharCount, doc.ChunkCount, doc.Status, strings.Join(doc.FactIDs, ","), doc.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert doc: %w", err)
	}

	for i := range chunks {
		emb, err := encodeEmbedding(chunks[i].Embedding)
		if err != nil {
			return fmt.Errorf("encode emb: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO chunks
			(id, doc_id, kb_id, content, start_offset, end_offset, embedding, sequence, wiki_page_ref, fact_ids)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			chunks[i].ID, chunks[i].DocID, chunks[i].KBID, chunks[i].Content,
			chunks[i].StartOffset, chunks[i].EndOffset, emb,
			chunks[i].Sequence, chunks[i].WikiPageRef, strings.Join(chunks[i].FactIDs, ","))
		if err != nil {
			return fmt.Errorf("insert chunk: %w", err)
		}
		// Also insert into FTS5
		_, err = tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO chunks_fts (content, chunk_id, kb_id)
			VALUES (?, ?, ?)`, chunks[i].Content, chunks[i].ID, chunks[i].KBID)
		if err != nil {
			// FTS5 may reject long content — truncate if needed
			short := chunks[i].Content
			if len(short) > 10000 {
				short = short[:10000]
			}
			_, _ = tx.ExecContext(ctx, `INSERT OR REPLACE INTO chunks_fts (content, chunk_id, kb_id) VALUES (?, ?, ?)`,
				short, chunks[i].ID, chunks[i].KBID)
		}
	}

	return tx.Commit()
}

// DeleteDocument removes a document and all its chunks (cascade).
func (s *VectorStore) DeleteDocument(ctx context.Context, docID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get chunk IDs for FTS cleanup
	rows, err := tx.QueryContext(ctx, `SELECT id FROM chunks WHERE doc_id = ?`, docID)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()

	for _, id := range ids {
		_, _ = tx.ExecContext(ctx, `DELETE FROM chunks_fts WHERE chunk_id = ?`, id)
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM documents WHERE id = ?`, docID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Search does a hybrid search: vector KNN + FTS5 and merges with RRF.
func (s *VectorStore) Search(ctx context.Context, kbID string, queryEmbedding []float32, queryText string, topK int, minScore float64) ([]Chunk, error) {
	// 1. Vector search
	vecResults, err := s.vectorSearch(ctx, kbID, queryEmbedding, topK*2)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	// 2. FTS5 search
	ftsResults, err := s.ftsSearch(ctx, kbID, queryText, topK*2)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}

	// 3. RRF merge
	merged := rrfMerge(vecResults, ftsResults, topK)
	if minScore > 0 {
		var filtered []Chunk
		for _, c := range merged {
			if c.sim >= minScore {
				filtered = append(filtered, c.Chunk)
			}
		}
		return filtered, nil
	}
	var results []Chunk
	for _, c := range merged {
		results = append(results, c.Chunk)
	}
	return results, nil
}

// vectorResult wraps a chunk with its cosine similarity score.
type vectorResult struct {
	Chunk
	sim float64
}

func (s *VectorStore) vectorSearch(ctx context.Context, kbID string, query []float32, limit int) ([]vectorResult, error) {
	if s.dim > 0 && len(query) != s.dim {
		return nil, fmt.Errorf("query dim %d != store dim %d", len(query), s.dim)
	}
	qNorm := normalizeL2(query) // borrowed from embedder.go

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, doc_id, kb_id, content, start_offset, end_offset, embedding, sequence, wiki_page_ref, fact_ids
		FROM chunks WHERE kb_id = ?`, kbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []vectorResult
	for rows.Next() {
		var c Chunk
		var embBlob []byte
		var factStr string
		err := rows.Scan(&c.ID, &c.DocID, &c.KBID, &c.Content, &c.StartOffset, &c.EndOffset, &embBlob, &c.Sequence, &c.WikiPageRef, &factStr)
		if err != nil {
			continue
		}
		emb, err := decodeEmbedding(embBlob)
		if err != nil || len(emb) == 0 {
			continue
		}
		c.Embedding = emb
		c.FactIDs = splitNull(factStr)
		sim, _ := CosineSimilarity(qNorm, normalizeL2(emb))
		all = append(all, vectorResult{Chunk: c, sim: sim})
	}

	// sort by similarity descending
	for i := 0; i < len(all)-1; i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].sim > all[i].sim {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	if limit > len(all) {
		limit = len(all)
	}
	return all[:limit], nil
}

func (s *VectorStore) ftsSearch(ctx context.Context, kbID, query string, limit int) ([]vectorResult, error) {
	// Escape FTS5 special chars
	escaped := strings.ReplaceAll(query, `"`, `""`)
	escaped = `"` + escaped + `"`

	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.doc_id, c.kb_id, c.content, c.start_offset, c.end_offset, c.embedding, c.sequence, c.wiki_page_ref, c.fact_ids
		FROM chunks_fts fts
		JOIN chunks c ON c.id = fts.chunk_id
		WHERE chunks_fts MATCH ? AND fts.kb_id = ?
		ORDER BY rank
		LIMIT ?`, escaped, kbID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []vectorResult
	for rows.Next() {
		var c Chunk
		var embBlob []byte
		var factStr string
		if err := rows.Scan(&c.ID, &c.DocID, &c.KBID, &c.Content, &c.StartOffset, &c.EndOffset, &embBlob, &c.Sequence, &c.WikiPageRef, &factStr); err != nil {
			continue
		}
		c.Embedding, _ = decodeEmbedding(embBlob)
		c.FactIDs = splitNull(factStr)
		results = append(results, vectorResult{Chunk: c, sim: 0}) // sim computed later
	}
	return results, nil
}

// rrfMerge merges vector and FTS results using Reciprocal Rank Fusion.
func rrfMerge(vec, fts []vectorResult, k int) []vectorResult {
	const rrfK = 60
	score := make(map[string]float64)
	chunkMap := make(map[string]vectorResult)

	for rank, r := range vec {
		score[r.ID] += 1.0 / (float64(rank) + 1.0 + rrfK)
		chunkMap[r.ID] = r
	}
	for rank, r := range fts {
		score[r.ID] += 1.0 / (float64(rank) + 1.0 + rrfK)
		if _, ok := chunkMap[r.ID]; !ok {
			chunkMap[r.ID] = r
		}
	}

	type item struct {
		id    string
		score float64
	}
	var items []item
	for id, sc := range score {
		items = append(items, item{id, sc})
	}
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].score > items[i].score {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	if k > len(items) {
		k = len(items)
	}
	var out []vectorResult
	for i := 0; i < k; i++ {
		it := items[i]
		c := chunkMap[it.id]
		c.sim = it.score // reuse sim field for RRF score
		out = append(out, c)
	}
	return out
}

func encodeEmbedding(v []float32) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeEmbedding(b []byte) ([]float32, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var v []float32
	dec := gob.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

func splitNull(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// ListDocuments returns all documents for a knowledge base.
func (s *VectorStore) ListDocuments(ctx context.Context, kbID string) ([]Document, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kb_id, name, source_path, file_type, content, char_count, chunk_count, status, fact_ids, created_at
		FROM documents WHERE kb_id = ?`, kbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []Document
	for rows.Next() {
		var d Document
		var factStr string
		var created time.Time
		err := rows.Scan(&d.ID, &d.KBID, &d.Name, &d.SourcePath, &d.FileType, &d.Content, &d.CharCount, &d.ChunkCount, &d.Status, &factStr, &created)
		if err != nil {
			continue
		}
		d.FactIDs = splitNull(factStr)
		d.CreatedAt = created
		docs = append(docs, d)
	}
	return docs, nil
}
