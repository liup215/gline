package adkagent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/session/compaction"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/genai"

	glineagent "github.com/liup215/gline/internal/agent" // StreamCallback interface
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/prompts"
	"github.com/liup215/gline/internal/memory"
	"github.com/liup215/gline/internal/provider"
	"github.com/liup215/gline/internal/storage"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
)

const (
	// AppName and UserID identify gline sessions inside the ADK session
	// service. They must match the values used by internal/sessionstore.
	AppName = "gline"
	UserID  = "local"

	ModePlan = "plan"
	ModeAct  = "act"
)

// legacyToolNames are registry entries kept for the old hand-written loop
// that the ADK loop must NOT advertise: attempt_completion and
// plan_mode_respond are replaced by the model's natural turn-ending text,
// and the use_mcp_tool/access_mcp_resource wrappers are superseded by
// MCP tools registered directly into the registry.
var legacyToolNames = map[string]bool{
	string(types.ToolAttemptCompletion): true,
	string(types.ToolPlanModeRespond):   true,
	string(types.ToolUseMcpTool):        true,
	string(types.ToolAccessMcpResource): true,
}

// Options configures a new Agent.
type Options struct {
	// Provider is the LLM provider name: opencode, openai, volcano,
	// openrouter, anthropic.
	Provider string
	// Model is the model identifier passed to the provider.
	Model string
	// APIKey authenticates the provider.
	APIKey string
	// BaseURL overrides the provider endpoint (optional).
	BaseURL string
	// ThinkingLevel adjusts provider reasoning effort (optional).
	ThinkingLevel string

	// Tools is the gline tool registry. Nil uses tools.DefaultRegistry.
	Tools *tools.Registry

	// WorkingDir is shown in the system instruction (optional).
	WorkingDir string

	// Skills lists available skills for the SKILLS section of the system
	// instruction (optional). The model activates one via use_skill.
	Skills []types.SkillMeta

	// Mode is the initial mode: "plan" or "act" (default act).
	Mode string

	// Yolo is deprecated as an option: the agent auto-approves all tools by
	// default (user preference). Kept for API compatibility; SetYolo can
	// still re-enable confirmation prompts at runtime.
	Yolo bool

	// SessionService stores conversation sessions. Nil uses an in-memory
	// service (tests); production callers pass internal/sessionstore.Open.
	SessionService session.Service

	// SessionID resumes an existing session when non-empty.
	SessionID string

	// Store is the gline task index (SQLite). When set, the agent creates
	// a task on the first turn, stores the session ID mapping, and persists
	// the transcript so history browsing keeps working. Nil skips all task
	// bookkeeping (tests).
	Store storage.Store

	// Compaction enables ADK context compaction for this agent's sessions
	// (tail retention and/or sliding window). Nil disables compaction.
	// The default summarizer runs on the agent's own model.
	Compaction *compaction.Config

	// MemoryEngine enables background fact extraction after a completed
	// run (facts layer only; wiki/RAG wiring stays with the legacy loop
	// until Phase 7 removes it). Nil skips fact extraction.
	MemoryEngine *memory.UnifiedEngine
}

// Agent drives the ADK-backed agent loop. It owns one root LLMAgent, one
// runner, and one ADK session; RunWithCallback feeds model events into
// gline's StreamCallback interface so existing UIs keep working.
type Agent struct {
	opts       Options
	llm        model.LLM
	rootAgent  agent.Agent
	runner     *runner.Runner
	sessionSvc session.Service
	sessionID  string

	mu      sync.Mutex
	mode    string
	yolo    bool
	cb      glineagent.StreamCallback // active callback during a run
	cancel  context.CancelFunc
	running bool

	taskID    string // task index row (empty until the first turn creates one)
	taskTitle string

	workingDir string            // dynamic working dir for the system instruction
	skills     []types.SkillMeta // dynamic skill metadata for the system instruction
}

