// fact_store_sqlite.go implements the Layer-1 FactStore backed by SQLite.
package memory

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteFactStore implements FactStore using pure SQLite (no vector extension needed).
type SQLiteFactStore struct {
	db *sql.DB
}

// NewSQLiteFactStore opens or creates the facts database.
func NewSQLiteFactStore(path string) (*SQLiteFactStore, error) {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_busy_timeout=5000&_foreign_keys=1")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	fs := &SQLiteFactStore{db: db}
	if err := fs.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return fs, nil
}

func (s *SQLiteFactStore) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS facts (
			id TEXT PRIMARY KEY,
			kb_id TEXT,
			category TEXT NOT NULL,
			subject TEXT NOT NULL,
			predicate TEXT NOT NULL,
			object TEXT NOT NULL,
			confidence REAL DEFAULT 0.0,
			source TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			access_count INTEGER DEFAULT 0,
			last_access DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_facts_kb ON facts(kb_id)`,
		`CREATE INDEX IF NOT EXISTS idx_facts_subject ON facts(subject)`,
		`CREATE INDEX IF NOT EXISTS idx_facts_category ON facts(category)`,
		`CREATE INDEX IF NOT EXISTS idx_facts_confidence ON facts(confidence)`,
		`CREATE INDEX IF NOT EXISTS idx_facts_last_access ON facts(last_access)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS facts_fts USING fts5(subject, predicate, object, fact_id UNINDEXED)`,
		`CREATE TABLE IF NOT EXISTS entities (
			name TEXT PRIMARY KEY,
			first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("fact schema: %w", err)
		}
	}
	return nil
}

// Close implements FactStore.
func (s *SQLiteFactStore) Close() error { return s.db.Close() }

// Add implements FactStore (stub — populated by FactExtractor in Phase 4).
func (s *SQLiteFactStore) Add(ctx context.Context, text string, source ConversationRef) ([]FactChange, error) {
	return nil, fmt.Errorf("Add not yet implemented — use Apply() with pre-extracted changes")
}

// Apply persists pre-extracted fact changes directly (used by LLM-driven extraction).
//
// Smart-merge logic for ADD:
//   - If a fact with the same (subject, predicate) already exists, treat ADD
//     as UPDATE (update object/confidence/updated_at but keep the old ID).
//   - This compensates for the fact that LLMs can't know existing row IDs.
//   - DELETE is handled by (subject, predicate) search as a fallback when
//     the LLM-generated ID does not match existing rows.
func (s *SQLiteFactStore) Apply(ctx context.Context, changes []FactChange) error {
	if len(changes) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, ch := range changes {
		switch ch.Action {
		case "ADD":
			var existingID string
			_ = tx.QueryRowContext(ctx,
				`SELECT id FROM facts WHERE subject = ? AND predicate = ? LIMIT 1`,
				ch.Fact.Subject, ch.Fact.Predicate).Scan(&existingID)
			if existingID != "" {
				ch.Fact.ID = existingID
				ch.Fact.UpdatedAt = time.Now().UTC()
			}
			if err := s.upsertFact(ctx, tx, &ch.Fact); err != nil {
				_ = tx.Rollback()
				return err
			}
		case "UPDATE":
			var targetID string
			_ = tx.QueryRowContext(ctx,
				`SELECT id FROM facts WHERE id = ? LIMIT 1`, ch.Fact.ID).Scan(&targetID)
			if targetID == "" {
				_ = tx.QueryRowContext(ctx,
					`SELECT id FROM facts WHERE subject = ? AND predicate = ? LIMIT 1`,
					ch.Fact.Subject, ch.Fact.Predicate).Scan(&targetID)
			}
			if targetID != "" {
				ch.Fact.ID = targetID
				ch.Fact.UpdatedAt = time.Now().UTC()
				if err := s.upsertFact(ctx, tx, &ch.Fact); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		case "DELETE":
			result, err := tx.ExecContext(ctx, `DELETE FROM facts WHERE id = ?`, ch.Fact.ID)
			if err != nil {
				_ = tx.Rollback()
				return err
			}
			rows, _ := result.RowsAffected()
			if rows == 0 {
				_, _ = tx.ExecContext(ctx,
					`DELETE FROM facts WHERE subject = ? AND predicate = ?`,
					ch.Fact.Subject, ch.Fact.Predicate)
			}
			_, _ = tx.ExecContext(ctx, `DELETE FROM facts_fts WHERE fact_id = ?`, ch.Fact.ID)
		}
	}
	return tx.Commit()
}

// upsertFact inserts or replaces a fact.
func (s *SQLiteFactStore) upsertFact(ctx context.Context, tx *sql.Tx, f *Fact) error {
	var execer interface {
		ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	}
	if tx != nil {
		execer = tx
	} else {
		execer = s.db
	}

	now := time.Now().UTC()
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now
	}
	if f.UpdatedAt.IsZero() {
		f.UpdatedAt = now
	}
	if f.LastAccess.IsZero() {
		f.LastAccess = now
	}
	if f.Confidence == 0 {
		f.Confidence = 0.7
	}

	_, err := execer.ExecContext(ctx, `
		INSERT INTO facts (id, kb_id, category, subject, predicate, object, confidence, source, created_at, updated_at, access_count, last_access)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kb_id = excluded.kb_id,
			category = excluded.category,
			subject = excluded.subject,
			predicate = excluded.predicate,
			object = excluded.object,
			confidence = excluded.confidence,
			source = excluded.source,
			updated_at = excluded.updated_at,
			last_access = excluded.last_access`,
		f.ID, f.KBID, f.Category, f.Subject, f.Predicate, f.Object, f.Confidence, f.Source,
		f.CreatedAt, f.UpdatedAt, f.AccessCount, f.LastAccess)
	if err != nil {
		return err
	}
	_, _ = execer.ExecContext(ctx, `
		INSERT INTO entities (name, last_seen) VALUES (?, ?)
		ON CONFLICT(name) DO UPDATE SET last_seen = excluded.last_seen`,
		f.Subject, time.Now().UTC())
	_, _ = execer.ExecContext(ctx, `DELETE FROM facts_fts WHERE fact_id = ?`, f.ID)
	_, err = execer.ExecContext(ctx, `
		INSERT INTO facts_fts (subject, predicate, object, fact_id)
		VALUES (?, ?, ?, ?)`, f.Subject, f.Predicate, f.Object, f.ID)
	return err
}

