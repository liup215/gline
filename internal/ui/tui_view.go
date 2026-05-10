// Package ui: view helpers extracted from tui.go
package ui

import (
"bytes"
"encoding/json"
"fmt"
"strings"

"github.com/charmbracelet/glamour"
"github.com/charmbracelet/lipgloss"

"github.com/liup215/gline/internal/agent"
"github.com/liup215/gline/pkg/types"
)

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