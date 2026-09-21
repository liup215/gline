package adkagent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liup215/gline/internal/sessionstore"
	"github.com/liup215/gline/internal/storage"
	"github.com/liup215/gline/internal/tools"
)

// openTestSessionStore opens a disk-backed ADK session service for live tests.
func openTestSessionStore(path string) (*sessionstore.Store, error) {
	return sessionstore.Open(sessionstore.Options{Path: path})
}

// TestLiveOpenCodeHistoryResume drives the full Phase 6 resume chain against
// the real provider: run a turn with a persistent session + task store, then
// attach a fresh agent to the same session via ResumeSession and confirm the
// conversation context carried over.
//
// GLINE_LIVE_SMOKE=1 OPENCODE_API_KEY=... go test ./internal/adkagent/ -run TestLiveHistoryResume -v
func TestLiveOpenCodeHistoryResume(t *testing.T) {
	key := os.Getenv("OPENCODE_API_KEY")
	if os.Getenv("GLINE_LIVE_SMOKE") != "1" || key == "" {
		t.Skip("set GLINE_LIVE_SMOKE=1 and OPENCODE_API_KEY to run")
	}

	dir := t.TempDir()
	ss, err := openTestSessionStore(filepath.Join(dir, "sessions.db"))
	if err != nil {
		t.Fatalf("sessionstore: %v", err)
	}
	defer ss.Close()
	store, err := storage.NewSQLiteStore(filepath.Join(dir, "gline.db"))
	if err != nil {
		t.Fatalf("sqlite store: %v", err)
	}
	defer store.Close()

	registry := tools.InitDefaultRegistry(nil, nil)
	ctx := context.Background()

	// Agent 1: learns a codeword, persisted to disk.
	a1, err := New(ctx, Options{
		Provider:       "opencode",
		Model:          "mimo-v2.5",
		APIKey:         key,
		BaseURL:        "https://opencode.ai/zen/go/v1",
		Tools:          registry,
		Yolo:           true,
		SessionService: ss.Service(),
		Store:          store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cb1 := newRecordingCallback()
	if _, err := a1.RunWithCallback(ctx, "Remember this for later: the codeword is ZEBRA-7734.", cb1); err != nil {
		t.Fatalf("turn 1: %v", err)
	}
	sessID := a1.SessionID()
	if a1.TaskID() == "" {
		t.Fatal("task row must exist after turn 1")
	}

	// Agent 2: brand-new process state, resumes the recorded session.
	a2, err := New(ctx, Options{
		Provider:       "opencode",
		Model:          "mimo-v2.5",
		APIKey:         key,
		BaseURL:        "https://opencode.ai/zen/go/v1",
		Tools:          registry,
		Yolo:           true,
		SessionService: ss.Service(),
	})
	if err != nil {
		t.Fatalf("New agent 2: %v", err)
	}
	if err := a2.ResumeSession(ctx, sessID); err != nil {
		t.Fatalf("ResumeSession: %v", err)
	}
	cb2 := newRecordingCallback()
	if _, err := a2.RunWithCallback(ctx, "What was the codeword? Reply with it only.", cb2); err != nil {
		t.Fatalf("turn 2: %v", err)
	}
	if !strings.Contains(cb2.content.String(), "ZEBRA-7734") {
		t.Fatalf("resumed agent lost context, got %q", cb2.content.String())
	}
	t.Logf("resume OK: %q", cb2.content.String())
}
