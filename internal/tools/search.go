package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SearchFilesTool searches for patterns in files
type SearchFilesTool struct {
	BaseTool
}

// SearchFilesInput represents the input for search_files tool
type SearchFilesInput struct {
	Path        string `json:"path"`
	Regex       string `json:"regex"`
	FilePattern string `json:"file_pattern,omitempty"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Path        string `json:"path"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Match       string `json:"match"`
	Context     string `json:"context"`
	ContextLine int    `json:"context_line"`
}

// SearchFilesOutput represents the output of search_files tool
type SearchFilesOutput struct {
	Results    []SearchResult `json:"results"`
	TotalFiles int            `json:"total_files"`
	TotalMatches int          `json:"total_matches"`
}

// NewSearchFilesTool creates a new search_files tool
func NewSearchFilesTool() *SearchFilesTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The directory to search in"
			},
			"regex": {
				"type": "string",
				"description": "The regex pattern to search for"
			},
			"file_pattern": {
				"type": "string",
				"description": "Optional glob pattern to filter files (e.g., '*.go')"
			}
		},
		"required": ["path", "regex"]
	}`)

	return &SearchFilesTool{
		BaseTool: BaseTool{
			name:        "search_files",
			description: "Search for a regex pattern in files within a directory. Returns context-rich results with file paths, line numbers, and surrounding context.",
			inputSchema: schema,
		},
	}
}

// Execute searches for the pattern
func (t *SearchFilesTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req SearchFilesInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}

	if req.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	if req.Regex == "" {
		return "", fmt.Errorf("regex is required")
	}

	// Compile regex
	pattern, err := regexp.Compile(req.Regex)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Clean path
	path := filepath.Clean(req.Path)

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path not found: %s", path)
		}
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	// If path is a file, search just that file
	var files []string
	if !info.IsDir() {
		files = []string{path}
	} else {
		// Find all files
		files, err = findFiles(path, req.FilePattern)
		if err != nil {
			return "", fmt.Errorf("failed to find files: %w", err)
		}
	}

	// Search in files
	output := &SearchFilesOutput{
		Results:    make([]SearchResult, 0),
		TotalFiles: len(files),
	}

	for _, file := range files {
		results, err := searchInFile(file, pattern)
		if err != nil {
			// Skip files we can't read
			continue
		}
		output.Results = append(output.Results, results...)
		output.TotalMatches += len(results)
	}

	// Format output
	return formatSearchResults(output), nil
}

// findFiles finds all files matching the pattern
func findFiles(root string, pattern string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Apply file pattern filter
		if pattern != "" {
			matched, err := filepath.Match(pattern, info.Name())
			if err != nil {
				return nil
			}
			if !matched {
				return nil
			}
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// searchInFile searches for pattern in a single file
func searchInFile(path string, pattern *regexp.Regexp) ([]SearchResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []SearchResult
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Read all lines for context
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Search each line
	for i, line := range lines {
		lineNum = i + 1
		matches := pattern.FindAllStringIndex(line, -1)

		for _, match := range matches {
			start, end := match[0], match[1]

			// Get context (2 lines before and after)
			contextStart := i - 2
			if contextStart < 0 {
				contextStart = 0
			}
			contextEnd := i + 3
			if contextEnd > len(lines) {
				contextEnd = len(lines)
			}

			var contextLines []string
			for j := contextStart; j < contextEnd; j++ {
				prefix := "  "
				if j == i {
					prefix = "> "
				}
				contextLines = append(contextLines, fmt.Sprintf("%s%d: %s", prefix, j+1, lines[j]))
			}

			result := SearchResult{
				Path:        path,
				Line:        lineNum,
				Column:      start + 1,
				Match:       line[start:end],
				Context:     strings.Join(contextLines, "\n"),
				ContextLine: i - contextStart + 1,
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// formatSearchResults formats search results for display
func formatSearchResults(output *SearchFilesOutput) string {
	if len(output.Results) == 0 {
		return fmt.Sprintf("No matches found in %d files.", output.TotalFiles)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Found %d matches in %d files:\n\n", output.TotalMatches, output.TotalFiles))

	// Group by file
	fileResults := make(map[string][]SearchResult)
	for _, r := range output.Results {
		fileResults[r.Path] = append(fileResults[r.Path], r)
	}

	for path, results := range fileResults {
		builder.WriteString(fmt.Sprintf("📄 %s\n", path))
		for _, r := range results {
			builder.WriteString(fmt.Sprintf("   Line %d, Col %d: %s\n", r.Line, r.Column, r.Match))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// ListCodeDefinitionNamesTool lists code definitions (functions, classes, etc.)
type ListCodeDefinitionNamesTool struct {
	BaseTool
}

// ListCodeDefinitionNamesInput represents the input for list_code_definition_names tool
type ListCodeDefinitionNamesInput struct {
	Path        string `json:"path"`
	FilePattern string `json:"file_pattern,omitempty"`
}

// CodeDefinition represents a code definition
type CodeDefinition struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // function, class, method, etc.
	Line     int    `json:"line"`
	File     string `json:"file"`
	Language string `json:"language"`
}

// NewListCodeDefinitionNamesTool creates a new list_code_definition_names tool
func NewListCodeDefinitionNamesTool() *ListCodeDefinitionNamesTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The directory to analyze"
			},
			"file_pattern": {
				"type": "string",
				"description": "Optional glob pattern to filter files (e.g., '*.go')"
			}
		},
		"required": ["path"]
	}`)

	return &ListCodeDefinitionNamesTool{
		BaseTool: BaseTool{
			name:        "list_code_definition_names",
			description: "List definition names (functions, classes, methods, etc.) in code files within a directory. Provides insights into codebase structure.",
			inputSchema: schema,
		},
	}
}

