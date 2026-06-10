package main

import (
	"fmt"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func reopenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reopen <query>",
		Short: "Reopen a completed idea",
		Long: `Reopen a matching done idea in the current worktree's backlog.

<query> matches a done idea by its ID or by a case-insensitive substring of
its text. If the query matches more than one done idea it is refused and the
ambiguous matches are listed, so you can be more specific or use the exact ID.
--main targets the main worktree's backlog and --file / IDEAS_FILE point
elsewhere (see "idea --help").

  idea reopen a7k2
  idea reopen "dark mode"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			i, backfilled, err := idea.Reopen(path, args[0])
			if err != nil {
				return err
			}

			printBackfillNotice(cmd, backfilled)
			fmt.Printf("Reopened: %s\n", idea.FormatLine(i))
			return nil
		},
	}
}
