package view

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/agent"
)

// HeaderData holds the data needed to render the header.
type HeaderData struct {
	Mode      agent.Mode
	Provider  string
	ModelName string
}

// RenderHeader renders the title bar with provider/model info and mode badge.
func RenderHeader(data HeaderData) string {
	modeBadge := "UNKNOWN"
	if data.Mode == agent.ModeAct {
		modeBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("[ACT]")
	} else {
		modeBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("[PLAN]")
	}
	prov := data.Provider
	if prov == "" {
		prov = "-"
	}
	mdl := data.ModelName
	if mdl == "" {
		mdl = "-"
	}
	headerContent := fmt.Sprintf(" 🚀 gline   ●  %s / %s    %s ", prov, mdl, modeBadge)
	return lipgloss.NewStyle().Margin(0, 1).Bold(true).Render(headerContent)
}

// RenderHelp renders the help text at the bottom of the TUI.
func RenderHelp() string {
	return HelpStyle.Render("enter: send • tab: toggle mode • esc: interrupt • ctrl+l: clear • ctrl+c: quit")
}

// RenderInputBox wraps a pre-rendered input view with a border box.
func RenderInputBox(inputView string) string {
	return InputBoxStyle.BorderForeground(lipgloss.Color("#888888")).Render(inputView)
}

// LayoutData holds all the pre-rendered sections for the main layout.
type LayoutData struct {
	CompactBar     string // minimal header (just gline logo)
	Content        string // viewport content
	Menu           string // menu content (empty if no menu)
	InputView      string // pre-rendered textarea view
	InputStatusBar string // status bar below input (model, mode, etc)
	Help           string
	Height         int    // total terminal height
	InputHeight    int    // input box height
}

// RenderLayout assembles all sections into the final TUI output.
// The viewport component manages its own height and clipping, so we just
// assemble all sections in order without applying additional height constraints.
func RenderLayout(data LayoutData) string {
	// The viewport component (data.Content) is already pre-rendered with its
	// assigned height and handles its own clipping/scrolling. We don't need to
	// apply an additional Height() wrapper here - doing so can cause issues when
	// the content exceeds the height, as lipgloss Height() sets minimum height
	// but doesn't clip overflow. Just use the viewport's output directly.
	sections := []string{
		data.CompactBar,
		data.Content,
	}

	// Add menu if present (above input)
	if data.Menu != "" {
		sections = append(sections, data.Menu)
	}

	// Render the input box. The InputBoxStyle includes borders which add 2 lines
	// (top + bottom) to the textarea content height. We don't set an explicit
	// Height here - the style and content height are already managed by the textarea.
	sections = append(sections,
		RenderInputBox(data.InputView),
		data.InputStatusBar,
	)

	if data.Help != "" {
		sections = append(sections, data.Help)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
