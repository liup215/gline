package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/liup215/gline/internal/ui/bridge"
	uimodel "github.com/liup215/gline/internal/ui/model"
	"github.com/liup215/gline/pkg/types"
)

// ---------------------------------------------------------------------------
// Phase 7: Independent handler unit tests
// Each test verifies that the handler mutates Model state correctly and
// returns the expected needsRefresh flag, WITHOUT calling updateViewport().
// ---------------------------------------------------------------------------

func TestHandleAgentContentReturnsNeedsRefresh(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// Set up an active assistant slot
	m.activeAssistantIndex = m.conversation.AppendMessage(uimodel.Message{
		Role:      types.RoleAssistant,
		Content:   "",
		Timestamp: time.Now(),
	})

	needsRefresh, cmds := handleAgentContent(m, bridge.ContentEvent{Delta: "hello"})

	if !needsRefresh {
		t.Error("expected needsRefresh=true after content update")
	}
	if len(cmds) != 0 {
		t.Errorf("expected no cmds, got %d", len(cmds))
	}
	if m.conversation.Messages[m.activeAssistantIndex].Content != "hello" {
		t.Errorf("expected content 'hello', got %q", m.conversation.Messages[m.activeAssistantIndex].Content)
	}
}

func TestHandleAgentContentCreatesAssistantSlotWhenMissing(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// No active assistant slot — handler should create one
	m.activeAssistantIndex = -1
	needsRefresh, _ := handleAgentContent(m, bridge.ContentEvent{Delta: "new content"})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	if m.activeAssistantIndex < 0 {
		t.Error("expected activeAssistantIndex to be set")
	}
	if m.conversation.Messages[m.activeAssistantIndex].Content != "new content" {
		t.Errorf("expected content 'new content', got %q", m.conversation.Messages[m.activeAssistantIndex].Content)
	}
}

func TestHandleAgentContentAppendsDelta(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	m.activeAssistantIndex = m.conversation.AppendMessage(uimodel.Message{
		Role:      types.RoleAssistant,
		Content:   "Hello",
		Timestamp: time.Now(),
	})

	// Append delta
	handleAgentContent(m, bridge.ContentEvent{Delta: " world"})

	if m.conversation.Messages[m.activeAssistantIndex].Content != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", m.conversation.Messages[m.activeAssistantIndex].Content)
	}
}

func TestHandleAgentToolStartReturnsNeedsRefresh(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	needsRefresh, cmds := handleAgentToolStart(m, bridge.ToolStartEvent{
		Name:  "read_file",
		Input: `{"path": "test.txt"}`,
	})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	if len(cmds) != 0 {
		t.Errorf("expected no cmds, got %d", len(cmds))
	}
	if m.currentTool != "read_file" {
		t.Errorf("expected currentTool='read_file', got %q", m.currentTool)
	}
	if len(m.conversation.ToolHistory) != 1 {
		t.Errorf("expected 1 tool history entry, got %d", len(m.conversation.ToolHistory))
	}
	if m.conversation.ToolHistory[0].Name != "read_file" {
		t.Errorf("expected tool name 'read_file', got %q", m.conversation.ToolHistory[0].Name)
	}
}

func TestHandleAgentToolStartAttemptCompletion(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	needsRefresh, _ := handleAgentToolStart(m, bridge.ToolStartEvent{
		Name:  "attempt_completion",
		Input: `{"result": "Task completed successfully"}`,
	})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	// Should append an assistant message with the result
	if len(m.conversation.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.conversation.Messages))
	}
	if m.conversation.Messages[0].Role != types.RoleAssistant {
		t.Errorf("expected assistant role, got %v", m.conversation.Messages[0].Role)
	}
	if !strings.Contains(m.conversation.Messages[0].Content, "Task completed successfully") {
		t.Errorf("expected result in content, got %q", m.conversation.Messages[0].Content)
	}
}

func TestHandleAgentToolStartAskFollowupQuestion(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	needsRefresh, _ := handleAgentToolStart(m, bridge.ToolStartEvent{
		Name:  "ask_followup_question",
		Input: `{"question": "What is your preference?"}`,
	})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	// Should NOT append any message (handled by AskQuestionEvent)
	if len(m.conversation.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(m.conversation.Messages))
	}
}

func TestHandleAgentToolStartPlanModeRespond(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	needsRefresh, _ := handleAgentToolStart(m, bridge.ToolStartEvent{
		Name:  "plan_mode_respond",
		Input: `{"response": "Here is my plan"}`,
	})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	// Should NOT append any message (handled by toolComplete)
	if len(m.conversation.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(m.conversation.Messages))
	}
}

