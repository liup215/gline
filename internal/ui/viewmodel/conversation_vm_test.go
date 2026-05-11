package viewmodel

import (
	"strings"
	"testing"
	"time"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/ui/model"
	"github.com/liup215/gline/pkg/types"
)

func TestNewConversationViewModel(t *testing.T) {
	vm := NewConversationViewModel()
	if vm == nil {
		t.Fatal("expected non-nil ViewModel")
	}
	if !vm.IsDirty() {
		t.Error("expected fresh ViewModel to be dirty")
	}
	if vm.Content() != "" {
		t.Errorf("expected empty content, got %q", vm.Content())
	}
	if vm.ToolAreaContent() != "" {
		t.Errorf("expected empty tool area, got %q", vm.ToolAreaContent())
	}
}

func TestRefreshEmptyConversation(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	vm.Refresh(conv, 80, 3, false, -1)

	if vm.IsDirty() {
		t.Error("expected ViewModel to be clean after Refresh")
	}
	if vm.Content() != "" {
		t.Errorf("expected empty content for empty conversation, got %q", vm.Content())
	}
	// Tool area should show border even when empty
	if vm.ToolAreaContent() == "" {
		t.Error("expected tool area border for empty history")
	}
}

func TestRefreshUserMessage(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	conv.AppendMessage(model.Message{
		Role:      types.RoleUser,
		Content:   "hello",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	vm.Refresh(conv, 80, 3, false, -1)

	content := vm.Content()
	if !strings.Contains(content, "You: ") {
		t.Errorf("expected user header, got: %q", content)
	}
	if !strings.Contains(content, "hello") {
		t.Errorf("expected message content, got: %q", content)
	}
	if !strings.Contains(content, "12:00") {
		t.Errorf("expected timestamp, got: %q", content)
	}
}

func TestRefreshAssistantMessage(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	conv.AppendMessage(model.Message{
		Role:      types.RoleAssistant,
		Content:   "Hi there",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	vm.Refresh(conv, 80, 3, false, -1)

	content := vm.Content()
	if !strings.Contains(content, "Assistant: ") {
		t.Errorf("expected assistant header, got: %q", content)
	}
	if !strings.Contains(content, "Hi there") {
		t.Errorf("expected message content, got: %q", content)
	}
}

func TestRefreshSystemErrorMessage(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	conv.AppendMessage(model.Message{
		Role:      types.RoleSystem,
		Content:   "Error: something went wrong",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	vm.Refresh(conv, 80, 3, false, -1)

	content := vm.Content()
	if !strings.Contains(content, "Error: something went wrong") {
		t.Errorf("expected error content, got: %q", content)
	}
}

func TestRefreshSystemToolMessage(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	conv.AppendMessage(model.Message{
		Role:      types.RoleSystem,
		Content:   "🔧 Running: read_file",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	vm.Refresh(conv, 80, 3, false, -1)

	content := vm.Content()
	if !strings.Contains(content, "🔧 Running: read_file") {
		t.Errorf("expected tool content, got: %q", content)
	}
}

func TestRefreshQuestionMessage(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	conv.AppendMessage(model.Message{
		Role:      types.RoleSystem,
		Content:   "❓ What is your preference?",
		Options:   []string{"A", "B", "C"},
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	vm.Refresh(conv, 80, 3, false, -1)

	content := vm.Content()
	if !strings.Contains(content, "What is your preference?") {
		t.Errorf("expected question content, got: %q", content)
	}
	for _, opt := range []string{"1. A", "2. B", "3. C"} {
		if !strings.Contains(content, opt) {
			t.Errorf("expected option %q, got: %q", opt, content)
		}
	}
}

func TestRefreshStreamingIndicator(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	idx := conv.AppendMessage(model.Message{
		Role:      types.RoleAssistant,
		Content:   "typing",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	vm.Refresh(conv, 80, 3, true, idx)

	content := vm.Content()
	if !strings.Contains(content, "▌") {
		t.Errorf("expected streaming indicator, got: %q", content)
	}
}

func TestRefreshNoStreamingIndicatorWhenNotActive(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	conv.AppendMessage(model.Message{
		Role:      types.RoleAssistant,
		Content:   "done",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	// isStreaming=true but activeAssistantIndex=-1 means no active stream
	vm.Refresh(conv, 80, 3, true, -1)

	content := vm.Content()
	if strings.Contains(content, "▌") {
		t.Errorf("did not expect streaming indicator, got: %q", content)
	}
}

func TestRefreshToolAreaWithHistory(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	conv.AddToolStart("read_file")
	vm.Refresh(conv, 80, 3, false, -1)

	toolArea := vm.ToolAreaContent()
	if !strings.Contains(toolArea, "read_file") {
		t.Errorf("expected tool name in area, got: %q", toolArea)
	}
	if !strings.Contains(toolArea, "⏳") {
		t.Errorf("expected running indicator, got: %q", toolArea)
	}
}

func TestRefreshToolAreaCompleted(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	conv.AddToolStart("read_file")
	conv.MarkToolComplete("read_file")
	vm.Refresh(conv, 80, 3, false, -1)

	toolArea := vm.ToolAreaContent()
	if !strings.Contains(toolArea, "✓") {
		t.Errorf("expected completed indicator, got: %q", toolArea)
	}
}

func TestRefreshToolAreaFailed(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	conv.AddToolStart("read_file")
	conv.MarkToolFailed("read_file")
	vm.Refresh(conv, 80, 3, false, -1)

	toolArea := vm.ToolAreaContent()
	if !strings.Contains(toolArea, "✗") {
		t.Errorf("expected failed indicator, got: %q", toolArea)
	}
}

func TestRefreshToolAreaRespectsMaxEntries(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	for i := 0; i < 10; i++ {
		conv.AddToolStart("tool_" + string(rune('A'+i)))
	}
	vm.Refresh(conv, 80, 3, false, -1)

	toolArea := vm.ToolAreaContent()
	// Only the last 3 tools should be visible (toolAreaHeight=3)
	if strings.Contains(toolArea, "tool_A") {
		t.Error("did not expect tool_A to be visible (should be truncated)")
	}
	if !strings.Contains(toolArea, "tool_J") {
		t.Errorf("expected tool_J to be visible, got: %q", toolArea)
	}
}

func TestMarkDirtyAndIsDirty(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()

	vm.Refresh(conv, 80, 3, false, -1)
	if vm.IsDirty() {
		t.Error("expected clean after Refresh")
	}

	vm.MarkDirty()
	if !vm.IsDirty() {
		t.Error("expected dirty after MarkDirty")
	}
}

func TestSystemMessageWithoutPrefixIsDropped(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	conv.AppendMessage(model.Message{
		Role:      types.RoleSystem,
		Content:   "random system text without prefix",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	vm.Refresh(conv, 80, 3, false, -1)

	content := vm.Content()
	if strings.Contains(content, "random system text") {
		t.Errorf("expected unmatched system message to be dropped, got: %q", content)
	}
}

func TestConversationModeAndProvider(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()
	conv.Mode = agent.ModePlan
	conv.Provider = "openai"
	conv.ModelName = "gpt-4"
	conv.AppendMessage(model.Message{
		Role:      types.RoleUser,
		Content:   "test",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	vm.Refresh(conv, 80, 3, false, -1)

	// ViewModel content should not include mode/provider (those are status-bar only)
	content := vm.Content()
	if strings.Contains(content, "openai") {
		t.Error("ViewModel content should not include provider name")
	}
	if !strings.Contains(content, "test") {
		t.Errorf("expected user message content, got: %q", content)
	}
}
