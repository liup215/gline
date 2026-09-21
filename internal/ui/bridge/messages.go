// Package bridge provides type-safe event types for Agent-TUI communication.
package bridge

import "sync"

// AgentEvent is the unified interface for all events produced by Agent callbacks.
type AgentEvent interface {
	agentEvent()
}

// ContentEvent signals that a new content delta has arrived from the LLM stream.
type ContentEvent struct {
	Delta string
}

// ReasoningEvent signals that a new reasoning/thinking delta has arrived.
type ReasoningEvent struct {
	Delta string
}

// ToolStartEvent signals that a tool call has started.
type ToolStartEvent struct {
	Name  string
	Input string
}

// ToolCompleteEvent signals that a tool call has completed with a result.
type ToolCompleteEvent struct {
	Name   string
	Result string
}

// ErrorEvent signals that an error occurred during processing.
type ErrorEvent struct {
	Err error
}

// StreamStartEvent signals the beginning of a streaming response.
type StreamStartEvent struct{}

// StreamEndEvent signals the end of a streaming response.
type StreamEndEvent struct{}

// CompleteEvent signals that the agent has finished processing this turn.
type CompleteEvent struct{}

// AskQuestionEvent signals that the agent needs to ask the user a follow-up question.
type AskQuestionEvent struct {
	Question string
	Options  []string
	Reply    *PendingAsk
}

// PendingAsk is one unanswered AskFollowupQuestion: the reply channel plus a
// close-once guard. Multiple tool calls can ask concurrently (parallel tool
// calls), and both the UI (Esc) and the agent (abort/teardown) may close the
// question — Abort is idempotent so a double close cannot panic.
type PendingAsk struct {
	Ch   chan string
	once sync.Once
}

// Ask delivers the user's answer without blocking (channel is buffered).
func (p *PendingAsk) Ask(answer string) {
	select {
	case p.Ch <- answer:
	default:
	}
}

// Abort closes the reply channel exactly once, unblocking the waiting tool
// goroutine with context.Canceled. Safe to call from every abort path.
func (p *PendingAsk) Abort() {
	p.once.Do(func() { close(p.Ch) })
}

// Compile-time interface assertions.
func (ContentEvent) agentEvent()      {}
func (ReasoningEvent) agentEvent()    {}
func (ToolStartEvent) agentEvent()    {}
func (ToolCompleteEvent) agentEvent() {}
func (ErrorEvent) agentEvent()        {}
func (StreamStartEvent) agentEvent()  {}
func (StreamEndEvent) agentEvent()    {}
func (CompleteEvent) agentEvent()     {}
func (AskQuestionEvent) agentEvent()  {}
