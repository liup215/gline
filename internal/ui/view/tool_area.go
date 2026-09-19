package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/liup215/gline/internal/ui/model"
)

// RenderToolArea renders the tool status area below the viewport.
// The content string is already pre-rendered by the ViewModel.
func RenderToolArea(content string) string {
	return content
}

// RenderToolAreaContent renders the tool history into a display string.
// This is a pure function with no side effects.
// Parameters:
//   - history: the list of tool statuses to render
//   - width: the width of the display area
//   - maxEntries: maximum number of entries to show
func RenderToolAreaContent(history []model.ToolStatus, width int, maxEntries int) string {
	if len(history) == 0 {
		return ToolAreaBorderStyle.Render(strings.Repeat("─", width))
	}

	if maxEntries < 1 {
		maxEntries = 1
	}

	start := 0
	if len(history) > maxEntries {
		start = len(history) - maxEntries
	}

	var lines []string
	for i := start; i < len(history); i++ {
		ts := history[i]
		switch ts.Status {
		case "running":
			lines = append(lines, ToolRunningStyle.Render(fmt.Sprintf("  🔧 %s ⏳", ts.Name)))
		case "completed":
			lines = append(lines, ToolCompletedStyle.Render(fmt.Sprintf("  🔧 %s ✓", ts.Name)))
		case "failed":
			lines = append(lines, ToolFailedStyle.Render(fmt.Sprintf("  🔧 %s ✗", ts.Name)))
		default:
			lines = append(lines, SystemStyle.Render(fmt.Sprintf("  🔧 %s", ts.Name)))
		}
	}

	border := ToolAreaBorderStyle.Render(strings.Repeat("─", width))
	all := []string{border}
	all = append(all, lines...)
	return lipgloss.JoinVertical(lipgloss.Left, all...)
}
