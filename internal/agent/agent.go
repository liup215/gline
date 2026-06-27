// Package agent provides the core Agent functionality for gline.
// The Agent is responsible for managing the conversation loop with LLM providers,
// executing tools, and handling Plan/Act mode switching.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/memory"
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

	// GetToolRegistry returns the tool registry
	GetToolRegistry() *tools.Registry
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

	// Skills is the list of available skills for the system prompt.
	// Only metadata (name + description) is passed; actual skill contents
	// are loaded on-demand via the use_skill tool.
	Skills []types.SkillMeta

	// Store is the optional persistent storage for conversation history
	Store storage.Store

	// MemoryEngine is the optional unified memory engine ( facts + wiki + rag )
	MemoryEngine *memory.UnifiedEngine

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

	store        storage.Store        // persistent storage
	memoryEngine *memory.UnifiedEngine // optional unified memory engine
	taskID       string
	taskTitle    string
	workingDir   string

	skills []types.SkillMeta // available skills metadata for system prompt

	// Stream pre-dispatch: tool calls start executing while the SSE stream is still ongoing.
	pendingToolCallsMu     sync.Mutex
	pendingToolCalls       []ToolCall
	preDispatchedResultsMu sync.Mutex
	preDispatchedResults   map[string]preDispatchedResult
	// preDispatchedIDs tracks call IDs that were pre-dispatched during the stream
	// so the main loop can avoid calling OnToolCallStart twice for the same call.
	preDispatchedIDsMu sync.Mutex
	preDispatchedIDs   map[string]bool
}

