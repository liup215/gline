package model

import (
	"testing"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/pkg/types"
)

func TestNewConversation(t *testing.T) {
	c := NewConversation()
	if c == nil {
		t.Fatal("NewConversation returned nil")
	}
	if len(c.Messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(c.Messages))
	}
	if len(c.ToolHistory) != 0 {
		t.Fatalf("expected 0 tool history, got %d", len(c.ToolHistory))
	}
	if c.Mode != agent.ModeAct {
		t.Fatalf("expected ModeAct, got %v", c.Mode)
	}
}

func TestAppendMessage(t *testing.T) {
	c := NewConversation()
	idx := c.AppendMessage(Message{
		Role:    types.RoleUser,
		Content: "hello",
	})
	if idx != 0 {
		t.Fatalf("expected index 0, got %d", idx)
	}
	if c.MessageCount() != 1 {
		t.Fatalf("expected 1 message, got %d", c.MessageCount())
	}

	idx2 := c.AppendMessage(Message{
		Role:    types.RoleAssistant,
		Content: "hi",
	})
	if idx2 != 1 {
		t.Fatalf("expected index 1, got %d", idx2)
	}
}

func TestGetMessage(t *testing.T) {
	c := NewConversation()
	if c.GetMessage(0) != nil {
		t.Fatal("expected nil for empty conversation")
	}
	c.AppendMessage(Message{Role: types.RoleUser, Content: "test"})
	m := c.GetMessage(0)
	if m == nil {
		t.Fatal("expected message, got nil")
	}
	if m.Content != "test" {
		t.Fatalf("expected 'test', got %q", m.Content)
	}
	if c.GetMessage(-1) != nil || c.GetMessage(99) != nil {
		t.Fatal("expected nil for out-of-bounds indices")
	}
}

func TestUpdateMessageContent(t *testing.T) {
	c := NewConversation()
	c.AppendMessage(Message{Role: types.RoleAssistant, Content: "hello"})
	c.UpdateMessageContent(0, " world")
	if c.Messages[0].Content != "hello world" {
		t.Fatalf("expected 'hello world', got %q", c.Messages[0].Content)
	}
	// Out of bounds should be no-op
	c.UpdateMessageContent(-1, "x")
	c.UpdateMessageContent(99, "x")
}

func TestSetMessageContent(t *testing.T) {
	c := NewConversation()
	c.AppendMessage(Message{Role: types.RoleAssistant, Content: "old"})
	c.SetMessageContent(0, "new")
	if c.Messages[0].Content != "new" {
		t.Fatalf("expected 'new', got %q", c.Messages[0].Content)
	}
	// Out of bounds should be no-op
	c.SetMessageContent(-1, "x")
	c.SetMessageContent(99, "x")
}

func TestLastUserMessage(t *testing.T) {
	c := NewConversation()
	_, ok := c.LastUserMessage()
	if ok {
		t.Fatal("expected false for empty conversation")
	}
	c.AppendMessage(Message{Role: types.RoleSystem, Content: "system"})
	_, ok = c.LastUserMessage()
	if ok {
		t.Fatal("expected false when no user message")
	}
	c.AppendMessage(Message{Role: types.RoleUser, Content: "user1"})
	c.AppendMessage(Message{Role: types.RoleAssistant, Content: "assistant"})
	c.AppendMessage(Message{Role: types.RoleUser, Content: "user2"})
	last, ok := c.LastUserMessage()
	if !ok || last != "user2" {
		t.Fatalf("expected 'user2', got %q (ok=%v)", last, ok)
	}
}

func TestClear(t *testing.T) {
	c := NewConversation()
	c.AppendMessage(Message{Role: types.RoleUser, Content: "hello"})
	c.AddToolStart("tool")
	c.Clear()
	if len(c.Messages) != 0 || len(c.ToolHistory) != 0 {
		t.Fatalf("expected empty after Clear, got %d messages, %d tools", len(c.Messages), len(c.ToolHistory))
	}
}

func TestAddToolStart(t *testing.T) {
	c := NewConversation()
	c.AddToolStart("read_file")
	if len(c.ToolHistory) != 1 {
		t.Fatalf("expected 1 tool history entry, got %d", len(c.ToolHistory))
	}
	if c.ToolHistory[0].Name != "read_file" || c.ToolHistory[0].Status != "running" {
		t.Fatalf("unexpected tool status: %+v", c.ToolHistory[0])
	}
	if c.ToolHistory[0].StartTime.IsZero() {
		t.Fatal("expected non-zero StartTime")
	}
}

func TestMarkToolComplete(t *testing.T) {
	c := NewConversation()
	c.AddToolStart("tool1")
	c.AddToolStart("tool2")
	c.MarkToolComplete("tool2")
	if c.ToolHistory[0].Status != "running" {
		t.Fatalf("expected tool1 still running, got %s", c.ToolHistory[0].Status)
	}
	if c.ToolHistory[1].Status != "completed" {
		t.Fatalf("expected tool2 completed, got %s", c.ToolHistory[1].Status)
	}
	// Marking non-existent tool should be no-op
	c.MarkToolComplete("nonexistent")
}

