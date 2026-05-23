// Package storage provides persistent storage for gline.
package storage

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"database/sql"
	_ "modernc.org/sqlite"
)

// DefaultDBPath returns the default SQLite database file path.
func DefaultDBPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".gline", "gline.db")
	}
	return filepath.Join(homeDir, ".gline", "gline.db")
}

// Open opens a SQLite database at the given path, creating the directory
// and applying migrations if necessary. It also enables WAL mode for better concurrency.
func Open(dbPath string) (*sql.DB, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_busy_timeout=5000&_foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Apply migrations
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Explicitly enable foreign key support (required for ON DELETE CASCADE)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return db, nil
}

// migrate applies database schema migrations.
func migrate(db *sql.DB) error {
	// Create migrations tracking table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	var version sql.NullInt64
	err := db.QueryRow("SELECT MAX(version) FROM migrations").Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get migration version: %w", err)
	}
	v := 0
	if version.Valid {
		v = int(version.Int64)
	}

	// Apply pending migrations
	// Version 1: Initial schema
	if v < 1 {
		if err := applyV1(db); err != nil {
			return err
		}
		if _, err := db.Exec("INSERT INTO migrations (version) VALUES (1)"); err != nil {
			return fmt.Errorf("failed to record v1 migration: %w", err)
		}
	}

	// Enable foreign keys for cascade deletes
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return nil
}

func applyV1(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			title TEXT,
			prompt TEXT NOT NULL,
			mode TEXT NOT NULL,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			status TEXT DEFAULT 'running',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT,
			reasoning_content TEXT,
			tool_calls TEXT,
			tool_call_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS tool_calls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT NOT NULL,
			tool_name TEXT NOT NULL,
			input TEXT,
			output TEXT,
			error TEXT,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_task_id ON messages(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_task_id ON tool_calls(task_id)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute schema: %w", err)
		}
	}

	return nil
}

// generateUUID generates a simple UUID v4 string.
func generateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Version 4 UUID
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// now returns the current UTC time.
func now() time.Time {
	return time.Now().UTC()
}
