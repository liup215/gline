package adkagent

import (
	"context"
	"iter"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session/compaction"
	"google.golang.org/genai"

	"github.com/liup215/gline/internal/memory"
	"github.com/liup215/gline/pkg/types"
)

// ─── compaction option ───────────────────────────────────────────────────────

func TestNewAcceptsCompactionConfig(t *testing.T) {
	m := &fakeModel{turns: []fakeTurn{{final: modelText("ok")}}}
	opts := Options{Model: "fake", Tools: testRegistry(t, ""), Compaction: &compaction.Config{
		TokenThreshold:     1000,
		EventRetentionSize: 2,
	}}
	if _, err := NewWithModel(context.Background(), opts, m); err != nil {
		t.Fatalf("New with compaction config: %v", err)
	}
}

func TestNewRejectsInvalidCompactionConfig(t *testing.T) {
	m := &fakeModel{turns: []fakeTurn{{final: modelText("ok")}}}
	opts := Options{Model: "fake", Tools: testRegistry(t, ""), Compaction: &compaction.Config{
		TokenThreshold: 1000, // missing EventRetentionSize
	}}
	if _, err := NewWithModel(context.Background(), opts, m); err == nil {
		t.Fatal("expected validation error for compaction config without EventRetentionSize")
	}
}

// ─── fact extraction ─────────────────────────────────────────────────────────

// factTestModel serves scripted streaming turns for the main loop and a
// fixed non-streaming response (the fact-extraction call carries a system
// instruction, which normal turns never have).
type factTestModel struct {
	mu       sync.Mutex
	turns    []*genai.Content // streaming turn finals
	extractN int              // number of non-streaming extraction calls served
	reqs     []*model.LLMRequest
}

func (f *factTestModel) Name() string { return "fact-test" }

func (f *factTestModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		f.mu.Lock()
		f.reqs = append(f.reqs, req)
		var final *genai.Content
		if !stream && req.Config != nil && req.Config.SystemInstruction != nil {
			// Fact-extraction one-shot.
			final = modelText(`[{"category":"preference","subject":"User","predicate":"prefers","object":"tabs over spaces","confidence":0.9,"action":"ADD"}]`)
			f.extractN++
		} else if len(f.turns) > 0 {
			final = f.turns[0]
			f.turns = f.turns[1:]
		}
		f.mu.Unlock()

		if !stream {
			if final != nil {
				yield(&model.LLMResponse{Content: final}, nil)
			}
			return
		}
		if final != nil {
			// Stream text through as partials, then the aggregate.
			for _, p := range final.Parts {
				if p.Text != "" {
					if !yield(&model.LLMResponse{
						Content: &genai.Content{Role: "model", Parts: []*genai.Part{{Text: p.Text}}},
						Partial: true,
					}, nil) {
						return
					}
				}
			}
			yield(&model.LLMResponse{Content: final, TurnComplete: true}, nil)
		}
	}
}

// captureFactStore records Apply calls and signals each one.
type captureFactStore struct {
	mu     sync.Mutex
	apps   [][]memory.FactChange
	signal chan struct{}
}

func newCaptureFactStore() *captureFactStore {
	return &captureFactStore{signal: make(chan struct{}, 8)}
}

func (s *captureFactStore) Add(ctx context.Context, text string, source memory.ConversationRef) ([]memory.FactChange, error) {
	return nil, nil
}

func (s *captureFactStore) Apply(ctx context.Context, changes []memory.FactChange) error {
	s.mu.Lock()
	s.apps = append(s.apps, changes)
	s.mu.Unlock()
	s.signal <- struct{}{}
	return nil
}

func (s *captureFactStore) Search(ctx context.Context, query string, opts memory.FactSearchOptions) ([]memory.Fact, error) {
	return nil, nil
}
func (s *captureFactStore) GetByEntity(ctx context.Context, entity string) ([]memory.Fact, error) {
	return nil, nil
}
func (s *captureFactStore) GetByCategory(ctx context.Context, cat memory.FactCategory) ([]memory.Fact, error) {
	return nil, nil
}
func (s *captureFactStore) Decay(ctx context.Context) error { return nil }
func (s *captureFactStore) Close() error                    { return nil }

func (s *captureFactStore) applied(t *testing.T) []memory.FactChange {
	t.Helper()
	select {
	case <-s.signal:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for fact Apply")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.apps[len(s.apps)-1]
}

func TestFactExtractionOnCompletedRun(t *testing.T) {
	m := &factTestModel{turns: []*genai.Content{modelText("User prefers tabs over spaces. All done.")}}
	store := newCaptureFactStore()
	engine := &memory.UnifiedEngine{FactStore: store}

	opts := Options{Model: "fake", Tools: testRegistry(t, ""), MemoryEngine: engine}
	a, err := NewWithModel(context.Background(), opts, m)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cb := newRecordingCallback()
	if _, err := a.RunWithCallback(context.Background(), "I prefer tabs over spaces for indentation", cb); err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}

	changes := store.applied(t)
	if len(changes) != 1 {
		t.Fatalf("expected 1 fact change, got %d: %+v", len(changes), changes)
	}
	if changes[0].Action != "ADD" || changes[0].Fact.Predicate != "prefers" {
		t.Fatalf("unexpected change: %+v", changes[0])
	}
	if changes[0].Fact.Source == "" {
		t.Fatal("EnrichFacts should stamp the conversation source")
	}
}

func TestFactExtractionSkippedWithoutEngine(t *testing.T) {
	m := &factTestModel{turns: []*genai.Content{modelText("done")}}
	opts := Options{Model: "fake", Tools: testRegistry(t, "")}
	a, err := NewWithModel(context.Background(), opts, m)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cb := newRecordingCallback()
	if _, err := a.RunWithCallback(context.Background(), "hello", cb); err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}
	// No memory engine: the extraction one-shot must never be issued.
	time.Sleep(100 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.extractN != 0 {
		t.Fatalf("fact extraction ran without a memory engine (%d calls)", m.extractN)
	}
}

func TestTranscriptFromMessages(t *testing.T) {
	out := transcriptFromMessages([]types.Message{
		{Role: types.RoleUser, Content: "hi"},
		{Role: types.RoleAssistant, Content: "hello"},
		{Role: types.RoleTool, Content: "ignored"},
	})
	if !strings.Contains(out, "User: hi") || !strings.Contains(out, "Assistant: hello") {
		t.Fatalf("unexpected transcript: %q", out)
	}
	if strings.Contains(out, "ignored") {
		t.Fatal("tool messages must not appear in the fact transcript")
	}
}