func TestHandleAgentToolStartSystemMessage(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	needsRefresh, _ := handleAgentToolStart(m, bridge.ToolStartEvent{
		Name:  "write_to_file",
		Input: `{"path": "test.txt"}`,
	})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	// Should append a system message with tool display
	if len(m.conversation.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.conversation.Messages))
	}
	if m.conversation.Messages[0].Role != types.RoleSystem {
		t.Errorf("expected system role, got %v", m.conversation.Messages[0].Role)
	}
	if !strings.Contains(m.conversation.Messages[0].Content, "🔧") {
		t.Errorf("expected tool icon in content, got %q", m.conversation.Messages[0].Content)
	}
}

func TestHandleAgentToolCompleteReturnsNeedsRefresh(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// Set up a running tool
	m.conversation.AddToolStart("read_file")
	m.currentTool = "read_file"

	needsRefresh, cmds := handleAgentToolComplete(m, bridge.ToolCompleteEvent{
		Name:   "read_file",
		Result: "file content",
	})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	if len(cmds) != 0 {
		t.Errorf("expected no cmds, got %d", len(cmds))
	}
	if m.currentTool != "" {
		t.Errorf("expected currentTool cleared, got %q", m.currentTool)
	}
	// Should have appended a system message
	if len(m.conversation.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.conversation.Messages))
	}
	if !strings.Contains(m.conversation.Messages[0].Content, "Completed") {
		t.Errorf("expected 'Completed' in content, got %q", m.conversation.Messages[0].Content)
	}
}

func TestHandleAgentToolCompleteAttemptCompletion(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	needsRefresh, _ := handleAgentToolComplete(m, bridge.ToolCompleteEvent{
		Name:   "attempt_completion",
		Result: "done",
	})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	// Should NOT append any message (handled by toolStart)
	if len(m.conversation.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(m.conversation.Messages))
	}
}

func TestHandleAgentToolCompletePlanModeRespond(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	needsRefresh, _ := handleAgentToolComplete(m, bridge.ToolCompleteEvent{
		Name:   "plan_mode_respond",
		Result: "Here is my detailed plan...",
	})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	// Should append an assistant message with the full result
	if len(m.conversation.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.conversation.Messages))
	}
	if m.conversation.Messages[0].Role != types.RoleAssistant {
		t.Errorf("expected assistant role, got %v", m.conversation.Messages[0].Role)
	}
	if !strings.Contains(m.conversation.Messages[0].Content, "Here is my detailed plan") {
		t.Errorf("expected plan content, got %q", m.conversation.Messages[0].Content)
	}
}

func TestHandleAgentToolCompleteAskFollowupQuestion(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	needsRefresh, _ := handleAgentToolComplete(m, bridge.ToolCompleteEvent{
		Name:   "ask_followup_question",
		Result: "",
	})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	// Should NOT append any message (handled by AskQuestionEvent)
	if len(m.conversation.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(m.conversation.Messages))
	}
}

func TestHandleAgentErrorReturnsNeedsRefresh(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// Set up a running tool to be marked as failed
	m.conversation.AddToolStart("read_file")
	m.isProcessing = true
	m.isStreaming = true

	needsRefresh, cmds := handleAgentError(m, bridge.ErrorEvent{Err: errors.New("something went wrong")})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	if len(cmds) == 0 {
		t.Error("expected at least one cmd (textarea.Blink)")
	}
	if m.isProcessing {
		t.Error("expected isProcessing=false after error")
	}
	if m.isStreaming {
		t.Error("expected isStreaming=false after error")
	}
	if m.err == nil || m.err.Error() != "something went wrong" {
		t.Errorf("expected error 'something went wrong', got %v", m.err)
	}
	// Running tool should be marked as failed
	if len(m.conversation.ToolHistory) > 0 && m.conversation.ToolHistory[0].Status != "failed" {
		t.Errorf("expected tool status 'failed', got %q", m.conversation.ToolHistory[0].Status)
	}
	// Should have appended exactly one error message (no extra "🔧 Failed" system message)
	if len(m.conversation.Messages) != 1 {
		t.Errorf("expected exactly 1 error message, got %d", len(m.conversation.Messages))
	}
	if !strings.Contains(m.conversation.Messages[0].Content, "Error: something went wrong") {
		t.Errorf("expected error message content, got %q", m.conversation.Messages[0].Content)
	}
}

func TestHandleAgentErrorWithoutRunningTool(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// No running tools — should still work
	needsRefresh, _ := handleAgentError(m, bridge.ErrorEvent{Err: errors.New("test error")})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	if m.err == nil || m.err.Error() != "test error" {
		t.Errorf("expected error 'test error', got %v", m.err)
	}
}

