// Package prompts manages system prompts and tool definitions for gline.
package prompts

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/liup215/gline/pkg/types"
)

// GetSystemPrompt returns the appropriate system prompt for the given mode.
func GetSystemPrompt(mode string, tools []ToolDescription, customRules string, skills []types.SkillMeta) string {
	var prompt strings.Builder

	// Core identity - minimal
	prompt.WriteString(`You are gline, an AI coding assistant. You help users with software engineering tasks.

`)

	// System info
	prompt.WriteString(getSystemInfoSection())

	// Tool use - explicit
	prompt.WriteString(`

# Tools

You have access to tools for file operations, code search, command execution, and more.
You MUST use tools to act. Do not just describe what you would do.
When you need to explore or modify the project, invoke the appropriate tool using the native tool_call format.
After each tool use, wait for the result before proceeding.
End your work with attempt_completion.

`)

	// Mode-specific behavior
	if mode == "plan" {
		prompt.WriteString(`# Mode: Plan

You are in Plan Mode. Explore the codebase, understand the task, and present a plan.
Do not make changes. Use plan_mode_respond to present your plan and ask for feedback.

`)
	} else {
		prompt.WriteString(`# Mode: Act

You are in Act Mode. Execute tasks by reading, writing, and modifying files.
Use tools to complete the work. End with attempt_completion when done.

`)
	}

	// Append custom rules if provided
	if customRules != "" {
		prompt.WriteString("\n# Custom Rules\n\n")
		prompt.WriteString(customRules)
	}

	// Append SKILLS section if skills are available
	if len(skills) > 0 {
		prompt.WriteString("\n\n# Skills\n\n")
		prompt.WriteString(buildSkillsSection(skills))
	}

	// Append tool descriptions
	toolSection := buildToolSection(tools)
	if toolSection != "" {
		prompt.WriteString("\n\n")
		prompt.WriteString(toolSection)
	}

	return prompt.String()
}

// buildSkillsSection generates the skills section for the system prompt.
func buildSkillsSection(skills []types.SkillMeta) string {
	var b strings.Builder
	b.WriteString("Available skills:\n")
	for _, s := range skills {
		b.WriteString(fmt.Sprintf("- %s: %s\n", s.Name, s.Description))
	}
	b.WriteString("\nUse use_skill tool to load a skill when needed.")
	return b.String()
}

// getSystemInfoSection returns system information.
func getSystemInfoSection() string {
	goVersion := runtime.Version()
	osName := runtime.GOOS
	osArch := runtime.GOARCH
	cwd := getCWD()

	shell := "bash"
	if osName == "windows" {
		shell = "PowerShell"
	}

	homeDir, _ := os.UserHomeDir()

	return fmt.Sprintf(`# System

OS: %s (%s)
Shell: %s
Home: %s
Working Directory: %s
Go: %s`, osName, osArch, shell, homeDir, cwd, goVersion)
}

// getCWD returns the current working directory.
func getCWD() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

// ToolDescription describes a tool for the system prompt
type ToolDescription struct {
	Name        string
	Description string
	InputSchema string
}

// buildToolSection builds the tool descriptions section.
func buildToolSection(tools []ToolDescription) string {
	if len(tools) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("# Available Tools\n\n")

	for _, tool := range tools {
		builder.WriteString(fmt.Sprintf("## %s\n%s\n", tool.Name, tool.Description))
		if len(tool.InputSchema) > 0 && tool.InputSchema != "{}" {
			builder.WriteString(fmt.Sprintf("Input: %s\n\n", tool.InputSchema))
		}
	}

	return builder.String()
}

// GetToolDescriptions returns descriptions for all built-in tools
func GetToolDescriptions() []ToolDescription {
	return []ToolDescription{
		{
			Name:        "read_file",
			Description: "Read up to 50 lines of a file. Omit line_number to read lines 1-50; provide line_number to read 50 lines starting from that line.",
			InputSchema: `{"type":"object","properties":{"path":{"type":"string"},"line_number":{"type":"integer"}},"required":["path"]}`,
		},
		{
			Name:        "write_to_file",
			Description: "Create or overwrite a file.",
			InputSchema: `{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`,
		},
		{
			Name:        "replace_in_file",
			Description: "Replace specific content in a file using exact search/replace.",
			InputSchema: `{"type":"object","properties":{"path":{"type":"string"},"search":{"type":"string"},"replace":{"type":"string"}},"required":["path","search","replace"]}`,
		},
		{
			Name:        "list_files",
			Description: "List files and directories.",
			InputSchema: `{"type":"object","properties":{"path":{"type":"string"},"recursive":{"type":"boolean"}},"required":["path"]}`,
		},
		{
			Name:        "search_files",
			Description: "Search for regex pattern in files.",
			InputSchema: `{"type":"object","properties":{"path":{"type":"string"},"regex":{"type":"string"}},"required":["path","regex"]}`,
		},
		{
			Name:        "list_code_definition_names",
			Description: "List code definitions (functions, classes, etc.).",
			InputSchema: `{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`,
		},
		{
			Name:        "execute_command",
			Description: "Execute a CLI command.",
			InputSchema: `{"type":"object","properties":{"command":{"type":"string"}},"required":["command"]}`,
		},
		{
			Name:        "ask_followup_question",
			Description: "Ask the user a question for clarification.",
			InputSchema: `{"type":"object","properties":{"question":{"type":"string"},"options":{"type":"array","items":{"type":"string"}}},"required":["question"]}`,
		},
		{
			Name:        "attempt_completion",
			Description: "Signal task completion with a summary.",
			InputSchema: `{"type":"object","properties":{"result":{"type":"string"}},"required":["result"]}`,
		},
		{
			Name:        "plan_mode_respond",
			Description: "Present a plan or ask questions in Plan mode.",
			InputSchema: `{"type":"object","properties":{"response":{"type":"string"}},"required":["response"]}`,
		},
		{
			Name:        "use_skill",
			Description: "Load and activate a skill for specialized instructions.",
			InputSchema: `{"type":"object","properties":{"skill_name":{"type":"string"}},"required":["skill_name"]}`,
		},
		{
			Name:        "summarize_file",
			Description: "Summarize a large file using AI.",
			InputSchema: `{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`,
		},
		{
			Name:        "web_fetch",
			Description: "Fetch web page content as Markdown.",
			InputSchema: `{"type":"object","properties":{"url":{"type":"string"}},"required":["url"]}`,
		},
		{
			Name:        "use_subagents",
			Description: "Run focused subagents for parallel exploration.",
			InputSchema: `{"type":"object","properties":{"prompt_1":{"type":"string"}},"required":["prompt_1"]}`,
		},
	}
}

// GetPlanModeToolDescriptions returns tool descriptions for plan mode
func GetPlanModeToolDescriptions() []ToolDescription {
	allTools := GetToolDescriptions()
	var planTools []ToolDescription

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
