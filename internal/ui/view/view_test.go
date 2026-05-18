package view

import (
	"regexp"
	"strings"
	"testing"

	"github.com/liup215/gline/internal/agent"
)

// stripANSI removes ANSI escape sequences from a string for test assertions.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestNormalizeToolName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"readFile", "read_file"},
		{"writeToFile", "write_to_file"},
		{"replaceInFile", "replace_in_file"},
		{"executeCommand", "execute_command"},
		{"read_file", "read_file"}, // already snake_case
		{"ABC", "abc"},             // all caps → just lowercased (no lowercase→uppercase boundary)
	}
	for _, tt := range tests {
		got := NormalizeToolName(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeToolName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGetToolDescription(t *testing.T) {
	if d := GetToolDescription("read_file"); d != "read this file" {
		t.Errorf("expected 'read this file', got %q", d)
	}
	if d := GetToolDescription("unknown_tool"); d != "used a tool" {
		t.Errorf("expected fallback 'used a tool', got %q", d)
	}
	// CamelCase should be normalized
	if d := GetToolDescription("ReadFile"); d != "read this file" {
		t.Errorf("expected 'read this file' after normalization, got %q", d)
	}
}

func TestGetToolMainArg(t *testing.T) {
	// Path argument
	if got := GetToolMainArg("read_file", `{"path":"/tmp/x"}`); got != "/tmp/x" {
		t.Errorf("expected /tmp/x, got %q", got)
	}
	// Command argument (truncation)
	longCmd := strings.Repeat("a", 150)
	input := `{"command":"` + longCmd + `"}`
	got := GetToolMainArg("execute_command", input)
	if len(got) != 120 { // 117 + "..."
		t.Errorf("expected truncated to 120 chars, got %d: %q", len(got), got)
	}
	// Empty input
	if got := GetToolMainArg("read_file", ""); got != "" {
		t.Errorf("expected empty for empty input, got %q", got)
	}
	// Invalid JSON
	if got := GetToolMainArg("read_file", "not json"); got != "" {
		t.Errorf("expected empty for invalid JSON, got %q", got)
	}
	// Regex + path
	if got := GetToolMainArg("search_files", `{"regex":"TODO","path":"."}`); got != "'TODO' in ." {
		t.Errorf("expected 'TODO in .', got %q", got)
	}
}

func TestFormatToolResultLines(t *testing.T) {
	// Under limit
	lines := FormatToolResultLines("a\nb\nc", 5)
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	// Over limit
	input := "a\nb\nc\nd\ne\nf\ng"
	lines = FormatToolResultLines(input, 3)
	if len(lines) != 4 { // 3 display + 1 "more" footer
		t.Errorf("expected 4 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[3], "4 more lines") {
		t.Errorf("expected '4 more lines' footer, got %q", lines[3])
	}
}

func TestRenderStatusBar(t *testing.T) {
	// Basic rendering — ACT mode
	s := stripANSI(RenderStatusBar(StatusBarData{
		Mode:      agent.ModeAct,
		Provider:  "openai",
		ModelName: "gpt-4",
		Width:     80,
	}))
	if !strings.Contains(s, "ACT") {
		t.Errorf("status bar missing ACT mode: %q", s)
	}
	if !strings.Contains(s, "openai") {
		t.Errorf("status bar missing provider: %q", s)
	}

	// PLAN mode
	s = stripANSI(RenderStatusBar(StatusBarData{
		Mode:      agent.ModePlan,
		Provider:  "anthropic",
		ModelName: "claude",
		Width:     80,
	}))
	if !strings.Contains(s, "PLAN") {
		t.Errorf("status bar missing PLAN mode: %q", s)
	}

	// Processing + streaming
	s = stripANSI(RenderStatusBar(StatusBarData{
		Mode:         agent.ModeAct,
		Provider:     "openai",
		ModelName:    "gpt-4",
		IsProcessing: true,
		IsStreaming:   true,
		SpinnerView:  "⠋",
		Width:        80,
	}))
	if !strings.Contains(s, "AI is responding") {
		t.Errorf("status bar missing streaming text: %q", s)
	}

	// Processing + tool running
	s = stripANSI(RenderStatusBar(StatusBarData{
		Mode:         agent.ModeAct,
		Provider:     "openai",
		ModelName:    "gpt-4",
		IsProcessing: true,
		CurrentTool:  "read_file",
		SpinnerView:  "⠋",
		Width:        80,
	}))
	if !strings.Contains(s, "Running: read_file") {
		t.Errorf("status bar missing running tool: %q", s)
	}

	// Empty provider/model defaults to "-"
	s = stripANSI(RenderStatusBar(StatusBarData{
		Mode:  agent.ModeAct,
		Width: 80,
	}))
	if !strings.Contains(s, "-") {
		t.Errorf("status bar missing default dash: %q", s)
	}
}

func TestRenderHeader(t *testing.T) {
	s := stripANSI(RenderHeader(HeaderData{
		Mode:      agent.ModeAct,
		Provider:  "openai",
		ModelName: "gpt-4",
	}))
	if !strings.Contains(s, "gline") {
		t.Errorf("header missing 'gline': %q", s)
	}
	if !strings.Contains(s, "ACT") {
		t.Errorf("header missing ACT: %q", s)
	}
	if !strings.Contains(s, "openai") {
		t.Errorf("header missing provider: %q", s)
	}
}

func TestRenderHelp(t *testing.T) {
	s := stripANSI(RenderHelp())
	if !strings.Contains(s, "enter: send") {
		t.Errorf("help missing 'enter: send': %q", s)
	}
	if !strings.Contains(s, "ctrl+c: quit") {
		t.Errorf("help missing 'ctrl+c: quit': %q", s)
	}
}

func TestRenderToolArea(t *testing.T) {
	content := "test content"
	if got := RenderToolArea(content); got != content {
		t.Errorf("RenderToolArea should pass through content, got %q", got)
	}
}

func TestRenderInputBox(t *testing.T) {
	s := RenderInputBox("hello")
	if s == "" {
		t.Error("RenderInputBox returned empty string")
	}
	// Should contain the input text (even with ANSI codes stripped)
	if !strings.Contains(stripANSI(s), "hello") {
		t.Errorf("RenderInputBox missing input text: %q", stripANSI(s))
	}
}

func TestRenderLayout(t *testing.T) {
	s := stripANSI(RenderLayout(LayoutData{
		CompactBar:     "COMPACT",
		Content:        "CONTENT",
		InputView:      "INPUT",
		InputStatusBar: "STATUS",
		Help:           "HELP",
		Height:         24,
		InputHeight:    3,
	}))
	if !strings.Contains(s, "COMPACT") {
		t.Errorf("layout missing COMPACT: %q", s)
	}
	if !strings.Contains(s, "CONTENT") {
		t.Errorf("layout missing CONTENT: %q", s)
	}
	if !strings.Contains(s, "HELP") {
		t.Errorf("layout missing HELP: %q", s)
	}
}

// ---------------------------------------------------------------------------
// Phase 8: Tool formatting function tests
// ---------------------------------------------------------------------------

func TestFormatToolStartDisplay(t *testing.T) {
	tests := []struct {
		name, toolName, input, wantSub string
	}{
		{
			name:     "empty input",
			toolName: "read_file",
			input:    "",
			wantSub:  "🔧 read this file",
		},
		{
			name:     "with main arg",
			toolName: "read_file",
			input:    `{"path": "/tmp/x"}`,
			wantSub:  "🔧 read this file: /tmp/x",
		},
		{
			name:     "attempt_completion keeps full input",
			toolName: "attempt_completion",
			input:    `{"result": "Task done"}`,
			wantSub:  "🔧 completed the task\n\n{\"result\": \"Task done\"}",
		},
		{
			name:     "unknown tool falls back to description",
			toolName: "unknown_tool",
			input:    `{"foo": "bar"}`,
			wantSub:  "🔧 used a tool",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatToolStartDisplay(tt.toolName, tt.input)
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("FormatToolStartDisplay(%q, %q) = %q, want substring %q", tt.toolName, tt.input, got, tt.wantSub)
			}
		})
	}
}

func TestFormatAttemptCompletionContent(t *testing.T) {
	tests := []struct {
		name, input, wantSub string
	}{
		{
			name:     "string result",
			input:    `{"result": "Task completed successfully"}`,
			wantSub:  "Task completed successfully",
		},
		{
			name:     "string content",
			input:    `{"content": "Some content"}`,
			wantSub:  "Some content",
		},
		{
			name:     "object result renders as JSON code block",
			input:    `{"result": {"key": "value"}}`,
			wantSub:  "```json",
		},
		{
			name:     "invalid JSON returns raw input",
			input:    `not json`,
			wantSub:  "not json",
		},
		{
			name:     "empty object falls back to JSON code block",
			input:    `{"other": "data"}`,
			wantSub:  "```json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAttemptCompletionContent(tt.input)
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("FormatAttemptCompletionContent(%q) = %q, want substring %q", tt.input, got, tt.wantSub)
			}
		})
	}
}

func TestFormatToolCompleteDisplay(t *testing.T) {
	tests := []struct {
		name, toolName, result, status, wantSub string
	}{
		{
			name:     "completed with result",
			toolName: "read_file",
			result:   "file content",
			status:   "completed",
			wantSub:  "🔧 Completed: read_file",
		},
		{
			name:     "failed status",
			toolName: "execute_command",
			result:   "error occurred",
			status:   "failed",
			wantSub:  "🔧 Failed: execute_command",
		},
		{
			name:     "empty result",
			toolName: "search_files",
			result:   "",
			status:   "completed",
			wantSub:  "🔧 Completed: search_files",
		},
		{
			name:     "result with truncated lines",
			toolName: "read_file",
			result:   "line1\nline2\nline3\nline4\nline5\nline6\nline7",
			status:   "completed",
			wantSub:  "2 more lines",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatToolCompleteDisplay(tt.toolName, tt.result, tt.status)
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("FormatToolCompleteDisplay(%q, %q, %q) = %q, want substring %q", tt.toolName, tt.result, tt.status, got, tt.wantSub)
			}
		})
	}
}
