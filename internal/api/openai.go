// Package api provides LLM provider implementations
package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/pkg/types"
)

const (
	defaultOpenAIURL = "https://api.openai.com/v1/chat/completions"
)

// OpenAIProvider implements the Provider interface for OpenAI-compatible APIs
// This provider can be used with:
// - OpenAI official API
// - OpenRouter
// - Local models (Ollama, LM Studio, etc.)
// - Any OpenAI-compatible endpoint
type OpenAIProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// OpenAIMessage represents a message in OpenAI's format
type OpenAIMessage struct {
	Role       string             `json:"role"`
	Content    string             `json:"content"`
	ToolCalls  []OpenAIToolCall   `json:"tool_calls,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
	Name       string             `json:"name,omitempty"`
}

// OpenAIToolCall represents a tool call in OpenAI's format
type OpenAIToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function OpenAIFunction `json:"function"`
}

// OpenAIFunction represents a function call in OpenAI's format
type OpenAIFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAITool represents a tool definition in OpenAI's format
type OpenAITool struct {
	Type     string              `json:"type"`
	Function OpenAIToolFunction `json:"function"`
}

// OpenAIToolFunction represents the function definition
type OpenAIToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// OpenAIRequest represents the request body for OpenAI API
type OpenAIRequest struct {
	Model       string         `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Tools       []OpenAITool   `json:"tools,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
}

// OpenAIResponse represents the response from OpenAI API
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

// OpenAIStreamResponse represents a streaming response chunk from OpenAI API
type OpenAIStreamResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []OpenAIStreamChoice `json:"choices"`
}

// OpenAIStreamChoice represents a choice in the streaming response
type OpenAIStreamChoice struct {
	Index        int                `json:"index"`
	Delta        OpenAIStreamDelta  `json:"delta"`
	FinishReason string             `json:"finish_reason"`
}

// OpenAIStreamDelta represents the delta in a streaming response
type OpenAIStreamDelta struct {
	Role       string           `json:"role,omitempty"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []OpenAIStreamToolCall `json:"tool_calls,omitempty"`
}

// OpenAIStreamToolCall represents a tool call in a streaming response
type OpenAIStreamToolCall struct {
	Index    int             `json:"index"`
	ID       string          `json:"id,omitempty"`
	Type     string          `json:"type,omitempty"`
	Function OpenAIStreamFunction `json:"function,omitempty"`
}

