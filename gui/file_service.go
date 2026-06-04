package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DirEntry represents a file or directory entry for the frontend file browser.
type DirEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
}

// skipDirs contains directory names that should be hidden from the file picker.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".vscode":      true,
	".idea":        true,
	"__pycache__":  true,
	".next":        true,
	".nuxt":        true,
	"dist":         true,
	"vendor":       true,
	".cache":       true,
}

// maxFileReadSize is the maximum bytes to read from a referenced file (1 MB).
const maxFileReadSize = 1024 * 1024

// ListDirEntries returns files and subdirectories under the given relative path.
// dirPath is relative to the project's working directory; empty string means root.
func (c *ChatService) ListDirEntries(dirPath string) ([]DirEntry, error) {
	if c.workingDir == "" {
		return nil, fmt.Errorf("no project directory selected")
	}

	absPath := filepath.Join(c.workingDir, dirPath)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var result []DirEntry
	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files/dirs and common noise directories
		if strings.HasPrefix(name, ".") || skipDirs[name] {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		relPath := filepath.Join(dirPath, name)
		// Use forward slashes for consistency across platforms
		relPath = filepath.ToSlash(relPath)

		result = append(result, DirEntry{
			Name:    name,
			Path:    relPath,
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		})
	}

	// Sort: directories first, then files; alphabetically within each group
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result, nil
}

// ReadFileContent reads a file's content relative to the working directory.
// Returns the file content as a string. Binary files are detected and skipped.
// Files larger than maxFileReadSize are truncated.
func (c *ChatService) ReadFileContent(relPath string) (string, error) {
	if c.workingDir == "" {
		return "", fmt.Errorf("no project directory selected")
	}

	absPath := filepath.Join(c.workingDir, relPath)

	// Check file size first
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	if info.Size() > maxFileReadSize {
		data, err := os.ReadFile(absPath)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return string(data[:maxFileReadSize]) + "\n[... truncated, file too large]", nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Binary file detection: check first 512 bytes for null bytes
	checkLen := len(data)
	if checkLen > 512 {
		checkLen = 512
	}
	for i := 0; i < checkLen; i++ {
		if data[i] == 0 {
			return "[Binary file, content skipped]", nil
		}
	}

	return string(data), nil
}

// SendMessageWithContext sends a message with file references attached.
// fileRefsJSON is a JSON array of relative file paths, e.g. ["src/main.go","README.md"].
// The referenced file contents are read and prepended to the prompt as context.
func (c *ChatService) SendMessageWithContext(prompt string, fileRefsJSON string) error {
	var relPaths []string
	if err := json.Unmarshal([]byte(fileRefsJSON), &relPaths); err != nil {
		return fmt.Errorf("invalid file references JSON: %w", err)
	}

	if len(relPaths) == 0 {
		// No file references, just send normally
		return c.SendMessage(prompt)
	}

	// Build context block from referenced files
	var sb strings.Builder
	sb.WriteString("<referenced_files>\n")

	for _, relPath := range relPaths {
		content, err := c.ReadFileContent(relPath)
		if err != nil {
			sb.WriteString(fmt.Sprintf("<file path=%q>\n[Error reading file: %s]\n</file>\n", relPath, err.Error()))
			continue
		}
		sb.WriteString(fmt.Sprintf("<file path=%q>\n%s\n</file>\n", relPath, content))
	}

	sb.WriteString("</referenced_files>\n\n")
	sb.WriteString(prompt)

	enhancedPrompt := sb.String()
	return c.SendMessage(enhancedPrompt)
}