// New builds the LLM client, wraps the gline tool registry for ADK, creates
// the LLMAgent + runner, and opens (or resumes) a session.
func New(ctx context.Context, opts Options) (*Agent, error) {
	if opts.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	llm, err := provider.NewLLM(ctx, opts.Provider, opts.Model, opts.APIKey, opts.BaseURL, opts.ThinkingLevel, nil)
	if err != nil {
		return nil, fmt.Errorf("creating LLM client: %w", err)
	}
	return NewWithModel(ctx, opts, llm)
}

// NewWithModel is New with an explicit model.LLM — the injection point for
// tests (scripted fakes) and future provider swaps.
func NewWithModel(ctx context.Context, opts Options, llm model.LLM) (*Agent, error) {
	if opts.Model == "" {
		opts.Model = llm.Name()
	}
	if opts.Mode != ModePlan && opts.Mode != ModeAct {
		opts.Mode = ModeAct
	}

	sessionSvc := opts.SessionService
	if sessionSvc == nil {
		sessionSvc = session.InMemoryService()
	}

	a := &Agent{
		opts:       opts,
		llm:        llm,
		sessionSvc: sessionSvc,
		mode:       opts.Mode,
		// Yolo defaults to true: all tools run without permission prompts.
		// Runtime can still re-enable prompts via SetYolo(false).
		yolo:       true,
		workingDir: opts.WorkingDir,
		skills:     opts.Skills,
	}

	adkTools, err := a.buildTools()
	if err != nil {
		return nil, err
	}

	rootAgent, err := llmagent.New(llmagent.Config{
		Name:                AppName,
		Description:         "gline, an AI coding assistant",
		Model:               llm,
		InstructionProvider: a.instructionProvider,
		Tools:               adkTools,
		BeforeToolCallbacks: []llmagent.BeforeToolCallback{a.planGate},
	})
	if err != nil {
		return nil, fmt.Errorf("creating LLM agent: %w", err)
	}
	a.rootAgent = rootAgent

	r, err := runner.New(runner.Config{
		AppName:        AppName,
		Agent:          rootAgent,
		SessionService: sessionSvc,
		Compaction:     opts.Compaction,
	})
	if err != nil {
		return nil, fmt.Errorf("creating runner: %w", err)
	}
	a.runner = r

	if err := a.openSession(ctx); err != nil {
		return nil, err
	}
	return a, nil
}

// buildTools wraps the gline registry in ADK tool adapters, skipping
// legacy-loop-only entries.
func (a *Agent) buildTools() ([]tool.Tool, error) {
	registry := a.opts.Tools
	if registry == nil {
		registry = tools.DefaultRegistry
	}
	var out []tool.Tool
	for _, info := range registry.GetAllInfo() {
		if legacyToolNames[string(info.Tool.Name())] {
			continue
		}
		out = append(out, newADKTool(info.Tool, info, a))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no tools registered")
	}
	return out, nil
}

// openSession creates a fresh ADK session or resumes opts.SessionID.
func (a *Agent) openSession(ctx context.Context) error {
	if a.opts.SessionID != "" {
		resp, err := a.sessionSvc.Get(ctx, &session.GetRequest{
			AppName: AppName, UserID: UserID, SessionID: a.opts.SessionID,
		})
		if err != nil {
			return fmt.Errorf("resuming session %s: %w", a.opts.SessionID, err)
		}
		a.sessionID = resp.Session.ID()
		return nil
	}
	resp, err := a.sessionSvc.Create(ctx, &session.CreateRequest{
		AppName: AppName, UserID: UserID,
	})
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	a.sessionID = resp.Session.ID()
	return nil
}

// instructionProvider is the dynamic system instruction. It runs on every
// model call and reads the live mode, which is how Plan/Act switching takes
// effect without rebuilding the agent.
func (a *Agent) instructionProvider(ctx agent.ReadonlyContext) (string, error) {
	return a.SystemInstruction(), nil
}

