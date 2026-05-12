package view

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/agent"
)

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