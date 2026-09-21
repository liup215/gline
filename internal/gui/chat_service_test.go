package gui

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/api"
	"github.com/liup215/gline/internal/config"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/internal/ui"
	"github.com/liup215/gline/pkg/types"
)

// newLegacyChatService builds a ChatService wired to a legacy BaseAgent
// (offline provider; no LLM calls are made) to verify the capability-based
// agent routing in the service layer.
func newLegacyChatService(t *testing.T) *ChatService {
	t.Helper()
	// Isolate config loading from the real user profile.
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	cfg := config.NewManager()
	if err := cfg.Load(); err != nil {
		t.Fatalf("config load: %v", err)
	}
	prov := api.NewOpenAIProvider("test-key", "gpt-4o-mini", "")
	registry := tools.InitDefaultRegistry(nil, nil)
	base, err := agent.New(agent.Options{
		Provider:     prov,
		ToolRegistry: registry,
		Mode:         agent.ModeAct,
		AutoApprove:  true,
	})
	if err != nil {
		t.Fatalf("agent.New: %v", err)
	}
	return &ChatService{Backend: &Backend{ag: ui.LegacyRunner(base), cfg: cfg}}
}

func TestChatServiceModeRoutingLegacy(t *testing.T) {
	c := newLegacyChatService(t)
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

func TestChatServiceStartNewConversationLegacy(t *testing.T) {
	c := newLegacyChatService(t)
	conv := c.Backend.ag.(ui.ConversationProvider).GetConversation()
	conv.AddMessage(types.Message{Role: "user", Content: "hello"})
	c.StartNewConversation()
	if msgs := conv.GetMessages(); len(msgs) != 0 {
		t.Fatalf("conversation not cleared: %d messages", len(msgs))
	}
	if c.workingDir != "" {
		t.Fatalf("workingDir = %q, want empty", c.workingDir)
	}
}

func TestChatServiceGetStatusAndConversationStateLegacy(t *testing.T) {
	c := newLegacyChatService(t)
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
}

func TestChatServiceCompactConversationLegacy(t *testing.T) {
	c := newLegacyChatService(t)
	if _, err := c.CompactConversation(); err != nil {
		t.Fatalf("CompactConversation: %v", err)
	}
}

func TestChatServiceNewSessionIsImplemented(t *testing.T) {
	c := newLegacyChatService(t)
	if err := c.Backend.ag.NewSession(context.Background()); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
}
