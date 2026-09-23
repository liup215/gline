package adkagent

import (
	"errors"
	"testing"

	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"

)

// partialEvent builds a partial model event carrying one text delta.
func partialEvent(text string) *session.Event {
	return &session.Event{
		LLMResponse: model.LLMResponse{
			Content: &genai.Content{Role: "model", Parts: []*genai.Part{{Text: text}}},
			Partial: true,
		},
	}
}

// aggregateEvent builds a non-partial model event carrying text.
func aggregateEvent(text string) *session.Event {
	ev := partialEvent(text)
	ev.Partial = false
	ev.TurnComplete = true
	return ev
}

// userEvent builds a non-model event (tool results ride these).
func userEvent(text string) *session.Event {
	return &session.Event{
		LLMResponse: model.LLMResponse{
			Content: &genai.Content{Role: "user", Parts: []*genai.Part{{Text: text}}},
		},
	}
}

func TestStreamDedupSuppressesAggregateResend(t *testing.T) {
	var d StreamDedup
	texts := []string{}
	for _, ev := range []*session.Event{
		partialEvent("Hel"), partialEvent("lo "), partialEvent("world"),
		aggregateEvent("Hello world"), // SSE aggregate re-send
	} {
		d.BeginEvent(ev)
		for _, p := range ev.Content.Parts {
			if p.Text != "" && !d.SkipText(ev) {
				texts = append(texts, p.Text)
			}
		}
	}
	want := "Hello world"
	got := texts[0] + texts[1] + texts[2]
	if len(texts) != 3 || got != want {
		t.Fatalf("texts=%v (n=%d), want deltas only totaling %q", texts, len(texts), want)
	}
}

func TestStreamDedupLetsBareAggregateThrough(t *testing.T) {
	var d StreamDedup
	// Non-streaming style: user event first (tool result), then a bare
	// aggregate model turn with no deltas in front of it. User-role text
	// is never collected (the bridge only emits model turns).
	got := []string{}
	for _, ev := range []*session.Event{userEvent("tool result"), aggregateEvent("full reply")} {
		d.BeginEvent(ev)
		for _, p := range ev.Content.Parts {
			if p.Text != "" && ev.Content.Role == "model" && !d.SkipText(ev) {
				got = append(got, p.Text)
			}
		}
	}
	if len(got) != 1 || got[0] != "full reply" {
		t.Fatalf("got=%v, want bare aggregate preserved", got)
	}
}

func TestEventError(t *testing.T) {
	mk := func(code, msg string) *session.Event {
		ev := &session.Event{LLMResponse: model.LLMResponse{}}
		ev.ErrorCode = code
		ev.ErrorMessage = msg
		return ev
	}
	if err := EventError(nil); err != nil {
		t.Fatalf("nil event: %v", err)
	}
	if err := EventError(aggregateEvent("fine")); err != nil {
		t.Fatalf("clean event: %v", err)
	}
	err := EventError(mk("STREAM_ERROR", "boom"))
	if err == nil || err.Error() != "boom" {
		t.Fatalf("STREAM_ERROR = %v, want bare message", err)
	}
	err = EventError(mk("AUTH", "denied"))
	if err == nil || err.Error() != "AUTH: denied" {
		t.Fatalf("AUTH = %v, want prefixed", err)
	}
	err = EventError(mk("AUTH", ""))
	if err == nil || !errors.Is(err, err) || err.Error() != "AUTH" {
		t.Fatalf("empty msg = %v, want code only", err)
	}
}

// thinkingEvent builds a partial thinking-role event carrying one delta.
func thinkingEvent(text string) *session.Event {
	return &session.Event{
		LLMResponse: model.LLMResponse{
			Content: &genai.Content{Role: "thinking", Parts: []*genai.Part{{Text: text}}},
			Partial: true,
		},
	}
}

// toolCallAggregate builds a non-partial model event with one function call.
func toolCallAggregate() *session.Event {
	return &session.Event{
		LLMResponse: model.LLMResponse{
			Content: &genai.Content{Role: "model", Parts: []*genai.Part{
				{FunctionCall: &genai.FunctionCall{ID: "call-1", Name: "read_file"}},
			}},
			TurnComplete: true,
		},
	}
}

// TestBridgeThinkingToolTurnClosesStream covers the thinking → tool-call turn
// (no answer text): the stream slot must open at the first reasoning delta and
// close before the tool call is announced, or UIs keyed on the streaming flag
// keep showing the thinking bubble as still running after the turn ended.
func TestBridgeThinkingToolTurnClosesStream(t *testing.T) {
	rec := &recordingCallback{}
	b := newEventBridge(rec)

	for _, ev := range []*session.Event{
		thinkingEvent("用户"), thinkingEvent("在思考"),
		toolCallAggregate(),
	} {
		b.dedup.BeginEvent(ev)
		if err := b.deliver(ev); err != nil {
			t.Fatalf("deliver: %v", err)
		}
	}

	// recordingCallback records: stream_start, reasoning, stream_end,
	// tool_start:<name> — in callback order.
	want := []string{"stream_start", "reasoning", "reasoning", "stream_end", "tool_start:read_file"}
	if got := rec.order; !equalStrings(got, want) {
		t.Fatalf("event order = %v, want %v", got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
