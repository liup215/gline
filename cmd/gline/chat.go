package main

import (
	"fmt"
	"os"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/api"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/mcp"
	"github.com/liup215/gline/internal/prompts"
	"github.com/liup215/gline/internal/skills"
	"github.com/liup215/gline/internal/storage"
	"github.com/liup215/gline/internal/subagent"
	"github.com/liup215/gline/internal/tools"
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

	// Create tool registry with all built-in tools
	registry := tools.InitDefaultRegistry()

	// Initialize skills registry and wire up use_skill tool
	skillReg := skills.NewRegistry()
	skillReg.LoadFromDirs(skills.DefaultSkillDirs...)
	if skillReg.Count() > 0 {
		log.Infof("Loaded %d skills", skillReg.Count())
	}
	tools.RegisterSkillTool(registry, skillReg)
	subagent.RegisterTool(registry, provider, registry, "", customRules, skillReg.GetMeta())
	opts.ToolRegistry = registry
	opts.Skills = skillReg.GetMeta()

	log.Infof("Initialized %d tools", registry.Count())

	// Debug: Log MCP configuration
	log.Infof("MCP config: %d servers configured", len(cfg.MCP.Servers))
	for i, server := range cfg.MCP.Servers {
		log.Infof("  MCP server %d: name=%s, command=%s, url=%s, disabled=%v", 
			i, server.Name, server.Command, server.URL, server.Disabled)
	}

	// Initialize MCP Manager if configured
	if len(cfg.MCP.Servers) > 0 {
		mcpManager := mcp.NewManager(&cfg.MCP, registry)
		if err := mcpManager.Start(nil); err != nil {
			log.Warnf("Failed to start MCP manager: %v", err)
		} else {
			// Get status for logging
			statuses := mcpManager.GetServerStatus()
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
	}

	// Create agent
	agentInstance, err := agent.New(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return agentInstance, nil
}
