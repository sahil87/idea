package main

import (
	"fmt"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func pruneCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Bulk-remove all done ideas from the backlog",
		Long: `Bulk-remove all done ([x]) ideas from the current worktree's backlog.

Without --force this is a free dry run: each done idea that would be removed
is listed on stdout and nothing is written. --force performs the deletion and
prints only a count. There is no archive — the backlog is committed, so git
history is the recovery path. --main targets the main worktree's backlog, and
--file / IDEAS_FILE point elsewhere (see "idea --help").

  idea prune
  idea prune --force`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			pruned, backfilled, err := idea.Prune(path, force)
			if err != nil {
				return err
			}

			if len(pruned) == 0 {
				fmt.Println("No done ideas to prune.")
				return nil
			}

			if !force {
				// Free dry run: stdout carries exactly the removable lines
				// (pipe-friendly); the confirm hint is advisory, so it goes
				// to stderr like the backfill notice (Constitution VI).
				for _, i := range pruned {
					fmt.Println(idea.FormatLine(i))
				}
				fmt.Fprintln(cmd.ErrOrStderr(), "Re-run with --force to confirm.")
				return nil
			}

			printBackfillNotice(cmd, backfilled)
			fmt.Printf("Pruned %d done idea(s).\n", len(pruned))
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Confirm deletion of all done ideas")

	return cmd
}
