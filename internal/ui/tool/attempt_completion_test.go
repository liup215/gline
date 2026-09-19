package tool

import (
	"strings"
	"testing"

	"github.com/liup215/gline/pkg/types"
)

// sampleCompletionOutput mirrors the formatted string returned by
// AttemptCompletionTool.Execute (banner + separators + result).
const sampleCompletionOutput = "\n✅ Task Completed\n==================================================\n\nFixed the TUI tool result rendering.\n\n- Removed the banner\n- Render markdown\n\n==================================================\n"

func TestAttemptCompletionRendererSkipsStart(t *testing.T) {
	r := &AttemptCompletionRenderer{}
	res := r.Render(RenderRequest{Phase: types.ToolPhaseStart, Input: `{"result":"x"}`})
	if !res.Skip {
		t.Error("expected Start phase to be skipped")
	}
}

func TestAttemptCompletionRendererStripsBanner(t *testing.T) {
	r := &AttemptCompletionRenderer{}
	res := r.Render(RenderRequest{Phase: types.ToolPhaseComplete, Input: sampleCompletionOutput, Status: "completed"})

	if res.Skip {
		t.Fatal("expected Complete phase to produce a message")
	}
	if res.Role != types.RoleSystem {
		t.Errorf("expected RoleSystem, got %v", res.Role)
	}
	if res.Strategy != types.StrategyMarkdown {
		t.Errorf("expected StrategyMarkdown, got %v", res.Strategy)
	}

	content := res.Content
	// Icon prefix should be present once
	if !strings.HasPrefix(content, "✅ ") {
		t.Errorf("expected icon prefix, got: %q", content)
	}
	// Banner decorations must be removed
	if strings.Contains(content, "Task Completed") {
		t.Errorf("banner header should be stripped, got: %q", content)
	}
	if strings.Contains(content, "====") {
		t.Errorf("separator lines should be stripped, got: %q", content)
	}
	// Actual result text must be preserved (multi-line)
	for _, want := range []string{"Fixed the TUI tool result rendering.", "- Removed the banner", "- Render markdown"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected result line %q preserved, got: %q", want, content)
		}
	}
}

func TestAttemptCompletionRendererJSONInput(t *testing.T) {
	r := &AttemptCompletionRenderer{}
	res := r.Render(RenderRequest{Phase: types.ToolPhaseComplete, Input: `{"result": "All done"}`, Status: "completed"})
	if res.Skip {
		t.Fatal("expected Complete phase to produce a message")
	}
	if !strings.Contains(res.Content, "All done") {
		t.Errorf("expected result text, got: %q", res.Content)
	}
}

func TestAttemptCompletionRendererEmptyInput(t *testing.T) {
	r := &AttemptCompletionRenderer{}
	res := r.Render(RenderRequest{Phase: types.ToolPhaseComplete, Input: "", Status: "completed"})
	if !res.Skip {
		t.Error("expected empty input to be skipped")
	}
}

func TestStripCompletionBannerPlainResult(t *testing.T) {
	got := stripCompletionBanner("✅ Task Completed\n==========\n\nhello world\n\n==========")
	if got != "hello world" {
		t.Errorf("stripCompletionBanner = %q, want %q", got, "hello world")
	}
}
