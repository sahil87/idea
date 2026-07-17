package main

import (
	"fmt"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func rmCmd() *cobra.Command {
	var force, yes, dryRun bool

	cmd := &cobra.Command{
		Use:   "rm <query>",
		Short: "Delete an idea from the backlog",
		Long: `Delete a matching idea from the current worktree's backlog.

<query> matches an idea (open or done) by its ID or by a case-insensitive
substring of its text. If it matches more than one idea it is refused and the
ambiguous matches are listed, so you can be more specific or use the exact ID.
Confirmation is required to delete: pass --yes/-y (or the equivalent --force);
without consent the command refuses to remove anything. --dry-run previews the
idea that would be deleted (resolved via the same match path) and writes
nothing — no consent needed, and it wins over --yes/--force. --main targets the
main worktree's backlog, --system targets the system-level backlog
(~/.config/idea/backlog.md), and --file / IDEAS_FILE point elsewhere (see
"idea --help"). Outside a git repo the system backlog is used automatically.

  idea rm a7k2 --yes
  idea rm a7k2 --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			// --dry-run is a non-destructive preview that shares the live match
			// path; it wins over any consent flag so it never deletes.
			if dryRun {
				i, err := idea.RmPreview(path, args[0])
				if err != nil {
					return err
				}
				fmt.Println(idea.FormatLine(i))
				return nil
			}

			// --yes/-y and --force are equivalent consent (additive alias).
			i, backfilled, err := idea.Rm(path, args[0], force || yes)
			if err != nil {
				return err
			}

			printBackfillNotice(cmd, backfilled)
			fmt.Printf("Removed: %s\n", idea.FormatLine(i))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm deletion (non-interactive consent)")
	cmd.Flags().BoolVar(&force, "force", false, "Confirm deletion (alias of --yes)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview the idea that would be deleted; write nothing")

	return cmd
}
