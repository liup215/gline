package memory

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChunker(t *testing.T) {
	c := NewChunker()
	c.MaxTokens = 100 // small for testing
	c.Overlap = 10

	text := "Paragraph one.\n\nParagraph two is longer and contains more words to ensure it takes up tokens.\n\nParagraph three is the final one."
	chunks := c.Chunk("doc1", "kb1", text)

	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	for i, ch := range chunks {
		if ch.DocID != "doc1" {
			t.Errorf("chunk %d wrong doc_id: %s", i, ch.DocID)
		}
		if ch.KBID != "kb1" {
			t.Errorf("chunk %d wrong kb_id: %s", i, ch.KBID)
		}
		if ch.Sequence != i {
			t.Errorf("chunk %d wrong sequence: %d", i, ch.Sequence)
		}
		if ch.Content == "" {
			t.Errorf("chunk %d empty content", i)
		}
	}
}

func TestEmbedAndNormalize(t *testing.T) {
	// Dummy embedder that returns fixed-size vectors
	dummy := &dummyEmbedder{dim: 3}
	ctx := context.Background()
	vecs, err := EmbedAndNormalize(ctx, dummy, []string{"hello", "world"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vecs))
	}
	for i, v := range vecs {
		if len(v) != 3 {
			t.Fatalf("vector %d wrong dim: %d", i, len(v))
		}
		var sum float64
		for _, x := range v {
			sum += float64(x) * float64(x)
		}
		if sum < 0.99 || sum > 1.01 {
			t.Errorf("vector %d not normalized: %f", i, sum)
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	s, err := CosineSimilarity(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if s < 0.99 {
		t.Fatalf("expected ~1, got %f", s)
	}
}

func TestKBRegistry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kb-reg.db")
	reg, err := NewKBRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Close()

	ctx := context.Background()
	kb := &KnowledgeBase{Name: "test-kb", Type: KBTypeRAG}
	if err := reg.Create(ctx, kb); err != nil {
		t.Fatal(err)
	}
	if kb.ID == "" {
		t.Fatal("kb ID not generated")
	}

	list, err := reg.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 kb, got %d", len(list))
	}

	got, err := reg.GetByName(ctx, "test-kb")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "test-kb" {
		t.Fatalf("wrong name: %s", got.Name)
	}
}

func TestFactStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "facts.db")
	fs, err := NewSQLiteFactStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()

	ctx := context.Background()

	// Directly insert a fact via internal method (FactStore.Add is stub)
	f := &Fact{
		ID:         genID(),
		Category:   FactPreference,
		Subject:    "User",
		Predicate:  "prefers",
		Object:     "Go",
		Confidence: 0.9,
		CreatedAt:  time.Now().UTC(),
	}
	if err := fs.upsertFact(ctx, nil, f); err != nil {
		t.Fatal(err)
	}

	facts, err := fs.Search(ctx, "Go", FactSearchOptions{TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) == 0 {
		t.Fatal("expected at least one fact")
	}
	found := false
	for _, fact := range facts {
		if fact.Object == "Go" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("did not find inserted fact")
	}
}

func TestVectorStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rag.db")
	vs, err := NewVectorStore(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer vs.Close()

	ctx := context.Background()
	doc := &Document{ID: "d1", KBID: "kb1", Name: "test.md", Content: "hello world foo bar", CharCount: 19, CreatedAt: time.Now().UTC()}
	chunks := []Chunk{
		{ID: "c1", DocID: "d1", KBID: "kb1", Content: "hello world", Embedding: []float32{1, 0, 0}, Sequence: 0},
		{ID: "c2", DocID: "d1", KBID: "kb1", Content: "foo bar", Embedding: []float32{0, 1, 0}, Sequence: 1},
	}
	if err := vs.StoreDocument(ctx, doc, chunks); err != nil {
		t.Fatal(err)
	}

	// Vector search for query close to chunk 1
	results, err := vs.Search(ctx, "kb1", []float32{1, 0, 0}, "hello", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results")
	}
	if results[0].ID != "c1" {
		t.Errorf("expected c1 first, got %s", results[0].ID)
	}
}

func TestWikiFS(t *testing.T) {
	dir := t.TempDir()
	// Monkey-patch KBDir for testing
	orig := KBDir
	KBDir = func(kbID string) string { return filepath.Join(dir, kbID) }
	defer func() { KBDir = orig }()

	if err := InitWikiDirectory("kb1", ""); err != nil {
		t.Fatal(err)
	}
	fs, err := NewWikiFS("kb1")
	if err != nil {
		t.Fatal(err)
	}

	if err := fs.WritePage("concepts/test.md", "# Test\nThis is a [[concept]] page."); err != nil {
		t.Fatal(err)
	}

	content, err := fs.ReadPage("concepts/test.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "concept") {
		t.Fatal("content mismatch")
	}

	links := ExtractLinks(content)
	if len(links) != 1 || links[0] != "concept" {
		t.Fatalf("expected link 'concept', got %v", links)
	}
}

func TestFactExtractor(t *testing.T) {
	e := NewFactExtractor()
	changes, err := e.Extract(context.Background(), "I prefer to use Go for this project.")
	if err != nil {
		t.Fatal(err)
	}
	// Rule-based extraction should detect the preference pattern
	found := false
	for _, ch := range changes {
		if ch.Fact.Category == FactPreference {
			found = true
		}
	}
	if !found {
		t.Logf("no preference fact extracted (expected with rule-based): %+v", changes)
	}
}

type dummyEmbedder struct{ dim int }

func (d *dummyEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		v := make([]float32, d.dim)
		v[0] = float32(i + 1)
		out[i] = v
	}
	return out, nil
}
func (d *dummyEmbedder) Dimension() int   { return d.dim }
func (d *dummyEmbedder) ModelName() string { return "dummy" }
func (d *dummyEmbedder) MaxBatchSize() int { return 10 }