// Execute lists code definitions
func (t *ListCodeDefinitionNamesTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req ListCodeDefinitionNamesInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}

	if req.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Clean path
	path := filepath.Clean(req.Path)

	// Find files
	files, err := findFiles(path, req.FilePattern)
	if err != nil {
		return "", fmt.Errorf("failed to find files: %w", err)
	}

	// Analyze each file
	var allDefs []CodeDefinition
	for _, file := range files {
		defs := analyzeFile(file)
		allDefs = append(allDefs, defs...)
	}

	// Format output
	return formatCodeDefinitions(allDefs), nil
}

// analyzeFile analyzes a single file for code definitions
func analyzeFile(path string) []CodeDefinition {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var defs []CodeDefinition
	language := detectLanguage(path)

	switch language {
	case "go":
		defs = analyzeGoCode(path, string(content))
	case "python":
		defs = analyzePythonCode(path, string(content))
	case "javascript", "typescript":
		defs = analyzeJavaScriptCode(path, string(content))
	default:
		// Generic analysis
		defs = analyzeGenericCode(path, string(content))
	}

	return defs
}

// detectLanguage detects the programming language from file extension
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".rs":
		return "rust"
	default:
		return "unknown"
	}
}

// analyzeGoCode analyzes Go code for definitions
func analyzeGoCode(path string, content string) []CodeDefinition {
	var defs []CodeDefinition
	lines := strings.Split(content, "\n")

	// Simple regex patterns for Go
	funcPattern := regexp.MustCompile(`^func\s+(\w+)`)
	typePattern := regexp.MustCompile(`^type\s+(\w+)\s+(struct|interface)`)

	for i, line := range lines {
		lineNum := i + 1

		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			defs = append(defs, CodeDefinition{
				Name:     matches[1],
				Type:     "function",
				Line:     lineNum,
				File:     path,
				Language: "go",
			})
		}

		if matches := typePattern.FindStringSubmatch(line); matches != nil {
			defs = append(defs, CodeDefinition{
				Name:     matches[1],
				Type:     matches[2],
				Line:     lineNum,
				File:     path,
				Language: "go",
			})
		}
	}

	return defs
}

