// wiki_engine.go provides the high-level Wiki layer wrapper.
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// WikiManager wraps WikiFS with LLM-driven Ingest/Query/Lint.
type WikiManager struct{}

// NewWikiManager creates a manager.
func NewWikiManager() *WikiManager { return &WikiManager{} }

// Close is a no-op for now.
func (w *WikiManager) Close() error { return nil }

// IngestPrompt is the system prompt for LLM wiki page generation.
const IngestPrompt = `You are a knowledge-base curator. Given a document, generate structured wiki pages in Markdown.

Output a JSON object with these keys:
- "concepts": array of objects { "title": "Concept Name", "content": "markdown summary (2-4 sentences, bullet points welcome)" }
- "entities": array of objects { "name": "Entity Name", "content": "markdown description" }
- "sources": array of objects { "title": "Document Title", "content": "markdown summary of the source document" }
- "index_updates": array of strings — each is a markdown bullet line to append to index.md (e.g. "- [[Concept Name]] — short description")

Rules:
- Extract 2–5 concepts, 0–3 entities, exactly 1 source.
- Keep content concise but specific (max 200 words per page).
- Use [[Wiki Link]] syntax for cross-references where relevant.
- Output ONLY valid JSON. No markdown fences, no explanation.`

// IngestAsync runs LLM-driven wiki generation in the background.
func (w *WikiManager) IngestAsync(kbID string, doc Document, caller func(ctx context.Context, systemPrompt, userContent string) (string, error)) {
	if caller == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Truncate very large documents to avoid token overflow
		content := doc.Content
		if len(content) > 12000 {
			content = content[:12000] + "\n... [truncated]"
		}

		resp, err := caller(ctx, IngestPrompt, fmt.Sprintf("Document: %s\n\n%s", doc.Name, content))
		if err != nil {
			return
		}

		var result struct {
			Concepts     []struct{ Title, Content string } `json:"concepts"`
			Entities     []struct{ Name, Content string } `json:"entities"`
			Sources      []struct{ Title, Content string } `json:"sources"`
			IndexUpdates []string                        `json:"index_updates"`
		}
		if err := parseJSON(resp, &result); err != nil {
			return
		}

		fs, err := NewWikiFS(kbID)
		if err != nil {
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		logEntry := fmt.Sprintf("\n%s — ingested **%s** (%d concepts, %d entities)", now, doc.Name, len(result.Concepts), len(result.Entities))

		// Write concepts
		for _, c := range result.Concepts {
			meta := map[string]string{
				"title":     c.Title,
				"type":      "concept",
				"source":    doc.Name,
				"created":   now,
				"updated":   now,
			}
			body := BuildFrontMatter(meta) + fmt.Sprintf("# %s\n\n%s\n", c.Title, c.Content)
			slug := slugify(c.Title) + ".md"
			_ = fs.WritePage(filepath.Join("concepts", slug), body)
			logEntry += fmt.Sprintf("\n  - wrote concepts/%s", slug)
		}

		// Write entities
		for _, e := range result.Entities {
			meta := map[string]string{
				"name":      e.Name,
				"type":      "entity",
				"source":    doc.Name,
				"created":   now,
			}
			body := BuildFrontMatter(meta) + fmt.Sprintf("# %s\n\n%s\n", e.Name, e.Content)
			slug := slugify(e.Name) + ".md"
			_ = fs.WritePage(filepath.Join("entities", slug), body)
			logEntry += fmt.Sprintf("\n  - wrote entities/%s", slug)
		}

		// Write source
		for _, s := range result.Sources {
			meta := map[string]string{
				"title":     s.Title,
				"type":      "source",
				"doc_id":    doc.ID,
				"created":   now,
			}
			body := BuildFrontMatter(meta) + fmt.Sprintf("# %s\n\n%s\n", s.Title, s.Content)
			slug := slugify(s.Title) + ".md"
			_ = fs.WritePage(filepath.Join("sources", slug), body)
			logEntry += fmt.Sprintf("\n  - wrote sources/%s", slug)
		}

		// Append to index.md
		if len(result.IndexUpdates) > 0 {
			indexBody, _ := fs.ReadPage("index.md")
			updates := strings.Join(result.IndexUpdates, "\n")
			_ = fs.WritePage("index.md", indexBody+"\n"+updates+"\n")
			logEntry += fmt.Sprintf("\n  - updated index.md (+ %d lines)", len(result.IndexUpdates))
		}

		// Append to log.md
		logBody, _ := fs.ReadPage("log.md")
		_ = fs.WritePage("log.md", logBody+logEntry+"\n")
	}()
}

// slugify creates a filesystem-safe slug from a title.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteRune('-')
		}
	}
	result := b.String()
	if result == "" {
		return "untitled"
	}
	return result
}

// parseJSON strips markdown fences and unmarshals.
func parseJSON(raw string, v interface{}) error {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	return json.Unmarshal([]byte(raw), v)
}
