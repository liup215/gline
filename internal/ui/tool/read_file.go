package tool

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liup215/gline/pkg/types"
)

// ReadFileRenderer renders read_file tool output
type ReadFileRenderer struct{}

func (r *ReadFileRenderer) Render(req RenderRequest) RenderResult {
	if req.Phase == types.ToolPhaseStart {
		path := r.extractPath(req.Input)
		return RenderResult{
			Content:  r.formatStartDisplay(path),
			Role:     types.RoleSystem,
			Strategy: types.StrategyPlain,
			Skip:     false,
		}
	}

	// ToolPhaseComplete
	return RenderResult{
		Content:  r.formatCompleteDisplay(req.Input, req.Status),
		Role:     types.RoleSystem,
		Strategy: types.StrategyPlain,
		Skip:     false,
	}
}

func (r *ReadFileRenderer) Name() types.ToolName {
	return types.ToolReadFile
}

func (r *ReadFileRenderer) Description() string {
	return "read this file"
}

func (r *ReadFileRenderer) Icon() string {
	return "📖"
}

func (r *ReadFileRenderer) extractPath(input string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		return ""
	}
	if p, ok := m["path"].(string); ok {
		return p
	}
	return ""
}

func (r *ReadFileRenderer) formatStartDisplay(path string) string {
	if path != "" {
		return fmt.Sprintf("🔧 %s: %s", r.Description(), path)
	}
	return "🔧 " + r.Description()
}

func (r *ReadFileRenderer) formatCompleteDisplay(result, status string) string {
	statusText := "Completed"
	if status == "failed" {
		statusText = "Failed"
	}

	content := fmt.Sprintf("🔧 %s: %s", statusText, r.Name())
	if result != "" {
		lines := r.formatResultLines(result, 5)
		content += "\n"
		for _, l := range lines {
			content += l + "\n"
		}
	}
	return content
}

func (r *ReadFileRenderer) formatResultLines(result string, maxLines int) []string {
	lines := strings.Split(result, "\n")
	if len(lines) <= maxLines {
		return lines
	}
	display := make([]string, 0, maxLines+1)
	display = append(display, lines[:maxLines]...)
	display = append(display, fmt.Sprintf("... %d more lines", len(lines)-maxLines))
	return display
}
