package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/slash"
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

	// Reserve a small buffer for the slash menu (when it appears, it won't cause overflow).
	// The slash menu typically shows ~5 items + borders = ~7-8 lines.
	// We subtract this from viewport height so total layout stays within terminal height.
	// Only reserve this buffer when slash menu is actually active; otherwise we leave
	// the space for the input box + status bar + help to sit flush at the bottom.
	if m.slashMenu != nil && m.slashMenu.Active {
		menuBuffer := 8
		if viewportH > menuBuffer+3 {
			viewportH -= menuBuffer
		}
	}

	// Set viewport height
	m.viewport.Height = viewportH

	// Update textarea height
	m.textarea.SetHeight(inputH)

	// Compute inner width available for textarea content (subtract border + horizontal padding).
	// inputBoxStyle has Padding(0, 3) which gives 3 cols on left and right, border (2 cols).
	innerWidth := msg.Width - 8
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

	// Slash menu navigation takes precedence when active
	if m.slashMenu != nil && m.slashMenu.Active {
		switch msg.Type {
		case tea.KeyEsc:
			m.slashMenu.ExitSlashMode()
			return cmds

		case tea.KeyTab, tea.KeyDown:
			m.slashMenu.Next()
			return cmds

		case tea.KeyUp:
			m.slashMenu.Prev()
			return cmds

		case tea.KeyEnter:
			cmd := m.slashMenu.SelectedCommand()
			if cmd != nil {
				// Insert the selected command into the textarea
				m.textarea.SetValue("/" + cmd.Name + " ")
				m.textarea.SetCursor(len(m.textarea.Value()))
			}
			m.slashMenu.ExitSlashMode()
			return cmds

		case tea.KeyCtrlC:
			m.slashMenu.ExitSlashMode()
			cmds = append(cmds, tea.Quit)
			return cmds
		}
	}

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
				// Check if this is a standalone slash command and execute it immediately
				input := strings.TrimSpace(m.textarea.Value())
				if slash.IsStandaloneCommand(input) {
					cmds = append(cmds, executeSlashCommand(m, input)...)
				} else {
					// Normal send-message behavior
					cmds = append(cmds, submitUserMessage(m)...)
				}
			}
		}

	case tea.KeyCtrlL:
		// Clear screen
		m.conversation.Clear()
		m.updateViewport()
	}

	return cmds
}

// executeSlashCommand runs a standalone slash command immediately.
func executeSlashCommand(m *Model, input string) []tea.Cmd {
	var cmds []tea.Cmd
	name, args := slash.ParseCommand(input)
	if name == "" {
		return cmds
	}

	cmd, ok := m.slashMenu.Registry.Get(name)
	if !ok {
		m.addErrorMessage("Unknown command: /" + name)
		m.textarea.Reset()
		cmds = append(cmds, textarea.Blink)
		m.updateViewport()
		return cmds
	}

	consumed, err := cmd.Handler(args)
	if err != nil {
		m.addErrorMessage("Error executing /" + name + ": " + err.Error())
	}

	if consumed {
		m.textarea.Reset()
		cmds = append(cmds, textarea.Blink)
	}

	// Handle command results that affect the TUI
	// Note: The actual result handling (clear, quit, etc.) is wired via the CommandContext
	// set up in New(). We trigger a viewport refresh to show any system messages.
	m.updateViewport()
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
			// receiver not ready or channel full/Closed — drop answer safely
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
		// Clear the textarea value but keep the view rendered and at the
		// configured height. Blur to indicate the input is temporarily
		// disabled while the agent processes the message. Avoid calling
		// Reset() here because in some terminal/styling combinations that
		// can lead to the input box collapsing/vanishing visually.
		m.textarea.SetValue("")
		m.textarea.Placeholder = "Type your message..."
		m.textarea.Blur()
		// Start the agent with callback
		cmds = append(cmds, m.startAgent())
	}
	return cmds
}
