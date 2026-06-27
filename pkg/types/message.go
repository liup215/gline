// Package types contains shared types used across the gline codebase
package types

import (
	"encoding/json"
	"fmt"
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

	// PerMessageSoftCap is the maximum allowed tokens for a single message
	// before it is locally truncated. 0 means disabled.
	PerMessageSoftCap int

	// ResponseBuffer is the number of tokens reserved for the model's response
	// when checking the request budget in agent.go.
	ResponseBuffer int
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
		Messages:          make([]Message, 0),
		MaxTokens:         128000, // Default context window (~128K tokens)
		PerMessageSoftCap: 6000,
		ResponseBuffer:    8192,
	}
}

// AddMessage adds a message to the conversation. If the message content
// exceeds PerMessageSoftCap it is locally truncated to keep a single message
// from exploding the context.
func (c *Conversation) AddMessage(msg Message) {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	msg = c.truncateMessage(msg)
	c.Messages = append(c.Messages, msg)
	c.updateTokenCount()
}

// truncateMessage applies a soft cap to a single message's content. It keeps
// the head and tail of the content and replaces the middle with an omission
// marker so useful context at both ends is preserved.
func (c *Conversation) truncateMessage(msg Message) Message {
	softCap := c.PerMessageSoftCap
	if softCap <= 0 {
		return msg
	}
	contentTokens := EstimateTokens(msg.Content)
	if contentTokens <= softCap {
		return msg
	}

	targetHead := softCap / 4
	targetTail := softCap / 4
	// Ensure at least a small middle window remains when possible.
	if targetHead < 1 {
		targetHead = 1
	}
	if targetTail < 1 {
		targetTail = 1
	}

	head := truncateToTokens(msg.Content, targetHead)
	tail := truncateFromEndToTokens(msg.Content, targetTail)
	omitted := contentTokens - EstimateTokens(head) - EstimateTokens(tail)
	if omitted < 0 {
		omitted = contentTokens / 2
	}
	msg.Content = fmt.Sprintf(
		"%s\n[... %d tokens omitted ...]\n%s",
		head,
		omitted,
		tail,
	)
	return msg
}

// truncateToTokens returns the leading portion of s whose token count does not
// exceed maxTokens.
func truncateToTokens(s string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	tokens := 0
	for i, r := range s {
		runeTokens := 1
		if r < 128 && (r == ' ' || r == '\t' || r == '\n' || r == '\r') {
			runeTokens = 0
		} else if r < 128 {
			// Approximation; callers use this on word boundaries so this is
			// acceptable for the head/tail truncation heuristic.
			runeTokens = 1
		}
		tokens += runeTokens
		if tokens > maxTokens {
			return s[:i]
		}
	}
	return s
}

// truncateFromEndToTokens returns the trailing portion of s whose token count
// does not exceed maxTokens.
func truncateFromEndToTokens(s string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	tokens := 0
	var start int
	for i := len(s); i > 0; {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		i -= size
		runeTokens := 1
		if r < 128 && (r == ' ' || r == '\t' || r == '\n' || r == '\r') {
			runeTokens = 0
		}
		tokens += runeTokens
		if tokens > maxTokens {
			start = i + size
			break
		}
		start = i
	}
	return s[start:]
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
		total += EstimateTokens(msg.Content)
		total += EstimateTokens(string(msg.Role))
		total += EstimateTokens(msg.ReasoningContent)
		for _, tc := range msg.ToolCalls {
			total += EstimateTokens(tc.Name)
			total += EstimateTokens(string(tc.Input))
		}
	}
	c.CurrentTokens = total
}

