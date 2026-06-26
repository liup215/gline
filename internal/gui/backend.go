package gui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/api"
	"github.com/liup215/gline/internal/config"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/mcp"
	"github.com/liup215/gline/internal/memory"
	"github.com/liup215/gline/internal/skills"
	"github.com/liup215/gline/internal/storage"
	"github.com/liup215/gline/internal/subagent"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
)

// BackendInstance is the global GUI backend. It is initialised by InitBackend.
var BackendInstance *Backend

// InitBackend initialises the global backend for GUI mode.
func InitBackend() error {
	// Initialize logger with file output so diagnostic info persists even in GUI mode.
	logDir := getGlobalConfigDir()
	logPath := filepath.Join(logDir, "gline.log")
	// Force file-only logging. In Windows GUI mode (-H windowsgui) stderr
	// is nil, so a ConsoleWriter would silently break the MultiWriter chain.
	if err := log.Init(log.Config{
		Level:   "info",
		File:    logPath,
		Console: false,
		Color:   false,
	}); err != nil {
		// Non-fatal: proceed without logging if init fails
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
	}
	log.Infof("=== GUI session started, log file: %s ===", logPath)

	BackendInstance = &Backend{}
	if err := BackendInstance.initConfig(); err != nil {
		return fmt.Errorf("init config: %w", err)
	}
	if err := BackendInstance.initStorage(); err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	if err := BackendInstance.initAgent(); err != nil {
		return fmt.Errorf("init agent: %w", err)
	}
	return nil
}

// Backend exposes the gline core to Wails frontend
type Backend struct {
	cfg            *config.Manager
	store          storage.Store
	ag             agent.Agent
	skillRegistry  *skills.Registry
	mcpManager     *mcp.Manager
}

func (b *Backend) initConfig() error {
	b.cfg = config.NewManager()
	return b.cfg.Load()
}

func (b *Backend) initStorage() error {
	dir := getGlobalConfigDir()
	dbPath := filepath.Join(dir, "gline.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		return err
	}
	b.store = store
	return nil
}

func getGlobalConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".gline"
	}
	return filepath.Join(homeDir, ".gline")
}

