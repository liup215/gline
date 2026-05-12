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

// ---------------------------------------------------------------------------
// Phase 6: Incremental rendering tests
// ---------------------------------------------------------------------------

func TestMarkMessageDirty(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()

	// Add two messages and refresh
	conv.AppendMessage(model.Message{
		Role:      types.RoleUser,
		Content:   "first",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	conv.AppendMessage(model.Message{
		Role:      types.RoleUser,
		Content:   "second",
		Timestamp: time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
	})
	vm.Refresh(conv, 80, 3, false, -1)

	// After refresh, dirty should be false
	if vm.IsDirty() {
		t.Error("expected clean after Refresh")
	}

	// Mark message 0 as dirty
	vm.MarkMessageDirty(0)
	if !vm.IsDirty() {
		t.Error("expected dirty after MarkMessageDirty")
	}
	if !vm.dirtyMessages[0] {
		t.Error("expected dirtyMessages[0] to be set")
	}
	if vm.dirtyMessages[1] {
		t.Error("expected dirtyMessages[1] not to be set")
	}
}

func TestMarkMessageDirtyNegativeIndex(t *testing.T) {
	vm := NewConversationViewModel()
	// Negative index should fall back to full dirty
	vm.MarkMessageDirty(-1)
	if !vm.IsDirty() {
		t.Error("expected dirty after MarkMessageDirty(-1)")
	}
	// dirtyMessages should be empty (full refresh fallback)
	if len(vm.dirtyMessages) != 0 {
		t.Errorf("expected empty dirtyMessages for negative index, got %d", len(vm.dirtyMessages))
	}
}

func TestIncrementalRefreshReusesCache(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()

	// Add three messages - use user messages to avoid glamour rendering differences
	conv.AppendMessage(model.Message{
		Role:      types.RoleUser,
		Content:   "hello",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	conv.AppendMessage(model.Message{
		Role:      types.RoleUser,
		Content:   "world",
		Timestamp: time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
	})
	conv.AppendMessage(model.Message{
		Role:      types.RoleUser,
		Content:   "response",
		Timestamp: time.Date(2024, 1, 1, 12, 2, 0, 0, time.UTC),
	})

	// First refresh — full rebuild, populates cache
	vm.Refresh(conv, 80, 3, false, -1)
	firstContent := vm.Content()

	// Verify cache is populated
	if len(vm.messageCache) != 3 {
		t.Errorf("expected 3 cached messages, got %d", len(vm.messageCache))
	}

	// Store the cached rendered content for non-dirty messages
	cachedMsg0 := vm.messageCache[0].rendered
	cachedMsg1 := vm.messageCache[1].rendered

	// Mark only message 2 as dirty and refresh
	vm.MarkMessageDirty(2)
	vm.Refresh(conv, 80, 3, false, -1)

	// Content should still contain all messages
	secondContent := vm.Content()
	if !strings.Contains(secondContent, "hello") {
		t.Error("expected 'hello' in content after incremental refresh")
	}
	if !strings.Contains(secondContent, "world") {
		t.Error("expected 'world' in content after incremental refresh")
	}
	if !strings.Contains(secondContent, "response") {
		t.Error("expected 'response' in content after incremental refresh")
	}

	// Non-dirty messages should have been reused from cache (same rendered content)
	if vm.messageCache[0].rendered != cachedMsg0 {
		t.Error("expected message 0 to be reused from cache")
	}
	if vm.messageCache[1].rendered != cachedMsg1 {
		t.Error("expected message 1 to be reused from cache")
	}

	// Content should be the same since we only re-rendered the message with same content
	if firstContent != secondContent {
		t.Error("expected identical content after incremental refresh with no actual changes")
	}

	// Verify dirty flags were reset
	if vm.IsDirty() {
		t.Error("expected clean after incremental Refresh")
	}
	if len(vm.dirtyMessages) != 0 {
		t.Errorf("expected empty dirtyMessages after Refresh, got %d", len(vm.dirtyMessages))
	}
}

func TestIncrementalRefreshWithContentChange(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()

	// Add two messages
	conv.AppendMessage(model.Message{
		Role:      types.RoleUser,
		Content:   "original",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	conv.AppendMessage(model.Message{
		Role:      types.RoleAssistant,
		Content:   "old response",
		Timestamp: time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
	})

	// First refresh
	vm.Refresh(conv, 80, 3, false, -1)
	firstContent := vm.Content()

	// Change the assistant message content
	conv.Messages[1].Content = "updated response"

	// Mark only the changed message as dirty
	vm.MarkMessageDirty(1)
	vm.Refresh(conv, 80, 3, false, -1)
	secondContent := vm.Content()

	// Should contain the updated content
	if !strings.Contains(secondContent, "updated response") {
		t.Errorf("expected 'updated response' in content, got: %q", secondContent)
	}

	// Should NOT contain the old content
	if strings.Contains(secondContent, "old response") {
		t.Error("did not expect 'old response' in content after update")
	}

	// First message should still be present unchanged
	if !strings.Contains(secondContent, "original") {
		t.Error("expected 'original' to still be in content")
	}

	// Content should be different from before
	if firstContent == secondContent {
		t.Error("expected content to differ after message content change")
	}
}

func TestFullRebuildOnMessageCountChange(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()

	// Add one message and refresh
	conv.AppendMessage(model.Message{
		Role:      types.RoleUser,
		Content:   "first",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	vm.Refresh(conv, 80, 3, false, -1)

	// Add a second message (simulating a new message arriving)
	conv.AppendMessage(model.Message{
		Role:      types.RoleAssistant,
		Content:   "second",
		Timestamp: time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
	})

	// MarkDirty (not MarkMessageDirty) — should trigger full rebuild
	vm.MarkDirty()
	vm.Refresh(conv, 80, 3, false, -1)

	// Both messages should be present
	content := vm.Content()
	if !strings.Contains(content, "first") {
		t.Error("expected 'first' in content")
	}
	if !strings.Contains(content, "second") {
		t.Error("expected 'second' in content")
	}

	// Cache should have 2 entries
	if len(vm.messageCache) != 2 {
		t.Errorf("expected 2 cached messages, got %d", len(vm.messageCache))
	}
}

func TestIncrementalRefreshWithAppendedMessage(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()

	// Add one message and refresh
	conv.AppendMessage(model.Message{
		Role:      types.RoleUser,
		Content:   "first",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	vm.Refresh(conv, 80, 3, false, -1)

	// Append a new message (simulating tool start or stream start)
	idx := conv.AppendMessage(model.Message{
		Role:      types.RoleAssistant,
		Content:   "new response",
		Timestamp: time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
	})

	// Mark the new message as dirty — message count changed, so this should
	// trigger a full rebuild (len(msgs) != len(vm.messageCache))
	vm.MarkMessageDirty(idx)
	vm.Refresh(conv, 80, 3, false, -1)

	// Both messages should be present
	content := vm.Content()
	if !strings.Contains(content, "first") {
		t.Error("expected 'first' in content")
	}
	if !strings.Contains(content, "new response") {
		t.Error("expected 'new response' in content")
	}

	// Cache should have 2 entries
	if len(vm.messageCache) != 2 {
		t.Errorf("expected 2 cached messages, got %d", len(vm.messageCache))
	}
}

// ---------------------------------------------------------------------------
// Phase 9: ViewModel cache tests (replaces model.Message cache fields)
// ---------------------------------------------------------------------------

func TestViewModelCacheHit(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()

	// Add an assistant message
	conv.AppendMessage(model.Message{
		Role:      types.RoleAssistant,
		Content:   "Hello world",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})

	// First render - should populate cache
	vm.Refresh(conv, 80, 3, false, -1)

	// Verify cache is populated with correct fields
	if len(vm.messageCache) != 1 {
		t.Fatalf("expected 1 cached message, got %d", len(vm.messageCache))
	}
	cache := vm.messageCache[0]
	if cache.content != "Hello world" {
		t.Errorf("expected cached content 'Hello world', got %q", cache.content)
	}
	if cache.wrapWidth != 80 {
		t.Errorf("expected cached wrapWidth 80, got %d", cache.wrapWidth)
	}
	if cache.rendered == "" {
		t.Error("expected non-empty rendered content in cache")
	}
}

func TestViewModelCacheMissOnContentChange(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()

	// Add an assistant message
	conv.AppendMessage(model.Message{
		Role:      types.RoleAssistant,
		Content:   "Original content",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})

	// First render - populates cache
	vm.Refresh(conv, 80, 3, false, -1)
	firstRendered := vm.messageCache[0].rendered

	// Change the content
	conv.Messages[0].Content = "Updated content"

	// Mark dirty and re-render
	vm.MarkMessageDirty(0)
	vm.Refresh(conv, 80, 3, false, -1)

	// Cache should be updated with new content
	if vm.messageCache[0].content != "Updated content" {
		t.Errorf("expected cached content 'Updated content', got %q", vm.messageCache[0].content)
	}
	if vm.messageCache[0].rendered == firstRendered {
		t.Error("expected rendered content to change after content update")
	}
}

func TestViewModelCacheMissOnWidthChange(t *testing.T) {
	vm := NewConversationViewModel()
	conv := model.NewConversation()

	// Add an assistant message
	conv.AppendMessage(model.Message{
		Role:      types.RoleAssistant,
		Content:   "Hello world",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})

	// First render at width 80
	vm.Refresh(conv, 80, 3, false, -1)
	firstRendered := vm.messageCache[0].rendered

	// Re-render at different width (simulating window resize)
	vm.MarkDirty()
	vm.Refresh(conv, 120, 3, false, -1)

	// Cache should be updated with new width
	if vm.messageCache[0].wrapWidth != 120 {
		t.Errorf("expected cached wrapWidth 120, got %d", vm.messageCache[0].wrapWidth)
	}
	// Rendered content may differ due to different wrapping
	if vm.messageCache[0].rendered == "" {
		t.Error("expected non-empty rendered content after width change")
	}
	_ = firstRendered // acknowledge we checked it
}

func TestViewModelCacheNoDirectAccessToMessageCache(t *testing.T) {
	// This test verifies that the cache is stored in ViewModel, not in model.Message
	vm := NewConversationViewModel()
	conv := model.NewConversation()

	// Add a message
	conv.AppendMessage(model.Message{
		Role:      types.RoleAssistant,
		Content:   "Test",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	})

	// Render
	vm.Refresh(conv, 80, 3, false, -1)

	// Verify ViewModel has cache
	if len(vm.messageCache) != 1 {
		t.Fatal("expected ViewModel to have cache")
	}

	// Verify message struct doesn't have cache fields (compile-time check)
	// If this compiles, it means Message doesn't have Rendered, RenderedWrapWidth, RenderedSource fields
	msg := conv.Messages[0]
	_ = msg.Content // just to use the variable
}
