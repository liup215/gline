package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	globalRulesSubdir = "rules"
	workspaceRulesDir = ".gline/rules"
	supportedExtMd    = ".md"
	supportedExtTxt     = ".txt"
)

// RuleFileInfo holds metadata about a loaded rule file.
type RuleFileInfo struct {
	Name    string `json:"name"`
	Source  string `json:"source"` // "global" or "workspace"
	Size    int64  `json:"size"`   // file size in bytes
	ModTime int64  `json:"modTime"` // unix timestamp (seconds)
}

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

// LoadCustomRulesWithMeta loads custom rules and returns both the combined content
// and metadata about each loaded rule file.
func LoadCustomRulesWithMeta() (string, []RuleFileInfo, error) {
	var sections []string
	var infos []RuleFileInfo

	// Load global rules
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalDir := filepath.Join(homeDir, ".gline", globalRulesSubdir)
		content, meta := loadRulesFromDirWithMeta(globalDir, "global")
		if content != "" {
			sections = append(sections, "# Global Rules\n\n"+content)
		}
		infos = append(infos, meta...)
	}

	// Load workspace rules
	workspaceContent, meta := loadRulesFromDirWithMeta(workspaceRulesDir, "workspace")
	if workspaceContent != "" {
		sections = append(sections, "# Workspace Rules\n\n"+workspaceContent)
	}
	infos = append(infos, meta...)

	if len(sections) == 0 {
		return "", infos, nil
	}

	return strings.Join(sections, "\n\n"), infos, nil
}

// GetCustomRulesInfo returns metadata about available rule files without reading their contents.
func GetCustomRulesInfo() ([]RuleFileInfo, error) {
	var infos []RuleFileInfo

	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalDir := filepath.Join(homeDir, ".gline", globalRulesSubdir)
		infos = append(infos, listRuleFiles(globalDir, "global")...)
	}

	infos = append(infos, listRuleFiles(workspaceRulesDir, "workspace")...)
	return infos, nil
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

// loadRulesFromDirWithMeta is like loadRulesFromDir but also returns metadata for each file.
func loadRulesFromDirWithMeta(dir string, source string) (string, []RuleFileInfo) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "", nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil
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
		return "", nil
	}

	sort.Strings(files)

	var parts []string
	var infos []RuleFileInfo
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

		// Collect metadata
		if fi, err := os.Stat(path); err == nil {
			infos = append(infos, RuleFileInfo{
				Name:    name,
				Source:  source,
				Size:    fi.Size(),
				ModTime: fi.ModTime().Unix(),
			})
		}
	}

	if len(parts) == 0 {
		return "", nil
	}

	return strings.Join(parts, "\n\n"), infos
}

// listRuleFiles returns metadata about rule files in a directory without reading contents.
func listRuleFiles(dir string, source string) []RuleFileInfo {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var infos []RuleFileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != supportedExtMd && ext != supportedExtTxt {
			continue
		}

		path := filepath.Join(dir, name)
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		infos = append(infos, RuleFileInfo{
			Name:    name,
			Source:  source,
			Size:    fi.Size(),
			ModTime: fi.ModTime().Unix(),
		})
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})

	return infos
}

// FormatRulesInfo returns a human-readable summary of loaded rules.
func FormatRulesInfo(infos []RuleFileInfo) string {
	if len(infos) == 0 {
		return "No custom rules loaded.\n\nPlace `.md` or `.txt` files in:\n- `~/.gline/rules/` (global)\n- `.gline/rules/` (workspace)"
	}

	var b strings.Builder
	b.WriteString("### 📋 Loaded Custom Rules\n\n")
	b.WriteString("| File | Source | Size |\n")
	b.WriteString("|------|--------|------|\n")
	for _, info := range infos {
		sizeStr := ""
		if info.Size < 1024 {
			sizeStr = fmt.Sprintf("%d B", info.Size)
		} else if info.Size < 1024*1024 {
			sizeStr = fmt.Sprintf("%.1f KB", float64(info.Size)/1024)
		} else {
			sizeStr = fmt.Sprintf("%.1f MB", float64(info.Size)/(1024*1024))
		}
		mod := time.Unix(info.ModTime, 0).Format("2006-01-02 15:04")
		b.WriteString(fmt.Sprintf("| **%s** | %s | %s (%s) |\n", info.Name, info.Source, sizeStr, mod))
	}
	return b.String()
}
