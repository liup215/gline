package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
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

// Run executes a subagent with the given prompt and reports progress.
func (r *Runner) Run(ctx context.Context, prompt string, onProgress func(ProgressUpdate)) RunResult {
	r.abortMu.Lock()
	r.abortReq = false
	r.abortMu.Unlock()

	restrictedRegistry := r.builder.BuildRestrictedRegistry()
	convertedTools := r.builder.ConvertTools()
	systemPrompt := r.builder.BuildSystemPrompt(string(agent.ModeAct))

	conv := types.NewConversation()
	conv.MaxTokens = 262000

	envBlock := r.builder.BuildEnvironmentBlock()
	initialContent := prompt
	if envBlock != "" {
		initialContent += "\n\n" + envBlock
	}
	conv.AddMessage(types.Message{Role: types.RoleUser, Content: initialContent})

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

		conv.TrimToMaxTokens()
		needsTool := needsToolSet(convertedTools)

		req := &agent.MessageRequest{
			Messages:     conv.GetMessages(),
			Tools:        convertedTools,
			SystemPrompt: systemPrompt,
		}
		if needsTool {
			req.ToolChoice = agent.ToolChoiceAuto
		}

		log.Infof("SubagentRunner: requesting tools=%d needsTool=%v", len(convertedTools), needsTool)

		streamChan, err := r.builder.Provider.CreateMessageStream(ctx, req)
		if err != nil {
			res := RunResult{Status: StatusFailed, Error: err.Error()}
			onProgress(ProgressUpdate{Status: StatusFailed, Error: res.Error})
			return res
		}

		content, reasoning, assistantToolCalls, usage, err := r.processStream(ctx, streamChan)
		if err != nil {
			res := RunResult{Status: StatusFailed, Error: err.Error()}
			onProgress(ProgressUpdate{Status: StatusFailed, Error: res.Error})
			return res
		}

		if usage.TotalTokens > 0 {
			conv.AddActualTokens(usage.InputTokens, usage.OutputTokens)
			inputTokens += usage.InputTokens
			outputTokens += usage.OutputTokens
		}

		// Fallback: parse XML tool calls if native tool_calls were not received.
		if len(assistantToolCalls) == 0 {
			parsedXML := agent.ParseXMLToolCalls(content, convertedTools)
			for _, tc := range parsedXML {
				assistantToolCalls = append(assistantToolCalls, agent.ToolCall{
					ID:    tc.ID,
					Name:  tc.Name,
					Input: tc.Input,
				})
			}
			if len(assistantToolCalls) > 0 {
				log.Infof("SubagentRunner: parsed %d XML tool calls", len(assistantToolCalls))
			}
		}

		var typesToolCalls []types.ToolCall
		for _, tc := range assistantToolCalls {
			typesToolCalls = append(typesToolCalls, types.ToolCall{
				ID:    tc.ID,
				Name:  tc.Name,
				Input: []byte(tc.Input),
			})
		}
		conv.AddMessage(types.Message{
			Role:             types.RoleAssistant,
			Content:          content,
			ReasoningContent: reasoning,
			ToolCalls:        typesToolCalls,
		})

		// Handle empty response (no tools while tools are required).
		if !conv.IsComplete() && len(assistantToolCalls) == 0 && needsTool {
			emptyRetries++
			if emptyRetries > maxEmptyRetries {
				res := RunResult{Status: StatusFailed, Error: "subagent did not call attempt_completion"}
				onProgress(ProgressUpdate{Status: StatusFailed, Error: res.Error})
				return res
			}
			conv.AddMessage(types.Message{Role: types.RoleUser, Content: noToolsUsedMsg})
			continue
		}
		emptyRetries = 0

		// Execute tool calls.
		if len(assistantToolCalls) > 0 {
			toolResults, completed, completionResult, shouldStop, err := r.executeToolCalls(ctx, assistantToolCalls, restrictedRegistry, onProgress)
			if err != nil {
				res := RunResult{Status: StatusFailed, Error: err.Error()}
				onProgress(ProgressUpdate{Status: StatusFailed, Error: res.Error})
				return res
			}
			toolCallsCount += len(assistantToolCalls)
			for _, tr := range toolResults {
				conv.AddMessage(types.Message{
					Role:       types.RoleTool,
					ToolCallID: tr.callID,
					Content:    tr.result,
				})
			}
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
		}

		if conv.IsComplete() {
			res := RunResult{
				Status:       StatusCompleted,
				Result:       content,
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
	result string
}

func (r *Runner) executeToolCalls(ctx context.Context, calls []agent.ToolCall, registry *tools.Registry, onProgress func(ProgressUpdate)) ([]toolResult, bool, string, bool, error) {
	var results []toolResult
	for _, tc := range calls {
		if r.shouldAbort() {
			return results, false, "", true, fmt.Errorf("subagent aborted")
		}

		log.Infof("SubagentRunner: executing tool %s", tc.Name)
		onProgress(ProgressUpdate{LatestTool: fmt.Sprintf("%s(...)", tc.Name)})

		tool, err := registry.Get(tc.Name)
		if err != nil {
			results = append(results, toolResult{
				callID: tc.ID,
				result: fmt.Sprintf("Error: Tool '%s' not found: %v", tc.Name, err),
			})
			continue
		}

		if tc.Name == types.ToolAskFollowupQuestion.String() {
			results = append(results, toolResult{
				callID: tc.ID,
				result: "Error: ask_followup_question is not available in subagent mode.",
			})
			continue
		}

		if tc.Name == types.ToolAttemptCompletion.String() {
			var input struct {
				Result string `json:"result"`
			}
			if jsonErr := json.Unmarshal([]byte(tc.Input), &input); jsonErr == nil && input.Result != "" {
				return results, true, input.Result, false, nil
			}
			results = append(results, toolResult{
				callID: tc.ID,
				result: "Error: attempt_completion requires a 'result' parameter.",
			})
			continue
		}

		result, execErr := tool.Execute(ctx, []byte(tc.Input))
		if execErr != nil {
			result = fmt.Sprintf("Error: %v", execErr)
		}
		results = append(results, toolResult{callID: tc.ID, result: result})
	}
	return results, false, "", false, nil
}

func (r *Runner) processStream(ctx context.Context, streamChan <-chan agent.StreamChunk) (string, string, []agent.ToolCall, agent.TokenUsage, error) {
	var content strings.Builder
	var reasoning strings.Builder
	var toolCalls []agent.ToolCall
	var usage agent.TokenUsage

	for chunk := range streamChan {
		if chunk.Error != nil {
			return "", "", nil, usage, chunk.Error
		}
		if chunk.Done {
			break
		}
		if chunk.Content != "" {
			content.WriteString(chunk.Content)
		}
		if chunk.ReasoningContent != "" {
			reasoning.WriteString(chunk.ReasoningContent)
		}
		if chunk.ToolCall != nil && !chunk.IsPartial {
			toolCalls = append(toolCalls, *chunk.ToolCall)
		}
		if chunk.Usage.TotalTokens > 0 {
			usage = chunk.Usage
		}
	}

	return content.String(), reasoning.String(), toolCalls, usage, nil
}

// needsToolSet returns true if the tool list contains non-terminal tools.
func needsToolSet(toolsDefs []agent.ToolDefinition) bool {
	for _, t := range toolsDefs {
		if t.Name != types.ToolAttemptCompletion.String() &&
			t.Name != types.ToolAskFollowupQuestion.String() &&
			t.Name != types.ToolPlanModeRespond.String() {
			return true
		}
	}
	return false
}
