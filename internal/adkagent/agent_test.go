package adkagent

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"

	glineagent "github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
)

// fakeModel replays scripted turns through the real ADK runner.
type fakeTurn struct {
	deltas []string       // partial text chunks (streamed)
	final  *genai.Content // the aggregate turn content (may hold FunctionCalls)
}

type fakeModel struct {
	mu    sync.Mutex
	turns []fakeTurn
	reqs  []*model.LLMRequest
	calls int
}

func (f *fakeModel) Name() string { return "fake" }

func (f *fakeModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		f.mu.Lock()
		i := f.calls
		f.calls++
		f.reqs = append(f.reqs, req)
		turn := f.turns[i]
		f.mu.Unlock()

		if !stream {
			if turn.final != nil {
				yield(&model.LLMResponse{Content: turn.final}, nil)
			}
			return
		}
		for _, d := range turn.deltas {
			resp := &model.LLMResponse{
				Content: &genai.Content{Role: "model", Parts: []*genai.Part{{Text: d}}},
				Partial: true,
			}
			if !yield(resp, nil) {
				return
			}
		}
		if turn.final != nil {
			yield(&model.LLMResponse{Content: turn.final, TurnComplete: true}, nil)
		}
	}
}

// modelText builds a plain-text aggregate content.
func modelText(s string) *genai.Content {
	return &genai.Content{Role: "model", Parts: []*genai.Part{{Text: s}}}
}

// modelWithCalls builds an aggregate content holding text and function calls.
func modelWithCalls(text string, calls ...*genai.FunctionCall) *genai.Content {
	parts := make([]*genai.Part, 0, len(calls)+1)
	if text != "" {
		parts = append(parts, &genai.Part{Text: text})
	}
	for _, c := range calls {
		parts = append(parts, &genai.Part{FunctionCall: c})
	}
	return &genai.Content{Role: "model", Parts: parts}
}

// recordingCallback records every callback invocation in order.
type recordingCallback struct {
	mu      sync.Mutex
	order   []string
	content strings.Builder
	results map[string]string
	answers []string // scripted AskFollowupQuestion replies
	answerI int
	errs    []error
	done    bool
}

func newRecordingCallback(answers ...string) *recordingCallback {
	return &recordingCallback{results: map[string]string{}, answers: answers}
}

func (r *recordingCallback) rec(ev string) {
	r.mu.Lock()
	r.order = append(r.order, ev)
	r.mu.Unlock()
}

func (r *recordingCallback) OnContent(delta string) {
	r.mu.Lock()
	r.content.WriteString(delta)
	r.mu.Unlock()
	r.rec("content")
}
func (r *recordingCallback) OnReasoning(delta string) { r.rec("reasoning") }
func (r *recordingCallback) OnStreamStart()           { r.rec("stream_start") }
func (r *recordingCallback) OnStreamEnd()             { r.rec("stream_end") }
func (r *recordingCallback) OnError(err error) {
	r.mu.Lock()
	r.errs = append(r.errs, err)
	r.mu.Unlock()
	r.rec("error")
}
func (r *recordingCallback) OnComplete() {
	r.mu.Lock()
	r.done = true
	r.mu.Unlock()
	r.rec("complete")
}
func (r *recordingCallback) OnTaskCreated(id string) {}

func (r *recordingCallback) OnToolCallStart(tc glineagent.ToolCall) {
	r.rec("tool_start:" + tc.Name)
}

func (r *recordingCallback) OnToolCallComplete(tc glineagent.ToolCall, result string) {
	r.mu.Lock()
	r.results[tc.Name] = result
	r.mu.Unlock()
	r.rec("tool_complete:" + tc.Name)
}

func (r *recordingCallback) AskFollowupQuestion(question string, options []string) (string, error) {
	r.mu.Lock()
	i := r.answerI
	r.answerI++
	r.mu.Unlock()
	if i >= len(r.answers) {
		return "Yes", nil
	}
	return r.answers[i], nil
}

