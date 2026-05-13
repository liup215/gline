package ui

import (
"strings"
"github.com/liup215/gline/internal/agent"
 
"github.com/charmbracelet/bubbles/textarea"
tea "github.com/charmbracelet/bubbletea"
)

// handleWindowSize moves WindowSizeMsg handling out of Update.
func handleWindowSize(m *Model, msg tea.WindowSizeMsg) []tea.Cmd {
	// Update dimensions
	m.width = msg.Width
	m.height = msg.Height
	m.viewport.Width = msg.Width

	// Calculate flexible layout
	viewportH, toolH, inputH := calculateLayout(msg.Height)
	m.toolAreaHeight = toolH
	m.inputHeight = inputH

	// Set viewport height
	m.viewport.Height = viewportH

	// Update textarea height
	m.textarea.SetHeight(inputH)

	// Compute inner width available for textarea content (subtract border + horizontal padding and left margin).
	// inputBoxStyle has Padding(0, 3) which gives 3 cols on left and right, border (2 cols), plus we render with a left margin of 1.
	innerWidth := msg.Width - 9
	if innerWidth < 10 {
		innerWidth = 10
	}
	m.textarea.SetWidth(innerWidth)

	m.updateViewport()
	return nil
}

// handleKeyMsg extracts keyboard-driven state transitions from Update.
func handleKeyMsg(m *Model, msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	switch msg.Type {
	case tea.KeyCtrlC:
		// Quit the program
		cmds = append(cmds, tea.Quit)

case tea.KeyEsc:
if m.isProcessing {
// Cancel the running agent via protected cancel func (broadcast via context).
m.cancelLock.RLock()
cancel := m.agentCancel
m.cancelLock.RUnlock()
if cancel != nil {
cancel()
}
 // Legacy behavior for tests: attempt to send the cancel function into
 // the legacy cancelCh without blocking (channel is buffered).
 select {
 case m.cancelCh <- cancel:
 default:
 }
 // Close pending reply channel and clear reference.
 if m.pendingReply != nil {
 close(m.pendingReply)
 m.pendingReply = nil
 }
// Notify user of interruption
m.addErrorMessage("✗ Interrupted by user (Esc)")
// Ensure processing flags updated; agent callback will also handle cleanup
m.isProcessing = false
m.isStreaming = false
m.textarea.Focus()
cmds = append(cmds, textarea.Blink)
m.updateViewport()
} else {
m.textarea.Reset()
m.updateViewport()
}

	case tea.KeyTab:
		// Toggle between Plan and Act mode
		if m.conversation.Mode == agent.ModePlan {
			m.conversation.Mode = agent.ModeAct
		} else {
			m.conversation.Mode = agent.ModePlan
		}
		if m.agentInstance != nil {
			m.agentInstance.SetMode(m.conversation.Mode)
		}
		m.updateViewport()

	case tea.KeyEnter:
		if msg.Alt {
			// Alt+Enter for new line
			m.textarea.InsertString("\n")
		} else {
			// If the UI is awaiting a reply for AskFollowupQuestion, deliver it instead of starting the agent.
			if m.pendingReply != nil {
				cmds = append(cmds, submitPendingReply(m)...)
			} else {
				// Normal send-message behavior
				cmds = append(cmds, submitUserMessage(m)...)
			}
		}

	case tea.KeyCtrlL:
		// Clear screen
		m.conversation.Clear()
		m.updateViewport()
	}

	return cmds
}

func submitPendingReply(m *Model) []tea.Cmd {
var cmds []tea.Cmd
answer := strings.TrimSpace(m.textarea.Value())
if answer != "" && m.pendingReply != nil {
// Non-blocking send to avoid blocking or panic if channel closed.
select {
case m.pendingReply <- answer:
// sent
default:
// receiver not ready or channel full/closed — drop answer safely
}
}
// Clear pending state and reset input box without starting a new agent run.
m.pendingReply = nil
m.textarea.Reset()
m.textarea.Placeholder = "Type your message..."
m.textarea.Focus()
cmds = append(cmds, textarea.Blink)
m.updateViewport()
return cmds
}

func submitUserMessage(m *Model) []tea.Cmd {
	var cmds []tea.Cmd
	input := strings.TrimSpace(m.textarea.Value())
	if input != "" && !m.isProcessing {
		m.sendMessage(input)
		m.textarea.Reset()
		m.textarea.Blur()
		// Start the agent with callback
		cmds = append(cmds, m.startAgent())
	}
	return cmds
}