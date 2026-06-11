// Package config provides configuration management for gline.
// It supports three levels of configuration with the following priority:
// 1. Workspace config (.gline/config.yaml in current directory)
// 2. Global config (~/.gline/config.yaml)
// 3. Environment variables (GLINE_* prefix)
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for gline
type Config struct {
	// LLM Provider settings
	Provider ProviderConfig `mapstructure:"provider"`

	// UI settings
	UI UIConfig `mapstructure:"ui"`

	// Logging settings
	Log LogConfig `mapstructure:"log"`

	// Memory / knowledge base settings
	Memory MemoryConfig `mapstructure:"memory"`
}

// ProviderConfig holds LLM provider settings
type ProviderConfig struct {
	// Default provider to use (openai)
	Default string `mapstructure:"default"`

	// OpenAI provider settings
	OpenAI ProviderSettings `mapstructure:"openai"`
}

// ProviderSettings holds settings for a specific LLM provider
type ProviderSettings struct {
	// API Key for the provider
	APIKey string `mapstructure:"api_key"`

	// Model to use
	Model string `mapstructure:"model"`

	// Base URL for API (optional, for custom endpoints)
	BaseURL string `mapstructure:"base_url"`

	// Max context tokens for this provider/model (0 = use default based on model)
	MaxContextTokens int `mapstructure:"max_context_tokens"`
}

// UIConfig holds UI-related settings
type UIConfig struct {
	// Theme for TUI (default, dark, light)
	Theme string `mapstructure:"theme"`

	// Enable animations in TUI
	Animations bool `mapstructure:"animations"`
}

// LogConfig holds logging settings
type LogConfig struct {
	// Log level (debug, info, warn, error)
	Level string `mapstructure:"level"`

	// Log file path
	File string `mapstructure:"file"`
}

// MemoryConfig holds knowledge base and memory layer settings
type MemoryConfig struct {
	// Enable memory/knowledge base
	Enabled bool `mapstructure:"enabled"`

	// Embedding provider: openai, ollama
	Embedding MemoryEmbeddingConfig `mapstructure:"embedding"`

	// Retrieval parameters
	Retrieval MemoryRetrievalConfig `mapstructure:"retrieval"`
}

// MemoryEmbeddingConfig configures the embedding model used for RAG.
type MemoryEmbeddingConfig struct {
	Provider string `mapstructure:"provider"` // openai | ollama
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
}

// MemoryRetrievalConfig controls how results are fetched from memory layers.
type MemoryRetrievalConfig struct {
	TopK      int     `mapstructure:"top_k"`
	MinScore  float64 `mapstructure:"min_score"`
	MaxTokens int     `mapstructure:"max_tokens"`
}

// Manager handles configuration loading and access
type Manager struct {
	viper      *viper.Viper
	config     *Config
	configPath string
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		viper: viper.New(),
	}
}

// Load loads configuration from all sources
// Priority: workspace > global > environment variables
func (m *Manager) Load() error {
	// Set up viper with defaults
	m.setupDefaults()

	// Load global config first
	if err := m.loadGlobalConfig(); err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// Load workspace config (overrides global)
	if err := m.loadWorkspaceConfig(); err != nil {
		return fmt.Errorf("failed to load workspace config: %w", err)
	}

	// Load environment variables (lowest priority per user requirement)
	m.loadEnvironmentVariables()

	// Unmarshal into struct
	m.config = &Config{}
	if err := m.viper.Unmarshal(m.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// setupDefaults sets default configuration values
func (m *Manager) setupDefaults() {
	m.viper.SetDefault("provider.default", "openai")
	m.viper.SetDefault("ui.theme", "default")
	m.viper.SetDefault("ui.animations", true)
	m.viper.SetDefault("log.level", "info")
	m.viper.SetDefault("log.file", filepath.Join(getGlobalConfigDir(), "logs", "gline.log"))
	m.viper.SetDefault("memory.enabled", true)
	m.viper.SetDefault("memory.embedding.provider", "openai")
	m.viper.SetDefault("memory.embedding.model", "text-embedding-3-small")
	m.viper.SetDefault("memory.retrieval.top_k", 5)
	m.viper.SetDefault("memory.retrieval.min_score", 0.6)
	m.viper.SetDefault("memory.retrieval.max_tokens", 2000)
}

// loadGlobalConfig loads configuration from global config directory
func (m *Manager) loadGlobalConfig() error {
	configDir := getGlobalConfigDir()
	configFile := filepath.Join(configDir, "config.yaml")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config file if it doesn't exist
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := m.createDefaultConfig(configFile); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}
	}

	m.viper.SetConfigFile(configFile)
	m.viper.SetConfigType("yaml")

	// Read config file (ignore if not found)
	if err := m.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	return nil
}

