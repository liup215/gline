package bridge

import (
	"errors"
	"testing"
	"time"

	"github.com/liup215/gline/internal/agent"
)

func TestTUIBridge_OnStreamStart(t *testing.T) {
	ch := make(chan AgentEvent, 1)
	b := NewTUIBridge(ch)

	b.OnStreamStart()

	select {
	case evt := <-ch:
		if _, ok := evt.(StreamStartEvent); !ok {
			t.Fatalf("expected StreamStartEvent, got %T", evt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for StreamStartEvent")
	}
}

func TestTUIBridge_OnContent(t *testing.T) {
	ch := make(chan AgentEvent, 1)
	b := NewTUIBridge(ch)

	b.OnContent("hello world")

	select {
	case evt := <-ch:
		ce, ok := evt.(ContentEvent)
		if !ok {
			t.Fatalf("expected ContentEvent, got %T", evt)
		}
		if ce.Delta != "hello world" {
			t.Fatalf("expected Delta='hello world', got %q", ce.Delta)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ContentEvent")
	}
}

func TestTUIBridge_OnToolCallStart(t *testing.T) {
	ch := make(chan AgentEvent, 1)
	b := NewTUIBridge(ch)

	b.OnToolCallStart(agent.ToolCall{Name: "read_file", Input: `{"path":"."}`})

	select {
	case evt := <-ch:
		tse, ok := evt.(ToolStartEvent)
		if !ok {
			t.Fatalf("expected ToolStartEvent, got %T", evt)
		}
		if tse.Name != "read_file" {
			t.Fatalf("expected Name='read_file', got %q", tse.Name)
		}
		if tse.Input != `{"path":"."}` {
			t.Fatalf("expected Input='{\"path\":\".\"}', got %q", tse.Input)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ToolStartEvent")
	}
}

func TestTUIBridge_OnToolCallComplete(t *testing.T) {
	ch := make(chan AgentEvent, 1)
	b := NewTUIBridge(ch)

	b.OnToolCallComplete(agent.ToolCall{Name: "read_file"}, "file content")

	select {
	case evt := <-ch:
		tce, ok := evt.(ToolCompleteEvent)
		if !ok {
			t.Fatalf("expected ToolCompleteEvent, got %T", evt)
		}
		if tce.Name != "read_file" {
			t.Fatalf("expected Name='read_file', got %q", tce.Name)
		}
		if tce.Result != "file content" {
			t.Fatalf("expected Result='file content', got %q", tce.Result)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ToolCompleteEvent")
	}
}

func TestTUIBridge_OnError(t *testing.T) {
	ch := make(chan AgentEvent, 1)
	b := NewTUIBridge(ch)

	testErr := errors.New("something went wrong")
	b.OnError(testErr)

	select {
	case evt := <-ch:
		ee, ok := evt.(ErrorEvent)
		if !ok {
			t.Fatalf("expected ErrorEvent, got %T", evt)
		}
		if ee.Err.Error() != "something went wrong" {
			t.Fatalf("expected Err='something went wrong', got %v", ee.Err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ErrorEvent")
	}
}

func TestTUIBridge_OnComplete(t *testing.T) {
	ch := make(chan AgentEvent, 1)
	b := NewTUIBridge(ch)

	b.OnComplete()

	select {
	case evt := <-ch:
		if _, ok := evt.(CompleteEvent); !ok {
			t.Fatalf("expected CompleteEvent, got %T", evt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for CompleteEvent")
	}
}

func TestTUIBridge_AskFollowupQuestion(t *testing.T) {
	ch := make(chan AgentEvent, 1)
	b := NewTUIBridge(ch)

	// Start AskFollowupQuestion in a goroutine — it blocks until reply is sent
	answerCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		answer, err := b.AskFollowupQuestion("Continue?", []string{"Yes", "No"})
		if err != nil {
			errCh <- err
			return
		}
		answerCh <- answer
	}()

	// Wait for the AskQuestionEvent to arrive on the channel
	select {
	case evt := <-ch:
		aqe, ok := evt.(AskQuestionEvent)
		if !ok {
			t.Fatalf("expected AskQuestionEvent, got %T", evt)
		}
		if aqe.Question != "Continue?" {
			t.Fatalf("expected Question='Continue?', got %q", aqe.Question)
		}
		if len(aqe.Options) != 2 || aqe.Options[0] != "Yes" || aqe.Options[1] != "No" {
			t.Fatalf("expected Options=['Yes','No'], got %v", aqe.Options)
		}

		// Simulate user reply
		aqe.Reply <- "Yes"
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for AskQuestionEvent")
	}

	// Verify the answer was received
	select {
	case answer := <-answerCh:
		if answer != "Yes" {
			t.Fatalf("expected answer='Yes', got %q", answer)
		}
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for answer")
	}
}

func TestTUIBridge_MultipleEvents(t *testing.T) {
	ch := make(chan AgentEvent, 8)
	b := NewTUIBridge(ch)

	// Send multiple events rapidly
	b.OnStreamStart()
	b.OnContent("hello")
	b.OnContent(" world")
	b.OnComplete()

	// Verify order and types
	expected := []struct {
		eventType string
	}{
		{"StreamStartEvent"},
		{"ContentEvent"},
		{"ContentEvent"},
		{"CompleteEvent"},
	}

	for i, exp := range expected {
		select {
		case evt := <-ch:
			switch exp.eventType {
			case "StreamStartEvent":
				if _, ok := evt.(StreamStartEvent); !ok {
					t.Fatalf("event %d: expected StreamStartEvent, got %T", i, evt)
				}
			case "ContentEvent":
				if _, ok := evt.(ContentEvent); !ok {
					t.Fatalf("event %d: expected ContentEvent, got %T", i, evt)
				}
			case "CompleteEvent":
				if _, ok := evt.(CompleteEvent); !ok {
					t.Fatalf("event %d: expected CompleteEvent, got %T", i, evt)
				}
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for event %d (%s)", i, exp.eventType)
		}
	}
}