// eventsSnapshot flattens the session event log for structural assertions.
func eventsSnapshot(t *testing.T, svc session.Service, sessID string) []*session.Event {
	t.Helper()
	resp, err := svc.Get(context.Background(), &session.GetRequest{
		AppName: AppName, UserID: UserID, SessionID: sessID,
	})
	if err != nil {
		t.Fatalf("Get session: %v", err)
	}
	out := make([]*session.Event, 0, resp.Session.Events().Len())
	for ev := range resp.Session.Events().All() {
		out = append(out, ev)
	}
	return out
}

// testRegistry builds a minimal registry: read (plan-safe), write and run
// (act-only, confirmation-required).
func testRegistry(t *testing.T, dir string) *tools.Registry {
	t.Helper()
	r := tools.NewRegistry()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("registry register: %v", err)
		}
	}
	must(r.Register(&tools.ToolInfo{Tool: tools.NewReadFileTool(), Category: tools.CategoryFile, AllowedModes: []string{"plan", "act"}}))
	must(r.Register(&tools.ToolInfo{Tool: tools.NewWriteFileTool(), Category: tools.CategoryFile, AllowedModes: []string{"act"}, RequiresConfirmation: true}))
	must(r.Register(&tools.ToolInfo{Tool: tools.NewExecuteCommandTool(), Category: tools.CategoryCommand, AllowedModes: []string{"act"}, RequiresConfirmation: true}))
	return r
}

func TestGoldenNoOrphanToolResults(t *testing.T) {
	fm := &fakeModel{turns: []fakeTurn{
		{deltas: []string{"Let me "}, final: modelWithCalls("Let me check.", &genai.FunctionCall{
			ID: "call-1", Name: "read", Args: map[string]any{"path": "does-not-exist.txt"},
		})},
		{final: modelText("Done. The file does not exist.")},
	}}

	a, err := NewWithModel(context.Background(), Options{
		Model: "fake", Tools: testRegistry(t, ""),
	}, fm)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	cb := newRecordingCallback()
	if _, err := a.RunWithCallback(context.Background(), "check the file", cb); err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}

	// Structural invariant: every FunctionCall in the event log has a
	// matching FunctionResponse.
	events := eventsSnapshot(t, a.sessionSvc, a.SessionID())
	calls, responses := 0, 0
	for _, ev := range events {
		if ev.Content == nil {
			continue
		}
		for _, part := range ev.Content.Parts {
			if part.FunctionCall != nil {
				calls++
			}
			if part.FunctionResponse != nil {
				responses++
			}
		}
	}
	if calls != 1 || responses != 1 {
		t.Fatalf("orphan check: calls=%d responses=%d (want 1/1)", calls, responses)
	}
	if !cb.done {
		t.Fatal("OnComplete not called")
	}
}

func TestGoldenCallbackOrdering(t *testing.T) {
	fm := &fakeModel{turns: []fakeTurn{
		{deltas: []string{"Working", "..."}, final: modelWithCalls("Working...", &genai.FunctionCall{
			ID: "c1", Name: "read", Args: map[string]any{"path": "x.txt"},
		})},
		{final: modelText("All done.")},
	}}
	a, err := NewWithModel(context.Background(), Options{Model: "fake", Tools: testRegistry(t, "")}, fm)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	cb := newRecordingCallback()
	if _, err := a.RunWithCallback(context.Background(), "go", cb); err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}

	want := []string{
		"stream_start", "content", "content", // deltas
		"stream_end", // aggregate closes text slot
		"tool_start:read",
		"tool_complete:read",
		"stream_start", "content", // second turn text
		"stream_end",
		"complete",
	}
	got := cb.order
	if len(got) != len(want) {
		t.Fatalf("event order mismatch:\n got: %v\nwant: %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] = %q, want %q\nfull got: %v", i, got[i], want[i], got)
		}
	}
	if cb.content.String() != "Working...All done." {
		t.Fatalf("content = %q", cb.content.String())
	}
}

