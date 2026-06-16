package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	var all, done, jsonOut, reverse, full bool
	var sortField string

	cmd := &cobra.Command{
		Use:     "list [id...]",
		Aliases: []string{"ls"},
		Short:   "List ideas from the backlog",
		Long: `List ideas from the current worktree's backlog (fab/backlog.md).

Open ideas are shown by default. Use --all/-a to include done ideas, or --done
to show only completed ones. Pass one or more 4-char IDs to list only those
ideas. --json emits the structured records (id, date, status, text) for piping
into other tools. --sort accepts "date" (default) or "id", and --reverse flips
the order.

On a terminal, long idea text is truncated to fit the width (the [id] date:
prefix is never clipped) and the prefix is dimmed; --full shows the complete
text. When the output is piped or redirected, full canonical lines are emitted
regardless of --full so downstream tools see machine-parseable records. As with
every backlog command, --main targets the main worktree's backlog, --system
targets the system-level backlog (~/.config/idea/backlog.md), and
--file / IDEAS_FILE point elsewhere (see "idea --help"). Outside a git repo the
system backlog is used automatically.

  idea list
  idea list --all --sort id
  idea ls a7k2 b3c9 --full
  idea list --json`,
		Args: func(cmd *cobra.Command, args []string) error {
			for _, a := range args {
				if err := idea.ValidateID(a); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
				if jsonOut {
					fmt.Println("[]")
				} else {
					fmt.Println("No ideas file yet. Add one with: idea add \"your idea\"")
				}
				return nil
			}

			filter := idea.FilterOpen
			if all {
				filter = idea.FilterAll
			} else if done {
				filter = idea.FilterDone
			}

			ideas, err := idea.List(path, filter, sortField, reverse)
			if err != nil {
				return err
			}

			// Optional [id...] positional filter: keep only ideas whose ID was
			// requested, warn on stderr for any well-formed-but-absent ID, and
			// still list the rest (pipe-friendly posture — Constitution VI).
			if len(args) > 0 {
				ideas = filterByIDs(cmd, ideas, args)
			}

			if len(ideas) == 0 {
				if jsonOut {
					fmt.Println("[]")
				} else {
					fmt.Println("No ideas found.")
				}
				return nil
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(ideas)
			}

			// Display-only rendering is TTY-gated (see printIdeaLines): on a
			// terminal truncate to width (unless --full) and color; when piped
			// emit full canonical FormatLine output so the pipe contract holds.
			printIdeaLines(cmd.OutOrStdout(), ideas, full)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show all ideas (open + done)")
	cmd.Flags().BoolVar(&done, "done", false, "Show only done ideas")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&sortField, "sort", "date", "Sort by field (id or date)")
	cmd.Flags().BoolVar(&reverse, "reverse", false, "Reverse sort order")
	cmd.Flags().BoolVar(&full, "full", false, "Show full idea text on a terminal (no truncation)")

	return cmd
}

// filterByIDs returns the subset of ideas whose ID is in wantIDs, preserving
// the input order. Any requested ID with no match is reported once on stderr
// (warn-and-list-the-rest), consistent with the pipe-friendly stdout posture.
func filterByIDs(cmd *cobra.Command, ideas []idea.Idea, wantIDs []string) []idea.Idea {
	want := make(map[string]bool, len(wantIDs))
	for _, id := range wantIDs {
		want[id] = true
	}

	var result []idea.Idea
	found := make(map[string]bool, len(wantIDs))
	for _, i := range ideas {
		if want[i.ID] {
			result = append(result, i)
			found[i.ID] = true
		}
	}

	warned := make(map[string]bool, len(wantIDs))
	for _, id := range wantIDs {
		if !found[id] && !warned[id] {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: no idea with ID %q\n", id)
			warned[id] = true
		}
	}
	return result
}
