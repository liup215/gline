// Package ui: view helpers extracted from tui.go
package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/agent"
)

// updateViewport refreshes the viewport content via the ViewModel.
func (m *Model) updateViewport() {
	m.convVM.Refresh(m.conversation, m.viewport.Width, m.toolAreaHeight, m.isStreaming, m.activeAssistantIndex)
	m.viewport.SetContent(m.convVM.Content())
	m.viewport.GotoBottom()
}

// renderToolArea renders the tool status area below the viewport.
func (m *Model) renderToolArea() string {
	return m.convVM.ToolAreaContent()
}

// renderStatusBar renders the status bar.
func (m *Model) renderStatusBar() string {
	modeStr := string(m.conversation.Mode)
	if m.conversation.Mode == agent.ModeAct {
		modeStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("ACT")
	} else {
		modeStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("PLAN")
	}

	provider := m.conversation.Provider
	if provider == "" {
		provider = "-"
	}
	mdl := m.conversation.ModelName
	if mdl == "" {
		mdl = "-"
	}

	status := fmt.Sprintf("[%s] Provider: %s | Model: %s", modeStr, provider, mdl)

	if m.isProcessing {
		if m.isStreaming {
			status += fmt.Sprintf(" | %s AI is responding...", m.spinner.View())
		} else if m.currentTool != "" {
			status += fmt.Sprintf(" | %s Running: %s", m.spinner.View(), m.currentTool)
		} else {
			status += fmt.Sprintf(" | %s Processing...", m.spinner.View())
		}
	}

	return statusBarStyle.Width(m.width).Render(status)
}
