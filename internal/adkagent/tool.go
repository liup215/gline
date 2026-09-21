package adkagent

import (
	"encoding/json"
	"fmt"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/toolutils"
	"google.golang.org/genai"

	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
)

// adkTool adapts a gline tools.Tool to ADK's tool.Tool interface so the
// registry's implementations (file ops, run, search, MCP, skills) run
// unmodified inside the ADK loop.
//
// ADK's llm flow requires every tool to implement toolinternal.RequestProcessor
// (ProcessRequest) to pack its declaration into the LLM request; toolutils.
// PackTool does exactly what functiontool does internally.
type adkTool struct {
	inner tools.Tool
	info  *tools.ToolInfo
	a     *Agent
}

func newADKTool(inner tools.Tool, info *tools.ToolInfo, a *Agent) *adkTool {
	return &adkTool{inner: inner, info: info, a: a}
}

func (t *adkTool) Name() string        { return t.inner.Name() }
func (t *adkTool) Description() string { return t.inner.Description() }
func (t *adkTool) IsLongRunning() bool { return false }

// Declaration builds the genai function declaration from the gline tool's
// JSON schema. The schema arrives as json.RawMessage, so it is decoded into
// plain any (map) — genai.FunctionDeclaration.ParametersJsonSchema is typed
// as any and is serialized verbatim into the provider request.
func (t *adkTool) Declaration() *genai.FunctionDeclaration {
	decl := &genai.FunctionDeclaration{
		Name:        t.Name(),
		Description: t.Description(),
	}
	if raw := t.inner.InputSchema(); len(raw) > 0 {
		var schema any
		if err := json.Unmarshal(raw, &schema); err == nil {
			decl.ParametersJsonSchema = schema
		}
	}
	return decl
}

// ProcessRequest packs this tool's declaration into the LLM request.
func (t *adkTool) ProcessRequest(ctx agent.Context, req *model.LLMRequest) error {
	return toolutils.PackTool(req, t)
}

// Run executes the gline tool. Args arrive as a map (decoded from the
// model's function call); they are re-encoded to JSON for the gline tool's
// Execute(json.RawMessage) signature. The string result is wrapped as
// {"result": ...}, the shape ADK serializes into the FunctionResponse.
func (t *adkTool) Run(ctx agent.Context, args any) (map[string]any, error) {
	m, ok := args.(map[string]any)
	if !ok || m == nil {
		m = map[string]any{}
	}
	// Repair model-produced args before execution: parameter aliases
	// (file_path → path) and type coercion ("3" → 3). See tools.RepairToolArgs.
	m = tools.RepairToolArgs(t.Name(), t.inner.InputSchema(), m)

	// Route ask_followup_question through the active run's StreamCallback
	// (same path as tool approvals) so the question surfaces in the TUI
	// option picker, GUI dialog, or CLI prompt. Without this the tool falls
	// back to reading os.Stdin directly, which competes with Bubbletea for
	// stdin, garbles the input area, and never completes. Re-wire per call:
	// the bridge is created per run, one run at a time, so overwriting the
	// handler with the current cb is idempotent within a run.
	if t.Name() == string(types.ToolAskFollowupQuestion) {
		if asker, ok := t.inner.(*tools.AskFollowupQuestionTool); ok {
			t.a.mu.Lock()
			cb := t.a.cb
			t.a.mu.Unlock()
			if cb != nil {
				asker.SetHandler(cb.AskFollowupQuestion)
			}
		}
	}

	raw, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("tool %q: encoding args: %w", t.Name(), err)
	}

	// Confirmation gate: interactive tools (and anything the registry marks
	// as requiring confirmation) need user approval unless yolo mode is on.
	if t.info != nil && t.info.RequiresConfirmation && !t.a.Yolo() {
		if approved := t.a.requestApproval(t.Name(), m); !approved {
			return map[string]any{
				"result": fmt.Sprintf("User declined to run %s. Do not retry the same call; ask the user how to proceed.", t.Name()),
			}, nil
		}
	}

	out, err := t.inner.Execute(ctx, raw)
	if err != nil {
		return nil, err
	}
	return map[string]any{"result": out}, nil
}

// planGate is the BeforeToolCallback enforcing Plan mode: tools the registry
// does not allow in plan mode (write, edit, run) are blocked with a text
// result so the model can read the reason and stay in exploration. Returning
// (result, nil) short-circuits the real tool execution.
func (a *Agent) planGate(ctx agent.Context, tool tool.Tool, args map[string]any) (map[string]any, error) {
	if a.Mode() != ModePlan || a.opts.Tools == nil {
		return nil, nil
	}
	if a.opts.Tools.IsAllowed(ModePlan, tool.Name()) {
		return nil, nil
	}
	return map[string]any{
		"result": fmt.Sprintf(
			"Tool %q is blocked in Plan mode. Plan mode is read-only: explore the codebase and present a plan. "+
				"Switch to Act mode to make changes.", tool.Name(),
		),
	}, nil
}

// toolCallJSON renders tool args as compact JSON for callbacks (bounded).
func toolCallJSON(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	b, err := json.Marshal(args)
	if err != nil {
		return ""
	}
	s := string(b)
	const maxLen = 2048
	if len(s) > maxLen {
		s = s[:maxLen] + "…"
	}
	return s
}
