// Package agent provides the core Agent functionality for gline.
// The Agent is responsible for managing the conversation loop with LLM providers,
// executing tools, and handling Plan/Act mode switching.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/prompts"
	"github.com/liup215/gline/internal/storage"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
)

// noToolsUsedMsg is injected into the conversation when the assistant returns a
// response without calling any tools while tool-calling is required.
const noToolsUsedMsg = `[ERROR] You did not use a tool in your previous response.
When you have a task to perform, you MUST use one of the available tools.
Only calling attempt_completion, ask_followup_question, or plan_mode_respond can end the conversation.
Please review the task and call the appropriate tool(s).`

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

	// Compact manually compacts the conversation history.
	Compact() bool

	// ReloadCustomRules reloads custom rules from disk and returns true if any rules were loaded.
	ReloadCustomRules() (bool, []prompts.RuleFileInfo, error)
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

	// CustomRules is extra text appended to the system prompt
	CustomRules string

	// Store is the optional persistent storage for conversation history
	Store storage.Store

	// Title is the optional task title (used when creating a new task)
	Title string

	// MaxTokens is the maximum context tokens for the conversation (0 = default 128k)
	MaxTokens int
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
	customRules            string

	store     storage.Store // persistent storage
	taskID    string        // current task ID
	taskTitle string        // current task title
	workingDir string      // project working directory
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

	conv := types.NewConversation()
	if opts.MaxTokens > 0 {
		conv.MaxTokens = opts.MaxTokens
	}

	return &BaseAgent{
		provider:               opts.Provider,
		toolRegistry:           opts.ToolRegistry,
		mode:                   mode,
		conversation:           conv,
		autoApprove:            opts.AutoApprove,
		maxConsecutiveMistakes: maxMistakes,
		customRules:            opts.CustomRules,
		store:                  opts.Store,
		taskTitle:              opts.Title,
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
	// Ensure running is always reset, even on panic.
	defer func() {
		a.running = false
		// Also clear the abort flag so subsequent runs start fresh.
		a.abort = false
	}()

	// Each new user turn must reopen the conversation loop.
	a.conversation.MarkIncomplete()

	// Add user message to conversation
	a.conversation.AddMessage(types.Message{
		Role:    types.RoleUser,
		Content: prompt,
	})

	// --- Storage: create task and save user message ---
	if a.store != nil {
		if a.taskID == "" {
			providerName := a.provider.GetProviderName()
			model := a.provider.GetModel()
			id, err := a.store.CreateTask(a.taskTitle, prompt, string(a.mode), providerName, model, a.workingDir)
			if err != nil {
				log.Warnf("Failed to create task record: %v", err)
			} else {
				a.taskID = id
				callback.OnTaskCreated(id)
			}
		}
		if a.taskID != "" {
			if lastMsg := a.conversation.GetLastMessage(); lastMsg != nil {
				if err := a.store.SaveMessage(a.taskID, *lastMsg); err != nil {
					log.Warnf("Failed to save user message: %v", err)
				}
			}
		}
	}

	// Main conversation loop
	for !a.abort {
		// Get available tools for current mode
		availableTools := a.toolRegistry.GetForMode(string(a.mode))

		// Build system prompt with tool descriptions
		toolDescs := prompts.GetToolDescriptions()
		if a.mode == ModePlan {
			toolDescs = prompts.GetPlanModeToolDescriptions()
		}
		systemPrompt := prompts.GetSystemPrompt(string(a.mode), toolDescs, a.customRules)

		// Trim conversation if it exceeds token budget before sending.
		a.conversation.TrimToMaxTokens()

		// Auto-compact if tokens exceed 80% of the max context
		a.AutoCompact()

		// Determine whether the assistant still has pending work.
		// We require tools when:
		//   - there is at least one non-completion/non-question tool available
		//   - the conversation is not already marked complete
		needsTool := a.needsTool(availableTools)

		// Create LLM request
		req := &MessageRequest{
			Messages:     a.conversation.GetMessages(),
			Tools:        convertTools(availableTools),
			SystemPrompt: systemPrompt,
		}
		if needsTool {
			req.ToolChoice = ToolChoiceRequired
		}
		log.Infof("RunWithCallback: availableTools=%d, needsTool=%v, toolChoice=%s", len(availableTools), needsTool, req.ToolChoice)

		// Serialize available tool names for diagnostic storage. We attach this
		// to the assistant message record so users can verify whether the
		// request actually contained tools when debugging empty tool_calls.
		var availableToolsJSON json.RawMessage
		if len(availableTools) > 0 {
			names := make([]string, 0, len(availableTools))
			for _, t := range availableTools {
				names = append(names, t.Name())
			}
			b, err := json.Marshal(names)
			if err != nil {
				log.Warnf("Failed to marshal available tools: %v", err)
			} else {
				availableToolsJSON = b
			}
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
			if a.store != nil && a.taskID != "" {
				if dbErr := a.store.FailTask(a.taskID, err.Error()); dbErr != nil {
					log.Warnf("Failed to mark task as failed: %v", dbErr)
				}
			}
			callback.OnError(err)
			return err
		}

		// Save assistant message to storage, attaching the list of tools that
		// were available to the assistant so users can diagnose why tool_calls
		// may be empty.
		if a.store != nil && a.taskID != "" {
			if msgs := a.conversation.GetMessages(); len(msgs) > 0 {
				lastMsg := msgs[len(msgs)-1]
				if lastMsg.Role == types.RoleAssistant {
					lastMsg.AvailableTools = availableToolsJSON
					if err := a.store.SaveMessage(a.taskID, lastMsg); err != nil {
						log.Warnf("Failed to save assistant message: %v", err)
					}
				}
			}
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

					// Check if tool is allowed in current mode
					if !a.toolRegistry.IsAllowed(string(a.mode), tc.Name) {
						errorMsg := fmt.Sprintf("Error: Tool '%s' is not allowed in %s mode.", tc.Name, a.mode)
						a.conversation.AddMessage(types.Message{
							Role:       types.RoleTool,
							ToolCallID: tc.ID,
							Content:    errorMsg,
						})
						callback.OnToolCallComplete(ToolCall{
							ID:    tc.ID,
							Name:  tc.Name,
							Input: string(tc.Input),
						}, errorMsg)
						continue
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

					// Record tool call start in storage
					var callID int64
					if a.store != nil && a.taskID != "" {
						cid, err := a.store.StartToolCall(a.taskID, tc.Name, tc.Input)
						if err != nil {
							log.Warnf("Failed to record tool call start: %v", err)
						} else {
							callID = cid
						}
					}

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

					// Save tool result message and complete tool call record
					if a.store != nil && a.taskID != "" {
						if lastMsg := a.conversation.GetLastMessage(); lastMsg != nil {
							if dbErr := a.store.SaveMessage(a.taskID, *lastMsg); dbErr != nil {
								log.Warnf("Failed to save tool result message: %v", dbErr)
							}
						}
						if callID > 0 {
							if err != nil {
								if dbErr := a.store.FailToolCall(callID, err); dbErr != nil {
									log.Warnf("Failed to record tool call failure: %v", dbErr)
								}
							} else {
								if dbErr := a.store.CompleteToolCall(callID, result); dbErr != nil {
									log.Warnf("Failed to record tool call completion: %v", dbErr)
								}
							}
						}
					}

					// Notify callback that tool is complete
					callback.OnToolCallComplete(ToolCall{
						ID:    tc.ID,
						Name:  tc.Name,
						Input: string(tc.Input),
					}, result)

					// Special tools that can terminate the conversation
					switch tc.Name {
					case types.ToolAttemptCompletion.String(),
						types.ToolAskFollowupQuestion.String(),
						types.ToolPlanModeRespond.String():
						a.conversation.SetComplete()
					}
				}
			}
		}

		// If the conversation is not complete but the last assistant message has
		// no tool calls while tools are still needed, inject the no-tools-used
		// reminder so the next loop iteration forces the model to call a tool.
		if !a.conversation.IsComplete() && !a.abort {
			messages := a.conversation.GetMessages()
			if len(messages) > 0 {
				lastMsg := messages[len(messages)-1]
				if lastMsg.Role == types.RoleAssistant && len(lastMsg.ToolCalls) == 0 && needsTool {
					a.consecutiveMistakes++
					a.conversation.AddMessage(types.Message{
						Role:    types.RoleUser,
						Content: noToolsUsedMsg,
					})
					continue // go to next loop iteration instead of falling through
				}
			}
		}

		// Check if conversation is complete
		if a.conversation.IsComplete() || a.abort {
			if a.store != nil && a.taskID != "" {
				status := "completed"
				if a.abort {
					status = "failed"
				}
				if status == "completed" {
					if err := a.store.CompleteTask(a.taskID); err != nil {
						log.Warnf("Failed to mark task as completed: %v", err)
					}
				} else {
					if err := a.store.FailTask(a.taskID, "aborted"); err != nil {
						log.Warnf("Failed to mark task as failed: %v", err)
					}
				}
			}
			break
		}
	}

	callback.OnComplete()
	return nil
}

