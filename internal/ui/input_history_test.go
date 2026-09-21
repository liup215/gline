package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liup215/gline/internal/storage"
)

func key(k tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: k} }

func TestInputHistoryRecallViaArrowKeys(t *testing.T) {
	m := New(nil, nil)
	m.width = 100
	m.height = 30

	m.addToInputHistory("first prompt")
	m.addToInputHistory("second prompt")

	// ↑ recalls the newest entry
	m.Update(key(tea.KeyUp))
	if got := m.textarea.Value(); got != "second prompt" {
		t.Fatalf("first ↑: expected %q, got %q", "second prompt", got)
	}
	// ↑ again recalls the older entry
	m.Update(key(tea.KeyUp))
	if got := m.textarea.Value(); got != "first prompt" {
		t.Fatalf("second ↑: expected %q, got %q", "first prompt", got)
	}
	// ↑ at the oldest entry stays put
	m.Update(key(tea.KeyUp))
	if got := m.textarea.Value(); got != "first prompt" {
		t.Fatalf("↑ past oldest: expected %q, got %q", "first prompt", got)
	}
	// ↓ returns toward newer entries
	m.Update(key(tea.KeyDown))
	if got := m.textarea.Value(); got != "second prompt" {
		t.Fatalf("↓: expected %q, got %q", "second prompt", got)
	}
}

func TestInputHistoryDraftRestored(t *testing.T) {
	m := New(nil, nil)
	m.width = 100
	m.height = 30
	m.addToInputHistory("sent prompt")

	// Type a draft, then browse history
	m.textarea.SetValue("draft in progress")
	m.Update(key(tea.KeyUp))
	if got := m.textarea.Value(); got != "sent prompt" {
		t.Fatalf("↑: expected %q, got %q", "sent prompt", got)
	}
	// ↓ past the newest entry restores the draft
	m.Update(key(tea.KeyDown))
	if got := m.textarea.Value(); got != "draft in progress" {
		t.Fatalf("↓ past newest: expected draft %q, got %q", "draft in progress", got)
	}
}

func TestInputHistoryMultilineKeepsCursorMovement(t *testing.T) {
	m := New(nil, nil)
	m.width = 100
	m.height = 30
	m.addToInputHistory("sent prompt")

	m.textarea.SetValue("line one\nline two")
	m.Update(key(tea.KeyUp))
	if got := m.textarea.Value(); got != "line one\nline two" {
		t.Fatalf("multi-line content must not be replaced by history, got %q", got)
	}
	if m.histIdx != -1 {
		t.Fatalf("browsing must not start for multi-line content, histIdx=%d", m.histIdx)
	}
}

func TestInputHistoryNoDuplicatesAndBrowsingResetOnSend(t *testing.T) {
	m := New(nil, nil)
	m.width = 100
	m.height = 30
	m.addToInputHistory("same prompt")
	m.addToInputHistory("same prompt")
	if len(m.inputHistory) != 1 {
		t.Fatalf("expected consecutive duplicate suppressed, got %d entries", len(m.inputHistory))
	}
	m.addToInputHistory("")
	if len(m.inputHistory) != 1 {
		t.Fatalf("expected empty input skipped, got %d entries", len(m.inputHistory))
	}

	// Browsing position resets after a submit (submitUserMessage path).
	m.textarea.SetValue("new prompt")
	m.Update(key(tea.KeyUp)) // browse
	if m.histIdx == -1 {
		t.Fatal("expected browsing active after ↑")
	}
	// Simulate the submit-path bookkeeping (agent-independent part).
	m.addToInputHistory(strings.TrimSpace(m.textarea.Value()))
	m.resetHistoryBrowsing()
	if m.histIdx != -1 || m.histDraft != "" {
		t.Fatalf("expected browsing reset, histIdx=%d draft=%q", m.histIdx, m.histDraft)
	}
}

func TestHistoryScreenArrowNavigation(t *testing.T) {
	m := New(nil, nil)
	m.width = 100
	m.height = 30
	m.historyTasks = []storage.TaskRecord{
		{ID: "t1", Title: "task one"},
		{ID: "t2", Title: "task two"},
		{ID: "t3", Title: "task three"},
	}
	m.screen = ScreenHistory

	if v := stripANSI(m.View()); !strings.Contains(v, "▸ ● task one") {
		t.Fatalf("initial marker should be on first task:\n%s", v)
	}

	updated, _ := m.Update(key(tea.KeyDown))
	m1 := updated.(*Model)
	if m1.historySelected != 1 {
		t.Fatalf("↓: expected selected=1, got %d", m1.historySelected)
	}
	if v := stripANSI(m1.View()); !strings.Contains(v, "▸ ● task two") {
		t.Fatalf("marker should move to second task:\n%s", v)
	}

	updated2, _ := m1.Update(key(tea.KeyDown))
	m2 := updated2.(*Model)
	if m2.historySelected != 2 {
		t.Fatalf("↓: expected selected=2, got %d", m2.historySelected)
	}
	// ↓ at the last entry stays put
	updated3, _ := m2.Update(key(tea.KeyDown))
	m3 := updated3.(*Model)
	if m3.historySelected != 2 {
		t.Fatalf("↓ at last: expected selected=2, got %d", m3.historySelected)
	}

	updated4, _ := m3.Update(key(tea.KeyUp))
	m4 := updated4.(*Model)
	if m4.historySelected != 1 {
		t.Fatalf("↑: expected selected=1, got %d", m4.historySelected)
	}
}

func TestHistoryScreenSlashModeStaysInactive(t *testing.T) {
	m := New(nil, nil)
	m.width = 100
	m.height = 30
	m.screen = ScreenHistory

	// A lingering "/" in the textarea must not re-activate slash mode on
	// the history screen (it would silently eat ↑/↓).
	m.textarea.SetValue("/")
	m.Update(key(tea.KeyRunes))

	if m.slashMenu.Active {
		t.Fatal("slash mode must stay inactive on the history screen")
	}
	// Arrow keys still navigate the list
	m.historyTasks = []storage.TaskRecord{
		{ID: "t1", Title: "task one"},
		{ID: "t2", Title: "task two"},
	}
	updated, _ := m.Update(key(tea.KeyDown))
	mm := updated.(*Model)
	if mm.historySelected != 1 {
		t.Fatalf("↓: expected selected=1, got %d", mm.historySelected)
	}
}
