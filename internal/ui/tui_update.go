// Package ui: extracted agent update handlers
package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func handleAgentUpdate(m *Model, msg agentUpdateMsg) []tea.Cmd {
	var cmds []tea.Cmd

	switch msg.updateType {
	case "content":
		cmds = append(cmds, handleAgentContent(m, msg)...)
	case "toolStart":
		cmds = append(cmds, handleAgentToolStart(m, msg)...)
	case "toolComplete":
		cmds = append(cmds, handleAgentToolComplete(m, msg)...)
	case "error":
		cmds = append(cmds, handleAgentError(m, msg)...)
	case "complete":
		cmds = append(cmds, handleAgentComplete(m, msg)...)
	case "streamStart":
		cmds = append(cmds, handleAgentStreamStart(m, msg)...)
	case "streamEnd":
		cmds = append(cmds, handleAgentStreamEnd(m, msg)...)
	default:
		// unknown update type — no-op
	}

	return cmds
}
