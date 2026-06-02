package main

import (
	"fmt"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func editCmd() *cobra.Command {
	var newID, newDate string

	cmd := &cobra.Command{
		Use:   "edit <query> <new-text>",
		Short: "Modify an idea's text",
		Long: `Replace a matching idea's text in the current worktree's backlog.

<query> matches an idea (open or done) by its ID or by a case-insensitive
substring of its text. If it matches more than one idea it is refused and the
ambiguous matches are listed, so you can be more specific or use the exact ID.
--id and --date additionally change the matched idea's ID or date. --main
targets the main worktree's backlog and --file / IDEAS_FILE point elsewhere
(see "idea --help").

  idea edit a7k2 "wire up dark mode toggle"
  idea edit a7k2 --date 2026-06-01 "backdated rewrite"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			i, err := idea.Edit(path, args[0], args[1], newID, newDate)
			if err != nil {
				return err
			}

			fmt.Printf("Updated: %s\n", idea.FormatLine(i))
			return nil
		},
	}

	cmd.Flags().StringVar(&newID, "id", "", "Change the idea's ID")
	cmd.Flags().StringVar(&newDate, "date", "", "Change the idea's date")

	return cmd
}
