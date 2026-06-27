package subagent

import (
	"context"
	"errors"

	"github.com/liup215/gline/internal/summarizer"
)

// SubagentSummarizerCaller adapts a subagent.Builder to the summarizer.Caller
// interface. Each summary request is handled by an independent subagent run,
// which lets the chunk summarizer use tools if needed.
type SubagentSummarizerCaller struct {
	builder *Builder
}

// NewSummarizerCaller creates a summarizer.Caller that runs each chunk summary
// through a dedicated subagent.
func NewSummarizerCaller(builder *Builder) summarizer.Caller {
	return &SubagentSummarizerCaller{builder: builder}
}

// Call implements summarizer.Caller.
func (c *SubagentSummarizerCaller) Call(ctx context.Context, prompt string) (string, error) {
	if c.builder == nil {
		return "", nil
	}
	runner := NewRunner(c.builder)
	result := runner.Run(ctx, prompt, func(u ProgressUpdate) {})
	if result.Status != StatusCompleted {
		if result.Error != "" {
			return "", errors.New(result.Error)
		}
		return "", errors.New("subagent did not complete")
	}
	return result.Result, nil
}
