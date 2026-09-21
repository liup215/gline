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
	ScrollOffset  int // index of the first visible task (windowed list)
	ShowDetail    bool
	DetailTask    *storage.TaskRecord
	DetailMsgs    []storage.MessageRecord
	ConfirmDelete string // task ID awaiting deletion confirmation
	Width         int
	Height        int
}

// Layout constants for the history list: title block (2 lines),
// footer block (2 lines) and 3 lines per task row (title + meta + blank).
const (
	historyChromeLines = 4
	historyRowLines    = 3
	maxDetailMessages  = 40
)

// HistoryVisibleRows returns how many task rows fit on the history screen
// for the given terminal height. Shared by the renderer and the model so
// scroll bookkeeping and drawing always agree.
func HistoryVisibleRows(height int) int {
	rows := 1
	if height > historyChromeLines+historyRowLines {
		rows = (height - historyChromeLines) / historyRowLines
	}
	return rows
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

	// Windowed rendering: only the rows that fit on screen, starting at
	// ScrollOffset, so long lists stay navigable on short terminals.
	visible := HistoryVisibleRows(data.Height)
	start := data.ScrollOffset
	if start < 0 {
		start = 0
	}
	if start > len(data.Tasks) {
		start = len(data.Tasks)
	}
	end := start + visible
	if end > len(data.Tasks) {
		end = len(data.Tasks)
	}

	for i, t := range data.Tasks[start:end] {
		i := start + i
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

	// Footer help with a position indicator when the list is windowed.
	b.WriteString("\n")
	help := "↑/↓ select • Enter: load & continue • D: delete • Esc: back"
	if data.ConfirmDelete != "" {
		help = "Press Y to confirm deletion, N to cancel"
	}
	if visible < len(data.Tasks) {
		help += fmt.Sprintf("    [%d–%d / %d]", start+1, end, len(data.Tasks))
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
	b.WriteString(fmt.Sprintf("  ID:       %s\n", t.ID))
	b.WriteString(fmt.Sprintf("  Status:   %s\n", statusLabel(t.Status)))
	b.WriteString(fmt.Sprintf("  Mode:     %s\n", t.Mode))
	b.WriteString(fmt.Sprintf("  Provider: %s / %s\n", t.Provider, t.Model))
	b.WriteString(fmt.Sprintf("  Created:  %s\n", formatHistoryTime(t.CreatedAt)))
	b.WriteString("\n")

	// Messages
	b.WriteString(SystemStyle.Render(fmt.Sprintf("  Messages (%d):\n", len(data.DetailMsgs))))
	shown := len(data.DetailMsgs)
	if shown > maxDetailMessages {
		shown = maxDetailMessages
	}
	for j, m := range data.DetailMsgs[:shown] {
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
		b.WriteString(fmt.Sprintf("    [%d] %s: %s\n", j+1, roleLabel, preview))
	}
	if shown < len(data.DetailMsgs) {
		b.WriteString(HelpStyle.Render(fmt.Sprintf("    … and %d more (open the task to view full history)\n", len(data.DetailMsgs)-shown)))
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
