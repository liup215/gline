package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/storage"
)

// --- Tea Messages ---

// appModel is the root Bubbletea model for the TUI.
type appModel struct {
	// Layout
	width  int
	height int

	// Sub-models
	status  statusModel
	chat    chatModel
	input   *inputModel
	sidebar sidebarModel

	// Agent
	agent    agent.Agent
	callback *TuiCallback
	program  *tea.Program // set after program creation
	store    storage.Store

	// State
	isLoading   bool
	sidebarOpen bool
	followup    *followupState
	cancelFunc  context.CancelFunc
	quitting    bool

	// Config
	keys KeyMap
}

// followupState holds an active followup question.
type followupState struct {
	question string
	options  []string
	answerCh chan<- string
}

// NewApp creates a new TUI application model.
func NewApp(ag agent.Agent, store storage.Store) *appModel {
	return &appModel{
		agent:   ag,
		keys:    DefaultKeyMap(),
		status:  newStatusModel(),
		store:   store,
		sidebar: newSidebarModel(store),
		input:   newInputModel(80),
	}
}

// SetProgram sets the tea.Program reference (called after program creation).
func (m *appModel) SetProgram(p *tea.Program) {
	m.program = p
	m.callback = NewTuiCallback(p)
}

// Init implements tea.Model.
func (m *appModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tea.SetWindowTitle("gline"),
	)
}

// Update implements tea.Model.
func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		// Focus input on first WindowSizeMsg
		if !m.input.textinput.Focused() {
			m.input.Focus()
		}
		// Show welcome message on first size
		if len(m.chat.messages) == 0 {
			m.chat.appendMessage(ChatMessage{
				Role:    RoleSystem,
				Content: "Welcome to gline! Type a message to start. (Tab: mode, /: commands)",
			})
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	// --- Agent messages ---
	case streamStartMsg:
		m.isLoading = true
		m.chat.startStreaming()
		return m, nil

	case contentDeltaMsg:
		m.chat.appendStreamContent(msg.Delta)
		return m, nil

	case reasoningDeltaMsg:
		// Could show in a collapsible panel; for now, ignore
		return m, nil

	case toolStartMsg:
		m.chat.addToolStart(msg.ID, msg.Name, msg.Input)
		return m, nil

	case toolCompleteMsg:
		m.chat.setToolResult(msg.ID, msg.Result)
		return m, nil

	case streamCompleteMsg:
		m.isLoading = false
		m.chat.finishStreaming()
		m.input.Reset()
		m.input.Focus()
		return m, nil

	case streamErrorMsg:
		m.isLoading = false
		m.chat.finishStreaming()
		m.chat.appendMessage(ChatMessage{
			Role:    RoleSystem,
			Content: fmt.Sprintf("Error: %v", msg.Err),
		})
		m.input.Reset()
		m.input.Focus()
		return m, nil

	case taskCreatedMsg:
		m.status.updateCWD(m.getWorkingDir())
		return m, nil

	case followupQuestionMsg:
		m.followup = &followupState{
			question: msg.Question,
			options:  msg.Options,
			answerCh: msg.AnswerCh,
		}
		return m, nil

	case providerInfoMsg:
		m.status.updateProvider(msg.Provider, msg.Model)
		return m, nil

	case tokenUsageMsg:
		m.status.updateTokens(msg.InputTokens+msg.OutputTokens, 0)
		return m, nil
	}

	// Update sub-models
	cmd := m.chat.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleKey processes keyboard input.
