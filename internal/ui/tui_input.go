package ui

import (
	"fmt"
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

	// Keep the history list window in range when the terminal shrinks.
	if m.screen == ScreenHistory {
		m.adjustHistoryScroll()
	}

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

	// History screen keyboard handling takes precedence when active
	if m.screen == ScreenHistory {
		return handleHistoryKeyMsg(m, msg)
	}

	// Interactive question picker: while an AskFollowupQuestion with options
	// is pending, arrows/digits/Right drive option selection before any
	// other input handling (so input history and Tab mode-toggle don't
	// swallow the keys). Free-form questions (no options) fall through.
	if len(m.pendingQuestions) > 0 {
		if handled, qcmds := handleQuestionSelection(m, msg); handled {
			// Stop Update from also forwarding this key to the textarea and
			// viewport (it would type characters or scroll the conversation).
			m.consumedPickerKey = true
			return qcmds
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
			// Also set the agent's internal abort flag to break out of any loop
			// that is not watching the context.
			if m.agentInstance != nil {
				m.agentInstance.Abort()
			}
			// Legacy behavior for tests: attempt to send the cancel function into
			// the legacy cancelCh without blocking (channel is buffered).
			select {
			case m.cancelCh <- cancel:
			default:
			}
			// Close all pending questions and clear the queue.
			for _, q := range m.pendingQuestions {
				q.ask.Abort()
			}
			m.pendingQuestions = nil
			// Notify user of interruption
			m.addErrorMessage("✗ Interrupted by user (Esc)")
			// Ensure processing flags updated; agent callback will also handle cleanup
			m.isProcessing = false
			m.isStreaming = false
			m.currentTool = ""
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
			m.agentInstance.SetMode(string(m.conversation.Mode))
		}
		m.updateViewport()

	case tea.KeyEnter:
		if msg.Alt {
			// Alt+Enter for new line
			m.textarea.InsertString("\n")
		} else {
			// If the UI is awaiting replies for AskFollowupQuestion (tool
			// approvals etc.), deliver the answer to the oldest pending one.
			if len(m.pendingQuestions) > 0 {
				m.consumedEnter = true
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

	case tea.KeyUp:
		// Recall the previous prompt when the input is single-line;
		// multi-line content keeps native cursor movement.
		if m.canBrowseHistory() {
			m.prevInputHistory()
		}

	case tea.KeyDown:
		// Recall the next prompt (or restore the draft past the newest).
		if m.canBrowseHistory() {
			m.nextInputHistory()
		}

	case tea.KeyCtrlL:
		// Clear screen
		m.conversation.Clear()
		m.updateViewport()

	case tea.KeyCtrlH:
		// Show history screen
		m.enterHistoryScreen()
	}

	return cmds
}

// handleHistoryKeyMsg handles keyboard input when on the history screen.
func handleHistoryKeyMsg(m *Model, msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	// Deletion confirmation
	if m.historyConfirmID != "" {
		switch msg.Type {
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "y", "Y":
				m.deleteHistoryTask()
			}
			m.historyConfirmID = ""
		case tea.KeyEsc:
			m.historyConfirmID = ""
		}
		return cmds
	}

	// Detail view
	if m.historyDetail != nil {
		switch msg.Type {
		case tea.KeyEsc:
			m.historyDetail = nil
			m.historyMessages = nil
		case tea.KeyEnter:
			m.loadHistoryTask()
		}
		return cmds
	}

	// List view
	switch msg.Type {
	case tea.KeyEsc:
		m.screen = ScreenChat
		m.textarea.Focus()
		cmds = append(cmds, textarea.Blink)

	case tea.KeyUp:
		if m.historySelected > 0 {
			m.historySelected--
			m.adjustHistoryScroll()
		}

	case tea.KeyDown:
		if m.historySelected < len(m.historyTasks)-1 {
			m.historySelected++
			m.adjustHistoryScroll()
		}

	case tea.KeyEnter:
		m.enterHistoryDetail()

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "d", "D":
			if m.historySelected >= 0 && m.historySelected < len(m.historyTasks) {
				m.historyConfirmID = m.historyTasks[m.historySelected].ID
			}
		}
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
	if len(m.pendingQuestions) == 0 {
		return cmds
	}
	q := m.pendingQuestions[0]
	answer := strings.TrimSpace(m.textarea.Value())
	if answer == "" && q.selected >= 0 && q.selected < len(q.options) {
		// Empty input: send the currently highlighted option verbatim.
		answer = q.options[q.selected]
	}
	if answer == "" {
		// Nothing typed and no selectable option: keep the question pending
		// instead of popping it and orphaning the waiting tool goroutine.
		return cmds
	}
	// Non-blocking send through the close-once wrapper (safe if the question
	// was concurrently aborted).
	q.ask.Ask(answer)
	// Pop the answered question; remaining queued approvals stay active.
	m.pendingQuestions = m.pendingQuestions[1:]
	// Clear the input box and keep the UI ready for the next answer (or a
	// new message). While more approvals are queued, stay focused so the user
	// can type the next answer right away; once the queue is empty, blur to
	// mirror submitUserMessage so the Enter key that submitted this answer
	// cannot leak into the textarea as a newline.
	m.textarea.Reset()
	if len(m.pendingQuestions) > 0 {
		m.textarea.Placeholder = pendingReplyPlaceholder(len(m.pendingQuestions))
		m.textarea.Focus()
	} else {
		m.textarea.Placeholder = "Type your message..."
		m.textarea.Blur()
	}
	cmds = append(cmds, textarea.Blink)
	m.updateViewport()
	return cmds
}

// handleQuestionSelection drives the interactive option picker for the oldest
// pending AskFollowupQuestion. It reports whether the key was consumed; keys
// it does not handle fall through to normal input handling (so free-text
// typing and, for option-less questions, history browsing still work).
func handleQuestionSelection(m *Model, msg tea.KeyMsg) (bool, []tea.Cmd) {
	q := m.pendingQuestions[0]
	if len(q.options) == 0 {
		return false, nil // free-form question: no picker
	}
	switch {
	case msg.Type == tea.KeyUp:
		q.selected = (q.selected - 1 + len(q.options)) % len(q.options)
	case msg.Type == tea.KeyDown || msg.Type == tea.KeyTab:
		q.selected = (q.selected + 1) % len(q.options)
	case msg.Type == tea.KeyRight:
		// Load the selected option into the input box so the user can edit
		// it and append free-form text before sending.
		appendOptionToInput(m, q.options[q.selected])
		return true, nil // no re-render needed; input box shows the edit
	case msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] >= '1' && msg.Runes[0] <= '9':
		n := int(msg.Runes[0] - '1')
		if n >= len(q.options) {
			return false, nil // digit beyond the option list: type it as text
		}
		q.selected = n
	default:
		return false, nil
	}
	refreshQuestionSelection(m, q)
	return true, nil
}

