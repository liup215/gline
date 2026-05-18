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

// RenderCompactBar renders a minimal header bar with just the gline logo.
func RenderCompactBar(data CompactBarData) string {
	// Just show the gline logo and spinner if processing
	leftContent := "🚀 gline"
	if data.IsProcessing {
		leftContent = fmt.Sprintf("%s %s", data.SpinnerView, leftContent)
	}
	return lipgloss.NewStyle().Bold(true).Render(leftContent)
}

// InputStatusBarData holds data for the status bar below input.
type InputStatusBarData struct {
	Mode      agent.Mode
	Provider  string
	ModelName string
	Width     int
}

// RenderInputStatusBar renders the multi-line status bar below input (cline style).
// Line 1: Model info (left) | Plan/Act mode (right)
// Line 2: Current working directory
func RenderInputStatusBar(data InputStatusBarData) string {
	prov := data.Provider
	if prov == "" {
		prov = "-"
	}
	mdl := data.ModelName
	if mdl == "" {
		mdl = "-"
	}

	// Line 1: Model info | Plan/Act toggle
	modelInfo := fmt.Sprintf("%s · %s", prov, mdl)

	// Mode toggle (○ Plan ● Act)
	var planIndicator, actIndicator string
	if data.Mode == agent.ModePlan {
		planIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("● Plan")
		actIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("○ Act")
	} else {
		planIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("○ Plan")
		actIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#00AAFF")).Render("● Act")
	}
	modeToggle := fmt.Sprintf("%s %s (Tab)", planIndicator, actIndicator)

	// Join line 1 with spacing
	modelWidth := lipgloss.Width(modelInfo)
	modeWidth := lipgloss.Width(modeToggle)
	available := data.Width - modelWidth - modeWidth - 2 // 2 for padding
	if available < 0 {
		available = 0
	}

	line1 := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Render(modelInfo),
		lipgloss.NewStyle().Width(available).Render(" "),
		lipgloss.NewStyle().Render(modeToggle),
	)

	// Line 2: Current directory (would need to be passed in data)
	// For now, just use a placeholder or empty
	line2 := "" // Could add WorkingDir to data if needed

	if line2 != "" {
		return lipgloss.JoinVertical(lipgloss.Left, line1, line2)
	}
	return line1
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