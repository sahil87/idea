package main

import (
	"fmt"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func rmCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "rm <query>",
		Short: "Delete an idea from the backlog",
		Long: `Delete a matching idea from the current worktree's backlog.

<query> matches an idea (open or done) by its ID or by a case-insensitive
substring of its text. If it matches more than one idea it is refused and the
ambiguous matches are listed, so you can be more specific or use the exact ID.
--force is required to confirm the deletion; without it the command refuses to
remove anything. --main targets the main worktree's backlog, and
--file / IDEAS_FILE point elsewhere (see "idea --help").

  idea rm a7k2 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			i, backfilled, err := idea.Rm(path, args[0], force)
			if err != nil {
				return err
			}

			printBackfillNotice(cmd, backfilled)
			fmt.Printf("Removed: %s\n", idea.FormatLine(i))
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Confirm deletion")

	return cmd
}
