// Package bridge provides type-safe Agent-TUI communication.
// TUIBridge implements agent.StreamCallback by sending typed events over a channel.
package bridge

import (
	"context"

	"github.com/liup215/gline/internal/agent"
)

// TUIBridge implements agent.StreamCallback by sending typed events over a channel.
// This decouples Agent callbacks from the Bubbletea Program, allowing the bridge
// to be unit-tested independently.
type TUIBridge struct {
	eventCh chan<- AgentEvent
}

// NewTUIBridge creates a new TUIBridge that sends events to the given channel.
// The channel should be buffered to avoid blocking the Agent on high-frequency events.
func NewTUIBridge(eventCh chan<- AgentEvent) *TUIBridge {
	return &TUIBridge{eventCh: eventCh}
}

// OnStreamStart sends a StreamStartEvent.
func (b *TUIBridge) OnStreamStart() {
	b.eventCh <- StreamStartEvent{}
}

// OnContent sends a ContentEvent with the incremental text delta.
func (b *TUIBridge) OnContent(delta string) {
	b.eventCh <- ContentEvent{Delta: delta}
}

// OnToolCallStart sends a ToolStartEvent when a tool call begins.
func (b *TUIBridge) OnToolCallStart(toolCall agent.ToolCall) {
	b.eventCh <- ToolStartEvent{
		Name:  toolCall.Name,
		Input: toolCall.Input,
	}
}

// OnToolCallComplete sends a ToolCompleteEvent when a tool call finishes.
func (b *TUIBridge) OnToolCallComplete(toolCall agent.ToolCall, result string) {
	b.eventCh <- ToolCompleteEvent{
		Name:   toolCall.Name,
		Result: result,
	}
}

// OnError sends an ErrorEvent when an error occurs.
func (b *TUIBridge) OnError(err error) {
	b.eventCh <- ErrorEvent{Err: err}
}

// OnComplete sends a CompleteEvent when the agent finishes processing.
func (b *TUIBridge) OnComplete() {
	b.eventCh <- CompleteEvent{}
}

// AskFollowupQuestion sends an AskQuestionEvent and blocks until the user
// provides an answer via the Reply channel. This synchronous blocking is
// intentional — the Agent goroutine waits for user input before continuing.
func (b *TUIBridge) AskFollowupQuestion(question string, options []string) (string, error) {
	reply := make(chan string, 1)
	b.eventCh <- AskQuestionEvent{
		Question: question,
		Options:  options,
		Reply:    reply,
	}
	// Block until the TUI sends back the user's answer
	answer, ok := <-reply
	if !ok {
		// UI closed the reply channel (e.g., user cancelled) — treat as canceled.
		return "", context.Canceled
	}
	return answer, nil
}

// Compile-time assertion that TUIBridge implements agent.StreamCallback.
var _ agent.StreamCallback = (*TUIBridge)(nil)