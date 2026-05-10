package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
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
			// render with glamour
			r, _ := glamour.NewTermRenderer(glamour.WithWordWrap(wrapWidth))
			if r != nil {
				if out, err := r.Render(msg.Content); err == nil {
					rendered = out
				} else {
					rendered = msg.Content
				}
			} else {
				// fallback: raw content
				rendered = msg.Content
			}
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
		var tb strings.Builder
		for _, tc := range msg.ToolCalls {
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
		rendered = strings.TrimRight(rendered, "\n") + "\n" + lipgloss.NewStyle().Padding(0, 0).Render(tb.String())
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