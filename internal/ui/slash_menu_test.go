package ui

import (
	"testing"

	"github.com/liup215/gline/internal/slash"
)

func TestNewSlashMenuState(t *testing.T) {
	registry := slash.NewDefaultRegistry(nil, nil)
	state := NewSlashMenuState(registry)

	if state.Active {
		t.Fatal("expected inactive by default")
	}
	if state.Selected != 0 {
		t.Fatalf("expected selected 0, got %d", state.Selected)
	}
	if len(state.Filtered) != 0 {
		t.Fatalf("expected 0 filtered, got %d", len(state.Filtered))
	}
}

func TestSlashMenuState_EnterSlashMode(t *testing.T) {
	registry := slash.NewDefaultRegistry(nil, nil)
	state := NewSlashMenuState(registry)

	state.EnterSlashMode()

	if !state.Active {
		t.Fatal("expected active after EnterSlashMode")
	}
	if len(state.Filtered) != 7 { // clear, help, exit, q, newtask, smol, compact
		t.Fatalf("expected 7 commands, got %d", len(state.Filtered))
	}
}

func TestSlashMenuState_UpdateQuery(t *testing.T) {
	registry := slash.NewDefaultRegistry(nil, nil)
	state := NewSlashMenuState(registry)

	// Enter slash mode with empty query
	state.EnterSlashMode()

	// Filter with '/c' - should match clear, compact
	state.UpdateQuery("/c", 2)
	if !state.Active {
		t.Fatal("expected still active after '/c'")
	}
	if len(state.Filtered) != 2 {
		t.Fatalf("expected 2 matches for 'c', got %d", len(state.Filtered))
	}

	// Filter with '/cl' - should match clear only
	state.UpdateQuery("/cl", 3)
	if len(state.Filtered) != 1 {
		t.Fatalf("expected 1 match for 'cl', got %d", len(state.Filtered))
	}
	if state.Filtered[0].Name != "clear" {
		t.Fatalf("expected /clear, got /%s", state.Filtered[0].Name)
	}

	// Non-matching query should still keep active but empty filtered
	state.UpdateQuery("/xyz", 4)
	if !state.Active {
		t.Fatal("expected still active even with no matches")
	}
	if len(state.Filtered) != 0 {
		t.Fatalf("expected 0 matches for 'xyz', got %d", len(state.Filtered))
	}
}

func TestSlashMenuState_NextPrev(t *testing.T) {
	registry := slash.NewDefaultRegistry(nil, nil)
	state := NewSlashMenuState(registry)
	state.EnterSlashMode()

	// Should have 7 commands, cycle through them
	state.Next()
	if state.Selected != 1 {
		t.Fatalf("expected selected 1, got %d", state.Selected)
	}

	state.Next()
	if state.Selected != 2 {
		t.Fatalf("expected selected 2, got %d", state.Selected)
	}

	state.Prev()
	if state.Selected != 1 {
		t.Fatalf("expected selected 1, got %d", state.Selected)
	}

	// Wrap around forward
	for i := 0; i < 10; i++ {
		state.Next()
	}
	if state.Selected != 4 { // (1 + 10) % 7 = 4
		t.Fatalf("expected wrap to 4, got %d", state.Selected)
	}

	// Wrap around backward from 0
	state.Selected = 0
	state.Prev()
	if state.Selected != 6 {
		t.Fatalf("expected wrap to 6, got %d", state.Selected)
	}
}

func TestSlashMenuState_SelectedCommand(t *testing.T) {
	registry := slash.NewDefaultRegistry(nil, nil)
	state := NewSlashMenuState(registry)
	state.EnterSlashMode()

	cmd := state.SelectedCommand()
	if cmd == nil {
		t.Fatal("expected a selected command")
	}
	if cmd.Name != "clear" { // first by sorted order
		t.Fatalf("expected /clear, got /%s", cmd.Name)
	}
}

func TestSlashMenuState_ExitSlashMode(t *testing.T) {
	registry := slash.NewDefaultRegistry(nil, nil)
	state := NewSlashMenuState(registry)
	state.EnterSlashMode()

	state.ExitSlashMode()
	if state.Active {
		t.Fatal("expected inactive after exit")
	}
	if len(state.Filtered) != 0 {
		t.Fatalf("expected filtered cleared, got %d", len(state.Filtered))
	}
}
