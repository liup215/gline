package ui

import (
	"context"

	glineagent "github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/adkagent"
	"github.com/liup215/gline/internal/memory"
	"github.com/liup215/gline/internal/prompts"
	"github.com/liup215/gline/pkg/types"
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

// The capability interfaces below are optional agent features consumed via
// type assertion (mainly by the GUI ChatService). The ADK agent implements
// the task/working-dir/skills/memory set; the legacy agent implements all.
// Callers must degrade gracefully when a capability is absent.

// WorkingDirSetter updates the agent's notion of the project directory.
type WorkingDirSetter interface {
	SetWorkingDir(dir string)
}

// TaskIDProvider reports the current task index row.
type TaskIDProvider interface {
	GetTaskID() string
}

// TaskResetter detaches the current task so the next turn starts a new one.
type TaskResetter interface {
	ResetTask()
}

// SkillsSetter replaces the skill metadata rendered into the prompt.
type SkillsSetter interface {
	SetSkills(meta []types.SkillMeta)
}

// MemoryProvider exposes the unified memory engine (nil when absent).
type MemoryProvider interface {
	MemoryEngine() *memory.UnifiedEngine
}

// ConversationProvider exposes the legacy in-memory conversation. The ADK
// agent keeps its transcript in the ADK session + storage, so it does not
// implement this; use the storage-backed views instead.
type ConversationProvider interface {
	GetConversation() *types.Conversation
}

// Compactor manually compacts the legacy conversation. The ADK agent
// compacts automatically (tail retention) and does not implement this.
type Compactor interface {
	Compact() bool
}

// RulesReloader reloads custom rule files into the legacy system prompt.
type RulesReloader interface {
	ReloadCustomRules() (bool, []prompts.RuleFileInfo, error)
}

// ResumeSessionResumer points the agent at an existing recorded session
// (history resume). Only the ADK agent implements it.
type ResumeSessionResumer interface {
	ResumeSession(ctx context.Context, sessionID string) error
}

// adkRunner adapts *adkagent.Agent to AgentRunner.
type adkRunner struct{ *adkagent.Agent }

func (a adkRunner) RunWithCallback(ctx context.Context, prompt string, cb glineagent.StreamCallback) error {
	_, err := a.Agent.RunWithCallback(ctx, prompt, cb)
	return err
}

func (a adkRunner) GetTaskID() string { return a.Agent.TaskID() }

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
	_ AgentRunner          = adkRunner{}
	_ AgentRunner          = legacyRunner{}
	_ taskManager          = legacyRunner{}
	_ taskManager          = adkRunner{}
	_ WorkingDirSetter     = adkRunner{}
	_ WorkingDirSetter     = legacyRunner{}
	_ TaskIDProvider       = adkRunner{}
	_ TaskIDProvider       = legacyRunner{}
	_ TaskResetter         = adkRunner{}
	_ TaskResetter         = legacyRunner{}
	_ SkillsSetter         = adkRunner{}
	_ SkillsSetter         = legacyRunner{}
	_ MemoryProvider       = adkRunner{}
	_ ConversationProvider = legacyRunner{}
	_ Compactor            = legacyRunner{}
	_ RulesReloader        = legacyRunner{}
	_ ResumeSessionResumer = adkRunner{}
)
