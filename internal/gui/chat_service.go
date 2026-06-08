package gui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/memory"
	"github.com/liup215/gline/internal/prompts"
	"github.com/liup215/gline/internal/skills"
	"github.com/liup215/gline/internal/slash"
	"github.com/liup215/gline/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// pickProjectDir opens a directory picker dialog.
// Returns the selected directory path; empty string means cancelled.
func (c *ChatService) pickProjectDir() (string, error) {
	if c.App == nil {
		return "", fmt.Errorf("app not ready")
	}
	selected, err := c.App.Dialog.OpenFile().
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
	if c.Backend.ag != nil {
		c.Backend.ag.(*agent.BaseAgent).SetWorkingDir(selected)
	}
	return selected, nil
}

// StartNewConversation resets the conversation and clears the working directory.
// Used by New Chat button and /newtask to start a completely fresh session.
func (c *ChatService) StartNewConversation() {
	c.workingDir = ""
	if c.Backend.ag != nil {
		c.Backend.ag.(*agent.BaseAgent).ResetTask()
		c.Backend.ag.(*agent.BaseAgent).SetWorkingDir("")
		c.Backend.ag.GetConversation().Clear()
		// Note: Memory system reset removed (incompatible with current agent architecture)
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
	if c.Backend.ag != nil {
		if taskID := c.Backend.ag.(*agent.BaseAgent).GetTaskID(); taskID != "" {
			if err := c.Backend.store.UpdateTaskWorkingDir(taskID, dir); err != nil {
				log.Warnf("Failed to update task working_dir: %v", err)
			}
		}
	}
	return dir, nil
}

// ChatService exposes chat operations to the Wails front-end.
type ChatService struct {
	App         *application.App
	Backend     *Backend
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
			if c.Backend.ag == nil {
				return 0, "", fmt.Errorf("agent not initialised")
			}
			baseAg, ok := c.Backend.ag.(*agent.BaseAgent)
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

	// Register skill slash commands
	if c.Backend.skillRegistry != nil {
		skillCmds := skills.BuildSlashCommands(c.Backend.skillRegistry)
		for _, cmd := range skillCmds {
			c.cmdReg.Register(cmd)
		}
		// Keep the agent's skills metadata in sync with the registry.
		if c.Backend.ag != nil {
			if baseAg, ok := c.Backend.ag.(*agent.BaseAgent); ok {
				baseAg.SetSkills(c.Backend.skillRegistry.GetMeta())
			}
		}
	}
}

func (c *ChatService) memoryEngine() (*memory.UnifiedEngine, error) {
	if c.Backend == nil || c.Backend.ag == nil {
		return nil, fmt.Errorf("agent not initialised")
	}
	baseAg, ok := c.Backend.ag.(*agent.BaseAgent)
	if !ok {
		return nil, fmt.Errorf("agent type mismatch")
	}
	e := baseAg.GetMemoryEngine()
	if e == nil {
		return nil, fmt.Errorf("memory system not configured")
	}
	return e, nil
}

// ─── memoryService adapts ChatService to slash.MemoryService ───────────────

type memoryService struct {
	chat *ChatService
}

func (m *memoryService) engine() (*memory.UnifiedEngine, error) {
	return m.chat.memoryEngine()
}

func (m *memoryService) Note(content string) error {
	eng, err := m.engine()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = eng.FactStore.Add(ctx, content, memory.ConversationRef{})
	return err
}

func (m *memoryService) Recall(query string) (string, error) {
	eng, err := m.engine()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var parts []string
	// Fact layer
	if facts, err := eng.FactStore.Search(ctx, query, memory.FactSearchOptions{TopK: 5}); err == nil && len(facts) > 0 {
		parts = append(parts, "### Facts")
		for _, f := range facts {
			parts = append(parts, fmt.Sprintf("- **%s**: %s %s %s (%.0f%%)", f.Category, f.Subject, f.Predicate, f.Object, f.Confidence*100))
		}
	}
	// Wiki layer – scan all KB wiki directories for matching content
	wikis, _ := eng.ListKB(ctx)
	for _, kb := range wikis {
		wikiRoot := filepath.Join(memory.KBDir(kb.ID), "wiki")
		_ = filepath.Walk(wikiRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if strings.Contains(strings.ToLower(string(data)), strings.ToLower(query)) {
				rel, _ := filepath.Rel(wikiRoot, path)
				parts = append(parts, fmt.Sprintf("### Wiki: %s", rel))
				parts = append(parts, memory.Truncate(string(data), 200))
				return filepath.SkipDir // stop after first match per KB
			}
			return nil
		})
	}
	// KB layer
	kbs, _ := eng.ListKB(ctx)
	if len(kbs) > 0 {
		for _, kb := range kbs {
			if kb.ChunkCount == 0 {
				continue
			}
			vecs, err := memory.EmbedAndNormalize(ctx, eng.Embedder, []string{query})
			if err == nil {
				chunks, _ := eng.RAGEngine.Search(ctx, kb.ID, vecs[0], query, 3, 0.5)
				if len(chunks) > 0 {
					parts = append(parts, fmt.Sprintf("### Knowledge Base: %s", kb.Name))
					for _, c := range chunks {
						parts = append(parts, fmt.Sprintf("- [%s] %s", c.DocID, memory.Truncate(c.Content, 200)))
					}
					break
				}
			}
		}
	}
	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, "\n"), nil
}

