package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liup215/gline/internal/ui/bridge"
)

// startAgent starts the agent with a TUIBridge callback.
// TUIBridge sends typed events over a channel (eventCh) which are forwarded
// into the Bubbletea event loop by a goroutine set up in Run().
func (m *Model) startAgent() tea.Cmd {
	return func() tea.Msg {
		if m.agentInstance == nil {
			return bridge.ErrorEvent{Err: fmt.Errorf("agent not initialized")}
		}

		// Get the last user message
		lastUserMessage, _ := m.conversation.LastUserMessage()

		if lastUserMessage == "" {
			return bridge.ErrorEvent{Err: fmt.Errorf("no user message found")}
		}

		// Create a TUIBridge that sends events through the channel
		callback := bridge.NewTUIBridge(m.eventCh)

		// Create cancellable context for this run
		ctx, cancel := context.WithCancel(m.ctx)
		m.cancelFn = cancel

		// Run the agent with callback using cancellable context
		err := m.agentInstance.RunWithCallback(ctx, lastUserMessage, callback)

		// Clear cancelFn after run returns
		m.cancelFn = nil

		if err != nil {
			return bridge.ErrorEvent{Err: err}
		}

		// RunWithCallback will invoke OnComplete via the callback; avoid sending a duplicate complete message.
		return nil
	}
}