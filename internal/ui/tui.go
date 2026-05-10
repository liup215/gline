// Package ui provides the TUI (Terminal User Interface) for gline
package ui

import (
"context"
"fmt"
"time"

"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/pkg/types"
)

// ToolStatus represents the status of a tool call in the UI
type ToolStatus struct {
	Name      string
	Status    string // "running", "completed", "failed"
	StartTime time.Time
}

// Message represents a single message in the conversation
type Message struct {
	Role      types.Role
	Content   string
	ToolCalls []types.ToolCall
	Options   []string // Options for ask_followup_question display (nil for non-question messages)
	Timestamp time.Time

	// Cached rendered markdown to avoid repeated glamour rendering.
	Rendered           string
	RenderedWrapWidth  int
	RenderedSource     string // original Content used to produce Rendered
}

// Model represents the TUI state
type Model struct {
	// UI components
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	// State
	messages            []Message
	mode                agent.Mode
	provider            string
	model               string
	inputHeight         int
	toolAreaHeight      int
	isProcessing        bool
	isStreaming         bool
	err                 error
	currentTool         string
	activeAssistantIndex int

	// Tool status history (displayed in a fixed area below the viewport)
	toolHistory []ToolStatus

	// Agent components
	agentInstance *agent.BaseAgent
	ctx           context.Context
	cancelFn      context.CancelFunc

	// Program reference for sending messages
	program *tea.Program

	// Pending reply channel when the UI is answering an AskFollowupQuestion
	pendingReply chan string

	// Glamour renderer cache for current wrap width (avoid recreating renderer every redraw)
	renderer          *glamour.TermRenderer
	rendererWrapWidth int

	// Dimensions
	width  int
	height int
}


// New creates a new TUI model
func New(agentInstance *agent.BaseAgent) *Model {
	// Create textarea for input
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.Prompt = ""

	// Create viewport for messages
	vp := viewport.New(80, 20)

	// Create spinner for loading animation
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00AAFF"))

	// Get provider and model info from agent
	var providerName, modelName string
	if agentInstance != nil {
		if provider := agentInstance.GetProvider(); provider != nil {
			providerName = provider.GetProviderName()
			modelName = provider.GetModel()
		}
	}

	return &Model{
		textarea:             ta,
		viewport:             vp,
		spinner:              s,
		messages:             []Message{},
		mode:                 agent.ModeAct,
		provider:             providerName,
		model:                modelName,
		inputHeight:          3,
		toolAreaHeight:       3,
		activeAssistantIndex: -1,
		agentInstance:        agentInstance,
		ctx:                  context.Background(),
	}
}

// SetProgram sets the program reference for sending messages
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// Init initializes the TUI
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		textarea.Blink,
		m.spinner.Tick,
	)
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
case tea.WindowSizeMsg:
	cmds = append(cmds, handleWindowSize(m, msg)...)

case tea.KeyMsg:
	cmds = append(cmds, handleKeyMsg(m, msg)...)

	case spinner.TickMsg:
		// Update spinner animation
		if m.isProcessing {
			newSpinner, cmd := m.spinner.Update(msg)
			m.spinner = newSpinner
			cmds = append(cmds, cmd)
		}

	case askQuestionMsg:
		// Display the follow-up question and options, set pending reply channel
		m.messages = append(m.messages, Message{
			Role:      types.RoleSystem,
			Content:   "❓ " + msg.Question,
			Options:   msg.Options,
			Timestamp: time.Now(),
		})
		// Set the reply channel so Enter will send the answer back to the agent
		m.pendingReply = msg.Reply
		m.textarea.Reset()
		m.textarea.Placeholder = "Type option number or your answer..."
		m.textarea.Focus()
		cmds = append(cmds, textarea.Blink)
		m.updateViewport()

	case agentUpdateMsg:
		// Delegate to extracted handler
		cmds = append(cmds, handleAgentUpdate(m, msg)...)

	// Handle internal messages for real-time viewport updates
	case tickMsg:
		if m.isProcessing {
			cmds = append(cmds, m.tick())
		}
	}

	// Update textarea
	newTextarea, textareaCmd := m.textarea.Update(msg)
	m.textarea = newTextarea
	cmds = append(cmds, textareaCmd)

	// Update viewport
	newViewport, viewportCmd := m.viewport.Update(msg)
	m.viewport = newViewport
	cmds = append(cmds, viewportCmd)

	return m, tea.Batch(cmds...)
}

// tickMsg is used for periodic viewport refresh during processing
type tickMsg time.Time

// tick returns a command that sends a tick message for periodic updates
func (m *Model) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// View renders the TUI
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Build the view
	var sections []string

	// Header (title + provider/model + mode badge)
	modeBadge := "UNKNOWN"
	if m.mode == agent.ModeAct {
		modeBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("[ACT]")
	} else {
		modeBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("[PLAN]")
	}
	prov := m.provider
	if prov == "" {
		prov = "-"
	}
	mdl := m.model
	if mdl == "" {
		mdl = "-"
	}
	headerContent := fmt.Sprintf(" 🚀 gline   ●  %s / %s    %s ", prov, mdl, modeBadge)
	header := lipgloss.NewStyle().Margin(0,1).Bold(true).Render(headerContent)
	sections = append(sections, header)

	// Messages viewport
	sections = append(sections, m.viewport.View())

	// Tool status area
	sections = append(sections, m.renderToolArea())

	// Input area — textarea wrapped with a border
	// textarea width was already set via SetWidth(innerWidth) in the WindowSizeMsg handler,
	// so we render it directly. inputBoxStyle adds border (2) + padding (6) + MarginLeft (1) = 9,
	// which makes the total width exactly m.width.
	inputBox := inputBoxStyle.BorderForeground(lipgloss.Color("#888888")).MarginLeft(1).Render(m.textarea.View())
	sections = append(sections, inputBox)

	// Status bar
	status := m.renderStatusBar()
	sections = append(sections, status)

	// Help text
	help := helpStyle.Render("enter: send • tab: toggle mode • esc: interrupt • ctrl+l: clear • ctrl+c: quit")
	sections = append(sections, help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// sendMessage adds a user message and prepares for response
func (m *Model) sendMessage(content string) {
	// Clear tool history for new conversation turn
	m.toolHistory = nil

	// Add user message
	m.messages = append(m.messages, Message{
		Role:      types.RoleUser,
		Content:   content,
		Timestamp: time.Now(),
	})

	// Do not pre-create assistant message slot here.
	// The UI will create the assistant slot when the agent signals streamStart.
	m.activeAssistantIndex = -1

	m.isProcessing = true
	m.isStreaming = false
	m.updateViewport()
}

// addErrorMessage adds an error message
func (m *Model) addErrorMessage(content string) {
	m.messages = append(m.messages, Message{
		Role:      types.RoleSystem,
		Content:   content,
		Timestamp: time.Now(),
	})
	m.updateViewport()
}




type askQuestionMsg struct {
	Question string
	Options  []string
	Reply    chan string
}

// agentUpdateMsg represents an update from the agent callback
type agentUpdateMsg struct {
	updateType string
	content    string
	toolName   string
	toolInput  string
	toolResult string
	err        error
}


// Run starts the TUI
func Run(agentInstance *agent.BaseAgent) error {
	model := New(agentInstance)
	p := tea.NewProgram(model, tea.WithAltScreen())
	model.SetProgram(p)
	_, err := p.Run()
	return err
}