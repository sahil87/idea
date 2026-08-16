package main

import (
	"fmt"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func doneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "done <query>...",
		Short: "Mark an idea as done",
		Long: `Mark matching open ideas as done in the current worktree's backlog.

Each <query> matches an open idea by its ID or by a case-insensitive substring
of its text. If a query matches more than one open idea it is refused and the
ambiguous matches are listed, so you can be more specific or use the exact ID.
Multiple queries resolve all-or-nothing: every query is resolved before any
change is written, so a failing query aborts the batch with the backlog
untouched. Two queries resolving to the same idea act on it once (a note on
stderr reports the dedupe); confirmations print one per acted idea. --main
targets the main worktree's backlog, --system targets the system-level
backlog (~/.config/idea/backlog.md), and --file / IDEAS_FILE point elsewhere
(see "idea --help"). Outside a git repo the system backlog is used
automatically.

  idea done a7k2
  idea done "dark mode"
  idea done a7k2 x9m1 auth-cleanup`,
		Args: usageArgs(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			acted, backfilled, err := idea.Done(path, args)
			if err != nil {
				return err
			}

			printBackfillNotice(cmd, backfilled)
			printDedupeNotice(cmd, len(args), len(acted))
			for _, i := range acted {
				fmt.Printf("Done: %s\n", idea.FormatLine(i))
			}
			return nil
		},
	}
}

// printDedupeNotice writes an advisory note to the command's stderr when some
// queries resolved to an already-selected idea and therefore acted once.
// stdout carries only per-item confirmation lines, so the note goes to stderr
// with the other advisory notices (see printBackfillNotice). With all-or-
// nothing resolution every query resolves to exactly one idea, so the dedupe
// count is simply queries minus acted.
func printDedupeNotice(cmd *cobra.Command, queries, acted int) {
	if dupes := queries - acted; dupes > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "note: %d duplicate query(ies) matched an already-selected idea; acted once\n", dupes)
	}
}