func TestPlanModeBlocksWrite(t *testing.T) {
	dir := t.TempDir()
	fm := &fakeModel{turns: []fakeTurn{
		{final: modelWithCalls("", &genai.FunctionCall{
			ID: "c1", Name: "write", Args: map[string]any{"path": "evil.txt", "content": "no"},
		})},
		{final: modelText("Understood, staying read-only.")},
	}}
	a, err := NewWithModel(context.Background(), Options{Model: "fake", Tools: testRegistry(t, ""), Mode: ModePlan, WorkingDir: dir}, fm)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	if a.Mode() != ModePlan {
		t.Fatalf("mode = %q", a.Mode())
	}
	cb := newRecordingCallback()
	if _, err := a.RunWithCallback(context.Background(), "write a file", cb); err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}

	if res := cb.results["write"]; !strings.Contains(res, "Plan mode") {
		t.Fatalf("write result = %q, want Plan-mode block message", res)
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.txt")); !os.IsNotExist(err) {
		t.Fatal("write tool executed in plan mode; file exists")
	}
	// The write FunctionResponse must still be recorded (no orphans).
	events := eventsSnapshot(t, a.sessionSvc, a.SessionID())
	calls, responses := 0, 0
	for _, ev := range events {
		if ev.Content == nil {
			continue
		}
		for _, part := range ev.Content.Parts {
			if part.FunctionCall != nil {
				calls++
			}
			if part.FunctionResponse != nil {
				responses++
			}
		}
	}
	if calls != 1 || responses != 1 {
		t.Fatalf("plan-mode block broke pairing: calls=%d responses=%d", calls, responses)
	}
}

func TestConfirmationAndYolo(t *testing.T) {
	// Non-yolo (SetYolo(false)): run requires confirmation; scripted answer
	// "Yes" approves.
	fm := &fakeModel{turns: []fakeTurn{
		{final: modelWithCalls("", &genai.FunctionCall{
			ID: "c1", Name: "run", Args: map[string]any{"command": "echo hi"},
		})},
		{final: modelText("ran it")},
	}}
	a, err := NewWithModel(context.Background(), Options{Model: "fake", Tools: testRegistry(t, "")}, fm)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	a.SetYolo(false)
	cb := newRecordingCallback("Yes")
	if _, err := a.RunWithCallback(context.Background(), "echo", cb); err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}
	if res := cb.results["run"]; !strings.Contains(res, "hi") {
		t.Fatalf("run result = %q, want command output", res)
	}

	// Declined (SetYolo(false)): the tool does not execute.
	fm2 := &fakeModel{turns: []fakeTurn{
		{final: modelWithCalls("", &genai.FunctionCall{
			ID: "c1", Name: "run", Args: map[string]any{"command": "echo secret"},
		})},
		{final: modelText("ok, skipped")},
	}}
	a2, err := NewWithModel(context.Background(), Options{Model: "fake", Tools: testRegistry(t, "")}, fm2)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	a2.SetYolo(false)
	cb2 := newRecordingCallback("No")
	if _, err := a2.RunWithCallback(context.Background(), "echo", cb2); err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}
	if res := cb2.results["run"]; strings.Contains(res, "secret") {
		t.Fatalf("declined tool still executed: %q", res)
	}

	// Default (yolo on): no question asked; order has no confirmation
	// round-trip.
	fm3 := &fakeModel{turns: []fakeTurn{
		{final: modelWithCalls("", &genai.FunctionCall{
			ID: "c1", Name: "run", Args: map[string]any{"command": "echo yolo"},
		})},
		{final: modelText("done")},
	}}
	a3, err := NewWithModel(context.Background(), Options{Model: "fake", Tools: testRegistry(t, "")}, fm3)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	cb3 := newRecordingCallback() // no scripted answers; if asked, defaults "Yes" — assert via tool result only
	if _, err := a3.RunWithCallback(context.Background(), "echo", cb3); err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}
	if res := cb3.results["run"]; !strings.Contains(res, "yolo") {
		t.Fatalf("yolo run result = %q", res)
	}
}

