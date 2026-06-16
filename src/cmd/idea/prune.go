package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func pruneCmd() *cobra.Command {
	var force, full bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Bulk-remove all done ideas from the backlog",
		Long: `Bulk-remove all done ([x]) ideas from the current worktree's backlog.

Without --force this lists each done idea that would be removed. On a terminal
it then prompts for confirmation ([y/N]) and deletes only if you confirm; when
the output is piped it stays a free dry run (removable lines on stdout, a hint
on stderr) and never prompts. --force skips the prompt and deletes immediately,
printing only a count. There is no archive — the backlog is committed, so git
history is the recovery path. Long idea text is truncated to the terminal width
(--full shows it in full); piped output is always full and machine-parseable.
--main targets the main worktree's backlog, --system targets the system-level
backlog (~/.config/idea/backlog.md), and --file / IDEAS_FILE point elsewhere
(see "idea --help"). Outside a git repo the system backlog is used
automatically.

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

			if force {
				printBackfillNotice(cmd, backfilled)
				fmt.Printf("Pruned %d done idea(s).\n", len(pruned))
				return nil
			}

			// Dry-run / pre-confirm path. The leading count header is the
			// primary signal, printed to stderr (advisory) before the list so a
			// human sees the action first regardless of list length; stdout
			// still carries exactly the removable lines (pipe-friendly).
			fmt.Fprintf(cmd.ErrOrStderr(), "%d done idea(s) would be pruned\n", len(pruned))
			printIdeaLines(cmd.OutOrStdout(), pruned, full)

			// Interactive confirm only when stdout is a TTY (a prompt on a pipe
			// would hang). On a TTY the prompt replaces the trailing hint; on a
			// pipe we fall back to the classic dry run with the trailing hint.
			if !idea.IsTTY(os.Stdout) {
				fmt.Fprintln(cmd.ErrOrStderr(), "Re-run with --force to confirm.")
				return nil
			}

			if !confirmPrune(cmd, len(pruned)) {
				fmt.Fprintln(cmd.ErrOrStderr(), "Aborted — no ideas removed.")
				return nil
			}

			// Confirmed: perform the deletion via the same force path. Use the
			// count from this confirmed prune (not the earlier dry run) so the
			// reported total is accurate even if the file changed in between.
			pruned, backfilled, err = idea.Prune(path, true)
			if err != nil {
				return err
			}
			printBackfillNotice(cmd, backfilled)
			fmt.Printf("Pruned %d done idea(s).\n", len(pruned))
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Confirm deletion of all done ideas")
	cmd.Flags().BoolVar(&full, "full", false, "Show full idea text on a terminal (no truncation)")

	return cmd
}

// confirmPrune writes the [y/N] prompt to stderr and reads one line from the
// command's stdin. It returns true only for "y"/"yes" (case-insensitive,
// trimmed); any other input (including EOF) aborts.
func confirmPrune(cmd *cobra.Command, n int) bool {
	fmt.Fprintf(cmd.ErrOrStderr(), "Prune %d done idea(s)? [y/N] ", n)
	reader := bufio.NewReader(cmd.InOrStdin())
	line, _ := reader.ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