func (m *appModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle followup question
	if m.followup != nil {
		return m.handleFollowupKey(msg)
	}

	// Global keys
	switch {
	case msg.String() == "ctrl+c" || msg.String() == "ctrl+d":
		if m.isLoading {
			// Stop the running task first
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			m.agent.Abort()
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit

	case msg.String() == "ctrl+n":
		return m.handleNewChat()

	case msg.String() == "tab":
		return m.handleToggleMode()

	case msg.String() == "ctrl+b":
		m.sidebarOpen = !m.sidebarOpen
		if m.sidebarOpen {
			m.sidebar.loadTasks()
		}
		m.updateLayout()
		return m, nil

	case msg.String() == "ctrl+l":
		m.chat.clearMessages()
		return m, nil
	}

	// If agent is running, only allow stop
	if m.isLoading {
		switch {
		case msg.String() == "esc":
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			m.agent.Abort()
			return m, nil
		}
		// Ignore other keys while loading
		return m, nil
	}

	// Slash autocomplete navigation
	if m.input.slashVisible {
		switch msg.String() {
		case "up":
			if m.input.slashSelected > 0 {
				m.input.slashSelected--
			}
			return m, nil
		case "down":
			if m.input.slashSelected < len(m.input.slashSuggestions)-1 {
				m.input.slashSelected++
			}
			return m, nil
		case "tab", "enter":
			m.input.selectSlashCommand()
			return m, nil
		case "esc":
			m.input.slashVisible = false
			return m, nil
		}
	}

	// Input handling
	switch {
	case msg.String() == "enter":
		return m.handleSubmit()

	case msg.String() == "up":
		if m.input.Value() == "" {
			m.input.prevHistory()
			return m, nil
		}

	case msg.String() == "down":
		if m.input.Value() == "" || m.input.histIdx >= 0 {
			m.input.nextHistory()
			return m, nil
		}
	}

	// Forward to input
	cmd := m.input.Update(msg)
	return m, cmd
}

// handleFollowupKey processes keys when a followup question is active.
func (m *appModel) handleFollowupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.followup == nil {
		return m, nil
	}

	switch msg.String() {
	case "esc":
		// Cancel
		close(m.followup.answerCh)
		m.followup = nil
		m.input.Focus()
		return m, nil

	case "enter":
		val := m.input.Value()
		if val == "" && len(m.followup.options) > 0 {
			val = "1" // default to first option
		}
		m.followup.answerCh <- val
		m.followup = nil
		m.input.Reset()
		m.input.Focus()
		return m, nil
	}

	// Allow number selection
	if len(msg.String()) == 1 && msg.String()[0] >= '1' && msg.String()[0] <= '9' {
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < len(m.followup.options) {
			m.followup.answerCh <- m.followup.options[idx]
			m.followup = nil
			m.input.Reset()
			m.input.Focus()
			return m, nil
		}
	}

	// Forward to input
	cmd := m.input.Update(msg)
	return m, cmd
}

// handleSubmit sends the current input to the agent.
func (m *appModel) handleSubmit() (tea.Model, tea.Cmd) {
	prompt := strings.TrimSpace(m.input.Value())
	if prompt == "" {
		return m, nil
	}

	// Add to history
	m.input.addToHistory(prompt)

	// Display user message
	m.chat.appendMessage(ChatMessage{
		Role:    RoleUser,
		Content: prompt,
	})

	// Check for slash commands
	if strings.HasPrefix(prompt, "/") {
		return m.handleSlashCommand(prompt)
	}

	// Start agent run
	m.isLoading = true
	m.input.Reset()

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	ag := m.agent
	cb := m.callback
	p := m.program

	return m, func() tea.Msg {
		go func() {
			// Send provider info at start
			if baseAg, ok := ag.(*agent.BaseAgent); ok {
				if prov := baseAg.GetProvider(); prov != nil {
					p.Send(providerInfoMsg{
						Provider: prov.GetProviderName(),
						Model:    prov.GetModel(),
					})
				}
			}
			if err := ag.RunWithCallback(ctx, prompt, cb); err != nil {
				if p != nil {
					p.Send(streamErrorMsg{Err: err})
				}
			}
		}()
		return nil
	}
}

// handleSlashCommand processes a slash command.
func (m *appModel) handleSlashCommand(prompt string) (tea.Model, tea.Cmd) {
	parts := strings.SplitN(strings.TrimPrefix(prompt, "/"), " ", 2)
	name := parts[0]
	_ = parts[1] // args reserved for future use

	switch name {
	case "clear", "cls":
		m.chat.clearMessages()
		m.input.Reset()
		return m, nil

	case "exit", "q", "quit":
		return m, tea.Quit

	case "help":
		m.chat.appendMessage(ChatMessage{
			Role:    RoleSystem,
			Content: m.buildHelpText(),
		})
		m.input.Reset()
		return m, nil

	case "newtask":
		m.chat.clearMessages()
		m.input.Reset()
		return m, nil

	case "smol", "compact":
		if m.agent != nil {
			compacted := m.agent.Compact()
			msg := "Conversation compacted"
			if !compacted {
				msg = "Nothing to compact"
			}
			m.chat.appendMessage(ChatMessage{
				Role:    RoleSystem,
				Content: msg,
			})
		}
		m.input.Reset()
		return m, nil

	default:
		m.chat.appendMessage(ChatMessage{
			Role:    RoleSystem,
			Content: fmt.Sprintf("Unknown command: /%s. Type /help for available commands.", name),
		})
		m.input.Reset()
		return m, nil
	}
}