func (b *Backend) initAgent() error {
	cfg := b.cfg.Get()
	providerName := cfg.Provider.Default
	if providerName == "" {
		providerName = "openai"
	}

	var provider agent.Provider
	var maxTokens int
	switch providerName {
	case "openai":
		s := cfg.Provider.OpenAI
		provider = api.NewOpenAIProvider(s.APIKey, s.Model, s.BaseURL)
		maxTokens = s.MaxContextTokens
	default:
		return fmt.Errorf("unknown provider: %s", providerName)
	}

	customRules := loadCustomRules()

	// Try to initialize memory engine if enabled and embedding is configured
	var memoryEngine *memory.UnifiedEngine
	memCfg := cfg.Memory
	if memCfg.Enabled && (memCfg.Embedding.Provider != "" || memCfg.Embedding.APIKey != "") {
		var embedder memory.Embedder
		switch memCfg.Embedding.Provider {
		case "ollama":
			embedder = memory.NewOllamaEmbedder(memCfg.Embedding.Model)
		default:
			apiKey := memCfg.Embedding.APIKey
			if apiKey == "" {
				apiKey = cfg.Provider.OpenAI.APIKey
			}
			embedder = memory.NewOpenAIEmbedder(apiKey, memCfg.Embedding.Model)
			if memCfg.Embedding.BaseURL != "" {
				embedder.(*memory.OpenAIEmbedder).BaseURL = memCfg.Embedding.BaseURL
			}
		}
		var err error
		memoryEngine, err = memory.NewUnifiedEngine(embedder)
		if err != nil {
			log.Warnf("Memory engine not initialised: %v", err)
		} else {
			// Wire LLM caller for wiki ingest and future memory layers
			memoryEngine.Caller = func(ctx context.Context, systemPrompt, userContent string) (string, error) {
				req := &agent.MessageRequest{
					Messages: []types.Message{
						{Role: types.RoleUser, Content: userContent},
					},
					SystemPrompt:  systemPrompt,
					MaxTokens:     2048,
					Temperature:   0.0,
				}
				resp, err := provider.CreateMessage(ctx, req)
				if err != nil {
					return "", err
				}
				return resp.Content, nil
			}
			log.Info("Memory engine initialised")
		}
	}

	// Initialize and load skills FIRST so they are available for the use_skill tool
	b.skillRegistry = skills.NewRegistry()
	b.skillRegistry.LoadFromDirs(skills.DefaultSkillDirs...)

	// Initialize tool registry and register use_skill with the skill registry
	registry := tools.InitDefaultRegistry(memoryEngine)
	tools.RegisterSkillTool(registry, b.skillRegistry)

	// Register use_subagents tool
	subagent.RegisterTool(registry, provider, registry, "", customRules, b.skillRegistry.GetMeta())

	ag, err := agent.New(agent.Options{
		Provider:     provider,
		ToolRegistry:   registry,
		Mode:           agent.ModeAct,
		AutoApprove:    false,
		CustomRules:    customRules,
		Store:          b.store,
		MaxTokens:      maxTokens,
		MemoryEngine:   memoryEngine,
		Skills:         b.skillRegistry.GetMeta(),
	})
	if err != nil {
		return err
	}
	b.ag = ag

	// Initialize MCP Manager if configured
	if len(cfg.MCP.Servers) > 0 {
		b.mcpManager = mcp.NewManager(&cfg.MCP, registry)
		if err := b.mcpManager.Start(context.Background()); err != nil {
			log.Warnf("Failed to start MCP manager: %v", err)
			b.mcpManager = nil // Clean up on failure
		} else {
			// Get status for logging
			statuses := b.mcpManager.GetServerStatus()
			totalTools := 0
			for _, status := range statuses {
				if status.Initialized {
					log.Infof("MCP server '%s' connected with %d tools", status.Name, status.Tools)
					totalTools += status.Tools
				}
			}
			log.Infof("MCP initialized: %d servers, %d total tools", len(statuses), totalTools)
		}
	}

	return nil
}

func loadCustomRules() string {
	homeDir, _ := os.UserHomeDir()
	var content string

	globalRulesDir := filepath.Join(homeDir, ".gline", "rules")
	content += loadRulesFromDir(globalRulesDir)

	workspaceRulesDir := filepath.Join(".gline", "rules")
	content += loadRulesFromDir(workspaceRulesDir)

	return content
}

func loadRulesFromDir(dir string) string {
	var result string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".md" && filepath.Ext(name) != ".txt" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || len(data) == 0 {
			continue
		}
		result += string(data) + "\n"
	}
	return result
}

