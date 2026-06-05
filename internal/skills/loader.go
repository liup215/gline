package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liup215/gline/pkg/types"
	"gopkg.in/yaml.v3"
)

// LoadSkillsFromDir scans dir for skill files (.yaml, .yml, .json) and
// returns a slice of parsed skills.  Files that fail to parse are
// reported through the error but do not prevent other files from loading.
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
		if entry.IsDir() || !IsSkillFile(entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		skill, err := LoadSkillFromFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}
		// Derive name from filename if omitted so that /cmd-name always
		// matches the file stem.  Validation requires a name.
		stem := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if skill.Name == "" {
			skill.Name = strings.ToLower(stem)
		}
		skill.Source = path

		if err := Validate(skill); err != nil {
			errs = append(errs, fmt.Sprintf("%s: invalid: %v", entry.Name(), err))
			continue
		}
		skills = append(skills, skill)
	}

	if len(errs) > 0 {
		return skills, fmt.Errorf("some skill files failed to load:\n  %s", strings.Join(errs, "\n  "))
	}
	return skills, nil
}

// LoadSkillFromFile reads a single skill definition from path.
func LoadSkillFromFile(path string) (*types.Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return parseYAML(data)
	case ".json":
		return parseJSON(data)
	default:
		return nil, fmt.Errorf("unsupported skill file extension: %s", ext)
	}
}

func parseYAML(data []byte) (*types.Skill, error) {
	var s types.Skill
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("yaml parse: %w", err)
	}
	return &s, nil
}

func parseJSON(data []byte) (*types.Skill, error) {
	var s types.Skill
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("json parse: %w", err)
	}
	return &s, nil
}
