package view

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/agent"
)

// CompactBarData holds all data needed for the merged header/status bar.
type CompactBarData struct {
	Mode         agent.Mode
	Provider     string
	ModelName    string
	IsProcessing bool
	IsStreaming  bool
	CurrentTool  string
	SpinnerView  string
	Width        int
}

// RenderCompactBar renders a single line combining header info and dynamic status.
// Left side: 🚀 gline · Provider/Model · [MODE]
// Right side: dynamic status (Processing/Streaming/Tool with spinner)
func RenderCompactBar(data CompactBarData) string {
	// Left section: logo, provider/model, mode badge
	modeBadge := "UNKNOWN"
	if data.Mode == agent.ModeAct {
		modeBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("[ACT]")
	} else {
		modeBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("[PLAN]")
	}

	prov := data.Provider
	if prov == "" {
		prov = "-"
	}
	mdl := data.ModelName
	if mdl == "" {
		mdl = "-"
	}

	leftContent := fmt.Sprintf("🚀 gline · %s/%s · %s", prov, mdl, modeBadge)
	leftSection := lipgloss.NewStyle().Bold(true).Render(leftContent)

	// Right section: dynamic status
	rightSection := ""
	if data.IsProcessing {
		if data.IsStreaming {
			rightSection = fmt.Sprintf("%s AI is responding...", data.SpinnerView)
		} else if data.CurrentTool != "" {
			rightSection = fmt.Sprintf("%s Running: %s", data.SpinnerView, data.CurrentTool)
		} else {
			rightSection = fmt.Sprintf("%s Processing...", data.SpinnerView)
		}
		rightSection = lipgloss.NewStyle().Foreground(lipgloss.Color("#00AAFF")).Render(rightSection)
	}

	// Calculate left width and pad right to align
	leftWidth := lipgloss.Width(leftSection)
	rightWidth := lipgloss.Width(rightSection)

	// If no status, just return the left section
	if rightSection == "" {
		return StatusBarStyle.Width(data.Width).Render(leftSection)
	}

	// Join with padding to push right section to the edge
	// Available space for middle padding
	available := data.Width - leftWidth - rightWidth
	if available < 1 {
		available = 1
	}

	// Use lipgloss.JoinHorizontal with the calculated spacing
	result := lipgloss.JoinHorizontal(lipgloss.Top, leftSection, lipgloss.NewStyle().Width(available).Render(""), rightSection)
	return StatusBarStyle.Width(data.Width).Render(result)
}

// StatusBarData holds the data needed to render the status bar.
type StatusBarData struct {
	Mode         agent.Mode
	Provider     string
	ModelName    string
	IsProcessing bool
	IsStreaming   bool
	CurrentTool  string
	SpinnerView  string // pre-rendered spinner string from Bubbletea
	Width        int
}

// RenderStatusBar renders the status bar as a pure function.
func RenderStatusBar(data StatusBarData) string {
	modeStr := string(data.Mode)
	if data.Mode == agent.ModeAct {
		modeStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("ACT")
	} else {
		modeStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("PLAN")
	}

	provider := data.Provider
	if provider == "" {
		provider = "-"
	}
	mdl := data.ModelName
	if mdl == "" {
		mdl = "-"
	}

	status := fmt.Sprintf("[%s] Provider: %s | Model: %s", modeStr, provider, mdl)

	if data.IsProcessing {
		if data.IsStreaming {
			status += fmt.Sprintf(" | %s AI is responding...", data.SpinnerView)
		} else if data.CurrentTool != "" {
			status += fmt.Sprintf(" | %s Running: %s", data.SpinnerView, data.CurrentTool)
		} else {
			status += fmt.Sprintf(" | %s Processing...", data.SpinnerView)
		}
	}

	return StatusBarStyle.Width(data.Width).Render(status)
}