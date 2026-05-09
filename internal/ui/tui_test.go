package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestContentUpdateSurvivesSystemMessages(t *testing.T) {
	model := New(nil)
	model.width = 100
	model.height = 30

	model.sendMessage("read the file")
	model.addSystemMessage("🔧 Running: read_file")

	updatedModel, _ := model.Update(agentUpdateMsg{updateType: "content", content: "tool text from model"})
	updated := updatedModel.(*Model)

	if updated.activeAssistantIndex != len(updated.messages)-2 {
		t.Fatalf("expected assistant slot to remain stable, got index %d with %d messages", updated.activeAssistantIndex, len(updated.messages))
	}

	assistant := updated.messages[updated.activeAssistantIndex]
	if !strings.Contains(assistant.Content, "tool text from model") {
		t.Fatalf("assistant content was not updated: %q", assistant.Content)
	}

	view := updated.View()
	if !strings.Contains(view, "AI: tool text from model") {
		t.Fatalf("view missing assistant output: %q", view)
	}
	if !strings.Contains(view, "You: read the file") {
		t.Fatalf("view missing user output: %q", view)
	}
	if !strings.Contains(view, "Running: read_file") {
		t.Fatalf("view missing system output: %q", view)
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