package slash

import "testing"

func TestIsStandaloneCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"/clear", true},
		{"/exit", true},
		{"/newtask hello", true},
		{"/smol", true},
		{"/q", true},
		{"/help", true},
		{"/settings api", true},
		{"/ not_a_command", false},  // space before slash
		{"hello /clear", false},     // not at start
		{"text", false},             // no slash
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsStandaloneCommand(tt.input)
			if got != tt.expected {
				t.Errorf("IsStandaloneCommand(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input         string
		wantName      string
		wantArgs      string
	}{
		{"/clear", "clear", ""},
		{"/exit", "exit", ""},
		{"/newtask hello world", "newtask", "hello world"},
		{"/newtask  hello", "newtask", "hello"},
		{"/smol", "smol", ""},
		{"not a command", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, args := ParseCommand(tt.input)
			if name != tt.wantName {
				t.Errorf("ParseCommand(%q) name = %q, want %q", tt.input, name, tt.wantName)
			}
			if args != tt.wantArgs {
				t.Errorf("ParseCommand(%q) args = %q, want %q", tt.input, args, tt.wantArgs)
			}
		})
	}
}
