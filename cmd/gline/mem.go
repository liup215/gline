package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/liup215/gline/internal/memory"
	"github.com/spf13/cobra"
)

var memCmd = &cobra.Command{
	Use:   "mem",
	Short: "Fact and memory layer commands",
	Long:  `Query, manage and inspect the mem0-style semantic fact layer.`,
}

func init() {
	rootCmd.AddCommand(memCmd)
	memCmd.AddCommand(memFactsCmd)
	memCmd.AddCommand(memRecallCmd)
	memCmd.AddCommand(memDecayCmd)
}

var memFactsCmd = &cobra.Command{
	Use:   "facts [query]",
	Short: "List or search facts",
	Run: func(cmd *cobra.Command, args []string) {
		engine, err := newMemoryEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialise engine: %v\n", err)
			os.Exit(1)
		}
		defer engine.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		query := ""
		if len(args) > 0 {
			query = args[0]
		}

		entity, _ := cmd.Flags().GetString("entity")
		cat, _ := cmd.Flags().GetString("category")

		opts := memory.FactSearchOptions{TopK: 20}
		if entity != "" {
			opts.Entities = []string{entity}
		}
		if cat != "" {
			opts.Categories = []memory.FactCategory{memory.FactCategory(cat)}
		}

		facts, err := engine.FactStore.Search(ctx, query, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Search failed: %v\n", err)
			os.Exit(1)
		}
		if len(facts) == 0 {
			fmt.Println("No facts found.")
			return
		}
		for _, f := range facts {
			fmt.Printf("[%s] %s  (conf=%.2f, id=%s)\n", f.Category, f.Sentence(), f.Confidence, f.ID)
		}
	},
}

var memRecallCmd = &cobra.Command{
	Use:   "recall <query>",
	Short: "Cross-layer recall: facts + RAG chunks (Phase 5)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		fmt.Printf("Recall query: %s\n", query)
		fmt.Println("Cross-layer unified recall will be fully implemented in Phase 5.")
		fmt.Println("For now, use `gline kb search` for RAG and `gline mem facts` for facts.")
	},
}

var memDecayCmd = &cobra.Command{
	Use:   "decay",
	Short: "Trigger fact confidence decay",
	Run: func(cmd *cobra.Command, args []string) {
		engine, err := newMemoryEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialise engine: %v\n", err)
			os.Exit(1)
		}
		defer engine.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := engine.FactStore.Decay(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Decay failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Fact decay applied.")
	},
}
