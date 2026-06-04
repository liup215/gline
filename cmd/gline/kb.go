package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/memory"
	"github.com/spf13/cobra"
)

var kbCmd = &cobra.Command{
	Use:   "kb",
	Short: "Knowledge base management (RAG + Wiki)",
	Long:  `Create and manage knowledge bases supporting RAG retrieval, Wiki synthesis, and hybrid modes.`,
}

func init() {
	rootCmd.AddCommand(kbCmd)

	kbInitCmd.Flags().String("type", "hybrid", "KB type: rag | wiki | hybrid")
	kbCmd.AddCommand(kbInitCmd)
	kbCmd.AddCommand(kbListCmd)
	kbCmd.AddCommand(kbRemoveCmd)
	kbCmd.AddCommand(kbStatusCmd)
	kbCmd.AddCommand(kbAddCmd)
	kbCmd.AddCommand(kbSearchCmd)
}

var kbInitCmd = &cobra.Command{
	Use:   "init <name> [description]",
	Short: "Create a new knowledge base",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		desc := ""
		if len(args) > 1 {
			desc = strings.Join(args[1:], " ")
		}
		kbType := memory.KBType(cmd.Flag("type").Value.String())
		if kbType != memory.KBTypeRAG && kbType != memory.KBTypeWiki && kbType != memory.KBTypeHybrid {
			fmt.Fprintf(os.Stderr, "Invalid type: %s\n", kbType)
			os.Exit(1)
		}

		engine, err := newMemoryEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialise engine: %v\n", err)
			os.Exit(1)
		}
		defer engine.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		kb, err := engine.InitKB(ctx, name, desc, kbType)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create knowledge base: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Created knowledge base %q [%s]: %s\n", kb.Name, kb.Type, kb.ID)
	},
}

var kbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all knowledge bases",
	Run: func(cmd *cobra.Command, args []string) {
		engine, err := newMemoryEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialise engine: %v\n", err)
			os.Exit(1)
		}
		defer engine.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		list, err := engine.ListKB(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list: %v\n", err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println("No knowledge bases yet.")
			fmt.Println("Run: gline kb init <name> [--type rag|wiki|hybrid]")
			return
		}
		fmt.Printf("%-12s %-10s %-8s %-8s %-8s %-8s  %s\n", "ID", "Type", "Docs", "Chunks", "Facts", "Pages", "Name")
		fmt.Println(strings.Repeat("-", 70))
		for _, kb := range list {
			fmt.Printf("%-12s %-10s %-8d %-8d %-8d %-8d  %s\n",
				kb.ID, kb.Type, kb.DocCount, kb.ChunkCount, kb.FactCount, kb.WikiPageCount, kb.Name)
		}
	},
}

var kbRemoveCmd = &cobra.Command{
	Use:   "remove <id|name>",
	Short: "Delete a knowledge base and its data",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		engine, err := newMemoryEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialise engine: %v\n", err)
			os.Exit(1)
		}
		defer engine.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		idOrName := args[0]
		kb, err := engine.GetKB(ctx, idOrName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Knowledge base not found: %s\n", idOrName)
			os.Exit(1)
		}

		if err := engine.RemoveKB(ctx, kb.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Removed knowledge base %q and its data.\n", kb.Name)
	},
}

var kbStatusCmd = &cobra.Command{
	Use:   "status <id|name>",
	Short: "Show knowledge base details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		engine, err := newMemoryEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialise engine: %v\n", err)
			os.Exit(1)
		}
		defer engine.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		idOrName := args[0]
		kb, err := engine.GetKB(ctx, idOrName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Knowledge base not found: %s\n", idOrName)
			os.Exit(1)
		}

		fmt.Printf("ID:          %s\n", kb.ID)
		fmt.Printf("Name:        %s\n", kb.Name)
		fmt.Printf("Type:        %s\n", kb.Type)
		fmt.Printf("Description: %s\n", kb.Description)
		fmt.Printf("Docs:        %d\n", kb.DocCount)
		fmt.Printf("Chunks:      %d\n", kb.ChunkCount)
		fmt.Printf("Facts:       %d\n", kb.FactCount)
		fmt.Printf("Wiki Pages:  %d\n", kb.WikiPageCount)
		fmt.Printf("Created:     %s\n", kb.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("Path:        %s\n", memory.KBDir(kb.ID))
	},
}

