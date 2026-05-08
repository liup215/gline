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
	anthropicAPIURL    = "https://api.anthropic.com/v1/messages"
	anthropicAPIVersion = "2023-06-01"
)

// AnthropicProvider implements the Provider interface for Anthropic's Claude API
type AnthropicProvider struct {
	apiKey  string
	model   string
	baseURL string
	httpClient *http.Client
}

// AnthropicMessage represents a message in Anthropic's format
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicTool represents a tool definition in Anthropic's format
type AnthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// AnthropicRequest represents the request body for Anthropic API
type AnthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Messages    []AnthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	Tools       []AnthropicTool    `json:"tools,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

// AnthropicResponse represents the response from Anthropic API
type AnthropicResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Content      []AnthropicContent `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage `json:"usage"`
}

// AnthropicContent represents a content block in the response
type AnthropicContent struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// AnthropicUsage represents token usage
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicError represents an error response
type AnthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	return &AnthropicProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: anthropicAPIURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// CreateMessage sends a message to the Anthropic API
func (p *AnthropicProvider) CreateMessage(ctx context.Context, req *agent.MessageRequest) (*agent.MessageResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Convert messages to Anthropic format
	anthropicMessages := make([]AnthropicMessage, 0, len(req.Messages))
	var systemPrompt string

	for _, msg := range req.Messages {
		switch msg.Role {
		case types.RoleSystem:
			systemPrompt = msg.Content
		case types.RoleUser:
			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role:    "user",
				Content: msg.Content,
			})
		case types.RoleAssistant:
			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role:    "assistant",
				Content: msg.Content,
			})
		case types.RoleTool:
			// Tool results are sent as user messages with special formatting
			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role:    "user",
				Content: fmt.Sprintf("Tool result: %s", msg.Content),
			})
		}
	}

	// Convert tools to Anthropic format
	anthropicTools := make([]AnthropicTool, 0, len(req.Tools))
	for _, tool := range req.Tools {
		anthropicTools = append(anthropicTools, AnthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}

	// Use system prompt from request or default
	if req.SystemPrompt != "" {
		systemPrompt = req.SystemPrompt
	}

	// Build request
	anthropicReq := AnthropicRequest{
		Model:       p.model,
		MaxTokens:   req.MaxTokens,
		Messages:    anthropicMessages,
		System:      systemPrompt,
		Tools:       anthropicTools,
		Temperature: req.Temperature,
	}

	// Set defaults
	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = 4096
	}
	if anthropicReq.Temperature == 0 {
		anthropicReq.Temperature = 0.0
	}

	// Serialize request
	jsonBody, err := json.Marshal(anthropicReq)
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
	httpReq.Header.Set("X-API-Key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

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
		var anthropicErr AnthropicError
		if err := json.Unmarshal(body, &anthropicErr); err == nil && anthropicErr.Message != "" {
			return nil, fmt.Errorf("Anthropic API error: %s", anthropicErr.Message)
		}
		return nil, fmt.Errorf("Anthropic API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to agent response
	return p.convertResponse(&anthropicResp), nil
}

// convertResponse converts Anthropic response to agent response
func (p *AnthropicProvider) convertResponse(resp *AnthropicResponse) *agent.MessageResponse {
	result := &agent.MessageResponse{
		Usage: agent.TokenUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		FinishReason: resp.StopReason,
	}

	// Extract content and tool calls
	var contentParts []string
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			contentParts = append(contentParts, block.Text)
		case "tool_use":
			// Convert tool use to ToolCall
			toolCall := agent.ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: string(block.Input),
			}
			result.ToolCalls = append(result.ToolCalls, toolCall)
		}
	}

	result.Content = ""
	if len(contentParts) > 0 {
		result.Content = contentParts[0]
	}

	return result
}

