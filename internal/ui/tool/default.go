package tool

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liup215/gline/pkg/types"
)

// DefaultRenderer is a generic renderer for standard tools
type DefaultRenderer struct {
	name        types.ToolName
	description string
	icon        string
}

// NewDefaultRenderer creates a default renderer for a tool
func NewDefaultRenderer(name types.ToolName, description string) *DefaultRenderer {
	return &DefaultRenderer{
		name:        name,
		description: description,
		icon:        "🔧",
	}
}

func (r *DefaultRenderer) Render(req RenderRequest) RenderResult {
	if req.Phase == types.ToolPhaseStart {
		content := r.formatStartDisplay(req.Input)
		return RenderResult{
			Content:  content,
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

func (r *DefaultRenderer) Name() types.ToolName {
	return r.name
}

func (r *DefaultRenderer) Description() string {
	return r.description
}

func (r *DefaultRenderer) Icon() string {
	return r.icon
}

func (r *DefaultRenderer) formatStartDisplay(input string) string {
	if input == "" {
		return fmt.Sprintf("%s %s", r.icon, r.description)
	}

	// Try to extract main argument
	if main := r.extractMainArg(input); main != "" {
		return fmt.Sprintf("%s %s: %s", r.icon, r.description, main)
	}

	// Fallback: show compact input (truncate if too long)
	input = strings.TrimSpace(input)
	if len(input) > 100 {
		input = input[:97] + "..."
	}
	return fmt.Sprintf("%s %s: %s", r.icon, r.description, input)
}

func (r *DefaultRenderer) formatCompleteDisplay(input, status string) string {
	statusText := "Completed"
	if status == "failed" {
		statusText = "Failed"
	}
	return fmt.Sprintf("%s %s: %s", r.icon, statusText, r.name)
}

func (r *DefaultRenderer) extractMainArg(input string) string {
	if input == "" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		return ""
	}

	// Check for regex search
	if regex, ok := m["regex"].(string); ok {
		if path, ok2 := m["path"].(string); ok2 {
			return fmt.Sprintf("'%s' in %s", regex, path)
		}
	}

	// File path
	if p, ok := m["path"].(string); ok {
		return p
	}
	if fp, ok := m["file_path"].(string); ok {
		return fp
	}

	// Command - truncate long commands
	if cmd, ok := m["command"].(string); ok {
		if len(cmd) > 120 {
			return cmd[:117] + "..."
		}
		return cmd
	}

	// URL / query
	if u, ok := m["url"].(string); ok {
		return u
	}
	if q, ok := m["query"].(string); ok {
		return q
	}

	return ""
}
