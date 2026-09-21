package adkagent

import (
	"fmt"

	"google.golang.org/adk/v2/session"

	glineagent "github.com/liup215/gline/internal/agent"
)

// eventBridge maps the ADK event stream onto gline's StreamCallback
// interface. One bridge tracks one run; it is not safe for concurrent use
// (events are delivered sequentially from the run loop).
//
// Ordering guarantees the TUI relies on:
//   - OnStreamEnd always precedes the OnToolCallStart of the tool calls
//     found in that same model turn (so the tool panel opens after the
//     text slot is finalized);
//   - FunctionCall/FunctionResponse pairs stay paired across events.
type eventBridge struct {
	cb    glineagent.StreamCallback
	dedup StreamDedup

	started bool // an OnStreamStart is open for the current model turn
}

func newEventBridge(cb glineagent.StreamCallback) *eventBridge {
	return &eventBridge{cb: cb}
}

// deliver routes one ADK event into callbacks.
func (b *eventBridge) deliver(ev *session.Event) {
	b.dedup.BeginEvent(ev)
	if ev.Content == nil {
		return
	}
	isModelTurn := modelRoles[ev.Content.Role]

	// Pass 1: text and reasoning deltas. Only model-authored text is
	// emitted — the runner also echoes the user's input back as a user-role
	// event, and tool results ride user-role events too; neither may leak
	// into the assistant text slot.
	for _, part := range ev.Content.Parts {
		switch {
		case part.Text != "" && ev.Content.Role == "thinking":
			// Reasoning text obeys the same dedup rules as body text:
			// emit only the partial deltas, never the aggregate re-send.
			if ev.Partial || !b.dedup.SkipText(ev) {
				if ev.Partial {
					b.cb.OnReasoning(part.Text)
				}
			}
		case part.Text != "" && isModelTurn:
			if b.dedup.SkipText(ev) {
				continue
			}
			if !b.started {
				b.cb.OnStreamStart()
				b.started = true
			}
			b.cb.OnContent(part.Text)
		}
	}

	// Close the text slot before announcing tool calls from this turn.
	if isModelTurn && !ev.Partial && b.started {
		b.cb.OnStreamEnd()
		b.started = false
	}

	// Pass 2: tool activity. FunctionCalls arrive in the model turn's
	// aggregate; FunctionResponses arrive later as their own user-role
	// events after the runner executes the tools.
	for _, part := range ev.Content.Parts {
		if part.FunctionCall != nil {
			b.cb.OnToolCallStart(glineagent.ToolCall{
				ID:    part.FunctionCall.ID,
				Name:  part.FunctionCall.Name,
				Input: toolCallJSON(part.FunctionCall.Args),
			})
		}
		if part.FunctionResponse != nil {
			b.cb.OnToolCallComplete(glineagent.ToolCall{
				ID:   part.FunctionResponse.ID,
				Name: part.FunctionResponse.Name,
			}, functionResponseText(part.FunctionResponse.Response))
		}
	}
}

// functionResponseText extracts the human/model-readable result from a
// FunctionResponse payload. gline tools wrap their output as
// {"result": "<text>"}; anything else is stringified.
func functionResponseText(resp map[string]any) string {
	if resp == nil {
		return ""
	}
	if r, ok := resp["result"].(string); ok {
		return r
	}
	return fmt.Sprintf("%v", resp)
}
