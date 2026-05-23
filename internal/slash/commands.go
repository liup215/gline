package slash

import (
	"fmt"
	"strings"

	"github.com/liup215/gline/pkg/types"
)

// CommandResult indicates what action the TUI should take after a command executes.
type CommandResult int

const (
	ResultNone CommandResult = iota
	ResultClearScreen
	ResultQuit
	ResultNewTask
	ResultCompact
	ResultShowHelp
	ResultShowHistory
)

// CommandContext holds the dependencies needed by slash command handlers.
type CommandContext struct {
	// Conversation holds the current conversation state.
	Conversation interface {
		Clear()
	}

	// OnResult is called when a command produces a TUI-level action.
	OnResult func(result CommandResult, message string)
}

// DefaultCommands returns the built-in slash commands for gline.
func DefaultCommands(ctx *CommandContext) []*types.SlashCommand {
	return []*types.SlashCommand{
		{
			Name:        "clear",
			Description: "Clear the current conversation and start fresh",
			Section:     types.SectionDefault,
			Handler: func(args string) (bool, error) {
				if ctx != nil && ctx.Conversation != nil {
					ctx.Conversation.Clear()
				}
				if ctx != nil && ctx.OnResult != nil {
					ctx.OnResult(ResultClearScreen, "Conversation cleared")
				}
				return true, nil
			},
		},
		{
			Name:        "help",
			Description: "Show available slash commands and shortcuts",
			Section:     types.SectionDefault,
			Handler: func(args string) (bool, error) {
				if ctx != nil && ctx.OnResult != nil {
					ctx.OnResult(ResultShowHelp, buildHelpText())
				}
				return true, nil
			},
		},
		{
			Name:        "exit",
			Description: "Exit gline (alternative to Ctrl+C)",
			Section:     types.SectionDefault,
			Handler: func(args string) (bool, error) {
				if ctx != nil && ctx.OnResult != nil {
					ctx.OnResult(ResultQuit, "Goodbye!")
				}
				return true, nil
			},
		},
		{
			Name:        "q",
			Description: "Exit gline (shorthand for /exit)",
			Section:     types.SectionDefault,
			Handler: func(args string) (bool, error) {
				if ctx != nil && ctx.OnResult != nil {
					ctx.OnResult(ResultQuit, "Goodbye!")
				}
				return true, nil
			},
		},
		{
			Name:        "newtask",
			Description: "Start a new task while preserving system context",
			Section:     types.SectionDefault,
			Handler: func(args string) (bool, error) {
				msg := "Starting new task"
				if args != "" {
					msg = fmt.Sprintf("Starting new task: %s", args)
				}
				if ctx != nil && ctx.OnResult != nil {
					ctx.OnResult(ResultNewTask, msg)
				}
				return true, nil
			},
		},
		{
			Name:        "smol",
			Description: "Compact the conversation context to save tokens",
			Section:     types.SectionDefault,
			Handler: func(args string) (bool, error) {
				if ctx != nil && ctx.OnResult != nil {
					ctx.OnResult(ResultCompact, "Conversation compacted")
				}
				return true, nil
			},
		},
		{
			Name:        "compact",
			Description: "Alias for /smol - compact the conversation context",
			Section:     types.SectionDefault,
			Handler: func(args string) (bool, error) {
				if ctx != nil && ctx.OnResult != nil {
					ctx.OnResult(ResultCompact, "Conversation compacted")
				}
				return true, nil
			},
		},
		{
			Name:        "history",
			Description: "Show conversation history and load previous tasks",
			Section:     types.SectionDefault,
			Handler: func(args string) (bool, error) {
				if ctx != nil && ctx.OnResult != nil {
					ctx.OnResult(ResultShowHistory, "")
				}
				return true, nil
			},
		},
	}
}

func buildHelpText() string {
	var b strings.Builder
	b.WriteString("Available slash commands:\n\n")
	commands := []struct {
		name, desc string
	}{
		{"/clear", "Clear the current conversation"},
		{"/help", "Show this help message"},
		{"/exit or /q", "Exit gline"},
		{"/newtask [name]", "Start a new task (preserves system context)"},
		{"/smol or /compact", "Compact conversation to save tokens"},
		{"/history", "Show conversation history"},
	}
	for _, c := range commands {
		b.WriteString(fmt.Sprintf("  %-18s %s\n", c.name, c.desc))
	}
	b.WriteString("\nShortcuts:\n")
	b.WriteString("  Tab          Toggle Plan/Act mode\n")
	b.WriteString("  Ctrl+L       Clear screen\n")
	b.WriteString("  Ctrl+C       Quit\n")
	b.WriteString("  Esc          Interrupt running task\n")
	b.WriteString("  Ctrl+H       Show history\n")
	return b.String()
}
