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
var systemFlag bool

// version is the binary version, overridden via -ldflags "-X main.version=..." at build time.
var version = "dev"

// errSilent is a sentinel returned from a subcommand's RunE when the command
// has already written a user-facing error message itself. The top-level error
// handler in main exits non-zero without printing anything additional.
var errSilent = errors.New("silent")

// usageError marks an error as stemming from a malformed invocation (bad flag,
// wrong arg count, conflicting target flags) so main() maps it to exit 2 per the
// toolkit exit-code convention (0 success / 1 operational failure / 2 usage
// error). Exit-code policy is a cmd/ concern (Constitution IV); internal/idea
// stays policy-free.
//
// Unwrap() is load-bearing: it lets a usage error compose with errSilent — a
// self-printed usage error returns &usageError{errSilent} (exit 2, no extra
// "ERROR:" line) — and keeps errors.Is/errors.As classification working in main().
type usageError struct{ err error }

func (u *usageError) Error() string { return u.err.Error() }
func (u *usageError) Unwrap() error { return u.err }

// usageArgs wraps a cobra positional-args validator so its rejection is
// classified as a usage error (exit 2). SetFlagErrorFunc does not catch
// arg-count errors, so each subcommand's Args validator is wrapped explicitly.
func usageArgs(v cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := v(cmd, args); err != nil {
			return &usageError{err}
		}
		return nil
	}
}

// newRootCmd builds the root command and registers all subcommands. It is the
// single source of the live cobra tree: both main() and the help-dump producer
// (which walks cmd.Root()) operate on the identical tree this factory builds.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "idea [text]",
		Short: "Backlog idea management (current worktree; use --main for main worktree)",
		Long: `Backlog idea management for the command line.

Targets (which backlog a command operates on):
  (default)      current worktree's backlog
  -m, --main     main worktree's backlog (shared)
  -s, --system   ~/.config/idea/backlog.md (cross-repo; also the default outside a repo)

--main and --system are mutually exclusive. --file/-f overrides the backlog
path within the selected root (ignored with --system).

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

	// Classify every flag-parse error as a usage error (exit 2). Cobra inherits
	// FlagErrorFunc from the parent, so all subcommands are covered without
	// per-command wiring. The message text is unchanged.
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return &usageError{err}
	})

	root.PersistentFlags().StringVarP(&fileFlag, "file", "f", "", "Override backlog file path (relative to the git root, or to ~/.config/idea when outside a repo)")
	root.PersistentFlags().BoolVarP(&mainFlag, "main", "m", false, "Operate on the main worktree's backlog instead of the current worktree")
	root.PersistentFlags().BoolVarP(&systemFlag, "system", "s", false, "Operate on the system-level backlog (~/.config/idea/backlog.md) instead of a repo backlog")

	root.AddCommand(
		addCmd(),
		listCmd(),
		showCmd(),
		doneCmd(),
		reopenCmd(),
		editCmd(),
		rmCmd(),
		pruneCmd(),
		fmtCmd(),
		updateCmd(),
		skillCmd(),
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
		// Toolkit exit-code convention: usage errors (bad flag, wrong arg
		// count, conflicting target flags) exit 2; every other operational
		// failure exits 1.
		code := 1
		var uerr *usageError
		if errors.As(err, &uerr) {
			code = 2
		}
		os.Exit(code)
	}
}
