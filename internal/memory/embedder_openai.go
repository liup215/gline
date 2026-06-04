package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIEmbedder calls any OpenAI-compatible embedding API.
type OpenAIEmbedder struct {
	APIKey     string
	BaseURL    string
	Model      string
	Dim        int
	MaxBatch   int
	HTTPClient *http.Client
}

// NewOpenAIEmbedder creates an embedder.  Defaults to OpenAI official endpoint.
func NewOpenAIEmbedder(apiKey, model string) *OpenAIEmbedder {
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &OpenAIEmbedder{
		APIKey:   apiKey,
		BaseURL:  "https://api.openai.com/v1",
		Model:    model,
		Dim:      1536, // text-embedding-3-small
		MaxBatch: 100,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (e *OpenAIEmbedder) Dimension() int   { return e.Dim }
func (e *OpenAIEmbedder) ModelName() string { return e.Model }
func (e *OpenAIEmbedder) MaxBatchSize() int { return e.MaxBatch }

func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	var all [][]float32
	for i := 0; i < len(texts); i += e.MaxBatch {
		end := i + e.MaxBatch
		if end > len(texts) {
			end = len(texts)
		}
		batch, err := e.embedBatch(ctx, texts[i:end])
		if err != nil {
			return nil, fmt.Errorf("embed batch %d-%d: %w", i, end, err)
		}
		all = append(all, batch...)
	}
	return all, nil
}

func (e *OpenAIEmbedder) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	payload := map[string]interface{}{
		"model": e.Model,
		"input": texts,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", e.BaseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.APIKey)

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embeddings API %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("embeddings API error: %s", result.Error.Message)
	}
	// Re-order by index
	vecs := make([][]float32, len(texts))
	for _, d := range result.Data {
		v := make([]float32, len(d.Embedding))
		for i := range d.Embedding {
			v[i] = float32(d.Embedding[i])
		}
		vecs[d.Index] = v
	}
	return vecs, nil
}

// ─── Ollama Embedder ─────────────────────────────────────────────────────────

// OllamaEmbedder calls a local Ollama server for embeddings.
type OllamaEmbedder struct {
	BaseURL    string
	Model      string
	Dim        int
	MaxBatch   int
	HTTPClient *http.Client
}

// NewOllamaEmbedder creates an embedder for local Ollama.
func NewOllamaEmbedder(model string) *OllamaEmbedder {
	if model == "" {
		model = "nomic-embed-text"
	}
	return &OllamaEmbedder{
		BaseURL:  "http://localhost:11434",
		Model:    model,
		Dim:      768, // nomic-embed-text
		MaxBatch: 100,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (e *OllamaEmbedder) Dimension() int   { return e.Dim }
func (e *OllamaEmbedder) ModelName() string { return e.Model }
func (e *OllamaEmbedder) MaxBatchSize() int { return e.MaxBatch }

func (e *OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	var all [][]float32
	for _, text := range texts {
		payload := map[string]interface{}{
			"model": e.Model,
			"prompt": text,
		}
		body, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, "POST", e.BaseURL+"/api/embeddings", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := e.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		var result struct {
			Embedding []float64 `json:"embedding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		v := make([]float32, len(result.Embedding))
		for i := range result.Embedding {
			v[i] = float32(result.Embedding[i])
		}
		all = append(all, v)
	}
	return all, nil
}
