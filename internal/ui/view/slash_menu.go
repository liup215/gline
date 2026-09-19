package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/liup215/gline/pkg/types"
)

// SlashMenuStyle is the style for the slash command menu.
var (
	// SlashMenuBoxStyle matches InputBoxStyle for alignment:
	// - Border: 2 chars (left+right)
	// - Padding(0, 3): 6 chars total (3 left + 3 right)
	// - MarginLeft(1): 1 char left margin
	// This matches the input box exactly so they align visually
	SlashMenuBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56C4")).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 3)

	SlashMenuItemStyle = lipgloss.NewStyle().
				Padding(0, 1)

	SlashMenuSelectedStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(lipgloss.Color("#000000")).
				Background(lipgloss.Color("#00AAFF")).
				Bold(true)

	SlashMenuDescriptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Italic(true)

	SlashMenuHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666")).
				Italic(true)
)

// SlashMenuData holds the data needed to render the slash command menu.
type SlashMenuData struct {
	Commands      []*types.SlashCommand
	SelectedIndex int
	Width         int
	Query         string
}

// RenderSlashMenu renders a scrollable command selection menu.
// maxVisible controls how many items are shown at once.
func RenderSlashMenu(data SlashMenuData, maxVisible int) string {
	if len(data.Commands) == 0 {
		if data.Query != "" {
			return SlashMenuBoxStyle.Render(fmt.Sprintf("No commands match \"%s\"", data.Query))
		}
		return ""
	}

	// Calculate visible window
	visible, startIdx := getVisibleWindow(len(data.Commands), data.SelectedIndex, maxVisible)

	var lines []string
	for i, idx := range visible {
		cmd := data.Commands[idx]
		style := SlashMenuItemStyle
		if idx == data.SelectedIndex {
			style = SlashMenuSelectedStyle
		}

		// Command name with / prefix
		name := style.Render(fmt.Sprintf("/%s", cmd.Name))
		// Description
		desc := SlashMenuDescriptionStyle.Render(cmd.Description)

		line := lipgloss.JoinHorizontal(lipgloss.Top, name, "  ", desc)
		lines = append(lines, line)

		// Number prefix for keyboard shortcuts (1-9)
		if i < 9 {
			numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
			lines[len(lines)-1] = lipgloss.JoinHorizontal(lipgloss.Top, numStyle.Render(fmt.Sprintf(" %d ", i+1)), line)
		}
	}

	// Scroll indicators
	if startIdx > 0 {
		hint := SlashMenuHintStyle.Render("▲ more")
		lines = append([]string{hint}, lines...)
	}
	if startIdx+maxVisible < len(data.Commands) {
		hint := SlashMenuHintStyle.Render("▼ more")
		lines = append(lines, hint)
	}

	// Query hint
	if data.Query != "" {
		hint := SlashMenuHintStyle.Render(fmt.Sprintf("query: /%s", data.Query))
		lines = append(lines, "", hint)
	}

	menuContent := strings.Join(lines, "\n")
	// Full width minus border(2), no margin
	return SlashMenuBoxStyle.Width(data.Width - 2).Render(menuContent)
}

// getVisibleWindow returns the indices of items that should be visible,
// keeping the selected item centered when possible.
func getVisibleWindow(total, selected, maxVisible int) ([]int, int) {
	if total <= maxVisible {
		indices := make([]int, total)
		for i := 0; i < total; i++ {
			indices[i] = i
		}
		return indices, 0
	}

	half := maxVisible / 2
	start := selected - half
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	indices := make([]int, 0, maxVisible)
	for i := start; i < end; i++ {
		indices = append(indices, i)
	}
	return indices, start
}
