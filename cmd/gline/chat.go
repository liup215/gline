package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liup215/gline/internal/adkagent"
	glineagent "github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/api"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/memory"
	"github.com/liup215/gline/internal/prompts"
	"github.com/liup215/gline/internal/sessionstore"
	"github.com/liup215/gline/internal/skills"
	"github.com/liup215/gline/internal/storage"
	"github.com/liup215/gline/internal/subagent"
	"github.com/liup215/gline/internal/summarizer"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/internal/ui"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/session/compaction"
)

// providerSettings carries the resolved provider configuration. The legacy
// loop consumes it via a go-llm provider instance; the ADK agent consumes
// the raw fields directly (Phase 5 assembly switch).
type providerSettings struct {
	name    string // provider id: opencode-go, volcano, openrouter, openai, mock
	model   string
	apiKey  string
	baseURL string
}

// resolveProviderSettings maps config + env to a providerSettings.
// Returns an error for unknown providers or missing credentials.
func resolveProviderSettings() (*providerSettings, error) {
	cfg := configManager.Get()

	providerName := cfg.Provider.Default
	if providerName == "" {
		providerName = "openai"
	}

	switch providerName {
	case "openai":
		apiKey := cfg.Provider.OpenAI.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("GLINE_OPENAI_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key not configured. Set it in config or GLINE_OPENAI_API_KEY environment variable")
		}
		model := cfg.Provider.OpenAI.Model
		if model == "" {
			model = "gpt-4"
		}
		return &providerSettings{name: "openai", model: model, apiKey: apiKey, baseURL: cfg.Provider.OpenAI.BaseURL}, nil

	case "opencode-go":
		settings := cfg.Provider.OpenCodeGo
		apiKey := settings.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENCODE_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("OpenCode Go API key not configured")
		}
		model := settings.Model
		if model == "" {
			model = "kimi-k2.7-code"
		}
		baseURL := settings.BaseURL
		if baseURL == "" {
			baseURL = api.OpenCodeGoBaseURL
		}
		return &providerSettings{name: "opencode-go", model: model, apiKey: apiKey, baseURL: baseURL}, nil

	case "openrouter":
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OpenRouter API key not configured. Set OPENROUTER_API_KEY environment variable")
		}
		return &providerSettings{name: "openrouter", model: "anthropic/claude-sonnet-4", apiKey: apiKey, baseURL: "https://openrouter.ai/api/v1"}, nil

	case "volcano":
		settings := cfg.Provider.Volcano
		apiKey := settings.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("VOLCANO_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("Volcano API key not configured. Set it in config or VOLCANO_API_KEY environment variable")
		}
		model := settings.Model
		if model == "" {
			model = "ark-code-latest"
		}
		baseURL := settings.BaseURL
		if baseURL == "" {
			baseURL = "https://ark.cn-beijing.volces.com/api/plan/v3"
		}
		return &providerSettings{name: "volcano", model: model, apiKey: apiKey, baseURL: baseURL}, nil

	case "mock":
		return &providerSettings{name: "mock"}, nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}

// initializeAgent assembles the agent used by the TUI and the chat CLI.
// The ADK-backed agent (internal/adkagent) is the only engine; the legacy
// hand-written loop was removed in Phase 7. A legacy-style provider is
// still built for the sub-LLM tools (summarize_file, use_subagents).
// defaultCompactionConfig enables ADK tail-retention compaction: once the
// prompt grows past TokenThreshold, everything but the most recent
// EventRetentionSize events is summarized before the next model call.
// Sliding-window compaction stays off — tail retention alone bounds prompt
// growth without interval/retention arithmetic.
func defaultCompactionConfig() *compaction.Config {
	return &compaction.Config{
		TokenThreshold:     80_000,
		EventRetentionSize: 12,
	}
}

// initializeMemoryEngine builds the optional unified memory engine (facts
// + wiki + RAG) when memory is enabled and an embedder can be constructed.
// Mirrors internal/gui/backend.go; failures degrade to a nil engine.
func initializeMemoryEngine() *memory.UnifiedEngine {
	cfg := configManager.Get()
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
	log.Info("Memory engine initialised")
	return engine
}

