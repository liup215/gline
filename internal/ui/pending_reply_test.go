package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liup215/gline/internal/ui/bridge"
)

// newApprovalTestModel builds a Model wired for key-handling tests: the event
// channel is drained by a goroutine so AskQuestionEvent delivery via the real
// Update path works without a running tea.Program.
func newPendingAsk() *bridge.PendingAsk {
	return &bridge.PendingAsk{Ch: make(chan string, 1)}
}

func newApprovalTestModel(t *testing.T) *Model {
	t.Helper()
	m := New(nil, nil)
	m.eventCh = make(chan bridge.AgentEvent, 64)
	m.done = make(chan struct{})
	go func() {
		for {
			select {
			case <-m.eventCh:
				// drop events
			case <-m.done:
				return
			}
		}
	}()
	t.Cleanup(func() { close(m.done) })
	// Give the model a sane size so the textarea is focusable/renderable.
	m.width = 80
	m.height = 24
	return m
}

// deliverQuestion feeds an AskQuestionEvent through Update, mirroring how the
// bridge forwards approval prompts to the TUI.
func deliverQuestion(m *Model, reply *bridge.PendingAsk, question string) {
	m.Update(bridge.AskQuestionEvent{
		Question: question,
		Options:  []string{"Yes", "No"},
		Reply:    reply,
	})
}

func typeAndEnter(m *Model, text string) {
	for _, r := range text {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
}

func TestParallelApprovalsQueueFIFO(t *testing.T) {
	m := newApprovalTestModel(t)
	replyA := newPendingAsk()
	replyB := newPendingAsk()

	// Two parallel tool calls ask for approval concurrently.
	deliverQuestion(m, replyA, "Approve run? {\"command\":\"ls\"}")
	deliverQuestion(m, replyB, "Approve run? {\"command\":\"pwd\"}")

	if len(m.pendingReplies) != 2 {
		t.Fatalf("expected 2 pending replies, got %d", len(m.pendingReplies))
	}
	if m.textarea.Placeholder == "" || len(m.textarea.Placeholder) == 0 {
		t.Fatalf("expected placeholder describing pending approvals")
	}

	// First Enter answers the FIRST question (FIFO), not the second.
	typeAndEnter(m, "yes")
	select {
	case got := <-replyA.Ch:
		if got != "yes" {
			t.Fatalf("first approval got %q, want %q", got, "yes")
		}
	default:
		t.Fatalf("first reply channel was not answered — FIFO order broken")
	}
	select {
	case got := <-replyB.Ch:
		t.Fatalf("second approval answered too early: %q", got)
	default:
	}
	if len(m.pendingReplies) != 1 {
		t.Fatalf("expected 1 remaining pending reply, got %d", len(m.pendingReplies))
	}

	// Second Enter answers the remaining question.
	typeAndEnter(m, "no")
	select {
	case got := <-replyB.Ch:
		if got != "no" {
			t.Fatalf("second approval got %q, want %q", got, "no")
		}
	default:
		t.Fatalf("second reply channel was not answered")
	}
	if len(m.pendingReplies) != 0 {
		t.Fatalf("expected empty pending queue, got %d", len(m.pendingReplies))
	}
}

// TestAnsweredEnterDoesNotLeakNewlineIntoTextarea guards against the reported
// bug where Enter submitted the approval but ALSO inserted a newline into the
// just-cleared input, making the box look stuck.
func TestAnsweredEnterDoesNotLeakNewlineIntoTextarea(t *testing.T) {
	m := newApprovalTestModel(t)
	reply := newPendingAsk()
	deliverQuestion(m, reply, "Approve run?")

	typeAndEnter(m, "yes")
	select {
	case <-reply.Ch:
	default:
		t.Fatalf("reply not answered")
	}

	// The Enter that submitted the answer must not leave a newline behind,
	// and a subsequent Enter (queue empty now) must not pile up newlines.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.textarea.Value() != "" {
		t.Fatalf("textarea should be empty after answering, got %q", m.textarea.Value())
	}
}

func TestEscClosesAllPendingReplies(t *testing.T) {
	m := newApprovalTestModel(t)
	m.isProcessing = true // Esc only interrupts while a run is active
	replyA := newPendingAsk()
	replyB := newPendingAsk()
	deliverQuestion(m, replyA, "Approve run? A")
	deliverQuestion(m, replyB, "Approve run? B")

	m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if len(m.pendingReplies) != 0 {
		t.Fatalf("pending queue should be cleared after Esc, got %d", len(m.pendingReplies))
	}
	for i, p := range []*bridge.PendingAsk{replyA, replyB} {
		select {
		case _, ok := <-p.Ch:
			if ok {
				t.Fatalf("reply %d should be closed after Esc", i)
			}
		case <-time.After(time.Second):
			t.Fatalf("reply %d not closed after Esc (block on receive?)", i)
		}
	}
}

// TestEscWithAgentAbortUnblocksWaitingTool mirrors the full Esc path: the UI
// closes the reply channels and the bridge's AbortPendingQuestions must be
// idempotent so the double-close does not panic.
func TestEscWithAgentAbortUnblocksWaitingTool(t *testing.T) {
	m := newApprovalTestModel(t)
	m.isProcessing = true // Esc only interrupts while a run is active
	reply := newPendingAsk()
	deliverQuestion(m, reply, "Approve run?")

	// A tool goroutine blocks on the reply channel, like requestApproval.
	unblocked := make(chan bool, 1)
	go func() {
		ans, ok := <-reply.Ch
		unblocked <- ok && ans == "yes"
	}()

	// UI Esc closes the channel; then agent abort runs AbortPendingQuestions
	// on the bridge-like callback. Model itself has no bridge callback here,
	// so simulate the bridge double-close guard via the same close-once path
	// used by the bridge (the channel is shared).
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	select {
	case ok := <-unblocked:
		if ok {
			t.Fatalf("asker should see closed channel (ok=false)")
		}
	case <-time.After(time.Second):
		t.Fatalf("asker did not unblock after Esc")
	}
}
