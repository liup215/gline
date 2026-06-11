package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/pkg/types"
)

// Handler coordinates parallel execution of multiple subagents.
type Handler struct {
	Builder *Builder
}

// NewHandler creates a new handler with the shared builder.
func NewHandler(builder *Builder) *Handler {
	return &Handler{Builder: builder}
}

// SubagentItem represents a single subagent's state during execution.
type SubagentItem struct {
	Index        int
	Prompt       string
	Status       RunStatus
	Result       string
	Error        string
	InputTokens  int
	OutputTokens int
	ToolCalls    int
	LatestTool   string
}

// RunParallel executes up to 4 subagent prompts in parallel.
func (h *Handler) RunParallel(ctx context.Context, prompts []string, onProgress func([]SubagentItem)) []RunResult {
	maxSubagents := 4
	if len(prompts) > maxSubagents {
		prompts = prompts[:maxSubagents]
	}

	items := make([]SubagentItem, len(prompts))
	for i, p := range prompts {
		items[i] = SubagentItem{
			Index:  i + 1,
			Prompt: p,
			Status: StatusFailed, // default until started
		}
	}

	results := make([]RunResult, len(prompts))
	var wg sync.WaitGroup

	for i := range prompts {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			runner := NewRunner(h.Builder)

			// Wrap progress to update the shared items slice.
			onRunnerProgress := func(u ProgressUpdate) {
				items[idx].Status = u.Status
				if u.Result != "" {
					items[idx].Result = u.Result
				}
				if u.Error != "" {
					items[idx].Error = u.Error
				}
				items[idx].InputTokens = u.InputTokens
				items[idx].OutputTokens = u.OutputTokens
				items[idx].ToolCalls = u.ToolCalls
				items[idx].LatestTool = u.LatestTool
				onProgress(append([]SubagentItem(nil), items...)) // Send a copy to avoid races
			}

			items[idx].Status = RunStatus("running")
			onRunnerProgress(ProgressUpdate{Status: RunStatus("running")})

			result := runner.Run(ctx, prompts[idx], onRunnerProgress)
			results[idx] = result
			items[idx].Status = result.Status
			items[idx].Result = result.Result
			items[idx].Error = result.Error
			items[idx].InputTokens = result.InputTokens
			items[idx].OutputTokens = result.OutputTokens
			items[idx].ToolCalls = result.ToolCalls
			onRunnerProgress(ProgressUpdate{
				Status:       result.Status,
				Result:       result.Result,
				Error:        result.Error,
				InputTokens:  result.InputTokens,
				OutputTokens: result.OutputTokens,
				ToolCalls:    result.ToolCalls,
			})
		}(i)
	}

	wg.Wait()
	return results
}