// refreshQuestionSelection re-renders the active question message so the
// highlighted option marker follows the keyboard selection.
func refreshQuestionSelection(m *Model, q *pendingQuestion) {
	if msg := m.conversation.GetMessage(q.msgIndex); msg != nil {
		sel := q.selected
		msg.SelectedOption = &sel
	}
	m.convVM.MarkMessageDirty(q.msgIndex)
	m.updateViewport()
}

// appendOptionToInput puts the option text at the end of the input box
// (separated by a space) without duplicating it, positioning the cursor at
// the end so the user can keep typing free-form text after it.
func appendOptionToInput(m *Model, opt string) {
	cur := strings.TrimRight(m.textarea.Value(), " ")
	switch {
	case cur == "":
		m.textarea.SetValue(opt + " ")
	case cur == opt || strings.HasSuffix(cur, " "+opt):
		m.textarea.SetValue(cur + " ")
	default:
		m.textarea.SetValue(cur + " " + opt + " ")
	}
	m.textarea.SetCursor(len(m.textarea.Value()))
}

func submitUserMessage(m *Model) []tea.Cmd {
	var cmds []tea.Cmd
	input := strings.TrimSpace(m.textarea.Value())
	if input != "" && !m.isProcessing {
		m.sendMessage(input)
		m.addToInputHistory(input)
		m.resetHistoryBrowsing()
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

// pendingReplyPlaceholder describes the interactive question picker in the
// input box. With parallel tool calls several approvals can be waiting at
// once, so the user can also see how many are left to answer.
func pendingReplyPlaceholder(remaining int) string {
	hint := "↑/↓ choose · Enter send · → edit, or type your own answer"
	if remaining > 1 {
		hint += fmt.Sprintf(" (%d pending)", remaining)
	}
	return hint
}
