package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the TUI.
type KeyMap struct {
	// Navigation
	Up   key.Binding
	Down key.Binding
	Help key.Binding

	// Actions
	Submit     key.Binding
	Stop       key.Binding
	NewChat    key.Binding
	ToggleMode key.Binding
	Clear      key.Binding

	// Sidebar
	ToggleSidebar key.Binding

	// History navigation in input
	PrevInput key.Binding
	NextInput key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "scroll down"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),

		// Actions
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send"),
		),
		Stop: key.NewBinding(
			key.WithKeys("esc", "ctrl+c"),
			key.WithHelp("esc", "stop"),
		),
		NewChat: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "new chat"),
		),
		ToggleMode: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle plan/act"),
		),
		Clear: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear"),
		),

		// Sidebar
		ToggleSidebar: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl+b", "sidebar"),
		),

		// History navigation
		PrevInput: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "prev input"),
		),
		NextInput: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "next input"),
		),
	}
}
