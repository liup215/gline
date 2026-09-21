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
	stuck stuckDetector

	started bool // an OnStreamStart is open for the current model turn
}

func newEventBridge(cb glineagent.StreamCallback) *eventBridge {
	return &eventBridge{cb: cb}
}

// deliver routes one ADK event into callbacks. It returns a *stuckError when
// the loop detector calls the run dead (Phase 5b): the caller should surface
// the error and abort the run. pi-go additionally tells the model why and
// resumes; that recovery loop lands in Phase 6.
func (b *eventBridge) deliver(ev *session.Event) error {
	b.dedup.BeginEvent(ev)
	b.stuck.beginEvent()
	if ev.Content == nil {
		return nil
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
					if err := stuckErr(b.stuck.observeOutput(part.Text)); err != nil {
						return err
					}
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
			if err := stuckErr(b.stuck.observeOutput(part.Text)); err != nil {
				return err
			}
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
			if err := stuckErr(b.stuck.observe(part.FunctionCall.ID, part.FunctionCall.Name, part.FunctionCall.Args)); err != nil {
				return err
			}
			b.cb.OnToolCallStart(glineagent.ToolCall{
				ID:    part.FunctionCall.ID,
				Name:  part.FunctionCall.Name,
				Input: toolCallJSON(part.FunctionCall.Args),
			})
		}
		if part.FunctionResponse != nil {
			// A changed result on a repeated call is progress, not a loop —
			// let it reset the identical-call streak before the next call
			// is observed. Cycle detection also runs here: a cycle is only
			// decidable once every call in the window has its result.
			if err := stuckErr(b.stuck.observeResult(part.FunctionResponse.ID, part.FunctionResponse.Name, part.FunctionResponse.Response)); err != nil {
				return err
			}
			// ADK wraps tool errors as {"error": ...}; anything else is a
			// success and resets the error streaks.
			_, isErr := part.FunctionResponse.Response["error"]
			if err := stuckErr(b.stuck.observeError(part.FunctionResponse.ID, part.FunctionResponse.Name, isErr)); err != nil {
				return err
			}
			b.cb.OnToolCallComplete(glineagent.ToolCall{
				ID:   part.FunctionResponse.ID,
				Name: part.FunctionResponse.Name,
			}, functionResponseText(part.FunctionResponse.Response))
		}
	}
	return nil
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
