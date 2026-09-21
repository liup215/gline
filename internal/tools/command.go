package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/liup215/gline/internal/shell"
)

// ExecuteCommandTool executes a CLI command
type ExecuteCommandTool struct {
	BaseTool
}

// ExecuteCommandInput represents the input for execute_command tool
type ExecuteCommandInput struct {
	Command string `json:"command"`
	Cwd     string `json:"cwd,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

// ExecuteCommandOutput represents the output of execute_command tool
type ExecuteCommandOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Duration int64  `json:"duration_ms"`
}

// NewExecuteCommandTool creates a new execute_command tool.
//
// The description bakes in the shell resolved at construction time (pi's
// lesson: only the description tells the model which syntax to write —
// a "bash when available" hedge leaves it guessing between &&-chaining
// and &-chaining on every call).
func NewExecuteCommandTool() *ExecuteCommandTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The shell command to execute"
			},
			"cwd": {
				"type": "string",
				"description": "Working directory for the command (optional)"
			},
			"timeout": {
				"type": "integer",
				"description": "Timeout in SECONDS (default 60). Write 300 for five minutes, not 300000.",
				"default": 60
			}
		},
		"required": ["command"]
	}`)

	sh := shell.Resolve()
	var syntax string
	switch sh.Name() {
	case "bash":
		syntax = "Commands run under bash: POSIX syntax works (&& chains, $() substitution, $VAR variables, test -f)."
	case "cmd":
		syntax = "This machine has no bash; commands run through cmd.exe. Write cmd syntax: & or a newline instead of &&, %VAR% instead of $VAR, if exist instead of test -f. Native tools (git, go, npm) work as usual."
	default: // sh (Unix without bash)
		syntax = "Commands run under sh: POSIX syntax works (&& chains, $() substitution, $VAR variables)."
	}

	return &ExecuteCommandTool{
		BaseTool: BaseTool{
			name: "run",
			description: fmt.Sprintf(
				"Execute a shell command and return its combined output. %s "+
					"Use it for builds, tests, git, and other system operations. Do NOT use it to read files (use read) "+
					"or search code (use grep/glob). A command that outlives its timeout is killed and reports the "+
					"partial output with a timeout note — raise timeout for work you already know is long, like a full "+
					"test suite. timeout is in SECONDS (default 60): write 300 for five minutes, not 300000. "+
					"cwd runs the command in that directory.", syntax),
			inputSchema: schema,
		},
	}
}

// Execute runs the command
func (t *ExecuteCommandTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req ExecuteCommandInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}

	if req.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	// Set default timeout
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 60
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Prepare command: prefer bash (POSIX semantics) and fall back to
	// cmd.exe on Windows machines without a usable bash.
	sh := shell.Resolve()
	args := make([]string, 0, len(sh.Args)+1)
	args = append(args, sh.Args...)
	args = append(args, req.Command)
	cmd := exec.CommandContext(ctx, sh.Path, args...)
	hideConsole(cmd)

	// Set working directory if specified
	if req.Cwd != "" {
		cmd.Dir = req.Cwd
	} else {
		cmd.Dir = "."
	}

	// Set environment
	cmd.Env = os.Environ()

	// Capture output
	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start).Milliseconds()

	// Prepare result
	result := ExecuteCommandOutput{
		Duration: duration,
	}

	// Parse output
	outputStr := string(output)
	if idx := strings.LastIndex(outputStr, "\nexit status "); idx != -1 {
		// Try to parse exit code from combined output
		result.Stdout = strings.TrimSpace(outputStr[:idx])
	} else {
		result.Stdout = strings.TrimSpace(outputStr)
	}

	// Get exit code
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	// Handle errors
	if ctx.Err() == context.DeadlineExceeded {
		result.Stderr = fmt.Sprintf("Command timed out after %d seconds", timeout)
		result.ExitCode = -1
	} else if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Stderr = string(exitErr.Stderr)
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Stderr = err.Error()
			result.ExitCode = -1
		}
	}

	// Format output
	var outputBuilder strings.Builder
	outputBuilder.WriteString(fmt.Sprintf("Command: %s\n", req.Command))
	outputBuilder.WriteString(fmt.Sprintf("Exit code: %d\n", result.ExitCode))
	outputBuilder.WriteString(fmt.Sprintf("Duration: %dms\n", result.Duration))
	outputBuilder.WriteString("\n")

	if result.Stdout != "" {
		outputBuilder.WriteString(fmt.Sprintf("Output:\n%s\n", result.Stdout))
	}

	if result.Stderr != "" {
		outputBuilder.WriteString(fmt.Sprintf("\nErrors:\n%s\n", result.Stderr))
	}

	return outputBuilder.String(), nil
}

// IsDestructiveCommand checks if a command might be destructive
func IsDestructiveCommand(command string) bool {
	destructivePatterns := []string{
		"rm ", "del ", "rmdir ", "rd ",
		"mv ", "move ", "ren ",
		"cp -r", "xcopy /s",
		"git reset --hard",
		"git clean -f",
		"docker rm", "docker rmi",
		"kubectl delete",
	}

	lowerCmd := strings.ToLower(command)
	for _, pattern := range destructivePatterns {
		if strings.Contains(lowerCmd, pattern) {
			return true
		}
	}
	return false
}

// IsNetworkCommand checks if a command involves network operations
func IsNetworkCommand(command string) bool {
	networkPatterns := []string{
		"curl", "wget", "http", "https://",
		"git clone", "git fetch", "git pull",
		"npm install", "pip install", "go get",
		"docker pull", "docker push",
	}

	lowerCmd := strings.ToLower(command)
	for _, pattern := range networkPatterns {
		if strings.Contains(lowerCmd, pattern) {
			return true
		}
	}
	return false
}
