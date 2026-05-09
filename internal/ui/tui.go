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
	Timestamp time.Time
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

	toolCompletedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00AA00"))

	toolFailedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF4444"))

	statusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	streamingIndicatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Bold(true)

toolAreaBorderStyle = lipgloss.NewStyle().
Foreground(lipgloss.Color("#555555"))

inputBoxStyle = lipgloss.NewStyle().
Border(lipgloss.RoundedBorder()).
BorderForeground(lipgloss.Color("#666666")).
Padding(0, 3).
MarginTop(0)

inputTitleStyle = lipgloss.NewStyle().
Foreground(lipgloss.Color("#AAAAAA")).
Italic(true)
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
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		// Reserve space for: title (1) + tool area (toolAreaHeight) + input (inputHeight) + status bar (1) + help (1)
		m.viewport.Height = msg.Height - m.inputHeight - m.toolAreaHeight - 4
		if m.viewport.Height < 3 {
			m.viewport.Height = 3
		}
 // Compute inner width available for textarea content (subtract border + horizontal padding and left margin).
 // inputBoxStyle has Padding(0, 3) which gives 3 cols on left and right, border (2 cols), plus we render with a left margin of 1.
 innerWidth := msg.Width - 9
 if innerWidth < 10 {
 innerWidth = 10
 }
 m.textarea.SetWidth(innerWidth)
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
			m.toolHistory = nil
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
	// Append content to the active assistant slot.
	// Ensure an assistant slot exists; create one if not present (e.g., tool status arrived first).
	if m.activeAssistantIndex < 0 || m.activeAssistantIndex >= len(m.messages) || m.messages[m.activeAssistantIndex].Role != types.RoleAssistant {
		// Create a new assistant message slot and set it active.
		m.messages = append(m.messages, Message{
			Role:      types.RoleAssistant,
			Content:   "",
			Timestamp: time.Now(),
		})
		m.activeAssistantIndex = len(m.messages) - 1
	}
	m.messages[m.activeAssistantIndex].Content += msg.content
	m.updateViewport()

		case "toolStart":
			m.currentTool = msg.toolName
			// Add to tool history instead of system messages
			m.toolHistory = append(m.toolHistory, ToolStatus{
				Name:      msg.toolName,
				Status:    "running",
				StartTime: time.Now(),
			})
			m.updateViewport()

		case "toolComplete":
			// Update tool history entry status
			result := msg.toolResult
			newStatus := "completed"
			if strings.Contains(result, "Original input:") {
				newStatus = "failed"
			}
			// Find the last entry for this tool name that is still "running"
			for i := len(m.toolHistory) - 1; i >= 0; i-- {
				if m.toolHistory[i].Name == msg.toolName && m.toolHistory[i].Status == "running" {
					m.toolHistory[i].Status = newStatus
					break
				}
			}
			m.currentTool = ""
			m.updateViewport()

case "error":
m.err = msg.err
m.isProcessing = false
m.isStreaming = false
m.addErrorMessage(fmt.Sprintf("Error: %v", msg.err))
m.textarea.Focus()
cmds = append(cmds, textarea.Blink)
m.updateViewport()

case "complete":
m.isProcessing = false
m.isStreaming = false
m.currentTool = ""
m.textarea.Focus()
cmds = append(cmds, textarea.Blink)
m.updateViewport()

		case "streamStart":
			m.isStreaming = true
			// Create a new assistant message slot for the new stream round
			m.messages = append(m.messages, Message{
				Role:      types.RoleAssistant,
				Content:   "",
				Timestamp: time.Now(),
			})
			m.activeAssistantIndex = len(m.messages) - 1
			m.updateViewport()

		case "streamEnd":
			m.isStreaming = false
			m.updateViewport()
		}

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
	help := helpStyle.Render("enter: send • tab: toggle mode • ctrl+l: clear • ctrl+c: quit")
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

 // updateViewport refreshes the viewport content
func (m *Model) updateViewport() {
var content strings.Builder

for i, msg := range m.messages {
switch msg.Role {
case types.RoleUser:
content.WriteString(userStyle.Render("You: "))
content.WriteString(msg.Content)
content.WriteString("\n")
content.WriteString(systemStyle.Render(msg.Timestamp.Format("15:04")))
content.WriteString("\n\n")

case types.RoleAssistant:
// Render assistant content as markdown (glamour) when possible
rendered := msg.Content
if msg.Content != "" {
if out, err := glamour.Render(msg.Content, "dark"); err == nil {
rendered = out
}
}

// Append streaming cursor if this is the active streaming assistant message
if m.isStreaming && i == m.activeAssistantIndex {
rendered = rendered + streamingIndicatorStyle.Render(" ▌")
}

// Include tool calls if present
if len(msg.ToolCalls) > 0 {
var tools strings.Builder
for _, tc := range msg.ToolCalls {
tools.WriteString(toolStyle.Render(fmt.Sprintf("\n  🔧 %s", tc.Name)))
}
rendered = rendered + tools.String()
}

content.WriteString(assistantStyle.Render("AI: "))
content.WriteString("\n")
content.WriteString(rendered)
content.WriteString("\n")
content.WriteString(systemStyle.Render(msg.Timestamp.Format("15:04")))
content.WriteString("\n\n")

case types.RoleSystem:
// Only render non-tool system messages (e.g., errors)
if strings.HasPrefix(msg.Content, "Error:") || strings.HasPrefix(msg.Content, "✗") {
content.WriteString(errorStyle.Render(msg.Content))
content.WriteString("\n\n")
}
}
}

m.viewport.SetContent(content.String())
m.viewport.GotoBottom()
}

// renderToolArea renders the tool status area below the viewport
func (m *Model) renderToolArea() string {
	if len(m.toolHistory) == 0 {
		// Show empty border line when no tools are active
		return toolAreaBorderStyle.Render(strings.Repeat("─", m.width))
	}

 // Determine how many tool entries to show (limited by toolAreaHeight)
maxEntries := m.toolAreaHeight
if maxEntries < 1 {
maxEntries = 1
}

	var lines []string

	// Show the most recent tool entries
	start := 0
	if len(m.toolHistory) > maxEntries {
		start = len(m.toolHistory) - maxEntries
	}

	for i := start; i < len(m.toolHistory); i++ {
		ts := m.toolHistory[i]
		switch ts.Status {
		case "running":
			lines = append(lines, toolRunningStyle.Render(fmt.Sprintf("  🔧 %s ⏳", ts.Name)))
		case "completed":
			lines = append(lines, toolCompletedStyle.Render(fmt.Sprintf("  🔧 %s ✓", ts.Name)))
		case "failed":
			lines = append(lines, toolFailedStyle.Render(fmt.Sprintf("  🔧 %s ✗", ts.Name)))
		}
	}

	// Top border
	border := toolAreaBorderStyle.Render(strings.Repeat("─", m.width))

	// Combine border and tool lines
	var allLines []string
	allLines = append(allLines, border)
	allLines = append(allLines, lines...)

	return lipgloss.JoinVertical(lipgloss.Left, allLines...)
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

func (c *tuiCallback) OnStreamStart() {
	if c.program != nil {
		c.program.Send(agentUpdateMsg{updateType: "streamStart"})
	}
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

	// RunWithCallback will invoke OnComplete via the callback; avoid sending a duplicate complete message.
	return nil
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