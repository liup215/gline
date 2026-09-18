package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/liup215/gline/internal/storage"
)

// sidebarModel manages the task history sidebar.
type sidebarModel struct {
	tasks      []storage.TaskRecord
	selected   int
	loading    bool
	width      int
	height     int
	store      storage.Store
}

func newSidebarModel(store storage.Store) sidebarModel {
	return sidebarModel{
		store:    store,
		tasks:    make([]storage.TaskRecord, 0),
		selected: -1,
	}
}

func (m *sidebarModel) setWidthHeight(width, height int) {
	m.width = width
	m.height = height
}

// loadTasks fetches tasks from storage.
func (m *sidebarModel) loadTasks() {
	if m.store == nil {
		return
	}
	m.loading = true
	tasks, err := m.store.ListTasks(20, 0)
	if err != nil {
		m.loading = false
		return
	}
	m.tasks = tasks
	m.loading = false
}

// selectTask selects a task by index.
func (m *sidebarModel) selectTask(idx int) {
	if idx >= 0 && idx < len(m.tasks) {
		m.selected = idx
	}
}

// getSelectedTask returns the currently selected task.
func (m *sidebarModel) getSelectedTask() *storage.TaskRecord {
	if m.selected >= 0 && m.selected < len(m.tasks) {
		return &m.tasks[m.selected]
	}
	return nil
}

// render renders the sidebar.
func (m *sidebarModel) render() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(SidebarTitleStyle.Render("📋 History"))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(SidebarItemStyle.Render("  Loading..."))
	} else if len(m.tasks) == 0 {
		b.WriteString(SidebarItemStyle.Render("  No tasks yet"))
		b.WriteString("\n")
		b.WriteString(SidebarItemStyle.Render("  Start a chat to"))
		b.WriteString("\n")
		b.WriteString(SidebarItemStyle.Render("  create your first task."))
	} else {
		for i, task := range m.tasks {
			item := m.renderTaskItem(task, i == m.selected)
			b.WriteString(item)
			b.WriteString("\n")
		}
	}

	// Pad to height
	content := b.String()
	lines := strings.Count(content, "\n")
	remaining := m.height - 1 - lines // minus status bar
	if remaining > 0 {
		content += strings.Repeat("\n", remaining)
	}

	return SidebarStyle.
		Width(m.width).
		Height(m.height - 1).
		Render(content)
}

// renderTaskItem renders a single task item.
func (m *sidebarModel) renderTaskItem(task storage.TaskRecord, selected bool) string {
	// Status icon
	icon := "○"
	if task.Status == "completed" {
		icon = "✓"
	} else if task.Status == "failed" {
		icon = "✗"
	} else if task.Status == "running" {
		icon = "●"
	}

	// Truncate title
	title := task.Title
	if len(title) > 18 {
		title = title[:15] + "..."
	}

	// Build the line
	line := fmt.Sprintf("%s %s", icon, title)

	// Mode badge
	modeStyle := lipgloss.NewStyle().
		Foreground(ColorDim).
		PaddingLeft(1)
	mode := modeStyle.Render(task.Mode)

	if selected {
		return SidebarItemActiveStyle.Render(line) + mode
	}
	return SidebarItemStyle.Render(line) + mode
}