// Search implements FactStore using FTS5 + LIKE fallback.
func (s *SQLiteFactStore) Search(ctx context.Context, query string, opts FactSearchOptions) ([]Fact, error) {
	if query == "" && len(opts.Categories) == 0 && len(opts.Entities) == 0 {
		return s.listAll(ctx, opts.TopK)
	}

	topK := opts.TopK
	if topK <= 0 {
		topK = 10
	}

	var count int
	escaped := "\"" + strings.ReplaceAll(query, "\"", "\"\"") + "\""
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM facts_fts WHERE facts_fts MATCH ?`, escaped).Scan(&count)
	if count > 0 {
		rows, err := s.db.QueryContext(ctx, `
			SELECT f.id, f.kb_id, f.category, f.subject, f.predicate, f.object, f.confidence, f.source, f.created_at, f.updated_at, f.access_count, f.last_access
			FROM facts_fts fts
			JOIN facts f ON f.id = fts.fact_id
			WHERE facts_fts MATCH ?
			ORDER BY rank
			LIMIT ?`, escaped, topK)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return s.scanFacts(rows, opts)
	}

	like := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kb_id, category, subject, predicate, object, confidence, source, created_at, updated_at, access_count, last_access
		FROM facts
		WHERE subject LIKE ? OR predicate LIKE ? OR object LIKE ?
		ORDER BY confidence DESC, last_access DESC
		LIMIT ?`, like, like, like, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanFacts(rows, opts)
}

func (s *SQLiteFactStore) listAll(ctx context.Context, limit int) ([]Fact, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, kb_id, category, subject, predicate, object, confidence, source, created_at, updated_at, access_count, last_access FROM facts ORDER BY last_access DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanFacts(rows, FactSearchOptions{})
}

// GetByEntity returns facts where subject matches entity name.
func (s *SQLiteFactStore) GetByEntity(ctx context.Context, entity string) ([]Fact, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, kb_id, category, subject, predicate, object, confidence, source, created_at, updated_at, access_count, last_access FROM facts WHERE subject = ? ORDER BY confidence DESC`, entity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanFacts(rows, FactSearchOptions{})
}

// GetByCategory filters facts by category.
func (s *SQLiteFactStore) GetByCategory(ctx context.Context, cat FactCategory) ([]Fact, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, kb_id, category, subject, predicate, object, confidence, source, created_at, updated_at, access_count, last_access FROM facts WHERE category = ? ORDER BY confidence DESC`, cat)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanFacts(rows, FactSearchOptions{})
}

// Decay lowers the access score of stale facts (soft-delete by lowering confidence).
func (s *SQLiteFactStore) Decay(ctx context.Context) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -30)
	_, err := s.db.ExecContext(ctx, `
		UPDATE facts SET confidence = confidence * 0.95
		WHERE last_access < ? AND confidence > 0.1`, cutoff)
	return err
}

func (s *SQLiteFactStore) scanFacts(rows *sql.Rows, opts FactSearchOptions) ([]Fact, error) {
	var facts []Fact
	for rows.Next() {
		var f Fact
		var last sql.NullTime
		err := rows.Scan(&f.ID, &f.KBID, &f.Category, &f.Subject, &f.Predicate, &f.Object, &f.Confidence, &f.Source, &f.CreatedAt, &f.UpdatedAt, &f.AccessCount, &last)
		if err != nil {
			continue
		}
		if last.Valid {
			f.LastAccess = last.Time
		}
		if opts.MinScore > 0 && f.Confidence < opts.MinScore {
			continue
		}
		if len(opts.Categories) > 0 {
			match := false
			for _, c := range opts.Categories {
				if f.Category == c {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if len(opts.Entities) > 0 {
			match := false
			for _, e := range opts.Entities {
				if f.Subject == e {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		facts = append(facts, f)
	}
	return facts, nil
}
