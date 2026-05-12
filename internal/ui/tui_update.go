// Package ui: extracted agent update handlers
package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/liup215/gline/internal/ui/bridge"
)

// handleAgentUpdate dispatches an AgentEvent to the appropriate handler.
// Returns (needsRefresh, cmds) where needsRefresh indicates whether the
// viewport should be refreshed after the state mutation.
func handleAgentUpdate(m *Model, msg bridge.AgentEvent) (bool, []tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case bridge.ContentEvent:
		needsRefresh, handlerCmds := handleAgentContent(m, msg)
		cmds = append(cmds, handlerCmds...)
		return needsRefresh, cmds
	case bridge.ToolStartEvent:
		needsRefresh, handlerCmds := handleAgentToolStart(m, msg)
		cmds = append(cmds, handlerCmds...)
		return needsRefresh, cmds
	case bridge.ToolCompleteEvent:
		needsRefresh, handlerCmds := handleAgentToolComplete(m, msg)
		cmds = append(cmds, handlerCmds...)
		return needsRefresh, cmds
	case bridge.ErrorEvent:
		needsRefresh, handlerCmds := handleAgentError(m, msg)
		cmds = append(cmds, handlerCmds...)
		return needsRefresh, cmds
	case bridge.CompleteEvent:
		needsRefresh, handlerCmds := handleAgentComplete(m, msg)
		cmds = append(cmds, handlerCmds...)
		return needsRefresh, cmds
	case bridge.StreamStartEvent:
		needsRefresh, handlerCmds := handleAgentStreamStart(m, msg)
		cmds = append(cmds, handlerCmds...)
		return needsRefresh, cmds
	case bridge.StreamEndEvent:
		needsRefresh, handlerCmds := handleAgentStreamEnd(m, msg)
		cmds = append(cmds, handlerCmds...)
		return needsRefresh, cmds
	default:
		// unknown event type — no-op
		return false, cmds
	}
}
