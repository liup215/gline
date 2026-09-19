package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	llm "github.com/pkieltyka/go-llm"
	"github.com/pkieltyka/go-llm/providers/chatcompletions"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/pkg/types"
)

// GoLLMProvider wraps go-llm's Provider to implement gline's agent.Provider interface.
// This enables support for any OpenAI-compatible endpoint (OpenCode Go, OpenRouter, etc.)
type GoLLMProvider struct {
	provider llm.Provider
	model    string
	name     string

	// partialTool accumulates streamed tool-call fragments from go-llm.
	// go-llm emits ToolCallStart (id+name) followed by ToolCallDelta (args
	// fragments) and finally ToolCallEnd. We must not emit a complete call
	// until the final fragment has arrived.
	partialTool     *agent.ToolCall
	partialToolArgs strings.Builder
}

// NewGoLLMProvider creates a new provider using go-llm.
// baseURL should point to the chat/completions endpoint (e.g., "https://opencode.ai/zen/go/v1").
func NewGoLLMProvider(apiKey, model, baseURL, name string) (*GoLLMProvider, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required for go-llm provider")
	}
	if model == "" {
		return nil, fmt.Errorf("model is required for go-llm provider")
	}
	if name == "" {
		name = "go-llm"
	}

	// Build compat options with default headers
	compat := chatcompletions.Compat{
		StreamIncludeUsage:       true,
		InferMissingFinishReason: true,
	}

	// Add OpenCode Go specific headers
	if name == "opencode-go" {
		sessionID := fmt.Sprintf("gline-%d", time.Now().UnixMilli()%1000000)
		compat.DefaultHeaders = http.Header{
			"X-OpenCode-Session": {sessionID},
			"User-Agent":         {"gline/1.0"},
		}
	}

	// Create go-llm provider using chatcompletions engine
	provider, err := chatcompletions.New(baseURL,
		chatcompletions.WithAPIKey(apiKey),
		chatcompletions.WithCompat(compat),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create go-llm provider: %w", err)
	}

	log.Infof("go-llm provider created: name=%s, model=%s, baseURL=%s", name, model, baseURL)

	return &GoLLMProvider{
		provider: provider,
		model:    model,
		name:     name,
	}, nil
}

// GetProviderName returns the provider name
func (p *GoLLMProvider) GetProviderName() string {
	return p.name
}

// GetModel returns the current model
func (p *GoLLMProvider) GetModel() string {
	return p.model
}

// SupportsTools returns true (go-llm supports tool calling)
func (p *GoLLMProvider) SupportsTools() bool {
	return true
}

// CreateMessage sends a blocking chat request
func (p *GoLLMProvider) CreateMessage(ctx context.Context, req *agent.MessageRequest) (*agent.MessageResponse, error) {
	// Convert gline messages to go-llm messages
	messages := convertToGoLLMMessages(req.Messages)

	// Build go-llm request
	llmReq := &llm.Request{
		Model:        p.model,
		Messages:     messages,
		System:       req.SystemPrompt,
		MaxTokens:    req.MaxTokens,
	}

	// Add tools if provided
	if len(req.Tools) > 0 {
		llmReq.Tools = convertToGoLLMTools(req.Tools)
	}

	// Set tool choice
	switch req.ToolChoice {
	case agent.ToolChoiceRequired:
		llmReq.ToolChoice = llm.ToolChoice{Mode: llm.ToolChoiceRequired}
	case agent.ToolChoiceNone:
		llmReq.ToolChoice = llm.ToolChoice{Mode: llm.ToolChoiceNone}
	default:
		llmReq.ToolChoice = llm.ToolChoice{Mode: llm.ToolChoiceAuto}
	}

	// Set temperature
	if req.Temperature > 0 {
		temp := req.Temperature
		llmReq.Temperature = &temp
	}

	// Call go-llm
	resp, err := p.provider.Chat(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("go-llm chat error: %w", err)
	}

	// Convert response to gline format
	return convertFromGoLLMResponse(resp), nil
}

