package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liup215/gline/internal/memory"
)

// ─── KB Search Tool ─────────────────────────────────────────────────────────

// KBSearchTool searches a knowledge base

type KBSearchTool struct {
	BaseTool
	engine *memory.UnifiedEngine
}

// KBSearchInput represents the input for kb_search
type KBSearchInput struct {
	Query   string `json:"query"`
	KBID    string `json:"kb_id,omitempty"`
	TopK    int    `json:"top_k,omitempty"`
	MinScore float64 `json:"min_score,omitempty"`
}

// NewKBSearchTool creates a new kb_search tool
func NewKBSearchTool(engine *memory.UnifiedEngine) *KBSearchTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The natural language query to search for"
			},
			"kb_id": {
				"type": "string",
				"description": "The knowledge base ID or name. Defaults to 'default' if not provided."
			},
			"top_k": {
				"type": "integer",
				"description": "Number of top documents to return (default 5)",
				"default": 5
			},
			"min_score": {
				"type": "number",
				"description": "Minimum relevance score threshold 0-1 (default 0.5)",
				"default": 0.5
			}
		},
		"required": ["query"]
	}`)

	return &KBSearchTool{
		BaseTool: BaseTool{
			name:        "kb_search",
			description: "Search a knowledge base for documents or facts relevant to a query. Use this when the user asks about previously indexed documents, codebases, or stored knowledge.",
			inputSchema: schema,
		},
		engine: engine,
	}
}

// Execute searches the knowledge base
func (t *KBSearchTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	if t.engine == nil {
		return "", fmt.Errorf("memory engine not available")
	}

	var req KBSearchInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}
	if req.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	kbID := req.KBID
	if kbID == "" {
		kbID = "default"
		if err := memory.EnsureDefaultKB(ctx, t.engine); err != nil {
			return "", fmt.Errorf("ensure default KB: %w", err)
		}
	}

	// Resolve kb name to id
	kb, err := t.engine.GetKB(ctx, kbID)
	if err != nil {
		return "", fmt.Errorf("KB not found: %s", kbID)
	}

	topK := req.TopK
	if topK <= 0 {
		topK = 5
	}
	minScore := req.MinScore
	if minScore <= 0 {
		minScore = 0.5
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("🔍 Searching knowledge base %q for: %s\n\n", kb.Name, req.Query))

	// RAG search
	result.WriteString("═══ Document Results (RAG) ═══\n")
	vecs, err := memory.EmbedAndNormalize(ctx, t.engine.Embedder, []string{req.Query})
	if err != nil {
		result.WriteString(fmt.Sprintf("Embedding error: %v\n", err))
	} else {
		chunks, err := t.engine.RAGEngine.Search(ctx, kb.ID, vecs[0], req.Query, topK, minScore)
		if err != nil {
			result.WriteString(fmt.Sprintf("RAG search error: %v\n", err))
		} else if len(chunks) == 0 {
			result.WriteString("No matching documents found.\n")
		} else {
			for i, c := range chunks {
				result.WriteString(fmt.Sprintf("%d. [%s] %s...\n", i+1, c.DocID, truncate(c.Content, 200)))
				result.WriteString(fmt.Sprintf("   Source: %s | Seq: %d\n\n", c.KBID, c.Sequence))
			}
		}
	}

	// Fact search
	result.WriteString("\n═══ Related Facts ═══\n")
	facts, err := t.engine.FactStore.Search(ctx, req.Query, memory.FactSearchOptions{TopK: topK})
	if err != nil {
		result.WriteString(fmt.Sprintf("Fact search error: %v\n", err))
	} else if len(facts) == 0 {
		result.WriteString("No matching facts found.\n")
	} else {
		for i, f := range facts {
			result.WriteString(fmt.Sprintf("%d. [%s] %s (conf=%.2f)\n", i+1, f.Category, f.Sentence(), f.Confidence))
		}
	}

	return result.String(), nil
}

// ─── KB Ingest Tool ─────────────────────────────────────────────────────────

// KBIngestTool ingests a file into a knowledge base
type KBIngestTool struct {
	BaseTool
	engine *memory.UnifiedEngine
}

// KBIngestInput represents the input for kb_ingest
type KBIngestInput struct {
	FilePath string `json:"file_path"`
	KBID     string `json:"kb_id,omitempty"`
}

// NewKBIngestTool creates a new kb_ingest tool
func NewKBIngestTool(engine *memory.UnifiedEngine) *KBIngestTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_path": {
				"type": "string",
				"description": "The path of the file to ingest into the knowledge base"
			},
			"kb_id": {
				"type": "string",
				"description": "The target knowledge base ID or name. Defaults to 'default' if not provided."
			}
		},
		"required": ["file_path"]
	}`)

	return &KBIngestTool{
		BaseTool: BaseTool{
			name:        "kb_ingest",
			description: "Ingest a file (code, document, PDF, etc.) into a knowledge base for later retrieval via kb_search. Use this when the user wants to add documents to their knowledge base.",
			inputSchema: schema,
		},
		engine: engine,
	}
}

// Execute ingests the file
func (t *KBIngestTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	if t.engine == nil {
		return "", fmt.Errorf("memory engine not available")
	}

	var req KBIngestInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}
	if req.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}

	kbID := req.KBID
	if kbID == "" {
		kbID = "default"
		if err := memory.EnsureDefaultKB(ctx, t.engine); err != nil {
			return "", fmt.Errorf("ensure default KB: %w", err)
		}
	}

	kb, err := t.engine.GetKB(ctx, kbID)
	if err != nil {
		return "", fmt.Errorf("KB not found: %s", kbID)
	}

	if err := t.engine.IngestFile(ctx, kb.ID, req.FilePath); err != nil {
		return "", fmt.Errorf("ingest failed: %w", err)
	}

	return fmt.Sprintf("Successfully ingested %s into knowledge base %q.", req.FilePath, kb.Name), nil
}

