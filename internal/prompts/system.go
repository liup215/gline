// Package prompts manages system prompts and tool definitions for gline
package prompts

import (
	"fmt"
	"strings"
)

// SystemPrompt contains the system prompt configuration
type SystemPrompt struct {
	// BasePrompt is the core system prompt
	BasePrompt string

	// Mode is the operating mode (plan or act)
	Mode string

	// Capabilities lists what the agent can do
	Capabilities []string

	// Rules are the guidelines for the agent
	Rules []string

	// ToolDescriptions contains descriptions of available tools
	ToolDescriptions string
}

// GetSystemPrompt returns the appropriate system prompt for the given mode
func GetSystemPrompt(mode string, tools []ToolDescription) string {
	var basePrompt string

	switch mode {
	case "plan":
		basePrompt = getPlanModePrompt()
	case "act":
		basePrompt = getActModePrompt()
	default:
		basePrompt = getActModePrompt()
	}

	// Add tool descriptions
	toolSection := buildToolSection(tools)

	return fmt.Sprintf("%s\n\n%s", basePrompt, toolSection)
}

// getPlanModePrompt returns the system prompt for plan mode
func getPlanModePrompt() string {
	return `You are gline, an AI programming assistant in PLAN MODE.

In Plan Mode, you focus on exploration, planning, and gathering information WITHOUT making any changes to files or executing commands.

YOUR CAPABILITIES:
- Read and analyze files to understand the codebase
- Search for patterns and code definitions
- List files and directories to explore structure
- Ask follow-up questions to clarify requirements
- Present plans and strategies for implementation

WHAT YOU CANNOT DO IN PLAN MODE:
- Write or modify files
- Execute commands
- Make any changes to the system

WORKFLOW:
1. Understand the user's request
2. Explore the codebase if needed (read files, search, etc.)
3. Ask clarifying questions if requirements are unclear
4. Present a detailed plan of action
5. Wait for user approval before proceeding to Act Mode

RESPONSE FORMAT:
- Be thorough in your analysis
- Present clear, actionable plans
- Use the plan_mode_respond tool to present your plan
- Do not use file modification or command execution tools`
}

// getActModePrompt returns the system prompt for act mode
func getActModePrompt() string {
	return `You are gline, an AI programming assistant in ACT MODE.

In Act Mode, you can execute tasks, modify files, and run commands to help the user accomplish their goals.

YOUR CAPABILITIES:
- Read and analyze files
- Write new files or modify existing ones
- Execute commands and scripts
- Search for patterns and code definitions
- List files and directories
- Ask follow-up questions when needed

WORKFLOW:
1. Understand the user's request
2. Plan your approach (you can think through this)
3. Execute the plan using available tools
4. Read files to verify changes
5. Use attempt_completion when finished

TOOL USAGE GUIDELINES:
- Use read_file to examine existing code
- Use write_to_file to create new files or completely rewrite existing ones
- Use replace_in_file for targeted modifications (SEARCH/REPLACE blocks)
- Use execute_command to run commands
- Use search_files to find patterns across the codebase
- Use list_files to explore directory structure
- Use ask_followup_question when you need clarification
- Use attempt_completion when the task is done

BEST PRACTICES:
- Always verify file contents before modifying
- Make minimal, focused changes
- Test your changes when possible
- Provide clear summaries of what was done`
}

// ToolDescription describes a tool for the system prompt
type ToolDescription struct {
	Name        string
	Description string
	InputSchema string
}

// buildToolSection builds the tool descriptions section
func buildToolSection(tools []ToolDescription) string {
	if len(tools) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("AVAILABLE TOOLS:\n\n")

	for _, tool := range tools {
		builder.WriteString(fmt.Sprintf("## %s\n", tool.Name))
		builder.WriteString(fmt.Sprintf("%s\n", tool.Description))
		if tool.InputSchema != "" {
			builder.WriteString(fmt.Sprintf("Input Schema:\n%s\n", tool.InputSchema))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// GetToolDescriptions returns descriptions for all built-in tools
func GetToolDescriptions() []ToolDescription {
	return []ToolDescription{
		{
			Name:        "read_file",
			Description: "Read the contents of a file at the specified path. Use this when you need to examine the contents of an existing file.",
			InputSchema: `{"path": "string (required) - The path of the file to read"}`,
		},
		{
			Name:        "write_to_file",
			Description: "Write content to a file at the specified path. If the file exists, it will be overwritten. Use this when creating new files or completely rewriting existing files.",
			InputSchema: `{"path": "string (required)", "content": "string (required)"}`,
		},
		{
			Name:        "replace_in_file",
			Description: "Replace specific content in a file using exact search/replace. Use this for targeted modifications to existing files.",
			InputSchema: `{"path": "string (required)", "search": "string (required) - Exact content to find", "replace": "string (required) - Content to replace with"}`,
		},
		{
			Name:        "list_files",
			Description: "List files and directories at the specified path. Use this to explore the file system.",
			InputSchema: `{"path": "string (required)", "recursive": "boolean (optional) - List recursively"}`,
		},
		{
			Name:        "search_files",
			Description: "Search for a regex pattern in files within a directory. Returns context-rich results with file paths, line numbers, and surrounding context.",
			InputSchema: `{"path": "string (required) - Directory to search", "regex": "string (required) - Pattern to search for", "file_pattern": "string (optional) - Glob pattern to filter files"}`,
		},
		{
			Name:        "list_code_definition_names",
			Description: "List definition names (functions, classes, methods, etc.) in code files within a directory. Provides insights into codebase structure.",
			InputSchema: `{"path": "string (required) - Directory to analyze", "file_pattern": "string (optional) - Glob pattern to filter files"}`,
		},
		{
			Name:        "execute_command",
			Description: "Execute a CLI command on the system. Use this when you need to perform system operations or run specific commands.",
			InputSchema: `{"command": "string (required) - The command to execute", "requires_approval": "boolean (optional) - Whether this command requires user approval", "cwd": "string (optional) - Working directory", "timeout": "integer (optional) - Timeout in seconds"}`,
		},
		{
			Name:        "ask_followup_question",
			Description: "Ask the user a question to gather clarifying information or make a decision. Use this when you need more information to complete a task.",
			InputSchema: `{"question": "string (required) - The question to ask", "options": "array of strings (optional) - Options for the user to choose from"}`,
		},
		{
			Name:        "attempt_completion",
			Description: "Indicate that you have completed the task and provide a summary of what was accomplished. Use this when you have finished all required work.",
			InputSchema: `{"result": "string (required) - Summary of what was accomplished", "command": "string (optional) - Command to showcase the result"}`,
		},
	}
}

// GetPlanModeToolDescriptions returns tool descriptions for plan mode
func GetPlanModeToolDescriptions() []ToolDescription {
	allTools := GetToolDescriptions()
	var planTools []ToolDescription

	// Filter out tools not allowed in plan mode
	actOnlyTools := map[string]bool{
		"write_to_file":   true,
		"replace_in_file": true,
		"execute_command": true,
	}

	for _, tool := range allTools {
		if !actOnlyTools[tool.Name] {
			planTools = append(planTools, tool)
		}
	}

	return planTools
}

// GetActModeToolDescriptions returns tool descriptions for act mode
func GetActModeToolDescriptions() []ToolDescription {
	return GetToolDescriptions()
}

// FormatToolUseExample returns an example of how to use a tool
func FormatToolUseExample(toolName string, input map[string]interface{}) string {
	return fmt.Sprintf("Using tool: %s with input: %v", toolName, input)
}