// CreateMessageStream sends a streaming chat request
func (p *GoLLMProvider) CreateMessageStream(ctx context.Context, req *agent.MessageRequest) (<-chan agent.StreamChunk, error) {
	// Convert gline messages to go-llm messages
	messages := convertToGoLLMMessages(req.Messages)

	// Build go-llm request
	llmReq := &llm.Request{
		Model:        p.model,
		Messages:     messages,
		System:       req.SystemPrompt,
		MaxTokens:    req.MaxTokens,
	}

	// Add tools if provided
	if len(req.Tools) > 0 {
		llmReq.Tools = convertToGoLLMTools(req.Tools)
	}

	// Set tool choice
	switch req.ToolChoice {
	case agent.ToolChoiceRequired:
		llmReq.ToolChoice = llm.ToolChoice{Mode: llm.ToolChoiceRequired}
	case agent.ToolChoiceNone:
		llmReq.ToolChoice = llm.ToolChoice{Mode: llm.ToolChoiceNone}
	default:
		llmReq.ToolChoice = llm.ToolChoice{Mode: llm.ToolChoiceAuto}
	}

	// Set temperature
	if req.Temperature > 0 {
		temp := req.Temperature
		llmReq.Temperature = &temp
	}

	// Create output channel
	ch := make(chan agent.StreamChunk, 100)

	// Debug: log the request being sent
	log.Infof("go-llm stream request: model=%s tools=%d toolChoice=%+v systemLen=%d", p.model, len(llmReq.Tools), llmReq.ToolChoice, len(llmReq.System))

	// Start streaming in goroutine
	go func() {
		defer close(ch)

		done := false
		for event, err := range p.provider.ChatStream(ctx, llmReq) {
			if err != nil {
				log.Warnf("go-llm stream error: %v", err)
				ch <- agent.StreamChunk{Error: err, Done: true}
				return
			}

			log.Infof("go-llm event: %T %+v", event, event)

			// Convert go-llm event to gline StreamChunk
			chunk := p.convertGoLLMEvent(event)
			if chunk != nil {
				if chunk.Done {
					done = true
				}
				ch <- *chunk
			}
		}

		// Ensure Done signal is sent if stream ended without MessageEnd
		if !done {
			ch <- agent.StreamChunk{Done: true}
		}
	}()

	return ch, nil
}

// --- Conversion functions ---

// convertToGoLLMMessages converts gline messages to go-llm messages
func convertToGoLLMMessages(msgs []types.Message) []llm.Message {
	result := make([]llm.Message, 0, len(msgs))

	for _, msg := range msgs {
		switch msg.Role {
		case types.RoleUser:
			result = append(result, llm.UserText(msg.Content))
		case types.RoleAssistant:
			// Create assistant message with parts
			parts := []llm.Part{llm.Text(msg.Content)}
			// Add tool calls if present
			for _, tc := range msg.ToolCalls {
				args := sanitizeToolCallArgs(tc.Input)
				parts = append(parts, llm.ToolCallPart{
					ID:   tc.ID,
					Name: tc.Name,
					Args: args,
				})
			}
			result = append(result, llm.Message{
				Role:  llm.RoleAssistant,
				Parts: parts,
			})
		case types.RoleTool:
			// Tool result message
			result = append(result, llm.Message{
				Role:  llm.RoleTool,
				Parts: []llm.Part{llm.ToolResult(msg.ToolCallID, msg.Content)},
			})
		}
	}

	return result
}

// convertToGoLLMTools converts gline tool definitions to go-llm tools
func convertToGoLLMTools(tools []agent.ToolDefinition) []llm.Tool {
	result := make([]llm.Tool, 0, len(tools))

	for _, tool := range tools {
		// Parse the input schema
		var schema map[string]interface{}
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			log.Warnf("Failed to parse tool schema for %s: %v", tool.Name, err)
			schema = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}

		result = append(result, llm.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schema,
		})
	}

	return result
}

