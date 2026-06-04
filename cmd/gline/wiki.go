package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/liup215/gline/internal/memory"
	"github.com/spf13/cobra"
)

var wikiCmd = &cobra.Command{
	Use:   "wiki",
	Short: "Wiki layer management",
	Long:  `Inspect and maintain the LLM-maintained markdown wiki for a knowledge base.`,
}

func init() {
	rootCmd.AddCommand(wikiCmd)
	wikiCmd.AddCommand(wikiShowCmd)
	wikiCmd.AddCommand(wikiLinksCmd)
	wikiCmd.AddCommand(wikiLintCmd)
	wikiCmd.AddCommand(wikiSyncCmd)
}

var wikiShowCmd = &cobra.Command{
	Use:   "show <kb-id> [page-path]",
	Short: "Display a wiki page (default: index.md)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		page := "index.md"
		if len(args) > 1 {
			page = args[1]
		}
		kbID := args[0]

		fs, err := memory.NewWikiFS(kbID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open wiki: %v\n", err)
			os.Exit(1)
		}
		content, err := fs.ReadPage(page)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Page not found: %s\n", page)
			os.Exit(1)
		}
		fmt.Println(content)
	},
}

var wikiLinksCmd = &cobra.Command{
	Use:   "links <kb-id> [page]",
	Short: "List wiki-link graph for a page",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		kbID := args[0]
		page := "index.md"
		if len(args) > 1 {
			page = args[1]
		}

		fs, err := memory.NewWikiFS(kbID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open wiki: %v\n", err)
			os.Exit(1)
		}
		content, err := fs.ReadPage(page)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Page not found: %s\n", page)
			os.Exit(1)
		}
		links := memory.ExtractLinks(content)
		if len(links) == 0 {
			fmt.Println("No outgoing links.")
			return
		}
		fmt.Printf("Links from %s:\n", page)
		for _, l := range links {
			fmt.Printf("  → [[%s]]\n", l)
		}
	},
}

var wikiLintCmd = &cobra.Command{
	Use:   "lint <kb-id>",
	Short: "Run wiki health checks (stub — Phase 3)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		kbID := args[0]
		fs, err := memory.NewWikiFS(kbID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open wiki: %v\n", err)
			os.Exit(1)
		}
		pages, err := fs.ListPages()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list pages: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Wiki pages found: %d\n", len(pages))
		var orphanCount int
		linkMap := make(map[string][]string)
		for _, p := range pages {
			content, _ := fs.ReadPage(p)
			links := memory.ExtractLinks(content)
			for _, l := range links {
				linkMap[l] = append(linkMap[l], p)
			}
		}
		for _, p := range pages {
			base := strings.TrimSuffix(p, ".md")
			if len(linkMap[base]) == 0 && p != "index.md" && p != "log.md" {
				orphanCount++
				fmt.Printf("  ⚠️  orphan: %s\n", p)
			}
		}
		if orphanCount == 0 {
			fmt.Println("✅ No orphan pages.")
		}
	},
}

var wikiSyncCmd = &cobra.Command{
	Use:   "sync <kb-id>",
	Short: "Re-ingest all raw documents into wiki (Phase 3)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Wiki sync is a no-op until Phase 3 (LLM-driven Ingest).")
		fmt.Println("For now, use `gline kb add` to index documents.")
	},
}
