package provider

// Live smoke tests. These hit the real OpenCode Go endpoint and are skipped
// unless:
//   go test ./internal/provider/ -run TestLive -v   (full mode)
// and GLINE_LIVE_SMOKE=1 is set. They are NOT part of the normal test run.

import (
	"context"
	"os"
	"testing"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

func liveSmokeEnabled(t *testing.T) string {
	t.Helper()
	if os.Getenv("GLINE_LIVE_SMOKE") != "1" {
		t.Skip("set GLINE_LIVE_SMOKE=1 to run live smoke tests")
	}
	key := os.Getenv("OPENCODE_API_KEY")
	if key == "" {
		t.Fatal("OPENCODE_API_KEY is required for live smoke tests")
	}
	return key
}

func TestLiveOpenCodeOneShot(t *testing.T) {
	key := liveSmokeEnabled(t)
	ctx := context.Background()
	llm, err := NewLLM(ctx, "opencode", "mimo-v2.5", key, "", "", nil)
	if err != nil {
		t.Fatalf("NewLLM() error = %v", err)
	}

	for resp, err := range llm.GenerateContent(ctx, &model.LLMRequest{
		Model: "mimo-v2.5",
		Contents: []*genai.Content{{
			Role:  "user",
			Parts: []*genai.Part{{Text: "Reply with exactly: PONG"}},
		}},
		Config: &genai.GenerateContentConfig{MaxOutputTokens: 100},
	}, false) {
		if err != nil {
			t.Fatalf("GenerateContent() error = %v", err)
		}
		if resp.Content == nil || len(resp.Content.Parts) == 0 {
			t.Fatalf("empty response: %+v", resp)
		}
		text := ""
		for _, p := range resp.Content.Parts {
			if p.Text != "" {
				text += p.Text
			}
		}
		if text == "" {
			t.Fatalf("no text in response: %+v", resp.Content)
		}
		t.Logf("model said: %q", text)
	}
}

func TestLiveVolcanoOneShot(t *testing.T) {
	if os.Getenv("GLINE_LIVE_SMOKE") != "1" {
		t.Skip("set GLINE_LIVE_SMOKE=1 to run live smoke tests")
	}
	key := os.Getenv("VOLCANO_API_KEY")
	if key == "" {
		t.Fatal("VOLCANO_API_KEY is required for live smoke tests")
	}
	ctx := context.Background()
	// "volcano" routes through the OpenAI chat-completions client at Ark's
	// OpenAI-compatible endpoint.
	llm, err := NewLLM(ctx, "volcano", os.Getenv("VOLCANO_MODEL"), key, "", "", nil)
	if err != nil {
		t.Fatalf("NewLLM() error = %v", err)
	}
	if os.Getenv("VOLCANO_MODEL") == "" {
		t.Skip("VOLCANO_MODEL not set")
	}

	for resp, err := range llm.GenerateContent(ctx, &model.LLMRequest{
		Model: os.Getenv("VOLCANO_MODEL"),
		Contents: []*genai.Content{{
			Role:  "user",
			Parts: []*genai.Part{{Text: "Reply with exactly: PONG"}},
		}},
		Config: &genai.GenerateContentConfig{MaxOutputTokens: 100},
	}, false) {
		if err != nil {
			t.Fatalf("GenerateContent() error = %v", err)
		}
		text := ""
		if resp.Content != nil {
			for _, p := range resp.Content.Parts {
				text += p.Text
			}
		}
		t.Logf("volcano said: %q", text)
	}
}
