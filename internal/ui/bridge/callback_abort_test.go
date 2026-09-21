package bridge

import (
	"context"
	"testing"
	"time"
)

// pendingCount reads the bridge's pending-queue length under its mutex so
// tests can poll without racing with AskFollowupQuestion registrations.
func pendingCount(b *TUIBridge) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending)
}

// TestAbortPendingQuestionsUnblocksAsker verifies that a tool goroutine
// blocked in AskFollowupQuestion returns context.Canceled when the agent
// aborts — this is what lets cancelled runs fully unwind.
func TestAbortPendingQuestionsUnblocksAsker(t *testing.T) {
	eventCh := make(chan AgentEvent, 16)
	b := NewTUIBridge(eventCh)
	// Drain events so the send never blocks.
	go func() {
		for range eventCh {
		}
	}()

	done := make(chan error, 1)
	go func() {
		_, err := b.AskFollowupQuestion("Approve run?", []string{"Yes", "No"})
		done <- err
	}()

	// Wait for the question to be registered, then abort.
	deadline := time.After(2 * time.Second)
	for pendingCount(b) == 0 {
		select {
		case <-deadline:
			t.Fatalf("question never registered")
		case <-time.After(5 * time.Millisecond):
		}
	}
	b.AbortPendingQuestions()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("asker not unblocked by AbortPendingQuestions")
	}
	if n := pendingCount(b); n != 0 {
		t.Fatalf("pending queue should be empty after abort, got %d", n)
	}
}

// TestAbortPendingQuestionsIdempotentAfterUIClose covers the double-close
// hazard: the TUI (Esc) closes the reply channel directly, then the agent
// abort calls AbortPendingQuestions on the same channel. The second close
// must be a no-op instead of panicking.
func TestAbortPendingQuestionsIdempotentAfterUIClose(t *testing.T) {
	eventCh := make(chan AgentEvent, 16)
	b := NewTUIBridge(eventCh)
	go func() {
		for range eventCh {
		}
	}()

	done := make(chan error, 1)
	go func() {
		_, err := b.AskFollowupQuestion("Approve run?", []string{"Yes", "No"})
		done <- err
	}()

	deadline := time.After(2 * time.Second)
	for pendingCount(b) == 0 {
		select {
		case <-deadline:
			t.Fatalf("question never registered")
		case <-time.After(5 * time.Millisecond):
		}
	}

	// UI aborts the question (Esc path) through the close-once guard.
	b.pending[0].Abort()

	// Agent abort then runs — must not panic on the already-closed channel.
	b.AbortPendingQuestions()
	b.AbortPendingQuestions() // and a second time for good measure

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("asker not unblocked")
	}
}

// TestParallelAskFollowupQuestionFIFO verifies two concurrent askers each get
// their own question event and receive the answer addressed to them.
func TestParallelAskFollowupQuestionFIFO(t *testing.T) {
	eventCh := make(chan AgentEvent, 16)
	b := NewTUIBridge(eventCh)

	resA := make(chan error, 1)
	resB := make(chan error, 1)
	ansA := make(chan string, 1)
	ansB := make(chan string, 1)
	go func() {
		a, err := b.AskFollowupQuestion("Approve A?", []string{"Yes", "No"})
		resA <- err
		ansA <- a
	}()
	go func() {
		a, err := b.AskFollowupQuestion("Approve B?", []string{"Yes", "No"})
		resB <- err
		ansB <- a
	}()

	// Collect the two question events.
	var replyA, replyB *PendingAsk
	deadline := time.After(2 * time.Second)
	for replyA == nil || replyB == nil {
		select {
		case evt := <-eventCh:
			if q, ok := evt.(AskQuestionEvent); ok {
				if q.Question == "Approve A?" {
					replyA = q.Reply
				} else if q.Question == "Approve B?" {
					replyB = q.Reply
				}
			}
		case <-deadline:
			t.Fatalf("did not receive both question events")
		}
	}
	if replyA == nil || replyB == nil || replyA == replyB {
		t.Fatalf("each asker must have its own reply channel")
	}

	// Answer both; each asker must get its own answer.
	replyA.Ask("yes-a")
	replyB.Ask("no-b")
	for i := 0; i < 2; i++ {
		select {
		case err := <-resA:
			if err != nil {
				t.Fatalf("asker A: %v", err)
			}
			if <-ansA != "yes-a" {
				t.Fatalf("asker A got wrong answer")
			}
		case err := <-resB:
			if err != nil {
				t.Fatalf("asker B: %v", err)
			}
			if <-ansB != "no-b" {
				t.Fatalf("asker B got wrong answer")
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("askers not unblocked")
		}
	}
	if n := pendingCount(b); n != 0 {
		t.Fatalf("pending queue should be empty after answers, got %d", n)
	}
}
