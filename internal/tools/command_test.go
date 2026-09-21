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
