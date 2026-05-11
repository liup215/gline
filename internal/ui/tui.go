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
	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/ui/bridge"
	"github.com/liup215/gline/internal/ui/model"
	"github.com/liup215/gline/internal/ui/viewmodel"
	"github.com/liup215/gline/pkg/types"
)

// Model represents the TUI state.
// Business data (messages, tool history, mode, provider, model name) lives in
// *model.Conversation; UI-specific state (activeAssistantIndex, isProcessing,
// etc.) stays here.
type Model struct {
	// Domain model (extracted from the former god-object)
	conversation *model.Conversation

	// UI components
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	// UI-only state
	inputHeight          int
	toolAreaHeight       int
	isProcessing         bool
	isStreaming          bool
	err                  error
	currentTool          string
	activeAssistantIndex int

	// Agent components
	agentInstance *agent.BaseAgent
	ctx           context.Context
	cancelFn      context.CancelFunc

	// Bridge channel: TUIBridge sends events here; a forwarding goroutine
	// relays them to tea.Program.Send so that Bridge stays decoupled from Bubbletea.
	eventCh chan bridge.AgentEvent
	done    chan struct{} // signals the forwarding goroutine to stop

	// Pending reply channel when the UI is answering an AskFollowupQuestion
	pendingReply chan string

	// ViewModel derives rendered display state from the conversation.
	convVM *viewmodel.ConversationViewModel

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

	conv := model.NewConversation()
	conv.Provider = providerName
	conv.ModelName = modelName

	return &Model{
		textarea:             ta,
		viewport:             vp,
		spinner:              s,
		conversation:         conv,
		convVM:               viewmodel.NewConversationViewModel(),
		inputHeight:          3,
		toolAreaHeight:       3,
		activeAssistantIndex: -1,
		agentInstance:        agentInstance,
		ctx:                  context.Background(),
	}
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

	case bridge.AskQuestionEvent:
		// Display the follow-up question and options, set pending reply channel
		m.conversation.AppendMessage(model.Message{
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

	case bridge.AgentEvent:
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
	if m.conversation.Mode == agent.ModeAct {
		modeBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("[ACT]")
	} else {
		modeBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("[PLAN]")
	}
	prov := m.conversation.Provider
	if prov == "" {
		prov = "-"
	}
	mdl := m.conversation.ModelName
	if mdl == "" {
		mdl = "-"
	}
	headerContent := fmt.Sprintf(" 🚀 gline   ●  %s / %s    %s ", prov, mdl, modeBadge)
	header := lipgloss.NewStyle().Margin(0, 1).Bold(true).Render(headerContent)
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
	m.conversation.ClearToolHistory()

	// Add user message
	m.conversation.AppendMessage(model.Message{
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
	m.conversation.AppendMessage(model.Message{
		Role:      types.RoleSystem,
		Content:   content,
		Timestamp: time.Now(),
	})
	m.updateViewport()
}

// Run starts the TUI with a Bridge-based event forwarding architecture.
// A buffered channel carries events from TUIBridge (Agent side) to the
// Bubbletea Program; a goroutine relays them via program.Send.
func Run(agentInstance *agent.BaseAgent) error {
	m := New(agentInstance)
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Create the event channel and wire it into the Model
	eventCh := make(chan bridge.AgentEvent, 64)
	m.eventCh = eventCh
	done := make(chan struct{})
	m.done = done

	// Forward goroutine: reads events from the bridge channel and sends
	// them into the Bubbletea event loop. This keeps TUIBridge decoupled
	// from tea.Program while preserving ordered delivery.
	go func() {
		for {
			select {
			case evt := <-eventCh:
				p.Send(evt)
			case <-done:
				return
			}
		}
	}()

	_, err := p.Run()
	close(done) // signal the forwarding goroutine to stop
	return err
}
