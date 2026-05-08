package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/liup215/gline/internal/config"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/version"
)

var (
	// Global flags
	cfgFile string
	verbose bool

	// Global config manager
	configManager *config.Manager
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gline",
	Short: "AI programming assistant CLI tool",
	Long: `gline is an AI programming assistant that helps you write code,
debug issues, and manage projects through natural language conversation.

It supports two modes:
  - Plan Mode: Explore and plan without making changes
  - Act Mode: Execute tasks and modify files

Get started:
  gline chat "How do I implement a REST API in Go?"
  gline              # Start interactive session`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize configuration
		configManager = config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize logging
		cfg := configManager.Get()
		logConfig := log.Config{
			Level:   cfg.Log.Level,
			File:    cfg.Log.File,
			Console: true,
			Color:   true,
		}

		// Override with verbose flag
		if verbose {
			logConfig.Level = "debug"
		}

		if err := log.Init(logConfig); err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		log.Debug("Configuration loaded successfully")
		log.Debugf("Log level: %s", log.GetLevel())

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand, start interactive mode
		if len(args) == 0 {
			startInteractiveMode()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gline/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	}
}

// startInteractiveMode starts the interactive TUI mode
func startInteractiveMode() {
	log.Info("Starting gline in interactive mode")
	fmt.Println("🚀 Welcome to gline!")
	fmt.Println()
	fmt.Println("Interactive mode is not yet implemented.")
	fmt.Println("Use 'gline chat <message>' to start a conversation.")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  gline chat <message>  Start a chat session")
	fmt.Println("  gline config          Manage configuration")
	fmt.Println("  gline version         Show version information")
	fmt.Println("  gline --help          Show all options")
}

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Start a chat session with the AI assistant",
	Long: `Start a chat session with the AI assistant.

You can provide a message directly:
  gline chat "How do I implement a REST API in Go?"

Or start an interactive chat session:
  gline chat`,
	Example: `  gline chat "Explain this code"
  gline chat --file main.go "Review this code"
  gline chat`,
	Run: func(cmd *cobra.Command, args []string) {
		message := ""
		if len(args) > 0 {
			message = args[0]
		}

		log.Infof("Starting chat with message: %s", message)
		fmt.Println("💬 Chat mode")
		fmt.Println()

		if message != "" {
			fmt.Printf("You: %s\n", message)
		}

		fmt.Println("Chat functionality will be implemented in Phase 2.")
	},
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage gline configuration",
	Long: `Manage gline configuration settings.

Configuration is loaded from (in order of priority):
  1. Workspace config: .gline/config.yaml
  2. Global config: ~/.gline/config.yaml
  3. Environment variables: GLINE_*`,
}

func init() {
	// Config subcommands
	configCmd.AddCommand(&cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Example: `  gline config get provider.default
  gline config get log.level`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Error: key is required")
				cmd.Usage()
				return
			}
			key := args[0]
			value := configManager.GetString(key)
			fmt.Printf("%s: %s\n", key, value)
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Example: `  gline config set provider.default anthropic
  gline config set log.level debug`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				fmt.Println("Error: key and value are required")
				cmd.Usage()
				return
			}
			key := args[0]
			value := args[1]
			configManager.Set(key, value)
			if err := configManager.Save(); err != nil {
				log.Errorf("Failed to save config: %v", err)
				os.Exit(1)
			}
			log.Infof("Set %s = %s", key, value)
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := configManager.Get()
			fmt.Println("Current configuration:")
			fmt.Println()
			fmt.Printf("Provider:\n")
			fmt.Printf("  Default: %s\n", cfg.Provider.Default)
			fmt.Printf("  Anthropic Model: %s\n", cfg.Provider.Anthropic.Model)
			fmt.Printf("  OpenAI Model: %s\n", cfg.Provider.OpenAI.Model)
			fmt.Println()
			fmt.Printf("UI:\n")
			fmt.Printf("  Theme: %s\n", cfg.UI.Theme)
			fmt.Printf("  Animations: %v\n", cfg.UI.Animations)
			fmt.Println()
			fmt.Printf("Log:\n")
			fmt.Printf("  Level: %s\n", cfg.Log.Level)
			fmt.Printf("  File: %s\n", cfg.Log.File)
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Show configuration file paths",
		Run: func(cmd *cobra.Command, args []string) {
			homeDir, _ := os.UserHomeDir()
			fmt.Println("Configuration files:")
			fmt.Printf("  Global: %s/.gline/config.yaml\n", homeDir)
			fmt.Printf("  Workspace: .gline/config.yaml (in current directory)\n")
		},
	})
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information about gline.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.String())
	},
}
