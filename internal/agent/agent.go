// Package agent provides the core Agent functionality for gline.
// The Agent is responsible for managing the conversation loop with LLM providers,
// executing tools, and handling Plan/Act mode switching.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liup215/gline/internal/prompts"
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

	// RunWithCallback starts the Agent with a user prompt and a callback for streaming updates
	// This is used for TUI mode to receive real-time updates
	RunWithCallback(ctx context.Context, prompt string, callback StreamCallback) error

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
	// Use the no-op adapter for non-streaming scenarios
	return a.RunWithCallback(ctx, prompt, &StreamCallbackAdapter{})
}

// RunWithCallback starts the Agent with a user prompt and a callback for streaming updates
func (a *BaseAgent) RunWithCallback(ctx context.Context, prompt string, callback StreamCallback) error {
	if a.running {
		return fmt.Errorf("agent is already running")
	}

	a.running = true
	a.abort = false
	defer func() { a.running = false }()

	// Each new user turn must reopen the conversation loop.
	a.conversation.MarkIncomplete()

	// Add user message to conversation
	a.conversation.AddMessage(types.Message{
		Role:    types.RoleUser,
		Content: prompt,
	})

	// Main conversation loop
	for !a.abort {
		// Get available tools for current mode
		availableTools := a.toolRegistry.GetForMode(string(a.mode))

		// Build system prompt with tool descriptions
		toolDescs := prompts.GetToolDescriptions()
		if a.mode == ModePlan {
			toolDescs = prompts.GetPlanModeToolDescriptions()
		}
		systemPrompt := prompts.GetSystemPrompt(string(a.mode), toolDescs)

		// Create LLM request
		req := &MessageRequest{
			Messages:     a.conversation.GetMessages(),
			Tools:        convertTools(availableTools),
			SystemPrompt: systemPrompt,
		}

		// Use streaming API
		streamChan, err := a.provider.CreateMessageStream(ctx, req)
		if err != nil {
			a.consecutiveMistakes++
			callback.OnError(err)
			if a.consecutiveMistakes >= a.maxConsecutiveMistakes {
				return fmt.Errorf("max consecutive mistakes reached: %w", err)
			}
			continue
		}

		a.consecutiveMistakes = 0

		// Notify callback that a new stream is starting
		callback.OnStreamStart()

		// Process the stream
		if err := a.processStream(ctx, streamChan, callback); err != nil {
			callback.OnError(err)
			return err
		}

		// Execute any tool calls from the last assistant message
		messages := a.conversation.GetMessages()
		if len(messages) > 0 {
			lastMsg := messages[len(messages)-1]
			if lastMsg.Role == types.RoleAssistant && len(lastMsg.ToolCalls) > 0 {
				// Execute tools
				for _, tc := range lastMsg.ToolCalls {
					if a.abort {
						break
					}

					// Get the tool from registry
					tool, err := a.toolRegistry.Get(tc.Name)
					if err != nil {
						errorMsg := fmt.Sprintf("Error: Tool '%s' not found: %v", tc.Name, err)
						a.conversation.AddMessage(types.Message{
							Role:       types.RoleTool,
							ToolCallID: tc.ID,
							Content:    errorMsg,
						})
						continue
					}

					// Notify callback that tool is starting
					callback.OnToolCallStart(ToolCall{
						ID:    tc.ID,
						Name:  tc.Name,
						Input: string(tc.Input),
					})

					// If this is the ask_followup_question tool and the callback supports AskFollowupQuestion,
					// inject the TUI/Callback handler so the tool doesn't read directly from stdin.
					if askTool, ok := tool.(*tools.AskFollowupQuestionTool); ok {
						askTool.SetHandler(func(question string, options []string) (string, error) {
							return callback.AskFollowupQuestion(question, options)
						})
					}
					// Execute the tool
					result, err := tool.Execute(ctx, tc.Input)
					if err != nil {
						result = fmt.Sprintf("Error: %v", err)
					}

					// Add tool result to conversation
					a.conversation.AddMessage(types.Message{
						Role:       types.RoleTool,
						ToolCallID: tc.ID,
						Content:    result,
					})

					// Notify callback that tool is complete
					callback.OnToolCallComplete(ToolCall{
						ID:    tc.ID,
						Name:  tc.Name,
						Input: string(tc.Input),
					}, result)
				}
			}
		}

		// Check if conversation is complete
		if a.conversation.IsComplete() {
			break
		}
	}

	callback.OnComplete()
	return nil
}

