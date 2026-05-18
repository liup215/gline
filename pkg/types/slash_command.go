// Package types defines slash command types shared across the codebase.
package types

// SlashCommandSection categorizes commands into groups.
type SlashCommandSection string

const (
	SectionDefault SlashCommandSection = "default"
	SectionCustom  SlashCommandSection = "custom"
)

// SlashCommandHandler is the callback invoked when a slash command executes.
// It receives the full command text (including any args after the command name).
// The handler may return a bool indicating whether the command was fully consumed
// (standalone) or should remain in the input buffer (parameterized).
type SlashCommandHandler func(args string) (consumed bool, err error)

// SlashCommand defines a single slash command available in the TUI.
type SlashCommand struct {
	// Name is the command identifier without the leading slash.
	// e.g. "clear" for /clear.
	Name string

	// Description is a short human-readable explanation shown in the menu.
	Description string

	// Section groups the command in the menu (default vs custom).
	Section SlashCommandSection

	// Handler is invoked when the command is executed.
	Handler SlashCommandHandler
}
