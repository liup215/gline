// Package api provides LLM provider implementations
package api

import (
	"fmt"
	"strings"

	"github.com/liup215/gline/internal/log"
)

const (
	// OpenCodeGoBaseURL is the base URL for OpenCode Go API
	OpenCodeGoBaseURL = "https://opencode.ai/zen/go/v1"
)

// OpenCodeGoModels lists all available models in OpenCode Go
var OpenCodeGoModels = map[string]string{
	// Chat completions models (OpenAI-compatible)
	"glm-5.3-flash":             "GLM-5.3-Flash",
	"glm-5.3":                   "GLM-5.3",
	"glm-5.2":                   "GLM-5.2",
	"glm-5.1":                   "GLM-5.1",
	"kimi-k3":                   "Kimi K3",
	"kimi-k2.7-code":            "Kimi K2.7 Code",
	"kimi-k2.6":                 "Kimi K2.6",
	"longcat-2.0":               "LongCat-2.0",
	"deepseek-v4.1-flash":       "DeepSeek V4.1 Flash",
	"deepseek-v4-pro":           "DeepSeek V4 Pro",
	"deepseek-v4-flash":         "DeepSeek V4 Flash",
	"deepseek-v4-flash-vision-exp": "DeepSeek V4 Flash Vision Exp",
	"mimo-v2.5":                 "MiMo-V2.5",
	"mimo-v2.5-pro":             "MiMo-V2.5-Pro",
	"hy4-preview":               "Hy4 preview",
	"hy3":                       "Hy3",
	"gpt-5.6-luna":              "GPT 5.6 Luna",
	"grok-4.6":                  "Grok 4.6",
	"union-alpha":               "Union Alpha Free",
}

// OpenCodeGoProvider extends OpenAIProvider with OpenCode Go specific settings
type OpenCodeGoProvider struct {
	*OpenAIProvider
	sessionID string // for x-opencode-session header
}

// NewOpenCodeGoProvider creates a new OpenCode Go provider
func NewOpenCodeGoProvider(apiKey, model, baseURL string) *OpenCodeGoProvider {
	if baseURL == "" {
		baseURL = OpenCodeGoBaseURL
	}

	// Ensure base URL ends with /chat/completions
	if !strings.HasSuffix(baseURL, "/chat/completions") {
		baseURL = strings.TrimSuffix(baseURL, "/") + "/chat/completions"
	}

	// Set default model if not provided
	if model == "" {
		model = "kimi-k2.7-code" // Good default for coding
	}

	log.Infof("OpenCode Go provider: model=%s, baseURL=%s", model, baseURL)

	return &OpenCodeGoProvider{
		OpenAIProvider: NewOpenAIProvider(apiKey, model, baseURL),
		sessionID:      generateSessionID(),
	}
}

// GetProviderName returns the provider name
func (p *OpenCodeGoProvider) GetProviderName() string {
	return "opencode-go"
}

// GetModel returns the current model
func (p *OpenCodeGoProvider) GetModel() string {
	return p.model
}

// generateSessionID creates a simple session ID for x-opencode-session header
func generateSessionID() string {
	// Use a simple random-like ID based on timestamp
	return fmt.Sprintf("gline-%d", 1000)
}

// ListModels returns all available models in OpenCode Go
func ListOpenCodeGoModels() []string {
	models := make([]string, 0, len(OpenCodeGoModels))
	for id := range OpenCodeGoModels {
		models = append(models, id)
	}
	return models
}

// GetModelName returns the display name for a model ID
func GetOpenCodeGoModelName(modelID string) string {
	if name, ok := OpenCodeGoModels[modelID]; ok {
		return name
	}
	return modelID
}

// IsOpenCodeGoModel checks if a model ID is a valid OpenCode Go model
func IsOpenCodeGoModel(modelID string) bool {
	_, ok := OpenCodeGoModels[modelID]
	return ok
}
