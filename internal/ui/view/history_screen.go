package view

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/storage"
)

// HistoryScreenData holds the data needed to render the history screen.
type HistoryScreenData struct {
	Tasks         []storage.TaskRecord
	SelectedIndex int
	ShowDetail    bool
	DetailTask    *storage.TaskRecord
	DetailMsgs    []storage.MessageRecord
	ConfirmDelete string // task ID awaiting deletion confirmation
	Width         int
	Height        int
}

// RenderHistoryScreen renders the full-screen history view.
func RenderHistoryScreen(data HistoryScreenData) string {
	if data.ShowDetail && data.DetailTask != nil {
		return renderHistoryDetail(data)
	}
	return renderHistoryList(data)
}

func renderHistoryList(data HistoryScreenData) string {
	var b strings.Builder

	// Title
	title := TitleStyle.Render(" 📜 Conversation History")
	b.WriteString(title)
	b.WriteString("\n\n")

	if len(data.Tasks) == 0 {
		b.WriteString(SystemStyle.Render("  No tasks found. Start a conversation to create history.\n"))
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("  Press Esc to return"))
		return b.String()
	}

	for i, t := range data.Tasks {
		prefix := "  "
		if i == data.SelectedIndex {
			prefix = "▸ "
		}

		statusIcon := "●"
		statusColor := lipgloss.Color("#FFA500")
		if t.Status == "completed" {
			statusIcon = "✓"
			statusColor = lipgloss.Color("#00AA00")
		} else if t.Status == "failed" {
			statusIcon = "✗"
			statusColor = lipgloss.Color("#FF4444")
		}

		titleLine := fmt.Sprintf("%s%s %s", prefix,
			lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon),
			t.Title)

		meta := fmt.Sprintf("    [%s | %s | %s]  %s",
			t.Mode, t.Provider, t.Model, formatHistoryTime(t.CreatedAt))

		if i == data.SelectedIndex {
			titleLine = lipgloss.NewStyle().Bold(true).Render(titleLine)
			meta = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Render(meta)
		}

		b.WriteString(titleLine + "\n")
		b.WriteString(meta + "\n\n")
	}

	// Footer help
	b.WriteString("\n")
	help := "↑/↓ select • Enter: load & continue • D: delete • Esc: back"
	if data.ConfirmDelete != "" {
		help = "Press Y to confirm deletion, N to cancel"
	}
	b.WriteString(HelpStyle.Render("  " + help))
	return b.String()
}

func renderHistoryDetail(data HistoryScreenData) string {
	var b strings.Builder

	t := data.DetailTask

	// Header
	b.WriteString(TitleStyle.Render(" 📄 Task Details"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  Title:    %s\n", t.Title))
	b.WriteString(fmt.Sprintf("  ID:       %s\n", t.ID[:8]))
	b.WriteString(fmt.Sprintf("  Status:   %s\n", statusLabel(t.Status)))
	b.WriteString(fmt.Sprintf("  Mode:     %s\n", t.Mode))
	b.WriteString(fmt.Sprintf("  Provider: %s / %s\n", t.Provider, t.Model))
	b.WriteString(fmt.Sprintf("  Created:  %s\n", formatHistoryTime(t.CreatedAt)))
	b.WriteString("\n")

	// Messages
	b.WriteString(SystemStyle.Render(fmt.Sprintf("  Messages (%d):\n", len(data.DetailMsgs))))
	for i, m := range data.DetailMsgs {
		roleLabel := m.Role
		if roleLabel == "assistant" {
			roleLabel = "AI"
		} else if roleLabel == "user" {
			roleLabel = "You"
		} else if roleLabel == "tool" {
			roleLabel = "Tool"
		}

		preview := m.Content
		if len(preview) > 80 {
			preview = preview[:77] + "..."
		}
		if preview == "" {
			if m.ToolCalls != "" {
				preview = "[tool call]"
			} else {
				preview = "[empty]"
			}
		}
		b.WriteString(fmt.Sprintf("    [%d] %s: %s\n", i+1, roleLabel, preview))
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  Enter: load & continue • Esc: back to list"))
	return b.String()
}

func statusLabel(status string) string {
	switch status {
	case "completed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00AA00")).Render("completed")
	case "failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444")).Render("failed")
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("running")
	}
}

func formatHistoryTime(t time.Time) string {
	duration := time.Since(t)
	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	}
	if duration < 30*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	}
	return t.Format("2006-01-02")
}
