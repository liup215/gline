// Package agent provides the core Agent functionality for gline.
// The Agent is responsible for managing the conversation loop with LLM providers,
// executing tools, and handling Plan/Act mode switching.
package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
)

// Mode represents the operating mode of the Agent
type Mode string

const (
	// ModePlan is for exploration and planning without making changes
	ModePlan Mode = "plan"
	// ModeAct is for executing tasks and modifying files
	ModeAct Mode = "act"
)

// Agent is the core interface for the AI programming assistant
type Agent interface {
	// Run starts the Agent with a user prompt
	// This will initiate the conversation loop with the LLM
	Run(ctx context.Context, prompt string) error

	// SetMode switches between Plan and Act modes
	SetMode(mode Mode) error

	// GetMode returns the current operating mode
	GetMode() Mode

	// Abort stops the current execution
	Abort()

	// IsRunning returns true if the Agent is currently processing
	IsRunning() bool

	// GetConversation returns the current conversation state
	GetConversation() *types.Conversation
}

// Options contains configuration options for creating an Agent
type Options struct {
	// Provider is the LLM provider to use
	Provider Provider

	// ToolRegistry contains all available tools
	ToolRegistry *tools.Registry

	// Mode is the initial operating mode
	Mode Mode

	// AutoApprove enables automatic approval of tool calls (yolo mode)
	AutoApprove bool

	// MaxConsecutiveMistakes limits consecutive errors before stopping
	MaxConsecutiveMistakes int
}

// BaseAgent implements the Agent interface
type BaseAgent struct {
	provider     Provider
	toolRegistry *tools.Registry
	mode         Mode
	conversation *types.Conversation

	running                bool
	abort                  bool
	autoApprove            bool
	maxConsecutiveMistakes int
	consecutiveMistakes    int
}

// New creates a new Agent instance with the given options
func New(opts Options) (*BaseAgent, error) {
	if opts.Provider == nil {
		return nil, fmt.Errorf("provider is required")
	}

	if opts.ToolRegistry == nil {
		return nil, fmt.Errorf("tool registry is required")
	}

	mode := opts.Mode
	if mode == "" {
		mode = ModeAct
	}

	maxMistakes := opts.MaxConsecutiveMistakes
	if maxMistakes == 0 {
		maxMistakes = 3
	}

	return &BaseAgent{
		provider:               opts.Provider,
		toolRegistry:           opts.ToolRegistry,
		mode:                   mode,
		conversation:           types.NewConversation(),
		autoApprove:            opts.AutoApprove,
		maxConsecutiveMistakes: maxMistakes,
	}, nil
}

// Run starts the Agent with a user prompt
func (a *BaseAgent) Run(ctx context.Context, prompt string) error {
	if a.running {
		return fmt.Errorf("agent is already running")
	}

	a.running = true
	a.abort = false
	defer func() { a.running = false }()

	// Add user message to conversation
	a.conversation.AddMessage(types.Message{
		Role:    types.RoleUser,
		Content: prompt,
	})

	// Main conversation loop
	for !a.abort {
		// Get available tools for current mode
		availableTools := a.toolRegistry.GetForMode(string(a.mode))

		// Create LLM request
		req := &MessageRequest{
			Messages: a.conversation.GetMessages(),
			Tools:    convertTools(availableTools),
		}

		// Send to LLM
		resp, err := a.provider.CreateMessage(ctx, req)
		if err != nil {
			a.consecutiveMistakes++
			if a.consecutiveMistakes >= a.maxConsecutiveMistakes {
				return fmt.Errorf("max consecutive mistakes reached: %w", err)
			}
			continue
		}

		a.consecutiveMistakes = 0

		// Process response
		if err := a.processResponse(ctx, resp); err != nil {
			return err
		}

		// Check if conversation is complete
		if a.conversation.IsComplete() {
			break
		}
	}

	return nil
}

// processResponse handles the LLM response
func (a *BaseAgent) processResponse(ctx context.Context, resp *MessageResponse) error {
	// Convert ToolCalls from agent format to types format
	var toolCalls []types.ToolCall
	for _, tc := range resp.ToolCalls {
		toolCalls = append(toolCalls, types.ToolCall{
			ID:    tc.ID,
			Name:  tc.Name,
			Input: []byte(tc.Input),
		})
	}

	// Add assistant message to conversation
	a.conversation.AddMessage(types.Message{
		Role:      types.RoleAssistant,
		Content:   resp.Content,
		ToolCalls: toolCalls,
	})

	// Handle tool calls
	for _, tc := range resp.ToolCalls {
		if a.abort {
			return nil
		}

		// Check if tool is allowed in current mode
		if !a.toolRegistry.IsAllowed(string(a.mode), tc.Name) {
			return fmt.Errorf("tool %s is not allowed in %s mode", tc.Name, a.mode)
		}

		// Execute tool
		tool, err := a.toolRegistry.Get(tc.Name)
		if err != nil {
			return fmt.Errorf("tool not found: %s", tc.Name)
		}

		// Parse input
		var input json.RawMessage
		if err := json.Unmarshal([]byte(tc.Input), &input); err != nil {
			return fmt.Errorf("failed to parse tool input: %w", err)
		}

		// Execute
		result, err := tool.Execute(ctx, input)
		if err != nil {
			result = fmt.Sprintf("Error: %v", err)
		}

		// Add tool result to conversation
		a.conversation.AddMessage(types.Message{
			Role:       types.RoleTool,
			ToolCallID: tc.ID,
			Content:    result,
		})
	}

	return nil
}

// SetMode switches between Plan and Act modes
func (a *BaseAgent) SetMode(mode Mode) error {
	if mode != ModePlan && mode != ModeAct {
		return fmt.Errorf("invalid mode: %s", mode)
	}

	a.mode = mode
	return nil
}

// GetMode returns the current operating mode
func (a *BaseAgent) GetMode() Mode {
	return a.mode
}

// Abort stops the current execution
func (a *BaseAgent) Abort() {
	a.abort = true
}

// IsRunning returns true if the Agent is currently processing
func (a *BaseAgent) IsRunning() bool {
	return a.running
}

// GetConversation returns the current conversation state
func (a *BaseAgent) GetConversation() *types.Conversation {
	return a.conversation
}

// convertTools converts internal tool definitions to provider format
func convertTools(toolsList []tools.Tool) []ToolDefinition {
	defs := make([]ToolDefinition, len(toolsList))
	for i, t := range toolsList {
		defs[i] = ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		}
	}
	return defs
}
