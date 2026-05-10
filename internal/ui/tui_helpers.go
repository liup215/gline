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

// renderMarkdown renders markdown content using a cached Glamour renderer on the model.
// It falls back to glamour.Render and finally to raw content on error.
func renderMarkdown(m *Model, content string, wrapWidth int) string {
	if content == "" {
		return ""
	}

	// Ensure minimum wrap width
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	// Reuse renderer cached on the model when possible
	var r *glamour.TermRenderer
	var err error
	if m != nil && m.renderer != nil && m.rendererWrapWidth == wrapWidth {
		r = m.renderer
	} else {
		if r, err = glamour.NewTermRenderer(glamour.WithWordWrap(wrapWidth)); err == nil {
			if m != nil {
				m.renderer = r
				m.rendererWrapWidth = wrapWidth
			}
		} else {
			// failed to create renderer
			r = nil
		}
	}

	// Try rendering with term renderer first
	if r != nil {
		if out, err := r.Render(content); err == nil {
			return out
		}
	}

	// Fallback to glamour.Render default
	if out, err := glamour.Render(content, "dark"); err == nil {
		return out
	}

	// Final fallback: raw content
	return content
}

// formatToolCallsInline pretty-prints tool calls inline after an assistant message.
// Returns a styled string (ready to append to rendered markdown).
func formatToolCallsInline(calls []types.ToolCall) string {
	var tb strings.Builder
	for _, tc := range calls {
		tb.WriteString(fmt.Sprintf("\n  🔧 %s", tc.Name))
		// include input if present (pretty-print JSON when possible)
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
	// Render with minimal padding to keep visual parity with previous behavior.
	return lipgloss.NewStyle().Padding(0, 0).Render(tb.String())
}