package agent

import (
	"context"
	"encoding/json"

	"github.com/liup215/gline/pkg/types"
)

// Provider is the interface for LLM providers
type Provider interface {
	// CreateMessage sends a message to the LLM and returns the response
	CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error)

	// SupportsTools returns true if the provider supports tool/function calling
	SupportsTools() bool

	// GetModel returns the current model name
	GetModel() string

	// GetProviderName returns the provider name (e.g., "anthropic", "openai")
	GetProviderName() string
}

// MessageRequest represents a request to the LLM
type MessageRequest struct {
	// Messages is the conversation history
	Messages []types.Message

	// Tools is the list of available tools
	Tools []ToolDefinition

	// SystemPrompt is the system prompt to use
	SystemPrompt string

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int

	// Temperature controls randomness (0.0 - 1.0)
	Temperature float64
}

// MessageResponse represents a response from the LLM
type MessageResponse struct {
	// Content is the text content of the response
	Content string

	// ToolCalls contains any tool calls requested by the LLM
	ToolCalls []ToolCall

	// Usage contains token usage information
	Usage TokenUsage

	// FinishReason indicates why the response finished
	FinishReason string
}

// ToolCall represents a tool call requested by the LLM
type ToolCall struct {
	// ID is the unique identifier for this tool call
	ID string

	// Name is the name of the tool to call
	Name string

	// Input is the JSON input for the tool
	Input string
}

// ToolDefinition defines a tool that can be called by the LLM
type ToolDefinition struct {
	// Name is the tool name
	Name string

	// Description describes what the tool does
	Description string

	// InputSchema is the JSON schema for the tool's input parameters
	InputSchema json.RawMessage
}

// TokenUsage contains information about token usage
type TokenUsage struct {
	// InputTokens is the number of tokens in the input
	InputTokens int

	// OutputTokens is the number of tokens in the output
	OutputTokens int

	// TotalTokens is the total number of tokens
	TotalTokens int
}

// ProviderConfig contains common configuration for providers
type ProviderConfig struct {
	// APIKey is the API key for the provider
	APIKey string

	// Model is the model to use
	Model string

	// BaseURL is the base URL for the API (optional, for custom endpoints)
	BaseURL string

	// MaxTokens is the maximum tokens to generate
	MaxTokens int

	// Temperature controls randomness
	Temperature float64

	// Timeout is the request timeout in seconds
	Timeout int
}

// DefaultProviderConfig returns a default provider configuration
func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		MaxTokens:   4096,
		Temperature: 0.0,
		Timeout:     120,
	}
}
