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
	"github.com/liup215/gline/pkg/types"
)

// ---------------------------------------------------------------------------
// Styles (duplicated from ui package to avoid circular dependency)
// ---------------------------------------------------------------------------

var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AAFF")).
			Bold(true)

	systemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	toolRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFD700")).
				Bold(true)

	toolCompletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00AA00"))

	toolFailedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF4444"))

	streamingIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00")).
				Bold(true)

	questionIconStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFD700")).
				MarginLeft(1)

	questionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700")).
			MarginLeft(2)

	optionNumStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FFAA"))

	optionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3A3A5C")).
			Padding(0, 2).
			MarginLeft(4)

	optionHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666")).
				Italic(true).
				MarginLeft(4)

	toolAreaBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#555555"))
)

// ---------------------------------------------------------------------------
// ConversationViewModel
// ---------------------------------------------------------------------------

// ConversationViewModel derives rendered display state from a model.Conversation.
type ConversationViewModel struct {
	content           string
	toolAreaContent   string
	dirty             bool
	renderer          *glamour.TermRenderer
	rendererWrapWidth int
}

// NewConversationViewModel creates a new ViewModel ready for use.
func NewConversationViewModel() *ConversationViewModel {
	return &ConversationViewModel{
		dirty: true,
	}
}

// MarkDirty marks the ViewModel as needing a refresh.
func (vm *ConversationViewModel) MarkDirty() {
	vm.dirty = true
}

