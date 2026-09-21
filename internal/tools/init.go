package tools

import (
	"github.com/liup215/gline/internal/memory"
	"github.com/liup215/gline/internal/summarizer"
)

// SummarizeFileToolInput represents the input for summarize_file.
type SummarizeFileToolInput struct {
	Path string `json:"path"`
}

// InitDefaultRegistry initializes the default registry with all built-in tools.
// If a memory engine is provided, it also registers kb_search, kb_ingest,
// memory_recall and memory_note tools so the agent can use the knowledge base.
// If a summarizer is provided, it registers summarize_file.
func InitDefaultRegistry(engine *memory.UnifiedEngine, sum *summarizer.Summarizer) *Registry {
	registry := NewRegistry()

	// File operations - allowed in both modes (read-only in plan)
	registry.Register(&ToolInfo{
		Tool:                 NewReadFileTool(),
		Category:             CategoryFile,
		AllowedModes:         []string{"plan", "act"},
		RequiresConfirmation: false,
	})

	registry.Register(&ToolInfo{
		Tool:                 NewLsTool(),
		Category:             CategoryFile,
		AllowedModes:         []string{"plan", "act"},
		RequiresConfirmation: false,
	})

	// Note: summarize_file registration disabled (2026-09-21, tool pruning);
	// the `sum` parameter is kept for API compatibility. Re-enable with
	// RegisterSummarizeFileTool(registry, sum) if needed.

	// File write operations - act mode only; run without approval prompts
	// (user preference: full auto-approve).
	registry.Register(&ToolInfo{
		Tool:                 NewWriteFileTool(),
		Category:             CategoryFile,
		AllowedModes:         []string{"act"},
		RequiresConfirmation: false,
	})

	registry.Register(&ToolInfo{
		Tool:                 NewReplaceInFileTool(),
		Category:             CategoryFile,
		AllowedModes:         []string{"act"},
		RequiresConfirmation: false,
	})

	// Search operations - allowed in both modes
	registry.Register(&ToolInfo{
		Tool:                 NewGrepTool(),
		Category:             CategorySearch,
		AllowedModes:         []string{"plan", "act"},
		RequiresConfirmation: false,
	})

	registry.Register(&ToolInfo{
		Tool:                 NewGlobTool(),
		Category:             CategorySearch,
		AllowedModes:         []string{"plan", "act"},
		RequiresConfirmation: false,
	})

	// Note: list_code_definition_names disabled (2026-09-21, tool pruning).

	// Command execution - act mode only; runs directly without an approval
	// prompt (user preference: no confirmation for running commands).
	registry.Register(&ToolInfo{
		Tool:                 NewExecuteCommandTool(),
		Category:             CategoryCommand,
		AllowedModes:         []string{"act"},
		RequiresConfirmation: false,
	})

	// User interaction - allowed in both modes
	// ask_followup_question: skip both start & complete system messages;
	// the askQuestionMsg handler displays the question with styled options instead.
	registry.Register(&ToolInfo{
		Tool:                 NewAskFollowupQuestionTool(),
		Category:             CategoryInteraction,
		AllowedModes:         []string{"plan", "act"},
		RequiresConfirmation: false,
		Behavior: ToolBehavior{
			StartDisplayMode:    DisplaySkip,
			CompleteDisplayMode: DisplaySkip,
		},
	})

	// Plan mode response - plan mode only
	// plan_mode_respond: skip start; render the completed result as a full assistant message (markdown)
	registry.Register(&ToolInfo{
		Tool:                 NewPlanModeRespondTool(),
		Category:             CategoryInteraction,
		AllowedModes:         []string{"plan"},
		RequiresConfirmation: false,
		Behavior: ToolBehavior{
			StartDisplayMode:    DisplaySkip,
			CompleteDisplayMode: DisplayAssistant,
		},
	})

	// Completion - allowed in both modes
	// attempt_completion: show result as assistant message (markdown); skip duplicate complete message
	registry.Register(&ToolInfo{
		Tool:                 NewAttemptCompletionTool(),
		Category:             CategoryCompletion,
		AllowedModes:         []string{"plan", "act"},
		RequiresConfirmation: false,
		Behavior: ToolBehavior{
			StartDisplayMode:    DisplayAssistant,
			CompleteDisplayMode: DisplaySkip,
		},
	})

	// Note: web_fetch / browser_copy disabled (2026-09-21, tool pruning);
	// re-enable via RegisterSummarizeFileTool-style blocks here if needed.

	// Memory / knowledge base tools - optional, only if engine is available
	if engine != nil {
		e := engine
		registry.Register(&ToolInfo{
			Tool:                 NewKBSearchTool(e),
			Category:             CategorySearch,
			AllowedModes:         []string{"plan", "act"},
			RequiresConfirmation: false,
		})
		registry.Register(&ToolInfo{
			Tool:                 NewKBIngestTool(e),
			Category:             CategorySearch,
			AllowedModes:         []string{"act"},
			RequiresConfirmation: false,
		})
		registry.Register(&ToolInfo{
			Tool:                 NewMemoryRecallTool(e),
			Category:             CategorySearch,
			AllowedModes:         []string{"plan", "act"},
			RequiresConfirmation: false,
		})
		registry.Register(&ToolInfo{
			Tool:                 NewMemoryNoteTool(e),
			Category:             CategorySearch,
			AllowedModes:         []string{"act"},
			RequiresConfirmation: false,
		})
	}

	return registry
}

// RegisterSkillTool registers the use_skill tool with a skill registry instance.
// This must be called after InitDefaultRegistry and after the skill registry is
// created so the tool can look up skills on-demand.
func RegisterSkillTool(registry *Registry, skillRegistry SkillRegistry) {
	registry.Register(&ToolInfo{
		Tool:                 NewUseSkillTool(skillRegistry),
		Category:             CategoryInteraction,
		AllowedModes:         []string{"plan", "act"},
		RequiresConfirmation: false,
	})
}

// (GetDefaultTools / GetToolsForMode / IsToolAllowed — the pre-ADK loop's
// hardcoded dispatch helpers — were removed on 2026-09-21. Tool surface and
// mode gating now live entirely in the Registry + adkagent's buildTools.)