// OpenAIStreamFunction represents a function call in a streaming response
type OpenAIStreamFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// OpenAIChoice represents a choice in the response
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIUsage represents token usage
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIError represents an error response
type OpenAIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewOpenAIProvider creates a new OpenAI-compatible provider
// Parameters:
//   - apiKey: API key for authentication
//   - model: Model name (e.g., "gpt-4", "gpt-3.5-turbo")
//   - baseURL: Custom base URL (optional, defaults to OpenAI official API)
func NewOpenAIProvider(apiKey, model, baseURL string) *OpenAIProvider {
	if model == "" {
		model = "gpt-4"
	}

	if baseURL == "" {
		baseURL = defaultOpenAIURL
	}

	return &OpenAIProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// CreateMessage sends a message to the OpenAI-compatible API
func (p *OpenAIProvider) CreateMessage(ctx context.Context, req *agent.MessageRequest) (*agent.MessageResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Convert messages to OpenAI format
	openaiMessages := make([]OpenAIMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		switch msg.Role {
		case types.RoleSystem:
			openaiMessages = append(openaiMessages, OpenAIMessage{
				Role:    "system",
				Content: msg.Content,
			})
		case types.RoleUser:
			openaiMessages = append(openaiMessages, OpenAIMessage{
				Role:    "user",
				Content: msg.Content,
			})
		case types.RoleAssistant:
			openaiMessages = append(openaiMessages, OpenAIMessage{
				Role:    "assistant",
				Content: msg.Content,
			})
		case types.RoleTool:
			// Tool results are sent as tool messages
			openaiMessages = append(openaiMessages, OpenAIMessage{
				Role:       "tool",
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
			})
		}
	}

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		// Insert system message at the beginning
		systemMsg := OpenAIMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		}
		openaiMessages = append([]OpenAIMessage{systemMsg}, openaiMessages...)
	}

	// Convert tools to OpenAI format
	openaiTools := make([]OpenAITool, 0, len(req.Tools))
	for _, tool := range req.Tools {
		openaiTools = append(openaiTools, OpenAITool{
			Type: "function",
			Function: OpenAIToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	// Build request
	openaiReq := OpenAIRequest{
		Model:       p.model,
		Messages:    openaiMessages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	// Only add tools if there are any
	if len(openaiTools) > 0 {
		openaiReq.Tools = openaiTools
	}

	// Set defaults
	if openaiReq.MaxTokens == 0 {
		openaiReq.MaxTokens = 4096
	}
	if openaiReq.Temperature == 0 {
		openaiReq.Temperature = 0.0
	}

	// Serialize request
	jsonBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var openaiErr OpenAIError
		if err := json.Unmarshal(body, &openaiErr); err == nil && openaiErr.Error.Message != "" {
			return nil, fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)", 
				openaiErr.Error.Message, openaiErr.Error.Type, openaiErr.Error.Code)
		}
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openaiResp OpenAIResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to agent response
	return p.convertResponse(&openaiResp), nil
}

// convertResponse converts OpenAI response to agent response
func (p *OpenAIProvider) convertResponse(resp *OpenAIResponse) *agent.MessageResponse {
	if len(resp.Choices) == 0 {
		return &agent.MessageResponse{
			Content:      "",
			FinishReason: "error",
			Usage: agent.TokenUsage{
				InputTokens:  resp.Usage.PromptTokens,
				OutputTokens: resp.Usage.CompletionTokens,
				TotalTokens:  resp.Usage.TotalTokens,
			},
		}
	}

	choice := resp.Choices[0]
	result := &agent.MessageResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: agent.TokenUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}

	// Convert tool calls
	if len(choice.Message.ToolCalls) > 0 {
		for _, tc := range choice.Message.ToolCalls {
			toolCall := agent.ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: tc.Function.Arguments,
			}
			result.ToolCalls = append(result.ToolCalls, toolCall)
		}
	}

	return result
}

// SupportsTools returns true if the provider supports tool calling
func (p *OpenAIProvider) SupportsTools() bool {
	// Most modern OpenAI models and compatible models support tools
	// Models that don't support tools: gpt-3.5-turbo-instruct, text-* models
	nonToolModels := []string{
		"gpt-3.5-turbo-instruct",
		"text-davinci",
		"text-curie",
		"text-babbage",
		"text-ada",
	}

	for _, prefix := range nonToolModels {
		if len(p.model) >= len(prefix) && p.model[:len(prefix)] == prefix {
			return false
		}
	}
	return true
}

// GetModel returns the current model name
func (p *OpenAIProvider) GetModel() string {
	return p.model
}

// GetProviderName returns the provider name
func (p *OpenAIProvider) GetProviderName() string {
	return "openai"
}

// SetHTTPClient sets a custom HTTP client (useful for testing)
func (p *OpenAIProvider) SetHTTPClient(client *http.Client) {
	p.httpClient = client
}

// GetBaseURL returns the current base URL
func (p *OpenAIProvider) GetBaseURL() string {
	return p.baseURL
}

