package main

import (
	"errors"
	"fmt"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func promoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "promote <query>",
		Short: "Move an idea to the main worktree's backlog",
		Long: `Move a matching idea from the current worktree's backlog to the main
worktree's backlog — the retain step for ideas captured in an ephemeral linked
worktree.

The idea keeps its ID, date, and open/done status verbatim. The destination is
written first and the source second, so a crash mid-move duplicates the idea
rather than losing it; both files end in canonical form. If the main backlog
already holds an idea with the same ID the move is refused (the ID is never
re-minted — external references may key on it) and neither file is touched.
Run from the main worktree this is a no-op with a stderr note. --main and
--system are rejected: promote defines its own source (current worktree) and
destination (main worktree); --file / IDEAS_FILE apply within each root.
Outside a git repo there is no main worktree, so promote fails.

<query> matches an idea (open or done) by its ID or by a case-insensitive
substring of its text. If it matches more than one idea it is refused and the
ambiguous matches are listed, so you can be more specific or use the exact ID.

  idea promote a7k2
  idea promote "dark mode"`,
		Args: usageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Promote defines its own source and destination, so the root
			// selectors are a malformed invocation here (exit 2), classified
			// like the --system/--main conflict.
			if mainFlag {
				return &usageError{errors.New("--main cannot be used with promote: the destination is already the main worktree's backlog")}
			}
			if systemFlag {
				return &usageError{errors.New("--system cannot be used with promote: it moves ideas between repo worktrees, not the system backlog")}
			}

			// Source = current worktree root, destination = main worktree
			// root; --file / IDEAS_FILE apply within each. The destination
			// resolution is git-only, so outside a repo this fails
			// operationally with "not in a git repository".
			srcPath, err := idea.ResolveBacklogPath(false, false, fileFlag)
			if err != nil {
				return err
			}
			dstPath, err := idea.ResolveBacklogPath(false, true, fileFlag)
			if err != nil {
				return err
			}

			// Same file (main worktree, or an absolute --file collapse) is a
			// no-op with an advisory — the edit unchanged-buffer precedent.
			if srcPath == dstPath {
				fmt.Fprintln(cmd.ErrOrStderr(), "note: already in the main worktree — nothing to promote")
				return nil
			}

			i, srcBackfilled, dstBackfilled, err := idea.Promote(srcPath, dstPath, args[0])
			if err != nil {
				return err
			}

			printBackfillNotice(cmd, srcBackfilled+dstBackfilled)
			fmt.Printf("Promoted: %s\n", idea.FormatLine(i))
			return nil
		},
	}
}
