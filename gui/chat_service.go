package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/prompts"
	"github.com/liup215/gline/internal/slash"
	"github.com/liup215/gline/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// pickProjectDir opens a directory picker dialog.
// Returns the selected directory path; empty string means cancelled.
func (c *ChatService) pickProjectDir() (string, error) {
	if c.app == nil {
		return "", fmt.Errorf("app not ready")
	}
	selected, err := c.app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle("Select project directory").
		PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	if selected == "" {
		return "", nil
	}
	if err := os.Chdir(selected); err != nil {
		return "", fmt.Errorf("failed to change directory: %w", err)
	}
	c.workingDir = selected
	// Sync to agent so CreateTask stores the correct working directory
	if c.backend.ag != nil {
		c.backend.ag.(*agent.BaseAgent).SetWorkingDir(selected)
	}
	return selected, nil
}

// StartNewConversation resets the conversation and clears the working directory.
// Used by New Chat button and /newtask to start a completely fresh session.
func (c *ChatService) StartNewConversation() {
	c.workingDir = ""
	if c.backend.ag != nil {
		c.backend.ag.(*agent.BaseAgent).ResetTask()
		c.backend.ag.(*agent.BaseAgent).SetWorkingDir("")
		c.backend.ag.GetConversation().Clear()
	}
}

// SelectProjectDir opens a directory picker and sets the working directory without resetting conversation.
// If a task is currently active, it also updates the task's working_dir in the database.
func (c *ChatService) SelectProjectDir() (string, error) {
	dir, err := c.pickProjectDir()
	if err != nil || dir == "" {
		return dir, err
	}
	// Update the database record if a task is loaded
	if c.backend.ag != nil {
		if taskID := c.backend.ag.(*agent.BaseAgent).GetTaskID(); taskID != "" {
			if err := c.backend.store.UpdateTaskWorkingDir(taskID, dir); err != nil {
				log.Warnf("Failed to update task working_dir: %v", err)
			}
		}
	}
	return dir, nil
}

// ChatService exposes chat operations to the Wails front-end.
type ChatService struct {
	app         *application.App
	backend     *Backend
	cmdReg      *slash.Registry
	cancelFn    context.CancelFunc
	followupCh  chan string
	mu          sync.Mutex
	workingDir  string // user-selected project directory; empty means not selected yet
	agentDone   chan struct{} // closed when the agent goroutine exits
}

// InitSlashRegistry initialises the slash command registry for this service.
func (c *ChatService) InitSlashRegistry() {
	c.cmdReg = slash.NewRegistry()
	ctx := &slash.CommandContext{
		ReloadRules: func() (int, string, error) {
			if c.backend.ag == nil {
				return 0, "", fmt.Errorf("agent not initialised")
			}
			baseAg, ok := c.backend.ag.(*agent.BaseAgent)
			if !ok {
				return 0, "", fmt.Errorf("agent type mismatch")
			}
			_, infos, err := baseAg.ReloadCustomRules()
			if err != nil {
				return 0, "", err
			}
			return len(infos), prompts.FormatRulesInfo(infos), nil
		},
	}
	for _, cmd := range slash.DefaultCommands(ctx) {
		c.cmdReg.Register(cmd)
	}
}

// SetApp sets the application reference (called after app creation).
func (c *ChatService) SetApp(app *application.App) {
	c.app = app
}

// GetSlashCommands returns all available slash commands.
func (c *ChatService) GetSlashCommands() []SlashCommandInfo {
	if c.cmdReg == nil {
		return nil
	}
	cmds := c.cmdReg.GetAll()
	result := make([]SlashCommandInfo, 0, len(cmds))
	for _, cmd := range cmds {
		result = append(result, SlashCommandInfo{
			Name:        cmd.Name,
			Description: cmd.Description,
			Section:     string(cmd.Section),
		})
	}
	return result
}

