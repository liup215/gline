package ui

import (
	"strings"

	"github.com/liup215/gline/internal/ui/view"
)

// Input history recall (↑/↓ in the chat input).
//
// inputHistory stores previously submitted prompts (oldest first).
// histIdx is the browsing position (-1 = not browsing). When browsing
// starts from a non-empty input, the in-progress text is saved in
// histDraft and restored when the user navigates past the newest entry.
// Browsing only engages while the textarea holds a single line, so
// multi-line editing keeps native ↑/↓ cursor movement.

const maxInputHistory = 100

// addToInputHistory records a submitted prompt, skipping empties and
// consecutive duplicates, capped at maxInputHistory entries.
func (m *Model) addToInputHistory(val string) {
	val = strings.TrimSpace(val)
	if val == "" {
		return
	}
	if len(m.inputHistory) > 0 && m.inputHistory[len(m.inputHistory)-1] == val {
		return
	}
	m.inputHistory = append(m.inputHistory, val)
	if len(m.inputHistory) > maxInputHistory {
		m.inputHistory = m.inputHistory[len(m.inputHistory)-maxInputHistory:]
	}
}

// resetHistoryBrowsing stops history browsing. The next ↑ starts from the
// newest entry again.
func (m *Model) resetHistoryBrowsing() {
	m.histIdx = -1
	m.histDraft = ""
}

// adjustHistoryScroll keeps the selected history task inside the visible
// window of the /history list so long lists stay navigable on short
// terminals.
func (m *Model) adjustHistoryScroll() {
	visible := view.HistoryVisibleRows(m.height)
	if m.historyScroll > m.historySelected {
		m.historyScroll = m.historySelected
	}
	if m.historySelected >= m.historyScroll+visible {
		m.historyScroll = m.historySelected - visible + 1
	}
	if m.historyScroll < 0 {
		m.historyScroll = 0
	}
	if max := len(m.historyTasks) - visible; max < 0 {
		m.historyScroll = 0
	} else if m.historyScroll > max {
		m.historyScroll = max
	}
}

// canBrowseHistory reports whether ↑/↓ should be intercepted for history
// recall instead of native textarea cursor movement.
func (m *Model) canBrowseHistory() bool {
	return !strings.Contains(m.textarea.Value(), "\n")
}

// prevInputHistory recalls the previous (older) entry with ↑.
func (m *Model) prevInputHistory() {
	if len(m.inputHistory) == 0 {
		return
	}
	if m.histIdx == -1 {
		m.histDraft = m.textarea.Value()
		m.histIdx = len(m.inputHistory) - 1
	} else if m.histIdx > 0 {
		m.histIdx--
	}
	m.textarea.SetValue(m.inputHistory[m.histIdx])
	m.textarea.CursorEnd()
}

// nextInputHistory recalls the next (newer) entry with ↓; navigating past
// the newest entry restores the saved draft.
func (m *Model) nextInputHistory() {
	if m.histIdx == -1 {
		return
	}
	if m.histIdx < len(m.inputHistory)-1 {
		m.histIdx++
		m.textarea.SetValue(m.inputHistory[m.histIdx])
		m.textarea.CursorEnd()
	} else {
		m.histIdx = -1
		draft := m.histDraft
		m.histDraft = ""
		m.textarea.SetValue(draft)
		m.textarea.CursorEnd()
	}
}