// loadWorkspaceConfig loads configuration from current workspace
func (m *Manager) loadWorkspaceConfig() error {
	workspaceConfig := filepath.Join(".gline", "config.yaml")

	// Check if workspace config exists
	if _, err := os.Stat(workspaceConfig); os.IsNotExist(err) {
		return nil // No workspace config, that's fine
	}

	// Load workspace config (overrides global)
	workspaceViper := viper.New()
	workspaceViper.SetConfigFile(workspaceConfig)
	workspaceViper.SetConfigType("yaml")

	if err := workspaceViper.ReadInConfig(); err != nil {
		return err
	}

	// Merge workspace config into main viper (workspace takes precedence)
	for _, key := range workspaceViper.AllKeys() {
		m.viper.Set(key, workspaceViper.Get(key))
	}

	return nil
}

// loadEnvironmentVariables loads configuration from environment variables
func (m *Manager) loadEnvironmentVariables() {
	m.viper.SetEnvPrefix("GLINE")
	m.viper.AutomaticEnv()

	// Map specific environment variables
	m.viper.BindEnv("provider.openai.api_key", "GLINE_OPENAI_API_KEY")
	m.viper.BindEnv("provider.default", "GLINE_PROVIDER")
	m.viper.BindEnv("log.level", "GLINE_LOG_LEVEL")
}

// createDefaultConfig creates a default configuration file
func (m *Manager) createDefaultConfig(configFile string) error {
	defaultConfig := `# Gline Configuration File
# This is the global configuration file for gline.
# Workspace-specific settings can be placed in .gline/config.yaml

# LLM Provider Settings
provider:
  # Default provider to use (openai)
  default: openai

  # OpenAI settings
  # Supports OpenAI official API, OpenRouter, and any OpenAI-compatible endpoint
  openai:
    # API key (can also be set via GLINE_OPENAI_API_KEY env var)
    api_key: ""
    # Model to use (gpt-4, gpt-4-turbo, gpt-3.5-turbo, etc.)
    model: gpt-4
    # Max context tokens (0 = default ~262K)
    # GPT-4: ~8192 | GPT-4-turbo: ~128000 | GPT-3.5-turbo: ~16000
    max_context_tokens: 0
    # Base URL for API (optional, defaults to OpenAI official API)
    # Examples:
    #   OpenAI: https://api.openai.com/v1
    #   OpenRouter: https://openrouter.ai/api/v1
    #   DashScope: https://dashscope.aliyuncs.com/compatible-mode/v1
    #   Local (Ollama): http://localhost:11434/v1
    base_url: ""

# UI Settings
ui:
  # Theme: default, dark, light
  theme: default
  # Enable animations in TUI
  animations: true

# Logging Settings
log:
  # Log level: debug, info, warn, error
  level: info
  # Log file path
  file: ""
`

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return err
	}

	return os.WriteFile(configFile, []byte(defaultConfig), 0644)
}

// Get returns the loaded configuration
func (m *Manager) Get() *Config {
	return m.config
}

// GetViper returns the underlying viper instance
func (m *Manager) GetViper() *viper.Viper {
	return m.viper
}

// getGlobalConfigDir returns the global configuration directory
func getGlobalConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".gline"
	}
	return filepath.Join(homeDir, ".gline")
}

// Save persists the current configuration to file
func (m *Manager) Save() error {
	if m.configPath == "" {
		m.configPath = filepath.Join(getGlobalConfigDir(), "config.yaml")
	}
	return m.viper.WriteConfigAs(m.configPath)
}

// Set sets a configuration value
func (m *Manager) Set(key string, value interface{}) {
	m.viper.Set(key, value)
}

// GetString gets a string configuration value
func (m *Manager) GetString(key string) string {
	return m.viper.GetString(key)
}

// GetBool gets a boolean configuration value
func (m *Manager) GetBool(key string) bool {
	return m.viper.GetBool(key)
}

// GetInt gets an integer configuration value
func (m *Manager) GetInt(key string) int {
	return m.viper.GetInt(key)
}
