package view

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatToolStartDisplay creates a compact human-friendly display string for a tool start event.
// It handles special tools (attempt_completion, ask_followup_question, plan_mode_respond)
// differently from regular tools.
func FormatToolStartDisplay(name, input string) string {
	desc := GetToolDescription(name)
	if input == "" {
		return fmt.Sprintf("🔧 %s", desc)
	}

	// attempt_completion often carries the final summary/result; keep it intact
	if NormalizeToolName(name) == "attempt_completion" {
		return fmt.Sprintf("🔧 %s\n\n%s", desc, input)
	}

	// Try to show the single most relevant argument (path, command, url, etc.)
	if main := GetToolMainArg(name, input); main != "" {
		return fmt.Sprintf("🔧 %s: %s", desc, main)
	}

	// Fallback: show compact input (truncate if too long)
	input = strings.TrimSpace(input)
	if len(input) > 100 {
		return fmt.Sprintf("🔧 %s: %s...", desc, input[:97])
	}
	return fmt.Sprintf("🔧 %s: %s", desc, input)
}

// FormatAttemptCompletionContent extracts a human-friendly result string from
// an attempt_completion tool's JSON input. Returns the result/content field,
// or a JSON code block if the result is an object, or the raw input as fallback.
func FormatAttemptCompletionContent(input string) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(input), &parsed); err != nil {
		return input
	}

	// Prefer result as non-empty string
	if r, ok := parsed["result"].(string); ok && strings.TrimSpace(r) != "" {
		return r
	}
	if c, ok := parsed["content"].(string); ok && strings.TrimSpace(c) != "" {
		return c
	}
	if mres, ok := parsed["result"].(map[string]interface{}); ok {
		// If result is an object, pretty-print it and render as a JSON code block
		if pretty, err2 := json.MarshalIndent(mres, "", "  "); err2 == nil {
			return "```json\n" + string(pretty) + "\n```"
		}
		return input
	}

	// Fallback: pretty-print the whole parsed JSON as a JSON code block
	if pretty, err2 := json.MarshalIndent(parsed, "", "  "); err2 == nil {
		return "```json\n" + string(pretty) + "\n```"
	}
	return input
}

// FormatToolCompleteDisplay creates a summary display string for a tool completion event.
// It includes the status (Completed/Failed) and a truncated result preview.
func FormatToolCompleteDisplay(name, result, status string) string {
	statusText := "Completed"
	if status == "failed" {
		statusText = "Failed"
	}
	content := fmt.Sprintf("🔧 %s: %s", statusText, name)
	if result != "" {
		lines := FormatToolResultLines(result, 3)
		if len(lines) > 0 {
			content += " | " + lines[0]
			if len(lines) > 1 {
				content += "..."
			}
		}
	}
	return content
}