// SystemInstruction assembles the prompt for the current mode.
func (a *Agent) SystemInstruction() string {
	mode := a.Mode()
	var b strings.Builder
	b.WriteString(`You are gline, an AI coding assistant. You help users with software engineering tasks.

You have access to tools for file operations, code search, and command execution.
Use tools to act; do not just describe what you would do. After each tool call,
wait for its result before proceeding.

When the task is complete, simply reply with a concise summary as plain text.
`)
	if mode == ModePlan {
		b.WriteString(`
# Mode: Plan

You are in Plan Mode. Explore the codebase and present a plan as plain text.
Write tools (write, edit, run) are disabled; do not attempt them.
`)
	} else {
		b.WriteString(`
# Mode: Act

You are in Act Mode. Execute tasks by reading, writing, and modifying files.
`)
	}
	if wd := a.workingDirLocked(); wd != "" {
		fmt.Fprintf(&b, "\nCurrent working directory: %s\n", wd)
	}
	// Custom rules (~/.gline/rules/*.md|txt and ./.gline/rules/*.md|txt).
	// Loaded on every call: the instruction provider already runs per model
	// call (that is how Plan/Act switching works), the files are small, and
	// re-reading keeps rule edits live without an agent rebuild.
	if rules, err := prompts.LoadCustomRules(); err == nil && rules != "" {
		b.WriteString("\n")
		b.WriteString(rules)
		b.WriteString("\n")
	}
	if skills := a.skillsLocked(); len(skills) > 0 {
		b.WriteString("\n# Skills\n\n")
		b.WriteString(`Specialized instructions are available as skills. When the user's request matches one, activate it with the use_skill tool (once per task), then follow its instructions.

`)
		for _, s := range skills {
			fmt.Fprintf(&b, "- %s: %s\n", s.Name, s.Description)
		}
	}
	return b.String()
}

// workingDirLocked returns the current working directory shown in the
// system instruction. Callers must not hold a.mu.
func (a *Agent) workingDirLocked() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.workingDir
}

// skillsLocked returns the current skill metadata. Callers must not hold a.mu.
func (a *Agent) skillsLocked() []types.SkillMeta {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.skills
}

// SessionID returns the ADK session this agent runs against.
func (a *Agent) SessionID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sessionID
}

// ProviderInfo reports the provider and model names for status display.
func (a *Agent) ProviderInfo() (providerName, modelName string) {
	return a.opts.Provider, a.opts.Model
}

// NewSession drops the current conversation context: the old ADK session is
// deleted and a fresh one is created, so the next run starts clean. Used by
// /clear and /new-task.
func (a *Agent) NewSession(ctx context.Context) error {
	a.mu.Lock()
	old := a.sessionID
	a.mu.Unlock()
	if old != "" {
		_ = a.sessionSvc.Delete(ctx, &session.DeleteRequest{
			AppName: AppName, UserID: UserID, SessionID: old,
		})
	}
	resp, err := a.sessionSvc.Create(ctx, &session.CreateRequest{
		AppName: AppName, UserID: UserID,
	})
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	a.mu.Lock()
	a.sessionID = resp.Session.ID()
	// A new conversation is a new task: forget the previous index row so
	// the next turn creates a fresh one linked to the new session.
	a.taskID = ""
	a.taskTitle = ""
	a.mu.Unlock()
	return nil
}

// Mode returns the current Plan/Act mode.
func (a *Agent) Mode() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mode
}

// SetMode switches between plan and act. The next model call picks up the
// new instruction; the planGate callback starts blocking writes immediately.
func (a *Agent) SetMode(mode string) {
	if mode != ModePlan && mode != ModeAct {
		return
	}
	a.mu.Lock()
	a.mode = mode
	a.mu.Unlock()
}

// Yolo reports whether tool confirmations are auto-approved.
func (a *Agent) Yolo() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.yolo
}

// SetYolo toggles auto-approval of confirmation-gated tools.
func (a *Agent) SetYolo(y bool) {
	a.mu.Lock()
	a.yolo = y
	a.mu.Unlock()
}

// IsRunning reports whether a run is in flight.
func (a *Agent) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

// SetTaskID attaches an existing task index row to this conversation
// (history resume). Implements the TUI's taskManager contract.
func (a *Agent) SetTaskID(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.taskID = id
}

// SetTaskTitle names the task record created on the next first turn.
func (a *Agent) SetTaskTitle(title string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.taskTitle = title
}