// ResetTask clears the current task ID so the next RunWithCallback creates a new task.
// This is called by /newtask slash command to start a fresh conversation.
func (a *BaseAgent) ResetTask() {
	a.taskID = ""
}

// SetTaskTitle sets the title for the next task to be created.
func (a *BaseAgent) SetTaskTitle(title string) {
	a.taskTitle = title
}

// Compact manually compacts the conversation by removing oldest messages,
// keeping the system prompt and the most recent conversation turns.
// Returns true if compaction actually occurred.
func (a *BaseAgent) Compact() bool {
	before := a.conversation.MessageCount()
	a.conversation.AutoCompact()
	after := a.conversation.MessageCount()
	compacted := after < before
	if compacted {
		log.Infof("Compacted conversation: %d -> %d messages, %d tokens", before, after, a.conversation.GetTotalTokens())
	}
	return compacted
}

// needsTool determines whether the assistant should be forced to call at
// least one tool.
//
// Returns false when the only available tools are conversation-ending
// tools (attempt_completion, ask_followup_question, plan_mode_respond) or
// when there are no tools at all.
func (a *BaseAgent) needsTool(toolsList []tools.Tool) bool {
	for _, t := range toolsList {
		name := t.Name()
		if name != types.ToolAttemptCompletion.String() &&
			name != types.ToolAskFollowupQuestion.String() &&
			name != types.ToolPlanModeRespond.String() {
			return true
		}
	}
	return false
}

