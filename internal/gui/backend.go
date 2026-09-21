package gui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/liup215/gline/internal/adkagent"
	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/api"
	"github.com/liup215/gline/internal/config"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/mcp"
	"github.com/liup215/gline/internal/memory"
	"github.com/liup215/gline/internal/sessionstore"
	"github.com/liup215/gline/internal/skills"
	"github.com/liup215/gline/internal/storage"
	"github.com/liup215/gline/internal/subagent"
	"github.com/liup215/gline/internal/summarizer"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/internal/ui"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/session/compaction"
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
	ag             ui.AgentRunner
	toolRegistry   *tools.Registry
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

	// Debug: Log MCP configuration
	log.Infof("MCP config check: %d servers in config", len(cfg.MCP.Servers))
	for i, server := range cfg.MCP.Servers {
		log.Infof("  MCP server %d: name=%s, transport=%s, url=%s, disabled=%v", 
			i, server.Name, server.TransportType, server.URL, server.Disabled)
	}
	
	providerName := cfg.Provider.Default
	if providerName == "" {
		providerName = "openai"
	}
	useLegacy := os.Getenv("GLINE_AGENT") == "legacy"

	// Legacy provider interface — always built: the ADK path's
	// summarize_file / use_subagents tools still make sub-LLM calls
	// through it.
	legacyProvider, maxTokens, err := b.buildLegacyProvider(providerName)
	if err != nil {
		return err
	}
	_ = maxTokens

	memoryEngine := b.initMemoryEngine(legacyProvider)
	customRules := loadCustomRules()

	// Initialize and load skills FIRST so they are available for the use_skill tool
	b.skillRegistry = skills.NewRegistry()
	b.skillRegistry.LoadFromDirs(skills.DefaultSkillDirs...)

	// Initialize tool registry first (without summarizer) to break the
	// circular dependency between registry, subagent builder and summarizer.
	registry := tools.InitDefaultRegistry(memoryEngine, nil)
	b.toolRegistry = registry

	// Subagent builder for large-file summarizer, using the real registry.
	subBuilder := subagent.NewBuilder(legacyProvider, registry, "", customRules, b.skillRegistry.GetMeta())
	sum := summarizer.NewSummarizer(subagent.NewSummarizerCaller(subBuilder), summarizer.DefaultOptions())

	// Register summarization tool and remaining tools.
	_ = tools.RegisterSummarizeFileTool(registry, sum)
	tools.RegisterSkillTool(registry, b.skillRegistry)

	// Register use_subagents tool
	subagent.RegisterTool(registry, legacyProvider, registry, "", customRules, b.skillRegistry.GetMeta())

	if useLegacy {
		ag, err := agent.New(agent.Options{
			Provider:       legacyProvider,
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
		b.ag = ui.LegacyRunner(ag)
		log.Info("GUI using legacy agent loop (GLINE_AGENT=legacy)")
	} else {
		adkAg, err := b.buildAdkAgent(providerName, registry, memoryEngine)
		if err != nil {
			return err
		}
		b.ag = ui.AdkRunner(adkAg)
		provName, modelName := adkAg.ProviderInfo()
		log.Infof("GUI using ADK agent with provider %s model %s", provName, modelName)
	}

	// Initialize MCP Manager if configured
	if len(cfg.MCP.Servers) > 0 {
		log.Infof("Starting MCP manager with %d servers", len(cfg.MCP.Servers))
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
				} else {
					log.Warnf("MCP server '%s' failed to initialize: %s", status.Name, status.LastError)
				}
			}
			log.Infof("MCP initialized: %d servers, %d total tools", len(statuses), totalTools)
		}
	} else {
		log.Info("No MCP servers configured, skipping MCP initialization")
	}

	return nil
}

// guiProviderSettings carries the resolved provider coordinates.
type guiProviderSettings struct {
	id      string // internal provider id (openai / opencode-go / openrouter / volcano)
	model   string
	apiKey  string
	baseURL string
}

// resolveProviderSettings maps the configured default provider name to
// concrete credentials and endpoints.
func (b *Backend) resolveProviderSettings(name string) (guiProviderSettings, error) {
	cfg := b.cfg.Get()
	switch name {
	case "openai":
		s := cfg.Provider.OpenAI
		return guiProviderSettings{id: "openai", model: s.Model, apiKey: s.APIKey, baseURL: s.BaseURL}, nil
	case "opencode-go":
		s := cfg.Provider.OpenCodeGo
		apiKey := s.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENCODE_API_KEY")
		}
		model := s.Model
		if model == "" {
			model = "kimi-k2.7-code"
		}
		baseURL := s.BaseURL
		if baseURL == "" {
			baseURL = api.OpenCodeGoBaseURL
		}
		return guiProviderSettings{id: "opencode-go", model: model, apiKey: apiKey, baseURL: baseURL}, nil
	case "openrouter":
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			return guiProviderSettings{}, fmt.Errorf("OpenRouter API key not configured")
		}
		return guiProviderSettings{id: "openrouter", model: "anthropic/claude-sonnet-4", apiKey: apiKey, baseURL: "https://openrouter.ai/api/v1"}, nil
	default:
		return guiProviderSettings{}, fmt.Errorf("unknown provider: %s. Supported: openai, opencode-go, openrouter", name)
	}
}

