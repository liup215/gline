package memory

import (
	"fmt"
	"strings"
)

// Chunker splits text into token-aware chunks with configurable overlap.
type Chunker struct {
	MaxTokens  int   // target tokens per chunk (default 512)
	Overlap    int   // overlap tokens (default ~20% of MaxTokens, e.g. 100)
	chunkToken func(string) int // lightweight token estimator
}

// NewChunker creates a Chunker with defaults.
func NewChunker() *Chunker {
	return &Chunker{
		MaxTokens:  512,
		Overlap:    100,
		chunkToken: estimateTokens,
	}
}

// SetMaxTokens overrides the chunk size.
func (c *Chunker) SetMaxTokens(n int) { c.MaxTokens = n; c.Overlap = n / 5 }

// Chunk produces a sequential slice of chunks from a document.
func (c *Chunker) Chunk(docID, kbID, text string) []Chunk {
	if text == "" {
		return nil
	}
	// Split on paragraph boundaries first
	paragraphs := splitParagraphs(text)
	var chunks []Chunk
	var buf strings.Builder
	var seq int
	var offset int

	flush := func() {
		content := strings.TrimSpace(buf.String())
		if content == "" {
			return
		}
		chunks = append(chunks, Chunk{
			ID:          fmt.Sprintf("%s-c%d", docID, seq),
			DocID:       docID,
			KBID:        kbID,
			Content:     content,
			StartOffset: offset,
			EndOffset:   offset + len(content),
			Sequence:    seq,
		})
		seq++
		// Overlap strategy: keep last N tokens worth of text
		off, overlapText := c.overlapTail(content)
		offset += len(content) - off
		buf.Reset()
		buf.WriteString(overlapText)
	}

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if buf.Len() > 0 && c.chunkToken(buf.String()+"\n\n"+para) > c.MaxTokens {
			flush()
		}
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(para)
	}
	flush()
	return chunks
}

func (c *Chunker) overlapTail(text string) (int, string) {
	words := strings.Fields(text)
	if len(words) <= c.Overlap {
		return 0, text
	}
	start := len(words) - c.Overlap
	tail := strings.Join(words[start:], " ")
	// find position of tail in original text
	idx := strings.LastIndex(text, tail)
	if idx < 0 {
		idx = len(text) - len(tail)
		if idx < 0 {
			idx = 0
		}
	}
	return idx, tail
}

func splitParagraphs(text string) []string {
	// Normalise line endings then split on double newline
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.Split(text, "\n\n")
}

// estimateTokens does a rough 1 token ≈ 4 chars (fast, no external deps).
func estimateTokens(s string) int {
	return len(s) / 4
}
