// Package ui: extracted agent update handlers
package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textarea"

	"github.com/liup215/gline/pkg/types"
)

// handleAgentUpdate centralizes handling of agentUpdateMsg cases previously in Update.
func handleAgentUpdate(m *Model, msg agentUpdateMsg) []tea.Cmd {
	var cmds []tea.Cmd

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

	return cmds
}