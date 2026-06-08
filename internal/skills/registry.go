package skills

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/liup215/gline/pkg/types"
)

// Registry manages loaded skills in a thread-safe manner.
type Registry struct {
	mu     sync.RWMutex
	skills map[string]*types.Skill
}

// NewRegistry creates an empty skill registry.
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]*types.Skill),
	}
}

// Register adds or overwrites a skill.  The skill name is normalised to
// lower case.
func (r *Registry) Register(skill *types.Skill) error {
	if err := Validate(skill); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills[skill.Name] = skill
	return nil
}

// Unregister removes a skill by name.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	name = strings.ToLower(strings.TrimSpace(name))
	delete(r.skills, name)
}

// Get retrieves a skill by name (case-insensitive).
func (r *Registry) Get(name string) (*types.Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name = strings.ToLower(strings.TrimSpace(name))
	s, ok := r.skills[name]
	return s, ok
}

// GetAll returns all registered skills sorted by name.
func (r *Registry) GetAll() []*types.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*types.Skill, 0, len(r.skills))
	for _, s := range r.skills {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// GetMeta returns lightweight SkillMeta summaries for UI lists and system prompts.
func (r *Registry) GetMeta() []types.SkillMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]types.SkillMeta, 0, len(r.skills))
	for _, s := range r.skills {
		result = append(result, types.SkillMeta{
			Name:        s.Name,
			Description: s.Description,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// GetInstructions returns the full instructions (Markdown body) for a skill.
// If the skill is not found, it returns an error with available skill names.
func (r *Registry) GetInstructions(name string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name = strings.ToLower(strings.TrimSpace(name))
	s, ok := r.skills[name]
	if !ok {
		var names []string
		for n := range r.skills {
			names = append(names, n)
		}
		sort.Strings(names)
		return "", fmt.Errorf("skill %q not found. Available skills: %s", name, strings.Join(names, ", "))
	}
	return s.Contents, nil
}

// Count returns the number of registered skills.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

// LoadFromDirs walks the provided directories in order and loads every
// valid skill found.  Later directories override earlier ones when
// skill names collide.
func (r *Registry) LoadFromDirs(dirs ...string) error {
	for _, dir := range dirs {
		skills, err := LoadSkillsFromDir(dir)
		if err != nil {
			// Log/skipped files are acceptable; directory not existing is okay.
			if len(skills) == 0 {
				continue
			}
			// non-fatal: we still register what we managed to load
		}
		for _, s := range skills {
			_ = r.Register(s)
		}
	}
	return nil
}
