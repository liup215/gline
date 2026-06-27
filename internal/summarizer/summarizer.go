package summarizer

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/liup215/gline/pkg/types"
)

const defaultSummaryPrompt = `You are a precise summarization assistant. Your task is to extract the most important information from the provided file chunk and produce a concise, structured summary.

Focus on:
- Purpose of the file (only if you can infer it from this chunk)
- Key functions, types, classes, structs, interfaces, or constants
- Important logic, decision points, or TODOs
- Error handling patterns or critical caveats
- Any code snippets that are essential to understand the chunk

Keep the summary compact. Use bullet points. Do not add commentary outside the summary.

Here is the chunk to summarize:

%s`

const mergeSummaryPrompt = `You are a precise summarization assistant. Merge the following per-chunk summaries of a single file into one coherent, compact overview.

Rules:
- Preserve the most important functions/types/constants from each chunk.
- Remove duplicates.
- Keep the final output much shorter than the combined input.
- Use bullet points and short code snippets only when necessary.

Per-chunk summaries:

%s`

// Caller abstracts the LLM call used by the summarizer. Implementations should
// send the provided prompt and return the model's text response. This keeps the
// summarizer package independent from agent/subagent internals.
type Caller interface {
	Call(ctx context.Context, prompt string) (string, error)
}

// CallerFunc adapts a plain function to the Caller interface.
type CallerFunc func(ctx context.Context, prompt string) (string, error)

// Call implements Caller.
func (f CallerFunc) Call(ctx context.Context, prompt string) (string, error) {
	return f(ctx, prompt)
}

// Options controls summarization behavior.
type Options struct {
	// MaxChunkTokens is the maximum tokens per chunk sent to the LLM.
	MaxChunkTokens int
	// MaxSummaryTokens is the desired maximum tokens of the final summary.
	MaxSummaryTokens int
	// MaxParallelChunks limits concurrent chunk summarization calls.
	MaxParallelChunks int
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		MaxChunkTokens:    2000,
		MaxSummaryTokens:  2000,
		MaxParallelChunks: 4,
	}
}

// Summarizer produces structured summaries of large files.
type Summarizer struct {
	caller Caller
	opts   Options
}

// NewSummarizer creates a Summarizer using the given Caller.
func NewSummarizer(caller Caller, opts Options) *Summarizer {
	if opts.MaxChunkTokens <= 0 {
		opts.MaxChunkTokens = 2000
	}
	if opts.MaxSummaryTokens <= 0 {
		opts.MaxSummaryTokens = 2000
	}
	if opts.MaxParallelChunks <= 0 {
		opts.MaxParallelChunks = 4
	}
	return &Summarizer{
		caller: caller,
		opts:   opts,
	}
}

// SummarizeFile reads the file, chunks it, summarizes each chunk in parallel,
// and returns a compact structured summary.
func (s *Summarizer) SummarizeFile(ctx context.Context, path string, content []byte) (string, error) {
	if len(content) == 0 {
		return "", fmt.Errorf("empty content for %s", path)
	}
	chunker := NewChunker(s.opts.MaxChunkTokens)
	chunks := chunker.Split(fmt.Sprintf("file:%s", path), string(content))
	if len(chunks) == 0 {
		return "", fmt.Errorf("no chunks produced for %s", path)
	}


	summaries := make([]string, len(chunks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.opts.MaxParallelChunks)
	var firstErr error
	var errMu sync.Mutex

	for i, chunk := range chunks {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, c Chunk) {
			defer wg.Done()
			defer func() { <-sem }()

			prompt := fmt.Sprintf(defaultSummaryPrompt, c.Content)
			summary, err := s.summarizeChunk(ctx, prompt, c.ID)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				return
			}
			summaries[idx] = summary
		}(i, chunk)
	}
	wg.Wait()

	if firstErr != nil {
		return "", firstErr
	}

	merged, err := s.mergeSummaries(ctx, summaries)
	if err != nil {
		return "", err
	}

	final := fmt.Sprintf("[Summary of %s]\n%s\n[Original size: ~%d tokens; summarized to ~%d tokens]",
		path, merged, types.EstimateTokens(string(content)), types.EstimateTokens(merged))
	return final, nil
}

// summarizeChunk sends a single chunk to the caller and returns the summary.
func (s *Summarizer) summarizeChunk(ctx context.Context, prompt, id string) (string, error) {
	if s.caller == nil {
		return "", fmt.Errorf("summarizer caller not configured")
	}
	resp, err := s.caller.Call(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("summary call %s failed: %w", id, err)
	}
	return strings.TrimSpace(resp), nil
}

// mergeSummaries combines per-chunk summaries into a final summary. If the
// combined summaries are still large, it recursively summarizes them.
func (s *Summarizer) mergeSummaries(ctx context.Context, summaries []string) (string, error) {
	combined := strings.Builder{}
	for i, sum := range summaries {
		if sum == "" {
			continue
		}
		if i > 0 {
			combined.WriteString("\n\n---\n\n")
		}
		combined.WriteString(fmt.Sprintf("Chunk %d summary:\n%s", i+1, sum))
	}

	text := combined.String()
	if types.EstimateTokens(text) <= s.opts.MaxSummaryTokens || len(summaries) == 1 {
		return text, nil
	}

	// Recursively summarize the summaries until they fit.
	chunker := NewChunker(s.opts.MaxSummaryTokens)
	chunks := chunker.Split("merge", text)
	var next []string
	for _, c := range chunks {
		prompt := fmt.Sprintf(mergeSummaryPrompt, c.Content)
		merged, err := s.summarizeChunk(ctx, prompt, c.ID)
		if err != nil {
			return "", err
		}
		next = append(next, merged)
	}
	return s.mergeSummaries(ctx, next)
}
