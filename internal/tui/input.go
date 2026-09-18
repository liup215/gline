package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// slashCommand defines a slash command for autocomplete.
type slashCommand struct {
	Name        string
	Description string
}

// defaultSlashCommands are the available slash commands.
var defaultSlashCommands = []slashCommand{
	{Name: "clear", Description: "Clear conversation"},
	{Name: "help", Description: "Show help"},
	{Name: "exit", Description: "Exit gline"},
	{Name: "q", Description: "Exit (shorthand)"},
	{Name: "newtask", Description: "Start new task"},
	{Name: "smol", Description: "Compact context"},
	{Name: "compact", Description: "Compact (alias)"},
	{Name: "history", Description: "Show history"},
	{Name: "reload", Description: "Reload rules"},
}

// inputModel manages the text input area at the bottom of the screen.
type inputModel struct {
	textinput textinput.Model
	history   []string // input history
	histIdx   int      // current position in history (-1 = not browsing)
	width     int

	// Slash command autocomplete
	slashSuggestions []slashCommand
	slashVisible    bool
	slashSelected   int
}

func newInputModel(width int) *inputModel {
	ti := textinput.New()
	ti.Placeholder = "Type a message... (Tab: mode, /: commands)"
	ti.CharLimit = 8192
	if width <= 0 {
		width = 80
	}
	ti.Width = width - 4

	return &inputModel{
		textinput:       ti,
		history:         make([]string, 0),
		histIdx:         -1,
		width:           width,
		slashSuggestions: defaultSlashCommands,
	}
}

func (m *inputModel) setWidth(width int) {
	if width <= 0 {
		width = 80
	}
	m.width = width
	m.textinput.Width = width - 4
}

// SetValue sets the input value.
func (m *inputModel) SetValue(val string) {
	m.textinput.SetValue(val)
}

// Value returns the current input value.
func (m *inputModel) Value() string {
	return m.textinput.Value()
}

// Focus focuses the input.
func (m *inputModel) Focus() {
	if m.textinput.Focused() {
		return // already focused
	}
	m.textinput.Focus()
}

// Blur blurs the input.
func (m *inputModel) Blur() {
	m.textinput.Blur()
}

// Reset clears the input and resets history position.
func (m *inputModel) Reset() {
	m.textinput.SetValue("")
	m.histIdx = -1
	m.slashVisible = false
}

// updateSlashSuggestions updates the autocomplete suggestions based on input.
func (m *inputModel) updateSlashSuggestions() {
	val := m.textinput.Value()
	if !strings.HasPrefix(val, "/") {
		m.slashVisible = false
		return
	}

	prefix := strings.ToLower(val)
	var matches []slashCommand
	for _, cmd := range defaultSlashCommands {
		if strings.HasPrefix("/"+cmd.Name, prefix) {
			matches = append(matches, cmd)
		}
	}

	if len(matches) == 0 {
		m.slashVisible = false
		return
	}

	m.slashSuggestions = matches
	m.slashVisible = true
	m.slashSelected = 0
}

// selectSlashCommand selects the current highlighted slash command.
func (m *inputModel) selectSlashCommand() {
	if !m.slashVisible || m.slashSelected >= len(m.slashSuggestions) {
		return
	}
	cmd := m.slashSuggestions[m.slashSelected]
	m.textinput.SetValue("/" + cmd.Name + " ")
	m.slashVisible = false
}

// addToHistory adds the input to history.
func (m *inputModel) addToHistory(val string) {
	if val == "" {
		return
	}
	if len(m.history) > 0 && m.history[len(m.history)-1] == val {
		return
	}
	m.history = append(m.history, val)
}

// prevHistory navigates to the previous input in history.
func (m *inputModel) prevHistory() {
	if len(m.history) == 0 {
		return
	}
	if m.histIdx == -1 {
		m.histIdx = len(m.history) - 1
	} else if m.histIdx > 0 {
		m.histIdx--
	}
	m.textinput.SetValue(m.history[m.histIdx])
}

// nextHistory navigates to the next input in history.
func (m *inputModel) nextHistory() {
	if m.histIdx == -1 {
		return
	}
	if m.histIdx < len(m.history)-1 {
		m.histIdx++
		m.textinput.SetValue(m.history[m.histIdx])
	} else {
		m.histIdx = -1
		m.textinput.SetValue("")
	}
}

// Update handles input updates.
func (m *inputModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.textinput, cmd = m.textinput.Update(msg)
	m.updateSlashSuggestions()
	return cmd
}

// View renders the input area.
func (m *inputModel) View() string {
	var b strings.Builder

	prompt := PromptStyle.Render("❯ ")
	b.WriteString(prompt)
	b.WriteString(m.textinput.View())

	// Render slash autocomplete popup
	if m.slashVisible && len(m.slashSuggestions) > 0 {
		b.WriteString("\n")
		b.WriteString(m.renderSlashPopup())
	}

	return b.String()
}

// renderSlashPopup renders the slash command autocomplete popup.
func (m *inputModel) renderSlashPopup() string {
	var b strings.Builder

	popupStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorDim).
		Padding(0, 1)

	var items []string
	for i, cmd := range m.slashSuggestions {
		name := fmt.Sprintf("/%-12s", cmd.Name)
		desc := cmd.Description

		if i == m.slashSelected {
			name = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render(name)
			desc = lipgloss.NewStyle().Foreground(ColorAccent).Render(desc)
		} else {
			name = lipgloss.NewStyle().Foreground(ColorText).Render(name)
			desc = lipgloss.NewStyle().Foreground(ColorMuted).Render(desc)
		}

		items = append(items, name+" "+desc)
	}

	content := strings.Join(items, "\n")
	b.WriteString(popupStyle.Render(content))

	return b.String()
}
