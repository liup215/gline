package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFactChanges(t *testing.T) {
	extractor := NewFactExtractor()
	extractor.MinConfidence = 0.7

	tests := []struct {
		name    string
		input   string
		wantLen int
		wantAct string
		check   func(t *testing.T, changes []FactChange)
	}{
		{
			name:    "valid_json_array",
			input:   `[{"category":"preference","subject":"User","predicate":"prefers","object":"double quotes for strings","confidence":0.95,"action":"ADD"}]`,
			wantLen: 1,
			wantAct: "ADD",
			check: func(t *testing.T, changes []FactChange) {
				assert.Equal(t, "preference", string(changes[0].Fact.Category))
				assert.Equal(t, "User", changes[0].Fact.Subject)
				assert.Equal(t, "prefers", changes[0].Fact.Predicate)
				assert.Equal(t, "double quotes for strings", changes[0].Fact.Object)
				assert.InDelta(t, 0.95, changes[0].Fact.Confidence, 0.01)
			},
		},
		{
			name:    "markdown_fences",
			input:   "```json\n[{\"category\":\"preference\",\"subject\":\"User\",\"predicate\":\"prefers\",\"object\":\"double quotes\",\"confidence\":0.95,\"action\":\"ADD\"}]\n```",
			wantLen: 1,
			check:   func(t *testing.T, changes []FactChange) {},
		},
		{
			name:    "noop_is_skipped",
			input:   `[{"category":"preference","subject":"User","predicate":"prefers","object":"x","confidence":0.95,"action":"NOOP"}]`,
			wantLen: 0,
			check:   func(t *testing.T, changes []FactChange) {},
		},
		{
			name:    "low_confidence_filtered",
			input:   `[{"category":"preference","subject":"User","predicate":"prefers","object":"x","confidence":0.5,"action":"ADD"}]`,
			wantLen: 0,
			check:   func(t *testing.T, changes []FactChange) {},
		},
		{
			name:    "multiple_facts",
			input:   `[{"category":"preference","subject":"User","predicate":"prefers","object":"x","confidence":0.95,"action":"ADD"},{"category":"decision","subject":"Project","predicate":"uses","object":"PostgreSQL 15","confidence":0.9,"action":"ADD"}]`,
			wantLen: 2,
			check: func(t *testing.T, changes []FactChange) {
				assert.Equal(t, "preference", string(changes[0].Fact.Category))
				assert.Equal(t, "decision", string(changes[1].Fact.Category))
			},
		},
		{
			name:    "delete_action",
			input:   `[{"category":"preference","subject":"User","predicate":"uses","object":"Docker Compose","confidence":0.9,"action":"DELETE"}]`,
			wantLen: 1,
			wantAct: "DELETE",
			check: func(t *testing.T, changes []FactChange) {
				assert.Equal(t, "DELETE", changes[0].Action)
			},
		},
		{
			name:    "update_becomes_add",
			input:   `[{"category":"preference","subject":"User","predicate":"prefers","object":"x","confidence":0.95,"action":"UPDATE"}]`,
			wantLen: 1,
			wantAct: "ADD",
			check: func(t *testing.T, changes []FactChange) {
				assert.Equal(t, "ADD", changes[0].Action)
			},
		},
		{
			name:    "empty_array",
			input:   `[]`,
			wantLen: 0,
			check:   func(t *testing.T, changes []FactChange) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := extractor.ParseFactChanges(tt.input)
			require.NoError(t, err)
			assert.Len(t, changes, tt.wantLen)
			if tt.wantLen > 0 && tt.wantAct != "" {
				assert.Equal(t, tt.wantAct, changes[0].Action)
			}
			if tt.check != nil {
				tt.check(t, changes)
			}
		})
	}
}

func TestExtractWithMockLLM(t *testing.T) {
	extractor := NewFactExtractor()
	extractor.Caller = func(ctx context.Context, systemPrompt, userContent string) (string, error) {
		assert.Contains(t, systemPrompt, "knowledge extraction engine")
		assert.Contains(t, userContent, "User:")
		return `[{"category":"preference","subject":"User","predicate":"prefers","object":"dark theme","confidence":0.95,"action":"ADD"}]`, nil
	}

	changes, err := extractor.Extract(context.Background(), "User: I like dark theme.\nAssistant: OK.")
	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "preference", string(changes[0].Fact.Category))
	assert.Equal(t, "dark theme", changes[0].Fact.Object)
	assert.InDelta(t, 0.95, changes[0].Fact.Confidence, 0.01)
}

