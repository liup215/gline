package main

import (
	"context"
	"fmt"
	"os"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/api"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/prompts"
	"github.com/liup215/gline/internal/skills"
	"github.com/liup215/gline/internal/storage"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
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
	customRules, err := prompts.LoadCustomRules()
	if err != nil {
		log.Warnf("Failed to load custom rules: %v", err)
	} else if customRules != "" {
		log.Info("Custom rules loaded successfully")
	}

	// Create persistent storage
	store, err := storage.NewSQLiteStore("")
	if err != nil {
		log.Warnf("Failed to initialize storage: %v", err)
		store = nil // Continue without storage
	}

	// Create agent options
	opts := agent.Options{
		Provider:    provider,
		Mode:        agent.ModeAct,
		CustomRules: customRules,
		Store:       store,
	}

	// Optionally initialise memory engine if embedding is configured
	memoryEngine, err := newMemoryEngine()
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
		opts.MemoryEngine = memoryEngine
		log.Info("Memory engine initialised")
	}

	// Create tool registry with all built-in tools (including memory if available)
	var registry *tools.Registry
	if memoryEngine != nil {
		registry = tools.InitDefaultRegistry(memoryEngine)
	} else {
		registry = tools.InitDefaultRegistry()
	}

	// Initialize skills registry and wire up use_skill tool
	skillReg := skills.NewRegistry()
	skillReg.LoadFromDirs(skills.DefaultSkillDirs...)
	if skillReg.Count() > 0 {
		log.Infof("Loaded %d skills", skillReg.Count())
	}
	tools.RegisterSkillTool(registry, skillReg)
	opts.ToolRegistry = registry
	opts.Skills = skillReg.GetMeta()

	log.Infof("Initialized %d tools", registry.Count())

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

// runGUIChat starts the GUI desktop application
func runGUIChat(agentInstance *agent.BaseAgent) {
	log.Info("Starting gline GUI application")
	fmt.Println("🚀 Starting gline GUI application...")
	fmt.Println("If the window doesn't open, make sure you have WebView2 installed on Windows.")
	fmt.Println()
	
	// Disable console logging in GUI mode
	log.SetConsoleOutput(false)

	// Import gui package and start the application
	// This is done through the gui main package, not directly here
	fmt.Println("Please run 'gline-gui' or use the desktop shortcut to start the GUI.")
	fmt.Println("The gline GUI is started by running 'gline' without any arguments.")
}