// ─── Memory Recall Tool ─────────────────────────────────────────────────────

// MemoryRecallTool recalls facts about a subject
type MemoryRecallTool struct {
	BaseTool
	engine *memory.UnifiedEngine
}

// MemoryRecallInput represents the input for memory_recall
type MemoryRecallInput struct {
	Subject    string   `json:"subject"`
	Categories []string `json:"categories,omitempty"`
	TopK       int      `json:"top_k,omitempty"`
}

// NewMemoryRecallTool creates a new memory_recall tool
func NewMemoryRecallTool(engine *memory.UnifiedEngine) *MemoryRecallTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"subject": {
				"type": "string",
				"description": "The subject or entity to recall facts about (e.g. 'user preferences', 'project structure', 'API design decisions')"
			},
			"categories": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Optional fact categories to filter: entity, preference, decision, pattern, task, relation"
			},
			"top_k": {
				"type": "integer",
				"description": "Number of facts to return (default 5)",
				"default": 5
			}
		},
		"required": ["subject"]
	}`)

	return &MemoryRecallTool{
		BaseTool: BaseTool{
			name:        "memory_recall",
			description: "Recall previously remembered facts about a subject. Use this when you need to know what was previously learned about the user's preferences, past decisions, or project patterns.",
			inputSchema: schema,
		},
		engine: engine,
	}
}

// Execute recalls facts
func (t *MemoryRecallTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	if t.engine == nil {
		return "", fmt.Errorf("memory engine not available")
	}

	var req MemoryRecallInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}
	if req.Subject == "" {
		return "", fmt.Errorf("subject is required")
	}

	topK := req.TopK
	if topK <= 0 {
		topK = 5
	}

	opts := memory.FactSearchOptions{
		TopK:     topK,
		Entities: []string{req.Subject},
	}
	for _, c := range req.Categories {
		opts.Categories = append(opts.Categories, memory.FactCategory(c))
	}

	// First try entity search
	facts, err := t.engine.FactStore.GetByEntity(ctx, req.Subject)
	if err != nil || len(facts) == 0 {
		// Fallback to general search
		facts, err = t.engine.FactStore.Search(ctx, req.Subject, opts)
	}
	if err != nil {
		return "", fmt.Errorf("fact search: %w", err)
	}

	if len(facts) == 0 {
		return fmt.Sprintf("No facts found about %q.", req.Subject), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Recalled %d fact(s) about %q:\n\n", len(facts), req.Subject))
	for _, f := range facts {
		result.WriteString(fmt.Sprintf("[%s] %s\n", f.Category, f.Sentence()))
		if f.Confidence > 0 {
			result.WriteString(fmt.Sprintf("  confidence=%.2f  source=%s  updated=%s\n", f.Confidence, f.Source, f.UpdatedAt.Format("2006-01-02")))
		}
		result.WriteString("\n")
	}
	return result.String(), nil
}

// ─── Memory Note Tool ─────────────────────────────────────────────────────────

// MemoryNoteTool stores a fact for later recall
type MemoryNoteTool struct {
	BaseTool
	engine *memory.UnifiedEngine
}

// MemoryNoteInput represents the input for memory_note
type MemoryNoteInput struct {
	Text     string `json:"text"`
	Category string `json:"category,omitempty"`
	Subject  string `json:"subject,omitempty"`
}

// NewMemoryNoteTool creates a new memory_note tool
func NewMemoryNoteTool(engine *memory.UnifiedEngine) *MemoryNoteTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"text": {
				"type": "string",
				"description": "The fact or information to remember (e.g. 'User prefers dark mode' or 'Project uses go modules')"
			},
			"category": {
				"type": "string",
				"description": "Category of fact: entity, preference, decision, pattern, task, relation. Defaults to 'preference' if not provided."
			},
			"subject": {
				"type": "string",
				"description": "The subject entity this fact is about. Defaults to extracting from text if not provided."
			}
		},
		"required": ["text"]
	}`)

	return &MemoryNoteTool{
		BaseTool: BaseTool{
			name:        "memory_note",
			description: "Store a fact in memory for later recall via memory_recall. Use this when you learn something important about the user, their preferences, or the project that you want to remember across sessions.",
			inputSchema: schema,
		},
		engine: engine,
	}
}

// Execute stores a fact
func (t *MemoryNoteTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	if t.engine == nil {
		return "", fmt.Errorf("memory engine not available")
	}

	var req MemoryNoteInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}
	if req.Text == "" {
		return "", fmt.Errorf("text is required")
	}

	cat := memory.FactPreference
	if req.Category != "" {
		cat = memory.FactCategory(req.Category)
	}

	source := "agent_memory_note"
	_, err := t.engine.FactStore.Add(ctx, req.Text, memory.ConversationRef{})
	if err != nil {
		// If Add requires a caller, create a manual fact
		fact := memory.Fact{
			Category: cat,
			Subject:  req.Subject,
			Predicate: "is noted as",
			Object:   req.Text,
			Source:   source,
		}
		change := memory.FactChange{Action: "ADD", Fact: fact}
		err = t.engine.FactStore.Apply(ctx, []memory.FactChange{change})
		if err != nil {
			return "", fmt.Errorf("store fact: %w", err)
		}
	}

	return fmt.Sprintf("Remembered: [%s] %s", cat, req.Text), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
