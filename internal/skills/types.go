// Package skills provides skill loading and registration for gline.
// Skills follow the cline agent skill specification:
// each skill is a directory containing a SKILL.md file with YAML frontmatter.
package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liup215/gline/pkg/types"
)

var (
	// DefaultSkillDirs defines the well-known directories where skills are
	// searched.  Later directories have higher priority (they override earlier
	// entries with the same skill name).
	DefaultSkillDirs = []string{
		filepath.Join(mustUserHomeDir(), ".gline", "skills"),
		filepath.Join(mustUserHomeDir(), ".agents", "skills"),
		filepath.Join(mustUserHomeDir(), ".cline", "skills"),
		filepath.Join(mustUserHomeDir(), ".claude", "skills"),
	}
)

func mustUserHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

// Validate checks that a skill has the required fields.
func Validate(s *types.Skill) error {
	if s == nil {
		return fmt.Errorf("skill is nil")
	}
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("skill name is required")
	}
	if strings.TrimSpace(s.Description) == "" {
		return fmt.Errorf("skill description is required")
	}
	// Normalise name early so that keys and directory names stay consistent.
	s.Name = strings.ToLower(strings.TrimSpace(s.Name))
	return nil
}
