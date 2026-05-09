// Package ui provides the TUI (Terminal User Interface) for gline
package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/agent"
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
	spinner  spinner.Model

	// State
	messages     []Message
	mode         agent.Mode
	provider     string
	model        string
	inputHeight  int
	isProcessing bool
	isStreaming  bool
	err          error
	currentTool  string
	activeAssistantIndex int

	// Agent components
	agentInstance *agent.BaseAgent
	ctx           context.Context

	// Program reference for sending messages
	program *tea.Program

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
		Foreground(lipgloss.Color("#FF0000")).
		Bold(true)

	toolStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFA500"))

	toolRunningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")).
		Bold(true)

	statusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	streamingIndicatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Bold(true)
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
		textarea:      ta,
		viewport:      vp,
		spinner:       s,
		messages:      []Message{},
		mode:          agent.ModeAct,
		provider:      providerName,
		model:         modelName,
		inputHeight:   3,
		activeAssistantIndex: -1,
		agentInstance: agentInstance,
		ctx:           context.Background(),
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
					// Start the agent with callback
					cmds = append(cmds, m.startAgent())
				}
			}

		case tea.KeyCtrlL:
			// Clear screen
			m.messages = []Message{}
			m.updateViewport()
		}

	case spinner.TickMsg:
		// Update spinner animation
		if m.isProcessing {
			newSpinner, cmd := m.spinner.Update(msg)
			m.spinner = newSpinner
			cmds = append(cmds, cmd)
		}

	case agentUpdateMsg:
		// Handle agent callback updates
		switch msg.updateType {
		case "content":
			// Append content to the active assistant slot even if tool/system messages
			// were appended after streaming started.
			if m.activeAssistantIndex >= 0 && m.activeAssistantIndex < len(m.messages) && m.messages[m.activeAssistantIndex].Role == types.RoleAssistant {
				m.messages[m.activeAssistantIndex].Content += msg.content
				m.updateViewport()
			}

		case "toolStart":
			m.currentTool = msg.toolName
			m.addSystemMessage(fmt.Sprintf("🔧 Running: %s", msg.toolName))
			m.updateViewport()

		case "toolComplete":
			// Show tool completion with result
			result := msg.toolResult
			// Check if this is a JSON parsing error (contains "Original input:")
			if strings.Contains(result, "Original input:") {
				// Extract and show the original input for debugging
				parts := strings.Split(result, "Original input:")
				if len(parts) > 1 {
					originalInput := strings.TrimSpace(parts[1])
					m.addSystemMessage(fmt.Sprintf("🔧 Tool Call Failed: %s\n❌ JSON Parse Error\n📄 Original Input: %s", msg.toolName, originalInput))
				} else {
					m.addSystemMessage(fmt.Sprintf("🔧 Tool Call Failed: %s\n❌ %s", msg.toolName, result))
				}
			} else {
				// Normal tool result
				if len(result) > 200 {
					result = result[:200] + "..."
				}
				m.addSystemMessage(fmt.Sprintf("🔧 Completed: %s\nResult: %s", msg.toolName, result))
			}
			m.currentTool = ""
			m.updateViewport()

		case "error":
			m.err = msg.err
			m.isProcessing = false
			m.isStreaming = false
			m.activeAssistantIndex = -1
			m.addErrorMessage(fmt.Sprintf("Error: %v", msg.err))
			m.textarea.Focus()

		case "complete":
			m.isProcessing = false
			m.isStreaming = false
			m.currentTool = ""
			m.activeAssistantIndex = -1
			m.textarea.Focus()
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
	m.activeAssistantIndex = len(m.messages) - 1

	m.isProcessing = true
	m.isStreaming = true
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

// addErrorMessage adds an error message
func (m *Model) addErrorMessage(content string) {
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
			hasContent := msg.Content != "" || len(msg.ToolCalls) > 0

			if hasContent {
				content.WriteString(assistantStyle.Render("AI: "))
				if msg.Content != "" {
					content.WriteString(msg.Content)
					content.WriteString("\n")
				}

				// Show completed tool calls
				if len(msg.ToolCalls) > 0 {
					for _, tc := range msg.ToolCalls {
						content.WriteString(toolStyle.Render(fmt.Sprintf("  🔧 %s\n", tc.Name)))
					}
				}

				content.WriteString("\n")
			}

		case types.RoleSystem:
			// Check if it's an error message
			if strings.HasPrefix(msg.Content, "Error:") || strings.HasPrefix(msg.Content, "✗") {
				content.WriteString(errorStyle.Render(msg.Content))
			} else if strings.HasPrefix(msg.Content, "🔧") {
				content.WriteString(toolRunningStyle.Render(msg.Content))
			} else {
				content.WriteString(systemStyle.Render(msg.Content))
			}
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
		if m.isStreaming {
			status += fmt.Sprintf(" | %s AI is responding...", m.spinner.View())
		} else if m.currentTool != "" {
			status += fmt.Sprintf(" | %s Running: %s", m.spinner.View(), m.currentTool)
		} else {
			status += fmt.Sprintf(" | %s Processing...", m.spinner.View())
		}
	}

	return statusBarStyle.Width(m.width).Render(status)
}

// agentUpdateMsg represents an update from the agent callback
type agentUpdateMsg struct {
	updateType string
	content    string
	toolName   string
	toolResult string
	err        error
}

// tuiCallback implements the agent.StreamCallback interface
type tuiCallback struct {
	program *tea.Program
}

func (c *tuiCallback) OnContent(delta string) {
	if c.program != nil {
		c.program.Send(agentUpdateMsg{updateType: "content", content: delta})
	}
}

func (c *tuiCallback) OnToolCallStart(toolCall agent.ToolCall) {
	if c.program != nil {
		c.program.Send(agentUpdateMsg{updateType: "toolStart", toolName: toolCall.Name})
	}
}

func (c *tuiCallback) OnToolCallComplete(toolCall agent.ToolCall, result string) {
	if c.program != nil {
		c.program.Send(agentUpdateMsg{updateType: "toolComplete", toolName: toolCall.Name, toolResult: result})
	}
}

func (c *tuiCallback) OnError(err error) {
	if c.program != nil {
		c.program.Send(agentUpdateMsg{updateType: "error", err: err})
	}
}

func (c *tuiCallback) OnComplete() {
	if c.program != nil {
		c.program.Send(agentUpdateMsg{updateType: "complete"})
	}
}

// startAgent starts the agent with the TUI callback
func (m *Model) startAgent() tea.Cmd {
	return func() tea.Msg {
		if m.agentInstance == nil {
			return agentUpdateMsg{updateType: "error", err: fmt.Errorf("agent not initialized")}
		}

		// Get the last user message
		var lastUserMessage string
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Role == types.RoleUser {
				lastUserMessage = m.messages[i].Content
				break
			}
		}

		if lastUserMessage == "" {
			return agentUpdateMsg{updateType: "error", err: fmt.Errorf("no user message found")}
		}

		// Create callback with program reference
		callback := &tuiCallback{program: m.program}

		// Run the agent with callback
		err := m.agentInstance.RunWithCallback(m.ctx, lastUserMessage, callback)
		if err != nil {
			return agentUpdateMsg{updateType: "error", err: err}
		}

		return agentUpdateMsg{updateType: "complete"}
	}
}

// Run starts the TUI
func Run(agentInstance *agent.BaseAgent) error {
	model := New(agentInstance)
	p := tea.NewProgram(model, tea.WithAltScreen())
	model.SetProgram(p)
	_, err := p.Run()
	return err
}
