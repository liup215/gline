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
	active string // name of the currently active skill (empty = none)
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
	if r.active == name {
		r.active = ""
	}
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

// GetAllInfo returns lightweight SkillInfo summaries for UI lists.
func (r *Registry) GetAllInfo() []types.SkillInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]types.SkillInfo, 0, len(r.skills))
	for _, s := range r.skills {
		result = append(result, types.SkillInfo{
			Name:        s.Name,
			Description: s.Description,
			Active:      r.active == s.Name,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Count returns the number of registered skills.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

// LoadFromDirs walks the provided directories in order and loads every
// valid skill file found.  Later directories override earlier ones when
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

// Activate sets the named skill as the currently active one.
// Returns an error if the skill is not registered.
func (r *Registry) Activate(name string) (*types.Skill, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	name = strings.ToLower(strings.TrimSpace(name))
	s, ok := r.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill %q not found", name)
	}
	r.active = name
	return s, nil
}

// Deactivate clears the currently active skill.
func (r *Registry) Deactivate() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active = ""
}

// GetActive returns the currently active skill, or (nil, false) if none.
func (r *Registry) GetActive() (*types.Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active == "" {
		return nil, false
	}
	s, ok := r.skills[r.active]
	return s, ok
}

// IsActive reports whether name is the currently active skill.
func (r *Registry) IsActive(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.active == strings.ToLower(strings.TrimSpace(name))
}
