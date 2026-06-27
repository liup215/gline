package summarizer

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/liup215/gline/pkg/types"
)

// Chunk represents a slice of a larger document.
type Chunk struct {
	ID       string
	Content  string
	Sequence int
}

// Chunker splits text into token-aware chunks with overlap.
type Chunker struct {
	MaxTokens   int
	Overlap     int
	estimatorFn func(string) int
}

// NewChunker creates a Chunker with the given max tokens per chunk.
func NewChunker(maxTokens int) *Chunker {
	if maxTokens <= 0 {
		maxTokens = 2000
	}
	return &Chunker{
		MaxTokens:   maxTokens,
		Overlap:     maxTokens / 10, // 10% overlap by default
		estimatorFn: types.EstimateTokens,
	}
}

// SetEstimator overrides the token estimator.
func (c *Chunker) SetEstimator(fn func(string) int) {
	c.estimatorFn = fn
}

// Split breaks a document into sequential chunks. It attempts to keep natural
// boundaries (blank lines) and falls back to line-based or hard truncation so
// no chunk exceeds MaxTokens.
func (c *Chunker) Split(docID, text string) []Chunk {
	if text == "" {
		return nil
	}

	var chunks []Chunk
	var seq int
	var buf strings.Builder

	flush := func() {
		content := strings.TrimSpace(buf.String())
		if content == "" {
			return
		}
		chunks = append(chunks, Chunk{
			ID:       fmt.Sprintf("%s-c%d", docID, seq),
			Content:  content,
			Sequence: seq,
		})
		seq++
		// Overlap: keep enough trailing text to bridge context.
		buf.Reset()
		buf.WriteString(c.overlapTail(content))
	}

	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		candidate := buf.String()
		if candidate != "" {
			candidate += "\n" + line
		} else {
			candidate = line
		}

		if buf.Len() > 0 && c.estimatorFn(candidate) > c.MaxTokens {
			flush()
			buf.WriteString(line)
		} else {
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.WriteString(line)
		}
	}
	flush()

	// Safety check: any individual chunk may still exceed the limit if a single
	// line was enormous. Hard-truncate those.
	for i := range chunks {
		chunks[i].Content = c.hardTruncate(chunks[i].Content)
	}

	return chunks
}

// overlapTail returns the last ~Overlap tokens of text.
func (c *Chunker) overlapTail(text string) string {
	tokens := strings.Fields(text)
	if len(tokens) == 0 {
		return ""
	}
	keep := c.Overlap
	if keep <= 0 {
		return ""
	}
	if len(tokens) <= keep {
		return text
	}
	start := len(tokens) - keep
	return strings.Join(tokens[start:], " ")
}

// hardTruncate truncates content to MaxTokens using a character budget derived
// from the estimator. It keeps the head and tail and omits the middle.
func (c *Chunker) hardTruncate(content string) string {
	if c.estimatorFn(content) <= c.MaxTokens {
		return content
	}
	// Approximate 1 token ≈ 3 bytes on average for mixed text.
	charBudget := c.MaxTokens * 3
	if len(content) <= charBudget {
		return content
	}
	half := charBudget / 2
	omitted := c.estimatorFn(content) - c.MaxTokens
	return content[:half] + fmt.Sprintf("\n\n[... %d tokens omitted ...]\n\n", omitted) + content[len(content)-half:]
}
