package ui

import (
	"strings"

	"github.com/liup215/gline/internal/slash"
	"github.com/liup215/gline/pkg/types"
)

// SlashMenuState tracks the current slash command menu state.
type SlashMenuState struct {
	Active    bool               // whether slash mode is active
	Query     string             // current query after /
	Selected  int                // selected item index in filtered list
	Filtered  []*types.SlashCommand // filtered command list
	Registry  *slash.Registry    // command registry
}

// NewSlashMenuState creates a new slash menu state with the given registry.
func NewSlashMenuState(registry *slash.Registry) *SlashMenuState {
	return &SlashMenuState{
		Registry: registry,
		Filtered: make([]*types.SlashCommand, 0),
	}
}

// EnterSlashMode enters slash command mode and initializes the filter.
func (s *SlashMenuState) EnterSlashMode() {
	s.Active = true
	s.Query = ""
	s.Selected = 0
	s.Filtered = s.Registry.GetAll()
}

// UpdateQuery updates the filter query based on the current input text.
func (s *SlashMenuState) UpdateQuery(text string, cursorPos int) {
	if !s.Active {
		return
	}

	// Check if cursor is still in slash context (no space after /)
	if !IsSlashPrefix(text, cursorPos) {
		// Cursor moved out of slash context, exit slash mode
		s.Active = false
		s.Filtered = nil
		return
	}

	query := extractSlashQuery(text, cursorPos)
	s.Query = query
	if query == "" {
		// Empty query (just "/") shows all commands
		s.Filtered = s.Registry.GetAll()
	} else {
		s.Filtered = s.Registry.Filter(query)
	}

	// Clamp selection
	if s.Selected >= len(s.Filtered) {
		s.Selected = 0
	}
}

// Next selects the next command (wraps around).
func (s *SlashMenuState) Next() {
	if len(s.Filtered) == 0 {
		return
	}
	s.Selected = (s.Selected + 1) % len(s.Filtered)
}

// Prev selects the previous command (wraps around).
func (s *SlashMenuState) Prev() {
	if len(s.Filtered) == 0 {
		return
	}
	s.Selected--
	if s.Selected < 0 {
		s.Selected = len(s.Filtered) - 1
	}
}

// SelectedCommand returns the currently selected command, or nil if none.
func (s *SlashMenuState) SelectedCommand() *types.SlashCommand {
	if s.Selected < 0 || s.Selected >= len(s.Filtered) {
		return nil
	}
	return s.Filtered[s.Selected]
}

// ExitSlashMode deactivates slash mode.
func (s *SlashMenuState) ExitSlashMode() {
	s.Active = false
	s.Query = ""
	s.Selected = 0
	s.Filtered = nil
}

// IsSlashPrefix returns true if text should trigger slash mode.
// Slash mode is triggered when text before cursor is a lone / followed by optional chars.
func IsSlashPrefix(text string, cursorPos int) bool {
	if cursorPos < 1 {
		return false
	}
	before := text[:cursorPos]
	// Must end with a slash that's either at start or preceded by whitespace
	lastSlash := strings.LastIndex(before, "/")
	if lastSlash < 0 {
		return false
	}
	// Check that there's no space after the last slash in the before-cursor text
	afterSlash := before[lastSlash+1:]
	if strings.ContainsAny(afterSlash, " \t\n") {
		return false
	}
	// The slash must be at position 0 or preceded by whitespace
	if lastSlash > 0 {
		prev := before[lastSlash-1]
		if prev != ' ' && prev != '\t' && prev != '\n' {
			return false
		}
	}
	return true
}

// extractSlashQuery extracts the query text after the last slash.
func extractSlashQuery(text string, cursorPos int) string {
	if cursorPos < 1 {
		return ""
	}
	before := text[:cursorPos]
	lastSlash := strings.LastIndex(before, "/")
	if lastSlash < 0 {
		return ""
	}
	afterSlash := before[lastSlash+1:]
	if strings.ContainsAny(afterSlash, " \t\n") {
		return "" // space after slash exits slash mode
	}
	return afterSlash
}
