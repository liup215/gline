package adkagent

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/genai"
)

// TestStuckRecoveryResumesTurn feeds a first turn that trips the identical-call
// detector and verifies the run recovers instead of failing: the model gets the
// recovery prompt as a fresh user turn and the second scripted turn completes.
func TestStuckRecoveryResumesTurn(t *testing.T) {
	// Ten identical read calls with identical results trip the detector.
	calls := make([]*genai.FunctionCall, 0, maxRepeatToolCalls)
	for i := 0; i < maxRepeatToolCalls; i++ {
		calls = append(calls, &genai.FunctionCall{
			ID: "s" + string(rune('a'+i)), Name: "read",
			Args: map[string]any{"path": "same.txt"},
		})
	}
	fm := &fakeModel{turns: []fakeTurn{
		{final: modelWithCalls("", calls...)},
		{final: modelText("Recovered: using a different approach.")},
	}}

	a, err := NewWithModel(context.Background(), Options{
		Model: "fake", Tools: testRegistry(t, ""),
	}, fm)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	cb := newRecordingCallback()
	if _, err := a.RunWithCallback(context.Background(), "read same.txt", cb); err != nil {
		t.Fatalf("RunWithCallback should recover from stuck loop, got: %v", err)
	}
	if fm.calls != 2 {
		t.Fatalf("expected exactly 2 model calls (stuck + recovery), got %d", fm.calls)
	}
	if len(cb.errs) != 0 {
		t.Fatalf("successful recovery must not surface errors, got %v", cb.errs)
	}
	if got := cb.content.String(); !strings.Contains(got, "Recovered") {
		t.Fatalf("final content missing recovery text, got %q", got)
	}

	// The recovery turn's request must end with the recovery instruction.
	var recoveryUser string
	for _, c := range fm.reqs[1].Contents {
		if c.Role != "user" {
			continue
		}
		for _, p := range c.Parts {
			if p.Text != "" {
				recoveryUser = p.Text
			}
		}
	}
	if !strings.Contains(recoveryUser, "stopped automatically") {
		t.Fatalf("recovery prompt missing from second request, last user text: %q", recoveryUser)
	}
}

// TestStuckRecoveryGivesUp verifies that after maxStuckRecoveries failed
// attempts the run ends with an error instead of looping forever.
func TestStuckRecoveryGivesUp(t *testing.T) {
	// Every turn repeats the same degenerate read call.
	var turns []fakeTurn
	for i := 0; i < maxStuckRecoveries+1; i++ {
		calls := make([]*genai.FunctionCall, 0, maxRepeatToolCalls)
		for j := 0; j < maxRepeatToolCalls; j++ {
			calls = append(calls, &genai.FunctionCall{
				ID:   "t" + string(rune('a'+i)) + string(rune('a'+j)),
				Name: "read", Args: map[string]any{"path": "same.txt"},
			})
		}
		turns = append(turns, fakeTurn{final: modelWithCalls("", calls...)})
	}
	fm := &fakeModel{turns: turns}

	a, err := NewWithModel(context.Background(), Options{
		Model: "fake", Tools: testRegistry(t, ""),
	}, fm)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	cb := newRecordingCallback()
	_, runErr := a.RunWithCallback(context.Background(), "read same.txt", cb)
	if runErr == nil {
		t.Fatal("expected an error after exhausting recovery attempts")
	}
	if !strings.Contains(runErr.Error(), "gave up") {
		t.Fatalf("error should mention giving up, got: %v", runErr)
	}
	if fm.calls != maxStuckRecoveries+1 {
		t.Fatalf("expected %d model calls, got %d", maxStuckRecoveries+1, fm.calls)
	}
	if len(cb.errs) == 0 {
		t.Fatal("final failure must be surfaced through cb.OnError")
	}
}