// buildLegacyProvider constructs the legacy agent.Provider for the
// configured provider. Used by the legacy loop and by the ADK path's
// sub-LLM tools (summarize_file, use_subagents).
func (b *Backend) buildLegacyProvider(name string) (agent.Provider, int, error) {
	s, err := b.resolveProviderSettings(name)
	if err != nil {
		return nil, 0, err
	}
	cfg := b.cfg.Get()
	var maxTokens int
	var provider agent.Provider
	switch s.id {
	case "openai":
		maxTokens = cfg.Provider.OpenAI.MaxContextTokens
		provider = api.NewOpenAIProvider(s.apiKey, s.model, s.baseURL)
	case "opencode-go":
		maxTokens = cfg.Provider.OpenCodeGo.MaxContextTokens
		provider, err = api.NewGoLLMProvider(s.apiKey, s.model, s.baseURL, "opencode-go")
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create OpenCode Go provider: %w", err)
		}
	case "openrouter":
		provider, err = api.NewGoLLMProvider(s.apiKey, s.model, s.baseURL, "openrouter")
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create OpenRouter provider: %w", err)
		}
	}
	return provider, maxTokens, nil
}

// initMemoryEngine mirrors the CLI assembly: builds the unified memory
// engine when enabled and an embedder is configured.
func (b *Backend) initMemoryEngine(caller agent.Provider) *memory.UnifiedEngine {
	cfg := b.cfg.Get()
	memCfg := cfg.Memory
	if !memCfg.Enabled || (memCfg.Embedding.Provider == "" && memCfg.Embedding.APIKey == "") {
		return nil
	}
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
	engine, err := memory.NewUnifiedEngine(embedder)
	if err != nil {
		log.Warnf("Memory engine not initialised: %v", err)
		return nil
	}
	if caller != nil {
		engine.Caller = func(ctx context.Context, systemPrompt, userContent string) (string, error) {
			req := &agent.MessageRequest{
				Messages: []types.Message{
					{Role: types.RoleUser, Content: userContent},
				},
				SystemPrompt: systemPrompt,
				MaxTokens:    2048,
				Temperature:  0.0,
			}
			resp, err := caller.CreateMessage(ctx, req)
			if err != nil {
				return "", err
			}
			return resp.Content, nil
		}
	}
	log.Info("Memory engine initialised")
	return engine
}

// mapGUIProviderID maps internal provider ids to internal/provider ids.
func mapGUIProviderID(id string) string {
	if id == "opencode-go" {
		return "opencode"
	}
	return id
}

// buildAdkAgent assembles the ADK-backed agent (same shape as the CLI's
// initializeAgent): provider LLM, tool registry, memory engine, compaction
// and the persistent ADK session store.
func (b *Backend) buildAdkAgent(providerName string, registry *tools.Registry, memoryEngine *memory.UnifiedEngine) (*adkagent.Agent, error) {
	s, err := b.resolveProviderSettings(providerName)
	if err != nil {
		return nil, err
	}

	sessionStore, err := sessionstore.Open(sessionstore.Options{})
	if err != nil {
		log.Warnf("ADK session store unavailable (%v); using in-memory sessions", err)
		sessionStore = nil // adkagent falls back to an in-memory service
	}
	var sessionSvc session.Service
	if sessionStore != nil {
		sessionSvc = sessionStore.Service()
	}

	return adkagent.New(context.Background(), adkagent.Options{
		Provider:       mapGUIProviderID(s.id),
		Model:          s.model,
		APIKey:         s.apiKey,
		BaseURL:        s.baseURL,
		Tools:          registry,
		MemoryEngine:   memoryEngine,
		Skills:         b.skillRegistry.GetMeta(),
		Compaction:     defaultCompactionConfig(),
		Store:          b.store,
		SessionService: sessionSvc,
	})
}

// defaultCompactionConfig mirrors the CLI's tail-retention compaction.
func defaultCompactionConfig() *compaction.Config {
	return &compaction.Config{
		TokenThreshold:     80_000,
		EventRetentionSize: 12,
	}
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
		// Get tool registry from the backend
		if b.toolRegistry != nil {
			b.mcpManager = mcp.NewManager(&cfg.MCP, b.toolRegistry)
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

// LoadTask restores the agent's state for an existing task. Legacy agents
// replay the stored messages into the in-memory conversation; ADK agents
// resume the recorded ADK session when one exists (falling back to task-id
// reattachment for pre-sessionstore tasks).
func (b *Backend) LoadTask(taskID string) (*storage.TaskRecord, error) {
	if b.ag == nil {
		return nil, fmt.Errorf("agent not initialised")
	}
	// Load task metadata and messages from storage
	task, msgs, err := b.store.GetTaskSummary(taskID)
	if err != nil {
		return nil, fmt.Errorf("load task summary: %w", err)
	}
	if task != nil && task.WorkingDir != "" {
		if err := os.Chdir(task.WorkingDir); err == nil {
			if wd, ok := b.ag.(ui.WorkingDirSetter); ok {
				wd.SetWorkingDir(task.WorkingDir)
			}
		} else {
			log.Warnf("Failed to chdir to %s: %v", task.WorkingDir, err)
		}
	}

	if base, ok := b.ag.(ui.ConversationProvider); ok {
		// Legacy path: replay transcript into the conversation.
		if tm, ok := b.ag.(interface{ SetTaskID(string) }); ok {
			tm.SetTaskID(taskID)
		}
		base.GetConversation().Clear()
		for _, m := range msgs {
			msg, err := m.ToTypesMessage()
			if err != nil {
				log.Warnf("failed to convert message record: %v", err)
				continue
			}
			base.GetConversation().AddMessage(msg)
		}
		return task, nil
	}

	// ADK path: resume the recorded session when available.
	if resumer, ok := b.ag.(ui.ResumeSessionResumer); ok && task != nil && task.SessionID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := resumer.ResumeSession(ctx, task.SessionID); err != nil {
			log.Warnf("ResumeSession(%s) failed: %v; attaching task id only", task.SessionID, err)
		}
	}
	if tm, ok := b.ag.(interface{ SetTaskID(string) }); ok {
		tm.SetTaskID(taskID)
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
