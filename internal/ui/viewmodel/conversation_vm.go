// Package viewmodel derives rendered display state from model.Conversation.
// It owns the Glamour markdown renderer and produces the full viewport content string.
// It has no Bubbletea dependencies.
package viewmodel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/ui/model"
	"github.com/liup215/gline/internal/ui/view"
	"github.com/liup215/gline/pkg/types"
)

// ---------------------------------------------------------------------------
// ConversationViewModel
// ---------------------------------------------------------------------------

// cachedMessage holds the pre-rendered content and cache validation fields for a single message.
// This replaces the cache fields that were previously on model.Message.
type cachedMessage struct {
	content   string // original content used to produce rendered
	wrapWidth int    // width used for rendering
	rendered  string // the rendered output
}

// ConversationViewModel derives rendered display state from a model.Conversation.
type ConversationViewModel struct {
	content           string
	toolAreaContent   string
	dirty             bool
	dirtyMessages     map[int]bool       // per-message dirty flags
	messageCache      map[int]cachedMessage // pre-rendered message strings
	renderer          *glamour.TermRenderer
	rendererWrapWidth int
}

// NewConversationViewModel creates a new ViewModel ready for use.
func NewConversationViewModel() *ConversationViewModel {
	return &ConversationViewModel{
		dirty:         true,
		dirtyMessages: make(map[int]bool),
		messageCache:  make(map[int]cachedMessage),
	}
}

// MarkDirty marks the ViewModel as needing a full refresh.
func (vm *ConversationViewModel) MarkDirty() {
	vm.dirty = true
}

// MarkMessageDirty marks a specific message index as needing re-rendering.
// This is more efficient than MarkDirty() when only a single message changed.
func (vm *ConversationViewModel) MarkMessageDirty(idx int) {
	if idx < 0 {
		vm.dirty = true // negative index means unknown, fall back to full refresh
		return
	}
	vm.dirtyMessages[idx] = true
	vm.dirty = true
}

// IsDirty reports whether the ViewModel needs a refresh.
func (vm *ConversationViewModel) IsDirty() bool {
	return vm.dirty
}

// InvalidateCache clears the message cache.
// This should be called when the conversation is cleared to prevent memory growth.
func (vm *ConversationViewModel) InvalidateCache() {
	vm.messageCache = make(map[int]cachedMessage)
	vm.dirtyMessages = make(map[int]bool)
	vm.dirty = true
}

// Content returns the full rendered viewport content string.
func (vm *ConversationViewModel) Content() string {
	return vm.content
}

// ToolAreaContent returns the rendered tool status area content.
func (vm *ConversationViewModel) ToolAreaContent() string {
	return vm.toolAreaContent
}

// Refresh rebuilds content and toolAreaContent from the conversation.
// When only specific messages are dirty (via MarkMessageDirty), it performs
// incremental rendering: only re-renders changed messages and re-joins the
// cached results. Falls back to full rebuild when dirtyMessages is empty
// (MarkDirty was called) or when message count changed.
func (vm *ConversationViewModel) Refresh(conv *model.Conversation, width int, toolAreaHeight int, isStreaming bool, activeAssistantIndex int) {
	msgs := conv.Messages

	// Determine if we can do incremental refresh.
	// Full rebuild is needed when:
	//   1. No per-message dirty flags set (MarkDirty was used)
	//   2. Message count changed (messages were added/removed)
	//   3. Cache is empty (first call)
	doFullRebuild := len(vm.dirtyMessages) == 0 || len(msgs) != len(vm.messageCache) || len(vm.messageCache) == 0

	if doFullRebuild {
		var content strings.Builder
		for i := range msgs {
			rendered := vm.renderMessage(msgs, i, width, isStreaming && i == activeAssistantIndex)
			content.WriteString(rendered)
			// Cache with content and wrapWidth for proper cache validation
			vm.messageCache[i] = cachedMessage{
				content:   msgs[i].Content,
				wrapWidth: width,
				rendered:  rendered,
			}
		}
		vm.content = content.String()
	} else {
		// Incremental: only re-render dirty messages, reuse cache for others.
		var content strings.Builder
		for i := range msgs {
			if vm.dirtyMessages[i] {
				rendered := vm.renderMessage(msgs, i, width, isStreaming && i == activeAssistantIndex)
				content.WriteString(rendered)
				// Cache with content and wrapWidth for proper cache validation
				vm.messageCache[i] = cachedMessage{
					content:   msgs[i].Content,
					wrapWidth: width,
					rendered:  rendered,
				}
			} else {
				content.WriteString(vm.messageCache[i].rendered)
			}
		}
		vm.content = content.String()
	}

	vm.toolAreaContent = vm.renderToolArea(conv, width, toolAreaHeight)
	vm.dirty = false
	vm.dirtyMessages = make(map[int]bool) // reset dirty flags
}

