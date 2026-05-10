// Package ui: styles and rendering helpers extracted from tui.go
package ui

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Precompiled regex for camelCase -> snake_case conversion
var camelToSnakeRe = regexp.MustCompile("([a-z])([A-Z])")

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56C4")).
			MarginLeft(2)

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AAFF")).
			Bold(true)

	systemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	toolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500"))

	toolRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFD700")).
				Bold(true)

	toolCompletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00AA00"))

	toolFailedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	streamingIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00")).
				Bold(true)

	toolAreaBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#555555"))

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#666666")).
			Padding(0, 3).
			MarginTop(0)

	inputTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#AAAAAA")).
				Italic(true)

	questionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700")).
			MarginLeft(2)

	questionIconStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFD700")).
				MarginLeft(1)

	optionNumStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00FFAA"))

	optionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3A3A5C")).
			Padding(0, 2).
			MarginLeft(4)

	optionHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666")).
				Italic(true).
				MarginLeft(4)
)

// Tool descriptions and formatting helpers
var toolDescriptions = map[string]string{
	"read_file":               "read this file",
	"write_to_file":           "created a new file",
	"replace_in_file":         "edited this file",
	"execute_command":         "executed this command",
	"search_files":            "searched files",
	"attempt_completion":      "completed the task",
	"ask_followup_question":   "asked a question",
	"plan_mode_respond":       "provided a plan response",
	"use_mcp_tool":            "used an MCP tool",
	"access_mcp_resource":     "accessed an MCP resource",
}

// normalizeToolName converts camelCase to snake_case to make lookups predictable.
func normalizeToolName(name string) string {
	return strings.ToLower(camelToSnakeRe.ReplaceAllString(name, "${1}_${2}"))
}

// getToolDescription returns a short human-friendly description for a tool.
func getToolDescription(name string) string {
	n := normalizeToolName(name)
	if d, ok := toolDescriptions[n]; ok {
		return d
	}
	return "used a tool"
}

// getToolMainArg extracts the most relevant single argument from a tool input JSON.
// It returns an empty string when no main argument is found.
func getToolMainArg(toolName, inputJSON string) string {
	if inputJSON == "" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &m); err != nil {
		return ""
	}

	// Search regex in path
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

	// Command - truncate long commands for compact display
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

	// Question (for ask_followup_question tool)
	if q, ok := m["question"].(string); ok {
		return q
	}

	return ""
}

// formatToolResultLines truncates multi-line results to maxLines, adding a "... N more lines" footer when needed.
func formatToolResultLines(result string, maxLines int) []string {
	lines := strings.Split(result, "\n")
	if len(lines) <= maxLines {
		return lines
	}
	display := make([]string, 0, maxLines+1)
	display = append(display, lines[:maxLines]...)
	display = append(display, fmt.Sprintf("... %d more lines", len(lines)-maxLines))
	return display
}