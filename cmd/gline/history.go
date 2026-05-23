package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/liup215/gline/internal/storage"
	"github.com/spf13/cobra"
)

// historyCmd represents the history command
var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Manage task history",
	Long: `View and manage gline conversation history.

History is stored in ~/.gline/gline.db and includes:
  - Task metadata (title, mode, provider, model, status)
  - Message records (user, assistant, tool messages)
  - Tool call records with inputs and outputs`,
}

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.AddCommand(historyListCmd)
	historyCmd.AddCommand(historyShowCmd)
	historyCmd.AddCommand(historyDeleteCmd)
}

var historyListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List recent tasks",
	Long:    `List the most recent conversation tasks with pagination.`,
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := strconv.Atoi(cmd.Flags().Lookup("limit").Value.String())
		offset, _ := strconv.Atoi(cmd.Flags().Lookup("offset").Value.String())

		store, err := storage.NewSQLiteStore("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to initialize storage: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		tasks, err := store.ListTasks(limit, offset)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to list tasks: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(storage.FormatTaskList(tasks, true))
	},
}

var historyShowCmd = &cobra.Command{
	Use:     "show <task-id>",
	Aliases: []string{"get"},
	Short:   "Show details of a task",
	Long:    `Display a task's metadata and message history.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Error: task ID is required")
			cmd.Usage()
			os.Exit(1)
		}

		store, err := storage.NewSQLiteStore("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to initialize storage: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		taskID := args[0]
		task, msgs, err := store.GetTaskSummary(taskID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get task: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(storage.FormatTaskDetail(task, msgs))
	},
}

var historyDeleteCmd = &cobra.Command{
	Use:     "delete <task-id>",
	Aliases: []string{"rm"},
	Short:   "Delete a task from history",
	Long:    `Remove a task and all associated messages and tool calls from history.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Error: task ID is required")
			cmd.Usage()
			os.Exit(1)
		}

		store, err := storage.NewSQLiteStore("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to initialize storage: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		taskID := args[0]
		task, err := store.GetTaskByID(taskID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get task: %v\n", err)
			os.Exit(1)
		}
		if task == nil {
			fmt.Fprintf(os.Stderr, "Error: task not found: %s\n", taskID)
			os.Exit(1)
		}

		confirm := cmd.Flags().Lookup("yes").Value.String() == "true"
		if !confirm {
			fmt.Printf("Delete task '%s' (%s)? [y/N] ", task.Title, taskID)
			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" {
				fmt.Println("Cancelled.")
				os.Exit(0)
			}
		}

		if err := store.DeleteTask(taskID); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to delete task: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Task deleted.")
	},
}

func init() {
	historyListCmd.Flags().IntP("limit", "l", 20, "Maximum number of tasks to show")
	historyListCmd.Flags().IntP("offset", "o", 0, "Offset for pagination")
	historyDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}
