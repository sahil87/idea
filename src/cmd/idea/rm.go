package main

import (
	"fmt"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func rmCmd() *cobra.Command {
	var force, yes, dryRun bool

	cmd := &cobra.Command{
		Use:   "rm <query>...",
		Short: "Delete an idea from the backlog",
		Long: `Delete matching ideas from the current worktree's backlog.

Each <query> matches an idea (open or done) by its ID or by a case-insensitive
substring of its text. If a query matches more than one idea it is refused and
the ambiguous matches are listed, so you can be more specific or use the exact
ID. Multiple queries resolve all-or-nothing: every query is resolved before
any change is written, so a failing query aborts the batch with the backlog
untouched. Two queries resolving to the same idea act on it once (a note on
stderr reports the dedupe); confirmations print one per removed idea.
Confirmation is required to delete: pass --yes/-y (or the equivalent --force);
one consent flag covers the whole batch, and without consent the command
refuses to remove anything. --dry-run previews every idea that would be
deleted (resolved via the same match path) and writes nothing — no consent
needed, and it wins over --yes/--force. --main targets the main worktree's
backlog, --system targets the system-level backlog
(~/.config/idea/backlog.md), and --file / IDEAS_FILE point elsewhere (see
"idea --help"). Outside a git repo the system backlog is used automatically.

  idea rm a7k2 --yes
  idea rm a7k2 x9m1 auth-cleanup --yes
  idea rm a7k2 x9m1 --dry-run`,
		Args: usageArgs(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			// --dry-run is a non-destructive preview that shares the live match
			// path; it wins over any consent flag so it never deletes.
			if dryRun {
				matches, err := idea.RmPreview(path, args)
				if err != nil {
					return err
				}
				printDedupeNotice(cmd, len(args), len(matches))
				for _, i := range matches {
					fmt.Println(idea.FormatLine(i))
				}
				return nil
			}

			// --yes/-y and --force are equivalent consent (additive alias).
			removed, backfilled, err := idea.Rm(path, args, force || yes)
			if err != nil {
				return err
			}

			printBackfillNotice(cmd, backfilled)
			printDedupeNotice(cmd, len(args), len(removed))
			for _, i := range removed {
				fmt.Printf("Removed: %s\n", idea.FormatLine(i))
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm deletion (non-interactive consent)")
	cmd.Flags().BoolVar(&force, "force", false, "Confirm deletion (alias of --yes)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview the ideas that would be deleted; write nothing")

	return cmd
}
