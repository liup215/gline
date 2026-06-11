package subagent

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/prompts"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
)

// AllowedTools defines the tools available inside a subagent run.
var AllowedTools = []string{
	"read_file",
	"list_files",
	"search_files",
	"list_code_definition_names",
	"execute_command",
	"use_skill",
	"write_to_file",
	"attempt_completion",
}

// Builder constructs the running environment for a single subagent.
type Builder struct {
	Provider     agent.Provider
	FullRegistry *tools.Registry
	WorkingDir   string
	CustomRules  string
	Skills       []types.SkillMeta
}

// NewBuilder creates a new Builder with the given dependencies.
func NewBuilder(provider agent.Provider, fullRegistry *tools.Registry, workingDir, customRules string, skills []types.SkillMeta) *Builder {
	return &Builder{
		Provider:     provider,
		FullRegistry: fullRegistry,
		WorkingDir:   workingDir,
		CustomRules:  customRules,
		Skills:       skills,
	}
}

// BuildRestrictedRegistry creates a tool registry containing only the tools allowed in a subagent.
func (b *Builder) BuildRestrictedRegistry() *tools.Registry {
	restricted := tools.NewRegistry()
	for _, name := range AllowedTools {
		tool, err := b.FullRegistry.Get(name)
		if err != nil {
			log.Warnf("SubagentBuilder: tool %q not found in full registry, skipping", name)
			continue
		}
		// For subagent runs, all tools are auto-approved.
		info := &tools.ToolInfo{
			Tool:                 tool,
			Category:             tools.CategorySearch,
			AllowedModes:         []string{"*"},
			RequiresConfirmation: false,
		}
		if err := restricted.Register(info); err != nil {
			log.Warnf("SubagentBuilder: failed to register tool %q: %v", name, err)
		}
	}
	return restricted
}

// BuildSystemPrompt assembles the system prompt for a subagent run.
func (b *Builder) BuildSystemPrompt(mode string) string {
	restrictedRegistry := b.BuildRestrictedRegistry()
	toolDescs := make([]prompts.ToolDescription, 0)
	for _, t := range restrictedRegistry.GetAll() {
		toolDescs = append(toolDescs, prompts.ToolDescription{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: string(t.InputSchema()),
		})
	}

	basePrompt := prompts.GetSystemPrompt(mode, toolDescs, b.CustomRules, b.Skills)
	return basePrompt + SubagentSystemSuffix
}

// ConvertTools converts the restricted registry to agent tool definitions.
func (b *Builder) ConvertTools() []agent.ToolDefinition {
	restricted := b.BuildRestrictedRegistry()
	all := restricted.GetAll()
	defs := make([]agent.ToolDefinition, len(all))
	for i, t := range all {
		defs[i] = agent.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		}
	}
	return defs
}

// BuildEnvironmentBlock returns workspace metadata for the initial user message.
func (b *Builder) BuildEnvironmentBlock() string {
	cwd := b.WorkingDir
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			cwd = "."
		}
	}

	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "PowerShell"
	}

	homeDir, _ := os.UserHomeDir()

	workspacesJSON, _ := json.MarshalIndent(map[string]interface{}{
		"workspaces": map[string]interface{}{
			cwd: map[string]string{
				"hint": cwd,
			},
		},
	}, "", "  ")

	return fmt.Sprintf(`<environment_details>
# Workspace Configuration
%s

Operating System: %s
Default Shell: %s
Home Directory: %s
Current Working Directory: %s
</environment_details>`, string(workspacesJSON), runtime.GOOS, shell, homeDir, cwd)
}

// RegisterTool registers the use_subagents tool in the given registry.
func RegisterTool(registry *tools.Registry, provider agent.Provider, fullRegistry *tools.Registry, workingDir, customRules string, skills []types.SkillMeta) {
	builder := NewBuilder(provider, fullRegistry, workingDir, customRules, skills)
	_ = registry.Register(&tools.ToolInfo{
		Tool:                 NewUseSubagentsTool(builder),
		Category:             tools.CategoryInteraction,
		AllowedModes:         []string{"plan", "act"},
		RequiresConfirmation: false,
	})
}
