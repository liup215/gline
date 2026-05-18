package slash

import (
	"strings"
)

// IsStandaloneCommand checks if the text is a standalone slash command
// (starts with / and has a valid command name, with no other non-command content).
func IsStandaloneCommand(text string) bool {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return false
	}
	// Remove the leading /
	text = text[1:]
	// Find first space
	idx := strings.IndexFunc(text, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' })
	if idx == -1 {
		idx = len(text)
	}
	name := text[:idx]
	// Check name is valid: alphanumeric, underscore, hyphen only
	if name == "" {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// ParseCommand extracts the command name and arguments from slash command text.
// Returns ("", "") if the text is not a slash command.
func ParseCommand(text string) (name string, args string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", ""
	}
	// Remove leading /
	text = text[1:]
	// Find first whitespace
	idx := strings.IndexFunc(text, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' })
	if idx == -1 {
		return strings.ToLower(text), ""
	}
	name = strings.ToLower(text[:idx])
	args = strings.TrimSpace(text[idx:])
	return name, args
}
