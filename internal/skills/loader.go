package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liup215/gline/pkg/types"
	"gopkg.in/yaml.v3"
)

// frontmatterOnly holds the YAML frontmatter fields from SKILL.md.
type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// LoadSkillsFromDir scans dir for skill subdirectories containing SKILL.md.
// Skills are discovered from well-known directories (e.g. ~/.gline/skills).
// According to the cline spec, each skill is a directory with a SKILL.md file.
func LoadSkillsFromDir(dir string) ([]*types.Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read skill directory %s: %w", dir, err)
	}

	var skills []*types.Skill
	var errs []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillDir := filepath.Join(dir, entry.Name())
		skill, err := loadSkillFromDir(skillDir, entry.Name())
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}
		if skill == nil {
			continue // no SKILL.md found in this directory
		}

		if err := Validate(skill); err != nil {
			errs = append(errs, fmt.Sprintf("%s: invalid: %v", entry.Name(), err))
			continue
		}
		skills = append(skills, skill)
	}

	if len(errs) > 0 {
		return skills, fmt.Errorf("some skills failed to load:\n  %s", strings.Join(errs, "\n  "))
	}
	return skills, nil
}

// loadSkillFromDir reads a skill from a skill directory.
// It looks for SKILL.md and parses YAML frontmatter + markdown body.
func loadSkillFromDir(skillDir, dirName string) (*types.Skill, error) {
	skillMdPath := filepath.Join(skillDir, "SKILL.md")
	data, err := os.ReadFile(skillMdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // not a skill directory
		}
		return nil, fmt.Errorf("read SKILL.md: %w", err)
	}

	fm, body, err := parseSkillMarkdown(data)
	if err != nil {
		return nil, fmt.Errorf("parse SKILL.md: %w", err)
	}

	// Name must match directory name per spec
	skillName := strings.ToLower(strings.TrimSpace(fm.Name))
	if skillName == "" {
		skillName = strings.ToLower(strings.TrimSpace(dirName))
	}
	if skillName != strings.ToLower(strings.TrimSpace(dirName)) {
		return nil, fmt.Errorf("skill name %q does not match directory name %q", fm.Name, dirName)
	}

	skill := &types.Skill{
		Name:        skillName,
		Description: strings.TrimSpace(fm.Description),
		Contents:    strings.TrimSpace(body),
		Source:      skillMdPath,
	}

	return skill, nil
}

// parseSkillMarkdown parses a SKILL.md file into frontmatter and body.
// Frontmatter is delimited by --- at the start of the file.
func parseSkillMarkdown(data []byte) (frontmatter, string, error) {
	content := string(data)
	content = strings.TrimSpace(content)

	var fm frontmatter
	var body string

	// Check for YAML frontmatter delimiter
	if !strings.HasPrefix(content, "---") {
		// No frontmatter — the whole file is the body.
		// This shouldn't happen for valid skills, but we handle it gracefully.
		return fm, content, nil
	}

	// Find the closing ---
	rest := strings.TrimPrefix(content, "---")
	rest = strings.TrimPrefix(rest, "\n")
	rest = strings.TrimPrefix(rest, "\r\n")

	endIdx := strings.Index(rest, "---")
	if endIdx == -1 {
		// No closing delimiter — treat everything after first --- as body
		return fm, rest, nil
	}

	fmStr := strings.TrimSpace(rest[:endIdx])
	body = strings.TrimSpace(rest[endIdx+3:])
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimPrefix(body, "\r\n")

	if err := yaml.Unmarshal([]byte(fmStr), &fm); err != nil {
		return fm, body, fmt.Errorf("yaml frontmatter parse: %w", err)
	}

	return fm, body, nil
}