// CreateMessageStream sends a message to the OpenAI-compatible API and returns a stream of responses
func (p *OpenAIProvider) CreateMessageStream(ctx context.Context, req *agent.MessageRequest) (<-chan agent.StreamChunk, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Create the channel for streaming chunks
	chunkChan := make(chan agent.StreamChunk)

	// Build the request in a goroutine
	go func() {
		defer close(chunkChan)

		// Convert messages to OpenAI format
		openaiMessages := make([]OpenAIMessage, 0, len(req.Messages))
		for _, msg := range req.Messages {
			switch msg.Role {
			case types.RoleSystem:
				openaiMessages = append(openaiMessages, OpenAIMessage{
					Role:    "system",
					Content: msg.Content,
				})
			case types.RoleUser:
				openaiMessages = append(openaiMessages, OpenAIMessage{
					Role:    "user",
					Content: msg.Content,
				})
			case types.RoleAssistant:
				openaiMessages = append(openaiMessages, OpenAIMessage{
					Role:    "assistant",
					Content: msg.Content,
				})
			case types.RoleTool:
				openaiMessages = append(openaiMessages, OpenAIMessage{
					Role:       "tool",
					Content:    msg.Content,
					ToolCallID: msg.ToolCallID,
				})
			}
		}

		// Add system prompt if provided
		if req.SystemPrompt != "" {
			systemMsg := OpenAIMessage{
				Role:    "system",
				Content: req.SystemPrompt,
			}
			openaiMessages = append([]OpenAIMessage{systemMsg}, openaiMessages...)
		}

		// Convert tools to OpenAI format
		openaiTools := make([]OpenAITool, 0, len(req.Tools))
		for _, tool := range req.Tools {
			openaiTools = append(openaiTools, OpenAITool{
				Type: "function",
				Function: OpenAIToolFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.InputSchema,
				},
			})
		}

		// Build request
		openaiReq := OpenAIRequest{
			Model:       p.model,
			Messages:    openaiMessages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
			Stream:      true,
		}

		// Only add tools if there are any
		if len(openaiTools) > 0 {
			openaiReq.Tools = openaiTools
		}

		// Set defaults
		if openaiReq.MaxTokens == 0 {
			openaiReq.MaxTokens = 4096
		}
		if openaiReq.Temperature == 0 {
			openaiReq.Temperature = 0.0
		}

		// Serialize request
		jsonBody, err := json.Marshal(openaiReq)
		if err != nil {
			chunkChan <- agent.StreamChunk{Error: fmt.Errorf("failed to marshal request: %w", err), Done: true}
			return
		}

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			chunkChan <- agent.StreamChunk{Error: fmt.Errorf("failed to create request: %w", err), Done: true}
			return
		}

		// Set headers
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

		// Send request
		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			chunkChan <- agent.StreamChunk{Error: fmt.Errorf("failed to send request: %w", err), Done: true}
			return
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			var openaiErr OpenAIError
			if err := json.Unmarshal(body, &openaiErr); err == nil && openaiErr.Error.Message != "" {
				chunkChan <- agent.StreamChunk{
					Error: fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
						openaiErr.Error.Message, openaiErr.Error.Type, openaiErr.Error.Code),
					Done: true,
				}
			} else {
				chunkChan <- agent.StreamChunk{
					Error: fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body)),
					Done:  true,
				}
			}
			return
		}

		// Parse SSE stream
		reader := bufio.NewReader(resp.Body)
		toolCalls := make(map[int]*agent.ToolCall)

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					chunkChan <- agent.StreamChunk{Error: fmt.Errorf("error reading stream: %w", err), Done: true}
				}
				break
			}

			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			// Extract JSON data
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				chunkChan <- agent.StreamChunk{Done: true}
				break
			}

			// Parse stream response
			var streamResp OpenAIStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) == 0 {
				continue
			}

			choice := streamResp.Choices[0]
			delta := choice.Delta

			// Handle content
			if delta.Content != "" {
				chunkChan <- agent.StreamChunk{
					Content: delta.Content,
				}
			}

			// Handle tool calls
			for _, tc := range delta.ToolCalls {
				if tc.Index >= 0 {
					if toolCalls[tc.Index] == nil {
						toolCalls[tc.Index] = &agent.ToolCall{
							ID:   tc.ID,
							Name: tc.Function.Name,
						}
					}
					// Accumulate arguments
					toolCalls[tc.Index].Input += tc.Function.Arguments
				}
			}

			// Check finish reason
			if choice.FinishReason != "" {
				// Send any complete tool calls
				for i := 0; i < len(toolCalls); i++ {
					if tc, ok := toolCalls[i]; ok && tc.ID != "" {
						chunkChan <- agent.StreamChunk{
							ToolCall: tc,
						}
					}
				}
				chunkChan <- agent.StreamChunk{
					FinishReason: choice.FinishReason,
					Done:         true,
				}
				break
			}
		}
	}()

	return chunkChan, nil
}
