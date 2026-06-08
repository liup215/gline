package skills

import (
	"fmt"

	"github.com/liup215/gline/pkg/types"
)

// BuildSlashCommands generates slash commands for skill discovery.
// /skill lists all available skills; /skill-name prints a hint telling
// the user to use the use_skill tool (since skills are now loaded on-demand).
func BuildSlashCommands(registry *Registry) []*types.SlashCommand {
	var commands []*types.SlashCommand

	// /skill — list all available skills
	commands = append(commands, &types.SlashCommand{
		Name:        "skill",
		Description: "List all available skills",
		Section:     types.SectionDefault,
		Handler: func(args string) (bool, error) {
			metas := registry.GetMeta()
			if len(metas) == 0 {
				fmt.Println("No skills loaded.")
				fmt.Printf("Skill directories:\n")
				for _, dir := range DefaultSkillDirs {
					fmt.Printf("  %s\n", dir)
				}
				return true, nil
			}
			fmt.Println("Available skills:")
			for _, meta := range metas {
				fmt.Printf("  /%-15s %s\n", meta.Name, meta.Description)
			}
			fmt.Println()
			fmt.Println("Skills are loaded on-demand via the use_skill tool.")
			return true, nil
		},
	})

	// dynamic commands: /explain, /code-review, etc.
	// These just print info since activation is now done via use_skill tool.
	for _, skill := range registry.GetAll() {
		s := skill // capture
		commands = append(commands, &types.SlashCommand{
			Name:        s.Name,
			Description: s.Description,
			Section:     types.SectionCustom,
			Handler: func(args string) (bool, error) {
				fmt.Printf("Skill: %s\n", s.Name)
				fmt.Printf("Description: %s\n", s.Description)
				fmt.Println()
				fmt.Println("To activate this skill, use the use_skill tool with skill_name:", s.Name)
				return true, nil
			},
		})
	}

	return commands
}
