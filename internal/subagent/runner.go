package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

// RunStatus represents the final status of a subagent run.
type RunStatus string

const (
	StatusCompleted RunStatus = "completed"
	StatusFailed    RunStatus = "failed"
)

// RunResult contains the outcome of a single subagent run.
type RunResult struct {
	Status       RunStatus
	Result       string
	Error        string
	InputTokens  int
	OutputTokens int
	ToolCalls    int
}

// ProgressUpdate is sent during a subagent run to report progress.
type ProgressUpdate struct {
	Status       RunStatus
	Result       string
	Error        string
	InputTokens  int
	OutputTokens int
	ToolCalls    int
	LatestTool   string
}

// maxPromptTokens bounds the accumulated subagent conversation. When the
// estimate exceeds it the run fails instead of silently truncating history.
const maxPromptTokens = 200_000

// Runner executes a single subagent task with an independent conversation loop.
type Runner struct {
	builder  *Builder
	abortReq bool
	abortMu  sync.Mutex
}

// NewRunner creates a new subagent runner.
func NewRunner(builder *Builder) *Runner {
	return &Runner{builder: builder}
}

// Abort signals the runner to stop at the next safe point.
func (r *Runner) Abort() {
	r.abortMu.Lock()
	defer r.abortMu.Unlock()
	r.abortReq = true
}

func (r *Runner) shouldAbort() bool {
	r.abortMu.Lock()
	defer r.abortMu.Unlock()
	return r.abortReq
}

// noToolsUsedMsg is injected when the assistant returns a response without tools.
const noToolsUsedMsg = `[ERROR] You did not use a tool in your previous response.
When you have a task to perform, you MUST use one of the available tools.
Only calling attempt_completion can end the subagent run.
Please review your task and call the appropriate tool(s).`

// turn holds the accumulated result of one assistant turn.
type turn struct {
	text     string
	calls    []*genai.FunctionCall
	inTok    int
	outTok   int
	thinking string
}

