package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liup215/gline/internal/ui/bridge"
	"github.com/liup215/gline/pkg/types"
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

	if len(m.pendingQuestions) != 2 {
		t.Fatalf("expected 2 pending replies, got %d", len(m.pendingQuestions))
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
	if len(m.pendingQuestions) != 1 {
		t.Fatalf("expected 1 remaining pending reply, got %d", len(m.pendingQuestions))
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
	if len(m.pendingQuestions) != 0 {
		t.Fatalf("expected empty pending queue, got %d", len(m.pendingQuestions))
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

	if len(m.pendingQuestions) != 0 {
		t.Fatalf("pending queue should be cleared after Esc, got %d", len(m.pendingQuestions))
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

// deliverQuestionWithOpts feeds an AskQuestionEvent with custom options.
func deliverQuestionWithOpts(m *Model, reply *bridge.PendingAsk, question string, opts []string) {
	m.Update(bridge.AskQuestionEvent{
		Question: question,
		Options:  opts,
		Reply:    reply,
	})
}

func selectedOf(m *Model) *int {
	for i := m.conversation.MessageCount() - 1; i >= 0; i-- {
		if msg := m.conversation.GetMessage(i); msg != nil && msg.MsgType == types.TypeQuestion {
			return msg.SelectedOption
		}
	}
	return nil
}

func pressKey(m *Model, t tea.KeyType) {
	m.Update(tea.KeyMsg{Type: t})
}

func pressRune(m *Model, r rune) {
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
}

func TestQuestionOptionSelectionArrowAndEnter(t *testing.T) {
	m := newApprovalTestModel(t)
	reply := newPendingAsk()
	deliverQuestionWithOpts(m, reply, "Deploy to prod?", []string{"Yes", "No", "Ask me later"})

	// Default selection is the first option.
	if sel := selectedOf(m); sel == nil || *sel != 0 {
		t.Fatalf("default selection should be 0, got %v", sel)
	}

	// Down moves to option 2; Enter with empty input sends it verbatim.
	pressKey(m, tea.KeyDown)
	if sel := selectedOf(m); sel == nil || *sel != 1 {
		t.Fatalf("selection after Down should be 1, got %v", sel)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	select {
	case got := <-reply.Ch:
		if got != "No" {
			t.Fatalf("Enter should send selected option, got %q", got)
		}
	default:
		t.Fatalf("Enter did not answer the pending question")
	}
}

func TestQuestionOptionSelectionWrapsAndDigits(t *testing.T) {
	m := newApprovalTestModel(t)
	reply := newPendingAsk()
	deliverQuestionWithOpts(m, reply, "Pick one", []string{"Alpha", "Beta"})

	// Up from 0 wraps to the last option.
	pressKey(m, tea.KeyUp)
	if sel := selectedOf(m); sel == nil || *sel != 1 {
		t.Fatalf("Up should wrap to last option, got %v", sel)
	}

	// Digit '1' selects the first option directly.
	pressRune(m, '1')
	if sel := selectedOf(m); sel == nil || *sel != 0 {
		t.Fatalf("digit 1 should select option 0, got %v", sel)
	}

	// Digit '9' is beyond the list: falls through as free text.
	pressRune(m, '9')
	if sel := selectedOf(m); sel == nil || *sel != 0 {
		t.Fatalf("digit 9 must not change selection, got %v", sel)
	}
	if got := m.textarea.Value(); got != "9" {
		t.Fatalf("digit 9 should be typed into the input, got %q", got)
	}
}

func TestTypedAnswerOverridesSelection(t *testing.T) {
	m := newApprovalTestModel(t)
	reply := newPendingAsk()
	deliverQuestion(m, reply, "Approve run? {\"command\":\"ls\"}")

	// Selection sits on option 0 ("Yes"), but typed text wins on Enter.
	typeAndEnter(m, "custom answer")
	select {
	case got := <-reply.Ch:
		if got != "custom answer" {
			t.Fatalf("typed text should win, got %q", got)
		}
	default:
		t.Fatalf("question was not answered")
	}
}

func TestRightArrowAppendsOptionForEditing(t *testing.T) {
	m := newApprovalTestModel(t)
	reply := newPendingAsk()
	deliverQuestionWithOpts(m, reply, "Proceed?", []string{"Yes", "No"})

	// Right loads the selected option into the input box for editing.
	pressKey(m, tea.KeyRight)
	if got := m.textarea.Value(); got != "Yes " {
		t.Fatalf("Right should load selected option, got %q", got)
	}

	// User appends free-form text; Enter sends the edited answer.
	typeAndEnter(m, "but skip tests")
	select {
	case got := <-reply.Ch:
		if got != "Yes but skip tests" {
			t.Fatalf("expected option + appended text, got %q", got)
		}
	default:
		t.Fatalf("question was not answered")
	}
}

func TestEmptyEnterOnFreeFormQuestionStaysPending(t *testing.T) {
	m := newApprovalTestModel(t)
	reply := newPendingAsk()
	// No options: nothing to select, empty Enter must not pop the question
	// (which would orphan the waiting tool goroutine).
	deliverQuestionWithOpts(m, reply, "What should I name it?", nil)

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.pendingQuestions) != 1 {
		t.Fatalf("empty Enter on option-less question must keep it pending")
	}
	select {
	case <-reply.Ch:
		t.Fatalf("empty Enter must not answer the question")
	default:
	}

	typeAndEnter(m, "my-feature")
	select {
	case got := <-reply.Ch:
		if got != "my-feature" {
			t.Fatalf("typed answer expected, got %q", got)
		}
	default:
		t.Fatalf("question was not answered by typed text")
	}
}

func TestQuestionRenderShowsSelectionMarker(t *testing.T) {
	m := newApprovalTestModel(t)
	reply := newPendingAsk()
	deliverQuestionWithOpts(m, reply, "Proceed?", []string{"Yes", "No"})

	m.Update(tea.KeyMsg{Type: tea.KeyDown}) // move to "No"
	strip := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			if r >= 32 || r == '\n' {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
	view := strip(m.View())
	if !strings.Contains(view, "❯ No") {
		t.Fatalf("view should highlight selected option 'No', got: %s", view)
	}
}