// ResetTask forgets the current task so the next turn creates a fresh one.
func (a *Agent) ResetTask() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.taskID = ""
	a.taskTitle = ""
}

// TaskID returns the current task index row, if any.
func (a *Agent) TaskID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.taskID
}

// SetWorkingDir updates the working directory shown in the system
// instruction (effective on the next invocation) and used for future
// task records. The sandbox/tools follow the process cwd, which callers
// change separately.
func (a *Agent) SetWorkingDir(dir string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.workingDir = dir
}

// SetSkills replaces the skill metadata rendered in the system
// instruction (effective on the next invocation).
func (a *Agent) SetSkills(meta []types.SkillMeta) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.skills = meta
}

// MemoryEngine returns the configured memory engine (nil when absent).
func (a *Agent) MemoryEngine() *memory.UnifiedEngine {
	return a.opts.MemoryEngine
}

// ResumeSession points the agent at an existing ADK session (history
// resume). The runner keeps using the same session service, so the next
// turn continues with the full prior conversation as context.
func (a *Agent) ResumeSession(ctx context.Context, sessionID string) error {
	resp, err := a.sessionSvc.Get(ctx, &session.GetRequest{
		AppName: AppName, UserID: UserID, SessionID: sessionID,
	})
	if err != nil {
		return fmt.Errorf("resuming session %s: %w", sessionID, err)
	}
	a.mu.Lock()
	a.sessionID = resp.Session.ID()
	a.mu.Unlock()
	return nil
}

// Abort cancels the active run, if any. The runner surfaces cancellation as
// a context error through the event stream. Any tool goroutine blocked in an
// approval prompt is unblocked immediately: ADK executes a turn's tool calls
// concurrently and they ignore the run context while waiting for user input,
// so without closing their reply channels a cancel would never fully unwind.
func (a *Agent) Abort() {
	a.mu.Lock()
	cancel := a.cancel
	cb := a.cb
	a.mu.Unlock()
	if cb != nil {
		if pa, ok := cb.(interface{ AbortPendingQuestions() }); ok {
			pa.AbortPendingQuestions()
		}
	}
	if cancel != nil {
		cancel()
	}
}

// requestApproval asks the active callback for user confirmation of a tool
// call. Called from tool execution; falls back to approved when no callback
// is installed (headless tests).
func (a *Agent) requestApproval(toolName string, args map[string]any) bool {
	a.mu.Lock()
	cb := a.cb
	a.mu.Unlock()
	if cb == nil {
		return true
	}
	answer, err := cb.AskFollowupQuestion(
		fmt.Sprintf("Approve %s? %s", toolName, toolCallJSON(args)),
		[]string{"Yes", "No"},
	)
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(answer), "yes")
}

// RunWithCallback sends one user turn through the ADK loop, mapping events
// onto cb. It blocks until the run completes (all model turns and tool
// executions), returns the ADK session ID, and reports errors through cb.
// When the loop detector calls the turn dead (*stuckError), the reason is
// handed back to the model as a fresh instruction and the turn re-runs, up to
// maxStuckRecoveries times (pi-go's tell-the-model-and-resume pattern).
func (a *Agent) RunWithCallback(ctx context.Context, prompt string, cb glineagent.StreamCallback) (string, error) {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return a.sessionID, fmt.Errorf("agent is already running")
	}
	a.running = true
	runCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	a.cb = cb
	a.mu.Unlock()
	defer func() {
		cancel()
		// Unblock any tool goroutine still waiting on an approval prompt so
		// abandoned questions can never wedge a finished run.
		if pa, ok := cb.(interface{ AbortPendingQuestions() }); ok {
			pa.AbortPendingQuestions()
		}
		a.mu.Lock()
		a.running = false
		a.cancel = nil
		a.cb = nil
		a.mu.Unlock()
	}()

	for attempt := 0; ; attempt++ {
		acc := &transcriptAccumulator{}
		err := a.streamTurn(runCtx, prompt, cb, acc)
		defer a.persistTranscript(acc)
		var stuck *stuckError
		if !errors.As(err, &stuck) {
			if err == nil {
				cb.OnComplete()
				a.finishTask("completed")
				a.extractFactsAsync(transcriptFromMessages(acc.messages()))
			} else {
				a.finishTask("failed")
			}
			return a.sessionID, err
		}
		if attempt >= maxStuckRecoveries {
			gaveUp := fmt.Errorf("%w (gave up after %d recovery attempt(s))", err, attempt)
			cb.OnError(gaveUp)
			return a.sessionID, gaveUp
		}
		log.Infof("agent loop stuck (%s); telling the model and resuming (attempt %d of %d)",
			stuck.detail, attempt+1, maxStuckRecoveries)
		prompt = recoverStuckPrompt(stuck.detail)
	}
}

