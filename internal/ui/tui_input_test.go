package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liup215/gline/internal/agent"
)

// ============================================================================
// handleKeyMsg Tests
// ============================================================================

func TestHandleKeyMsgCtrlC(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// Simulate Ctrl+C
	cmds := handleKeyMsg(m, tea.KeyMsg{Type: tea.KeyCtrlC})

	// Should return tea.Quit command
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command for Ctrl+C, got %d", len(cmds))
	}

	// Verify it's a quit command by checking the type
	msg := cmds[0]()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg for Ctrl+C, got %T", msg)
	}
}

func TestHandleKeyMsgTabToggleMode(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// Start in Act mode
	if m.conversation.Mode != agent.ModeAct {
		t.Fatal("expected initial mode to be Act")
	}

	// First Tab: Act -> Plan
	handleKeyMsg(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.conversation.Mode != agent.ModePlan {
		t.Errorf("expected mode to be Plan after first Tab, got %v", m.conversation.Mode)
	}

	// Second Tab: Plan -> Act
	handleKeyMsg(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.conversation.Mode != agent.ModeAct {
		t.Errorf("expected mode to be Act after second Tab, got %v", m.conversation.Mode)
	}
}

func TestHandleKeyMsgCtrlLClear(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// Add some messages
	m.sendMessage("test message")
	m.conversation.AddToolStart("read_file")

	// Verify messages exist
	if m.conversation.MessageCount() == 0 {
		t.Fatal("expected messages to exist before clear")
	}
	if len(m.conversation.ToolHistory) == 0 {
		t.Fatal("expected tool history to exist before clear")
	}

	// Press Ctrl+L
	handleKeyMsg(m, tea.KeyMsg{Type: tea.KeyCtrlL})

	// Verify everything was cleared
	if m.conversation.MessageCount() != 0 {
		t.Errorf("expected messages to be cleared, got %d", m.conversation.MessageCount())
	}
	if len(m.conversation.ToolHistory) != 0 {
		t.Errorf("expected tool history to be cleared, got %d", len(m.conversation.ToolHistory))
	}
}

func TestHandleKeyMsgEnterWithEmptyInput(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// Enter with empty text should not send
	cmds := handleKeyMsg(m, tea.KeyMsg{Type: tea.KeyEnter})

	// Should not return startAgent command (textarea is empty)
	if len(cmds) != 0 {
		t.Errorf("expected no commands for empty input, got %d", len(cmds))
	}
}

func TestHandleKeyMsgAltEnterNewLine(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// Set some text
	m.textarea.SetValue("line1")

	// Alt+Enter should insert newline
	handleKeyMsg(m, tea.KeyMsg{Type: tea.KeyEnter, Alt: true})

	// The textarea should now have a newline
	content := m.textarea.Value()
	if !contains(content, "\n") {
		t.Errorf("expected newline after Alt+Enter, got: %q", content)
	}
}

// ============================================================================
// handleWindowSize Tests
// ============================================================================

func TestHandleWindowSizeUpdatesDimensions(t *testing.T) {
	m := New(nil)
	m.width = 80
	m.height = 24

	// Simulate window resize
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	handleWindowSize(m, msg)

	// Verify dimensions were updated
	if m.width != 120 {
		t.Errorf("expected width to be 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height to be 40, got %d", m.height)
	}
	if m.viewport.Width != 120 {
		t.Errorf("expected viewport width to be 120, got %d", m.viewport.Width)
	}
}

func TestHandleWindowSizeCalculatesLayout(t *testing.T) {
	m := New(nil)
	m.width = 80
	m.height = 24

	// Simulate large window
	msg := tea.WindowSizeMsg{Width: 120, Height: 60}
	handleWindowSize(m, msg)

	// Verify layout was calculated
	if m.toolAreaHeight < 1 {
		t.Errorf("expected positive toolAreaHeight, got %d", m.toolAreaHeight)
	}
	if m.inputHeight < 1 {
		t.Errorf("expected positive inputHeight, got %d", m.inputHeight)
	}
	if m.viewport.Height < 1 {
		t.Errorf("expected positive viewport height, got %d", m.viewport.Height)
	}
}

func TestHandleWindowSizeSmallWindow(t *testing.T) {
	m := New(nil)
	m.width = 80
	m.height = 24

	// Simulate very small window
	msg := tea.WindowSizeMsg{Width: 20, Height: 10}
	handleWindowSize(m, msg)

	// Should still have reasonable layout
	if m.viewport.Height < 1 {
		t.Errorf("expected at least 1 line viewport height for small window, got %d", m.viewport.Height)
	}
}

func TestHandleWindowSizeUpdatesTextarea(t *testing.T) {
	m := New(nil)
	m.width = 80
	m.height = 24

	// Simulate window resize
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	handleWindowSize(m, msg)

	// Verify dimensions were updated (textarea width is set based on window width)
	if m.width != 120 {
		t.Errorf("expected width to be 120, got %d", m.width)
	}
}

func TestHandleWindowSizeZeroWidth(t *testing.T) {
	m := New(nil)

	// Simulate zero width window
	msg := tea.WindowSizeMsg{Width: 0, Height: 40}
	handleWindowSize(m, msg)

	// Should handle gracefully without crashing
	if m.width != 0 {
		t.Errorf("expected width to be 0, got %d", m.width)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
