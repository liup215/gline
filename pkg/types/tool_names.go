package types

// ToolName defines all available tool names as constants
type ToolName string

const (
	ToolReadFile            ToolName = "read"
	ToolWriteToFile         ToolName = "write"
	ToolReplaceInFile       ToolName = "edit"
	ToolExecuteCommand      ToolName = "run"
	ToolGrep                ToolName = "grep"
	ToolGlob                ToolName = "glob"
	ToolLs                  ToolName = "ls"
	ToolSummarizeFile       ToolName = "summarize_file"
	ToolAttemptCompletion   ToolName = "attempt_completion"
	ToolAskFollowupQuestion ToolName = "ask_followup_question"
	ToolPlanModeRespond     ToolName = "plan_mode_respond"
	ToolUseMcpTool          ToolName = "use_mcp_tool"
	ToolAccessMcpResource   ToolName = "access_mcp_resource"
	ToolUseSkill            ToolName = "use_skill"
	ToolUseSubagents        ToolName = "use_subagents"
)

func (t ToolName) String() string {
	return string(t)
}

// IsSpecialTool returns true if the tool requires special handling// (IsSpecialTool removed 2026-09-21: no callers; special-casing lives in
// the
// adkagent.legacyToolNames and the TUI renderer registry.)
