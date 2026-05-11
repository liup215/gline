// Package ui: view helpers extracted from tui.go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/pkg/types"
)

 
// updateViewport refreshes the viewport content
func (m *Model) updateViewport() {
	var content strings.Builder
	msgs := m.conversation.Messages

	for i := range msgs {
		msg := msgs[i]
		switch msg.Role {
		case types.RoleUser:
			content.WriteString(userStyle.Render("You: "))
			content.WriteString(msg.Content)
			content.WriteString("\n")
			content.WriteString(systemStyle.Render(msg.Timestamp.Format("15:04")))
			content.WriteString("\n\n")

		case types.RoleAssistant:
					// Use centralized render helpers
					content.WriteString(m.renderMessageHeader(i))
					content.WriteString(renderAssistantContent(m, i))
					// renderAssistantContent ensures trailing newline; keep spacing consistent
					content.WriteString("\n")

		case types.RoleSystem:
			// Render errors and tool messages (preserve existing behavior)
			if strings.HasPrefix(msg.Content, "Error:") || strings.HasPrefix(msg.Content, "✗") {
				content.WriteString(errorStyle.Render(msg.Content))
				content.WriteString("\n\n")
			} else if strings.HasPrefix(msg.Content, "❓") || len(msg.Options) > 0 {
				content.WriteString(questionIconStyle.Render("❓ "))
				content.WriteString(questionStyle.Render(strings.TrimPrefix(msg.Content, "❓ ")))
				content.WriteString("\n")
				if len(msg.Options) > 0 {
					for i, opt := range msg.Options {
						num := optionNumStyle.Render(fmt.Sprintf("%d.", i+1))
						content.WriteString(optionStyle.Render(fmt.Sprintf("%s %s", num, opt)))
						content.WriteString("\n")
					}
					content.WriteString(optionHintStyle.Render("Enter option number or type your answer"))
					content.WriteString("\n")
				}
				content.WriteString("\n")
			} else if strings.HasPrefix(msg.Content, "🔧") {
				if strings.Contains(msg.Content, "Running") || strings.Contains(msg.Content, "running") || strings.Contains(msg.Content, "started") {
					content.WriteString(toolRunningStyle.Render(msg.Content))
				} else if strings.Contains(msg.Content, "Completed") || strings.Contains(msg.Content, "✓") || strings.Contains(msg.Content, "completed") {
					content.WriteString(toolCompletedStyle.Render(msg.Content))
				} else if strings.Contains(msg.Content, "Failed") || strings.Contains(msg.Content, "✗") || strings.Contains(msg.Content, "failed") {
					content.WriteString(toolFailedStyle.Render(msg.Content))
				} else {
					content.WriteString(systemStyle.Render(msg.Content))
				}
				content.WriteString("\n\n")
			}
		}
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

// renderToolArea renders the tool status area below the viewport
func (m *Model) renderToolArea() string {
	// Delegate to centralized renderer
	return renderToolCalls(m)
}

// renderStatusBar renders the status bar
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