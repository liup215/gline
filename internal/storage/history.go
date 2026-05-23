// Package storage provides persistent storage for gline.
package storage

import (
	"fmt"
	"strings"
	"time"
)

// FormatTaskList returns a formatted table string for console output.
func FormatTaskList(tasks []TaskRecord, withIndex bool) string {
	if len(tasks) == 0 {
		return "No tasks found."
	}

	var b strings.Builder

	for i, t := range tasks {
		if withIndex {
			b.WriteString(fmt.Sprintf("[%d] ", i))
		}
		statusIcon := "●"
		if t.Status == "completed" {
			statusIcon = "✓"
		} else if t.Status == "failed" {
			statusIcon = "✗"
		}
		b.WriteString(fmt.Sprintf("%s %s", statusIcon, t.Title))
		b.WriteString(fmt.Sprintf("  [%s | %s | %s]", t.Mode, t.Provider, t.Model))
		b.WriteString(fmt.Sprintf("  %s", formatTime(t.CreatedAt)))
		if t.CompletedAt != nil {
			b.WriteString(fmt.Sprintf(" → %s", formatTime(*t.CompletedAt)))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// FormatTaskDetail returns a detailed view of a task with messages.
func FormatTaskDetail(task *TaskRecord, msgs []MessageRecord) string {
	if task == nil {
		return "Task not found."
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("Task: %s\n", task.Title))
	b.WriteString(fmt.Sprintf("ID: %s\n", task.ID))
	b.WriteString(fmt.Sprintf("Status: %s\n", task.Status))
	b.WriteString(fmt.Sprintf("Mode: %s\n", task.Mode))
	b.WriteString(fmt.Sprintf("Provider: %s / %s\n", task.Provider, task.Model))
	b.WriteString(fmt.Sprintf("Created: %s\n", formatTime(task.CreatedAt)))
	if task.CompletedAt != nil {
		b.WriteString(fmt.Sprintf("Completed: %s\n", formatTime(*task.CompletedAt)))
	}
	b.WriteString(fmt.Sprintf("Prompt: %s\n", task.Prompt))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Messages (%d):\n", len(msgs)))
	for i, m := range msgs {
		roleLabel := m.Role
		if roleLabel == "assistant" {
			roleLabel = "AI"
		} else if roleLabel == "user" {
			roleLabel = "You"
		} else if roleLabel == "tool" {
			roleLabel = "Tool"
		} else if roleLabel == "system" {
			roleLabel = "System"
		}

		preview := m.Content
		if len(preview) > 100 {
			preview = preview[:97] + "..."
		}
		if preview == "" {
			if m.ToolCalls != "" {
				preview = "[tool call]"
			} else {
				preview = "[empty]"
			}
		}

		b.WriteString(fmt.Sprintf("  [%d] %s: %s\n", i+1, roleLabel, preview))
	}

	return b.String()
}

func formatTime(t time.Time) string {
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
	return t.Format("2006-01-02 15:04")
}
