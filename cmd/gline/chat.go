package main

import (
	"context"
	"fmt"
	"os"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/api"
	"github.com/liup215/gline/internal/log"
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
	case "anthropic":
		apiKey := cfg.Provider.Anthropic.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("GLINE_ANTHROPIC_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("Anthropic API key not configured. Set it in config or GLINE_ANTHROPIC_API_KEY environment variable")
		}
		model := cfg.Provider.Anthropic.Model
		if model == "" {
			model = "claude-3-5-sonnet-20241022"
		}
		provider = api.NewAnthropicProvider(apiKey, model)
		log.Infof("Using Anthropic provider with model: %s", model)

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

	// Create tool registry with all built-in tools
	registry := tools.InitDefaultRegistry()

	log.Infof("Initialized %d tools", registry.Count())

	// Create agent options
	opts := agent.Options{
		Provider:     provider,
		ToolRegistry: registry,
		Mode:         agent.ModeAct,
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
	// Logs will still be written to file if configured
	log.SetConsoleOutput(false)

	if err := ui.Run(agentInstance); err != nil {
		log.Errorf("TUI error: %v", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
