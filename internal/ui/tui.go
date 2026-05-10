// Package ui provides the TUI (Terminal User Interface) for gline
package ui

import (
"context"
"bytes"
"encoding/json"
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
case tea.KeyCtrlC:
return m, tea.Quit

case tea.KeyEsc:
if m.isProcessing {
if m.cancelFn != nil {
m.cancelFn()
}
// Notify user of interruption
m.addErrorMessage("✗ Interrupted by user (Esc)")
// Ensure processing flags updated; agent callback will also handle cleanup
m.isProcessing = false
m.isStreaming = false
m.textarea.Focus()
cmds = append(cmds, textarea.Blink)
m.updateViewport()
} else {
m.textarea.Reset()
m.updateViewport()
}

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
// If the UI is awaiting a reply for AskFollowupQuestion, deliver it instead of starting the agent.
if m.pendingReply != nil {
answer := strings.TrimSpace(m.textarea.Value())
if answer != "" {
// Send the user's answer back to the waiting tool goroutine.
m.pendingReply <- answer
}
// Clear pending state and reset input box without starting a new agent run.
m.pendingReply = nil
m.textarea.Reset()
m.textarea.Placeholder = "Type your message..."
m.textarea.Focus()
cmds = append(cmds, textarea.Blink)
m.updateViewport()
} else {
// Normal send-message behavior
input := strings.TrimSpace(m.textarea.Value())
if input != "" && !m.isProcessing {
m.sendMessage(input)
m.textarea.Reset()
m.textarea.Blur()
// Start the agent with callback
cmds = append(cmds, m.startAgent())
}
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
// Add to tool history
m.toolHistory = append(m.toolHistory, ToolStatus{
Name:      msg.toolName,
Status:    "running",
StartTime: time.Now(),
})

// Prepare a compact human-friendly display for the tool start.
// For attempt_completion we show the full content (rendered later for markdown).
desc := getToolDescription(msg.toolName)
display := ""
if msg.toolInput != "" {
	// attempt_completion often carries the final summary/result; keep it intact
	if normalizeToolName(msg.toolName) == "attempt_completion" {
		display = fmt.Sprintf("🔧 %s\n\n%s", desc, msg.toolInput)
	} else {
		// Try to show the single most relevant argument (path, command, url, etc.)
		if main := getToolMainArg(msg.toolName, msg.toolInput); main != "" {
			display = fmt.Sprintf("🔧 %s: %s", desc, main)
		} else {
			var buf bytes.Buffer
			if err := json.Indent(&buf, []byte(msg.toolInput), "  ", "  "); err == nil {
				display = fmt.Sprintf("🔧 %s\n  Input:\n%s", desc, buf.String())
			} else {
				display = fmt.Sprintf("🔧 %s\n  Input: %s", desc, msg.toolInput)
			}
		}
	}
} else {
	display = fmt.Sprintf("🔧 %s", desc)
}

if normalizeToolName(msg.toolName) == "attempt_completion" {
    // Parse JSON toolInput and extract a human-friendly result when possible.
    var assistantContent string
    var parsed map[string]interface{}
    if err := json.Unmarshal([]byte(msg.toolInput), &parsed); err == nil {
        // Prefer result as non-empty string
        if r, ok := parsed["result"].(string); ok && strings.TrimSpace(r) != "" {
            assistantContent = r
        } else if c, ok := parsed["content"].(string); ok && strings.TrimSpace(c) != "" {
            assistantContent = c
        } else if mres, ok := parsed["result"].(map[string]interface{}); ok {
            // If result is an object, pretty-print it and render as a JSON code block
            if pretty, err2 := json.MarshalIndent(mres, "", "  "); err2 == nil {
                assistantContent = "```json\n" + string(pretty) + "\n```"
            } else {
                assistantContent = msg.toolInput
            }
        } else {
            // Fallback: pretty-print the whole parsed JSON as a JSON code block
            if pretty, err2 := json.MarshalIndent(parsed, "", "  "); err2 == nil {
                assistantContent = "```json\n" + string(pretty) + "\n```"
            } else {
                assistantContent = msg.toolInput
            }
        }
    } else {
        assistantContent = msg.toolInput
    }

    m.messages = append(m.messages, Message{
        Role:      types.RoleAssistant,
        Content:   assistantContent,
        Timestamp: time.Now(),
    })
} else if normalizeToolName(msg.toolName) == "ask_followup_question" {
    // Skip adding a system message here; the askQuestionMsg handler (triggered by
    // the AskFollowupQuestion callback) will display the question with styled options.
} else if normalizeToolName(msg.toolName) == "plan_mode_respond" {
    // Skip: the completed result will be rendered as a full assistant message (markdown)
    // in the toolComplete handler, so no need to show the Input here.
} else {
    m.messages = append(m.messages, Message{
        Role:      types.RoleSystem,
        Content:   display,
        Timestamp: time.Now(),
    })
}
m.updateViewport()

case "toolComplete":
 // Update tool history entry status
 result := msg.toolResult
 newStatus := "completed"
 // Find the last entry for this tool name that is still "running"
 for i := len(m.toolHistory) - 1; i >= 0; i-- {
     if m.toolHistory[i].Name == msg.toolName && m.toolHistory[i].Status == "running" {
         m.toolHistory[i].Status = newStatus
         break
     }
 }
m.currentTool = ""

// For attempt_completion we avoid adding a duplicate small system line because the full result
// was already added on toolStart for clearer presentation.
// For ask_followup_question we also skip — the question+options are already displayed by
// the askQuestionMsg handler, and the answer is visible from user input.
// For plan_mode_respond we render the result as a full assistant message with markdown.
if normalizeToolName(msg.toolName) == "attempt_completion" || normalizeToolName(msg.toolName) == "ask_followup_question" {
m.updateViewport()
break
}

if normalizeToolName(msg.toolName) == "plan_mode_respond" {
    // Render the plan response as an assistant message (full markdown, no truncation)
    if result != "" {
        m.messages = append(m.messages, Message{
            Role:      types.RoleAssistant,
            Content:   result,
            Timestamp: time.Now(),
        })
    }
    m.updateViewport()
    break
}

// Append system message for conversation visibility with a short result summary
statusText := "Completed"
if newStatus == "failed" {
statusText = "Failed"
}
content := fmt.Sprintf("🔧 %s: %s", statusText, msg.toolName)
if result != "" {
	lines := formatToolResultLines(result, 5)
	content += "\n"
	for _, l := range lines {
		content += l + "\n"
	}
}
m.messages = append(m.messages, Message{
Role:      types.RoleSystem,
Content:   content,
Timestamp: time.Now(),
})
m.updateViewport()

case "error":
    m.err = msg.err
    m.isProcessing = false
    m.isStreaming = false
    // If an error occurred during a tool run, mark the most recent running tool as failed for visibility.
    for i := len(m.toolHistory) - 1; i >= 0; i-- {
        if m.toolHistory[i].Status == "running" {
            m.toolHistory[i].Status = "failed"
            // Append a short system message to make the failure obvious in the conversation
            m.messages = append(m.messages, Message{
                Role:      types.RoleSystem,
                Content:   fmt.Sprintf("🔧 Failed: %s", m.toolHistory[i].Name),
                Timestamp: time.Now(),
            })
            break
        }
    }
    if m.cancelFn != nil {
        m.cancelFn = nil
    }
    m.addErrorMessage(fmt.Sprintf("Error: %v", msg.err))
    m.textarea.Focus()
    cmds = append(cmds, textarea.Blink)
    m.updateViewport()

case "complete":
m.isProcessing = false
m.isStreaming = false
m.currentTool = ""
if m.cancelFn != nil {
m.cancelFn = nil
}
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

 // updateViewport refreshes the viewport content
func (m *Model) updateViewport() {
var content strings.Builder
pad := 3

for i, msg := range m.messages {
switch msg.Role {
case types.RoleUser:
content.WriteString(userStyle.Render("You: "))
content.WriteString(msg.Content)
content.WriteString("\n")
content.WriteString(systemStyle.Render(msg.Timestamp.Format("15:04")))
content.WriteString("\n\n")

case types.RoleAssistant:
		// Render assistant content as markdown (glamour) with word-wrap matching viewport width
		rendered := msg.Content
		if msg.Content != "" {
			// compute available wrap width for Glamour (subtract left/right padding)
			wrapWidth := m.viewport.Width - pad*2
			if wrapWidth < 20 {
				wrapWidth = 20
			}

			// Reuse cached rendered output when possible
			if msg.Rendered != "" && msg.RenderedSource == msg.Content && msg.RenderedWrapWidth == wrapWidth {
				rendered = msg.Rendered
			} else {
				// Ensure we have a renderer for this wrapWidth cached on the model.
				var r *glamour.TermRenderer
				var err error
				if m.renderer != nil && m.rendererWrapWidth == wrapWidth {
					r = m.renderer
				} else {
					if r, err = glamour.NewTermRenderer(glamour.WithWordWrap(wrapWidth)); err == nil {
						m.renderer = r
						m.rendererWrapWidth = wrapWidth
					} else {
						// failed to construct; fall back to default renderer later
						m.renderer = nil
						m.rendererWrapWidth = 0
					}
				}

				if r != nil {
					if out, err2 := r.Render(msg.Content); err2 == nil {
						rendered = out
					} else {
						// fallback to default renderer
						if out2, err3 := glamour.Render(msg.Content, "dark"); err3 == nil {
							rendered = out2
						}
					}
				} else {
					if out, err := glamour.Render(msg.Content, "dark"); err == nil {
						rendered = out
					}
				}

				// Update message cache
				msg.Rendered = rendered
				msg.RenderedWrapWidth = wrapWidth
				msg.RenderedSource = msg.Content
				m.messages[i] = msg
			}

			// Add horizontal padding to the rendered block
			rendered = lipgloss.NewStyle().Padding(0, pad).Render(rendered)
		}

// Append streaming cursor if this is the active streaming assistant message
if m.isStreaming && i == m.activeAssistantIndex {
rendered = rendered + streamingIndicatorStyle.Render(" ▌")
}

// Include tool calls if present
if len(msg.ToolCalls) > 0 {
var tools strings.Builder
for _, tc := range msg.ToolCalls {
line := fmt.Sprintf("\n  🔧 %s", tc.Name)
// include input if present (pretty-print JSON when possible)
if len(tc.Input) > 0 {
var buf bytes.Buffer
if err := json.Indent(&buf, tc.Input, "    ", "  "); err == nil {
line += "\n    Input:\n" + buf.String()
} else {
line += "\n    Input: " + string(tc.Input)
}
}
tools.WriteString(toolStyle.Render(line))
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
 // Render errors and tool messages
 if strings.HasPrefix(msg.Content, "Error:") || strings.HasPrefix(msg.Content, "✗") {
 content.WriteString(errorStyle.Render(msg.Content))
 content.WriteString("\n\n")
 } else if strings.HasPrefix(msg.Content, "❓") || len(msg.Options) > 0 {
 // AskFollowupQuestion: render question with styled options
 content.WriteString(questionIconStyle.Render("❓ "))
 content.WriteString(questionStyle.Render(strings.TrimPrefix(msg.Content, "❓ ")))
 content.WriteString("\n")
 if len(msg.Options) > 0 {
 for i, opt := range msg.Options {
 num := optionNumStyle.Render(fmt.Sprintf("%d.", i+1))
 content.WriteString(optionStyle.Render(fmt.Sprintf("%s %s", num, opt)))
 content.WriteString("\n")
 }
 content.WriteString(optionHintStyle.Render("Enter option number or type your answer"))
 content.WriteString("\n")
 }
 content.WriteString("\n")
 } else if strings.HasPrefix(msg.Content, "🔧") {
 // Tool messages: style based on keywords
 if strings.Contains(msg.Content, "Running") || strings.Contains(msg.Content, "running") || strings.Contains(msg.Content, "started") {
 content.WriteString(toolRunningStyle.Render(msg.Content))
 } else if strings.Contains(msg.Content, "Completed") || strings.Contains(msg.Content, "✓") || strings.Contains(msg.Content, "completed") {
 content.WriteString(toolCompletedStyle.Render(msg.Content))
 } else if strings.Contains(msg.Content, "Failed") || strings.Contains(msg.Content, "✗") || strings.Contains(msg.Content, "failed") {
 content.WriteString(toolFailedStyle.Render(msg.Content))
 } else {
 content.WriteString(systemStyle.Render(msg.Content))
 }
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