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
// Content fills remaining space so menu appears directly above input.
func RenderLayout(data LayoutData) string {
	// Calculate menu height
	menuHeight := 0
	if data.Menu != "" {
		menuHeight = 7 // approx menu height
	}

	// Calculate content height to fill remaining space
	// Total: header(1) + content(flexible) + menu(? ) + input(data.InputHeight) + status(1) + help(1)
	contentHeight := data.Height - 1 - menuHeight - data.InputHeight - 1 - 1 // -1 for header, -1 for status, -1 for help
	if contentHeight < 3 {
		contentHeight = 3
	}

	// Content fills remaining space (pushes menu+input+help to bottom)
	contentWithHeight := lipgloss.NewStyle().Height(contentHeight).Render(data.Content)

	sections := []string{
		data.CompactBar,
		contentWithHeight,
	}

	// Add menu if present (above input)
	if data.Menu != "" {
		sections = append(sections, data.Menu)
	}

	sections = append(sections,
		RenderInputBox(data.InputView),
		data.InputStatusBar,
	)

	if data.Help != "" {
		sections = append(sections, data.Help)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}