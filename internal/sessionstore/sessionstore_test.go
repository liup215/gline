package sessionstore

// Smoke tests for the ADK GORM SQLite session service wrapper. These verify
// the round-trip guarantees the refactor depends on:
//   - events get IDs and survive Create/AppendEvent/Get
//   - tool call and tool response parts round-trip as a pair
//   - EventActions.Compaction survives the round trip (ADK conformance
//     requirement — a lost summary means re-summarizing every turn)

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(Options{Path: filepath.Join(t.TempDir(), "sessions.db")})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func countEvents(t *testing.T, events session.Events) int {
	t.Helper()
	n := 0
	for range events.All() {
		n++
	}
	return n
}

func TestRoundTripBasicEvents(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	svc := st.Service()

	cr, err := svc.Create(ctx, &session.CreateRequest{
		AppName: AppName, UserID: DefaultUserID, SessionID: "sess-basic-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// User input event.
	evUser := session.NewEvent(ctx, "inv-1")
	evUser.Author = "user"
	evUser.Content = &genai.Content{
		Role:  "user",
		Parts: []*genai.Part{{Text: "list the files"}},
	}
	if err := svc.AppendEvent(ctx, cr.Session, evUser); err != nil {
		t.Fatalf("AppendEvent(user) error = %v", err)
	}

	// Model turn: text + tool call.
	evModel := session.NewEvent(ctx, "inv-1")
	evModel.Author = "gline"
	evModel.LLMResponse = model.LLMResponse{
		Content: &genai.Content{
			Role: "model",
			Parts: []*genai.Part{
				{Text: "I'll list the files."},
				{FunctionCall: &genai.FunctionCall{ID: "call_1", Name: "list_dir", Args: map[string]any{"path": "."}}},
			},
		},
		FinishReason: genai.FinishReasonStop,
	}
	if err := svc.AppendEvent(ctx, cr.Session, evModel); err != nil {
		t.Fatalf("AppendEvent(model) error = %v", err)
	}

	// Tool result paired with the call above.
	evTool := session.NewEvent(ctx, "inv-1")
	evTool.Author = "gline"
	evTool.Content = &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{FunctionResponse: &genai.FunctionResponse{ID: "call_1", Name: "list_dir", Response: map[string]any{"files": []string{"a.go"}}}},
		},
	}
	if err := svc.AppendEvent(ctx, cr.Session, evTool); err != nil {
		t.Fatalf("AppendEvent(tool) error = %v", err)
	}

	// Read back and verify.
	gr, err := svc.Get(ctx, &session.GetRequest{AppName: AppName, UserID: DefaultUserID, SessionID: "sess-basic-1"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if n := gr.Session.Events().Len(); n != 3 {
		t.Fatalf("expected 3 events, got %d", n)
	}
	i := 0
	for ev := range gr.Session.Events().All() {
		if ev.ID == "" {
			t.Errorf("event %d has no ID", i)
		}
		i++
	}

	// Structural no-orphan check on the stored event log — the same invariant
	// as internal/agent's golden test, now over ADK events.
	calls := map[string]bool{}
	for ev := range gr.Session.Events().All() {
		if ev.Content == nil {
			continue
		}
		for _, p := range ev.Content.Parts {
			switch {
			case p.FunctionCall != nil:
				calls[p.FunctionCall.ID] = true
			case p.FunctionResponse != nil:
				if !calls[p.FunctionResponse.ID] {
					t.Fatalf("ORPHAN function response for call %q in stored events", p.FunctionResponse.ID)
				}
			}
		}
	}
	if !calls["call_1"] {
		t.Fatal("tool call call_1 missing from stored events")
	}
}

func TestRoundTripCompactionSurvives(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	svc := st.Service()

	cr, err := svc.Create(ctx, &session.CreateRequest{
		AppName: AppName, UserID: DefaultUserID, SessionID: "sess-comp-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// An event carrying a compaction summary in its actions.
	ev := session.NewEvent(ctx, "inv-1")
	ev.Author = "gline"
	ev.Actions.Compaction = &session.EventCompaction{
		StartTimestamp: ev.Timestamp,
		EndTimestamp:   ev.Timestamp.Add(time.Second),
		CompactedContent: &genai.Content{
			Role:  "model",
			Parts: []*genai.Part{{Text: "summary of earlier turns"}},
		},
	}
	if err := svc.AppendEvent(ctx, cr.Session, ev); err != nil {
		t.Fatalf("AppendEvent(compaction) error = %v", err)
	}

	gr, err := svc.Get(ctx, &session.GetRequest{AppName: AppName, UserID: DefaultUserID, SessionID: "sess-comp-1"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if n := countEvents(t, gr.Session.Events()); n != 1 {
		t.Fatalf("expected 1 event, got %d", n)
	}
	for ev := range gr.Session.Events().All() {
		if ev.Actions.Compaction == nil {
			t.Fatal("EventActions.Compaction did not survive the round trip")
		}
		if got := ev.Actions.Compaction.CompactedContent.Parts[0].Text; got != "summary of earlier turns" {
			t.Fatalf("compacted content = %q", got)
		}
	}
}

func TestListAndDelete(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	svc := st.Service()

	for _, id := range []string{"s-1", "s-2"} {
		if _, err := svc.Create(ctx, &session.CreateRequest{AppName: AppName, UserID: DefaultUserID, SessionID: id}); err != nil {
			t.Fatalf("Create(%s) error = %v", id, err)
		}
	}

	lr, err := svc.List(ctx, &session.ListRequest{AppName: AppName, UserID: DefaultUserID})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(lr.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(lr.Sessions))
	}

	if err := svc.Delete(ctx, &session.DeleteRequest{AppName: AppName, UserID: DefaultUserID, SessionID: "s-1"}); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	lr, err = svc.List(ctx, &session.ListRequest{AppName: AppName, UserID: DefaultUserID})
	if err != nil {
		t.Fatalf("List() after delete error = %v", err)
	}
	if len(lr.Sessions) != 1 || lr.Sessions[0].ID() != "s-2" {
		t.Fatalf("after delete expected only s-2, got %d sessions", len(lr.Sessions))
	}
}

func TestDefaultPath(t *testing.T) {
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}
	if filepath.Base(p) != "sessions.db" {
		t.Fatalf("unexpected default path %q", p)
	}
}
