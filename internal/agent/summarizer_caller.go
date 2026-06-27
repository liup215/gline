package agent

import (
	"context"

	"github.com/liup215/gline/internal/summarizer"
	"github.com/liup215/gline/pkg/types"
)

// ProviderSummarizerCaller adapts an agent.Provider to the summarizer.Caller
// interface. It performs a direct single-turn LLM call, so it does not spawn
// additional tool-using subagents.
type ProviderSummarizerCaller struct {
	provider Provider
}

// NewSummarizerCaller creates a summarizer.Caller backed by the given provider.
func NewSummarizerCaller(provider Provider) summarizer.Caller {
	return &ProviderSummarizerCaller{provider: provider}
}

// Call implements summarizer.Caller.
func (c *ProviderSummarizerCaller) Call(ctx context.Context, prompt string) (string, error) {
	if c.provider == nil {
		return "", nil
	}
	req := &MessageRequest{
		Messages:  []types.Message{{Role: types.RoleUser, Content: prompt}},
		MaxTokens: 1024,
	}
	resp, err := c.provider.CreateMessage(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