// renderMessage renders a single message at index i into its full string representation.
func (vm *ConversationViewModel) renderMessage(msgs []model.Message, i int, width int, isActiveStreaming bool) string {
	if i < 0 || i >= len(msgs) {
		return ""
	}
	msg := msgs[i]
	switch msg.Role {
	case types.RoleUser:
		return vm.renderUserMessage(msg, width)
	case types.RoleAssistant:
		return vm.renderAssistantMessage(msgs, i, width, isActiveStreaming)
	case types.RoleSystem:
		return vm.renderSystemMessage(msg)
	default:
		return ""
	}
}

// ---------------------------------------------------------------------------
// Private rendering helpers — single-message renderers
// ---------------------------------------------------------------------------

// renderUserMessage renders a single user message to its full string.
func (vm *ConversationViewModel) renderUserMessage(msg model.Message, width int) string {
	var b strings.Builder
	b.WriteString(view.UserStyle.Render("You: "))
	// Apply markdown rendering to user message content for consistent formatting
	b.WriteString(vm.renderMarkdown(msg.Content, width))
	b.WriteString("\n")
	b.WriteString(view.SystemStyle.Render(msg.Timestamp.Format("15:04")))
	b.WriteString("\n\n")
	return b.String()
}

// renderAssistantMessage renders a single assistant message to its full string.
func (vm *ConversationViewModel) renderAssistantMessage(msgs []model.Message, idx int, width int, isActiveStreaming bool) string {
	var b strings.Builder
	b.WriteString(vm.renderMessageHeader(msgs, idx))
	b.WriteString(vm.renderAssistantContent(msgs, idx, width, isActiveStreaming))
	b.WriteString("\n")
	return b.String()
}