func TestSessionPersistence(t *testing.T) {
	fm := &fakeModel{turns: []fakeTurn{
		{final: modelText("first reply")},
		{final: modelText("second reply")},
	}}
	a, err := NewWithModel(context.Background(), Options{Model: "fake", Tools: testRegistry(t, "")}, fm)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	cb := newRecordingCallback()
	if _, err := a.RunWithCallback(context.Background(), "turn one", cb); err != nil {
		t.Fatalf("run 1: %v", err)
	}
	if _, err := a.RunWithCallback(context.Background(), "turn two", cb); err != nil {
		t.Fatalf("run 2: %v", err)
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()
	if len(fm.reqs) != 2 {
		t.Fatalf("model called %d times, want 2", len(fm.reqs))
	}
	// Second request must carry the whole prior conversation.
	var texts []string
	for _, c := range fm.reqs[1].Contents {
		for _, p := range c.Parts {
			if p.Text != "" {
				texts = append(texts, p.Text)
			}
		}
	}
	joined := strings.Join(texts, "\n")
	for _, want := range []string{"turn one", "first reply", "turn two"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("second request missing %q; contents: %q", want, joined)
		}
	}
}

func TestSetModeAffectsInstruction(t *testing.T) {
	fm := &fakeModel{turns: []fakeTurn{
		{final: modelText("one")},
		{final: modelText("two")},
	}}
	a, err := NewWithModel(context.Background(), Options{Model: "fake", Tools: testRegistry(t, ""), Mode: ModeAct}, fm)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	cb := newRecordingCallback()
	if _, err := a.RunWithCallback(context.Background(), "hi", cb); err != nil {
		t.Fatalf("run 1: %v", err)
	}
	a.SetMode(ModePlan)
	if _, err := a.RunWithCallback(context.Background(), "again", cb); err != nil {
		t.Fatalf("run 2: %v", err)
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()
	sys1, sys2 := systemInstructionOf(fm.reqs[0]), systemInstructionOf(fm.reqs[1])
	if !strings.Contains(sys1, "Act Mode") || strings.Contains(sys1, "Plan Mode") {
		t.Fatalf("first instruction not act mode: %q", sys1)
	}
	if !strings.Contains(sys2, "Plan Mode") || strings.Contains(sys2, "Act Mode") {
		t.Fatalf("second instruction not plan mode: %q", sys2)
	}
}

func systemInstructionOf(req *model.LLMRequest) string {
	if req.Config == nil {
		return ""
	}
	if req.Config.SystemInstruction == nil {
		return ""
	}
	b, _ := json.Marshal(req.Config.SystemInstruction)
	var texts []string
	var probe map[string]any
	if json.Unmarshal(b, &probe) == nil {
		if parts, ok := probe["parts"].([]any); ok {
			for _, p := range parts {
				if m, ok := p.(map[string]any); ok {
					if t, ok := m["text"].(string); ok {
						texts = append(texts, t)
					}
				}
			}
		}
	}
	return fmt.Sprint(texts)
}

func TestSystemInstructionIncludesSkills(t *testing.T) {
	a := &Agent{opts: Options{Model: "fake"},
		skills: []types.SkillMeta{{Name: "pdf-master", Description: "Handle PDFs"}},
	}
	si := a.SystemInstruction()
	if !strings.Contains(si, "Skills") || !strings.Contains(si, "pdf-master") || !strings.Contains(si, "Handle PDFs") {
		t.Fatalf("skills section missing:\n%s", si)
	}
	// No skills: no section at all.
	a2 := &Agent{opts: Options{Model: "fake"}}
	if strings.Contains(a2.SystemInstruction(), "Skills") {
		t.Fatal("SKILLS section rendered with empty skill list")
	}
}

func TestSystemInstructionIncludesCustomRules(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	rulesDir := filepath.Join(dir, ".gline", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "style.md"), []byte("Always answer in haiku."), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &Agent{opts: Options{Model: "fake"}}
	si := a.SystemInstruction()
	if !strings.Contains(si, "Workspace Rules") || !strings.Contains(si, "Always answer in haiku.") {
		t.Fatalf("custom rules not in system instruction:\n%s", si)
	}

	// Empty workspace: no Workspace Rules section. (Global rules from the
	// real HOME may legitimately appear on a developer machine, so only the
	// workspace section is asserted absent.)
	t.Chdir(t.TempDir())
	a2 := &Agent{opts: Options{Model: "fake"}}
	if strings.Contains(a2.SystemInstruction(), "Workspace Rules") {
		t.Fatalf("Workspace Rules rendered with no workspace rule files:\n%s", a2.SystemInstruction())
	}
}
