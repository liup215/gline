package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/storage"
)

// Run starts the TUI application with the given agent and store.
func Run(ag agent.Agent, store storage.Store) error {
	m := NewApp(ag, store)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Set the program reference so callbacks can send messages
	m.SetProgram(p)

	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	log.Info("TUI exited")
	return nil
}