// renderSystemMessage renders a single system message to its full string.
func (vm *ConversationViewModel) renderSystemMessage(msg model.Message) string {
	content := msg.Content
	var b strings.Builder

	// Handle based on Strategy field
	switch msg.Strategy {
	case types.StrategyMarkdown:
		// System message with markdown rendering
		b.WriteString(vm.renderMarkdown(content, 80))
		b.WriteString("\n\n")
		return b.String()
	case types.StrategyJSON:
		// JSON code block
		b.WriteString(view.SystemStyle.Render(content))
		b.WriteString("\n\n")
		return b.String()
	case types.StrategySpecial:
		// Special rendering handled elsewhere
		return ""
	case types.StrategySkip:
		// Skip this message
		return ""
	}

	// Fallback: legacy hardcoded detection
	if strings.HasPrefix(content, "Error:") || strings.HasPrefix(content, "✗") {
		b.WriteString(view.ErrorStyle.Render(content))
		b.WriteString("\n\n")
	} else if strings.HasPrefix(content, "❓") || len(msg.Options) > 0 {
		b.WriteString(view.QuestionIconStyle.Render("❓ "))
		b.WriteString(view.QuestionStyle.Render(strings.TrimPrefix(content, "❓ ")))
		b.WriteString("\n")
		if len(msg.Options) > 0 {
			for i, opt := range msg.Options {
				num := view.OptionNumStyle.Render(fmt.Sprintf("%d.", i+1))
				b.WriteString(view.OptionStyle.Render(fmt.Sprintf("%s %s", num, opt)))
				b.WriteString("\n")
			}
			b.WriteString(view.OptionHintStyle.Render("Enter option number or type your answer"))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	} else if strings.HasPrefix(content, "🔧") {
		if strings.Contains(content, "Running") || strings.Contains(content, "running") || strings.Contains(content, "started") {
			b.WriteString(view.ToolRunningStyle.Render(content))
		} else if strings.Contains(content, "Completed") || strings.Contains(content, "✓") || strings.Contains(content, "completed") {
			b.WriteString(view.ToolCompletedStyle.Render(content))
		} else if strings.Contains(content, "Failed") || strings.Contains(content, "✗") || strings.Contains(content, "failed") {
			b.WriteString(view.ToolFailedStyle.Render(content))
		} else {
			b.WriteString(view.SystemStyle.Render(content))
		}
		b.WriteString("\n\n")
	} else {
		// Default: display unknown system messages with SystemStyle instead of silently dropping them
		b.WriteString(view.SystemStyle.Render(content))
		b.WriteString("\n\n")
	}
	return b.String()
}

func (vm *ConversationViewModel) renderMessageHeader(msgs []model.Message, i int) string {
	if i < 0 || i >= len(msgs) {
		return ""
	}
	msg := msgs[i]
	author := ""
	style := view.UserStyle
	switch msg.Role {
	case types.RoleUser:
		author = "You"
		style = view.UserStyle
	case types.RoleSystem:
		author = "System"
		style = view.SystemStyle
	case types.RoleAssistant:
		author = "Assistant"
		style = view.AssistantStyle
	}
	return fmt.Sprintf("%s %s\n", style.Render(author+":"), msg.Timestamp.Format("15:04"))
}

func (vm *ConversationViewModel) renderAssistantContent(msgs []model.Message, i int, wrapWidth int, isActiveStreaming bool) string {
	if i < 0 || i >= len(msgs) {
		return ""
	}
	msg := msgs[i]
	rendered := ""

	// Use ViewModel's messageCache when possible.
	// Check if we have a cached entry for this message index with matching content and width.
	if cache, ok := vm.messageCache[i]; ok && cache.content == msg.Content && cache.wrapWidth == wrapWidth {
		rendered = cache.rendered
	} else {
		switch msg.Role {
		case types.RoleAssistant:
			rendered = vm.renderMarkdown(msg.Content, wrapWidth)
		default:
			rendered = msg.Content
		}

		// Cache rendered output in ViewModel's messageCache.
		// Note: This cache is updated in Refresh(), but we also update here
		// for consistency when renderAssistantContent is called directly.
		vm.messageCache[i] = cachedMessage{
			content:   msg.Content,
			wrapWidth: wrapWidth,
			rendered:  rendered,
		}
	}

	// Streaming indicator for active assistant message.
	if isActiveStreaming && msg.Role == types.RoleAssistant {
		rendered = strings.TrimRight(rendered, "\n") + "\n" + view.StreamingIndicatorStyle.Render("▌")
	}

	// If tool calls are attached to the message, pretty-print them after content.
	if len(msg.ToolCalls) > 0 {
		rendered = strings.TrimRight(rendered, "\n") + "\n" + vm.formatToolCallsInline(msg.ToolCalls)
	}

	// Ensure trailing newline.
	if !strings.HasSuffix(rendered, "\n") {
		rendered += "\n"
	}
	return rendered
}

func (vm *ConversationViewModel) renderMarkdown(content string, wrapWidth int) string {
	if content == "" {
		return ""
	}

	if wrapWidth < 20 {
		wrapWidth = 20
	}

	var r *glamour.TermRenderer
	var err error
	if vm.renderer != nil && vm.rendererWrapWidth == wrapWidth {
		r = vm.renderer
	} else {
		if r, err = glamour.NewTermRenderer(glamour.WithWordWrap(wrapWidth)); err == nil {
			vm.renderer = r
			vm.rendererWrapWidth = wrapWidth
		} else {
			r = nil
		}
	}

	if r != nil {
		if out, err := r.Render(content); err == nil {
			return out
		}
	}

	if out, err := glamour.Render(content, "dark"); err == nil {
		return out
	}

	return content
}

func (vm *ConversationViewModel) formatToolCallsInline(calls []types.ToolCall) string {
	var tb strings.Builder
	for _, tc := range calls {
		tb.WriteString(fmt.Sprintf("\n  🔧 %s", tc.Name))
		if len(tc.Input) > 0 {
			var buf bytes.Buffer
			if err := json.Indent(&buf, tc.Input, "    ", "  "); err == nil {
				tb.WriteString("\n    Input:\n")
				tb.WriteString(buf.String())
			} else {
				tb.WriteString("\n    Input: ")
				tb.WriteString(string(tc.Input))
			}
		}
	}
	return lipgloss.NewStyle().Padding(0, 0).Render(tb.String())
}

func (vm *ConversationViewModel) renderToolArea(conv *model.Conversation, width int, toolAreaHeight int) string {
	return view.RenderToolAreaContent(conv.ToolHistory, width, toolAreaHeight)
}