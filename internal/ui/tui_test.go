package ui

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liup215/gline/internal/ui/bridge"
	uimodel "github.com/liup215/gline/internal/ui/model"
)

// stripANSI removes ANSI escape sequences from a string for test assertions.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestContentUpdateSurvivesToolStatus(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	m.sendMessage("read the file")
	m.isProcessing = true // Set processing so tool status appears

	// Simulate tool start via bridge event - creates system message for tool
	m.Update(bridge.ToolStartEvent{Name: "read_file", Input: "{}"})

	// Simulate content arriving from the LLM
	updatedModel, _ := m.Update(bridge.ContentEvent{Delta: "text from model"})
	updated := updatedModel.(*Model)

	// Without system messages, the assistant is the last message
	msgs := updated.conversation.Messages
	if updated.activeAssistantIndex != len(msgs)-1 {
		t.Fatalf("expected assistant slot to be last message, got index %d with %d messages", updated.activeAssistantIndex, len(msgs))
	}

	assistant := msgs[updated.activeAssistantIndex]
	if !strings.Contains(assistant.Content, "text from model") {
		t.Fatalf("assistant content was not updated: %q", assistant.Content)
	}

	view := stripANSI(updated.View())
	if !strings.Contains(view, "text from model") {
		t.Fatalf("view missing assistant output: %q", view)
	}
	if !strings.Contains(view, "You: read the file") {
		t.Fatalf("view missing user output: %q", view)
	}

	// Tool status appears as system message (🔧 icon indicates tool)
	if !strings.Contains(view, "🔧") {
		t.Fatalf("view missing tool indicator (🔧): %q", view)
	}
}

func TestToolStatusArea(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// Verify initial state - no current tool
	if m.currentTool != "" {
		t.Fatalf("expected no current tool initially, got: %q", m.currentTool)
	}

	// Simulate tool start - should set currentTool
	m.Update(bridge.ToolStartEvent{Name: "read_file", Input: "{}"})
	if m.currentTool != "read_file" {
		t.Fatalf("expected currentTool='read_file', got: %q", m.currentTool)
	}

	// Verify tool appears in conversation as system message
	view := stripANSI(m.View())
	if !strings.Contains(view, "🔧") {
		t.Fatalf("expected tool indicator (🔧) in view, got: %q", view)
	}

	// Mark tool as completed - should clear currentTool
	m.Update(bridge.ToolCompleteEvent{Name: "read_file", Result: "done"})
	if m.currentTool != "" {
		t.Fatalf("expected currentTool cleared after complete, got: %q", m.currentTool)
	}
}

func TestToolHistoryDoesNotPushContent(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	m.sendMessage("do multiple things")

	// Add many tool statuses — these should NOT create system messages
	for i := 0; i < 10; i++ {
		m.conversation.ToolHistory = append(m.conversation.ToolHistory, uimodel.ToolStatus{
			Name:   "tool_" + string(rune('A'+i)),
			Status: "completed",
		})
	}

	// Add content to assistant
	m.Update(bridge.ContentEvent{Delta: "final answer"})

	// Only 2 messages: user + assistant (no system messages for tools)
	msgCount := len(m.conversation.Messages)
	if msgCount != 2 {
		t.Fatalf("expected 2 messages (user + assistant), got %d", msgCount)
	}

	// Tool history should have 10 entries
	if len(m.conversation.ToolHistory) != 10 {
		t.Fatalf("expected 10 tool history entries, got %d", len(m.conversation.ToolHistory))
	}

	// View should contain the assistant content
	view := stripANSI(m.View())
	if !strings.Contains(view, "final answer") {
		t.Fatalf("view missing assistant output: %q", view)
	}

	// Note: Tool history is no longer displayed in the view directly.
	// Current tool is shown via CompactBar when isProcessing is true.
	// Tool history is kept for internal tracking but not rendered in the TUI.
}

func TestWindowSizeUpdateKeepsModelUsable(t *testing.T) {
	m := New(nil)
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := updatedModel.(*Model)

	if updated.width != 120 || updated.height != 40 {
		t.Fatalf("unexpected dimensions: width=%d height=%d", updated.width, updated.height)
	}
	if updated.viewport.Width != 120 {
		t.Fatalf("unexpected viewport width: %d", updated.viewport.Width)
	}
	if updated.viewport.Height <= 0 {
		t.Fatalf("expected positive viewport height, got %d", updated.viewport.Height)
	}
}
