package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/api"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/prompts"
	"github.com/liup215/gline/internal/skills"
	"github.com/liup215/gline/internal/storage"
	"github.com/liup215/gline/internal/subagent"
	"github.com/liup215/gline/internal/summarizer"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/internal/ui"
)

// initializeAgent creates and configures the agent based on configuration
func initializeAgent() (*agent.BaseAgent, error) {
	cfg := configManager.Get()

	// Get provider configuration
	providerName := cfg.Provider.Default
	if providerName == "" {
		providerName = "openai"
	}

	// Create the appropriate provider
	var provider agent.Provider
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
		baseURL := cfg.Provider.OpenAI.BaseURL
		provider = api.NewOpenAIProvider(apiKey, model, baseURL)
		log.Infof("Using OpenAI provider with model: %s", model)

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
		var err error
		provider, err = api.NewGoLLMProvider(apiKey, model, baseURL, "opencode-go")
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenCode Go provider: %w", err)
		}
		log.Infof("Using OpenCode Go provider with model: %s", model)

	case "openrouter":
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OpenRouter API key not configured. Set OPENROUTER_API_KEY environment variable")
		}
		model := "anthropic/claude-sonnet-4"
		var err error
		provider, err = api.NewGoLLMProvider(apiKey, model, "https://openrouter.ai/api/v1", "openrouter")
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenRouter provider: %w", err)
		}
		log.Infof("Using OpenRouter provider with model: %s", model)

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
		var err error
		provider, err = api.NewGoLLMProvider(apiKey, model, baseURL, "volcano")
		if err != nil {
			return nil, fmt.Errorf("failed to create Volcano provider: %w", err)
		}
		log.Infof("Using Volcano provider with model: %s", model)

	case "mock":
		// Mock provider for testing streaming and tool calls
		scenario := os.Getenv("GLINE_MOCK_SCENARIO")
		if scenario == "" {
			scenario = "tool_call"
		}
		provider = api.NewMockProvider(api.MockScenario(scenario), 0, 0)
		log.Infof("Using Mock provider with scenario: %s", scenario)

	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	// Load custom rules from global and workspace directories
	customRules, _ := prompts.LoadCustomRules()

	// Initialize skills registry
	skillReg := skills.NewRegistry()
	skillReg.LoadFromDirs(skills.DefaultSkillDirs...)

	// Create tool registry first (without summarizer, to break the circular
	// dependency between registry -> subagent builder -> summarizer -> registry).
	registry := tools.InitDefaultRegistry(nil, nil)

	// Create subagent builder with the real registry.
	subBuilder := subagent.NewBuilder(provider, registry, "", customRules, skillReg.GetMeta())

	// Create summarizer now that the builder has a valid registry.
	sum := summarizer.NewSummarizer(subagent.NewSummarizerCaller(subBuilder), summarizer.DefaultOptions())

	// Register summarization tool and remaining tools.
	_ = tools.RegisterSummarizeFileTool(registry, sum)
	tools.RegisterSkillTool(registry, skillReg)
	subagent.RegisterTool(registry, provider, registry, "", customRules, skillReg.GetMeta())

	log.Infof("Initialized %d tools", registry.Count())

	// Create persistent storage
	store, err := storage.NewSQLiteStore("")
	if err != nil {
		log.Warnf("Failed to initialize storage: %v", err)
		store = nil // Continue without storage
	}

	// Create agent options
	opts := agent.Options{
		Provider:     provider,
		ToolRegistry: registry,
		Mode:         agent.ModeAct,
		CustomRules:  customRules,
		Store:        store,
	}

	// Create agent
	agentInstance, err := agent.New(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return agentInstance, nil
}

// runSingleMessage runs a single message non-interactively
func runSingleMessage(agentInstance *agent.BaseAgent, message string) {
	log.Infof("Running single message: %s", message)

	ctx := context.Background()

	fmt.Println("💬 Processing your request...")
	fmt.Println()

	err := agentInstance.Run(ctx, message)
	if err != nil {
		log.Errorf("Agent error: %v", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print the conversation
	conversation := agentInstance.GetConversation()
	for _, msg := range conversation.GetMessages() {
		switch msg.Role {
		case "user":
			fmt.Printf("You: %s\n", msg.Content)
		case "assistant":
			fmt.Printf("AI: %s\n", msg.Content)
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					fmt.Printf("  🔧 Tool: %s\n", tc.Name)
				}
			}
		}
	}
}

// runTUIChat starts the interactive TUI chat mode
func runTUIChat(agentInstance *agent.BaseAgent) {
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

	if err := ui.Run(agentInstance); err != nil {
		log.Errorf("TUI error: %v", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