// ExecuteSlashCommand runs a slash command and returns the result.
func (c *ChatService) ExecuteSlashCommand(name string, args string) (*SlashActionResult, error) {
	if c.cmdReg == nil {
		return nil, fmt.Errorf("slash commands not initialised")
	}
	_, ok := c.cmdReg.Get(name)
	if !ok {
		return nil, fmt.Errorf("unknown command: /%s", name)
	}

	var capturedAction slash.CommandResult
	var capturedMessage string

	ctx := &slash.CommandContext{
		OnResult: func(result slash.CommandResult, message string) {
			capturedAction = result
			capturedMessage = message
		},
		ReloadRules: func() (int, string, error) {
			if c.backend.ag == nil {
				return 0, "", fmt.Errorf("agent not initialised")
			}
			baseAg, ok := c.backend.ag.(*agent.BaseAgent)
			if !ok {
				return 0, "", fmt.Errorf("agent type mismatch")
			}
			_, infos, err := baseAg.ReloadCustomRules()
			if err != nil {
				return 0, "", err
			}
			return len(infos), prompts.FormatRulesInfo(infos), nil
		},
	}

	// Re-register commands with fresh context to capture result
	reg := slash.NewRegistry()
	for _, c := range slash.DefaultCommands(ctx) {
		reg.Register(c)
	}
	freshCmd, ok := reg.Get(name)
	if !ok {
		return nil, fmt.Errorf("command not found: /%s", name)
	}

	_, err := freshCmd.Handler(args)
	if err != nil {
		return nil, err
	}

	actionStr := commandResultToString(capturedAction)
	return &SlashActionResult{
		Action:  actionStr,
		Message: capturedMessage,
	}, nil
}

// FilterSlashCommands returns commands matching the given prefix.
func (c *ChatService) FilterSlashCommands(prefix string) []SlashCommandInfo {
	if c.cmdReg == nil {
		return nil
	}
	filtered := c.cmdReg.Filter(prefix)
	result := make([]SlashCommandInfo, 0, len(filtered))
	for _, cmd := range filtered {
		result = append(result, SlashCommandInfo{
			Name:        cmd.Name,
			Description: cmd.Description,
			Section:     string(cmd.Section),
		})
	}
	return result
}

// IsSlashCommand checks if text is a standalone slash command.
func (c *ChatService) IsSlashCommand(text string) bool {
	return slash.IsStandaloneCommand(text)
}

// ParseSlashCommand extracts name and args from slash text.
func (c *ChatService) ParseSlashCommand(text string) (string, string) {
	return slash.ParseCommand(text)
}

// BuildHelpText returns formatted help for slash commands.
func (c *ChatService) BuildHelpText() string {
	if c.cmdReg == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("### 🛠️ Available Slash Commands\n\n")
	b.WriteString("| Command | Description |\n")
	b.WriteString("|---------|-------------|\n")
	for _, cmd := range c.cmdReg.GetAll() {
		b.WriteString(fmt.Sprintf("| **/%s** | %s |\n", cmd.Name, cmd.Description))
	}
	b.WriteString("\n### ⌨️ Shortcuts\n\n")
	b.WriteString("| Shortcut | Action |\n")
	b.WriteString("|----------|--------|\n")
	b.WriteString("| Tab | Toggle Plan/Act mode |\n")
	b.WriteString("| Ctrl+N | New conversation |\n")
	b.WriteString("| Ctrl+K | Focus input |\n")
	b.WriteString("| Ctrl+B | Toggle sidebar |\n")
	return b.String()
}

func commandResultToString(r slash.CommandResult) string {
	switch r {
	case slash.ResultClearScreen:
		return "clear"
	case slash.ResultNewTask:
		return "newtask"
	case slash.ResultCompact:
		return "compact"
	case slash.ResultShowHelp:
		return "help"
	case slash.ResultShowHistory:
		return "history"
	case slash.ResultReloadRules:
		return "reload"
	case slash.ResultQuit:
		return "quit"
	default:
		return "none"
	}
}

// ReloadRules reloads custom rules from disk and updates the agent.
// Returns the number of rule files loaded and a formatted description.
func (c *ChatService) ReloadRules() (int, string, error) {
	if c.backend.ag == nil {
		return 0, "", fmt.Errorf("agent not initialised")
	}
	baseAg, ok := c.backend.ag.(*agent.BaseAgent)
	if !ok {
		return 0, "", fmt.Errorf("agent type mismatch")
	}
	_, infos, err := baseAg.ReloadCustomRules()
	if err != nil {
		return 0, "", err
	}
	return len(infos), prompts.FormatRulesInfo(infos), nil
}

// GetRulesInfo returns metadata about available custom rule files.
func (c *ChatService) GetRulesInfo() ([]prompts.RuleFileInfo, error) {
	return prompts.GetCustomRulesInfo()
}

// GetConfig returns the current configuration
func (c *ChatService) GetConfig() (string, error) {
	return c.backend.GetConfig()
}

// UpdateConfig updates a config key
func (c *ChatService) UpdateConfig(key string, value string) error {
	return c.backend.UpdateConfig(key, value)
}

