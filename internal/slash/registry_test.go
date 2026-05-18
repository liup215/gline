package slash

import (
	"testing"

	"github.com/liup215/gline/pkg/types"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r.Count() != 0 {
		t.Fatalf("expected 0 commands, got %d", r.Count())
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	cmd := &types.SlashCommand{
		Name:        "test",
		Description: "A test command",
		Section:     types.SectionDefault,
		Handler:     func(args string) (bool, error) { return true, nil },
	}
	if err := r.Register(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Count() != 1 {
		t.Fatalf("expected 1 command, got %d", r.Count())
	}

	// Duplicate should fail
	if err := r.Register(cmd); err == nil {
		t.Fatal("expected error for duplicate command")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	r.Register(&types.SlashCommand{
		Name: "clear",
		Handler: func(args string) (bool, error) { return true, nil },
	})

	cmd, ok := r.Get("clear")
	if !ok {
		t.Fatal("expected to find /clear command")
	}
	if cmd.Name != "clear" {
		t.Fatalf("expected name 'clear', got '%s'", cmd.Name)
	}

	// Case insensitive
	_, ok = r.Get("CLEAR")
	if !ok {
		t.Fatal("expected case-insensitive lookup to work")
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Fatal("expected not to find nonexistent command")
	}
}

func TestRegistry_GetAll(t *testing.T) {
	r := NewRegistry()
	r.Register(&types.SlashCommand{Name: "clear", Section: types.SectionDefault})
	r.Register(&types.SlashCommand{Name: "mycommand", Section: types.SectionCustom})
	r.Register(&types.SlashCommand{Name: "help", Section: types.SectionDefault})

	all := r.GetAll()
	if len(all) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(all))
	}

	// Custom commands should come first
	if all[0].Name != "mycommand" {
		t.Fatalf("expected custom command first, got '%s'", all[0].Name)
	}
}

func TestRegistry_Filter(t *testing.T) {
	r := NewRegistry()
	r.Register(&types.SlashCommand{Name: "clear", Section: types.SectionDefault})
	r.Register(&types.SlashCommand{Name: "compact", Section: types.SectionDefault})
	r.Register(&types.SlashCommand{Name: "exit", Section: types.SectionDefault})

	filtered := r.Filter("c")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 commands matching 'c', got %d", len(filtered))
	}

	filtered = r.Filter("cl")
	if len(filtered) != 1 || filtered[0].Name != "clear" {
		t.Fatalf("expected /clear, got %v", filtered)
	}

	filtered = r.Filter("xyz")
	if len(filtered) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(filtered))
	}
}
