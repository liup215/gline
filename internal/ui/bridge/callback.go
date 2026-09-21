// Package bridge provides type-safe Agent-TUI communication.
// TUIBridge implements agent.StreamCallback by sending typed events over a channel.
package bridge

import (
	"context"
	"sync"
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

	// pending tracks the unanswered AskFollowupQuestion calls so they can all
	// be unblocked on abort/cancel. Tools ask concurrently for parallel tool
	// calls; a blocked asker would otherwise wait forever when its channel
	// was overwritten in the UI queue or the run ended.
	mu      sync.Mutex
	pending []*PendingAsk
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

// OnStreamEnd sends a StreamEndEvent.
func (b *TUIBridge) OnStreamEnd() {
	b.send(StreamEndEvent{})
}

// OnContent sends a ContentEvent with the incremental text delta.
func (b *TUIBridge) OnContent(delta string) {
	b.send(ContentEvent{Delta: delta})
}

// OnReasoning sends a ReasoningEvent with the incremental reasoning delta.
func (b *TUIBridge) OnReasoning(delta string) {
	b.send(ReasoningEvent{Delta: delta})
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
	ask := &PendingAsk{Ch: make(chan string, 1)}
	b.mu.Lock()
	b.pending = append(b.pending, ask)
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		for i, p := range b.pending {
			if p == ask {
				b.pending = append(b.pending[:i], b.pending[i+1:]...)
				break
			}
		}
		b.mu.Unlock()
	}()
	b.eventCh <- AskQuestionEvent{
		Question: question,
		Options:  options,
		Reply:    ask,
	}
	// Block until the TUI sends back the user's answer or the question is
	// aborted (channel closed) — the latter is surfaced as cancellation.
	answer, ok := <-ask.Ch
	if !ok {
		// Question aborted (user cancel / run teardown) — treat as canceled.
		return "", context.Canceled
	}
	return answer, nil
}

// AbortPendingQuestions aborts every unanswered AskFollowupQuestion,
// unblocking the tool goroutines waiting for user input. Called when the run
// ends or is aborted so abandoned approvals cannot hang the agent loop.
// Idempotent: questions the UI already aborted are skipped by the close-once
// guard.
func (b *TUIBridge) AbortPendingQuestions() {
	b.mu.Lock()
	pending := b.pending
	b.pending = nil
	b.mu.Unlock()
	for _, p := range pending {
		p.Abort()
	}
}

// Compile-time assertion that TUIBridge implements agent.StreamCallback.
var _ agent.StreamCallback = (*TUIBridge)(nil)
