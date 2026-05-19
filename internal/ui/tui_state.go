// State mutation helpers extracted from handleAgentUpdate to isolate model changes.
// Each function mirrors one branch from the previous switch in internal/ui/tui_update.go.

package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/liup215/gline/internal/ui/bridge"
	"github.com/liup215/gline/internal/ui/model"
	"github.com/liup215/gline/internal/ui/tool"
	"github.com/liup215/gline/pkg/types"
)

// State mutation helpers extracted from handleAgentUpdate to isolate model changes.
// Each function mirrors one branch from the previous switch in internal/ui/tui_update.go.

// handleAgentContent appends streaming content to the active assistant message.
func handleAgentContent(m *Model, msg bridge.ContentEvent) (bool, []tea.Cmd) {
	// Ensure an assistant slot exists; create one if not present
	// Check bounds and role to handle cases where tool messages were added
	if m.activeAssistantIndex < 0 || m.activeAssistantIndex >= len(m.conversation.Messages) {
		// Create a new assistant message slot and set it active.
		m.activeAssistantIndex = m.conversation.AppendMessage(model.Message{
			Role:      types.RoleAssistant,
			Content:   "",
			Timestamp: time.Now(),
		})
	} else if m.conversation.Messages[m.activeAssistantIndex].Role != types.RoleAssistant {
		// The index points to a non-assistant message (e.g., tool message)
		// Create a new assistant message
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

	// Get tool renderer from registry
	toolName := tool.NormalizeToolName(msg.Name)
	renderer := m.toolRegistry.Get(toolName)

	// Request rendering from tool
	result := renderer.Render(tool.RenderRequest{
		Phase:  types.ToolPhaseStart,
		Input:  msg.Input,
		Status: "running",
	})

	if result.Skip {
		return true, nil
	}

	// Create message based on tool's decision
	newMsg := model.Message{
		Role:      result.Role,
		Content:   result.Content,
		Strategy:  result.Strategy,
		MsgType:   types.TypeToolStart,
		Timestamp: time.Now(),
	}
	// Set metadata for tool status
	newMsg.SetMeta(model.ToolMeta{
		ToolName: msg.Name,
		Status:   "running",
	})
	idx := m.conversation.AppendMessage(newMsg)
	m.convVM.MarkMessageDirty(idx)
	return true, nil
}

// handleAgentToolComplete updates tool history and optionally appends a summary/system message or assistant message.
func handleAgentToolComplete(m *Model, msg bridge.ToolCompleteEvent) (bool, []tea.Cmd) {
	m.conversation.MarkToolComplete(msg.Name)
	m.currentTool = ""

	// Get tool renderer from registry
	toolName := tool.NormalizeToolName(msg.Name)
	renderer := m.toolRegistry.Get(toolName)

	// Request rendering from tool
	result := renderer.Render(tool.RenderRequest{
		Phase:  types.ToolPhaseComplete,
		Input:  msg.Result,
		Status: "completed",
	})

	if result.Skip {
		return true, nil
	}

	// Create message based on tool's decision
	newMsg := model.Message{
		Role:      result.Role,
		Content:   result.Content,
		Strategy:  result.Strategy,
		MsgType:   types.TypeToolComplete,
		Timestamp: time.Now(),
	}
	// Set metadata for tool status
	newMsg.SetMeta(model.ToolMeta{
		ToolName: msg.Name,
		Status:   "completed",
	})
	idx := m.conversation.AppendMessage(newMsg)
	m.convVM.MarkMessageDirty(idx)
	return true, nil
}

// handleAgentError marks running tools as failed, surfaces an error message, and focuses the textarea.
func handleAgentError(m *Model, msg bridge.ErrorEvent) (bool, []tea.Cmd) {
	var cmds []tea.Cmd
	m.err = msg.Err
	m.isProcessing = false
	m.isStreaming = false
	// If an error occurred during a tool run, mark the most recent running tool as failed.
	for i := len(m.conversation.ToolHistory) - 1; i >= 0; i-- {
		if m.conversation.ToolHistory[i].Status == "running" {
			m.conversation.ToolHistory[i].Status = "failed"
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

// handleAgentStreamStart creates (or reuses) an assistant message slot for streaming.
func handleAgentStreamStart(m *Model, msg bridge.StreamStartEvent) (bool, []tea.Cmd) {
	m.isStreaming = true
	// Reuse the existing assistant message slot if it is still empty.
	// This prevents accumulating blank assistant messages across multiple
	// tool-call rounds where the LLM outputs no text before making a tool call.
	if m.activeAssistantIndex >= 0 &&
		m.activeAssistantIndex < len(m.conversation.Messages) &&
		m.conversation.Messages[m.activeAssistantIndex].Role == types.RoleAssistant &&
		m.conversation.Messages[m.activeAssistantIndex].Content == "" {
		// Reuse: just refresh the timestamp so it looks current.
		m.conversation.Messages[m.activeAssistantIndex].Timestamp = time.Now()
		m.convVM.MarkMessageDirty(m.activeAssistantIndex)
		return true, nil
	}
	// Create a new assistant message slot for this stream round.
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
