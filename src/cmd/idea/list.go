package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	var all, done, jsonOut, reverse, full bool
	var sortField, stale string

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

--stale filters to open ideas older than the given number of days ("90d" or
"90"); an idea dated exactly that many days ago is not yet stale, and --stale
cannot be combined with --done or --all.

On a terminal, long idea text is truncated to fit the width (the [id] date:
prefix is never clipped) and the prefix is dimmed; --full shows the complete
text. Ideas older than the effective staleness threshold (the --stale value
when passed, else 90 days) render entirely faint. When the output is piped or
redirected, full canonical lines are emitted regardless of --full so downstream
tools see machine-parseable records. As with every backlog command, --main
targets the main worktree's backlog, --system targets the system-level backlog
(~/.config/idea/backlog.md), and --file / IDEAS_FILE point elsewhere (see "idea
--help"). Outside a git repo the system backlog is used automatically.

  idea list
  idea list --all --sort id
  idea list --stale 90d
  idea ls a7k2 b3c9 --full
  idea list --json`,
		Args: usageArgs(func(cmd *cobra.Command, args []string) error {
			for _, a := range args {
				if err := idea.ValidateID(a); err != nil {
					return err
				}
			}
			return nil
		}),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			// --stale validation happens up front so a bad value is a usage
			// error (exit 2) even on an absent backlog.
			staleDays := idea.DefaultStaleDimDays
			if cmd.Flags().Changed("stale") {
				staleDays, err = idea.ParseStaleDays(stale)
				if err != nil {
					return &usageError{err}
				}
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
			// Runs BEFORE the stale filter so an existing-but-fresh requested
			// ID is dropped silently (it exists; it just isn't stale), with
			// warnings reserved for genuinely absent IDs.
			if len(args) > 0 {
				ideas = filterByIDs(cmd, ideas, args)
			}

			// --stale keeps only open ideas strictly older than the cutoff
			// (open-only is guaranteed: --done/--all are mutually exclusive
			// with --stale). One clock serves the filter and the dim pass.
			now := time.Now()
			if cmd.Flags().Changed("stale") {
				filtered := ideas[:0]
				for _, i := range ideas {
					if idea.IsStale(i, staleDays, now) {
						filtered = append(filtered, i)
					}
				}
				ideas = filtered
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
			// Ideas past the effective staleness threshold render faint.
			printIdeaLines(cmd.OutOrStdout(), ideas, full, staleDays, now)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show all ideas (open + done)")
	cmd.Flags().BoolVar(&done, "done", false, "Show only done ideas")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&sortField, "sort", "date", "Sort by field (id or date)")
	cmd.Flags().BoolVar(&reverse, "reverse", false, "Reverse sort order")
	cmd.Flags().BoolVar(&full, "full", false, "Show full idea text on a terminal (no truncation)")
	cmd.Flags().StringVar(&stale, "stale", "", "Show only open ideas older than N days (e.g. \"90d\" or \"90\")")

	// --stale implies open-only, so it cannot combine with the flags that
	// select done ideas — an explicit conflict beats a silent override.
	cmd.MarkFlagsMutuallyExclusive("stale", "done")
	cmd.MarkFlagsMutuallyExclusive("stale", "all")

	// Cobra's group validation runs AFTER PreRunE but classifies nothing, so
	// reject the conflict here first as a usage error (exit 2) per the repo's
	// 0/1/2 exit-code convention.
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("stale") && (done || all) {
			return &usageError{fmt.Errorf("--stale cannot be combined with --done or --all")}
		}
		return nil
	}

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
