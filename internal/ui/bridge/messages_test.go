package bridge

import (
	"errors"
	"testing"
)

// Verify that all event types implement AgentEvent at compile time.
var _ AgentEvent = ContentEvent{}
var _ AgentEvent = ToolStartEvent{}
var _ AgentEvent = ToolCompleteEvent{}
var _ AgentEvent = ErrorEvent{}
var _ AgentEvent = StreamStartEvent{}
var _ AgentEvent = StreamEndEvent{}
var _ AgentEvent = CompleteEvent{}
var _ AgentEvent = AskQuestionEvent{}

func TestContentEvent(t *testing.T) {
	evt := ContentEvent{Delta: "hello"}
	if evt.Delta != "hello" {
		t.Fatalf("expected Delta='hello', got %q", evt.Delta)
	}
}

func TestToolStartEvent(t *testing.T) {
	evt := ToolStartEvent{Name: "read_file", Input: `{"path":"."}`}
	if evt.Name != "read_file" {
		t.Fatalf("expected Name='read_file', got %q", evt.Name)
	}
	if evt.Input != `{"path":"."}` {
		t.Fatalf("expected Input='{\"path\":\".\"}', got %q", evt.Input)
	}
}

func TestToolCompleteEvent(t *testing.T) {
	evt := ToolCompleteEvent{Name: "read_file", Result: "content"}
	if evt.Name != "read_file" || evt.Result != "content" {
		t.Fatalf("unexpected fields: %+v", evt)
	}
}

func TestErrorEvent(t *testing.T) {
	err := errors.New("something went wrong")
	evt := ErrorEvent{Err: err}
	if evt.Err != err {
		t.Fatalf("expected Err=%v, got %v", err, evt.Err)
	}
}

func TestStreamStartEvent(t *testing.T) {
	_ = StreamStartEvent{}
}

func TestStreamEndEvent(t *testing.T) {
	_ = StreamEndEvent{}
}

func TestCompleteEvent(t *testing.T) {
	_ = CompleteEvent{}
}

func TestAskQuestionEvent(t *testing.T) {
	reply := make(chan string, 1)
	evt := AskQuestionEvent{Question: "Continue?", Options: []string{"Yes", "No"}, Reply: reply}
	if evt.Question != "Continue?" {
		t.Fatalf("unexpected Question: %q", evt.Question)
	}
	if len(evt.Options) != 2 || evt.Options[0] != "Yes" {
		t.Fatalf("unexpected Options: %v", evt.Options)
	}
	if evt.Reply != reply {
		t.Fatal("expected Reply channel to match")
	}
}
