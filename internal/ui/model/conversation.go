// Package model provides pure data structures for the TUI conversation domain.
package model

import (
	"time"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/pkg/types"
)

// ToolStatus represents the status of a tool call in the UI.
type ToolStatus struct {
	Name      string
	Status    string // "running", "completed", "failed"
	StartTime time.Time
}

// Conversation holds all business data for a single TUI session.
// It has zero Bubbletea/lipgloss dependencies and can be tested in isolation.
type Conversation struct {
	Messages    []Message
	ToolHistory []ToolStatus
	Mode        agent.Mode
	Provider    string
	ModelName   string
}

// NewConversation creates an empty conversation with sensible defaults.
func NewConversation() *Conversation {
	return &Conversation{
		Messages:    make([]Message, 0),
		ToolHistory: make([]ToolStatus, 0),
		Mode:        agent.ModeAct,
	}
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// AppendMessage adds a message to the conversation and returns its index.
func (c *Conversation) AppendMessage(msg Message) int {
	c.Messages = append(c.Messages, msg)
	return len(c.Messages) - 1
}

// MessageCount returns the number of messages.
func (c *Conversation) MessageCount() int {
	return len(c.Messages)
}

// GetMessage returns a pointer to the message at idx, or nil if out of bounds.
func (c *Conversation) GetMessage(idx int) *Message {
	if idx < 0 || idx >= len(c.Messages) {
		return nil
	}
	return &c.Messages[idx]
}

// UpdateMessageContent appends delta to the Content of the message at idx.
func (c *Conversation) UpdateMessageContent(idx int, delta string) {
	if idx < 0 || idx >= len(c.Messages) {
		return
	}
	c.Messages[idx].Content += delta
}

// SetMessageContent replaces the Content of the message at idx.
func (c *Conversation) SetMessageContent(idx int, content string) {
	if idx < 0 || idx >= len(c.Messages) {
		return
	}
	c.Messages[idx].Content = content
}

// LastUserMessage returns the content of the most recent user message.
func (c *Conversation) LastUserMessage() (string, bool) {
	for i := len(c.Messages) - 1; i >= 0; i-- {
		if c.Messages[i].Role == types.RoleUser {
			return c.Messages[i].Content, true
		}
	}
	return "", false
}

// Clear removes all messages and tool history.
func (c *Conversation) Clear() {
	c.Messages = c.Messages[:0]
	c.ToolHistory = c.ToolHistory[:0]
}

// ---------------------------------------------------------------------------
// Tool history
// ---------------------------------------------------------------------------

// AddToolStart records that a tool has started running.
func (c *Conversation) AddToolStart(name string) {
	c.ToolHistory = append(c.ToolHistory, ToolStatus{
		Name:      name,
		Status:    "running",
		StartTime: time.Now(),
	})
}

// MarkToolComplete marks the most recent running tool with the given name as completed.
func (c *Conversation) MarkToolComplete(name string) {
	for i := len(c.ToolHistory) - 1; i >= 0; i-- {
		if c.ToolHistory[i].Name == name && c.ToolHistory[i].Status == "running" {
			c.ToolHistory[i].Status = "completed"
			break
		}
	}
}

// MarkToolFailed marks the most recent running tool with the given name as failed.
func (c *Conversation) MarkToolFailed(name string) {
	for i := len(c.ToolHistory) - 1; i >= 0; i-- {
		if c.ToolHistory[i].Name == name && c.ToolHistory[i].Status == "running" {
			c.ToolHistory[i].Status = "failed"
			break
		}
	}
}

// ClearToolHistory removes all tool status entries.
func (c *Conversation) ClearToolHistory() {
	c.ToolHistory = c.ToolHistory[:0]
}

// LastRunningToolName returns the name of the most recent running tool, or empty string.
func (c *Conversation) LastRunningToolName() string {
	for i := len(c.ToolHistory) - 1; i >= 0; i-- {
		if c.ToolHistory[i].Status == "running" {
			return c.ToolHistory[i].Name
		}
	}
	return ""
}
