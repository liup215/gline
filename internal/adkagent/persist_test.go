package adkagent

import (
	"context"

	"strings"
	"testing"

	"google.golang.org/genai"

	"github.com/liup215/gline/internal/storage"
	"github.com/liup215/gline/pkg/types"
)

// TestTranscriptPersisted verifies the storage double-write: the task row is
// created on the first turn, linked to the ADK session, and the run's user /
// assistant / tool messages land in the messages table in the shape the TUI
// history view expects.
func TestTranscriptPersisted(t *testing.T) {
	fm := &fakeModel{turns: []fakeTurn{
		{final: modelWithCalls("Checking.", &genai.FunctionCall{
			ID: "call-1", Name: "read", Args: map[string]any{"path": "f.txt"},
		})},
		{final: modelText("All done.")},
	}}

	store, err := storage.NewSQLiteStoreInMemory()
	if err != nil {
		t.Fatalf("sqlite store: %v", err)
	}
	a, err := NewWithModel(context.Background(), Options{
		Model: "fake", Tools: testRegistry(t, ""), Store: store,
	}, fm)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	cb := newRecordingCallback()
	if _, err := a.RunWithCallback(context.Background(), "check f.txt", cb); err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}

	taskID := a.TaskID()
	if taskID == "" {
		t.Fatal("task row must be created on the first turn")
	}

	// Session mapping for history resume.
	sessID, err := store.GetTaskSessionID(taskID)
	if err != nil || sessID != a.SessionID() {
		t.Fatalf("session mapping: got %q err %v, want %q", sessID, err, a.SessionID())
	}

	msgs, err := store.GetMessages(taskID)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) < 4 {
		t.Fatalf("expected >=4 messages, got %d: %+v", len(msgs), msgs)
	}
	// Expected shape: user prompt, assistant (text+tool call), tool result,
	// assistant completion.
	if msgs[0].Role != string(types.RoleUser) || !strings.Contains(msgs[0].Content, "check f.txt") {
		t.Fatalf("msg[0] = %s/%q, want user prompt", msgs[0].Role, msgs[0].Content)
	}
	if msgs[1].Role != string(types.RoleAssistant) || !strings.Contains(msgs[1].Content, "Checking") || len(msgs[1].ToolCalls) == 0 {
		t.Fatalf("msg[1] = %s/%q calls=%d, want assistant with tool call", msgs[1].Role, msgs[1].Content, len(msgs[1].ToolCalls))
	}
	if msgs[2].Role != string(types.RoleTool) || msgs[2].ToolCallID != "call-1" {
		t.Fatalf("msg[2] = %s id=%q, want tool result for call-1", msgs[2].Role, msgs[2].ToolCallID)
	}
	last := msgs[len(msgs)-1]
	if last.Role != string(types.RoleAssistant) || !strings.Contains(last.Content, "All done.") {
		t.Fatalf("last = %s/%q, want assistant completion", last.Role, last.Content)
	}
}

// TestTaskRowSessionIDUniqueAcrossSessions guards /new semantics: a fresh
// session must produce a fresh task row, not reuse the old one.
func TestTaskRowSessionIDUniqueAcrossSessions(t *testing.T) {
	fm := &fakeModel{turns: []fakeTurn{
		{final: modelText("one")},
		{final: modelText("two")},
	}}
	store, err := storage.NewSQLiteStoreInMemory()
	if err != nil {
		t.Fatalf("sqlite store: %v", err)
	}
	a, err := NewWithModel(context.Background(), Options{
		Model: "fake", Tools: testRegistry(t, ""), Store: store,
	}, fm)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	cb := newRecordingCallback()
	ctx := context.Background()
	if _, err := a.RunWithCallback(ctx, "first", cb); err != nil {
		t.Fatalf("run 1: %v", err)
	}
	firstTask, firstSess := a.TaskID(), a.SessionID()

	if err := a.NewSession(ctx); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if _, err := a.RunWithCallback(ctx, "second", cb); err != nil {
		t.Fatalf("run 2: %v", err)
	}
	if a.TaskID() == firstTask {
		t.Fatal("new session must create a new task row")
	}
	if a.SessionID() == firstSess {
		t.Fatal("new session must have a new session id")
	}
	s1, _ := store.GetTaskSessionID(firstTask)
	if s1 != firstSess {
		t.Fatalf("task 1 session mapping changed: %q != %q", s1, firstSess)
	}

}
