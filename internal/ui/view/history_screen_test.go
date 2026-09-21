package view

import (
	"fmt"
	"strings"
	"testing"

	"github.com/liup215/gline/internal/storage"
)

func bigTaskList(n int) []storage.TaskRecord {
	tasks := make([]storage.TaskRecord, n)
	for i := range tasks {
		tasks[i] = storage.TaskRecord{ID: fmt.Sprintf("task-%d", i+1), Title: taskTitle(i), Status: "done"}
	}
	return tasks
}

func taskTitle(i int) string { return "task " + string(rune('a'+i%26)) + string(rune('0'+i/26%10)) }

func TestHistoryVisibleRows(t *testing.T) {
	if got := HistoryVisibleRows(24); got != 6 {
		t.Fatalf("height 24: expected 6 visible rows, got %d", got)
	}
	if got := HistoryVisibleRows(6); got != 1 {
		t.Fatalf("tiny height: expected at least 1 visible row, got %d", got)
	}
}

func TestHistoryListWindowedRendering(t *testing.T) {
	tasks := bigTaskList(20)
	base := HistoryScreenData{Tasks: tasks, SelectedIndex: 0, Height: 24, Width: 80}

	// ScrollOffset 0: first window visible, later tasks not drawn.
	top := RenderHistoryScreen(base)
	if !strings.Contains(top, "▸ ● task a0") || !strings.Contains(top, "● task f0") {
		t.Fatalf("window start should show first rows:\n%s", top)
	}
	if strings.Contains(top, "task k0") {
		t.Fatal("rows beyond the window must not be rendered")
	}
	if !strings.Contains(top, "[1–6 / 20]") {
		t.Fatalf("missing position indicator:\n%s", top)
	}

	// ScrollOffset 5: window shows tasks 5..10, top hidden.
	mid := RenderHistoryScreen(HistoryScreenData{Tasks: tasks, SelectedIndex: 5, ScrollOffset: 5, Height: 24, Width: 80})
	if !strings.Contains(mid, "▸ ● "+taskTitle(5)) {
		t.Fatalf("offset window should show selected row:\n%s", mid)
	}
	if strings.Contains(mid, taskTitle(0)) {
		t.Fatal("rows above the window must not be rendered")
	}
}

func TestHistoryDetailTruncatesMessages(t *testing.T) {
	msgs := make([]storage.MessageRecord, maxDetailMessages+5)
	for i := range msgs {
		msgs[i] = storage.MessageRecord{ID: int64(i), Role: "user", Content: "hello"}
	}
	out := RenderHistoryScreen(HistoryScreenData{
		Tasks: bigTaskList(1), ShowDetail: true,
		DetailTask: &storage.TaskRecord{ID: "task-1", Title: "task a0"},
		DetailMsgs: msgs, Height: 40, Width: 80,
	})
	if !strings.Contains(out, "and 5 more") {
		t.Fatalf("expected truncation note for excess messages:\n%s", out)
	}
}
