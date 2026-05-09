package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestContentUpdateSurvivesToolStatus(t *testing.T) {
	model := New(nil)
	model.width = 100
	model.height = 30

	model.sendMessage("read the file")

	// Simulate tool start via agentUpdateMsg (tool status goes to toolHistory, not messages)
	model.Update(agentUpdateMsg{updateType: "toolStart", toolName: "read_file"})

	// Simulate content arriving from the LLM
	updatedModel, _ := model.Update(agentUpdateMsg{updateType: "content", content: "text from model"})
	updated := updatedModel.(*Model)

	// Without system messages, the assistant is the last message
	if updated.activeAssistantIndex != len(updated.messages)-1 {
		t.Fatalf("expected assistant slot to be last message, got index %d with %d messages", updated.activeAssistantIndex, len(updated.messages))
	}

	assistant := updated.messages[updated.activeAssistantIndex]
	if !strings.Contains(assistant.Content, "text from model") {
		t.Fatalf("assistant content was not updated: %q", assistant.Content)
	}

	view := updated.View()
	if !strings.Contains(view, "AI: text from model") {
		t.Fatalf("view missing assistant output: %q", view)
	}
	if !strings.Contains(view, "You: read the file") {
		t.Fatalf("view missing user output: %q", view)
	}

	// Tool status should appear in the tool area
	if !strings.Contains(view, "read_file") {
		t.Fatalf("view missing tool status: %q", view)
	}
}

func TestToolStatusArea(t *testing.T) {
	model := New(nil)
	model.width = 100
	model.height = 30

	// No tools active — should show just a border
	view := model.View()
	if !strings.Contains(view, "──") {
		t.Fatalf("expected border line when no tools active, got: %q", view)
	}

	// Add a running tool
	model.toolHistory = append(model.toolHistory, ToolStatus{
		Name:   "read_file",
		Status: "running",
	})
	view = model.View()
	if !strings.Contains(view, "read_file") || !strings.Contains(view, "⏳") {
		t.Fatalf("expected running tool indicator, got: %q", view)
	}

	// Mark tool as completed
	model.toolHistory[0].Status = "completed"
	view = model.View()
	if !strings.Contains(view, "read_file") || !strings.Contains(view, "✓") {
		t.Fatalf("expected completed tool indicator, got: %q", view)
	}

	// Mark tool as failed
	model.toolHistory[0].Status = "failed"
	view = model.View()
	if !strings.Contains(view, "read_file") || !strings.Contains(view, "✗") {
		t.Fatalf("expected failed tool indicator, got: %q", view)
	}
}

func TestToolHistoryDoesNotPushContent(t *testing.T) {
	model := New(nil)
	model.width = 100
	model.height = 30

	model.sendMessage("do multiple things")

	// Add many tool statuses — these should NOT create system messages
	for i := 0; i < 10; i++ {
		model.toolHistory = append(model.toolHistory, ToolStatus{
			Name:   "tool_" + string(rune('A'+i)),
			Status: "completed",
		})
	}

	// Add content to assistant
	model.Update(agentUpdateMsg{updateType: "content", content: "final answer"})

	// Only 2 messages: user + assistant (no system messages for tools)
	msgCount := len(model.messages)
	if msgCount != 2 {
		t.Fatalf("expected 2 messages (user + assistant), got %d", msgCount)
	}

	// Tool history should have 10 entries
	if len(model.toolHistory) != 10 {
		t.Fatalf("expected 10 tool history entries, got %d", len(model.toolHistory))
	}

	// View should contain both the assistant content and tool status
	// Tool area shows only the most recent 4 entries (toolAreaHeight=5, minus 1 for border)
	view := model.View()
	if !strings.Contains(view, "AI: final answer") {
		t.Fatalf("view missing assistant output: %q", view)
	}
	if !strings.Contains(view, "tool_J") {
		t.Fatalf("view missing latest tool status (tool_J), got: %q", view)
	}
}

func TestWindowSizeUpdateKeepsModelUsable(t *testing.T) {
	model := New(nil)
	updatedModel, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
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