// EstimateTokens gives a conservative per-rune token estimate.
// Uses a more accurate heuristic that accounts for whitespace,
// punctuation, and mixed-script text commonly seen in prompts.
// It is exported so packages such as the summarizer can reuse the same
// estimation logic as Conversation.
func EstimateTokens(s string) int {
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
// Before dropping a tool result message it attempts to replace the content
// with a short placeholder summary, so the agent still knows that a tool
// was called. It uses GetTotalTokens() so real API usage has priority.
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

		// Try to preserve a hint for large tool results instead of deleting them
		// outright. This keeps the agent aware of work already performed.
		if removed.Role == RoleTool && len(removed.Content) > 200 {
			placeholder := fmt.Sprintf(
				"[Result of %s was %d tokens; truncated by context manager. Re-run the tool if needed.]",
				removed.ToolCallID,
				EstimateTokens(removed.Content),
			)
			removed.Content = placeholder
			c.Messages[startIdx] = removed
			c.updateTokenCount()
			// If this single replacement already brought us under budget, stop.
			if c.GetTotalTokens() <= c.MaxTokens {
				c.ResetActualTokens()
				return
			}
			// Otherwise continue removing from the next (now smaller) message.
			continue
		}

		removedTokens := messageTokens(removed)

		c.Messages = append(c.Messages[:startIdx], c.Messages[startIdx+1:]...)
		c.CurrentTokens -= removedTokens
		if c.CurrentTokens < 0 {
			c.CurrentTokens = 0
		}
		// Accumulated actual tokens are no longer valid after removing history.
		c.ResetActualTokens()
	}
}

// messageTokens returns the estimated token count of a single message.
func messageTokens(m Message) int {
	total := EstimateTokens(m.Content) +
		EstimateTokens(string(m.Role)) +
		EstimateTokens(m.ReasoningContent)
	for _, tc := range m.ToolCalls {
		total += EstimateTokens(tc.Name)
		total += EstimateTokens(string(tc.Input))
	}
	return total
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

// AutoCompact removes oldest messages to keep usage under the max context
// window. It now works in token-space rather than message-space:
//
// Strategy:
//   - Do nothing if total tokens are below the low watermark (60%).
//   - First truncate any single oversized old message (excluding system and
//     the most recent user/assistant pair) that exceeds PerMessageSoftCap.
//   - If still over the high watermark (80%), drop whole messages from the
//     middle, starting with the largest, while preserving the first
//     user-assistant pair and the most recent pair.
//   - Inject a summary marker so the model knows context was truncated.
func (c *Conversation) AutoCompact() {
	if c.MaxTokens <= 0 {
		return
	}
	total := c.GetTotalTokens()
	lowWatermark := c.MaxTokens * 6 / 10  // 60%
	highWatermark := c.MaxTokens * 8 / 10 // 80%

	if total <= lowWatermark {
		return
	}

	startIdx := 0
	if c.HasSystemPrompt() {
		startIdx = 1
	}

	// Always preserve the first user-assistant pair after system prompt.
	firstPairEnd := startIdx + 1 // inclusive index of first assistant msg
	if firstPairEnd >= len(c.Messages) {
		return
	}

	// Phase 1: truncate individual oversized messages in the middle region.
	if total > highWatermark {
		for i := firstPairEnd + 1; i < len(c.Messages)-2; i++ {
			msgTokens := messageTokens(c.Messages[i])
			if c.PerMessageSoftCap > 0 && msgTokens > c.PerMessageSoftCap {
				c.Messages[i] = c.truncateMessage(c.Messages[i])
			}
		}
		c.updateTokenCount()
		total = c.GetTotalTokens()
	}

	if total <= lowWatermark {
		return
	}

	// Phase 2: drop whole messages from the middle, preferring the largest
	// token consumers first. Keep the first pair and the last pair.
	for c.GetTotalTokens() > highWatermark && len(c.Messages) > startIdx+4 {
		// Find the largest removable message in the middle region.
		maxIdx := -1
		maxTokens := 0
		for i := firstPairEnd + 1; i < len(c.Messages)-2; i++ {
			t := messageTokens(c.Messages[i])
			if t > maxTokens {
				maxTokens = t
				maxIdx = i
			}
		}
		if maxIdx < 0 {
			break
		}

		removed := c.Messages[maxIdx]
		removedTokens := messageTokens(removed)
		c.Messages = append(c.Messages[:maxIdx], c.Messages[maxIdx+1:]...)
		c.CurrentTokens -= removedTokens
		if c.CurrentTokens < 0 {
			c.CurrentTokens = 0
		}
	}

	c.ResetActualTokens()
	c.updateTokenCount()

	// Insert a compact marker so the model knows context was trimmed.
	c.Messages = append(c.Messages[:firstPairEnd+1], append([]Message{{
		Role:    RoleSystem,
		Content: "[Context compacted: older messages were summarized or removed to fit token budget. Original task context preserved.]",
	}}, c.Messages[firstPairEnd+1:]...)...)
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
