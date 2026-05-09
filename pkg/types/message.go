// Package types contains shared types used across the gline codebase
package types

import (
	"encoding/json"
	"time"
)

// Role represents the role of a message sender
type Role string

const (
	// RoleSystem represents a system message
	RoleSystem Role = "system"
	// RoleUser represents a user message
	RoleUser Role = "user"
	// RoleAssistant represents an assistant message
	RoleAssistant Role = "assistant"
	// RoleTool represents a tool result message
	RoleTool Role = "tool"
)

// Message represents a single message in a conversation
type Message struct {
	// Role is the sender role
	Role Role

	// Content is the message content
	Content string

	// ToolCalls contains tool calls from the assistant
	ToolCalls []ToolCall

	// ToolCallID identifies which tool call this result is for
	ToolCallID string

	// Timestamp when the message was created
	Timestamp time.Time
}

// ToolCall represents a tool call in a message
type ToolCall struct {
	// ID is the unique identifier for this tool call
	ID string

	// Name is the name of the tool to call
	Name string

	// Input is the JSON input for the tool
	Input json.RawMessage
}

// Conversation represents a conversation between user and assistant
type Conversation struct {
	// Messages is the list of messages
	Messages []Message

	// MaxTokens is the maximum number of tokens allowed
	MaxTokens int

	// CurrentTokens is the estimated current token count
	CurrentTokens int

	// Complete indicates if the conversation is complete
	Complete bool
}

// NewConversation creates a new conversation
func NewConversation() *Conversation {
	return &Conversation{
		Messages:  make([]Message, 0),
		MaxTokens: 128000, // Default to a reasonable limit
	}
}

// AddMessage adds a message to the conversation
func (c *Conversation) AddMessage(msg Message) {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	c.Messages = append(c.Messages, msg)
	c.updateTokenCount()
}

// GetMessages returns all messages in the conversation
func (c *Conversation) GetMessages() []Message {
	return c.Messages
}

// GetLastMessage returns the last message in the conversation
func (c *Conversation) GetLastMessage() *Message {
	if len(c.Messages) == 0 {
		return nil
	}
	return &c.Messages[len(c.Messages)-1]
}

// Clear removes all messages from the conversation
func (c *Conversation) Clear() {
	c.Messages = make([]Message, 0)
	c.CurrentTokens = 0
	c.Complete = false
}

// MarkIncomplete marks the conversation as needing more processing.
func (c *Conversation) MarkIncomplete() {
	c.Complete = false
}

// SetComplete marks the conversation as complete
func (c *Conversation) SetComplete() {
	c.Complete = true
}

// IsComplete returns true if the conversation is complete
func (c *Conversation) IsComplete() bool {
	return c.Complete
}

// updateTokenCount estimates the token count
// This is a simple estimation; for production use, consider using a proper tokenizer
func (c *Conversation) updateTokenCount() {
	// Rough estimate: 1 token ≈ 4 characters for English text
	totalChars := 0
	for _, msg := range c.Messages {
		totalChars += len(msg.Content)
		totalChars += len(string(msg.Role))
		for _, tc := range msg.ToolCalls {
			totalChars += len(tc.Name)
			totalChars += len(tc.Input)
		}
	}
	c.CurrentTokens = totalChars / 4
}

// TrimToMaxTokens removes oldest messages to fit within token limit
func (c *Conversation) TrimToMaxTokens() {
	if c.CurrentTokens <= c.MaxTokens {
		return
	}

	// Keep system message if present, remove oldest user/assistant messages
	startIdx := 0
	if len(c.Messages) > 0 && c.Messages[0].Role == RoleSystem {
		startIdx = 1
	}

	// Remove messages from the start until we're under the limit
	for c.CurrentTokens > c.MaxTokens && len(c.Messages) > startIdx+2 {
		removed := c.Messages[startIdx]
		c.Messages = append(c.Messages[:startIdx], c.Messages[startIdx+1:]...)
		// Update token count
		c.CurrentTokens -= (len(removed.Content) + len(string(removed.Role))) / 4
	}
}

// ToJSON returns the conversation as JSON
func (c *Conversation) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MessageCount returns the number of messages
func (c *Conversation) MessageCount() int {
	return len(c.Messages)
}

// HasSystemPrompt returns true if the conversation has a system prompt
func (c *Conversation) HasSystemPrompt() bool {
	return len(c.Messages) > 0 && c.Messages[0].Role == RoleSystem
}

// SetSystemPrompt sets the system prompt
func (c *Conversation) SetSystemPrompt(content string) {
	if c.HasSystemPrompt() {
		c.Messages[0].Content = content
	} else {
		// Insert at the beginning
		c.Messages = append([]Message{{Role: RoleSystem, Content: content}}, c.Messages...)
	}
	c.updateTokenCount()
}
