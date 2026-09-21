package subagent

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/prompts"
	"github.com/liup215/gline/internal/shell"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

// AllowedTools defines the tools available inside a subagent run.
var AllowedTools = []string{
	"read",
	"ls",
	"grep",
	"glob",
	"list_code_definition_names",
	"run",
	"use_skill",
	"write",
	"attempt_completion",
}

// Builder constructs the running environment for a single subagent.
type Builder struct {
	// LLM is the model backend used for every assistant turn (internal/provider).
	LLM          model.LLM
	FullRegistry *tools.Registry
	WorkingDir   string
	CustomRules  string
	Skills       []types.SkillMeta
}

// NewBuilder creates a new Builder with the given dependencies.
func NewBuilder(llm model.LLM, fullRegistry *tools.Registry, workingDir, customRules string, skills []types.SkillMeta) *Builder {
	return &Builder{
		LLM:          llm,
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

// buildDeclarations converts the restricted registry into genai function
// declarations for the LLM request. The tool JSON schema arrives as
// json.RawMessage and is decoded into plain any, matching the shape
// genai.FunctionDeclaration.ParametersJsonSchema serializes verbatim.
func (b *Builder) buildDeclarations() ([]*genai.FunctionDeclaration, error) {
	restricted := b.BuildRestrictedRegistry()
	all := restricted.GetAll()
	decls := make([]*genai.FunctionDeclaration, 0, len(all))
	for _, t := range all {
		decl := &genai.FunctionDeclaration{
			Name:        t.Name(),
			Description: t.Description(),
		}
		if raw := t.InputSchema(); len(raw) > 0 {
			var schema any
			if err := json.Unmarshal(raw, &schema); err != nil {
				return nil, fmt.Errorf("tool %q: decoding schema: %w", t.Name(), err)
			}
			decl.ParametersJsonSchema = schema
		}
		decls = append(decls, decl)
	}
	return decls, nil
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

	shellLabel := shell.Resolve().Name()

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
</environment_details>`, string(workspacesJSON), runtime.GOOS, shellLabel, homeDir, cwd)
}

// RegisterTool registers the use_subagents tool in the given registry.
func RegisterTool(registry *tools.Registry, llm model.LLM, fullRegistry *tools.Registry, workingDir, customRules string, skills []types.SkillMeta) {
	builder := NewBuilder(llm, fullRegistry, workingDir, customRules, skills)
	_ = registry.Register(&tools.ToolInfo{
		Tool:                 NewUseSubagentsTool(builder),
		Category:             tools.CategoryInteraction,
		AllowedModes:         []string{"plan", "act"},
		RequiresConfirmation: false,
	})
}
