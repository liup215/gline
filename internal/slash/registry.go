package slash

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/liup215/gline/pkg/types"
)

// Registry manages all slash commands.
// It is safe for concurrent use.
type Registry struct {
	mu       sync.RWMutex
	commands map[string]*types.SlashCommand
}

// NewRegistry creates an empty slash command registry.
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]*types.SlashCommand),
	}
}

// Register adds a command to the registry.
// Returns an error if a command with the same name already exists.
func (r *Registry) Register(cmd *types.SlashCommand) error {
	if cmd == nil {
		return fmt.Errorf("cannot register nil command")
	}
	name := strings.ToLower(cmd.Name)
	if name == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.commands[name]; exists {
		return fmt.Errorf("command /%s already registered", name)
	}
	r.commands[name] = cmd
	return nil
}

// Get retrieves a command by name (case-insensitive).
func (r *Registry) Get(name string) (*types.SlashCommand, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmd, ok := r.commands[strings.ToLower(name)]
	return cmd, ok
}

// GetAll returns all registered commands sorted by section then name.
func (r *Registry) GetAll() []*types.SlashCommand {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*types.SlashCommand, 0, len(r.commands))
	for _, cmd := range r.commands {
		result = append(result, cmd)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Section != result[j].Section {
			return result[i].Section == types.SectionCustom && result[j].Section != types.SectionCustom
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result
}

// Filter returns commands whose names contain the given prefix (case-insensitive).
func (r *Registry) Filter(prefix string) []*types.SlashCommand {
	prefix = strings.ToLower(prefix)

	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*types.SlashCommand
	for _, cmd := range r.commands {
		if strings.HasPrefix(strings.ToLower(cmd.Name), prefix) {
			result = append(result, cmd)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Section != result[j].Section {
			return result[i].Section == types.SectionCustom && result[j].Section != types.SectionCustom
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result
}

// Count returns the number of registered commands.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.commands)
}
