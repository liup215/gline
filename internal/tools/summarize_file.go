package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/liup215/gline/internal/summarizer"
	"github.com/liup215/gline/pkg/types"
)

// SummarizeFileTool asks the summarizer to produce a structured summary of a
// large file. It is useful when search_files/read_file cannot return enough
// detail without overflowing the context window.
type SummarizeFileTool struct {
	BaseTool
	summarizer *summarizer.Summarizer
}

// NewSummarizeFileTool creates a new summarize_file tool.
func NewSummarizeFileTool(s *summarizer.Summarizer) *SummarizeFileTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path of the file to summarize."
			}
		},
		"required": ["path"]
	}`)

	return &SummarizeFileTool{
		BaseTool: BaseTool{
			name:        types.ToolSummarizeFile.String(),
			description: "Produce a compact structured summary of a large file. Use this when the file is too big to read in full and you only need a high-level overview of its purpose, key definitions, and important logic.",
			inputSchema: schema,
		},
		summarizer: s,
	}
}

// Execute summarizes the requested file.
func (t *SummarizeFileTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req SummarizeFileToolInput
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("failed to parse summarize_file input: %w", err)
	}
	if req.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	content, err := os.ReadFile(req.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", req.Path)
		}
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	summary, err := t.summarizer.SummarizeFile(ctx, req.Path, content)
	if err != nil {
		return "", fmt.Errorf("summarization failed: %w", err)
	}
	return summary, nil
}

// RegisterSummarizeFileTool registers the summarize_file tool in the given registry.
func RegisterSummarizeFileTool(registry *Registry, s *summarizer.Summarizer) error {
	return registry.Register(&ToolInfo{
		Tool:                 NewSummarizeFileTool(s),
		Category:             CategorySearch,
		AllowedModes:         []string{"plan", "act"},
		RequiresConfirmation: false,
	})
}
