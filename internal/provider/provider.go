// Package provider creates ADK model.LLM instances for gline's supported
// LLM backends. It is a lean port of pi-go's provider package: the
// OpenAI-compatible family (chat completions + responses API), Anthropic,
// OpenRouter and OpenCode Go. gline drops pi-go's codex OAuth backend,
// rate-limit pacing, HTTP tracing and model catalog — API keys and explicit
// base URLs cover everything gline needs.
package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/adk/v2/model"

	"github.com/liup215/gline/internal/log"
)

// LLMOptions carries optional per-client configuration.
type LLMOptions struct {
	// ExtraHeaders are injected into every outgoing request (below any
	// transport-level wrapping).
	ExtraHeaders map[string]string

	// MaxOutputTokens caps the reply, in tokens. Zero uses the provider
	// default.
	MaxOutputTokens int64

	// UseLegacyMaxTokens sends max_tokens instead of max_completion_tokens
	// on the Chat Completions wire (Ollama-class backends only understand
	// the legacy field).
	UseLegacyMaxTokens bool

	// AdvisorModel enables the Anthropic advisor tool (beta) when non-empty.
	AdvisorModel string
	// AdvisorMaxUses caps advisor calls per request (0 = unlimited).
	AdvisorMaxUses int
	// AdvisorCaching enables ephemeral prompt caching for the advisor.
	AdvisorCaching bool
	// DisablePromptCaching turns OFF Anthropic cache_control breakpoints.
	DisablePromptCaching bool

	// RawBaseURL disables the OpenAI "/v1" suffix normalization for a
	// custom baseURL. Needed by endpoints like Volcano Ark where chat
	// completions live directly at {base}/chat/completions.
	RawBaseURL bool
}

// BuildTransport returns an http.RoundTripper injecting opts.ExtraHeaders.
// It returns (nil, nil) when there is nothing to inject, so callers can leave
// the SDK's own client in place.
func BuildTransport(opts *LLMOptions) (http.RoundTripper, error) {
	if opts == nil || len(opts.ExtraHeaders) == 0 {
		return nil, nil
	}
	return &headerTransport{base: http.DefaultTransport, headers: opts.ExtraHeaders}, nil
}

// headerTransport injects fixed headers into every request.
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	return t.base.RoundTrip(req)
}

// normalizeBaseURL adds an http:// scheme when the caller omitted one.
func normalizeBaseURL(baseURL string) string {
	if baseURL == "" {
		return ""
	}
	if !strings.Contains(baseURL, "://") {
		return "http://" + baseURL
	}
	return baseURL
}

// errorBodyLoggingTransport wraps an http.RoundTripper and, on >=400
// responses, reads the body, logs it, and replaces the response body with a
// buffer so the SDK can still consume it. Providers whose streaming error
// channel strips the response body surface far more useful messages this way.
type errorBodyLoggingTransport struct{ base http.RoundTripper }

func (t *errorBodyLoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp == nil || resp.StatusCode < 400 || resp.Body == nil {
		return resp, err
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		return resp, err
	}
	log.Debugf("LLM backend %d on %s %s: %s",
		resp.StatusCode, req.Method, req.URL.Path, truncate(string(body), 1024))
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// Provider routing ----------------------------------------------------------------

// volcanoDefaultBaseURL is Volcano Engine Ark's OpenAI-compatible endpoint.
// gline treats "volcano" as an OpenAI chat-completions backend pointed at it.
const volcanoDefaultBaseURL = "https://ark.cn-beijing.volces.com/api/plan/v3"

// NewLLM creates a model.LLM for the given provider name, model ID, API key
// and optional base URL. thinkingLevel adjusts provider-specific reasoning
// effort ("" for the provider default).
func NewLLM(ctx context.Context, providerName, modelName, apiKey, baseURL, thinkingLevel string, opts *LLMOptions) (model.LLM, error) {
	switch providerName {
	case "opencode":
		return NewOpenCode(ctx, modelName, apiKey, baseURL, thinkingLevel, opts)
	case "openai":
		return NewOpenAI(ctx, modelName, apiKey, baseURL, opts)
	case "volcano":
		if baseURL == "" {
			baseURL = volcanoDefaultBaseURL
		}
		if opts == nil {
			opts = &LLMOptions{}
		}
		opts.RawBaseURL = true // Ark paths carry their own API version
		return NewOpenAI(ctx, modelName, apiKey, baseURL, opts)
	case "openrouter":
		return NewOpenRouter(ctx, modelName, apiKey, baseURL, thinkingLevel, opts)
	case "anthropic":
		return NewAnthropic(ctx, modelName, apiKey, baseURL, thinkingLevel, opts)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

// APIKeyEnvVar returns the environment variable a provider's key is read from
// when no key is configured.
func APIKeyEnvVar(providerName string) string {
	switch providerName {
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "openai", "volcano":
		return "OPENAI_API_KEY"
	case "openrouter":
		return "OPENROUTER_API_KEY"
	case "opencode":
		return "OPENCODE_API_KEY"
	default:
		return ""
	}
}

// KnownProviderPrefixes lists known vendor and gateway prefixes that may wrap
// model names in layered routing configurations (e.g. agentgateway/openai/...).
var KnownProviderPrefixes = []string{
	"agentgateway/",
	"anthropic/",
	"openai/",
	"gemini/",
	"google/",
	"mistral/",
	"xai/",
	"grok/",
	"ollama/",
	"ollama1/",
	"ollama2/",
	"ollama3/",
	"ollama-cloud/",
	"azure/",
	"opencode/",
	"openrouter/",
}

// StripKnownProviderPrefixes removes known provider prefix wrappers from a
// model name iteratively (e.g. "agentgateway/openai/gpt-5.6-luna" -> "gpt-5.6-luna").
// It preserves the casing of the un-prefixed model identifier.
func StripKnownProviderPrefixes(modelName string) string {
	current := modelName
	for {
		stripped := false
		lower := strings.ToLower(current)
		for _, p := range KnownProviderPrefixes {
			if strings.HasPrefix(lower, p) {
				current = current[len(p):]
				stripped = true
				break
			}
		}
		if !stripped {
			break
		}
	}
	return current
}