// ListTasks returns conversation history
func (c *ChatService) ListTasks(limit int, offset int) ([]storage.TaskRecord, error) {
	return c.backend.ListTasks(limit, offset)
}

// GetTaskSummary returns a task with its messages
func (c *ChatService) GetTaskSummary(taskID string) (*storage.TaskRecord, []storage.MessageRecord, error) {
	return c.backend.GetTaskSummary(taskID)
}

// DeleteTask deletes a task and its messages
func (c *ChatService) DeleteTask(taskID string) error {
	return c.backend.DeleteTask(taskID)
}
func (c *ChatService) SendMessage(prompt string) error {
	if c.backend.ag == nil {
		return fmt.Errorf("agent not initialised")
	}
	if c.app == nil {
		return fmt.Errorf("app not ready")
	}

	c.mu.Lock()
	// Cancel any previous run first.
	if c.cancelFn != nil {
		c.cancelFn()
	}
	// Wait for the previous goroutine to really exit so that
	// agent.running is cleared.
	if c.agentDone != nil {
		c.mu.Unlock()
		select {
		case <-c.agentDone:
		// exited cleanly
		case <-time.After(5 * time.Second):
			// forced continue — old goroutine may still be stuck on I/O
		}
		c.mu.Lock()
	}

	// Drain old followup channel
	if c.followupCh != nil {
		select {
		case <-c.followupCh:
		default:
		}
	}
	c.followupCh = make(chan string, 1)
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFn = cancel
	c.agentDone = make(chan struct{})
	c.mu.Unlock()

	cb := &guiStreamCallback{app: c.app, svc: c}

	go func(done chan struct{}) {
		defer close(done)
		if err := c.backend.ag.RunWithCallback(ctx, prompt, cb); err != nil {
			if ctx.Err() != context.Canceled {
				c.app.Event.Emit("chat:error", err.Error())
			}
		}
	}(c.agentDone)

	return nil
}

// AnswerFollowupQuestion sends the user's answer back to a pending AskFollowupQuestion call.
func (c *ChatService) AnswerFollowupQuestion(answer string) error {
	c.mu.Lock()
	ch := c.followupCh
	c.mu.Unlock()
	if ch == nil {
		return fmt.Errorf("no followup question pending")
	}
	ch <- answer
	return nil
}

// StopMessage aborts the current agent run.
func (c *ChatService) StopMessage() {
	c.mu.Lock()
	if c.cancelFn != nil {
		c.cancelFn()
		c.cancelFn = nil
	}
	if c.followupCh != nil {
		close(c.followupCh)
		c.followupCh = nil
	}
	c.mu.Unlock()
	if c.backend.ag != nil {
		c.backend.ag.Abort()
	}
	// Wait briefly for agent goroutine to finish so that subsequent
	// SendMessage does not trip the "already running" guard.
	c.mu.Lock()
	if c.agentDone != nil {
		c.mu.Unlock()
		select {
		case <-c.agentDone:
		case <-time.After(5 * time.Second):
		}
	} else {
		c.mu.Unlock()
	}
}

// ClearConversation clears the conversation and resets the task,
// but preserves the working directory so the user stays in the same project.
// Used by /clear slash command.
func (c *ChatService) ClearConversation() {
	if c.backend.ag != nil {
		c.backend.ag.(*agent.BaseAgent).ResetTask()
		c.backend.ag.GetConversation().Clear()
	}
}

// LoadTask restores agent state for an existing task.
func (c *ChatService) LoadTask(taskID string) (*storage.TaskRecord, error) {
	task, err := c.backend.LoadTask(taskID)
	if err != nil {
		return nil, err
	}
	if task != nil {
		c.workingDir = task.WorkingDir
	}
	return task, nil
}

// GetMode returns the current agent mode ("plan" or "act").
func (c *ChatService) GetMode() string {
	if c.backend.ag == nil {
		return "act"
	}
	return string(c.backend.ag.GetMode())
}

// SetMode switches the agent between plan and act modes.
func (c *ChatService) SetMode(mode string) error {
	if c.backend.ag == nil {
		return fmt.Errorf("agent not initialised")
	}
	switch mode {
	case "plan", "act":
		return c.backend.ag.SetMode(agent.Mode(mode))
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}
}

// CompactConversation triggers manual compaction of the conversation history.
func (c *ChatService) CompactConversation() (bool, error) {
	if c.backend.ag == nil {
		return false, fmt.Errorf("agent not initialised")
	}
	compacted := c.backend.ag.Compact()
	return compacted, nil
}