// processResponse handles the LLM response
func (a *BaseAgent) processResponse(ctx context.Context, resp *MessageResponse, callback StreamCallback) error {
	// Convert ToolCalls from agent format to types format
	var toolCalls []types.ToolCall
	for _, tc := range resp.ToolCalls {
		toolCalls = append(toolCalls, types.ToolCall{
			ID:    tc.ID,
			Name:  tc.Name,
			Input: []byte(tc.Input),
		})
	}

	// Add assistant message to conversation, include any reasoning_content the provider returned
	a.conversation.AddMessage(types.Message{
		Role:             types.RoleAssistant,
		Content:          resp.Content,
		ReasoningContent: resp.ReasoningContent,
		ToolCalls:        toolCalls,
	})

	// Handle tool calls
	if len(resp.ToolCalls) == 0 {
		// No tool calls, conversation is complete
		a.conversation.SetComplete()
		return nil
	}

	for _, tc := range resp.ToolCalls {
		if a.abort {
			return nil
		}

		// Check if tool is allowed in current mode
		if !a.toolRegistry.IsAllowed(string(a.mode), tc.Name) {
			return fmt.Errorf("tool %s is not allowed in %s mode", tc.Name, a.mode)
		}

		// Get the tool from registry
		tool, err := a.toolRegistry.Get(tc.Name)
		if err != nil {
			errorMsg := fmt.Sprintf("Error: Tool '%s' not found: %v", tc.Name, err)
			a.conversation.AddMessage(types.Message{
				Role:       types.RoleTool,
				ToolCallID: tc.ID,
				Content:    errorMsg,
			})
			continue
		}

		// Parse input
		var input json.RawMessage
		if err := json.Unmarshal([]byte(tc.Input), &input); err != nil {
			// Return the original input to LLM so it can retry with correct format
			errorMsg := fmt.Sprintf("Error: Invalid JSON in tool call '%s': %v. Please retry with properly formatted JSON arguments.\n\nOriginal input: %s", tc.Name, err, tc.Input)
			// Add tool result to conversation so LLM can see the error and retry
			a.conversation.AddMessage(types.Message{
				Role:       types.RoleTool,
				ToolCallID: tc.ID,
				Content:    errorMsg,
			})
			// Continue to next tool call instead of failing entirely
			continue
		}

		// If this is the ask_followup_question tool, inject the handler from the callback.
		if askTool, ok := tool.(*tools.AskFollowupQuestionTool); ok {
			askTool.SetHandler(func(question string, options []string) (string, error) {
				return callback.AskFollowupQuestion(question, options)
			})
		}

		// Execute the tool
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

// GetProvider returns the LLM provider
func (a *BaseAgent) GetProvider() Provider {
	return a.provider
}

// GetToolRegistry returns the tool registry
func (a *BaseAgent) GetToolRegistry() *tools.Registry {
	return a.toolRegistry
}

// processStream handles the streaming response from the LLM
func (a *BaseAgent) processStream(ctx context.Context, streamChan <-chan StreamChunk, callback StreamCallback) error {
	var content strings.Builder
	var reasoning strings.Builder
	var toolCalls []ToolCall

	for chunk := range streamChan {
		if chunk.Error != nil {
			return chunk.Error
		}

		if chunk.Done {
			break
		}

		// Handle content
		if chunk.Content != "" {
			content.WriteString(chunk.Content)
			callback.OnContent(chunk.Content)
		}

		// Handle reasoning content (internal/model thinking). Accumulate but don't mix into visible content.
		if chunk.ReasoningContent != "" {
			reasoning.WriteString(chunk.ReasoningContent)
			// Do not send reasoning to OnContent by default to avoid showing internal thoughts in the main UI.
			// If the UI wants to surface reasoning in the future, add a dedicated callback method.
		}

		// Handle tool call
		if chunk.ToolCall != nil {
			if chunk.IsPartial {
				// Partial tool calls from provider are already accumulated
				// Provider sends copies, so we don't need to track state here
				// Just ignore partials in processStream
			} else {
				// Complete tool call received
				toolCalls = append(toolCalls, *chunk.ToolCall)
				// Tool call status is communicated via OnToolCallStart/OnToolCallComplete
				// Do NOT mix tool call text into the content stream — this keeps
				// LLM text and tool status visually separated in the TUI.
			}
		}
	}

	// Convert accumulated tool calls to types.ToolCall
	var typesToolCalls []types.ToolCall
	for _, tc := range toolCalls {
		typesToolCalls = append(typesToolCalls, types.ToolCall{
			ID:    tc.ID,
			Name:  tc.Name,
			Input: []byte(tc.Input),
		})
	}

	// Add assistant message to conversation, including any accumulated reasoning content
	// Surface tool calls in the visible assistant content so tests and UIs can render them.
	toolText := formatToolCallText(toolCalls)
	fullContent := content.String()
	if toolText != "" {
		if fullContent != "" {
			fullContent = fullContent + "\n" + toolText
		} else {
			fullContent = toolText
		}
		// Also send tool text to callback so it's visible in the UI
		callback.OnContent("\n" + toolText)
	}
	a.conversation.AddMessage(types.Message{
		Role:             types.RoleAssistant,
		Content:          fullContent,
		ReasoningContent: reasoning.String(),
		ToolCalls:        typesToolCalls,
	})

	// If there are no tool calls, mark conversation as complete
	// Otherwise, the agent loop will continue to execute tools
	if len(typesToolCalls) == 0 {
		a.conversation.SetComplete()
	}

	return nil
}

func formatToolCallText(toolCalls []ToolCall) string {
	if len(toolCalls) == 0 {
		return ""
	}

	parts := make([]string, 0, len(toolCalls))
	for _, tc := range toolCalls {
		if tc.Name == "" {
			continue
		}

		input := strings.TrimSpace(tc.Input)
		if input == "" {
			input = "{}"
		}

		parts = append(parts, fmt.Sprintf("[tool:%s] %s", tc.Name, input))
	}

	return strings.Join(parts, "\n")
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