// handleNewChat starts a new chat session.
func (m *appModel) handleNewChat() (tea.Model, tea.Cmd) {
	m.chat.clearMessages()
	if m.agent != nil {
		if base, ok := m.agent.(*agent.BaseAgent); ok {
			base.ResetTask()
			base.SetWorkingDir("")
		}
	}
	m.input.Reset()
	m.input.Focus()
	return m, nil
}

// handleToggleMode switches between plan and act modes.
func (m *appModel) handleToggleMode() (tea.Model, tea.Cmd) {
	if m.agent == nil {
		return m, nil
	}

	current := m.agent.GetMode()
	var newMode agent.Mode
	if current == agent.ModePlan {
		newMode = agent.ModeAct
	} else {
		newMode = agent.ModePlan
	}

	if err := m.agent.SetMode(newMode); err != nil {
		m.chat.appendMessage(ChatMessage{
			Role:    RoleSystem,
			Content: fmt.Sprintf("Error: %v", err),
		})
		return m, nil
	}

	m.status.updateMode(string(newMode))
	return m, nil
}

// buildHelpText returns help text for slash commands.
func (m *appModel) buildHelpText() string {
	var b strings.Builder
	b.WriteString("Available commands:\n\n")
	commands := []struct{ name, desc string }{
		{"/clear", "Clear conversation"},
		{"/help", "Show this help"},
		{"/exit, /q", "Exit gline"},
		{"/newtask", "Start new task"},
		{"/smol", "Compact context"},
	}
	for _, c := range commands {
		b.WriteString(fmt.Sprintf("  %-16s %s\n", c.name, c.desc))
	}
	b.WriteString("\nShortcuts:\n")
	b.WriteString("  Tab       Toggle Plan/Act mode\n")
	b.WriteString("  Ctrl+N    New chat\n")
	b.WriteString("  Ctrl+B    Toggle sidebar\n")
	b.WriteString("  Ctrl+L    Clear screen\n")
	b.WriteString("  Esc       Stop / cancel\n")
	b.WriteString("  Ctrl+C    Quit\n")
	return b.String()
}

// getWorkingDir returns the current working directory from the agent.
func (m *appModel) getWorkingDir() string {
	if m.agent == nil {
		return ""
	}
	if base, ok := m.agent.(*agent.BaseAgent); ok {
		return base.GetWorkingDir()
	}
	return ""
}

// updateLayout recalculates dimensions for sub-models.
func (m *appModel) updateLayout() {
	statusHeight := 1
	inputHeight := 3 // prompt + input + padding
	chatHeight := m.height - statusHeight - inputHeight
	if chatHeight < 1 {
		chatHeight = 1
	}

	chatWidth := m.width
	if chatWidth < 1 {
		chatWidth = 80
	}
	if m.sidebarOpen {
		sidebarWidth := 24
		chatWidth = m.width - sidebarWidth
		if chatWidth < 40 {
			chatWidth = 40
		}
		m.sidebar.setWidthHeight(sidebarWidth, m.height)
	}

	m.chat.setWidthHeight(chatWidth, chatHeight)
	m.input.setWidth(chatWidth)
}

// View implements tea.Model.
func (m *appModel) View() string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	var b strings.Builder

	// Status bar
	b.WriteString(m.status.render(w))
	b.WriteString("\n")

	// Main content area
	chatView := m.chat.View()

	// Sidebar
	if m.sidebarOpen {
		sidebar := m.renderSidebar()
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, sidebar, chatView))
	} else {
		b.WriteString(chatView)
	}

	// Followup question overlay
	if m.followup != nil {
		b.WriteString(m.renderFollowup())
		b.WriteString("\n")
	} else {
		// Input area
		b.WriteString(strings.Repeat("─", w))
		b.WriteString("\n")
		b.WriteString(m.input.View())
	}

	// Loading indicator
	if m.isLoading {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("  ⏳ Processing..."))
	}

	return b.String()
}

// renderSidebar renders the sidebar.
func (m *appModel) renderSidebar() string {
	return m.sidebar.render()
}

// renderFollowup renders the followup question overlay.
func (m *appModel) renderFollowup() string {
	if m.followup == nil {
		return ""
	}

	w := m.width
	if w <= 0 {
		w = 80
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", w))
	b.WriteString("\n")
	b.WriteString(ErrorStyle.Render("❓ " + m.followup.question))
	b.WriteString("\n\n")

	for i, opt := range m.followup.options {
		num := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(fmt.Sprintf("%d", i+1))
		b.WriteString(fmt.Sprintf("  %s. %s\n", num, opt))
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("  Enter number or type answer, Esc to cancel"))

	return b.String()
}
