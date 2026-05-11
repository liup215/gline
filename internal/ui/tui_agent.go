package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/ui/bridge"
	"github.com/liup215/gline/pkg/types"
)

// tuiCallback implements the agent.StreamCallback interface
type tuiCallback struct {
	program *tea.Program
}

func (c *tuiCallback) OnStreamStart() {
	if c.program != nil {
		c.program.Send(bridge.StreamStartEvent{})
	}
}

func (c *tuiCallback) OnContent(delta string) {
	if c.program != nil {
		c.program.Send(bridge.ContentEvent{Delta: delta})
	}
}

func (c *tuiCallback) OnToolCallStart(toolCall agent.ToolCall) {
	if c.program != nil {
		c.program.Send(bridge.ToolStartEvent{
			Name:  toolCall.Name,
			Input: toolCall.Input,
		})
	}
}

func (c *tuiCallback) OnToolCallComplete(toolCall agent.ToolCall, result string) {
	if c.program != nil {
		c.program.Send(bridge.ToolCompleteEvent{Name: toolCall.Name, Result: result})
	}
}

func (c *tuiCallback) OnError(err error) {
	if c.program != nil {
		c.program.Send(bridge.ErrorEvent{Err: err})
	}
}

func (c *tuiCallback) OnComplete() {
	if c.program != nil {
		c.program.Send(bridge.CompleteEvent{})
	}
}

func (c *tuiCallback) AskFollowupQuestion(question string, options []string) (string, error) {
	if c.program == nil {
		return "", fmt.Errorf("no program")
	}
	reply := make(chan string, 1)
	// Send ask question event with reply channel
	c.program.Send(bridge.AskQuestionEvent{Question: question, Options: options, Reply: reply})
	// Wait for the UI to provide the answer
	answer := <-reply
	return answer, nil
}

// startAgent starts the agent with the TUI callback
func (m *Model) startAgent() tea.Cmd {
	return func() tea.Msg {
		if m.agentInstance == nil {
			return bridge.ErrorEvent{Err: fmt.Errorf("agent not initialized")}
		}

		// Get the last user message
		var lastUserMessage string
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Role == types.RoleUser {
				lastUserMessage = m.messages[i].Content
				break
			}
		}

		if lastUserMessage == "" {
			return bridge.ErrorEvent{Err: fmt.Errorf("no user message found")}
		}

		// Create callback with program reference
		callback := &tuiCallback{program: m.program}

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