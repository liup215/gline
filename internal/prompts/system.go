// Package prompts manages system prompts and tool definitions for gline.
package prompts

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/liup215/gline/internal/shell"
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

# Tool Usage Rules
- read: {"path": "...", "line_number": N, "limit": N} — read a file (~, relative and absolute paths; default 1000 lines, max 2000 per read)
- write: {"path": "...", "content": "..."} — create/overwrite a file
- edit: {"path": "...", "search": "...", "replace": "..."} — edit a file
- grep: {"path": "...", "regex": "...", "file_pattern": "*.go"} — search file CONTENTS by regex
- glob: {"path": "...", "pattern": "*.go"} — find files by name glob
- run: {"command": "..."} — execute a command
- attempt_completion: {"result": "..."} — ONLY has a 'result' field. Never add other fields like path or command.

# Code Search Strategy
- glob locates files by name; grep greps content by regex. Both are fast (ripgrep/fd-backed) and respect .gitignore, skipping hidden/vendor dirs.
- Prefer these over shelling out to grep/rg/find via run — they return structured, line-numbered results you can feed directly into read.
- Typical exploration flow: glob → read the relevant files → edit.
- grep caps at 500 matches; narrow with file_pattern or a more specific regex if flooded.

End your work with attempt_completion when the task is complete.

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

	shellLabel := shell.Resolve().Name()

	homeDir, _ := os.UserHomeDir()

	return fmt.Sprintf(`# System

OS: %s (%s)
Shell: %s
Home: %s
Working Directory: %s
Go: %s`, osName, osArch, shellLabel, homeDir, cwd, goVersion)
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

// Note: GetToolDescriptions / GetPlanModeToolDescriptions / GetActModeToolDescriptions
// (hardcoded 15-tool description tables from the pre-ADK loop) were removed on
// 2026-09-21 — the ADK agent derives its tool surface from the live registry,
// and the system instruction is assembled by adkagent.SystemInstruction.
// buildToolSection above remains for subagent.GetSystemPrompt.
