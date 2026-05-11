// Package bridge provides type-safe event types for Agent-TUI communication.
package bridge

// AgentEvent is the unified interface for all events produced by Agent callbacks.
type AgentEvent interface {
	agentEvent()
}

// ContentEvent signals that a new content delta has arrived from the LLM stream.
type ContentEvent struct {
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
	Reply    chan string
}

// Compile-time interface assertions.
func (ContentEvent) agentEvent()      {}
func (ToolStartEvent) agentEvent()    {}
func (ToolCompleteEvent) agentEvent() {}
func (ErrorEvent) agentEvent()        {}
func (StreamStartEvent) agentEvent()  {}
func (StreamEndEvent) agentEvent()    {}
func (CompleteEvent) agentEvent()     {}
func (AskQuestionEvent) agentEvent()  {}