func initializeAgent() (ui.AgentRunner, storage.Store, error) {

	settings, err := resolveProviderSettings()
	if err != nil {
		return nil, nil, err
	}
	if settings.name == "mock" {
		return nil, nil, fmt.Errorf("mock provider is no longer supported (legacy agent loop removed); configure a real provider")
	}

	// Shared tool-registry assembly.
	registry, store, skillReg, err := assembleSharedComponents()
	if err != nil {
		return nil, nil, err
	}

	// Legacy provider interface — kept solely for sub-LLM tools
	// (summarize_file / use_subagents).
	legacyProvider, err := api.NewGoLLMProvider(settings.apiKey, settings.model, settings.baseURL, settings.name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create %s provider: %w", settings.name, err)
	}
	log.Infof("Using %s provider with model: %s", settings.name, settings.model)

	customRules, _ := prompts.LoadCustomRules()
	subBuilder := subagent.NewBuilder(legacyProvider, registry, "", customRules, skillReg.GetMeta())
	sum := summarizer.NewSummarizer(subagent.NewSummarizerCaller(subBuilder), summarizer.DefaultOptions())
	_ = tools.RegisterSummarizeFileTool(registry, sum)
	subagent.RegisterTool(registry, legacyProvider, registry, "", customRules, skillReg.GetMeta())
	log.Infof("Initialized %d tools", registry.Count())

	ctx := context.Background()
	// Persistent ADK sessions (~/.gline/sessions.db). Falling back to the
	// in-memory service keeps the app usable; only history resume is lost.
	var sessionService session.Service
	if ss, err := sessionstore.Open(sessionstore.Options{}); err != nil {
		log.Warnf("session store unavailable, falling back to in-memory sessions: %v", err)
	} else {
		sessionService = ss.Service()
	}
	agentInstance, err := adkagent.New(ctx, adkagent.Options{
		Provider:       mapProviderID(settings.name),
		Model:          settings.model,
		APIKey:         settings.apiKey,
		BaseURL:        settings.baseURL,
		Tools:          registry,
		Mode:           string(glineagent.ModeAct),
		SessionService: sessionService,
		Store:          store,
		Compaction:     defaultCompactionConfig(),
		MemoryEngine:   initializeMemoryEngine(),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create agent: %w", err)
	}
	log.Infof("Using ADK agent with provider %s model %s", settings.name, settings.model)
	return ui.AdkRunner(agentInstance), store, nil
}

// mapProviderID translates legacy config provider IDs to the IDs used by
// internal/provider. "opencode-go" is the historical config name for the
// OpenCode Go endpoint that the provider package calls "opencode".
func mapProviderID(name string) string {
	if name == "opencode-go" {
		return "opencode"
	}
	return name
}

// assembleSharedComponents builds the tool registry, storage, skills and
// subagent wiring used by the agent engine.
func assembleSharedComponents() (*tools.Registry, storage.Store, *skills.Registry, error) {
	// Load custom rules from global and workspace directories
	_, _ = prompts.LoadCustomRules()

	// Initialize skills registry
	skillReg := skills.NewRegistry()

	// Create tool registry first (without summarizer, to break the circular
	// dependency between registry -> subagent builder -> summarizer -> registry).
	registry := tools.InitDefaultRegistry(nil, nil)

	// Create persistent storage
	store, err := storage.NewSQLiteStore("")
	if err != nil {
		log.Warnf("Failed to initialize storage: %v", err)
		store = nil // Continue without storage
	}

	tools.RegisterSkillTool(registry, skillReg)
	return registry, store, skillReg, nil
}

// printCallback streams agent events to stdout for non-interactive use.
type printCallback struct{}

func (printCallback) OnContent(delta string)   { fmt.Print(delta) }
func (printCallback) OnReasoning(delta string) {}
func (printCallback) OnStreamStart()           { fmt.Println("AI: ") }
func (printCallback) OnStreamEnd()             { fmt.Println() }
func (printCallback) OnToolCallStart(tc glineagent.ToolCall) {
	fmt.Printf("🔧 Tool: %s\n", tc.Name)
}
func (printCallback) OnToolCallComplete(tc glineagent.ToolCall, result string) {
	summary := strings.TrimSpace(result)
	if len(summary) > 200 {
		summary = summary[:200] + "..."
	}
	fmt.Printf("   ↳ %s\n", summary)
}
func (printCallback) AskFollowupQuestion(question string, options []string) (string, error) {
	fmt.Printf("❓ %s\n", question)
	if len(options) > 0 {
		fmt.Printf("   (defaulting to: %s)\n", options[0])
		return options[0], nil
	}
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	return strings.TrimSpace(answer), nil
}
func (printCallback) OnError(err error) {
	log.Errorf("Agent error: %v", err)
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}
func (printCallback) OnTaskCreated(taskID string) {}
func (printCallback) OnComplete()                 {}

// runSingleMessage runs a single message non-interactiveively.
func runSingleMessage(runner ui.AgentRunner, message string) {
	log.Infof("Running single message: %s", message)

	ctx := context.Background()

	fmt.Println("💬 Processing your request...")
	fmt.Println()

	if err := runner.RunWithCallback(ctx, message, printCallback{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runTUIChat starts the interactive TUI chat mode
func runTUIChat(runner ui.AgentRunner, store storage.Store) {
	// Disable console logging in TUI mode to prevent interference with TUI rendering
	// but keep file logging for diagnostics.
	cfg := configManager.Get()
	logFile := cfg.Log.File
	if logFile == "" {
		homeDir, _ := os.UserHomeDir()
		logFile = filepath.Join(homeDir, ".gline", "gline.log")
	}
	log.Init(log.Config{
		Level:   cfg.Log.Level,
		File:    logFile,
		Console: false,
		Color:   false,
	})

	if err := ui.Run(runner, store); err != nil {
		log.Errorf("TUI error: %v", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
