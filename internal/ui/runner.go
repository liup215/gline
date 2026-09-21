package ui

import (
	"context"

	glineagent "github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/adkagent"
	"github.com/liup215/gline/internal/memory"
	"github.com/liup215/gline/pkg/types"
)

// AgentRunner is the agent functionality the TUI depends on. The ADK-backed
// agent (adkagent.Agent) is the only implementation; the legacy hand-written
// loop was removed in Phase 7.
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

// taskManager is the optional task-index functionality. The TUI uses it
// when available and skips it otherwise.
type taskManager interface {
	SetTaskID(id string)
	SetTaskTitle(title string)
	ResetTask()
}

// The capability interfaces below are optional agent features consumed via
// type assertion (mainly by the GUI ChatService). Callers must degrade
// gracefully when a capability is absent.

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

// ResumeSessionResumer points the agent at an existing recorded session
// (history resume).
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

// AdkRunner wraps an *adkagent.Agent as an AgentRunner.
func AdkRunner(a *adkagent.Agent) AgentRunner { return adkRunner{a} }

// compile-time checks
var (
	_ AgentRunner          = adkRunner{}
	_ taskManager          = adkRunner{}
	_ WorkingDirSetter     = adkRunner{}
	_ TaskIDProvider       = adkRunner{}
	_ TaskResetter         = adkRunner{}
	_ SkillsSetter         = adkRunner{}
	_ MemoryProvider       = adkRunner{}
	_ ResumeSessionResumer = adkRunner{}
)