// IsDirty reports whether the ViewModel needs a refresh.
func (vm *ConversationViewModel) IsDirty() bool {
	return vm.dirty
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
// Call this when dirty is true and you need fresh rendered output.
func (vm *ConversationViewModel) Refresh(conv *model.Conversation, width int, toolAreaHeight int, isStreaming bool, activeAssistantIndex int) {
	var content strings.Builder
	msgs := conv.Messages

	for i := range msgs {
		msg := msgs[i]
		switch msg.Role {
		case types.RoleUser:
			vm.writeUserMessage(&content, msg)
		case types.RoleAssistant:
			vm.writeAssistantMessage(&content, msgs, i, width, isStreaming && i == activeAssistantIndex)
		case types.RoleSystem:
			vm.writeSystemMessage(&content, msg)
		}
	}

	vm.content = content.String()
	vm.toolAreaContent = vm.renderToolArea(conv, width, toolAreaHeight)
	vm.dirty = false
}

// ---------------------------------------------------------------------------
// Private rendering helpers
// ---------------------------------------------------------------------------

func (vm *ConversationViewModel) writeUserMessage(b *strings.Builder, msg model.Message) {
	b.WriteString(userStyle.Render("You: "))
	b.WriteString(msg.Content)
	b.WriteString("\n")
	b.WriteString(systemStyle.Render(msg.Timestamp.Format("15:04")))
	b.WriteString("\n\n")
}

func (vm *ConversationViewModel) writeAssistantMessage(b *strings.Builder, msgs []model.Message, idx int, width int, isActiveStreaming bool) {
	b.WriteString(vm.renderMessageHeader(msgs, idx))
	b.WriteString(vm.renderAssistantContent(msgs, idx, width, isActiveStreaming))
	b.WriteString("\n")
}

func (vm *ConversationViewModel) writeSystemMessage(b *strings.Builder, msg model.Message) {
	content := msg.Content
	if strings.HasPrefix(content, "Error:") || strings.HasPrefix(content, "✗") {
		b.WriteString(errorStyle.Render(content))
		b.WriteString("\n\n")
	} else if strings.HasPrefix(content, "❓") || len(msg.Options) > 0 {
		b.WriteString(questionIconStyle.Render("❓ "))
		b.WriteString(questionStyle.Render(strings.TrimPrefix(content, "❓ ")))
		b.WriteString("\n")
		if len(msg.Options) > 0 {
			for i, opt := range msg.Options {
				num := optionNumStyle.Render(fmt.Sprintf("%d.", i+1))
				b.WriteString(optionStyle.Render(fmt.Sprintf("%s %s", num, opt)))
				b.WriteString("\n")
			}
			b.WriteString(optionHintStyle.Render("Enter option number or type your answer"))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	} else if strings.HasPrefix(content, "🔧") {
		if strings.Contains(content, "Running") || strings.Contains(content, "running") || strings.Contains(content, "started") {
			b.WriteString(toolRunningStyle.Render(content))
		} else if strings.Contains(content, "Completed") || strings.Contains(content, "✓") || strings.Contains(content, "completed") {
			b.WriteString(toolCompletedStyle.Render(content))
		} else if strings.Contains(content, "Failed") || strings.Contains(content, "✗") || strings.Contains(content, "failed") {
			b.WriteString(toolFailedStyle.Render(content))
		} else {
			b.WriteString(systemStyle.Render(content))
		}
		b.WriteString("\n\n")
	}
	// Note: system messages that don't match any prefix are silently dropped,
	// preserving the original behavior.
}

func (vm *ConversationViewModel) renderMessageHeader(msgs []model.Message, i int) string {
	if i < 0 || i >= len(msgs) {
		return ""
	}
	msg := msgs[i]
	author := ""
	style := userStyle
	switch msg.Role {
	case types.RoleUser:
		author = "You"
		style = userStyle
	case types.RoleSystem:
		author = "System"
		style = systemStyle
	case types.RoleAssistant:
		author = "Assistant"
		style = assistantStyle
	}
	return fmt.Sprintf("%s %s\n", style.Render(author+":"), msg.Timestamp.Format("15:04"))
}

func (vm *ConversationViewModel) renderAssistantContent(msgs []model.Message, i int, wrapWidth int, isActiveStreaming bool) string {
	if i < 0 || i >= len(msgs) {
		return ""
	}
	msg := msgs[i]
	rendered := ""

	// Use cache when possible.
	if msg.Rendered != "" && msg.RenderedSource == msg.Content && msg.RenderedWrapWidth == wrapWidth {
		rendered = msg.Rendered
	} else {
		switch msg.Role {
		case types.RoleAssistant:
			rendered = vm.renderMarkdown(msg.Content, wrapWidth)
		default:
			rendered = msg.Content
		}

		// Cache rendered output on the message.
		msgs[i].Rendered = rendered
		msgs[i].RenderedWrapWidth = wrapWidth
		msgs[i].RenderedSource = msg.Content
	}

	// Streaming indicator for active assistant message.
	if isActiveStreaming && msg.Role == types.RoleAssistant {
		rendered = strings.TrimRight(rendered, "\n") + "\n" + streamingIndicatorStyle.Render("▌")
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
	history := conv.ToolHistory
	if len(history) == 0 {
		return toolAreaBorderStyle.Render(strings.Repeat("─", width))
	}

	maxEntries := toolAreaHeight
	if maxEntries < 1 {
		maxEntries = 1
	}
	start := 0
	if len(history) > maxEntries {
		start = len(history) - maxEntries
	}

	var lines []string
	for i := start; i < len(history); i++ {
		ts := history[i]
		switch ts.Status {
		case "running":
			lines = append(lines, toolRunningStyle.Render(fmt.Sprintf("  🔧 %s ⏳", ts.Name)))
		case "completed":
			lines = append(lines, toolCompletedStyle.Render(fmt.Sprintf("  🔧 %s ✓", ts.Name)))
		case "failed":
			lines = append(lines, toolFailedStyle.Render(fmt.Sprintf("  🔧 %s ✗", ts.Name)))
		default:
			lines = append(lines, systemStyle.Render(fmt.Sprintf("  🔧 %s", ts.Name)))
		}
	}

	border := toolAreaBorderStyle.Render(strings.Repeat("─", width))
	all := []string{border}
	all = append(all, lines...)
	return lipgloss.JoinVertical(lipgloss.Left, all...)
}