// SupportsTools returns true if the provider supports tool calling
func (p *AnthropicProvider) SupportsTools() bool {
	// Claude 3 models support tools
	supportedModels := []string{
		"claude-3-opus",
		"claude-3-sonnet",
		"claude-3-haiku",
		"claude-3-5-sonnet",
	}

	for _, prefix := range supportedModels {
		if len(p.model) >= len(prefix) && p.model[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// GetModel returns the current model name
func (p *AnthropicProvider) GetModel() string {
	return p.model
}

// GetProviderName returns the provider name
func (p *AnthropicProvider) GetProviderName() string {
	return "anthropic"
}

// SetHTTPClient sets a custom HTTP client (useful for testing)
func (p *AnthropicProvider) SetHTTPClient(client *http.Client) {
	p.httpClient = client
}

// CreateMessageStream sends a message to the Anthropic API and returns a stream of responses
func (p *AnthropicProvider) CreateMessageStream(ctx context.Context, req *agent.MessageRequest) (<-chan agent.StreamChunk, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Create the channel for streaming chunks
	chunkChan := make(chan agent.StreamChunk)

	// Build the request in a goroutine
	go func() {
		defer close(chunkChan)

		// Convert messages to Anthropic format
		anthropicMessages := make([]AnthropicMessage, 0, len(req.Messages))
		var systemPrompt string

		for _, msg := range req.Messages {
			switch msg.Role {
			case types.RoleSystem:
				systemPrompt = msg.Content
			case types.RoleUser:
				anthropicMessages = append(anthropicMessages, AnthropicMessage{
					Role:    "user",
					Content: msg.Content,
				})
			case types.RoleAssistant:
				anthropicMessages = append(anthropicMessages, AnthropicMessage{
					Role:    "assistant",
					Content: msg.Content,
				})
			case types.RoleTool:
				// Tool results are sent as user messages with special formatting
				anthropicMessages = append(anthropicMessages, AnthropicMessage{
					Role:    "user",
					Content: fmt.Sprintf("Tool result: %s", msg.Content),
				})
			}
		}

		// Convert tools to Anthropic format
		anthropicTools := make([]AnthropicTool, 0, len(req.Tools))
		for _, tool := range req.Tools {
			anthropicTools = append(anthropicTools, AnthropicTool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			})
		}

		// Use system prompt from request or default
		if req.SystemPrompt != "" {
			systemPrompt = req.SystemPrompt
		}

		// Build request
		anthropicReq := AnthropicRequest{
			Model:       p.model,
			MaxTokens:   req.MaxTokens,
			Messages:    anthropicMessages,
			System:      systemPrompt,
			Tools:       anthropicTools,
			Temperature: req.Temperature,
			Stream:      true,
		}

		// Set defaults
		if anthropicReq.MaxTokens == 0 {
			anthropicReq.MaxTokens = 4096
		}
		if anthropicReq.Temperature == 0 {
			anthropicReq.Temperature = 0.0
		}

		// Serialize request
		jsonBody, err := json.Marshal(anthropicReq)
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
		httpReq.Header.Set("X-API-Key", p.apiKey)
		httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

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
			var anthropicErr AnthropicError
			if err := json.Unmarshal(body, &anthropicErr); err == nil && anthropicErr.Message != "" {
				chunkChan <- agent.StreamChunk{
					Error: fmt.Errorf("Anthropic API error: %s", anthropicErr.Message),
					Done:  true,
				}
			} else {
				chunkChan <- agent.StreamChunk{
					Error: fmt.Errorf("Anthropic API returned status %d: %s", resp.StatusCode, string(body)),
					Done:  true,
				}
			}
			return
		}

		// Parse SSE stream
		reader := bufio.NewReader(resp.Body)
		var currentToolCall *agent.ToolCall

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					chunkChan <- agent.StreamChunk{Error: fmt.Errorf("error reading stream: %w", err), Done: true}
				}
				break
			}

			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "event: ") {
				continue
			}

			// Get event type
			eventType := strings.TrimPrefix(line, "event: ")

			// Read data line
			dataLine, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			dataLine = strings.TrimSpace(dataLine)
			if !strings.HasPrefix(dataLine, "data: ") {
				continue
			}
			data := strings.TrimPrefix(dataLine, "data: ")

			// Handle different event types
			switch eventType {
			case "content_block_delta":
				var delta struct {
					Delta struct {
						Type string `json:"type"`
						Text string `json:"text,omitempty"`
					} `json:"delta"`
				}
				if err := json.Unmarshal([]byte(data), &delta); err == nil && delta.Delta.Text != "" {
					chunkChan <- agent.StreamChunk{
						Content: delta.Delta.Text,
					}
				}

			case "content_block_start":
				var block struct {
					ContentBlock struct {
						Type string `json:"type"`
						ID   string `json:"id,omitempty"`
						Name string `json:"name,omitempty"`
					} `json:"content_block"`
				}
				if err := json.Unmarshal([]byte(data), &block); err == nil {
					if block.ContentBlock.Type == "tool_use" {
						currentToolCall = &agent.ToolCall{
							ID:   block.ContentBlock.ID,
							Name: block.ContentBlock.Name,
						}
					}
				}

			case "content_block_stop":
				if currentToolCall != nil {
					chunkChan <- agent.StreamChunk{
						ToolCall: currentToolCall,
					}
					currentToolCall = nil
				}

			case "message_delta":
				var msgDelta struct {
					Delta struct {
						StopReason string `json:"stop_reason"`
					} `json:"delta"`
				}
				if err := json.Unmarshal([]byte(data), &msgDelta); err == nil && msgDelta.Delta.StopReason != "" {
					chunkChan <- agent.StreamChunk{
						FinishReason: msgDelta.Delta.StopReason,
						Done:         true,
					}
					return
				}

			case "message_stop":
				chunkChan <- agent.StreamChunk{Done: true}
				return
			}
		}
	}()

	return chunkChan, nil
}
