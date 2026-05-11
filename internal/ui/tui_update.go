// Package ui: extracted agent update handlers
package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/liup215/gline/internal/ui/bridge"
)

func handleAgentUpdate(m *Model, msg bridge.AgentEvent) []tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case bridge.ContentEvent:
		cmds = append(cmds, handleAgentContent(m, msg)...)
	case bridge.ToolStartEvent:
		cmds = append(cmds, handleAgentToolStart(m, msg)...)
	case bridge.ToolCompleteEvent:
		cmds = append(cmds, handleAgentToolComplete(m, msg)...)
	case bridge.ErrorEvent:
		cmds = append(cmds, handleAgentError(m, msg)...)
	case bridge.CompleteEvent:
		cmds = append(cmds, handleAgentComplete(m, msg)...)
	case bridge.StreamStartEvent:
		cmds = append(cmds, handleAgentStreamStart(m, msg)...)
	case bridge.StreamEndEvent:
		cmds = append(cmds, handleAgentStreamEnd(m, msg)...)
	default:
		// unknown event type — no-op
	}

	return cmds
}