// maxStuckRecoveries is how many times a stuck turn is handed back to the
// model with the detector's reason before the run is ended for real. Each
// recovery costs a full model round-trip, so the bound stays small.
const maxStuckRecoveries = 2

// streamTurn runs one user turn through the runner and feeds every event into
// the bridge and the transcript accumulator. It returns a *stuckError when the
// loop detector fires; any other error has already been surfaced through cb,
// and nil means a clean finish.
func (a *Agent) streamTurn(runCtx context.Context, prompt string, cb glineagent.StreamCallback, acc *transcriptAccumulator) error {
	// Per-turn cancellation: a stuck verdict kills this turn's event stream
	// without tearing down the run context a recovery attempt needs.
	turnCtx, turnCancel := context.WithCancel(runCtx)
	defer turnCancel()

	// The user prompt opens this turn's transcript (the ADK runner stores it
	// in the session but never yields it as an event).
	acc.addUserPrompt(prompt)

	// Task bookkeeping: create the index row on the first turn of the
	// conversation, mirroring the legacy agent's behaviour.
	if a.opts.Store != nil && a.taskID == "" {
		prov, mdl := a.ProviderInfo()
		id, err := a.opts.Store.CreateTask(a.taskTitle, prompt, a.Mode(), prov, mdl, a.workingDir)
		if err != nil {
			log.Warnf("failed to create task record: %v", err)
		} else {
			a.taskID = id
			cb.OnTaskCreated(id)
			// Link the task to the ADK session for history resume.
			if err := a.opts.Store.SetTaskSessionID(id, a.sessionID); err != nil {
				log.Warnf("failed to link task to session: %v", err)
			}
		}
	}

	msg := genai.NewContentFromText(prompt, genai.RoleUser)
	events := a.runner.Run(turnCtx, UserID, a.sessionID, msg, agent.RunConfig{
		StreamingMode: agent.StreamingModeSSE,
	})

	bridge := newEventBridge(cb)
	for ev, err := range events {
		if err != nil {
			if runCtx.Err() != nil {
				err = fmt.Errorf("aborted: %w", runCtx.Err())
			}
			cb.OnError(err)
			return err
		}
		if ev == nil {
			continue
		}
		acc.addEvent(ev)
		// Provider-level failures ride events with a nil Go error.
		if evErr := EventError(ev); evErr != nil {
			cb.OnError(evErr)
			return evErr
		}
		evErr := bridge.deliver(ev)
		if evErr != nil {
			// Loop detector fired: stop this turn's stream; the caller
			// decides between recovery and giving up.
			turnCancel()
			return evErr
		}
	}
	return nil
}

// recoverStuckPrompt is the instruction handed to the model after the loop
// detector called the previous turn dead. It names the reason and forbids
// continuing the dead pattern (ported from pi-go).
func recoverStuckPrompt(detail string) string {
	return fmt.Sprintf(
		`Your previous turn was stopped automatically: %s.

That is a loop, not progress, and repeating it will stop the turn again. Do not
continue where you left off and do not restate what you already said.

Change approach:
- If you were repeating text, say the point once and move on.
- If you were repeating a tool call, the call is not going to start working —
  use a different tool, different arguments, or reason from what you already have.
- If you are genuinely blocked, say so plainly and stop, rather than filling
  the turn.

Continue the original task from here.`, detail)
}