// analyzePythonCode analyzes Python code for definitions
func analyzePythonCode(path string, content string) []CodeDefinition {
	var defs []CodeDefinition
	lines := strings.Split(content, "\n")

	// Simple patterns for Python
	funcPattern := regexp.MustCompile(`^def\s+(\w+)`)
	classPattern := regexp.MustCompile(`^class\s+(\w+)`)

	for i, line := range lines {
		lineNum := i + 1

		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			defs = append(defs, CodeDefinition{
				Name:     matches[1],
				Type:     "function",
				Line:     lineNum,
				File:     path,
				Language: "python",
			})
		}

		if matches := classPattern.FindStringSubmatch(line); matches != nil {
			defs = append(defs, CodeDefinition{
				Name:     matches[1],
				Type:     "class",
				Line:     lineNum,
				File:     path,
				Language: "python",
			})
		}
	}

	return defs
}

// analyzeJavaScriptCode analyzes JavaScript/TypeScript code for definitions
func analyzeJavaScriptCode(path string, content string) []CodeDefinition {
	var defs []CodeDefinition
	lines := strings.Split(content, "\n")

	// Simple patterns for JS/TS
	funcPattern := regexp.MustCompile(`^(async\s+)?function\s+(\w+)`)
	classPattern := regexp.MustCompile(`^class\s+(\w+)`)
	arrowFuncPattern := regexp.MustCompile(`(const|let|var)\s+(\w+)\s*=\s*(async\s*)?\(`)

	for i, line := range lines {
		lineNum := i + 1

		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			defs = append(defs, CodeDefinition{
				Name:     matches[2],
				Type:     "function",
				Line:     lineNum,
				File:     path,
				Language: "javascript",
			})
		}

		if matches := classPattern.FindStringSubmatch(line); matches != nil {
			defs = append(defs, CodeDefinition{
				Name:     matches[1],
				Type:     "class",
				Line:     lineNum,
				File:     path,
				Language: "javascript",
			})
		}

		if matches := arrowFuncPattern.FindStringSubmatch(line); matches != nil {
			defs = append(defs, CodeDefinition{
				Name:     matches[2],
				Type:     "function",
				Line:     lineNum,
				File:     path,
				Language: "javascript",
			})
		}
	}

	return defs
}

// analyzeGenericCode provides generic code analysis
func analyzeGenericCode(path string, content string) []CodeDefinition {
	// Fallback: try to find common patterns
	var defs []CodeDefinition
	lines := strings.Split(content, "\n")

	// Generic function pattern
	funcPattern := regexp.MustCompile(`^(func|function|def|void|int|string|bool)\s+(\w+)`)

	for i, line := range lines {
		lineNum := i + 1
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			defs = append(defs, CodeDefinition{
				Name:     matches[2],
				Type:     "function",
				Line:     lineNum,
				File:     path,
				Language: "unknown",
			})
		}
	}

	return defs
}

// formatCodeDefinitions formats code definitions for display
func formatCodeDefinitions(defs []CodeDefinition) string {
	if len(defs) == 0 {
		return "No code definitions found."
	}

	// Group by file
	fileDefs := make(map[string][]CodeDefinition)
	for _, d := range defs {
		fileDefs[d.File] = append(fileDefs[d.File], d)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Found %d definitions:\n\n", len(defs)))

	for file, definitions := range fileDefs {
		builder.WriteString(fmt.Sprintf("📄 %s\n", file))
		for _, d := range definitions {
			icon := "🔧"
			switch d.Type {
			case "class":
				icon = "🏗️"
			case "interface":
				icon = "🔗"
			case "struct":
				icon = "📦"
			}
			builder.WriteString(fmt.Sprintf("   %s %s (%s) at line %d\n", icon, d.Name, d.Type, d.Line))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}
