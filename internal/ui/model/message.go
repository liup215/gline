// Package model provides pure data structures for the TUI conversation domain.
// It has zero external UI dependencies (no Bubbletea, no lipgloss).
package model

import (
	"time"

	"github.com/liup215/gline/pkg/types"
)

// Message represents a single message in the conversation.
// It mirrors the old ui.Message but lives in a standalone domain package.
// This struct contains only pure data fields with no UI dependencies.
type Message struct {
	Role      types.Role
	Content   string
	ToolCalls []types.ToolCall
	Options   []string           // Options for ask_followup_question display (nil for non-question messages)
	Strategy  types.RenderStrategy // How to render this message (plain, markdown, etc.)
	Timestamp time.Time
}
