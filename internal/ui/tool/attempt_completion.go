package tool

import (
	"encoding/json"
	"strings"

	"github.com/liup215/gline/pkg/types"
)

// AttemptCompletionRenderer renders attempt_completion tool output
type AttemptCompletionRenderer struct{}

func (r *AttemptCompletionRenderer) Render(req RenderRequest) RenderResult {
	// Skip Start phase - tool result will be shown in Complete phase as system message
	if req.Phase == types.ToolPhaseStart {
		return RenderResult{Skip: true}
	}

	// ToolPhaseComplete: show completion result as system message.
	// req.Input here is the tool's formatted Execute() output (banner-wrapped
	// plain text), or occasionally the raw JSON input. extractContent handles
	// both and returns the bare result.
	content := r.extractContent(req.Input)
	if content == "" {
		return RenderResult{Skip: true}
	}
	return RenderResult{
		Content:  r.Icon() + " " + content,
		Role:     types.RoleSystem,
		Strategy: types.StrategyMarkdown,
		Skip:     false,
	}
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

// stripCompletionBanner removes the decorative "✅ Task Completed" header and
// "====" separator lines that AttemptCompletionTool.Execute wraps around the
// result, leaving only the bare result text.
func stripCompletionBanner(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			// Drop blank lines adjacent to the banner; keep interior blanks.
			if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" && !bannerOnly(out) {
				out = append(out, line)
			}
			continue
		}
		// Skip banner header and separator runs of '='
		if trimmed == "✅ Task Completed" || strings.Trim(trimmed, "=") == "" {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// bannerOnly reports whether all lines so far were banner decorations
// (i.e. we haven't reached real result content yet).
func bannerOnly(out []string) bool {
	for _, l := range out {
		t := strings.TrimSpace(l)
		if t != "" && strings.Trim(t, "=") != "" {
			return false
		}
	}
	return true
}

func (r *AttemptCompletionRenderer) extractContent(input string) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(input), &parsed); err != nil {
		// Not JSON: this is the tool's formatted Execute() output — strip the
		// decorative banner/separators so only the result text is shown.
		return stripCompletionBanner(input)
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
