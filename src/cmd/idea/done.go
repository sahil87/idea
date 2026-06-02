package main

import (
	"fmt"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func doneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "done <query>",
		Short: "Mark an idea as done",
		Long: `Mark a matching open idea as done in the current worktree's backlog.

<query> matches an open idea by its ID or by a case-insensitive substring of
its text. If the query matches more than one open idea it is refused and the
ambiguous matches are listed, so you can be more specific or use the exact ID.
--main targets the main worktree's backlog and --file / IDEAS_FILE point
elsewhere (see "idea --help").

  idea done a7k2
  idea done "dark mode"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			i, err := idea.Done(path, args[0])
			if err != nil {
				return err
			}

			fmt.Printf("Done: %s\n", idea.FormatLine(i))
			return nil
		},
	}
}