func (m *memoryService) Status() (string, error) {
	eng, err := m.engine()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var b strings.Builder
	// Fact stats
	b.WriteString("**Memory Status**\n\n")
	b.WriteString("📌 Facts: enabled\n")
	// Wiki stats – count files in wiki dir
	home, _ := os.UserHomeDir()
	wikiDir := filepath.Join(home, ".gline", "memory", "wiki")
	wikiFiles, _ := filepath.Glob(filepath.Join(wikiDir, "*.md"))
	b.WriteString(fmt.Sprintf("📚 Wiki: %d pages\n", len(wikiFiles)))
	// KB stats
	kbs, _ := eng.ListKB(ctx)
	if len(kbs) > 0 {
		b.WriteString(fmt.Sprintf("📄 Knowledge Bases: %d", len(kbs)))
		for _, kb := range kbs {
			b.WriteString(fmt.Sprintf(" (%s: %d chunks)", kb.Name, kb.ChunkCount))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("📄 Knowledge Bases: 0\n")
	}
	return b.String(), nil
}

// ─── Exported KB APIs ────────────────────────────────────────────────────────

// KBCreate creates a new knowledge base.
func (c *ChatService) KBCreate(name, description, kbType string) (string, error) {
	eng, err := c.memoryEngine()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	t := memory.KBType(kbType)
	if t != memory.KBTypeRAG {
		t = memory.KBTypeRAG
	}
	kb, err := eng.InitKB(ctx, name, description, t)
	if err != nil {
		return "", err
	}
	return kb.ID, nil
}

// KBList returns the list of knowledge bases as a JSON string.
func (c *ChatService) KBList() (string, error) {
	eng, err := c.memoryEngine()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	kbs, err := eng.ListKB(ctx)
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(kbs)
	return string(data), nil
}

// KBIngestFile ingests a file (e.g. PDF, DOCX, MD) into a knowledge base.
// Only performs RAG ingestion (chunk + embed + store).
func (c *ChatService) KBIngestFile(kbNameOrID, filePath string) error {
	eng, err := c.memoryEngine()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	kb, err := eng.GetKB(ctx, kbNameOrID)
	if err != nil {
		return fmt.Errorf("knowledge base not found: %s", kbNameOrID)
	}
	return eng.IngestFile(ctx, kb.ID, filePath)
}

// WikiIngestFile triggers LLM-based wiki generation for a file.
// This is completely separate from RAG ingestion and requires a configured LLM provider.
func (c *ChatService) WikiIngestFile(filePath, kbID string) error {
	eng, err := c.memoryEngine()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	return eng.WikiIngestFile(ctx, filePath, kbID)
}

// SetApp sets the application reference (called after app creation).
func (c *ChatService) SetApp(app *application.App) {
	c.App = app
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
			if c.Backend.ag == nil {
				return 0, "", fmt.Errorf("agent not initialised")
			}
			baseAg, ok := c.Backend.ag.(*agent.BaseAgent)
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
	// Also copy any custom commands (skills) from the persistent registry
	// so that /skill, /skill-off and dynamic skill commands work in the GUI.
	for _, cmd := range c.cmdReg.GetAll() {
		if _, exists := reg.Get(cmd.Name); !exists {
			_ = reg.Register(cmd)
		}
	}
	freshCmd, ok := reg.Get(name)
	if !ok {
		return nil, fmt.Errorf("command not found: /%s", name)
	}

	_, err := freshCmd.Handler(args)
	if err != nil {
		return nil, err
	}

	// Skill commands only print to stdout in TUI mode, which is invisible in GUI.
	// Provide GUI-friendly feedback when no other result was captured.
	if capturedAction == slash.ResultNone && capturedMessage == "" {
		if c.Backend.skillRegistry != nil {
			switch name {
			case "skill":
				metas := c.Backend.skillRegistry.GetMeta()
				if len(metas) == 0 {
					capturedMessage = "No skills loaded."
				} else {
					var b strings.Builder
					b.WriteString("Available skills:\n")
					for _, meta := range metas {
						b.WriteString(fmt.Sprintf("  /%-15s %s\n", meta.Name, meta.Description))
					}
					b.WriteString("\nSkills are loaded on-demand via the use_skill tool.")
					capturedMessage = b.String()
				}
			default:
				if s, ok := c.Backend.skillRegistry.Get(name); ok {
					var b strings.Builder
					b.WriteString(fmt.Sprintf("Skill: %s\n", s.Name))
					b.WriteString(fmt.Sprintf("Description: %s\n\n", s.Description))
					b.WriteString("To activate this skill, use the use_skill tool with skill_name: " + s.Name)
					capturedMessage = b.String()
				}
			}
		}
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
	if c.Backend.ag == nil {
		return 0, "", fmt.Errorf("agent not initialised")
	}
	baseAg, ok := c.Backend.ag.(*agent.BaseAgent)
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
	return c.Backend.GetConfig()
}

// UpdateConfig updates a config key
func (c *ChatService) UpdateConfig(key string, value string) error {
	return c.Backend.UpdateConfig(key, value)
}

// ListTasks returns conversation history
func (c *ChatService) ListTasks(limit int, offset int) ([]storage.TaskRecord, error) {
	return c.Backend.ListTasks(limit, offset)
}

// GetTaskSummary returns a task with its messages
func (c *ChatService) GetTaskSummary(taskID string) (*storage.TaskRecord, []storage.MessageRecord, error) {
	return c.Backend.GetTaskSummary(taskID)
}

// DeleteTask deletes a task and its messages
func (c *ChatService) DeleteTask(taskID string) error {
	return c.Backend.DeleteTask(taskID)
}
func (c *ChatService) SendMessage(prompt string) error {
	if c.Backend.ag == nil {
		return fmt.Errorf("agent not initialised")
	}
	if c.App == nil {
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

	cb := &guiStreamCallback{app: c.App, svc: c}

	go func(done chan struct{}) {
		defer close(done)
		if err := c.Backend.ag.RunWithCallback(ctx, prompt, cb); err != nil {
			if ctx.Err() != context.Canceled {
				c.App.Event.Emit("chat:error", err.Error())
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
	if c.Backend.ag != nil {
		c.Backend.ag.Abort()
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
	if c.Backend.ag != nil {
		c.Backend.ag.(*agent.BaseAgent).ResetTask()
		c.Backend.ag.GetConversation().Clear()
	}
}

// LoadTask restores agent state for an existing task.
func (c *ChatService) LoadTask(taskID string) (*storage.TaskRecord, error) {
	task, err := c.Backend.LoadTask(taskID)
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
	if c.Backend.ag == nil {
		return "act"
	}
	return string(c.Backend.ag.GetMode())
}

// SetMode switches the agent between plan and act modes.
func (c *ChatService) SetMode(mode string) error {
	if c.Backend.ag == nil {
		return fmt.Errorf("agent not initialised")
	}
	switch mode {
	case "plan", "act":
		return c.Backend.ag.SetMode(agent.Mode(mode))
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}
}

// CompactConversation triggers manual compaction of the conversation history.
func (c *ChatService) CompactConversation() (bool, error) {
	if c.Backend.ag == nil {
		return false, fmt.Errorf("agent not initialised")
	}
	compacted := c.Backend.ag.Compact()
	return compacted, nil
}

// GetStatus returns current provider, model, working directory, mode and token usage.
func (c *ChatService) GetStatus() (map[string]string, error) {
	cfg := c.Backend.cfg.Get()
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
	if c.Backend.ag != nil {
		conv := c.Backend.ag.GetConversation()
		if conv != nil {
			mode = string(c.Backend.ag.GetMode())
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
// Messages are truncated to avoid exceeding Wails IPC payload limits.
func (c *ChatService) GetConversationState() string {
	if c.Backend.ag == nil {
		return "[]"
	}
	conv := c.Backend.ag.GetConversation()
	msgs := conv.GetMessages()
	type msgView struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	const maxMsgContent = 16000   // per-message limit
	const maxMsgs = 200          // hard cap on number of messages returned
	start := 0
	if len(msgs) > maxMsgs {
		start = len(msgs) - maxMsgs
	}
	viewsCap := len(msgs) - start
	if viewsCap > maxMsgs {
		viewsCap = maxMsgs
	}
	views := make([]msgView, 0, viewsCap)
	for i := start; i < len(msgs); i++ {
		m := msgs[i]
		content := m.Content
		if len(content) > maxMsgContent {
			content = content[:maxMsgContent] + "\n\n[… truncated]"
		}
		views = append(views, msgView{Role: string(m.Role), Content: content})
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

// OnToolCallStart is called when a tool call starts
func (g *guiStreamCallback) OnToolCallStart(toolCall agent.ToolCall) {
	g.app.Event.Emit("chat:toolStart", map[string]string{
		"id":    toolCall.ID,
		"name":  toolCall.Name,
		"input": toolCall.Input,
	})
}

// maxEventPayload is the maximum bytes we will emit through the Wails event
// bus in a single call.  Anything larger is truncated to avoid IPC buffer
// issues on Windows.
const maxEventPayload = 2 << 20 // 2 MB

func truncateEventPayload(s string) string {
	if len(s) > maxEventPayload {
		return s[:maxEventPayload] + "\n\n[… truncated: result too large for UI display]"
	}
	return s
}

func (g *guiStreamCallback) OnToolCallComplete(toolCall agent.ToolCall, result string) {
	g.app.Event.Emit("chat:toolComplete", map[string]interface{}{
		"id":     toolCall.ID,
		"name":   toolCall.Name,
		"result": truncateEventPayload(result),
	})
	// Special tools whose result should be shown directly to the user
	// (not hidden behind a tool-call badge).
	safeResult := truncateEventPayload(result)
	switch toolCall.Name {
	case "plan_mode_respond":
		if safeResult != "" {
			g.app.Event.Emit("chat:systemMessage", map[string]interface{}{
				"role":    "system",
				"content": safeResult,
			})
		}
	case "attempt_completion":
		if safeResult != "" {
			g.app.Event.Emit("chat:systemMessage", map[string]interface{}{
				"role":    "system",
				"content": "📋 " + safeResult,
			})
		}
	case "ask_followup_question":
		// The question itself is already emitted via chat:followupQuestion
		// but we also show the answer prompt as a system message so the user
		// sees the question inline without needing the popup.
		if safeResult != "" {
			g.app.Event.Emit("chat:systemMessage", map[string]interface{}{
				"role":    "system",
				"content": "💬 " + safeResult,
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