var kbAddCmd = &cobra.Command{
	Use:   "add <id|name> <file/dir>...",
	Short: "Index files into a knowledge base",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		engine, err := newMemoryEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialise engine: %v\n", err)
			os.Exit(1)
		}
		defer engine.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		idOrName := args[0]
		kb, err := engine.GetKB(ctx, idOrName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Knowledge base not found: %s\n", idOrName)
			os.Exit(1)
		}

		var succeeded int
		for _, path := range args[1:] {
			info, err := os.Stat(path)
			if err != nil {
				log.Warnf("Skip %s: %v", path, err)
				continue
			}
			if info.IsDir() {
				// Recursively add supported files
				entries, err := os.ReadDir(path)
				if err != nil {
					log.Warnf("Skip %s: %v", path, err)
					continue
				}
				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}
					filePath := filepath.Join(path, entry.Name())
					if err := engine.IngestFile(ctx, kb.ID, filePath); err != nil {
						fmt.Fprintf(os.Stderr, "  ❌ %s: %v\n", entry.Name(), err)
					} else {
						fmt.Printf("  ✅ %s\n", entry.Name())
						succeeded++
					}
				}
			} else {
				if err := engine.IngestFile(ctx, kb.ID, path); err != nil {
					fmt.Fprintf(os.Stderr, "  ❌ %s: %v\n", filepath.Base(path), err)
				} else {
					fmt.Printf("  ✅ %s\n", filepath.Base(path))
					succeeded++
				}
			}
		}
		fmt.Printf("\nIndexed %d file(s) into %q.\n", succeeded, kb.Name)
	},
}

var kbSearchCmd = &cobra.Command{
	Use:   "search <id|name> <query>",
	Short: "Search a knowledge base",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		engine, err := newMemoryEngine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialise engine: %v\n", err)
			os.Exit(1)
		}
		defer engine.Close()

		ctxBg, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		idOrName := args[0]
		query := strings.Join(args[1:], " ")

		kb, err := engine.GetKB(ctxBg, idOrName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Knowledge base not found: %s\n", idOrName)
			os.Exit(1)
		}

		fmt.Printf("🔍 Searching %q for: %s\n\n", kb.Name, query)

		// RAG search
		fmt.Println("═══ RAG Results ═══")
		vecs, err := memory.EmbedAndNormalize(ctxBg, engine.Embedder, []string{query})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Embedding error: %v\n", err)
		} else {
			chunks, err := engine.RAGEngine.Search(ctxBg, kb.ID, vecs[0], query, 5, 0.0)
			if err != nil {
				fmt.Fprintf(os.Stderr, "RAG search error: %v\n", err)
			} else if len(chunks) == 0 {
				fmt.Println("No RAG results.")
			} else {
				for _, c := range chunks {
					fmt.Printf("  [%s] %s...\n", c.DocID, truncate(c.Content, 120))
				}
			}
		}

		// Fact search
		fmt.Println("\n═══ Fact Results ═══")
		facts, err := engine.FactStore.Search(ctxBg, query, memory.FactSearchOptions{TopK: 5})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fact search error: %v\n", err)
		} else if len(facts) == 0 {
			fmt.Println("No fact results.")
		} else {
			for _, f := range facts {
				fmt.Printf("  [%s] %s (conf=%.2f)\n", f.Category, f.Sentence(), f.Confidence)
			}
		}
	},
}

// newMemoryEngine creates a UnifiedEngine using the current config's embedding settings.
func newMemoryEngine() (*memory.UnifiedEngine, error) {
	cfg := configManager.Get()
	memCfg := cfg.Memory
	
	var embedder memory.Embedder
	
	switch memCfg.Embedding.Provider {
	case "ollama":
		embedder = memory.NewOllamaEmbedder(memCfg.Embedding.Model)
	case "openai":
		apiKey := memCfg.Embedding.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("GLINE_OPENAI_API_KEY")
		}
		if apiKey == "" {
			// Fall back to main provider OpenAI key
			apiKey = cfg.Provider.OpenAI.APIKey
		}
		if apiKey == "" {
			apiKey = os.Getenv("GLINE_OPENAI_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("no OpenAI API key: set memory.embedding.api_key, provider.openai.api_key, or GLINE_OPENAI_API_KEY")
		}
		baseURL := memCfg.Embedding.BaseURL
		if baseURL == "" {
			baseURL = cfg.Provider.OpenAI.BaseURL
		}
		emb := memory.NewOpenAIEmbedder(apiKey, memCfg.Embedding.Model)
		if baseURL != "" {
			emb.BaseURL = baseURL
		}
		embedder = emb
	default:
		return nil, fmt.Errorf("unknown embedding provider: %s", memCfg.Embedding.Provider)
	}
	
	return memory.NewUnifiedEngine(embedder)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