// SummarizeResults aggregates multiple subagent results into a single formatted string.
func SummarizeResults(results []RunResult, prompts []string) string {
	if len(results) == 0 {
		return "No subagents were executed."
	}

	failures := 0
	successes := 0
	totalToolCalls := 0
	totalInput := 0
	totalOutput := 0

	for _, r := range results {
		if r.Status == StatusCompleted {
			successes++
		} else {
			failures++
		}
		totalToolCalls += r.ToolCalls
		totalInput += r.InputTokens
		totalOutput += r.OutputTokens
	}

	var b strings.Builder
	b.WriteString("Subagent results:\n")
	b.WriteString(fmt.Sprintf("Total: %d\n", len(results)))
	b.WriteString(fmt.Sprintf("Succeeded: %d\n", successes))
	b.WriteString(fmt.Sprintf("Failed: %d\n", failures))
	b.WriteString(fmt.Sprintf("Tool calls: %d\n", totalToolCalls))
	b.WriteString(fmt.Sprintf("Tokens: %d in / %d out\n", totalInput, totalOutput))
	b.WriteString("\n")

	for i, r := range results {
		header := fmt.Sprintf("[%d] %s - %s", i+1, strings.ToUpper(string(r.Status)), excerpt(prompts[i], 200))
		b.WriteString(header)
		b.WriteString("\n")
		if r.Error != "" {
			b.WriteString(excerpt(r.Error, 400))
			b.WriteString("\n")
		} else if r.Result != "" {
			b.WriteString(excerpt(r.Result, 2000))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// excerpt truncates text to maxChars, adding ellipsis if needed.
func excerpt(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	return text[:maxChars] + "..."
}

// UseSubagentsToolInput is the expected input for the use_subagents tool.
type UseSubagentsToolInput struct {
	Prompt1 string `json:"prompt_1"`
	Prompt2 string `json:"prompt_2,omitempty"`
	Prompt3 string `json:"prompt_3,omitempty"`
	Prompt4 string `json:"prompt_4,omitempty"`
}

// CollectPrompts extracts non-empty prompts from the input.
func CollectPrompts(input UseSubagentsToolInput) []string {
	prompts := []string{}
	if input.Prompt1 != "" {
		prompts = append(prompts, input.Prompt1)
	}
	if input.Prompt2 != "" {
		prompts = append(prompts, input.Prompt2)
	}
	if input.Prompt3 != "" {
		prompts = append(prompts, input.Prompt3)
	}
	if input.Prompt4 != "" {
		prompts = append(prompts, input.Prompt4)
	}
	return prompts
}

// ExecuteSubagentTool is the main entrypoint used by the use_subagents tool.
func ExecuteSubagentTool(ctx context.Context, builder *Builder, input UseSubagentsToolInput, onProgress func([]SubagentItem)) string {
	prompts := CollectPrompts(input)
	if len(prompts) == 0 {
		return "Error: At least prompt_1 is required for use_subagents."
	}

	handler := NewHandler(builder)
	results := handler.RunParallel(ctx, prompts, onProgress)
	return SummarizeResults(results, prompts)
}

// UseSubagentsTool implements the tools.Tool interface for use_subagents.
type UseSubagentsTool struct {
	Builder *Builder
}

// NewUseSubagentsTool creates a new subagent coordinator tool.
func NewUseSubagentsTool(builder *Builder) *UseSubagentsTool {
	return &UseSubagentsTool{Builder: builder}
}

// Name returns the tool name.
func (t *UseSubagentsTool) Name() string {
	return types.ToolUseSubagents.String()
}

// Description returns the tool description.
func (t *UseSubagentsTool) Description() string {
	return "Run up to four focused in-process subagents in parallel. Each subagent gets its own prompt and returns a comprehensive research result. Use this for broad exploration when reading many files would consume the main agent's context window. You do not need to launch multiple subagents every time; using one subagent is valid when it avoids unnecessary context usage for light discovery work."
}

// InputSchema returns the input JSON schema (json.RawMessage).
func (t *UseSubagentsTool) InputSchema() json.RawMessage {
	return []byte(`{
		"type": "object",
		"properties": {
			"prompt_1": {
				"type": "string",
				"description": "First subagent prompt."
			},
			"prompt_2": {
				"type": "string",
				"description": "Optional second subagent prompt."
			},
			"prompt_3": {
				"type": "string",
				"description": "Optional third subagent prompt."
			},
			"prompt_4": {
				"type": "string",
				"description": "Optional fourth subagent prompt."
			}
		},
		"required": ["prompt_1"]
	}`)
}

// Execute runs the subagent coordinator.
func (t *UseSubagentsTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req UseSubagentsToolInput
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("failed to parse use_subagents input: %w", err)
	}

	log.Infof("UseSubagentsTool: executing with %d prompts", len(CollectPrompts(req)))

	// Synchronous result (progress logging happens but the tool itself is synchronous).
	// The TUI can enhance this by wrapping the tool call to emit live events.
	result := ExecuteSubagentTool(ctx, t.Builder, req, func(items []SubagentItem) {
		log.Infof("Subagent progress: %d/%d running/completed", countRunning(items), len(items))
	})

	return result, nil
}

func countRunning(items []SubagentItem) int {
	c := 0
	for _, it := range items {
		if it.Status == RunStatus("running") {
			c++
		}
	}
	return c
}