// sanitizeToolCallArgs ensures tool call arguments are valid JSON.
// If the input is nil, empty, or invalid JSON, it returns an empty object.
// This prevents API errors when the LLM or database produces malformed args.
func sanitizeToolCallArgs(input json.RawMessage) json.RawMessage {
	if len(input) == 0 {
		return json.RawMessage(`{}`)
	}
	// Try to parse as JSON
	var v interface{}
	if err := json.Unmarshal(input, &v); err != nil {
		log.Warnf("sanitizeToolCallArgs: invalid JSON (%v), replacing with {}", err)
		return json.RawMessage(`{}`)
	}
	// Re-serialize to normalize formatting
	cleaned, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return cleaned
}

// convertFromGoLLMResponse converts go-llm response to gline response
func convertFromGoLLMResponse(resp *llm.Response) *agent.MessageResponse {
	result := &agent.MessageResponse{
		FinishReason: string(resp.StopReason),
	}

	// Extract content from parts
	for _, part := range resp.Parts {
		switch p := part.(type) {
		case llm.TextPart:
			result.Content += p.Text
		case llm.ReasoningPart:
			result.ReasoningContent += p.Text
		case llm.ToolCallPart:
			result.ToolCalls = append(result.ToolCalls, agent.ToolCall{
				ID:    p.ID,
				Name:  p.Name,
				Input: string(p.Args),
			})
		}
	}

	// Convert usage
	result.Usage = agent.TokenUsage{
		InputTokens:  int(resp.Usage.InputTokens),
		OutputTokens: int(resp.Usage.OutputTokens),
		TotalTokens:  int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
	}

	return result
}

// convertGoLLMEvent converts a go-llm streaming event to a gline StreamChunk.
// Tool-call fragments are accumulated on the provider and only emitted as a
// complete StreamChunk when llm.ToolCallEnd arrives.
func (p *GoLLMProvider) convertGoLLMEvent(event llm.Event) *agent.StreamChunk {
	switch e := event.(type) {
	case llm.MessageStart:
		return &agent.StreamChunk{
			Done: false,
		}
	case llm.TextDelta:
		return &agent.StreamChunk{
			Content: e.Text,
			Done:    false,
		}
	case llm.ReasoningDelta:
		return &agent.StreamChunk{
			ReasoningContent: e.Text,
			Done:             false,
		}
	case llm.ToolCallStart:
		p.partialTool = &agent.ToolCall{
			ID:   e.ID,
			Name: e.Name,
		}
		p.partialToolArgs.Reset()
		return &agent.StreamChunk{
			ToolCall:  p.partialTool,
			IsPartial: true,
			Done:      false,
		}
	case llm.ToolCallDelta:
		if p.partialTool != nil {
			p.partialToolArgs.WriteString(e.ArgsFragment)
		}
		return &agent.StreamChunk{
			ToolCall: &agent.ToolCall{
				Input: e.ArgsFragment,
			},
			IsPartial: true,
			Done:      false,
		}
	case llm.ToolCallEnd:
		if p.partialTool != nil {
			completed := agent.ToolCall{
				ID:    p.partialTool.ID,
				Name:  p.partialTool.Name,
				Input: p.partialToolArgs.String(),
			}
			p.partialTool = nil
			p.partialToolArgs.Reset()
			return &agent.StreamChunk{
				ToolCall:  &completed,
				IsPartial: false,
				Done:      false,
			}
		}
		return nil
	case llm.MessageEnd:
		chunk := &agent.StreamChunk{
			Done: true,
		}
		// Add usage if available
		if e.Usage.InputTokens > 0 || e.Usage.OutputTokens > 0 {
			chunk.Usage = agent.TokenUsage{
				InputTokens:  int(e.Usage.InputTokens),
				OutputTokens: int(e.Usage.OutputTokens),
				TotalTokens:  int(e.Usage.InputTokens + e.Usage.OutputTokens),
			}
		}
		return chunk
	}

	return nil
}

// Ensure GoLLMProvider implements agent.Provider
var _ agent.Provider = (*GoLLMProvider)(nil)
