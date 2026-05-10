package ui

import (
"fmt"
"strings"

"github.com/charmbracelet/lipgloss"

"github.com/liup215/gline/pkg/types"
)

// renderMessageHeader returns the formatted header line (author + timestamp) for message at index i.
func (m *Model) renderMessageHeader(i int) string {
	if i < 0 || i >= len(m.messages) {
		return ""
	}
	msg := m.messages[i]
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

// renderAssistantContent renders and returns the body for message at index i.
// It handles caching of Glamour-rendered markdown per message and width, streaming indicator,
// and pretty-printing JSON for tool outputs.
func renderAssistantContent(m *Model, i int) string {
	if i < 0 || i >= len(m.messages) {
		return ""
	}
	msg := m.messages[i]
	wrapWidth := m.viewport.Width
	rendered := ""

 
// use cache when possible
if msg.Rendered != "" && msg.RenderedSource == msg.Content && msg.RenderedWrapWidth == wrapWidth {
	rendered = msg.Rendered
} else {
	switch msg.Role {
	case types.RoleAssistant:
		// delegate markdown rendering to centralized helper
		rendered = renderMarkdown(m, msg.Content, wrapWidth)
	default:
		rendered = msg.Content
	}

	// cache rendered output
	msg.Rendered = rendered
	msg.RenderedWrapWidth = wrapWidth
	msg.RenderedSource = msg.Content
	m.messages[i] = msg
}

	// streaming indicator for active assistant message
	if m.isStreaming && i == m.activeAssistantIndex && msg.Role == types.RoleAssistant {
		rendered = strings.TrimRight(rendered, "\n") + "\n" + streamingIndicatorStyle.Render("▌")
	}

 
// If tool calls are attached to the message, pretty-print them after content
if len(msg.ToolCalls) > 0 {
	rendered = strings.TrimRight(rendered, "\n") + "\n" + formatToolCallsInline(msg.ToolCalls)
}

	// ensure trailing newline
	if !strings.HasSuffix(rendered, "\n") {
		rendered += "\n"
	}
	return rendered
}

// renderToolCalls returns the rendered tool area (similar to previous renderToolArea but callable).
func renderToolCalls(m *Model) string {
	// If no history, return bordered empty line like original behaviour
	if len(m.toolHistory) == 0 {
		return toolAreaBorderStyle.Render(strings.Repeat("─", m.width))
	}

	// Determine how many entries to show (limit by toolAreaHeight)
	maxEntries := m.toolAreaHeight
	if maxEntries < 1 {
		maxEntries = 1
	}
	start := 0
	if len(m.toolHistory) > maxEntries {
		start = len(m.toolHistory) - maxEntries
	}

	var lines []string
	for i := start; i < len(m.toolHistory); i++ {
		ts := m.toolHistory[i]
		switch ts.Status {
		case "running":
			lines = append(lines, toolRunningStyle.Render(fmt.Sprintf("  🔧 %s ⏳", ts.Name)))
		case "completed":
			lines = append(lines, toolCompletedStyle.Render(fmt.Sprintf("  🔧 %s ✓", ts.Name)))
		case "failed":
			lines = append(lines, toolFailedStyle.Render(fmt.Sprintf("  🔧 %s ✗", ts.Name)))
		default:
			lines = append(lines, toolStyle.Render(fmt.Sprintf("  🔧 %s", ts.Name)))
		}
	}

	// Top border
	border := toolAreaBorderStyle.Render(strings.Repeat("─", m.width))

	// Combine border and tool lines
	all := []string{border}
	all = append(all, lines...)
	return lipgloss.JoinVertical(lipgloss.Left, all...)
}