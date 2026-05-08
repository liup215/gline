package api

import (
	"fmt"
	"sync"

	"github.com/liup215/gline/internal/agent"
)

// ProviderFactory is a function that creates a provider instance
type ProviderFactory func(config ProviderConfig) (agent.Provider, error)

// ProviderConfig contains configuration for creating a provider
type ProviderConfig struct {
	APIKey      string
	Model       string
	BaseURL     string
	MaxTokens   int
	Temperature float64
}

// Registry manages provider factories
type Registry struct {
	factories map[string]ProviderFactory
	mu        sync.RWMutex
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ProviderFactory),
	}
}

// Register adds a provider factory to the registry
func (r *Registry) Register(name string, factory ProviderFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("provider already registered: %s", name)
	}

	r.factories[name] = factory
	return nil
}

// Get retrieves a provider factory by name
func (r *Registry) Get(name string) (ProviderFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[name]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", name)
	}

	return factory, nil
}

// Create creates a provider instance
func (r *Registry) Create(name string, config ProviderConfig) (agent.Provider, error) {
	factory, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	return factory(config)
}

// List returns all registered provider names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered providers
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.factories)
}

// DefaultRegistry is the global default registry
var DefaultRegistry = NewRegistry()

// RegisterDefault registers a provider in the default registry
func RegisterDefault(name string, factory ProviderFactory) error {
	return DefaultRegistry.Register(name, factory)
}

// CreateDefault creates a provider from the default registry
func CreateDefault(name string, config ProviderConfig) (agent.Provider, error) {
	return DefaultRegistry.Create(name, config)
}

// InitDefaultRegistry initializes the default registry with built-in providers
func InitDefaultRegistry() {
	// Register Anthropic provider
	RegisterDefault("anthropic", func(config ProviderConfig) (agent.Provider, error) {
		return NewAnthropicProvider(config.APIKey, config.Model), nil
	})

	// TODO: Register other providers (OpenAI, OpenRouter, etc.)
}

// GetProvider creates a provider instance with the given configuration
func GetProvider(name string, apiKey string, model string) (agent.Provider, error) {
	// Initialize registry if not already done
	if DefaultRegistry.Count() == 0 {
		InitDefaultRegistry()
	}

	config := ProviderConfig{
		APIKey: apiKey,
		Model:  model,
	}

	return DefaultRegistry.Create(name, config)
}

// SupportedProviders returns a list of supported provider names
func SupportedProviders() []string {
	// Initialize registry if not already done
	if DefaultRegistry.Count() == 0 {
		InitDefaultRegistry()
	}
	return DefaultRegistry.List()
}

// IsProviderSupported checks if a provider is supported
func IsProviderSupported(name string) bool {
	providers := SupportedProviders()
	for _, p := range providers {
		if p == name {
			return true
		}
	}
	return false
}
