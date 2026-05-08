package main

import (
	"fmt"
	"os"

	"github.com/liup215/gline/internal/version"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version.String())
		os.Exit(0)
	}

	if len(os.Args) > 1 && os.Args[1] == "--help" {
		printHelp()
		os.Exit(0)
	}

	fmt.Println("gline - AI programming assistant CLI tool")
	fmt.Println("Run with --help for usage information")
}

func printHelp() {
	fmt.Println("gline - AI programming assistant CLI tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gline [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --version    Show version information")
	fmt.Println("  --help       Show this help message")
	fmt.Println()
	fmt.Println("Version:", version.String())
}
