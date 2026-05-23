// Package bridge provides type-safe Agent-TUI communication.
// TUIBridge implements agent.StreamCallback by sending typed events over a channel.
package bridge

import (
	"context"
	"time"

	"github.com/liup215/gline/internal/agent"
)

// sendTimeout is the max time to wait for the event channel to accept an event.
// If the TUI is blocked, we drop the event rather than stall the agent forever.
const sendTimeout = 5 * time.Second

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

// send attempts to send an event to the channel with a timeout.
// If the channel is full or the TUI is not reading, the event is dropped
// to prevent the agent goroutine from hanging indefinitely.
func (b *TUIBridge) send(evt AgentEvent) {
	select {
	case b.eventCh <- evt:
	case <-time.After(sendTimeout):
		// Channel blocked; drop event to avoid stalling agent.
	}
}

// OnStreamStart sends a StreamStartEvent.
func (b *TUIBridge) OnStreamStart() {
	b.send(StreamStartEvent{})
}

// OnContent sends a ContentEvent with the incremental text delta.
func (b *TUIBridge) OnContent(delta string) {
	b.send(ContentEvent{Delta: delta})
}

// OnToolCallStart sends a ToolStartEvent when a tool call begins.
func (b *TUIBridge) OnToolCallStart(toolCall agent.ToolCall) {
	b.send(ToolStartEvent{
		Name:  toolCall.Name,
		Input: toolCall.Input,
	})
}

// OnToolCallComplete sends a ToolCompleteEvent when a tool call finishes.
func (b *TUIBridge) OnToolCallComplete(toolCall agent.ToolCall, result string) {
	b.send(ToolCompleteEvent{
		Name:   toolCall.Name,
		Result: result,
	})
}

// OnError sends an ErrorEvent when an error occurs.
func (b *TUIBridge) OnError(err error) {
	b.send(ErrorEvent{Err: err})
}

// OnComplete sends a CompleteEvent when the agent finishes processing.
func (b *TUIBridge) OnComplete() {
	b.send(CompleteEvent{})
}

// OnTaskCreated is a no-op for the TUI bridge (task ID is managed by the agent layer).
func (b *TUIBridge) OnTaskCreated(taskID string) {}

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