// Run executes a subagent with the given prompt and reports progress.
func (r *Runner) Run(ctx context.Context, prompt string, onProgress func(ProgressUpdate)) RunResult {
	r.abortMu.Lock()
	r.abortReq = false
	r.abortMu.Unlock()

	restrictedRegistry := r.builder.BuildRestrictedRegistry()
	decls, err := r.builder.buildDeclarations()
	if err != nil {
		res := RunResult{Status: StatusFailed, Error: err.Error()}
		onProgress(ProgressUpdate{Status: StatusFailed, Error: res.Error})
		return res
	}
	systemPrompt := r.builder.BuildSystemPrompt("act")

	var contents []*genai.Content
	appendUser := func(text string) {
		contents = append(contents, genai.NewContentFromText(text, "user"))
	}
	envBlock := r.builder.BuildEnvironmentBlock()
	initialContent := prompt
	if envBlock != "" {
		initialContent += "\n\n" + envBlock
	}
	appendUser(initialContent)

	inputTokens := 0
	outputTokens := 0
	toolCallsCount := 0
	emptyRetries := 0
	const maxEmptyRetries = 3

	for {
		if r.shouldAbort() {
			res := RunResult{Status: StatusFailed, Error: "subagent run cancelled"}
			onProgress(ProgressUpdate{Status: StatusFailed, Error: res.Error})
			return res
		}

		if est := estimateContentsTokens(contents); est > maxPromptTokens {
			res := RunResult{Status: StatusFailed, Error: fmt.Sprintf("subagent prompt too large (~%d tokens)", est)}
			onProgress(ProgressUpdate{Status: StatusFailed, Error: res.Error})
			return res
		}

		t, err := r.generateTurn(ctx, systemPrompt, decls, contents)
		if err != nil {
			res := RunResult{Status: StatusFailed, Error: err.Error()}
			onProgress(ProgressUpdate{Status: StatusFailed, Error: res.Error})
			return res
		}
		inputTokens += t.inTok
		outputTokens += t.outTok
		if t.inTok+t.outTok > 0 {
			log.Debugf("SubagentRunner: turn usage in=%d out=%d", t.inTok, t.outTok)
		}

		// Record the assistant turn in the conversation.
		parts := make([]*genai.Part, 0, 1+len(t.calls))
		if t.text != "" {
			parts = append(parts, genai.NewPartFromText(t.text))
		}
		for _, fc := range t.calls {
			parts = append(parts, &genai.Part{FunctionCall: fc})
		}
		contents = append(contents, &genai.Content{Role: "model", Parts: parts})

		// Handle empty response (no tools while tools are required).
		if len(t.calls) == 0 && needsToolSet(decls) {
			emptyRetries++
			if emptyRetries > maxEmptyRetries {
				res := RunResult{Status: StatusFailed, Error: "subagent did not call attempt_completion"}
				onProgress(ProgressUpdate{Status: StatusFailed, Error: res.Error})
				return res
			}
			appendUser(noToolsUsedMsg)
			continue
		}
		emptyRetries = 0

		// Execute tool calls.
		if len(t.calls) > 0 {
			toolResults, completed, completionResult, shouldStop, err := r.executeToolCalls(ctx, t.calls, restrictedRegistry, onProgress)
			if err != nil {
				res := RunResult{Status: StatusFailed, Error: err.Error()}
				onProgress(ProgressUpdate{Status: StatusFailed, Error: res.Error})
				return res
			}
			toolCallsCount += len(t.calls)
			resultParts := make([]*genai.Part, 0, len(toolResults))
			for _, tr := range toolResults {
				resultParts = append(resultParts, &genai.Part{
					FunctionResponse: &genai.FunctionResponse{
						ID:       tr.callID,
						Name:     tr.name,
						Response: map[string]any{"result": tr.result},
					},
				})
			}
			contents = append(contents, &genai.Content{Role: "user", Parts: resultParts})
			if completed {
				res := RunResult{
					Status:       StatusCompleted,
					Result:       completionResult,
					InputTokens:  inputTokens,
					OutputTokens: outputTokens,
					ToolCalls:    toolCallsCount,
				}
				onProgress(ProgressUpdate{
					Status:       StatusCompleted,
					Result:       completionResult,
					InputTokens:  inputTokens,
					OutputTokens: outputTokens,
					ToolCalls:    toolCallsCount,
				})
				return res
			}
			if shouldStop {
				break
			}
		} else if t.text != "" {
			// No tool calls and tools were not required: a plain final
			// answer completes the run.
			res := RunResult{
				Status:       StatusCompleted,
				Result:       t.text,
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				ToolCalls:    toolCallsCount,
			}
			onProgress(ProgressUpdate{Status: StatusCompleted, Result: res.Result})
			return res
		}
		// Continue loop for next assistant turn.
	}

	// Should never reach here, but Go requires a return at the end of the function.
	res := RunResult{Status: StatusFailed, Error: "subagent loop exited unexpectedly"}
	onProgress(ProgressUpdate{Status: StatusFailed, Error: res.Error})
	return res
}

// toolResult holds a single tool execution outcome.
type toolResult struct {
	callID string
	name   string
	result string
}

