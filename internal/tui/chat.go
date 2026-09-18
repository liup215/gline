package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// Role represents a message role.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// ChatMessage represents a single message in the chat.
type ChatMessage struct {
	Role    Role
	Content string

	// Tool-specific fields
	ToolID     string
	ToolName   string
	ToolInput  string
	ToolResult string
	ToolDone   bool

	// Streaming state
	Streaming bool
}

// chatModel manages the chat message list and viewport.
type chatModel struct {
	viewport viewport.Model
	messages []ChatMessage

	// Streaming state
	streamingContent strings.Builder
	streamingActive  bool

	width  int
	height int
}

func newChatModel(width, height int) chatModel {
	// Start with reasonable defaults; will be updated on WindowSizeMsg
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	vp := viewport.New(width, height)
	return chatModel{
		viewport: vp,
		messages: make([]ChatMessage, 0),
		width:    width,
		height:   height,
	}
}

func (m *chatModel) setWidthHeight(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
}

// appendMessage adds a message and scrolls to bottom.
func (m *chatModel) appendMessage(msg ChatMessage) {
	m.messages = append(m.messages, msg)
	m.updateContent()
	m.viewport.GotoBottom()
}

// startStreaming prepares for a new streaming response.
func (m *chatModel) startStreaming() {
	m.streamingContent.Reset()
	m.streamingActive = true
	// Add a placeholder assistant message
	m.messages = append(m.messages, ChatMessage{
		Role:      RoleAssistant,
		Streaming: true,
	})
	m.updateContent()
	m.viewport.GotoBottom()
}

// appendStreamContent adds content to the current streaming message.
func (m *chatModel) appendStreamContent(delta string) {
	m.streamingContent.WriteString(delta)
	// Update the last message
	if len(m.messages) > 0 {
		last := &m.messages[len(m.messages)-1]
		last.Content = m.streamingContent.String()
	}
	m.updateContent()
	m.viewport.GotoBottom()
}

// finishStreaming marks the streaming as complete.
func (m *chatModel) finishStreaming() {
	m.streamingActive = false
	if len(m.messages) > 0 {
		last := &m.messages[len(m.messages)-1]
		last.Streaming = false
		// Remove empty messages
		if last.Content == "" && last.ToolID == "" {
			m.messages = m.messages[:len(m.messages)-1]
		}
	}
	m.updateContent()
	m.viewport.GotoBottom()
}

// addToolStart adds a tool call start message.
func (m *chatModel) addToolStart(id, name, input string) {
	m.messages = append(m.messages, ChatMessage{
		Role:      RoleTool,
		ToolID:    id,
		ToolName:  name,
		ToolInput: input,
		ToolDone:  false,
	})
	m.updateContent()
	m.viewport.GotoBottom()
}

// setToolResult updates the tool result for a given tool call.
func (m *chatModel) setToolResult(id, result string) {
	for i := range m.messages {
		if m.messages[i].ToolID == id {
			m.messages[i].ToolResult = result
			m.messages[i].ToolDone = true
			break
		}
	}
	m.updateContent()
	m.viewport.GotoBottom()
}

// clearMessages clears all messages.
func (m *chatModel) clearMessages() {
	m.messages = m.messages[:0]
	m.streamingContent.Reset()
	m.streamingActive = false
	m.updateContent()
}

// updateContent rebuilds the viewport content from messages.
func (m *chatModel) updateContent() {
	var b strings.Builder
	for _, msg := range m.messages {
		b.WriteString(m.renderMessage(msg))
		b.WriteString("\n")
	}
	m.viewport.SetContent(b.String())
}

// renderMessage renders a single message.
func (m *chatModel) renderMessage(msg ChatMessage) string {
	switch msg.Role {
	case RoleUser:
		return m.renderUserMessage(msg)
	case RoleAssistant:
		return m.renderAssistantMessage(msg)
	case RoleSystem:
		return m.renderSystemMessage(msg)
	case RoleTool:
		return m.renderToolMessage(msg)
	default:
		return msg.Content
	}
}

func (m *chatModel) renderUserMessage(msg ChatMessage) string {
	var b strings.Builder
	b.WriteString(UserLabelStyle.Render("👤 You"))
	b.WriteString("\n")
	b.WriteString(msg.Content)
	return b.String()
}

func (m *chatModel) renderAssistantMessage(msg ChatMessage) string {
	var b strings.Builder
	b.WriteString(AssistantLabelStyle.Render("🤖 Assistant"))
	b.WriteString("\n")
	if msg.Content != "" {
		if msg.Streaming {
			// Don't render markdown while streaming (incomplete)
			b.WriteString(msg.Content)
			b.WriteString(" ▌")
		} else {
			// Render markdown for complete messages
			rendered := m.renderMarkdown(msg.Content)
			b.WriteString(rendered)
		}
	} else if msg.Streaming {
		b.WriteString(" ▌")
	}
	return b.String()
}

