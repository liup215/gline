// Package sessionstore provides gline's persistent session storage for the
// ADK-based agent loop. It wraps the official ADK GORM SQLite session service
// so every agent turn — user input, model text, tool calls, tool results — is
// appended as an event in an append-only log. That event log is the single
// source of truth for the conversation; gline's own SQLite database remains
// the task index (task metadata + session_id mapping) for `gline history`.
package sessionstore

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	adkdb "google.golang.org/adk/v2/session/database"
	"google.golang.org/adk/v2/session"
)

// AppName is the ADK application name used for all gline sessions.
// ADK scopes sessions by (AppName, UserID, SessionID); keeping this constant
// in one place guarantees every component agrees on the scope.
const AppName = "gline"

// DefaultUserID is the single-user identity for the desktop/CLI product.
// gline does not have multi-user accounts; a stable constant keeps the ADK
// request shape uniform.
const DefaultUserID = "local"

// Options configures the session store.
type Options struct {
	// Path is the SQLite database file. Empty means DefaultPath().
	Path string
	// Verbose enables GORM SQL logging (off by default; the ADK service
	// logs red "record not found" lines that are expected, not errors).
	Verbose bool
}

// Store wraps the ADK session.Service backed by a pure-Go SQLite database
// (glebarez/sqlite, no CGO — consistent with gline's existing storage stack).
type Store struct {
	svc  session.Service
	db   *sql.DB
	path string
}

// DefaultPath returns the default session database location:
// ~/.gline/sessions.db
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home dir: %w", err)
	}
	return filepath.Join(home, ".gline", "sessions.db"), nil
}

// Open opens (creating if necessary) the session database and returns a
// ready-to-use Store. The ADK GORM service runs its own schema migration on
// first use, so no manual DDL is needed here.
func Open(opts Options) (*Store, error) {
	path := opts.Path
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating session dir %s: %w", dir, err)
		}
	}

	cfg := &gorm.Config{}
	if !opts.Verbose {
		cfg.Logger = logger.Discard
	}
	db, err := gorm.Open(sqlite.Open(path), cfg)
	if err != nil {
		return nil, fmt.Errorf("opening session db %s: %w", path, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("extracting sql db: %w", err)
	}

	svc, err := adkdb.NewSessionServiceFromDB(db)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("creating adk session service: %w", err)
	}
	if err := adkdb.AutoMigrate(svc); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("migrating session schema: %w", err)
	}

	return &Store{svc: svc, db: sqlDB, path: path}, nil
}

// Service returns the underlying ADK session service. Callers (the agent
// assembly layer, the runner) use this directly; everything else should go
// through the helpers below so scope constants stay in one place.
func (s *Store) Service() session.Service { return s.svc }

// Path returns the database file backing this store.
func (s *Store) Path() string { return s.path }

// Close releases the database connection.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
