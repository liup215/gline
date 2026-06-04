// Package types contains shared types used across the gline codebase
package types

import (
	"encoding/json"
	"sync"
	"time"
	"unicode/utf8"
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

	// ReasoningContent stores model-provided internal reasoning/thinking (if any)
	ReasoningContent string

	// ToolCalls contains tool calls from the assistant
	ToolCalls []ToolCall

	// ToolCallID identifies which tool call this result is for
	ToolCallID string

	// AvailableTools records the list of tools that were available to the
	// assistant when this message was sent.  This is stored as JSON on
	// assistant messages so that users can verify whether the request
	// actually included tools.
	AvailableTools json.RawMessage

	// ToolChoice records the tool_choice setting sent with the request.
	// Common values: "required", "auto", "any", "none", or a JSON object
	// like {"type":"any"}.  This helps diagnose why a model did or did
	// not use tools.
	ToolChoice string

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

	// CurrentTokens is the estimated current token count (rough estimate)
	CurrentTokens int

	// actualInputTokens is the real input token count from API usage
	actualInputTokens int

	// actualOutputTokens is the real output token count from API usage
	actualOutputTokens int

	// mu protects actual token counters
	mu sync.Mutex

	// Complete indicates if the conversation is complete
	Complete bool
}

// AddActualTokens sets real API token usage for the current turn.
// It replaces (not accumulates) because usage from the API already
// represents the total for the latest request.
func (c *Conversation) AddActualTokens(input, output int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.actualInputTokens = input
	c.actualOutputTokens = output
}

// GetActualTokens returns the accumulated real token usage.
func (c *Conversation) GetActualTokens() (input, output int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.actualInputTokens, c.actualOutputTokens
}

// ResetActualTokens resets the accumulated token counters.
func (c *Conversation) ResetActualTokens() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.actualInputTokens = 0
	c.actualOutputTokens = 0
}

// NewConversation creates a new conversation
func NewConversation() *Conversation {
	return &Conversation{
		Messages:  make([]Message, 0),
		MaxTokens: 262000, // Default context window (~262K tokens)
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
	c.ResetActualTokens()
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

// updateTokenCount estimates the token count.
// For CJK / emoji / non-ASCII text, 1 rune ≈ 1 token.
// For ASCII text, ~4 characters ≈ 1 token.
func (c *Conversation) updateTokenCount() {
	total := 0
	for _, msg := range c.Messages {
		total += estimateTokens(msg.Content)
		total += estimateTokens(string(msg.Role))
		total += estimateTokens(msg.ReasoningContent)
		for _, tc := range msg.ToolCalls {
			total += estimateTokens(tc.Name)
			total += estimateTokens(string(tc.Input))
		}
	}
	c.CurrentTokens = total
}

// estimateTokens gives a conservative per-rune token estimate.
func estimateTokens(s string) int {
	if s == "" {
		return 0
	}
	// Fast path when everything is ASCII (English / code).
	ascii := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r < 128 {
			ascii++
			i += size
			continue
		}
		// Non-ASCII: count remaining runes directly (~1 token per rune for CJK).
		remaining := 0
		for i < len(s) {
			r, size = utf8.DecodeRuneInString(s[i:])
			i += size
			remaining++
		}
		return ascii/4 + remaining
	}
	return ascii / 4
}

// TrimToMaxTokens removes oldest messages to fit within token limit.
// It uses GetTotalTokens() so real API usage has priority.
func (c *Conversation) TrimToMaxTokens() {
	if c.GetTotalTokens() <= c.MaxTokens || c.MaxTokens <= 0 {
		return
	}

	// Keep system message if present, remove oldest user/assistant messages
	startIdx := 0
	if len(c.Messages) > 0 && c.Messages[0].Role == RoleSystem {
		startIdx = 1
	}

	// Remove messages from the start until we're under the limit
	for c.GetTotalTokens() > c.MaxTokens && len(c.Messages) > startIdx+2 {
		removed := c.Messages[startIdx]
		removedTokens := estimateTokens(removed.Content) +
			estimateTokens(string(removed.Role)) +
			estimateTokens(removed.ReasoningContent)
		for _, tc := range removed.ToolCalls {
			removedTokens += estimateTokens(tc.Name)
			removedTokens += estimateTokens(string(tc.Input))
		}

		c.Messages = append(c.Messages[:startIdx], c.Messages[startIdx+1:]...)
		c.CurrentTokens -= removedTokens
		if c.CurrentTokens < 0 {
			c.CurrentTokens = 0
		}
		// Accumulated actual tokens are no longer valid after removing history.
		c.ResetActualTokens()
	}
}

// GetTotalTokens returns the best estimate of total tokens used.
// Uses actual API-reported tokens when available, falls back to estimation.
func (c *Conversation) GetTotalTokens() int {
	totalActual := c.actualInputTokens + c.actualOutputTokens
	if totalActual > 0 {
		return totalActual
	}
	return c.CurrentTokens
}

// AutoCompact removes oldest messages to keep usage under 80% of max.
// It preserves the system prompt and the most recent conversation turns.
func (c *Conversation) AutoCompact() {
	keep := 4 // preserve last 2 turns (user+assistant)
	startIdx := 0
	if c.HasSystemPrompt() {
		startIdx = 1
	}
	if len(c.Messages) <= startIdx+keep {
		return
	}
	splitIdx := len(c.Messages) - keep
	if splitIdx < startIdx {
		splitIdx = startIdx
	}
	newMessages := make([]Message, 0, startIdx+keep)
	if startIdx > 0 {
		newMessages = append(newMessages, c.Messages[0]) // system prompt
	}
	newMessages = append(newMessages, c.Messages[splitIdx:]...)
	c.Messages = newMessages
	c.ResetActualTokens()
	c.updateTokenCount()
}

// IsTokenAboveThreshold checks if current tokens exceed the given threshold percentage.
func (c *Conversation) IsTokenAboveThreshold(percent int) bool {
	if c.MaxTokens <= 0 {
		return false
	}
	return c.GetTotalTokens() > c.MaxTokens*percent/100
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