func (r *Runner) executeToolCalls(ctx context.Context, calls []*genai.FunctionCall, registry *tools.Registry, onProgress func(ProgressUpdate)) ([]toolResult, bool, string, bool, error) {
	var results []toolResult
	for _, fc := range calls {
		if r.shouldAbort() {
			return results, false, "", true, fmt.Errorf("subagent aborted")
		}

		name := fc.Name
		log.Infof("SubagentRunner: executing tool %s", name)
		onProgress(ProgressUpdate{LatestTool: fmt.Sprintf("%s(...)", name)})

		tool, err := registry.Get(name)
		if err != nil {
			results = append(results, toolResult{
				callID: fc.ID,
				name:   name,
				result: fmt.Sprintf("Error: Tool '%s' not found: %v", name, err),
			})
			continue
		}

		if name == types.ToolAskFollowupQuestion.String() {
			results = append(results, toolResult{
				callID: fc.ID,
				name:   name,
				result: "Error: ask_followup_question is not available in subagent mode.",
			})
			continue
		}

		if name == types.ToolAttemptCompletion.String() {
			var input struct {
				Result string `json:"result"`
			}
			raw, _ := json.Marshal(fc.Args)
			if jsonErr := json.Unmarshal(raw, &input); jsonErr == nil && input.Result != "" {
				return results, true, input.Result, false, nil
			}
			results = append(results, toolResult{
				callID: fc.ID,
				name:   name,
				result: "Error: attempt_completion requires a 'result' parameter.",
			})
			continue
		}

		raw, err := json.Marshal(fc.Args)
		if err != nil {
			raw = []byte("{}")
		}
		result, execErr := tool.Execute(ctx, raw)
		if execErr != nil {
			result = fmt.Sprintf("Error: %v", execErr)
		}
		results = append(results, toolResult{callID: fc.ID, name: name, result: result})
	}
	return results, false, "", false, nil
}

// generateTurn runs one streaming model turn and accumulates text parts,
// function calls and token usage.
func (r *Runner) generateTurn(ctx context.Context, systemPrompt string, decls []*genai.FunctionDeclaration, contents []*genai.Content) (turn, error) {
	var t turn
	req := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, "user"),
	}
	if len(decls) > 0 {
		req.Tools = []*genai.Tool{{FunctionDeclarations: decls}}
	}
	llmReq := &model.LLMRequest{
		Contents: contents,
		Config:   req,
	}
	var text strings.Builder
	var thinking strings.Builder
	for resp, err := range r.builder.LLM.GenerateContent(ctx, llmReq, true) {
		if err != nil {
			return t, err
		}
		if resp == nil {
			continue
		}
		if resp.Content != nil {
			for _, part := range resp.Content.Parts {
				if part == nil {
					continue
				}
				if part.Text != "" {
					if part.Thought {
						thinking.WriteString(part.Text)
					} else {
						text.WriteString(part.Text)
					}
				}
				if part.FunctionCall != nil {
					fc := part.FunctionCall
					if fc.ID == "" {
						fc.ID = fmt.Sprintf("call_%d", len(t.calls)+1)
					}
					if fc.Args == nil {
						fc.Args = map[string]any{}
					}
					t.calls = append(t.calls, fc)
				}
			}
		}
		if u := resp.UsageMetadata; u != nil {
			t.inTok = int(u.PromptTokenCount)
			t.outTok = int(u.CandidatesTokenCount)
		}
	}
	t.text = text.String()
	t.thinking = thinking.String()
	return t, nil
}

// needsToolSet returns true if the declarations contain non-terminal tools.
func needsToolSet(decls []*genai.FunctionDeclaration) bool {
	for _, d := range decls {
		if d.Name != types.ToolAttemptCompletion.String() &&
			d.Name != types.ToolAskFollowupQuestion.String() &&
			d.Name != types.ToolPlanModeRespond.String() {
			return true
		}
	}
	return false
}

// estimateContentsTokens gives a rough upper-bound token estimate for the
// conversation (same conservative per-rune heuristic as types.EstimateTokens).
func estimateContentsTokens(contents []*genai.Content) int {
	var total int
	for _, c := range contents {
		if c == nil {
			continue
		}
		for _, p := range c.Parts {
			switch {
			case p.Text != "":
				total += types.EstimateTokens(p.Text)
			case p.FunctionCall != nil:
				raw, _ := json.Marshal(p.FunctionCall)
				total += types.EstimateTokens(string(raw))
			case p.FunctionResponse != nil:
				raw, _ := json.Marshal(p.FunctionResponse)
				total += types.EstimateTokens(string(raw))
			}
		}
	}
	return total
}
