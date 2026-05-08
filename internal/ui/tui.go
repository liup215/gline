// Package ui provides the TUI (Terminal User Interface) for gline
package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/pkg/types"
)

// Message represents a single message in the conversation
type Message struct {
	Role      types.Role
	Content   string
	ToolCalls []types.ToolCall
	Timestamp time.Time
}

// Model represents the TUI state
type Model struct {
	// UI components
	viewport viewport.Model
	textarea textarea.Model

	// State
	messages      []Message
	mode          agent.Mode
	provider      string
	model         string
	inputHeight   int
	isProcessing  bool
	err           error

	// Agent components
	agentInstance *agent.BaseAgent
	ctx           context.Context

	// Dimensions
	width  int
	height int
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56C4")).
		MarginLeft(2)

	userStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Bold(true)

	assistantStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00AAFF")).
		Bold(true)

	systemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000"))

	toolStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFA500"))

	statusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))
)

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

	return &Model{
		textarea:      ta,
		viewport:      vp,
		messages:      []Message{},
		mode:          agent.ModeAct,
		inputHeight:   3,
		agentInstance: agentInstance,
		ctx:           context.Background(),
	}
}

// Init initializes the TUI
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		textarea.Blink,
	)
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - m.inputHeight - 2 // Reserve space for status bar
		m.textarea.SetWidth(msg.Width)
		m.updateViewport()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyTab:
			// Toggle between Plan and Act mode
			if m.mode == agent.ModePlan {
				m.mode = agent.ModeAct
			} else {
				m.mode = agent.ModePlan
			}
			if m.agentInstance != nil {
				m.agentInstance.SetMode(m.mode)
			}
			m.updateViewport()

		case tea.KeyEnter:
			if msg.Alt {
				// Alt+Enter for new line
				m.textarea.InsertString("\n")
			} else {
				// Send message
				input := strings.TrimSpace(m.textarea.Value())
				if input != "" && !m.isProcessing {
					m.sendMessage(input)
					m.textarea.Reset()
					m.textarea.Blur()
					cmds = append(cmds, m.processMessage())
				}
			}

		case tea.KeyCtrlL:
			// Clear screen
			m.messages = []Message{}
			m.updateViewport()
		}

	case streamMsg:
		// Handle streaming content
		if msg.done {
			m.isProcessing = false
			m.textarea.Focus()
		} else if msg.err != nil {
			m.err = msg.err
			m.isProcessing = false
			m.addSystemMessage(fmt.Sprintf("Error: %v", msg.err))
			m.textarea.Focus()
		} else {
			// Append content to last assistant message
			if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == types.RoleAssistant {
				m.messages[len(m.messages)-1].Content += msg.content
				m.updateViewport()
			}
		}

	case toolCallMsg:
		// Handle tool calls
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == types.RoleAssistant {
			m.messages[len(m.messages)-1].ToolCalls = append(
				m.messages[len(m.messages)-1].ToolCalls,
				msg.toolCall,
			)
			m.addSystemMessage(fmt.Sprintf("🔧 Tool: %s", msg.toolCall.Name))
		}

	case toolResultMsg:
		// Handle tool results
		m.addSystemMessage(fmt.Sprintf("✓ Result: %s", msg.result))
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

// View renders the TUI
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Build the view
	var sections []string

	// Title
	title := titleStyle.Render("🚀 gline - AI Programming Assistant")
	sections = append(sections, title)

	// Messages viewport
	sections = append(sections, m.viewport.View())

	// Input area
	sections = append(sections, m.textarea.View())

	// Status bar
	status := m.renderStatusBar()
	sections = append(sections, status)

	// Help text
	help := helpStyle.Render("enter: send • tab: toggle mode • ctrl+l: clear • ctrl+c: quit")
	sections = append(sections, help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// sendMessage adds a user message and prepares for response
func (m *Model) sendMessage(content string) {
	// Add user message
	m.messages = append(m.messages, Message{
		Role:      types.RoleUser,
		Content:   content,
		Timestamp: time.Now(),
	})

	// Add empty assistant message (will be filled by streaming)
	m.messages = append(m.messages, Message{
		Role:      types.RoleAssistant,
		Content:   "",
		Timestamp: time.Now(),
	})

	m.isProcessing = true
	m.updateViewport()
}

// addSystemMessage adds a system message
func (m *Model) addSystemMessage(content string) {
	m.messages = append(m.messages, Message{
		Role:      types.RoleSystem,
		Content:   content,
		Timestamp: time.Now(),
	})
	m.updateViewport()
}

// updateViewport refreshes the viewport content
func (m *Model) updateViewport() {
	var content strings.Builder

	for _, msg := range m.messages {
		switch msg.Role {
		case types.RoleUser:
			content.WriteString(userStyle.Render("You: "))
			content.WriteString(msg.Content)
			content.WriteString("\n\n")

		case types.RoleAssistant:
			if msg.Content != "" {
				content.WriteString(assistantStyle.Render("AI: "))
				content.WriteString(msg.Content)
				content.WriteString("\n")
			}
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					content.WriteString(toolStyle.Render(fmt.Sprintf("  🔧 %s\n", tc.Name)))
				}
			}
			content.WriteString("\n")

		case types.RoleSystem:
			content.WriteString(systemStyle.Render(msg.Content))
			content.WriteString("\n\n")
		}
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

// renderStatusBar renders the status bar
func (m *Model) renderStatusBar() string {
	modeStr := string(m.mode)
	if m.mode == agent.ModeAct {
		modeStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("ACT")
	} else {
		modeStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("PLAN")
	}

	provider := m.provider
	if provider == "" {
		provider = "-"
	}
	model := m.model
	if model == "" {
		model = "-"
	}

	status := fmt.Sprintf("[%s] Provider: %s | Model: %s", modeStr, provider, model)
	if m.isProcessing {
		status += " | ⏳ Processing..."
	}

	return statusBarStyle.Width(m.width).Render(status)
}

// processMessage handles the message processing with the agent
func (m *Model) processMessage() tea.Cmd {
	return func() tea.Msg {
		if m.agentInstance == nil {
			return streamMsg{err: fmt.Errorf("agent not initialized"), done: true}
		}

		// Get the last user message
		var lastUserMsg string
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Role == types.RoleUser {
				lastUserMsg = m.messages[i].Content
				break
			}
		}

		if lastUserMsg == "" {
			return streamMsg{err: fmt.Errorf("no user message found"), done: true}
		}

		// Run the agent
		err := m.agentInstance.Run(m.ctx, lastUserMsg)
		if err != nil {
			log.Errorf("Agent error: %v", err)
			return streamMsg{err: err, done: true}
		}

		return streamMsg{done: true}
	}
}

// streamMsg represents a streaming message update
type streamMsg struct {
	content string
	err     error
	done    bool
}

// toolCallMsg represents a tool call
type toolCallMsg struct {
	toolCall types.ToolCall
}

// toolResultMsg represents a tool result
type toolResultMsg struct {
	result string
}

// Run starts the TUI
func Run(agentInstance *agent.BaseAgent) error {
	p := tea.NewProgram(New(agentInstance), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