// GetStatus returns current provider, model, working directory, mode and token usage.
func (c *ChatService) GetStatus() (map[string]string, error) {
	cfg := c.backend.cfg.Get()
	provider := cfg.Provider.Default
	if provider == "" {
		provider = "openai"
	}
	model := ""
	maxTokens := 0
	switch provider {
	case "openai":
		model = cfg.Provider.OpenAI.Model
		maxTokens = cfg.Provider.OpenAI.MaxContextTokens
	}
	cwd := c.workingDir
	mode := "act"
	currentTokens := "0"
	maxTokensStr := fmt.Sprintf("%d", maxTokens)
	if c.backend.ag != nil {
		conv := c.backend.ag.GetConversation()
		if conv != nil {
			mode = string(c.backend.ag.GetMode())
			currentTokens = fmt.Sprintf("%d", conv.GetTotalTokens())
			if maxTokens == 0 {
				maxTokensStr = fmt.Sprintf("%d", conv.MaxTokens)
			}
		}
	}
	return map[string]string{
		"provider":      provider,
		"model":         model,
		"cwd":           cwd,
		"mode":          mode,
		"currentTokens": currentTokens,
		"maxTokens":     maxTokensStr,
	}, nil
}

// GetConversationState returns the current messages in JSON form.
func (c *ChatService) GetConversationState() string {
	if c.backend.ag == nil {
		return "[]"
	}
	conv := c.backend.ag.GetConversation()
	msgs := conv.GetMessages()
	type msgView struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	views := make([]msgView, len(msgs))
	for i, m := range msgs {
		views[i] = msgView{Role: string(m.Role), Content: m.Content}
	}
	data, _ := json.Marshal(views)
	return string(data)
}

// guiStreamCallback implements agent.StreamCallback and forwards events to the Wails front-end.
type guiStreamCallback struct {
	app *application.App
	svc *ChatService
}

func (g *guiStreamCallback) OnContent(delta string) {
	g.app.Event.Emit("chat:content", delta)
}

func (g *guiStreamCallback) OnReasoning(delta string) {
	g.app.Event.Emit("chat:reasoning", delta)
}

func (g *guiStreamCallback) OnStreamStart() {
	g.app.Event.Emit("chat:streamStart", "")
}

func (g *guiStreamCallback) OnToolCallStart(toolCall agent.ToolCall) {
	g.app.Event.Emit("chat:toolStart", map[string]string{
		"id":    toolCall.ID,
		"name":  toolCall.Name,
		"input": toolCall.Input,
	})
}

func (g *guiStreamCallback) OnToolCallComplete(toolCall agent.ToolCall, result string) {
	g.app.Event.Emit("chat:toolComplete", map[string]interface{}{
		"id":     toolCall.ID,
		"name":   toolCall.Name,
		"result": result,
	})
	// Special tools whose result should be shown directly to the user
	// (not hidden behind a tool-call badge).
	switch toolCall.Name {
	case "plan_mode_respond":
		if result != "" {
			g.app.Event.Emit("chat:systemMessage", map[string]interface{}{
				"role":    "system",
				"content": result,
			})
		}
	case "attempt_completion":
		if result != "" {
			g.app.Event.Emit("chat:systemMessage", map[string]interface{}{
				"role":    "system",
				"content": "📋 " + result,
			})
		}
	case "ask_followup_question":
		// The question itself is already emitted via chat:followupQuestion
		// but we also show the answer prompt as a system message so the user
		// sees the question inline without needing the popup.
		if result != "" {
			g.app.Event.Emit("chat:systemMessage", map[string]interface{}{
				"role":    "system",
				"content": "💬 " + result,
			})
		}
	}
}

func (g *guiStreamCallback) AskFollowupQuestion(question string, options []string) (string, error) {
	g.svc.mu.Lock()
	ch := g.svc.followupCh
	g.svc.mu.Unlock()
	if ch == nil {
		return "", fmt.Errorf("no followup channel")
	}
	g.app.Event.Emit("chat:followupQuestion", map[string]interface{}{
		"question": question,
		"options":  options,
	})
	select {
	case answer := <-ch:
		return answer, nil
	case <-time.After(30 * time.Minute):
		return "", fmt.Errorf("followup timeout")
	}
}

func (g *guiStreamCallback) OnError(err error) {
	g.app.Event.Emit("chat:error", err.Error())
}

func (g *guiStreamCallback) OnComplete() {
	g.app.Event.Emit("chat:complete", "")
}

func (g *guiStreamCallback) OnTaskCreated(taskID string) {
	g.app.Event.Emit("chat:taskCreated", taskID)
}