func TestMarkToolFailed(t *testing.T) {
	c := NewConversation()
	c.AddToolStart("tool1")
	c.MarkToolFailed("tool1")
	if c.ToolHistory[0].Status != "failed" {
		t.Fatalf("expected failed, got %s", c.ToolHistory[0].Status)
	}
}

func TestClearToolHistory(t *testing.T) {
	c := NewConversation()
	c.AddToolStart("tool1")
	c.ClearToolHistory()
	if len(c.ToolHistory) != 0 {
		t.Fatalf("expected 0 tool history after clear, got %d", len(c.ToolHistory))
	}
}

func TestLastRunningToolName(t *testing.T) {
	c := NewConversation()
	if c.LastRunningToolName() != "" {
		t.Fatal("expected empty string for no running tools")
	}
	c.AddToolStart("tool1")
	c.AddToolStart("tool2")
	c.MarkToolComplete("tool2")
	if c.LastRunningToolName() != "tool1" {
		t.Fatalf("expected 'tool1', got %q", c.LastRunningToolName())
	}
	c.MarkToolComplete("tool1")
	if c.LastRunningToolName() != "" {
		t.Fatalf("expected empty after all completed, got %q", c.LastRunningToolName())
	}
}

func TestMessageResetRenderCache(t *testing.T) {
	m := Message{
		Content:           "hello",
		Rendered:          "rendered",
		RenderedWrapWidth: 80,
		RenderedSource:    "hello",
	}
	m.ResetRenderCache()
	if m.Rendered != "" || m.RenderedWrapWidth != 0 || m.RenderedSource != "" {
		t.Fatal("ResetRenderCache did not clear all fields")
	}
}

func TestConversationComplexScenario(t *testing.T) {
	c := NewConversation()
	c.Provider = "openai"
	c.ModelName = "gpt-4"
	c.Mode = agent.ModePlan

	// User asks a question
	c.AppendMessage(Message{Role: types.RoleUser, Content: "read file"})

	// Assistant starts responding
	idx := c.AppendMessage(Message{Role: types.RoleAssistant, Content: ""})
	c.UpdateMessageContent(idx, "Let me")
	c.UpdateMessageContent(idx, " check")

	// Tool is invoked
	c.AddToolStart("read_file")
	c.MarkToolComplete("read_file")

	// More assistant content
	c.UpdateMessageContent(idx, " the file.")

	if c.MessageCount() != 2 {
		t.Fatalf("expected 2 messages, got %d", c.MessageCount())
	}
	if c.Messages[1].Content != "Let me check the file." {
		t.Fatalf("unexpected assistant content: %q", c.Messages[1].Content)
	}
	if len(c.ToolHistory) != 1 || c.ToolHistory[0].Status != "completed" {
		t.Fatal("unexpected tool history")
	}
	if c.Provider != "openai" || c.ModelName != "gpt-4" || c.Mode != agent.ModePlan {
		t.Fatal("metadata was corrupted")
	}

	// Clear and verify
	c.Clear()
	if c.MessageCount() != 0 || len(c.ToolHistory) != 0 {
		t.Fatal("Clear did not reset properly")
	}
	// Metadata should NOT be cleared
	if c.Provider != "openai" || c.ModelName != "gpt-4" {
		t.Fatal("metadata was incorrectly cleared")
	}
}

func TestToolHistoryMultipleSameName(t *testing.T) {
	c := NewConversation()
	c.AddToolStart("tool")
	c.AddToolStart("tool")
	c.MarkToolComplete("tool")
	// Should mark the MOST RECENT running tool
	if c.ToolHistory[0].Status != "running" {
		t.Fatalf("expected first tool still running, got %s", c.ToolHistory[0].Status)
	}
	if c.ToolHistory[1].Status != "completed" {
		t.Fatalf("expected second tool completed, got %s", c.ToolHistory[1].Status)
	}
}

func TestMessageCount(t *testing.T) {
	c := NewConversation()
	if c.MessageCount() != 0 {
		t.Fatalf("expected 0, got %d", c.MessageCount())
	}
	c.AppendMessage(Message{})
	c.AppendMessage(Message{})
	if c.MessageCount() != 2 {
		t.Fatalf("expected 2, got %d", c.MessageCount())
	}
}

func TestSetMessageContentOutOfBounds(t *testing.T) {
	c := NewConversation()
	// These should not panic
	c.SetMessageContent(-1, "x")
	c.SetMessageContent(0, "x")
	c.SetMessageContent(1, "x")
}

func TestUpdateMessageContentOutOfBounds(t *testing.T) {
	c := NewConversation()
	c.UpdateMessageContent(-1, "x")
	c.UpdateMessageContent(0, "x")
	c.UpdateMessageContent(1, "x")
	// No panic = pass
}
