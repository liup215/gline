// Package memory implements a four-layer unified memory engine for gline:
//   Layer 1 (Fact)    - mem0-style semantic facts with cross-session persistence
//   Layer 2 (Wiki)    - Karpathy-style LLM-maintained markdown wiki
//   Layer 3 (RAG)     - Vector search over original documents (sqlite-vec fallback)
//   Layer 4 (Conv)    - Conversation history with cross-layer linkage
package memory

import (
	"context"
	"time"
)

// ─── Layer 1: Fact (mem0 style) ─────────────────────────────────────────────

type FactCategory string

const (
	FactEntity       FactCategory = "entity"
	FactPreference   FactCategory = "preference"
	FactDecision     FactCategory = "decision"
	FactPattern      FactCategory = "pattern"
	FactTask         FactCategory = "task"
	FactRelationship FactCategory = "relation"
)

// Fact represents an extracted semantic fact.
type Fact struct {
	ID          string       `json:"id"`
	KBID        string       `json:"kb_id,omitempty"`     // optional; empty = global fact
	Category    FactCategory `json:"category"`
	Subject     string       `json:"subject"`
	Predicate   string       `json:"predicate"`
	Object      string       `json:"object"`
	Confidence  float64      `json:"confidence"`
	Source      string       `json:"source"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	AccessCount int          `json:"access_count"`
	LastAccess  time.Time    `json:"last_access"`
}

func (f Fact) Sentence() string {
	return "[" + string(f.Category) + "] " + f.Subject + " " + f.Predicate + " " + f.Object
}

// FactChange records an ADD / UPDATE / DELETE / NOOP operation.
type FactChange struct {
	Action string `json:"action"` // ADD | UPDATE | DELETE | NOOP
	Fact   Fact   `json:"fact"`
	Reason string `json:"reason"`
}

// FactSearchOptions for retrieval.
type FactSearchOptions struct {
	TopK       int
	MinScore   float64
	Categories []FactCategory
	Entities   []string
}

// FactStore is the Layer-1 persistence interface.
type FactStore interface {
	Add(ctx context.Context, text string, source ConversationRef) ([]FactChange, error)
	Search(ctx context.Context, query string, opts FactSearchOptions) ([]Fact, error)
	GetByEntity(ctx context.Context, entity string) ([]Fact, error)
	GetByCategory(ctx context.Context, cat FactCategory) ([]Fact, error)
	Decay(ctx context.Context) error
	Close() error
}

// ─── Layer 2: Wiki (Karpathy style) ─────────────────────────────────────────

// WikiPage is an LLM-maintained markdown page.
type WikiPage struct {
	ID         string    `json:"id"`
	KBID       string    `json:"kb_id"`
	Title      string    `json:"title"`
	FilePath   string    `json:"file_path"`
	Content    string    `json:"content"`
	Links      []string  `json:"links"`
	Backlinks  []string  `json:"backlinks"`
	SourceRefs []string  `json:"source_refs"`
	Tags       []string  `json:"tags"`
	FactIDs    []string  `json:"fact_ids"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WikiEngine is the Layer-2 interface.
type WikiEngine interface {
	InitKB(kbID string, schema string) error
	Ingest(ctx context.Context, kbID string, doc Document) error
	Search(ctx context.Context, query string, opts WikiSearchOptions) ([]WikiPage, error)
	Lint(ctx context.Context, kbID string) ([]Issue, error)
	GetPage(ctx context.Context, kbID, pagePath string) (*WikiPage, error)
	Close() error
}

// WikiSearchOptions for retrieval.
type WikiSearchOptions struct {
	TopK     int
	MinScore float64
}

// Issue reported by wiki lint.
type Issue struct {
	Severity string `json:"severity"` // warning | error
	Type     string `json:"type"`     // dead_link | orphan | conflict | missing_entity | outdated
	Page     string `json:"page"`
	Message  string `json:"message"`
}

// DefaultWikiSchema shipped with every new wiki knowledge base.
const DefaultWikiSchema = `# Wiki Schema for gline knowledge base
## Naming conventions
- concepts/<kebab-case>.md   — concept pages
- entities/<kebab-case>.md   — entity pages
- sources/<slug>.md          — source-document summaries
- synthesis/<topic>.md       — cross-cutting synthesis pages

## Page template
- Every page MUST have YAML front-matter: title, date, sources, tags
- Cross-references MUST use [[wiki-link]] syntax
- Conflicts MUST be marked with {{conflict:target-page}}

## Ingest workflow
1. Read raw document
2. Extract entities & concepts → create/update pages
3. Update index.md table of contents
4. Append one line to log.md

## Lint checks
- Dead links in [[...]]
- Conflicting statements marked {{conflict:...}}
- Orphan pages (no incoming links for 30 days)
- Missing entity pages for known entities
- Pages not updated in 30 days
`

// ─── Layer 3: RAG (vector search) ─────────────────────────────────────────────

// Document is an original source file registered for retrieval.
type Document struct {
	ID         string    `json:"id"`
	KBID       string    `json:"kb_id"`
	Name       string    `json:"name"`
	SourcePath string    `json:"source_path"`
	FileType   string    `json:"file_type"`
	Content    string    `json:"content"`
	CharCount  int       `json:"char_count"`
	ChunkCount int       `json:"chunk_count"`
	Status     string    `json:"status"` // indexed | indexing | failed
	FactIDs    []string  `json:"fact_ids"`
	CreatedAt  time.Time `json:"created_at"`
}

// Chunk is a token-aware slice of a Document with an embedding vector.
type Chunk struct {
	ID          string    `json:"id"`
	DocID       string    `json:"doc_id"`
	KBID        string    `json:"kb_id"`
	Content     string    `json:"content"`
	StartOffset int       `json:"start_offset"`
	EndOffset   int       `json:"end_offset"`
	Embedding   []float32 `json:"embedding,omitempty"`
	Sequence    int       `json:"sequence"`
	WikiPageRef string    `json:"wiki_page_ref,omitempty"`
	FactIDs     []string  `json:"fact_ids,omitempty"`
}

// RAGEngine is the Layer-3 interface.
type RAGEngine interface {
	AddDocument(ctx context.Context, doc Document) error
	RemoveDocument(ctx context.Context, docID string) error
	Search(ctx context.Context, query string, opts RAGSearchOptions) ([]Chunk, error)
	ListDocuments(ctx context.Context, kbID string) ([]Document, error)
	Close() error
}

// RAGSearchOptions for retrieval.
type RAGSearchOptions struct {
	TopK     int
	MinScore float64
}

// ─── Layer 4: Conversation (existing storage, extended) ──────────────────────

// ConversationRef identifies a conversation turn for fact provenance.
type ConversationRef struct {
	TaskID    string `json:"task_id"`
	MessageID int64  `json:"message_id"`
}

// MessageExt are fields that can be added to the existing storage.Message
// (migration in storage package).
type MessageExt struct {
	FactsExtracted   []string `json:"facts_extracted"`
	WikiPagesTouched []string `json:"wiki_pages_touched"`
	SourceDocs       []string `json:"source_docs"`
}

// ─── KnowledgeBase ────────────────────────────────────────────────────────────

// KBType selects the active memory layers for a knowledge base.
type KBType string

const (
	KBTypeRAG    KBType = "rag"
	KBTypeWiki   KBType = "wiki"
	KBTypeHybrid KBType = "hybrid"
)

// KnowledgeBase is a user-created knowledge container.
type KnowledgeBase struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        KBType    `json:"type"`
	DocCount    int       `json:"doc_count"`
	ChunkCount  int       `json:"chunk_count"`
	FactCount   int       `json:"fact_count"`
	WikiPageCount int     `json:"wiki_page_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ─── Unified Memory Engine ───────────────────────────────────────────────────

// Engine orchestrates all four layers.
type Engine struct {
	FactStore  FactStore
	WikiEngine WikiEngine
	RAGEngine  RAGEngine
}

// MemoryMode controls which layers participate in a single turn.
type MemoryMode string

const (
	ModeAuto       MemoryMode = "auto"
	ModeFact       MemoryMode = "fact"
	ModeWiki       MemoryMode = "wiki"
	ModeRAG        MemoryMode = "rag"
	ModeAll        MemoryMode = "all"
	ModeNone       MemoryMode = "none"
)

// ContextPack is the assembled retrieval result injected into agent prompt.
type ContextPack struct {
	Facts       []Fact     `json:"facts"`
	WikiPages   []WikiPage `json:"wiki_pages"`
	Chunks      []Chunk    `json:"chunks"`
	SystemPrompt string    `json:"system_prompt"`
	UserPrefix   string    `json:"user_prefix"`
}
