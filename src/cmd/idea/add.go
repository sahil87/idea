package main

import (
	"fmt"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func addCmd() *cobra.Command {
	var customID, customDate string

	cmd := &cobra.Command{
		Use:   "add <text>",
		Short: "Add a new idea to the backlog",
		Long: `Add a new idea to the current worktree's backlog (fab/backlog.md).

The idea is appended as a Markdown checklist line with a generated 4-char ID
and today's date. Use --id and --date to override those generated values
(handy when importing or backdating). By default the command writes the
current worktree's backlog; --main targets the main worktree's backlog,
--system targets the system-level backlog (~/.config/idea/backlog.md), and
--file / IDEAS_FILE point at a different file (see "idea --help"). Outside a
git repo the system backlog is used automatically.

  idea add "wire up dark mode"
  idea add --id a7k2 --date 2026-06-01 "backdated idea"`,
		Args: usageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}
			i, err := idea.Add(path, args[0], customID, customDate)
			if err != nil {
				return err
			}
			// Print the escaped single-line form: stdout stays one
			// machine-parseable line even for multiline idea text.
			fmt.Printf("Added: [%s] %s: %s\n", i.ID, i.Date, idea.EscapeText(i.Text))
			return nil
		},
	}

	cmd.Flags().StringVar(&customID, "id", "", "Custom 4-char ID")
	cmd.Flags().StringVar(&customDate, "date", "", "Custom date (YYYY-MM-DD)")

	return cmd
}
