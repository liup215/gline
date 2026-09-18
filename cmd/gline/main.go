package main

import (
	"flag"
	"fmt"
	"log"

	glineLog "github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/storage"
	"github.com/liup215/gline/internal/tui"
)

func main() {
	// Parse flags
	guiMode := flag.Bool("gui", false, "Launch GUI mode (Wails desktop app)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "gline - AI Programming Assistant\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  gline              Start TUI mode (default)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  gline --gui        Start GUI mode (Wails desktop app)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  gline chat         Single-prompt CLI mode\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  gline history      List conversation history\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Initialize configuration
	if err := InitConfig(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	if *guiMode {
		// GUI mode - original Wails app
		runGUI()
	} else {
		// TUI mode - default
		runTUI()
	}
}

// runTUI starts the TUI application.
func runTUI() {
	// Reinitialize logger without console output (interferes with TUI rendering)
	if err := glineLog.Init(glineLog.Config{
		Level:   "info",
		File:    "",
		Console: false,
		Color:   false,
	}); err != nil {
		log.Printf("Warning: Failed to reinitialize logger: %v", err)
	}

	ag, err := initializeAgent()
	if err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	// Initialize storage for task history
	store, err := storage.NewSQLiteStore("")
	if err != nil {
		log.Printf("Warning: Failed to initialize storage: %v", err)
	}

	if err := tui.Run(ag, store); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}
