package ui

import (
"strings"
"github.com/liup215/gline/internal/agent"
 
"github.com/charmbracelet/bubbles/textarea"
tea "github.com/charmbracelet/bubbletea"
)

// handleWindowSize moves WindowSizeMsg handling out of Update.
func handleWindowSize(m *Model, msg tea.WindowSizeMsg) []tea.Cmd {
	// Update dimensions and textarea width, then refresh viewport.
	m.width = msg.Width
	m.height = msg.Height
	m.viewport.Width = msg.Width

	// Reserve space for: title (1) + tool area (toolAreaHeight) + input (inputHeight) + status bar (1) + help (1)
	m.viewport.Height = msg.Height - m.inputHeight - m.toolAreaHeight - 4
	if m.viewport.Height < 3 {
		m.viewport.Height = 3
	}

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
			if m.cancelFn != nil {
				m.cancelFn()
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
		if m.mode == agent.ModePlan {
			m.mode = agent.ModeAct
		} else {
			m.mode = agent.ModePlan
		}
		if m.agentInstance != nil {
			m.agentInstance.SetMode(m.mode)
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
		m.messages = []Message{}
		m.toolHistory = nil
		m.updateViewport()
	}

	return cmds
}

func submitPendingReply(m *Model) []tea.Cmd {
	var cmds []tea.Cmd
	answer := strings.TrimSpace(m.textarea.Value())
	if answer != "" {
		// Send the user's answer back to the waiting tool goroutine.
		m.pendingReply <- answer
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