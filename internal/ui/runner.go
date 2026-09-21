package ui

import (
	"context"

	glineagent "github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/adkagent"
)

// AgentRunner is the agent functionality the TUI depends on. It is
// implemented by adapters around both the new ADK-backed agent
// (adkagent.Agent) and the legacy hand-written loop (agent.BaseAgent);
// GLINE_AGENT=legacy selects the latter at assembly time.
type AgentRunner interface {
	// RunWithCallback sends one user turn; it blocks until the run ends.
	RunWithCallback(ctx context.Context, prompt string, cb glineagent.StreamCallback) error
	// Abort cancels the active run, if any.
	Abort()
	// SetMode switches plan/act.
	SetMode(mode string)
	// Mode reports the current mode.
	Mode() string
	// ProviderInfo reports provider and model names for the status bar.
	ProviderInfo() (providerName, modelName string)
	// SessionID reports the current session identifier ("" if none).
	SessionID() string
	// NewSession drops the conversation context; the next run starts fresh.
	NewSession(ctx context.Context) error
}

// taskManager is the optional task-index functionality only the legacy
// agent provides. The TUI uses it when available and skips it otherwise.
type taskManager interface {
	SetTaskID(id string)
	SetTaskTitle(title string)
	ResetTask()
}

// adkRunner adapts *adkagent.Agent to AgentRunner.
type adkRunner struct{ *adkagent.Agent }

func (a adkRunner) RunWithCallback(ctx context.Context, prompt string, cb glineagent.StreamCallback) error {
	_, err := a.Agent.RunWithCallback(ctx, prompt, cb)
	return err
}

// legacyRunner adapts the legacy *agent.BaseAgent to AgentRunner.
type legacyRunner struct{ *glineagent.BaseAgent }

func (l legacyRunner) ProviderInfo() (string, string) {
	if p := l.GetProvider(); p != nil {
		return p.GetProviderName(), p.GetModel()
	}
	return "", ""
}

func (l legacyRunner) Mode() string { return string(l.GetMode()) }

func (l legacyRunner) SessionID() string { return "" }

func (l legacyRunner) NewSession(ctx context.Context) error {
	l.GetConversation().Clear()
	return nil
}

func (l legacyRunner) SetMode(mode string) { _ = l.BaseAgent.SetMode(glineagent.Mode(mode)) }

// LegacyRunner wraps a legacy *agent.BaseAgent as an AgentRunner.
func LegacyRunner(base *glineagent.BaseAgent) AgentRunner { return legacyRunner{base} }

// AdkRunner wraps an *adkagent.Agent as an AgentRunner.
func AdkRunner(a *adkagent.Agent) AgentRunner { return adkRunner{a} }

// compile-time checks
var (
	_ AgentRunner = adkRunner{}
	_ AgentRunner = legacyRunner{}
	_ taskManager = legacyRunner{}
)
