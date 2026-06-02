package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	var all, done, jsonOut, reverse bool
	var sortField string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List ideas from the backlog",
		Long: `List ideas from the current worktree's backlog (fab/backlog.md).

Open ideas are shown by default. Use --all/-a to include done ideas, or --done
to show only completed ones. --json emits the structured records (id, date,
status, text) for piping into other tools. --sort accepts "date" (default) or
"id", and --reverse flips the order. As with every backlog command, --main
targets the main worktree's backlog and --file / IDEAS_FILE point elsewhere
(see "idea --help").

  idea list
  idea list --all --sort id
  idea list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
				if jsonOut {
					fmt.Println("[]")
				} else {
					fmt.Println("No ideas file yet. Add one with: idea add \"your idea\"")
				}
				return nil
			}

			filter := idea.FilterOpen
			if all {
				filter = idea.FilterAll
			} else if done {
				filter = idea.FilterDone
			}

			ideas, err := idea.List(path, filter, sortField, reverse)
			if err != nil {
				return err
			}

			if len(ideas) == 0 {
				if jsonOut {
					fmt.Println("[]")
				} else {
					fmt.Println("No ideas found.")
				}
				return nil
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(ideas)
			}

			for _, i := range ideas {
				fmt.Println(idea.FormatLine(i))
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show all ideas (open + done)")
	cmd.Flags().BoolVar(&done, "done", false, "Show only done ideas")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&sortField, "sort", "date", "Sort by field (id or date)")
	cmd.Flags().BoolVar(&reverse, "reverse", false, "Reverse sort order")

	return cmd
}
