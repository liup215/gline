package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// statusModel manages the status bar at the top of the screen.
type statusModel struct {
	provider string
	model    string
	mode     string
	cwd      string
	tokens   int
	maxTokens int
}

func newStatusModel() statusModel {
	return statusModel{
		provider: "openai",
		model:    "gpt-4",
		mode:     "act",
	}
}

func (m *statusModel) updateProvider(provider, model string) {
	m.provider = provider
	m.model = model
}

func (m *statusModel) updateMode(mode string) {
	m.mode = mode
}

func (m *statusModel) updateTokens(tokens, maxTokens int) {
	m.tokens = tokens
	m.maxTokens = maxTokens
}

func (m *statusModel) updateCWD(cwd string) {
	m.cwd = cwd
}

// render renders the status bar.
func (m *statusModel) render(width int) string {
	// Left side: gline title
	title := StatusBarStyle.Render("gline")

	// Mode badge
	modeLabel := strings.ToUpper(m.mode)
	modeBadge := StatusBarModeStyle.Render(modeLabel)

	// Provider/model
	providerInfo := StatusBarModelStyle.Render(fmt.Sprintf("%s/%s", m.provider, m.model))

	// Token usage
	var tokenStr string
	if m.maxTokens > 0 {
		tokenStr = fmt.Sprintf(" %d/%d tok", m.tokens, m.maxTokens)
	} else {
		tokenStr = fmt.Sprintf(" %d tok", m.tokens)
	}

	// CWD (truncated if too long)
	cwdStr := ""
	if m.cwd != "" {
		cwdStr = m.cwd
		if len(cwdStr) > 30 {
			cwdStr = "..." + cwdStr[len(cwdStr)-27:]
		}
		cwdStr = lipgloss.NewStyle().Foreground(ColorDim).Render(" " + cwdStr)
	}

	// Assemble: title ... mode provider tokens cwd
	left := lipgloss.JoinHorizontal(lipgloss.Center, title, " ", modeBadge, " ", providerInfo)
	right := lipgloss.JoinHorizontal(lipgloss.Center, tokenStr, cwdStr)

	// Calculate padding
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := width - leftWidth - rightWidth
	if padding < 0 {
		padding = 0
	}

	bar := left + strings.Repeat(" ", padding) + right

	// Pad to full width and apply background
	return lipgloss.NewStyle().
		Width(width).
		Background(ColorPrimary).
		Foreground(lipgloss.Color("#FFFFFF")).
		Render(bar)
}
