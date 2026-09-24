package gui

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/config"
	"github.com/liup215/gline/internal/memory"
	"github.com/liup215/gline/pkg/types"
)

// fakeAgent implements ui.AgentRunner plus the capability interfaces the
// ChatService asserts, recording calls so tests can verify routing.
type fakeAgent struct {
	mode        string
	workingDir  string
	taskID      string
	resetCount  int
	newSessions int
}

func (f *fakeAgent) RunWithCallback(ctx context.Context, prompt string, cb agent.StreamCallback) error {
	return nil
}
func (f *fakeAgent) Abort()                                     {}
func (f *fakeAgent) SetMode(mode string)                        { f.mode = mode }
func (f *fakeAgent) Mode() string                               { return f.mode }
func (f *fakeAgent) ProviderInfo() (string, string)             { return "fake", "fake-model" }
func (f *fakeAgent) SessionID() string                          { return "" }
func (f *fakeAgent) NewSession(ctx context.Context) error       { f.newSessions++; return nil }
func (f *fakeAgent) SetWorkingDir(dir string)                   { f.workingDir = dir }
func (f *fakeAgent) GetTaskID() string                          { return f.taskID }
func (f *fakeAgent) ResetTask()                                 { f.resetCount++ }
func (f *fakeAgent) SetSkills(meta []types.SkillMeta)           {}
func (f *fakeAgent) MemoryEngine() *memory.UnifiedEngine        { return nil }
func (f *fakeAgent) ResumeSession(ctx context.Context, s string) error {
	return errors.New("not implemented")
}

func newChatService(t *testing.T) *ChatService {
	t.Helper()
	// Isolate config loading from the real user profile.
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	cfg := config.NewManager()
	if err := cfg.Load(); err != nil {
		t.Fatalf("config load: %v", err)
	}
	return &ChatService{Backend: &Backend{ag: &fakeAgent{mode: "act"}, cfg: cfg}}
}

func TestChatServiceModeRouting(t *testing.T) {
	c := newChatService(t)
	if got := c.GetMode(); got != "act" {
		t.Fatalf("initial mode = %q, want act", got)
	}
	if err := c.SetMode("plan"); err != nil {
		t.Fatalf("SetMode(plan): %v", err)
	}
	if got := c.GetMode(); got != "plan" {
		t.Fatalf("mode after SetMode = %q, want plan", got)
	}
	if err := c.SetMode("bogus"); err == nil {
		t.Fatal("SetMode(bogus) should error")
	}
}

func TestChatServiceStartNewConversation(t *testing.T) {
	c := newChatService(t)
	ag := c.Backend.ag.(*fakeAgent)
	ag.workingDir = "/tmp/old"
	c.StartNewConversation()
	if ag.resetCount != 1 {
		t.Fatalf("ResetTask called %d times, want 1", ag.resetCount)
	}
	if ag.newSessions != 1 {
		t.Fatalf("NewSession called %d times, want 1", ag.newSessions)
	}
	if ag.workingDir != "" {
		t.Fatalf("workingDir = %q, want empty", ag.workingDir)
	}
	if c.workingDir != "" {
		t.Fatalf("service workingDir = %q, want empty", c.workingDir)
	}
}

func TestChatServiceGetStatusAndConversationState(t *testing.T) {
	c := newChatService(t)
	status, err := c.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status["mode"] != "act" {
		t.Fatalf("status mode = %q, want act", status["mode"])
	}
	if status["currentTokens"] == "" {
		t.Fatal("status currentTokens empty")
	}
	state := c.GetConversationState()
	var views []map[string]string
	if err := json.Unmarshal([]byte(state), &views); err != nil {
		t.Fatalf("GetConversationState not valid JSON: %v (%q)", err, state)
	}
	if len(views) != 0 {
		t.Fatalf("expected empty transcript without a task, got %d entries", len(views))
	}
}

func TestChatServiceCompactConversationIsNoop(t *testing.T) {
	c := newChatService(t)
	ok, err := c.CompactConversation()
	if err != nil || !ok {
		t.Fatalf("CompactConversation = (%v, %v), want (true, nil)", ok, err)
	}
}

func TestChatServiceReloadRulesDegrades(t *testing.T) {
	c := newChatService(t)
	if _, _, err := c.ReloadRules(); err == nil {
		t.Fatal("ReloadRules should report unavailability on the ADK agent")
	}
}

func TestChatServiceClearConversation(t *testing.T) {
	c := newChatService(t)
	ag := c.Backend.ag.(*fakeAgent)
	ag.workingDir = "/tmp/keep"
	c.ClearConversation()
	if ag.resetCount != 1 {
		t.Fatalf("ResetTask called %d times, want 1", ag.resetCount)
	}
	if ag.workingDir != "/tmp/keep" {
		t.Fatalf("workingDir = %q, want preserved", ag.workingDir)
	}
}


// TestStreamSeqMonotonicAcrossRuns guards the 2026-09-24 "GUI froze on 'AI is
// thinking...'" regression: the frontend's ordered dispatcher keeps a single
// monotonic `expected` counter for the app lifetime, so the Go-side seq must
// NOT restart at 0 for each SendMessage. A per-run counter made every event of
// the second and later messages have seq < expected, so the dispatcher
// swallowed them as stale duplicates — nothing rendered while the agent run
// completed and persisted normally (DB had the full answer).
func TestStreamSeqMonotonicAcrossRuns(t *testing.T) {
	c := newChatService(t)

	// Two callbacks created the way SendMessage does (one per run) must draw
	// from one shared, strictly increasing counter.
	cb1 := c.newChatRunCallback()
	s1 := cb1.nextSeq()
	s2 := cb1.nextSeq()
	cb2 := c.newChatRunCallback()
	s3 := cb2.nextSeq()

	if !(s1 < s2 && s2 < s3) {
		t.Fatalf("seq not strictly increasing across runs: %d, %d, %d", s1, s2, s3)
	}
	if s3 <= s1 {
		t.Fatal("second run reused seq numbers; frontend ordered dispatcher would swallow run 2")
	}
}
