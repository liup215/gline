package adkagent

import (
	"context"
	"os"
	"strings"
	"testing"

	glineagent "github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/tools"
)

// TestLiveOpenCodeOneShot runs one real turn against opencode/mimo-v2.5.
// Gated: GLINE_LIVE_SMOKE=1 go test ./internal/adkagent/ -run TestLive -v
func TestLiveOpenCodeOneShot(t *testing.T) {
	key := os.Getenv("OPENCODE_API_KEY")
	if os.Getenv("GLINE_LIVE_SMOKE") != "1" || key == "" {
		t.Skip("set GLINE_LIVE_SMOKE=1 and OPENCODE_API_KEY to run")
	}

	registry, err := tools.InitDefaultRegistry(nil, nil), error(nil)
	a, err := New(context.Background(), Options{
		Provider: "opencode",
		Model:    "mimo-v2.5",
		APIKey:   key,
		BaseURL:  "https://opencode.ai/zen/go/v1",
		Tools:    registry,
		Yolo:     true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cb := newRecordingCallback()
	_, err = a.RunWithCallback(context.Background(), "Reply with exactly the word PONG and nothing else.", cb)
	if err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}
	got := cb.content.String()
	if !strings.Contains(strings.ToUpper(got), "PONG") {
		t.Fatalf("content = %q, want PONG", got)
	}
	t.Logf("content: %q, events: %v", got, cb.order)
}

// TestLiveOpenCodeToolRun makes the model execute a real tool through the
// ADK loop: read a temp file and report its contents.
func TestLiveOpenCodeToolRun(t *testing.T) {
	key := os.Getenv("OPENCODE_API_KEY")
	if os.Getenv("GLINE_LIVE_SMOKE") != "1" || key == "" {
		t.Skip("set GLINE_LIVE_SMOKE=1 and OPENCODE_API_KEY to run")
	}

	f := t.TempDir() + "/probe.txt"
	if err := os.WriteFile(f, []byte("ZEBRA-4242"), 0o644); err != nil {
		t.Fatal(err)
	}

	registry, _ := tools.InitDefaultRegistry(nil, nil), error(nil)
	a, err := New(context.Background(), Options{
		Provider: "opencode",
		Model:    "mimo-v2.5",
		APIKey:   key,
		BaseURL:  "https://opencode.ai/zen/go/v1",
		Tools:    registry,
		Yolo:     true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cb := newRecordingCallback()
	prompt := "Read the file " + f + " with the read tool, then tell me its exact contents."
	if _, err := a.RunWithCallback(context.Background(), prompt, cb); err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}
	if _, ok := cb.results["read"]; !ok {
		t.Fatalf("read tool never ran; events=%v", cb.order)
	}
	if !strings.Contains(cb.content.String(), "ZEBRA-4242") {
		t.Fatalf("final reply %q missing file contents", cb.content.String())
	}
	t.Logf("tool result ok, reply: %q", cb.content.String())
}

// compile-time interface check
var _ glineagent.StreamCallback = (*recordingCallback)(nil)

// TestLiveVolcanoOneShot runs one real turn against the Ark plan endpoint.
func TestLiveVolcanoOneShot(t *testing.T) {
	key := os.Getenv("ARK_API_KEY")
	if os.Getenv("GLINE_LIVE_SMOKE") != "1" || key == "" {
		t.Skip("set GLINE_LIVE_SMOKE=1 and ARK_API_KEY to run")
	}
	registry, _ := tools.InitDefaultRegistry(nil, nil), error(nil)
	a, err := New(context.Background(), Options{
		Provider: "volcano",
		Model:    "ark-code-latest",
		APIKey:   key,
		BaseURL:  "https://ark.cn-beijing.volces.com/api/plan/v3",
		Tools:    registry,
		Yolo:     true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cb := newRecordingCallback()
	if _, err := a.RunWithCallback(context.Background(), "Reply with exactly the word PONG and nothing else.", cb); err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}
	if !strings.Contains(strings.ToUpper(cb.content.String()), "PONG") {
		t.Fatalf("content = %q", cb.content.String())
	}
	t.Logf("content: %q", cb.content.String())
}
