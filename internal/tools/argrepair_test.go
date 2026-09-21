package tools

import (
	"encoding/json"
	"testing"
)

func TestRepairToolArgsAliases(t *testing.T) {
	tests := []struct {
		tool string
		in   map[string]any
		want map[string]any
	}{
		{
			tool: "read",
			in:   map[string]any{"file_path": "/tmp/a.go", "start_line": float64(5)},
			want: map[string]any{"path": "/tmp/a.go", "line_number": float64(5)},
		},
		{
			tool: "edit",
			in:   map[string]any{"file_path": "x.go", "old_string": "a", "new_string": "b"},
			want: map[string]any{"path": "x.go", "search": "a", "replace": "b"},
		},
		{
			tool: "grep",
			in:   map[string]any{"pattern": "foo", "include": "*.go"},
			want: map[string]any{"regex": "foo", "file_pattern": "*.go"},
		},
		{
			tool: "run",
			in:   map[string]any{"cmd": "go test", "workingDir": "/tmp"},
			want: map[string]any{"command": "go test", "cwd": "/tmp"},
		},
		{
			tool: "use_skill",
			in:   map[string]any{"skill": "release"},
			want: map[string]any{"skill_name": "release"},
		},
	}
	for _, tt := range tests {
		got := RepairToolArgs(tt.tool, nil, tt.in)
		if len(got) != len(tt.want) {
			t.Fatalf("%s: got %v, want %v", tt.tool, got, tt.want)
		}
		for k, v := range tt.want {
			if got[k] != v {
				t.Errorf("%s: key %q: got %#v, want %#v", tt.tool, k, got[k], v)
			}
		}
	}
}

func TestRepairToolArgsCanonicalWins(t *testing.T) {
	// When the model sends both the canonical name and an alias, the
	// canonical value must survive untouched.
	in := map[string]any{"path": "canonical.go", "file_path": "alias.go"}
	got := RepairToolArgs("read", nil, in)
	if got["path"] != "canonical.go" {
		t.Fatalf("canonical value overwritten: %v", got)
	}
	if _, ok := got["file_path"]; ok {
		t.Fatalf("alias key not removed: %v", got)
	}
}

func TestRepairToolArgsTypeCoercion(t *testing.T) {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"line_number": {"type": "integer"},
			"limit":       {"type": "integer"},
			"flag":        {"type": "boolean"},
			"items":       {"type": "array"},
			"opts":        {"type": "object"}
		}
	}`)
	in := map[string]any{
		"line_number": "5",
		"limit":       float64(2000.0), // 5.0-style integral float
		"flag":        "true",
		"items":       `["a","b"]`,
		"opts":        `{"x":1}`,
	}
	got := RepairToolArgs("read", schema, in)
	if got["line_number"] != 5 {
		t.Errorf("line_number: got %#v, want int 5", got["line_number"])
	}
	if got["limit"] != 2000 {
		t.Errorf("limit: got %#v, want int 2000", got["limit"])
	}
	if got["flag"] != true {
		t.Errorf("flag: got %#v, want bool true", got["flag"])
	}
	arr, ok := got["items"].([]any)
	if !ok || len(arr) != 2 {
		t.Errorf("items: got %#v, want decoded array", got["items"])
	}
	obj, ok := got["opts"].(map[string]any)
	if !ok || obj["x"] != float64(1) {
		t.Errorf("opts: got %#v, want decoded object", got["opts"])
	}
}

func TestRepairToolArgsNonIntegralFloatNotCoerced(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"line_number":{"type":"integer"}}}`)
	in := map[string]any{"line_number": 5.5}
	got := RepairToolArgs("read", schema, in)
	// 5.5 cannot be an int; leave as-is so the tool reports a real error
	// instead of silently rounding.
	if got["line_number"] != 5.5 {
		t.Fatalf("non-integral float was coerced: %#v", got["line_number"])
	}
}

func TestRepairToolArgsUnparseableStringLeftAlone(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"line_number":{"type":"integer"}}}`)
	in := map[string]any{"line_number": "abc"}
	got := RepairToolArgs("read", schema, in)
	if got["line_number"] != "abc" {
		t.Fatalf("unparseable string was dropped or altered: %#v", got["line_number"])
	}
}

func TestRepairToolArgsNilAndUnknownTool(t *testing.T) {
	if got := RepairToolArgs("read", nil, nil); got != nil {
		t.Fatalf("nil args: got %#v, want nil", got)
	}
	in := map[string]any{"file_path": "x"}
	got := RepairToolArgs("unknown_tool", nil, in)
	if got["file_path"] != "x" {
		t.Fatalf("unknown tool args altered: %v", got)
	}
}

func TestRepairToolArgsNoSelfAliasDeletion(t *testing.T) {
	// A regression guard: an alias entry that maps a name to itself must
	// never delete the value.
	in := map[string]any{"pattern": "foo"}
	got := RepairToolArgs("glob", nil, in)
	if got["pattern"] != "foo" {
		t.Fatalf("self-alias deleted value: %v", got)
	}
}
