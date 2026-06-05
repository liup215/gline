package skills

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/liup215/gline/pkg/types"
)

// BuildSlashCommands generates slash commands for every registered skill.
// onActivate is called when the user types /skill-name.
// onDeactivate is called when the user types /skill-off.
func BuildSlashCommands(
	registry *Registry,
	onActivate func(skill *types.Skill),
	onDeactivate func(),
) []*types.SlashCommand {
	var commands []*types.SlashCommand

	// /skill — list all available skills
	commands = append(commands, &types.SlashCommand{
		Name:        "skill",
		Description: "List all available skills and show which one is active",
		Section:     types.SectionDefault,
		Handler: func(args string) (bool, error) {
			 infos := registry.GetAllInfo()
			if len(infos) == 0 {
				fmt.Println("No skills loaded.")
				fmt.Printf("Skill directories:\n")
				for _, dir := range DefaultSkillDirs {
					fmt.Printf("  %s\n", dir)
				}
				return true, nil
			}
			fmt.Println("Available skills:")
			for _, info := range infos {
				marker := " "
				if info.Active {
					marker = "*"
				}
				fmt.Printf("  [%s] /%-15s %s\n", marker, info.Name, info.Description)
			}
			fmt.Println()
			fmt.Println("Use /skill-name to activate, /skill-off to deactivate.")
			return true, nil
		},
	})

	// /skill-off — deactivate the currently active skill
	commands = append(commands, &types.SlashCommand{
		Name:        "skill-off",
		Description: "Deactivate the currently active skill",
		Section:     types.SectionDefault,
		Handler: func(args string) (bool, error) {
			if _, ok := registry.GetActive(); !ok {
				fmt.Println("No skill is currently active.")
				return true, nil
			}
			onDeactivate()
			registry.Deactivate()
			fmt.Println("Skill deactivated.")
			return true, nil
		},
	})

	// dynamic commands: /explain, /code-review, etc.
	for _, skill := range registry.GetAll() {
		s := skill // capture
		commands = append(commands, &types.SlashCommand{
			Name:        s.Name,
			Description: s.Description,
			Section:     types.SectionCustom,
			Handler: func(args string) (bool, error) {
				if _, err := registry.Activate(s.Name); err != nil {
					return false, err
				}
				onActivate(s)
				fmt.Printf("Skill activated: %s\n", s.Name)
				return true, nil
			},
		})
	}

	return commands
}

// InitBuiltinSkills copies embedded built-in skills to the user-level
// ~/.gline/skills directory if they do not already exist.
// It returns the number of files written.
func InitBuiltinSkills() (int, error) {
	userDir := filepath.Join(mustUserHomeDir(), ".gline", "skills")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return 0, fmt.Errorf("create skills directory: %w", err)
	}

	files := []string{
		"explain.yaml",
		"code-review.yaml",
		"refactor.yaml",
		"debug.yaml",
		"doc.yaml",
	}

	written := 0
	for _, name := range files {
		dest := filepath.Join(userDir, name)
		if _, err := os.Stat(dest); err == nil {
			continue // already exists; do not overwrite
		}
		if err := writeBuiltinSkill(name, dest); err != nil {
			return written, fmt.Errorf("write %s: %w", name, err)
		}
		written++
	}
	return written, nil
}