// GetConfig returns the current configuration as JSON
func (b *Backend) GetConfig() (string, error) {
	cfg := b.cfg.Get()
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UpdateConfig updates a config key
func (b *Backend) UpdateConfig(key string, value string) error {
	// Handle special case for MCP servers (JSON array)
	if key == "mcp.servers" {
		// Parse the JSON array of servers
		var servers []mcp.ServerConfig
		if err := json.Unmarshal([]byte(value), &servers); err != nil {
			return fmt.Errorf("failed to parse MCP servers: %w", err)
		}
		// Update the config in both viper and memory struct
		b.cfg.Set("mcp.servers", servers)
		cfg := b.cfg.Get()
		cfg.MCP.Servers = servers
		if err := b.cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		// Restart MCP manager if agent is initialized
		if b.ag != nil {
			if err := b.restartMCPManager(); err != nil {
				return fmt.Errorf("failed to restart MCP manager: %w", err)
			}
		}
		return nil
	}

	b.cfg.Set(key, value)
	b.cfg.Save()

	if key == "provider.default" ||
		key == "provider.openai.max_context_tokens" ||
		key == "memory.enabled" ||
		key == "memory.embedding.provider" ||
		key == "memory.embedding.model" ||
		key == "memory.embedding.api_key" ||
		key == "memory.embedding.base_url" ||
		key == "memory.retrieval.top_k" ||
		key == "memory.retrieval.min_score" ||
		key == "memory.retrieval.max_tokens" {
		if err := b.initAgent(); err != nil {
			return fmt.Errorf("reinit agent: %w", err)
		}
	}
	return nil
}

// restartMCPManager restarts the MCP manager with updated config
func (b *Backend) restartMCPManager() error {
	// Close existing MCP manager if any
	if b.mcpManager != nil {
		if err := b.mcpManager.Close(); err != nil {
			log.Warnf("Error closing MCP manager: %v", err)
		}
		b.mcpManager = nil
	}

	// Create and start new MCP manager
	cfg := b.cfg.Get()
	if len(cfg.MCP.Servers) > 0 {
		// Get tool registry from agent
		if b.ag != nil {
			registry := b.ag.GetToolRegistry()
			b.mcpManager = mcp.NewManager(&cfg.MCP, registry)
			if err := b.mcpManager.Start(context.Background()); err != nil {
				// Don't return error here, just log it - we don't want to crash the app
				log.Warnf("Failed to start MCP manager: %v", err)
				b.mcpManager = nil
			}
		}
	}

	return nil
}

// ListTasks returns conversation history
func (b *Backend) ListTasks(limit int, offset int) ([]storage.TaskRecord, error) {
	return b.store.ListTasks(limit, offset)
}

// GetTaskSummary returns a task with its messages
func (b *Backend) GetTaskSummary(taskID string) (*storage.TaskRecord, []storage.MessageRecord, error) {
	return b.store.GetTaskSummary(taskID)
}

// DeleteTask deletes a task and its messages
func (b *Backend) DeleteTask(taskID string) error {
	return b.store.DeleteTask(taskID)
}

// LoadTask restores the agent's state for an existing task by setting its taskID
// and loading messages back into the conversation.
func (b *Backend) LoadTask(taskID string) (*storage.TaskRecord, error) {
	if b.ag == nil {
		return nil, fmt.Errorf("agent not initialised")
	}
	baseAg, ok := b.ag.(*agent.BaseAgent)
	if !ok {
		return nil, fmt.Errorf("agent type mismatch")
	}
	// Load task metadata and messages from storage
	task, msgs, err := b.store.GetTaskSummary(taskID)
	if err != nil {
		return nil, fmt.Errorf("load task summary: %w", err)
	}
	if task != nil && task.WorkingDir != "" {
		if err := os.Chdir(task.WorkingDir); err == nil {
			baseAg.SetWorkingDir(task.WorkingDir)
		} else {
			log.Warnf("Failed to chdir to %s: %v", task.WorkingDir, err)
		}
	}
	// Set task ID so responses are stored under the same task
	baseAg.SetTaskID(taskID)
	// Load messages from storage into the conversation
	b.ag.GetConversation().Clear()
	for _, m := range msgs {
		msg, err := m.ToTypesMessage()
		if err != nil {
			log.Warnf("failed to convert message record: %v", err)
			continue
		}
		b.ag.GetConversation().AddMessage(msg)
	}
	return task, nil
}

// MCPServerStatus represents the status of an MCP server for the frontend
type MCPServerStatus struct {
	Name        string   `json:"name"`
	Connected   bool     `json:"connected"`
	Initialized bool     `json:"initialized"`
	Tools       int      `json:"tools"`
	ToolNames   []string `json:"toolNames"`
	LastError   string   `json:"lastError"`
}

// GetMCPStatus returns the status of all MCP servers
func (b *Backend) GetMCPStatus() ([]MCPServerStatus, error) {
	if b.mcpManager == nil {
		return []MCPServerStatus{}, nil
	}

	statuses := b.mcpManager.GetServerStatus()
	result := make([]MCPServerStatus, len(statuses))
	for i, s := range statuses {
		result[i] = MCPServerStatus{
			Name:        s.Name,
			Connected:   s.Connected,
			Initialized: s.Initialized,
			Tools:       s.Tools,
			ToolNames:   s.ToolNames,
			LastError:   s.LastError,
		}
	}
	return result, nil
}