// renderMarkdown renders markdown content for terminal display.
func (m *chatModel) renderMarkdown(content string) string {
	// Use glamour for markdown rendering
	rendered, err := glamour.Render(content, "dark")
	if err != nil {
		// Fallback to plain text if rendering fails
		return content
	}
	// Remove trailing newline that glamour adds
	return strings.TrimRight(rendered, "\n")
}

func (m *chatModel) renderSystemMessage(msg ChatMessage) string {
	return SystemLabelStyle.Render("⚙ " + msg.Content)
}

func (m *chatModel) renderToolMessage(msg ChatMessage) string {
	var b strings.Builder

	// Tool icon and name with status
	icon := "⏳"
	if msg.ToolDone {
		icon = "✓"
	}

	// Parse tool input to show a summary
	summary := m.summarizeToolInput(msg.ToolName, msg.ToolInput)

	toolHeader := fmt.Sprintf("%s %s", icon, msg.ToolName)
	if summary != "" {
		toolHeader += ": " + summary
	}

	// Box style for tool calls
	boxWidth := m.width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	// Render the tool box
	b.WriteString(ToolBoxStyle.
		Width(boxWidth).
		Render(ToolLabelStyle.Render(toolHeader)))

	// Show result if done
	if msg.ToolDone && msg.ToolResult != "" {
		result := m.truncateResult(msg.ToolResult)
		b.WriteString("\n")
		resultStyle := lipgloss.NewStyle().
			Foreground(ColorMuted).
			PaddingLeft(2)
		b.WriteString(resultStyle.Render(result))
	} else if !msg.ToolDone {
		b.WriteString("\n")
		execStyle := lipgloss.NewStyle().
			Foreground(ColorDim).
			Italic(true).
			PaddingLeft(2)
		b.WriteString(execStyle.Render("executing..."))
	}

	return b.String()
}

// summarizeToolInput extracts a short summary from tool input JSON.
func (m *chatModel) summarizeToolInput(toolName, inputJSON string) string {
	if inputJSON == "" {
		return ""
	}

	var input map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return ""
	}

	switch toolName {
	case "read_file":
		if path, ok := input["path"].(string); ok {
			return truncatePath(path)
		}
	case "write_to_file":
		if path, ok := input["path"].(string); ok {
			return truncatePath(path)
		}
	case "replace_in_file":
		if path, ok := input["path"].(string); ok {
			return truncatePath(path)
		}
	case "search_files":
		if query, ok := input["query"].(string); ok {
			return fmt.Sprintf("%q", truncateStr(query, 30))
		}
	case "list_code_definition_names":
		if path, ok := input["path"].(string); ok {
			return truncatePath(path)
		}
	case "execute_command":
		if cmd, ok := input["command"].(string); ok {
			return fmt.Sprintf("%s", truncateStr(cmd, 40))
		}
	case "attempt_completion":
		if result, ok := input["result"].(string); ok {
			return truncateStr(result, 40)
		}
	case "ask_followup_question":
		if q, ok := input["question"].(string); ok {
			return truncateStr(q, 40)
		}
	case "plan_mode_respond":
		if resp, ok := input["response"].(string); ok {
			return truncateStr(resp, 40)
		}
	case "use_skill":
		if name, ok := input["skill_name"].(string); ok {
			return name
		}
	case "summarize_file":
		if path, ok := input["path"].(string); ok {
			return truncatePath(path)
		}
	}

	return ""
}

// truncateResult truncates a tool result for display.
func (m *chatModel) truncateResult(result string) string {
	// Remove leading/trailing whitespace
	result = strings.TrimSpace(result)

	lines := strings.Split(result, "\n")

	// For short results, show as-is
	if len(lines) <= 3 && len(result) <= 200 {
		return result
	}

	// For longer results, show first few lines + count
	var b strings.Builder
	maxLines := 4
	for i := 0; i < maxLines && i < len(lines); i++ {
		b.WriteString(lines[i])
		if i < maxLines-1 && i < len(lines)-1 {
			b.WriteString("\n")
		}
	}

	if len(lines) > maxLines {
		b.WriteString(fmt.Sprintf("\n... (%d more lines)", len(lines)-maxLines))
	}

	return b.String()
}

// truncatePath shortens a file path for display.
func truncatePath(path string) string {
	if len(path) <= 40 {
		return path
	}
	return "..." + path[len(path)-37:]
}

// truncateStr shortens a string to maxLen.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Update handles viewport updates.
func (m *chatModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

// View renders the viewport.
func (m *chatModel) View() string {
	return m.viewport.View()
}
