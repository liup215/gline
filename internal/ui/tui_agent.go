package ui

import (
"context"
"fmt"

tea "github.com/charmbracelet/bubbletea"
"github.com/liup215/gline/internal/agent"
"github.com/liup215/gline/pkg/types"
)

// tuiCallback implements the agent.StreamCallback interface
type tuiCallback struct {
	program *tea.Program
}

func (c *tuiCallback) OnStreamStart() {
	if c.program != nil {
		c.program.Send(agentUpdateMsg{updateType: "streamStart"})
	}
}

func (c *tuiCallback) OnContent(delta string) {
	if c.program != nil {
		c.program.Send(agentUpdateMsg{updateType: "content", content: delta})
	}
}

func (c *tuiCallback) OnToolCallStart(toolCall agent.ToolCall) {
	if c.program != nil {
		c.program.Send(agentUpdateMsg{
			updateType: "toolStart",
			toolName:   toolCall.Name,
			toolInput:  toolCall.Input,
		})
	}
}

func (c *tuiCallback) OnToolCallComplete(toolCall agent.ToolCall, result string) {
	if c.program != nil {
		c.program.Send(agentUpdateMsg{updateType: "toolComplete", toolName: toolCall.Name, toolResult: result})
	}
}

func (c *tuiCallback) OnError(err error) {
	if c.program != nil {
		c.program.Send(agentUpdateMsg{updateType: "error", err: err})
	}
}

func (c *tuiCallback) OnComplete() {
	if c.program != nil {
		c.program.Send(agentUpdateMsg{updateType: "complete"})
	}
}

func (c *tuiCallback) AskFollowupQuestion(question string, options []string) (string, error) {
	if c.program == nil {
		return "", fmt.Errorf("no program")
	}
	reply := make(chan string, 1)
	// Send ask question message with reply channel
	c.program.Send(askQuestionMsg{Question: question, Options: options, Reply: reply})
	// Wait for the UI to provide the answer
	answer := <-reply
	return answer, nil
}

// startAgent starts the agent with the TUI callback
func (m *Model) startAgent() tea.Cmd {
	return func() tea.Msg {
		if m.agentInstance == nil {
			return agentUpdateMsg{updateType: "error", err: fmt.Errorf("agent not initialized")}
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
			return agentUpdateMsg{updateType: "error", err: fmt.Errorf("no user message found")}
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
			return agentUpdateMsg{updateType: "error", err: err}
		}

		// RunWithCallback will invoke OnComplete via the callback; avoid sending a duplicate complete message.
		return nil
	}
}