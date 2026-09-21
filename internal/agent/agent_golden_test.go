package agent

// Golden tests: pin down the CURRENT behavior of the hand-written agent loop
// before the ADK refactor (docs/agent-loop-refactor-plan.md). After the loop is
// replaced, these invariants must still hold — they are the acceptance criteria.
//
// The most important one is TestGoldenNoOrphanToolResults: every tool result
// message in the conversation must reference a tool_call carried by an earlier
// assistant message. Orphan tool results are the bug class that motivated the
// refactor.

import (
	"context"
	"strings"
	"testing"

	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
)

// scriptedTurn is one model turn: the chunks the fake provider emits, in order.
type scriptedTurn struct {
	chunks []StreamChunk
}

// scriptedProvider replays scripted turns in order. Extra calls beyond the
// script return an error so tests fail loudly instead of looping.
type scriptedProvider struct {
	turns []scriptedTurn
	calls int
}

func (p *scriptedProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	return nil, context.DeadlineExceeded // not used by the streaming loop
}

func (p *scriptedProvider) CreateMessageStream(ctx context.Context, req *MessageRequest) (<-chan StreamChunk, error) {
	i := p.calls
	p.calls++
	if i >= len(p.turns) {
		ch := make(chan StreamChunk, 1)
		ch <- StreamChunk{Error: context.DeadlineExceeded, Done: true}
		close(ch)
		return ch, nil
	}
	ch := make(chan StreamChunk, len(p.turns[i].chunks)+1)
	for _, c := range p.turns[i].chunks {
		ch <- c
	}
	close(ch)
	return ch, nil
}

func (p *scriptedProvider) SupportsTools() bool          { return true }
func (p *scriptedProvider) GetModel() string             { return "golden-model" }
func (p *scriptedProvider) GetProviderName() string      { return "golden-provider" }

// orderedCallback records the exact callback event sequence.
type orderedCallback struct {
	events []string
}

func (c *orderedCallback) record(s string) { c.events = append(c.events, s) }

func (c *orderedCallback) OnContent(delta string)      { c.record("content:" + delta) }
func (c *orderedCallback) OnReasoning(delta string)    { c.record("reasoning:" + delta) }
func (c *orderedCallback) OnStreamStart()              { c.record("stream_start") }
func (c *orderedCallback) OnStreamEnd()                { c.record("stream_end") }
func (c *orderedCallback) OnToolCallStart(tc ToolCall) { c.record("tool_start:" + tc.Name) }
func (c *orderedCallback) OnToolCallComplete(tc ToolCall, result string) {
	c.record("tool_done:" + tc.Name)
}
func (c *orderedCallback) AskFollowupQuestion(q string, opts []string) (string, error) {
	c.record("followup:" + q)
	if len(opts) > 0 {
		return opts[0], nil
	}
	return "", nil
}
func (c *orderedCallback) OnError(err error)      { c.record("error:" + err.Error()) }
func (c *orderedCallback) OnComplete()            { c.record("complete") }
func (c *orderedCallback) OnTaskCreated(id string) { c.record("task_created:" + id) }

// threeTurnScript models the canonical flow: text+read → text+run → completion.
func threeTurnScript() []scriptedTurn {
	return []scriptedTurn{
		{
			chunks: []StreamChunk{
				{Content: "Let me look at the project."},
				{ToolCall: &ToolCall{ID: "call_read_1", Name: "read", Input: `{"path":"README.md"}`}},
				{FinishReason: "tool_calls", Done: true},
			},
		},
		{
			chunks: []StreamChunk{
				{Content: "Now checking the go version."},
				{ToolCall: &ToolCall{ID: "call_run_1", Name: "run", Input: `{"command":"go version"}`}},
				{FinishReason: "tool_calls", Done: true},
			},
		},
		{
			chunks: []StreamChunk{
				{Content: "Everything checks out."},
				{ToolCall: &ToolCall{ID: "call_done_1", Name: "attempt_completion", Input: `{"result":"All good."}`}},
				{FinishReason: "tool_calls", Done: true},
			},
		},
	}
}

