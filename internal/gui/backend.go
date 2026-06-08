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
	"github.com/liup215/gline/internal/memory"
	"github.com/liup215/gline/internal/skills"
	"github.com/liup215/gline/internal/storage"
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

	// Try to initialize memory engine if embedding is configured
	var memoryEngine *memory.UnifiedEngine
	memCfg := cfg.Memory
	if memCfg.Embedding.Provider != "" || memCfg.Embedding.APIKey != "" {
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
	b.cfg.Set(key, value)
	b.cfg.Save()

	if key == "provider.default" ||
		key == "provider.openai.max_context_tokens" {
		if err := b.initAgent(); err != nil {
			return fmt.Errorf("reinit agent: %w", err)
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
