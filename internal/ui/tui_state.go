package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textarea"

	"github.com/liup215/gline/internal/ui/bridge"
	"github.com/liup215/gline/internal/ui/model"
	"github.com/liup215/gline/pkg/types"
)

// State mutation helpers extracted from handleAgentUpdate to isolate model changes.
// Each function mirrors one branch from the previous switch in internal/ui/tui_update.go.

// handleAgentContent appends streaming content to the active assistant message.
func handleAgentContent(m *Model, msg bridge.ContentEvent) []tea.Cmd {
	msgs := m.conversation.Messages
	// Ensure an assistant slot exists; create one if not present (e.g., tool status arrived first).
	if m.activeAssistantIndex < 0 || m.activeAssistantIndex >= len(msgs) || msgs[m.activeAssistantIndex].Role != types.RoleAssistant {
		// Create a new assistant message slot and set it active.
		m.activeAssistantIndex = m.conversation.AppendMessage(model.Message{
			Role:      types.RoleAssistant,
			Content:   "",
			Timestamp: time.Now(),
		})
	}
	m.conversation.UpdateMessageContent(m.activeAssistantIndex, msg.Delta)
	m.updateViewport()
	return nil
}

// handleAgentToolStart handles a tool start update: record history and surface a short system message.
func handleAgentToolStart(m *Model, msg bridge.ToolStartEvent) []tea.Cmd {
	m.currentTool = msg.Name
	// Add to tool history
	m.conversation.AddToolStart(msg.Name)

	// Prepare a compact human-friendly display for the tool start.
	desc := getToolDescription(msg.Name)
	display := ""
	if msg.Input != "" {
		// attempt_completion often carries the final summary/result; keep it intact
		if normalizeToolName(msg.Name) == "attempt_completion" {
			display = fmt.Sprintf("🔧 %s\n\n%s", desc, msg.Input)
		} else {
			// Try to show the single most relevant argument (path, command, url, etc.)
			if main := getToolMainArg(msg.Name, msg.Input); main != "" {
				display = fmt.Sprintf("🔧 %s: %s", desc, main)
			} else {
				var buf bytes.Buffer
				if err := json.Indent(&buf, []byte(msg.Input), "  ", "  "); err == nil {
					display = fmt.Sprintf("🔧 %s\n  Input:\n%s", desc, buf.String())
				} else {
					display = fmt.Sprintf("🔧 %s\n  Input: %s", desc, msg.Input)
				}
			}
		}
	} else {
		display = fmt.Sprintf("🔧 %s", desc)
	}

	var assistantContent string
	if normalizeToolName(msg.Name) == "attempt_completion" {
		// Parse JSON Input and extract a human-friendly result when possible.
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Input), &parsed); err == nil {
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
					assistantContent = msg.Input
				}
			} else {
				// Fallback: pretty-print the whole parsed JSON as a JSON code block
				if pretty, err2 := json.MarshalIndent(parsed, "", "  "); err2 == nil {
					assistantContent = "```json\n" + string(pretty) + "\n```"
				} else {
					assistantContent = msg.Input
				}
			}
		} else {
			assistantContent = msg.Input
		}

		m.conversation.AppendMessage(model.Message{
			Role:      types.RoleAssistant,
			Content:   assistantContent,
			Timestamp: time.Now(),
		})
	} else if normalizeToolName(msg.Name) == "ask_followup_question" {
		// Skip adding a system message here; askQuestionMsg will display the question.
	} else if normalizeToolName(msg.Name) == "plan_mode_respond" {
		// Skip: the completed result will be rendered as a full assistant message in toolComplete.
	} else {
		m.conversation.AppendMessage(model.Message{
			Role:      types.RoleSystem,
			Content:   display,
			Timestamp: time.Now(),
		})
	}
	m.updateViewport()
	return nil
}

// handleAgentToolComplete updates tool history and optionally appends a summary/system message or assistant message.
func handleAgentToolComplete(m *Model, msg bridge.ToolCompleteEvent) []tea.Cmd {
	result := msg.Result
	newStatus := "completed"
	m.conversation.MarkToolComplete(msg.Name)
	m.currentTool = ""

	// Short-circuit cases handled elsewhere
	if normalizeToolName(msg.Name) == "attempt_completion" || normalizeToolName(msg.Name) == "ask_followup_question" {
		m.updateViewport()
		return nil
	}

	if normalizeToolName(msg.Name) == "plan_mode_respond" {
		// Render the plan response as an assistant message (full markdown, no truncation)
		if result != "" {
			m.conversation.AppendMessage(model.Message{
				Role:      types.RoleAssistant,
				Content:   result,
				Timestamp: time.Now(),
			})
		}
		m.updateViewport()
		return nil
	}

	// Append system message for conversation visibility with a short result summary
	statusText := "Completed"
	if newStatus == "failed" {
		statusText = "Failed"
	}
	content := fmt.Sprintf("🔧 %s: %s", statusText, msg.Name)
	if result != "" {
		lines := formatToolResultLines(result, 5)
		content += "\n"
		for _, l := range lines {
			content += l + "\n"
		}
	}
	m.conversation.AppendMessage(model.Message{
		Role:      types.RoleSystem,
		Content:   content,
		Timestamp: time.Now(),
	})
	m.updateViewport()
	return nil
}

// handleAgentError marks running tools as failed, surfaces an error message, and focuses the textarea.
func handleAgentError(m *Model, msg bridge.ErrorEvent) []tea.Cmd {
	var cmds []tea.Cmd
	m.err = msg.Err
	m.isProcessing = false
	m.isStreaming = false
	// If an error occurred during a tool run, mark the most recent running tool as failed for visibility.
	for i := len(m.conversation.ToolHistory) - 1; i >= 0; i-- {
		if m.conversation.ToolHistory[i].Status == "running" {
			m.conversation.ToolHistory[i].Status = "failed"
			// Append a short system message to make the failure obvious in the conversation
			m.conversation.AppendMessage(model.Message{
				Role:      types.RoleSystem,
				Content:   fmt.Sprintf("🔧 Failed: %s", m.conversation.ToolHistory[i].Name),
				Timestamp: time.Now(),
			})
			break
		}
	}
	if m.cancelFn != nil {
		m.cancelFn = nil
	}
	m.addErrorMessage(fmt.Sprintf("Error: %v", msg.Err))
	m.textarea.Focus()
	cmds = append(cmds, textarea.Blink)
	m.updateViewport()
	return cmds
}

// handleAgentComplete finalizes streaming/processing state and focuses textarea.
func handleAgentComplete(m *Model, msg bridge.CompleteEvent) []tea.Cmd {
	var cmds []tea.Cmd
	m.isProcessing = false
	m.isStreaming = false
	m.currentTool = ""
	if m.cancelFn != nil {
		m.cancelFn = nil
	}
	m.textarea.Focus()
	cmds = append(cmds, textarea.Blink)
	m.updateViewport()
	return cmds
}

// handleAgentStreamStart creates a new assistant message slot for streaming.
func handleAgentStreamStart(m *Model, msg bridge.StreamStartEvent) []tea.Cmd {
	m.isStreaming = true
	// Create a new assistant message slot for the new stream round
	m.activeAssistantIndex = m.conversation.AppendMessage(model.Message{
		Role:      types.RoleAssistant,
		Content:   "",
		Timestamp: time.Now(),
	})
	m.updateViewport()
	return nil
}

// handleAgentStreamEnd ends streaming.
func handleAgentStreamEnd(m *Model, msg bridge.StreamEndEvent) []tea.Cmd {
	m.isStreaming = false
	m.updateViewport()
	return nil
}