// AutoCompact checks whether current token usage exceeds 80% of the
// max context window and, if so, triggers a compaction.
func (a *BaseAgent) AutoCompact() {
	if a.conversation.IsTokenAboveThreshold(80) {
		a.Compact()
	}
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
		// No tool calls from a non-streaming response.
		// Do NOT mark as complete here; the main loop will inject noToolsUsedMsg
		// if there is still pending work.
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

// GetStore returns the persistent storage (may be nil).
func (a *BaseAgent) GetStore() storage.Store {
	return a.store
}

// SetStore sets the persistent storage.
func (a *BaseAgent) SetStore(s storage.Store) {
	a.store = s
}

// GetTaskID returns the current task ID.
func (a *BaseAgent) GetTaskID() string {
	return a.taskID
}

// SetTaskID sets the current task ID (used when resuming a task).
func (a *BaseAgent) SetTaskID(id string) {
	a.taskID = id
}

// GetWorkingDir returns the current working directory.
func (a *BaseAgent) GetWorkingDir() string {
	return a.workingDir
}

// SetWorkingDir sets the working directory.
func (a *BaseAgent) SetWorkingDir(dir string) {
	a.workingDir = dir
}

// ReloadCustomRules reloads custom rules from disk and updates the agent's customRules field.
// Returns true if any rules were loaded, along with metadata about the loaded files.
func (a *BaseAgent) ReloadCustomRules() (bool, []prompts.RuleFileInfo, error) {
	content, infos, err := prompts.LoadCustomRulesWithMeta()
	if err != nil {
		return false, nil, err
	}
	a.customRules = content
	return content != "", infos, nil
}

// GetCustomRulesInfo returns metadata about available rule files without reloading content.
func (a *BaseAgent) GetCustomRulesInfo() ([]prompts.RuleFileInfo, error) {
	return prompts.GetCustomRulesInfo()
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

		// Accumulate real token usage from the API whenever available
		if chunk.Usage.TotalTokens > 0 {
			a.conversation.AddActualTokens(chunk.Usage.InputTokens, chunk.Usage.OutputTokens)
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

	// Add assistant message to conversation, including any accumulated reasoning content.
	// Tool calls are stored in the ToolCalls field; the TUI renders them via OnToolCallStart/Complete callbacks.
	fullContent := content.String()
	a.conversation.AddMessage(types.Message{
		Role:             types.RoleAssistant,
		Content:          fullContent,
		ReasoningContent: reasoning.String(),
		ToolCalls:        typesToolCalls,
	})

	return nil
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