func TestExtractWithLLMError(t *testing.T) {
	extractor := NewFactExtractor()
	extractor.Caller = func(ctx context.Context, systemPrompt, userContent string) (string, error) {
		return "", fmt.Errorf("connection refused")
	}

	changes, err := extractor.Extract(context.Background(), "User: hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "llm extraction call failed")
	assert.Nil(t, changes)
}

func TestRuleBasedFallback(t *testing.T) {
	extractor := NewFactExtractor()
	// Caller is nil → rule-based fallback
	require.Nil(t, extractor.Caller)

	changes, err := extractor.Extract(context.Background(), "I prefer double quotes for my Go code")
	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "ADD", changes[0].Action)
	assert.Equal(t, "User", changes[0].Fact.Subject)
	assert.Contains(t, changes[0].Fact.Object, "prefer")
}

func TestMaxFactsLimit(t *testing.T) {
	extractor := NewFactExtractor()
	extractor.MaxFacts = 2
	extractor.Caller = func(ctx context.Context, sp, uc string) (string, error) {
		return `[{"category":"preference","subject":"User","predicate":"prefers","object":"a","confidence":0.95,"action":"ADD"},{"category":"preference","subject":"User","predicate":"prefers","object":"b","confidence":0.95,"action":"ADD"},{"category":"preference","subject":"User","predicate":"prefers","object":"c","confidence":0.95,"action":"ADD"}]`, nil
	}

	changes, err := extractor.Extract(context.Background(), "test")
	require.NoError(t, err)
	assert.Len(t, changes, 2)
}

func TestEnrichFacts(t *testing.T) {
	changes := []FactChange{
		{
			Action: "ADD",
			Fact: Fact{
				Subject:   "User",
				Predicate: "prefers",
				Object:    "dark mode",
			},
		},
	}

	enriched := EnrichFacts(changes, "task:abc123", "kb_1")
	require.Len(t, enriched, 1)

	f := enriched[0].Fact
	assert.Equal(t, "task:abc123", f.Source)
	assert.Equal(t, "kb_1", f.KBID)
	assert.False(t, f.CreatedAt.IsZero())
	assert.False(t, f.UpdatedAt.IsZero())
	assert.False(t, f.LastAccess.IsZero())
}

func TestEnrichFactsDefaults(t *testing.T) {
	changes := []FactChange{
		{
			Action: "ADD",
			Fact: Fact{
				Subject:    "User",
				Predicate:  "prefers",
				Object:     "dark mode",
				Confidence: 0, // unset
			},
		},
	}

	enriched := EnrichFacts(changes, "", "")
	assert.Equal(t, "", enriched[0].Fact.Source)
	assert.Equal(t, "", enriched[0].Fact.KBID)
}

func TestApplySmartMerge(t *testing.T) {
	dbPath := t.TempDir() + "/test_facts.db"
	store, err := NewSQLiteFactStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// First ADD
	changes1 := []FactChange{
		{
			Action: "ADD",
			Fact: Fact{
				ID:         "fact_1",
				Subject:    "User",
				Predicate:  "prefers_editor",
				Object:     "VS Code",
				Confidence: 0.9,
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			},
		},
	}
	err = store.Apply(context.Background(), changes1)
	require.NoError(t, err)

	facts, err := store.Search(context.Background(), "editor", FactSearchOptions{TopK: 10})
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "VS Code", facts[0].Object)

	// Second ADD with same (subject, predicate) → should UPDATE
	changes2 := []FactChange{
		{
			Action: "ADD",
			Fact: Fact{
				ID:         "fact_2", // different ID
				Subject:    "User",
				Predicate:  "prefers_editor",
				Object:     "Neovim", // changed object
				Confidence: 0.95,
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			},
		},
	}
	err = store.Apply(context.Background(), changes2)
	require.NoError(t, err)

	facts, err = store.Search(context.Background(), "editor", FactSearchOptions{TopK: 10})
	require.NoError(t, err)
	require.Len(t, facts, 1) // still only 1 fact
	assert.Equal(t, "Neovim", facts[0].Object)
	assert.InDelta(t, 0.95, facts[0].Confidence, 0.01)

	// DELETE by (subject, predicate) when ID doesn't match
	changes3 := []FactChange{
		{
			Action: "DELETE",
			Fact: Fact{
				ID:        "nonexistent_id",
				Subject:   "User",
				Predicate: "prefers_editor",
				Object:    "anything",
			},
		},
	}
	err = store.Apply(context.Background(), changes3)
	require.NoError(t, err)

	facts, err = store.Search(context.Background(), "editor", FactSearchOptions{TopK: 10})
	require.NoError(t, err)
	assert.Len(t, facts, 0)
}

func TestApplyEmptyChanges(t *testing.T) {
	dbPath := t.TempDir() + "/test_empty.db"
	store, err := NewSQLiteFactStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	err = store.Apply(context.Background(), nil)
	require.NoError(t, err)

	err = store.Apply(context.Background(), []FactChange{})
	require.NoError(t, err)
}
