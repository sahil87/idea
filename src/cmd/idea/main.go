package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var fileFlag string
var mainFlag bool

// version is the binary version, overridden via -ldflags "-X main.version=..." at build time.
var version = "dev"

// errSilent is a sentinel returned from a subcommand's RunE when the command
// has already written a user-facing error message itself. The top-level error
// handler in main exits non-zero without printing anything additional.
var errSilent = errors.New("silent")

// newRootCmd builds the root command and registers all subcommands. It is the
// single source of the live cobra tree: both main() and the help-dump producer
// (which walks cmd.Root()) operate on the identical tree this factory builds.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "idea [text]",
		Short: "Backlog idea management (current worktree; use --main for main worktree)",
		Long: `Backlog idea management (current worktree; use --main for main worktree).

Shorthand: "idea <text>" is equivalent to "idea add <text>".`,
		Version:       version,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			// Delegate to the "add" subcommand to keep behavior consistent.
			add := addCmd()
			add.SetIn(cmd.InOrStdin())
			add.SetOut(cmd.OutOrStdout())
			add.SetErr(cmd.ErrOrStderr())
			// addCmd expects exactly 1 arg; join multiple positional args.
			return add.RunE(add, []string{strings.Join(args, " ")})
		},
	}

	root.PersistentFlags().StringVar(&fileFlag, "file", "", "Override backlog file path (relative to git root)")
	root.PersistentFlags().BoolVar(&mainFlag, "main", false, "Operate on the main worktree's backlog instead of the current worktree")

	root.AddCommand(
		addCmd(),
		listCmd(),
		showCmd(),
		doneCmd(),
		reopenCmd(),
		editCmd(),
		rmCmd(),
		updateCmd(),
		newShellInitCmd(),
		helpDumpCmd(),
	)

	return root
}

// printBackfillNotice writes an advisory notice to the command's stderr when a
// mutating save stamped today's date on one or more previously-dateless ideas.
// It is suppressed entirely when count is 0 so stdout — and a zero-backfill
// stderr — stay clean and machine-parseable (Constitution VI). The notice goes
// to stderr (not stdout) because it is advisory, not part of the command's
// machine-readable result.
func printBackfillNotice(cmd *cobra.Command, count int) {
	if count <= 0 {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "note: stamped today's date on %d previously-dateless item(s)\n", count)
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		if !errors.Is(err, errSilent) {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		}
		os.Exit(1)
	}
}
