package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textarea"

	"github.com/liup215/gline/internal/ui/bridge"
	"github.com/liup215/gline/internal/ui/model"
	"github.com/liup215/gline/internal/ui/view"
	"github.com/liup215/gline/pkg/types"
)

// State mutation helpers extracted from handleAgentUpdate to isolate model changes.
// Each function mirrors one branch from the previous switch in internal/ui/tui_update.go.

// handleAgentContent appends streaming content to the active assistant message.
func handleAgentContent(m *Model, msg bridge.ContentEvent) (bool, []tea.Cmd) {
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
	m.convVM.MarkMessageDirty(m.activeAssistantIndex)
	return true, nil
}

// handleAgentToolStart handles a tool start update: record history and surface a short system message.
func handleAgentToolStart(m *Model, msg bridge.ToolStartEvent) (bool, []tea.Cmd) {
	m.currentTool = msg.Name
	// Add to tool history
	m.conversation.AddToolStart(msg.Name)

	// Delegate display formatting to the view package.
	display := view.FormatToolStartDisplay(msg.Name, msg.Input)

	if view.NormalizeToolName(msg.Name) == "attempt_completion" {
		// Extract a human-friendly result from the JSON input and create an assistant message.
		assistantContent := view.FormatAttemptCompletionContent(msg.Input)
		idx := m.conversation.AppendMessage(model.Message{
			Role:      types.RoleAssistant,
			Content:   assistantContent,
			Timestamp: time.Now(),
		})
		m.convVM.MarkMessageDirty(idx)
	} else if view.NormalizeToolName(msg.Name) == "ask_followup_question" {
		// Skip adding a system message here; askQuestionMsg will display the question.
	} else if view.NormalizeToolName(msg.Name) == "plan_mode_respond" {
		// Skip: the completed result will be rendered as a full assistant message in toolComplete.
	} else {
		idx := m.conversation.AppendMessage(model.Message{
			Role:      types.RoleSystem,
			Content:   display,
			Timestamp: time.Now(),
		})
		m.convVM.MarkMessageDirty(idx)
	}
	return true, nil
}

// handleAgentToolComplete updates tool history and optionally appends a summary/system message or assistant message.
func handleAgentToolComplete(m *Model, msg bridge.ToolCompleteEvent) (bool, []tea.Cmd) {
	m.conversation.MarkToolComplete(msg.Name)
	m.currentTool = ""

	// Short-circuit cases handled elsewhere
	if view.NormalizeToolName(msg.Name) == "attempt_completion" || view.NormalizeToolName(msg.Name) == "ask_followup_question" {
		return true, nil
	}

	if view.NormalizeToolName(msg.Name) == "plan_mode_respond" {
		// Render the plan response as an assistant message (full markdown, no truncation)
		if msg.Result != "" {
			idx := m.conversation.AppendMessage(model.Message{
				Role:      types.RoleAssistant,
				Content:   msg.Result,
				Timestamp: time.Now(),
			})
			m.convVM.MarkMessageDirty(idx)
		}
		return true, nil
	}

	// Delegate display formatting to the view package.
	content := view.FormatToolCompleteDisplay(msg.Name, msg.Result, "completed")
	idx := m.conversation.AppendMessage(model.Message{
		Role:      types.RoleSystem,
		Content:   content,
		Timestamp: time.Now(),
	})
	m.convVM.MarkMessageDirty(idx)
	return true, nil
}

// handleAgentError marks running tools as failed, surfaces an error message, and focuses the textarea.
func handleAgentError(m *Model, msg bridge.ErrorEvent) (bool, []tea.Cmd) {
	var cmds []tea.Cmd
	m.err = msg.Err
	m.isProcessing = false
	m.isStreaming = false
	// If an error occurred during a tool run, mark the most recent running tool as failed for visibility.
	for i := len(m.conversation.ToolHistory) - 1; i >= 0; i-- {
		if m.conversation.ToolHistory[i].Status == "running" {
			m.conversation.ToolHistory[i].Status = "failed"
			// Append a short system message to make the failure obvious in the conversation
			idx := m.conversation.AppendMessage(model.Message{
				Role:      types.RoleSystem,
				Content:   fmt.Sprintf("🔧 Failed: %s", m.conversation.ToolHistory[i].Name),
				Timestamp: time.Now(),
			})
			m.convVM.MarkMessageDirty(idx)
			break
		}
	}
	m.addErrorMessage(fmt.Sprintf("Error: %v", msg.Err))
	m.textarea.Focus()
	cmds = append(cmds, textarea.Blink)
	return true, cmds
}

// handleAgentComplete finalizes streaming/processing state and focuses textarea.
func handleAgentComplete(m *Model, msg bridge.CompleteEvent) (bool, []tea.Cmd) {
	var cmds []tea.Cmd
	m.isProcessing = false
	m.isStreaming = false
	m.currentTool = ""
	m.textarea.Focus()
	cmds = append(cmds, textarea.Blink)
	return true, cmds
}

// handleAgentStreamStart creates a new assistant message slot for streaming.
func handleAgentStreamStart(m *Model, msg bridge.StreamStartEvent) (bool, []tea.Cmd) {
	m.isStreaming = true
	// Create a new assistant message slot for the new stream round
	m.activeAssistantIndex = m.conversation.AppendMessage(model.Message{
		Role:      types.RoleAssistant,
		Content:   "",
		Timestamp: time.Now(),
	})
	m.convVM.MarkMessageDirty(m.activeAssistantIndex)
	return true, nil
}

// handleAgentStreamEnd ends streaming.
func handleAgentStreamEnd(m *Model, msg bridge.StreamEndEvent) (bool, []tea.Cmd) {
	m.isStreaming = false
	return true, nil
}
