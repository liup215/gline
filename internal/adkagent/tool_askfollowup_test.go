package adkagent

import (
	"strings"
	"testing"

	glineagent "github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/tools"
)

// fakeAsker records the question routed through the StreamCallback and
// returns a canned answer.
type fakeAsker struct {
	gotQuestion string
	gotOptions  []string
	answer      string
}

func (f *fakeAsker) AskFollowupQuestion(question string, options []string) (string, error) {
	f.gotQuestion = question
	f.gotOptions = options
	return f.answer, nil
}

// Satisfy the full StreamCallback interface; the routing only calls
// AskFollowupQuestion.
func (f *fakeAsker) OnContent(delta string)                          {}
func (f *fakeAsker) OnReasoning(delta string)                        {}
func (f *fakeAsker) OnStreamStart()                                  {}
func (f *fakeAsker) OnStreamEnd()                                    {}
func (f *fakeAsker) OnToolCallStart(tc glineagent.ToolCall)              {}
func (f *fakeAsker) OnToolCallComplete(tc glineagent.ToolCall, r string) {}
func (f *fakeAsker) OnError(err error)                               {}
func (f *fakeAsker) OnComplete()                                     {}
func (f *fakeAsker) OnTaskCreated(taskID string)                     {}

// findAskTool builds the default registry's ADK tools and returns the
// ask_followup_question adapter.
func findAskTool(t *testing.T, a *Agent) *adkTool {
	t.Helper()
	adkTools, err := a.buildTools()
	if err != nil {
		t.Fatalf("buildTools: %v", err)
	}
	for _, at := range adkTools {
		if gt, ok := at.(*adkTool); ok && gt.Name() == "ask_followup_question" {
			return gt
		}
	}
	t.Fatal("ask_followup_question not adapted")
	return nil
}

// TestAskFollowupRoutesThroughStreamCallback verifies that ask_followup_question
// is executed via the active run's StreamCallback (TUI option picker / GUI
// dialog / CLI prompt) instead of the tool's os.Stdin fallback, which
// competes with Bubbletea for stdin and corrupts the TUI input area.
// Regression: SetHandler was never wired in the ADK path.
func TestAskFollowupRoutesThroughStreamCallback(t *testing.T) {
	registry := tools.InitDefaultRegistry(nil, nil)
	a := &Agent{opts: Options{Tools: registry}}
	fake := &fakeAsker{answer: "option two"}
	a.cb = fake

	askTool := findAskTool(t, a)
	out, err := askTool.Run(nil, map[string]any{
		"question": "Which option?",
		"options":  []any{"one", "two", "three"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if fake.gotQuestion != "Which option?" {
		t.Errorf("callback question = %q, want %q", fake.gotQuestion, "Which option?")
	}
	if len(fake.gotOptions) != 3 {
		t.Errorf("callback options = %v, want 3 entries", fake.gotOptions)
	}
	result, _ := out["result"].(string)
	if !strings.Contains(result, "User answered: option two") {
		t.Errorf("result = %q, want it to contain the callback answer", result)
	}
}
