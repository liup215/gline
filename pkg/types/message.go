// Package types contains shared types used across the gline codebase
package types

import (
	"encoding/json"
	"fmt"
	"math"
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
// Uses a more accurate heuristic that accounts for whitespace,
// punctuation, and mixed-script text commonly seen in prompts.
func estimateTokens(s string) int {
	if s == "" {
		return 0
	}
	total := 0
	wordRun := 0    // consecutive non-space ASCII characters
	spaceRun := 0   // consecutive whitespace characters
	nonASCII := 0   // non-ASCII runes
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r < 128 {
			if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
				if wordRun > 0 {
					// A word is roughly ceil(len/3) tokens for short words,
					// or len/4 for longer runs.  Use an average.
					total += (wordRun + 2) / 3
					wordRun = 0
				}
				spaceRun++
			} else {
				if spaceRun > 0 {
					// Whitespace is usually merged with adjacent tokens.
					spaceRun = 0
				}
				wordRun++
			}
		} else {
			// Flush any pending ASCII word
			if wordRun > 0 {
				total += (wordRun + 2) / 3
				wordRun = 0
			}
			spaceRun = 0
			nonASCII++
		}
		i += size
	}
	if wordRun > 0 {
		total += (wordRun + 2) / 3
	}
	// Non-ASCII (CJK, emoji, etc.) ~1 token per rune.
	return total + nonASCII
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

// AutoCompact removes oldest messages to keep usage under the max
// context window.  Instead of keeping a fixed number of messages,
// it discards a *fraction* of the conversation, similar to Cline.
//
// Strategy:
//   - Always keep the first user-assistant pair (index 0-1).
//   - Keep the most recent messages (by default last half or quarter).
//   - Remove an even number of messages so user/assistant pairs stay
//     aligned.
//   - Inject a summary marker so the model knows context was truncated.
func (c *Conversation) AutoCompact() {
	startIdx := 0
	if c.HasSystemPrompt() {
		startIdx = 1
	}

	// Always preserve the first user-assistant pair after system prompt.
	// This keeps the original task description intact.
	firstPairEnd := startIdx + 1 // inclusive index of first assistant msg
	if firstPairEnd >= len(c.Messages) {
		return
	}

	// Total messages available for truncation (after first pair).
	truncatable := len(c.Messages) - firstPairEnd - 1
	if truncatable <= 0 {
		return
	}

	// Decide fraction to keep.  If current tokens exceed the context
	// window by a large margin we discard more aggressively.
	totalTokens := c.GetTotalTokens()
	keepFraction := 0.5 // default: keep last half
	if c.MaxTokens > 0 && totalTokens > c.MaxTokens {
		// Severely over budget → keep only the last quarter
		keepFraction = 0.25
	}

	// Compute number of messages to remove.  We want an even count so
	// user/assistant pairs stay aligned.
	messagesToRemove := int(math.Floor(float64(truncatable) * (1 - keepFraction)))
	if messagesToRemove%2 != 0 {
		messagesToRemove-- // force even
	}
	if messagesToRemove <= 0 {
		return
	}

	// The split point: messages [firstPairEnd+1 .. splitIdx] are removed.
	splitIdx := firstPairEnd + messagesToRemove
	if splitIdx >= len(c.Messages)-1 {
		// Don't remove the very last message.
		return
	}

	// Build a compact summary of dropped messages.
	var droppedToolCount int
	for _, m := range c.Messages[firstPairEnd+1 : splitIdx+1] {
		droppedToolCount += len(m.ToolCalls)
	}
	toolSuffix := ""
	if droppedToolCount != 1 {
		toolSuffix = "s"
	}
	ctxMsg := fmt.Sprintf(
		"[Context compacted: %d intermediate messages omitted (including %d tool call%s). Original task context preserved.]",
		messagesToRemove,
		droppedToolCount,
		toolSuffix,
	)

	// Rebuild messages: [system?, first pair, summary marker, kept tail]
	keptTail := len(c.Messages) - splitIdx - 1
	newCap := startIdx + 2 + 1 + keptTail // sys + pair + marker + tail
	newMessages := make([]Message, 0, newCap)
	if startIdx > 0 {
		newMessages = append(newMessages, c.Messages[0]) // system prompt
	}
	newMessages = append(newMessages, c.Messages[startIdx:firstPairEnd+1]...) // first user-assistant pair
	newMessages = append(newMessages, Message{
		Role:    RoleSystem,
		Content: ctxMsg,
	})
	newMessages = append(newMessages, c.Messages[splitIdx+1:]...)
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
