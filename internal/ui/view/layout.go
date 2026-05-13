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
	return InputBoxStyle.BorderForeground(lipgloss.Color("#888888")).MarginLeft(1).Render(inputView)
}

// LayoutData holds all the pre-rendered sections for the main layout.
type LayoutData struct {
	CompactBar string // merged header + status bar
	Content    string // viewport content
	ToolArea   string
	InputView  string // pre-rendered textarea view
	Help       string
}

// RenderLayout assembles all sections into the final TUI output.
func RenderLayout(data LayoutData) string {
	sections := []string{
		data.CompactBar,
		data.Content,
		data.ToolArea,
		RenderInputBox(data.InputView),
		data.Help,
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}