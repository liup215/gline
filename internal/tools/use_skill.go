package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liup215/gline/pkg/types"
)

// UseSkillTool loads and activates a skill by name.
// This follows the cline agent skill specification: the system prompt lists
// available skills with descriptions, and the LLM calls use_skill to load
// the full instructions on-demand.
type UseSkillTool struct {
	BaseTool
	registry SkillRegistry
}

// SkillRegistry is the interface the use_skill tool needs from the skill registry.
type SkillRegistry interface {
	Get(name string) (*types.Skill, bool)
	GetMeta() []types.SkillMeta
}

// UseSkillInput represents the input for the use_skill tool.
type UseSkillInput struct {
	SkillName string `json:"skill_name"`
}

// NewUseSkillTool creates a new use_skill tool.
func NewUseSkillTool(registry SkillRegistry) *UseSkillTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"skill_name": {
				"type": "string",
				"description": "The name of the skill to activate (must match exactly one of the available skill names)"
			}
		},
		"required": ["skill_name"]
	}`)
	return &UseSkillTool{
		BaseTool: BaseTool{
			name:        "use_skill",
			description: "Load and activate a skill by name. Skills provide specialized instructions for specific tasks. Use this tool ONCE when a user's request matches one of the available skill descriptions shown in the SKILLS section of your system prompt. After activation, follow the skill's instructions directly - do not call use_skill again.",
			inputSchema: schema,
		},
		registry: registry,
	}
}

// Execute loads the skill instructions and returns them as a tool result.
func (t *UseSkillTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req UseSkillInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}

	if strings.TrimSpace(req.SkillName) == "" {
		return "", fmt.Errorf("skill_name is required")
	}

	skill, ok := t.registry.Get(req.SkillName)
	if !ok {
		// Build helpful error with available skill names.
		metas := t.registry.GetMeta()
		if len(metas) == 0 {
			return "", fmt.Errorf(`skill %q not found. No skills are available.`, req.SkillName)
		}
		var names []string
		for _, m := range metas {
			names = append(names, fmt.Sprintf("%q", m.Name))
		}
		return "", fmt.Errorf(`skill %q not found. Available skills: %s`, req.SkillName, strings.Join(names, ", "))
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Skill \"%s\" is now active\n\n", skill.Name))
	b.WriteString(skill.Contents)
	b.WriteString("\n\n---\n")
	b.WriteString("IMPORTANT: The skill is now loaded. Do NOT call use_skill again for this task. Simply follow the instructions above to complete the user's request.")
	if skill.Source != "" {
		b.WriteString(fmt.Sprintf(" You may access other files in the skill directory at: %s", strings.TrimSuffix(skill.Source, "SKILL.md")))
	}
	b.WriteString("\n")

	return b.String(), nil
}
