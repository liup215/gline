package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/liup215/gline/internal/shell"
)

func TestExecuteCommandUsesResolvedShell(t *testing.T) {
	sh := shell.Resolve()
	tool := NewExecuteCommandTool()

	// $(( )) arithmetic is POSIX-shell syntax; cmd.exe cannot evaluate it,
	// so a "5" in the output proves the command ran under bash/sh.
	input, _ := json.Marshal(map[string]string{"command": "echo $((2+3))"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sh.IsBash {
		if !strings.Contains(result, "5") {
			t.Fatalf("expected arithmetic result 5 under bash, got: %s", result)
		}
	} else {
		// cmd.exe fallback: the expression is not expanded; just ensure the
		// command executed and reported an exit code.
		if !strings.Contains(result, "Exit code:") {
			t.Fatalf("expected exit code in output, got: %s", result)
		}
	}
}

func TestExecuteCommandRequiresCommand(t *testing.T) {
	tool := NewExecuteCommandTool()
	input, _ := json.Marshal(map[string]string{"command": ""})
	if _, err := tool.Execute(context.Background(), input); err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestExecuteCommandDescriptionMatchesResolvedShell(t *testing.T) {
	tool := NewExecuteCommandTool()
	desc := tool.Description()
	if desc == "" {
		t.Fatal("empty description")
	}
	// The description must commit to one shell syntax (pi's lesson: the
	// model writes for the shell the description names, never "either").
	if strings.Contains(desc, "when available") || strings.Contains(desc, "falls back") {
		t.Fatalf("description hedges shell choice instead of committing: %s", desc)
	}
	switch shell.Resolve().Name() {
	case "bash":
		if !strings.Contains(desc, "POSIX syntax works") {
			t.Errorf("bash machine but description lacks POSIX guidance: %s", desc)
		}
	case "cmd":
		if !strings.Contains(desc, "cmd.exe") {
			t.Errorf("cmd machine but description lacks cmd guidance: %s", desc)
		}
	case "sh":
		if !strings.Contains(desc, "under sh") {
			t.Errorf("sh machine but description lacks sh guidance: %s", desc)
		}
	}
	// Routing guidance must steer file work to the dedicated tools.
	if !strings.Contains(desc, "use read") || !strings.Contains(desc, "grep/glob") {
		t.Errorf("description lacks tool-routing guidance: %s", desc)
	}
}

func TestExecuteCommandSchemaDropsRequiresApproval(t *testing.T) {
	tool := NewExecuteCommandTool()
	if strings.Contains(string(tool.InputSchema()), "requires_approval") {
		t.Fatal("schema still advertises dead requires_approval parameter")
	}
}
