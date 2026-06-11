package types

// ToolName defines all available tool names as constants
type ToolName string

const (
	ToolReadFile             ToolName = "read_file"
	ToolWriteToFile          ToolName = "write_to_file"
	ToolReplaceInFile        ToolName = "replace_in_file"
	ToolExecuteCommand       ToolName = "execute_command"
	ToolSearchFiles          ToolName = "search_files"
	ToolAttemptCompletion    ToolName = "attempt_completion"
	ToolAskFollowupQuestion  ToolName = "ask_followup_question"
	ToolPlanModeRespond      ToolName = "plan_mode_respond"
	ToolUseMcpTool           ToolName = "use_mcp_tool"
	ToolAccessMcpResource    ToolName = "access_mcp_resource"
	ToolUseSkill             ToolName = "use_skill"
	ToolUseSubagents         ToolName = "use_subagents"
)

func (t ToolName) String() string {
	return string(t)
}

// IsSpecialTool returns true if the tool requires special handling
func (t ToolName) IsSpecialTool() bool {
	switch t {
	case ToolAttemptCompletion, ToolAskFollowupQuestion, ToolPlanModeRespond:
		return true
	default:
		return false
	}
}
