package tool

import (
	"encoding/json"
	"strings"

	"github.com/liup215/gline/pkg/types"
)

// AttemptCompletionRenderer renders attempt_completion tool output
type AttemptCompletionRenderer struct{}

func (r *AttemptCompletionRenderer) Render(req RenderRequest) RenderResult {
	if req.Phase == types.ToolPhaseStart {
		content := r.extractContent(req.Input)
		return RenderResult{
			Content:  content,
			Role:     types.RoleAssistant,
			Strategy: types.StrategyMarkdown,
			Skip:     false,
		}
	}
	// ToolPhaseComplete phase is skipped (already handled in Start)
	return RenderResult{Skip: true}
}

func (r *AttemptCompletionRenderer) Name() types.ToolName {
	return types.ToolAttemptCompletion
}

func (r *AttemptCompletionRenderer) Description() string {
	return "completed the task"
}

func (r *AttemptCompletionRenderer) Icon() string {
	return "✅"
}

func (r *AttemptCompletionRenderer) extractContent(input string) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(input), &parsed); err != nil {
		return input
	}

	// Prefer result as non-empty string
	if result, ok := parsed["result"].(string); ok && strings.TrimSpace(result) != "" {
		return result
	}
	if content, ok := parsed["content"].(string); ok && strings.TrimSpace(content) != "" {
		return content
	}

	// If result is an object, pretty-print as JSON code block
	if mres, ok := parsed["result"].(map[string]interface{}); ok {
		if pretty, err := json.MarshalIndent(mres, "", "  "); err == nil {
			return "```json\n" + string(pretty) + "\n```"
		}
	}

	// Fallback: pretty-print the whole JSON
	if pretty, err := json.MarshalIndent(parsed, "", "  "); err == nil {
		return "```json\n" + string(pretty) + "\n```"
	}

	return input
}