// preDispatchedResult holds the outcome of a tool call that was launched
// early, while the SSE stream was still in flight.
type preDispatchedResult struct {
	result string
	err    error
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
		memoryEngine:           opts.MemoryEngine,
		taskTitle:              opts.Title,
		skills:                 opts.Skills,
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

		// Build system prompt with tool descriptions from registry (includes MCP tools)
		toolDescs := make([]prompts.ToolDescription, 0, len(availableTools))
		for _, tool := range availableTools {
			toolDescs = append(toolDescs, prompts.ToolDescription{
				Name:        tool.Name(),
				Description: tool.Description(),
				InputSchema: string(tool.InputSchema()),
			})
		}
		if a.mode == ModePlan {
			toolDescs = filterPlanModeTools(toolDescs)
		}
		systemPrompt := prompts.GetSystemPrompt(string(a.mode), toolDescs, a.customRules, a.skills)

		// ── Memory context injection (Phase 5) ────────────────────────────
		memoryCtx := a.buildMemoryContext(ctx, prompt)
		if memoryCtx != "" {
			systemPrompt += "\n\n" + memoryCtx
		}

		// Trim conversation if it exceeds token budget before sending.
		a.conversation.TrimToMaxTokens()

		// Auto-compact if tokens exceed 60% of the max context.
			// Earlier compaction prevents the token budget from growing too
			// large and keeps API latency stable across long conversations.
			a.AutoCompact()

		// Determine whether the assistant still has pending work.
		// We require tools when:
		//   - there is at least one non-completion/non-question tool available
		//   - the conversation is not already marked complete
		needsTool := a.needsTool(availableTools)

		// Pre-request token budget check. Ensure system prompt + messages +
		// tool descriptions fit inside the context window with room for the
		// response. Trigger compaction if we are close to the limit.
		a.enforceTokenBudget(ctx, systemPrompt, availableTools)

		// Create LLM request
		req := &MessageRequest{
			Messages:     a.conversation.GetMessages(),
			Tools:        convertTools(availableTools),
			SystemPrompt: systemPrompt,
		}
		// Use "auto" instead of "required" for reasoning models (DeepSeek/Kimi).
		// "required" can cause confused-stop when the model sees a complex system
		// prompt, because it does not know which specific tool is required.
		// "auto" lets the model choose based on context; if it wrongly chooses
		// not to call, the noToolsUsedMsg fallback below corrects it.
		if needsTool {
			req.ToolChoice = ToolChoiceAuto
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
					lastMsg.ToolChoice = string(req.ToolChoice)
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
				// Execute tools. Pre-dispatched results (computed while the stream
				// was still active) are used when available; otherwise tools run
				// synchronously here.
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

					// Only fire OnToolCallStart if this tool wasn't already
					// pre-dispatched during the stream (where the callback was
					// already triggered when the complete chunk arrived).
					a.preDispatchedIDsMu.Lock()
					wasPreDispatched := a.preDispatchedIDs[tc.ID]
					delete(a.preDispatchedIDs, tc.ID)
					a.preDispatchedIDsMu.Unlock()
					if !wasPreDispatched {
						callback.OnToolCallStart(ToolCall{
							ID:    tc.ID,
							Name:  tc.Name,
							Input: string(tc.Input),
						})
					}

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

					var result string
					var execErr error

					// Use pre-dispatched result if available.
					pre, ok := a.takePreDispatchResult(tc.ID)
					if ok {
						result = pre.result
						execErr = pre.err
					} else {
						// Execute the tool synchronously as fallback.
						result, execErr = tool.Execute(ctx, tc.Input)
						if execErr != nil {
							result = fmt.Sprintf("Error: %v", execErr)
						}
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
							if execErr != nil {
								if dbErr := a.store.FailToolCall(callID, execErr); dbErr != nil {
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

					// Special tools that can terminate the conversation.
					// Only attempt_completion (task done) or plan_mode_respond
					// (plan mode turn finished) mark the conversation as complete.
					// ask_followup_question must NOT mark it complete — the agent
					// needs to continue the loop after receiving the user's answer.
					switch tc.Name {
					case types.ToolAttemptCompletion.String(),
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

	// ── Post-conversation async fact extraction (Phase 4) ──────────────
	if a.memoryEngine != nil && a.conversation.IsComplete() {
		a.extractFactsAsync()
	}

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

// SetSkills updates the available skills metadata for the system prompt.
// This should be called when the skill registry changes (e.g. after installing
// a new skill).  The actual skill contents are loaded on-demand via the
// use_skill tool, not pre-injected into the system prompt.
func (a *BaseAgent) SetSkills(skills []types.SkillMeta) {
	a.skills = skills
}

// GetMemoryEngine returns the optional unified memory engine, or nil.
func (a *BaseAgent) GetMemoryEngine() *memory.UnifiedEngine {
	return a.memoryEngine
}

// ResetMemoryExtraction no-op (placeholder for backward compatibility).
func (a *BaseAgent) ResetMemoryExtraction() {}

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

// buildMemoryContext queries all four memory layers and builds an
// LLM-friendly context block. It is injected into the system prompt.
// This is designed to be fast (<150ms) because it only reads from SQLite.
func (a *BaseAgent) buildMemoryContext(ctx context.Context, prompt string) string {
	if a.memoryEngine == nil {
		return ""
	}
	
	// Use a short timeout to avoid blocking the conversation
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	
	var parts []string
	
	// 1. Fact layer (always include, lightweight)
	facts, err := a.memoryEngine.FactStore.Search(ctx, prompt, memory.FactSearchOptions{TopK: 5})
	if err == nil && len(facts) > 0 {
		var b strings.Builder
		b.WriteString("## Relevant Facts\n")
		for _, f := range facts {
			b.WriteString("- ")
			b.WriteString(f.Sentence())
			b.WriteString(fmt.Sprintf(" (confidence: %.2f)\n", f.Confidence))
		}
		parts = append(parts, b.String())
	}
	
	// 2. RAG layer (if documents exist in any KB)
	// Query each KB with default search
	kbs, _ := a.memoryEngine.ListKB(ctx)
	for _, kb := range kbs {
		if kb.ChunkCount == 0 {
			continue
		}
		vecs, err := memory.EmbedAndNormalize(ctx, a.memoryEngine.Embedder, []string{prompt})
		if err != nil {
			continue
		}
		chunks, err := a.memoryEngine.RAGEngine.Search(ctx, kb.ID, vecs[0], prompt, 3, 0.5)
		if err == nil && len(chunks) > 0 {
			var b strings.Builder
			b.WriteString(fmt.Sprintf("## Knowledge Base: %s\n", kb.Name))
			for _, c := range chunks {
				b.WriteString(fmt.Sprintf("> [from %s] %s...\n", c.DocID, truncate(c.Content, 200)))
			}
			parts = append(parts, b.String())
		}
		// Only search the first KB with results to save tokens
		break
	}
	
	if len(parts) == 0 {
		return ""
	}
	
	var result strings.Builder
	result.WriteString("═══ Memory Context ═══\n")
	result.WriteString("The following context may help you answer. Use it if relevant.\n\n")
	for _, p := range parts {
		result.WriteString(p)
		result.WriteString("\n")
	}
	return result.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// extractFactsAsync runs after a completed conversation to extract
// semantic facts in the background. It never blocks the caller.
func (a *BaseAgent) extractFactsAsync() {
	if a.memoryEngine == nil || a.memoryEngine.FactStore == nil || a.provider == nil {
		return
	}
	// Build a simple conversation transcript
	msgs := a.conversation.GetMessages()
	if len(msgs) < 2 {
		return
	}
	var b strings.Builder
	for _, m := range msgs {
		if m.Role == types.RoleUser {
			b.WriteString("User: ")
			b.WriteString(m.Content)
			b.WriteString("\n")
		} else if m.Role == types.RoleAssistant {
			b.WriteString("Assistant: ")
			b.WriteString(m.Content)
			b.WriteString("\n")
		}
	}
	transcript := b.String()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		extractor := memory.NewFactExtractor()
		extractor.Caller = func(ctx context.Context, systemPrompt, userContent string) (string, error) {
			req := &MessageRequest{
				Messages: []types.Message{
					{Role: types.RoleUser, Content: userContent},
				},
				SystemPrompt: systemPrompt,
				MaxTokens:    2048,
				Temperature:  0.0,
			}
			resp, err := a.provider.CreateMessage(ctx, req)
			if err != nil {
				return "", err
			}
			return resp.Content, nil
		}

		changes, err := extractor.Extract(ctx, transcript)
		if err != nil || len(changes) == 0 {
			return
		}
		source := memory.ConversationRef{TaskID: a.taskID}.String()
		changes = memory.EnrichFacts(changes, source, "")
		if err := a.memoryEngine.FactStore.Apply(ctx, changes); err != nil {
			log.Warnf("fact extraction persist failed: %v", err)
		} else {
			log.Infof("fact extraction: %d facts applied (task=%s)", len(changes), a.taskID)
		}
	}()
}

// enforceTokenBudget checks whether the upcoming request would exceed the
// conversation budget and, if so, triggers compaction. It estimates tokens
// for the system prompt and the JSON-serialized tool descriptions because
// those are also sent to the provider.
func (a *BaseAgent) enforceTokenBudget(ctx context.Context, systemPrompt string, availableTools []tools.Tool) {
	conv := a.conversation
	if conv.MaxTokens <= 0 {
		return
	}

	budget := conv.MaxTokens
	if conv.ResponseBuffer > 0 {
		budget -= conv.ResponseBuffer
	}
	if budget <= 0 {
		budget = conv.MaxTokens
	}

	// Estimate current conversation tokens.
	total := conv.GetTotalTokens()

	// Add system prompt tokens.
	total += types.EstimateTokens(systemPrompt)

	// Add approximate tool description tokens.
	if len(availableTools) > 0 {
		toolJSON, err := json.Marshal(availableTools)
		if err == nil {
			total += types.EstimateTokens(string(toolJSON))
		}
	}

	if total <= budget*6/10 {
		return
	}

	log.Infof("Token budget at %d/%d; triggering compaction", total, conv.MaxTokens)
	conv.AutoCompact()

	if conv.GetTotalTokens() > budget*8/10 {
		conv.TrimToMaxTokens()
	}
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

		log.Debugf("processStream chunk: content=%d reasoning=%d toolCall=%v done=%v",
			len(chunk.Content), len(chunk.ReasoningContent), chunk.ToolCall != nil, chunk.Done)

		// Accumulate real token usage from the API whenever available
		if chunk.Usage.TotalTokens > 0 {
			a.conversation.AddActualTokens(chunk.Usage.InputTokens, chunk.Usage.OutputTokens)
		}

		if chunk.Done {
			break
		}

		// Handle content (actual assistant reply text)
		if chunk.Content != "" {
			content.WriteString(chunk.Content)
			callback.OnContent(chunk.Content)
		}

		// Handle reasoning content (internal/model thinking).
		// Cline keeps this strictly separate from assistant text and exposes
		// it via a dedicated callback so the UI can choose to show it in a
		// collapsible panel or hide it completely.  We never mix reasoning
		// into OnContent.
		if chunk.ReasoningContent != "" {
			reasoning.WriteString(chunk.ReasoningContent)
			callback.OnReasoning(chunk.ReasoningContent)
		}

		// Handle tool call
		if chunk.ToolCall != nil {
			if chunk.IsPartial {
				// Partial tool calls from provider are already accumulated
				// Provider sends copies, so we don't need to track state here
				// Just ignore partials in processStream
			} else {
				// Complete tool call received
				tc := *chunk.ToolCall
				toolCalls = append(toolCalls, tc)

				// Notify UI that a tool call has been detected in the stream.
				callback.OnToolCallStart(tc)

				// Track in pending list (used by tests / diagnostics).
				a.pendingToolCallsMu.Lock()
				a.pendingToolCalls = append(a.pendingToolCalls, tc)
				a.pendingToolCallsMu.Unlock()

				// Mark this call ID as pre-dispatched so the main loop skips
				// calling OnToolCallStart a second time.
				a.preDispatchedIDsMu.Lock()
				if a.preDispatchedIDs == nil {
					a.preDispatchedIDs = make(map[string]bool)
				}
				a.preDispatchedIDs[tc.ID] = true
				a.preDispatchedIDsMu.Unlock()

				// Pre-dispatch: launch tool execution in the background so it
				// can overlap with the remaining SSE stream.
				go a.preDispatchToolCall(ctx, tc)
			}
		}
	}

	// Build the final content strings after the stream is complete.
	fullContent := content.String()

	// Fallback: if no native tool_calls were received but the content contains
	// XML-style tool calls (<tool_name>...params...</tool_name>), parse them.
	if len(toolCalls) == 0 {
		availableTools := convertTools(a.toolRegistry.GetAll())
		parsedXML := ParseXMLToolCalls(fullContent, availableTools)
		if len(parsedXML) > 0 {
			log.Infof("Fallback: parsed %d XML tool calls from assistant content", len(parsedXML))
			toolCalls = append(toolCalls, parsedXML...)
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
	a.conversation.AddMessage(types.Message{
		Role:             types.RoleAssistant,
		Content:          fullContent,
		ReasoningContent: reasoning.String(),
		ToolCalls:        typesToolCalls,
	})

	return nil
}

// preDispatchToolCall executes a tool in the background while the SSE stream
// is still ongoing and stores the result so the main loop can reuse it.
func (a *BaseAgent) preDispatchToolCall(ctx context.Context, tc ToolCall) {
	if a.abort {
		return
	}

	// Validate before executing.
	if !a.toolRegistry.IsAllowed(string(a.mode), tc.Name) {
		a.recordPreDispatch(tc.ID, "", fmt.Errorf("tool %s is not allowed in %s mode", tc.Name, a.mode))
		return
	}

	tool, err := a.toolRegistry.Get(tc.Name)
	if err != nil {
		a.recordPreDispatch(tc.ID, "", fmt.Errorf("tool '%s' not found: %v", tc.Name, err))
		return
	}

	// Skip pre-dispatch for ask_followup_question because it needs the
	// callback handler injected by the main loop.
	if _, isAsk := tool.(*tools.AskFollowupQuestionTool); isAsk {
		return
	}

	// Skip pre-dispatch for tools with side effects (writes, DB updates,
	// command execution, browser automation) to avoid race conditions and
	// duplicate actions. Browser must not background-run due to resource cost.
	switch tc.Name {
	case "kb_ingest", "write_to_file", "replace_in_file", "execute_command", "memory_note", "browser_copy":
		return
	}

	res, execErr := tool.Execute(ctx, []byte(tc.Input))
	if execErr != nil {
		res = fmt.Sprintf("Error: %v", execErr)
	}
	a.recordPreDispatch(tc.ID, res, execErr)
}

// recordPreDispatch stores the outcome of a pre-dispatched tool call.
func (a *BaseAgent) recordPreDispatch(callID string, result string, execErr error) {
	a.preDispatchedResultsMu.Lock()
	defer a.preDispatchedResultsMu.Unlock()
	if a.preDispatchedResults == nil {
		a.preDispatchedResults = make(map[string]preDispatchedResult)
	}
	a.preDispatchedResults[callID] = preDispatchedResult{result: result, err: execErr}
}

// takePreDispatchResult retrieves and removes a pre-dispatched result for the
// given call ID. If no pre-dispatch was recorded it returns ok=false so the
// caller falls back to normal execution.
func (a *BaseAgent) takePreDispatchResult(callID string) (preDispatchedResult, bool) {
	a.preDispatchedResultsMu.Lock()
	defer a.preDispatchedResultsMu.Unlock()
	if a.preDispatchedResults == nil {
		return preDispatchedResult{}, false
	}
	r, ok := a.preDispatchedResults[callID]
	if ok {
		delete(a.preDispatchedResults, callID)
	}
	return r, ok
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

// filterPlanModeTools filters out act-only tools for plan mode
func filterPlanModeTools(tools []prompts.ToolDescription) []prompts.ToolDescription {
	actOnlyTools := map[string]bool{
		"write_to_file":   true,
		"replace_in_file": true,
		"execute_command": true,
	}

	var filtered []prompts.ToolDescription
	for _, tool := range tools {
		if !actOnlyTools[tool.Name] {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}