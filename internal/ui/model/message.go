// Package model provides pure data structures for the TUI conversation domain.
// It has zero external UI dependencies (no Bubbletea, no lipgloss).
package model

import (
	"time"

	"github.com/liup215/gline/pkg/types"
)

// Message represents a single message in the conversation.
// It mirrors the old ui.Message but lives in a standalone domain package.
type Message struct {
	Role      types.Role
	Content   string
	ToolCalls []types.ToolCall
	Options   []string    // Options for ask_followup_question display (nil for non-question messages)
	Timestamp time.Time

	// Cached rendered markdown to avoid repeated glamour rendering.
	Rendered          string
	RenderedWrapWidth int
	RenderedSource    string // original Content used to produce Rendered
}

// ResetRenderCache clears the Glamour-rendered cache fields.
// Call this when the content changes and needs re-rendering.
func (m *Message) ResetRenderCache() {
	m.Rendered = ""
	m.RenderedWrapWidth = 0
	m.RenderedSource = ""
}