func newGoldenAgent(t *testing.T, p Provider, cb StreamCallback) *BaseAgent {
	t.Helper()
	a, err := New(Options{
		Provider:     p,
		ToolRegistry: tools.InitDefaultRegistry(nil, nil),
		Mode:         ModeAct,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := a.RunWithCallback(context.Background(), "do a golden run", cb); err != nil {
		t.Fatalf("RunWithCallback() error = %v", err)
	}
	return a
}

// assertNoOrphanToolResults is the structural invariant the refactor must
// preserve: every tool result must trace back to a tool call in some assistant
// message that appears BEFORE the result in the conversation.
func assertNoOrphanToolResults(t *testing.T, msgs []types.Message) {
	t.Helper()
	seen := map[string]bool{}
	for i, m := range msgs {
		switch m.Role {
		case types.RoleAssistant:
			for _, tc := range m.ToolCalls {
				seen[tc.ID] = true
			}
		case types.RoleTool:
			if m.ToolCallID == "" {
				t.Fatalf("message %d: tool result has empty ToolCallID", i)
			}
			if !seen[m.ToolCallID] {
				t.Fatalf("message %d: ORPHAN tool result for call %q — no earlier assistant message carries this tool call", i, m.ToolCallID)
			}
		}
	}
}

// TestGoldenNoOrphanToolResults runs a three-turn conversation and asserts the
// no-orphan invariant plus completion state.
func TestGoldenNoOrphanToolResults(t *testing.T) {
	p := &scriptedProvider{turns: threeTurnScript()}
	a := newGoldenAgent(t, p, &orderedCallback{})

	msgs := a.GetConversation().GetMessages()
	assertNoOrphanToolResults(t, msgs)

	if !a.GetConversation().IsComplete() {
		t.Fatal("expected conversation to be complete after attempt_completion")
	}
	if p.calls != 3 {
		t.Fatalf("expected 3 provider calls, got %d", p.calls)
	}
}

// TestGoldenMessageOrdering pins the canonical message sequence for a
// text+tool / text+tool / completion run.
func TestGoldenMessageOrdering(t *testing.T) {
	p := &scriptedProvider{turns: threeTurnScript()}
	a := newGoldenAgent(t, p, &orderedCallback{})

	msgs := a.GetConversation().GetMessages()
	assertNoOrphanToolResults(t, msgs)

	type shape struct {
		role     types.Role
		toolName string
		text     string
	}
	var got []shape
	for _, m := range msgs {
		s := shape{role: m.Role, text: m.Content}
		if m.Role == types.RoleAssistant && len(m.ToolCalls) > 0 {
			s.toolName = m.ToolCalls[0].Name
		}
		got = append(got, s)
	}

	want := []shape{
		{role: types.RoleUser, text: "do a golden run"},
		{role: types.RoleAssistant, toolName: "read", text: "Let me look at the project."},
		{role: types.RoleTool},
		{role: types.RoleAssistant, toolName: "run", text: "Now checking the go version."},
		{role: types.RoleTool},
		{role: types.RoleAssistant, toolName: "attempt_completion", text: "Everything checks out."},
		// CURRENT BEHAVIOR: attempt_completion's own result is also recorded as a
		// tool message (with the banner). The ADK refactor removes
		// attempt_completion entirely — this row pins the status quo.
		{role: types.RoleTool},
	}
	if len(got) != len(want) {
		for i, g := range got {
			t.Logf("msg %d: role=%s tool=%s text=%q", i, g.role, g.toolName, g.text)
		}
		t.Fatalf("expected %d messages, got %d", len(want), len(got))
	}
	for i, w := range want {
		if got[i].role != w.role {
			t.Errorf("msg %d role = %q, want %q", i, got[i].role, w.role)
		}
		if w.toolName != "" && got[i].toolName != w.toolName {
			t.Errorf("msg %d tool call = %q, want %q", i, got[i].toolName, w.toolName)
		}
		if w.text != "" && !strings.Contains(got[i].text, w.text) {
			t.Errorf("msg %d text = %q, want contains %q", i, got[i].text, w.text)
		}
	}
}

// TestGoldenCallbackOrdering pins the callback event sequence the TUI bridge
// depends on: stream start/end brackets, tool start before tool done, complete
// exactly once at the very end.
func TestGoldenCallbackOrdering(t *testing.T) {
	p := &scriptedProvider{turns: threeTurnScript()}
	cb := &orderedCallback{}
	newGoldenAgent(t, p, cb)

	ev := cb.events
	// Brackets: 3 turns → 3 stream_start, 3 stream_end.
	starts, ends := 0, 0
	for _, e := range ev {
		switch e {
		case "stream_start":
			starts++
		case "stream_end":
			ends++
		}
	}
	if starts != 3 || ends != 3 {
		t.Fatalf("expected 3 stream_start/stream_end pairs, got starts=%d ends=%d", starts, ends)
	}

	// Tool lifecycle ordering within each turn.
	last := map[string]int{}
	for i, e := range ev {
		switch {
		case strings.HasPrefix(e, "tool_start:"):
			last[strings.TrimPrefix(e, "tool_start:")] = i
		case strings.HasPrefix(e, "tool_done:"):
			name := strings.TrimPrefix(e, "tool_done:")
			start, ok := last[name]
			if !ok {
				t.Fatalf("tool_done:%s at %d has no preceding tool_start", name, i)
			}
			if i < start {
				t.Fatalf("tool_done:%s at %d precedes its tool_start at %d", name, i, start)
			}
		}
	}

	// complete is exactly once and last.
	if n := len(ev); n == 0 || ev[n-1] != "complete" {
		t.Fatalf("expected final event to be complete, got %v", ev[max(0, n-3):])
	}
	completes := 0
	for _, e := range ev {
		if e == "complete" {
			completes++
		}
	}
	if completes != 1 {
		t.Fatalf("expected exactly one complete event, got %d", completes)
	}
}

// TestGoldenPartialToolCallFragments pins how the loop treats partial tool-call
// chunks: providers accumulate fragments and emit complete calls, so partial
// chunks must be ignored and the complete call processed exactly once.
func TestGoldenPartialToolCallFragments(t *testing.T) {
	p := &scriptedProvider{turns: []scriptedTurn{
		{
			chunks: []StreamChunk{
				{Content: "Reading..."},
				// Fragmented partial chunks must be skipped.
				{ToolCall: &ToolCall{ID: "call_p_1", Name: "read", Input: `{"pa`}, IsPartial: true},
				{ToolCall: &ToolCall{ID: "call_p_1", Name: "read", Input: `th":"`}, IsPartial: true},
				{ToolCall: &ToolCall{ID: "call_p_1", Name: "read", Input: `README.md"}`}, IsPartial: true},
				// Provider then emits the accumulated complete call.
				{ToolCall: &ToolCall{ID: "call_p_1", Name: "read", Input: `{"path":"README.md"}`}},
				{FinishReason: "tool_calls", Done: true},
			},
		},
		{
			chunks: []StreamChunk{
				{Content: "Done reading."},
				{ToolCall: &ToolCall{ID: "call_p_2", Name: "attempt_completion", Input: `{"result":"ok"}`}},
				{FinishReason: "tool_calls", Done: true},
			},
		},
	}}
	cb := &orderedCallback{}
	a := newGoldenAgent(t, p, cb)

	assertNoOrphanToolResults(t, a.GetConversation().GetMessages())

	starts, dones := 0, 0
	for _, e := range cb.events {
		switch e {
		case "tool_start:read":
			starts++
		case "tool_done:read":
			dones++
		}
	}
	if starts != 1 || dones != 1 {
		t.Fatalf("partial fragments leaked into tool lifecycle: starts=%d dones=%d (want 1/1)", starts, dones)
	}
}

// TestGoldenReasoningStreamPinned pins that reasoning content is delivered via
// OnReasoning only — never mixed into OnContent — and stored on the assistant
// message, not the content field.
func TestGoldenReasoningStreamPinned(t *testing.T) {
	p := &scriptedProvider{turns: []scriptedTurn{
		{
			chunks: []StreamChunk{
				{ReasoningContent: "thinking hard..."},
				{Content: "The answer is 42."},
				{ToolCall: &ToolCall{ID: "call_r_1", Name: "attempt_completion", Input: `{"result":"42"}`}},
				{FinishReason: "tool_calls", Done: true},
			},
		},
	}}
	cb := &orderedCallback{}
	a := newGoldenAgent(t, p, cb)

	var content, reasoning strings.Builder
	for _, e := range cb.events {
		if strings.HasPrefix(e, "content:") {
			content.WriteString(strings.TrimPrefix(e, "content:"))
		}
		if strings.HasPrefix(e, "reasoning:") {
			reasoning.WriteString(strings.TrimPrefix(e, "reasoning:"))
		}
	}
	if content.String() != "The answer is 42." {
		t.Fatalf("content = %q, want %q", content.String(), "The answer is 42.")
	}
	if reasoning.String() != "thinking hard..." {
		t.Fatalf("reasoning = %q, want %q", reasoning.String(), "thinking hard...")
	}

	var asst *types.Message
	for i := range a.conversation.Messages {
		if a.conversation.Messages[i].Role == types.RoleAssistant {
			asst = &a.conversation.Messages[i]
			break
		}
	}
	if asst == nil {
		t.Fatal("no assistant message")
	}
	if asst.ReasoningContent != "thinking hard..." {
		t.Fatalf("assistant ReasoningContent = %q", asst.ReasoningContent)
	}
	if asst.Content != "The answer is 42." {
		t.Fatalf("assistant Content = %q", asst.Content)
	}
}
