package prompts

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	globalRulesSubdir    = "rules"
	workspaceRulesDir    = ".gline/rules"
	supportedExtMd       = ".md"
	supportedExtTxt      = ".txt"
)

// LoadCustomRules loads custom rules from both global and workspace directories.
// Global rules are loaded from ~/.gline/rules/ and workspace rules from ./.gline/rules/.
// Rules are appended to the system prompt.
func LoadCustomRules() (string, error) {
	var sections []string

	// Load global rules
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalDir := filepath.Join(homeDir, ".gline", globalRulesSubdir)
		globalContent := loadRulesFromDir(globalDir)
		if globalContent != "" {
			sections = append(sections, "# Global Rules\n\n"+globalContent)
		}
	}

	// Load workspace rules
	workspaceContent := loadRulesFromDir(workspaceRulesDir)
	if workspaceContent != "" {
		sections = append(sections, "# Workspace Rules\n\n"+workspaceContent)
	}

	if len(sections) == 0 {
		return "", nil
	}

	return strings.Join(sections, "\n\n"), nil
}

// loadRulesFromDir scans a directory for .md and .txt files, reads their contents,
// and combines them into a single string. Files are sorted alphabetically.
// Subdirectories and files with unsupported extensions are skipped.
// Empty files and files that fail to read are silently skipped.
func loadRulesFromDir(dir string) string {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return ""
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != supportedExtMd && ext != supportedExtTxt {
			continue
		}
		files = append(files, name)
	}

	if len(files) == 0 {
		return ""
	}

	sort.Strings(files)

	var parts []string
	for _, name := range files {
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		trimmed := strings.TrimSpace(string(content))
		if trimmed == "" {
			continue
		}
		parts = append(parts, "## "+name+"\n\n"+trimmed)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n\n")
}