func TestHandleAgentCompleteReturnsNeedsRefresh(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	m.isProcessing = true
	m.isStreaming = true
	m.currentTool = "read_file"

	needsRefresh, cmds := handleAgentComplete(m, bridge.CompleteEvent{})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	if len(cmds) == 0 {
		t.Error("expected at least one cmd (textarea.Blink)")
	}
	if m.isProcessing {
		t.Error("expected isProcessing=false after complete")
	}
	if m.isStreaming {
		t.Error("expected isStreaming=false after complete")
	}
	if m.currentTool != "" {
		t.Errorf("expected currentTool cleared, got %q", m.currentTool)
	}
}

func TestHandleAgentStreamStartReturnsNeedsRefresh(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	needsRefresh, cmds := handleAgentStreamStart(m, bridge.StreamStartEvent{})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	if len(cmds) != 0 {
		t.Errorf("expected no cmds, got %d", len(cmds))
	}
	if !m.isStreaming {
		t.Error("expected isStreaming=true")
	}
	if m.activeAssistantIndex < 0 {
		t.Error("expected activeAssistantIndex to be set")
	}
	// Should have created an empty assistant message
	if len(m.conversation.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.conversation.Messages))
	}
	if m.conversation.Messages[0].Role != types.RoleAssistant {
		t.Errorf("expected assistant role, got %v", m.conversation.Messages[0].Role)
	}
}

func TestHandleAgentStreamEndReturnsNeedsRefresh(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	m.isStreaming = true

	needsRefresh, cmds := handleAgentStreamEnd(m, bridge.StreamEndEvent{})

	if !needsRefresh {
		t.Error("expected needsRefresh=true")
	}
	if len(cmds) != 0 {
		t.Errorf("expected no cmds, got %d", len(cmds))
	}
	if m.isStreaming {
		t.Error("expected isStreaming=false after stream end")
	}
}

// ---------------------------------------------------------------------------
// Integration test: verify that Update() calls updateViewport() when
// needsRefresh is true, and the viewport content is updated.
// ---------------------------------------------------------------------------

func TestUpdateCallsUpdateViewportOnAgentEvent(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// Send a user message first (this calls updateViewport directly via sendMessage)
	m.sendMessage("hello")

	// Simulate a stream start event via Update
	updatedModel, _ := m.Update(bridge.StreamStartEvent{})
	updated := updatedModel.(*Model)

	// The viewport should have content (the user message + empty assistant)
	view := stripANSI(updated.View())
	if !strings.Contains(view, "hello") {
		t.Errorf("expected 'hello' in view, got: %q", view)
	}
	if !strings.Contains(view, "Assistant:") {
		t.Errorf("expected 'Assistant:' in view, got: %q", view)
	}
}

func TestUpdateCallsUpdateViewportOnAskQuestionEvent(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	replyCh := make(chan string, 1)
	updatedModel, _ := m.Update(bridge.AskQuestionEvent{
		Question: "What is your choice?",
		Options:  []string{"A", "B"},
		Reply:    replyCh,
	})
	updated := updatedModel.(*Model)

	view := stripANSI(updated.View())
	if !strings.Contains(view, "What is your choice?") {
		t.Errorf("expected question in view, got: %q", view)
	}
	if !strings.Contains(view, "1. A") {
		t.Errorf("expected option '1. A' in view, got: %q", view)
	}
	if !strings.Contains(view, "2. B") {
		t.Errorf("expected option '2. B' in view, got: %q", view)
	}
}

func TestUpdateDoesNotCallUpdateViewportOnTickMsg(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// tickMsg should NOT trigger updateViewport (needsRefresh stays false)
	updatedModel, _ := m.Update(tickMsg{})
	_ = updatedModel.(*Model)

	// No assertion needed — just verifying no panic or unexpected behavior
}

func TestHandleAgentUpdateReturnsNeedsRefreshForAllEventTypes(t *testing.T) {
	m := New(nil)
	m.width = 100
	m.height = 30

	// Test each event type returns needsRefresh=true
	tests := []struct {
		name  string
		event bridge.AgentEvent
	}{
		{"ContentEvent", bridge.ContentEvent{Delta: "test"}},
		{"ToolStartEvent", bridge.ToolStartEvent{Name: "read_file", Input: `{"path": "."}`}},
		{"ToolCompleteEvent", bridge.ToolCompleteEvent{Name: "read_file", Result: "ok"}},
		{"ErrorEvent", bridge.ErrorEvent{Err: errors.New("test")}},
		{"CompleteEvent", bridge.CompleteEvent{}},
		{"StreamStartEvent", bridge.StreamStartEvent{}},
		{"StreamEndEvent", bridge.StreamEndEvent{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh model for each test to avoid state pollution
			m2 := New(nil)
			m2.width = 100
			m2.height = 30

			needsRefresh, _ := handleAgentUpdate(m2, tt.event)
			if !needsRefresh {
				t.Errorf("expected needsRefresh=true for %s", tt.name)
			}
		})
	}
}
