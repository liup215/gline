package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadFileTool reads the contents of a file
type ReadFileTool struct {
	BaseTool
}

// ReadFileInput represents the input for read_file tool
type ReadFileInput struct {
	Path string `json:"path"`
}

// NewReadFileTool creates a new read_file tool
func NewReadFileTool() *ReadFileTool {
	return &ReadFileTool{
		BaseTool: BaseTool{
			name:        "read_file",
			description: "Read the contents of a file at the specified path. Use this when you need to examine the contents of an existing file.",
			inputSchema: PathSchema,
		},
	}
}

// Execute reads the file
func (t *ReadFileTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req ReadFileInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}

	if req.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Clean the path
	path := filepath.Clean(req.Path)

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", path)
	}

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Check file size (limit to ~100KB to avoid huge outputs)
	const maxSize = 100 * 1024
	if len(content) > maxSize {
		return fmt.Sprintf("%s...\n\n[File truncated: %d bytes total, showing first %d bytes]",
			string(content[:maxSize]), len(content), maxSize), nil
	}

	return string(content), nil
}

// WriteFileTool writes content to a file
type WriteFileTool struct {
	BaseTool
}

// WriteFileInput represents the input for write_to_file tool
type WriteFileInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// NewWriteFileTool creates a new write_to_file tool
func NewWriteFileTool() *WriteFileTool {
	return &WriteFileTool{
		BaseTool: BaseTool{
			name:        "write_to_file",
			description: "Write content to a file at the specified path. If the file exists, it will be overwritten. Use this when creating new files or completely rewriting existing files.",
			inputSchema: PathAndContentSchema,
		},
	}
}

// Execute writes the file
func (t *WriteFileTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req WriteFileInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}

	if req.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Clean the path
	path := filepath.Clean(req.Path)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file exists
	existed := false
	if _, err := os.Stat(path); err == nil {
		existed = true
	}

	// Write file
	if err := os.WriteFile(path, []byte(req.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	action := "created"
	if existed {
		action = "overwritten"
	}

	return fmt.Sprintf("File %s successfully: %s (%d bytes)", action, path, len(req.Content)), nil
}

// ReplaceInFileTool replaces content in a file using SEARCH/REPLACE blocks
type ReplaceInFileTool struct {
	BaseTool
}

// ReplaceInFileInput represents the input for replace_in_file tool
type ReplaceInFileInput struct {
	Path   string `json:"path"`
	Search string `json:"search"`
	Replace string `json:"replace"`
}

// NewReplaceInFileTool creates a new replace_in_file tool
func NewReplaceInFileTool() *ReplaceInFileTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path of the file to modify"
			},
			"search": {
				"type": "string",
				"description": "The exact content to search for (including whitespace)"
			},
			"replace": {
				"type": "string",
				"description": "The content to replace with"
			}
		},
		"required": ["path", "search", "replace"]
	}`)

	return &ReplaceInFileTool{
		BaseTool: BaseTool{
			name:        "replace_in_file",
			description: "Replace specific content in a file using exact search/replace. Use this for targeted modifications to existing files.",
			inputSchema: schema,
		},
	}
}

// Execute replaces content in the file
func (t *ReplaceInFileTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req ReplaceInFileInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}

	if req.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Clean the path
	path := filepath.Clean(req.Path)

	// Read existing file
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Perform replacement
	oldContent := string(content)
	if !strings.Contains(oldContent, req.Search) {
		return "", fmt.Errorf("search content not found in file")
	}

	newContent := strings.Replace(oldContent, req.Search, req.Replace, 1)

	// Write back
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("File modified successfully: %s", path), nil
}

// ListFilesTool lists files and directories
type ListFilesTool struct {
	BaseTool
}

// ListFilesInput represents the input for list_files tool
type ListFilesInput struct {
	Path string `json:"path"`
}

// NewListFilesTool creates a new list_files tool
func NewListFilesTool() *ListFilesTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The directory path to list"
			}
		},
		"required": ["path"]
	}`)

	return &ListFilesTool{
		BaseTool: BaseTool{
			name:        "list_files",
			description: "List files and directories at the specified path. Use this to explore the file system.",
			inputSchema: schema,
		},
	}
}

// Execute lists files
func (t *ListFilesTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req ListFilesInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}

	if req.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Clean the path
	path := filepath.Clean(req.Path)

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path not found: %s", path)
		}
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", path)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Contents of %s:\n\n", path))

	err = listFiles(path, &result)

	if err != nil {
		return "", err
	}

	return result.String(), nil
}

func listFiles(path string, result *strings.Builder) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		prefix := "  "
		if entry.IsDir() {
			prefix = "📁 "
		} else {
			prefix = "📄 "
		}
		result.WriteString(fmt.Sprintf("%s%s\n", prefix, entry.Name()))
	}

	return nil
}

