package main

import (
	"fmt"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func fmtCmd() *cobra.Command {
	var check bool

	cmd := &cobra.Command{
		Use:   "fmt",
		Short: "Rewrite the backlog in canonical form, adopting bare checkboxes",
		Long: `Rewrite the current worktree's backlog into canonical form, gofmt-style.

Every recognized idea line is regenerated canonically: "-" bullet, no
indentation, LF endings, today's date stamped on dateless items, and legacy
lone backslashes doubled. Bare checkbox lines without a 4-char [id] anchor
(e.g. "- [ ] buy milk", also */+ bullets and [x]/[X]) are adopted as managed
ideas: each gets a fresh unique ID and today's date, with its checked state
preserved. Lines whose text starts with a bracket (e.g. "- [ ] [DEV-1011] ...")
and all other non-idea content keep their text verbatim; line endings
canonicalize file-wide (CRLF becomes LF, single trailing LF). A second run is
byte-stable, and an already-canonical file is not rewritten at all.

stdout stays empty; the report (one "adopted:" line per adopted idea plus
summary counts) goes to stderr. --check writes nothing, prints the same report,
and exits 1 when the file would change, 0 when it is already canonical. --main
targets the main worktree's backlog, --system targets the system-level backlog
($XDG_CONFIG_HOME/idea/backlog.md, else ~/.config/idea/backlog.md), and
--file / IDEAS_FILE point elsewhere (see "idea --help"). Outside a git repo the
system backlog is used automatically.

  idea fmt
  idea fmt --check`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			res, err := idea.Fmt(path, check)
			if err != nil {
				return err
			}

			for _, a := range res.Adopted {
				fmt.Fprintf(cmd.ErrOrStderr(), "adopted: [%s] %s\n", a.ID, idea.EscapeText(a.Text))
			}
			printBackfillNotice(cmd, res.Backfilled)
			if res.Changed {
				fmt.Fprintf(cmd.ErrOrStderr(), "fmt: %d line(s) normalized, %d line(s) adopted\n",
					res.Normalized, len(res.Adopted))
			}

			if check && res.Changed {
				// Non-canonical under --check: the report above is the
				// message; exit 1 without an extra ERROR line.
				return errSilent
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "Write nothing; report what would change and exit 1 if the file is not canonical")

	return cmd
}
