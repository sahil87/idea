package main

import (
	"fmt"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func editCmd() *cobra.Command {
	var newID, newDate string

	cmd := &cobra.Command{
		Use:   "edit <query> [new-text]",
		Short: "Modify an idea's text",
		Long: `Replace a matching idea's text in the current worktree's backlog.

With <new-text> the idea's text is replaced inline — the quick one-liner and
scripting path. Without it, your editor ($VISUAL, then $EDITOR, then vi)
opens on the idea's current text in decoded form (real newlines, real
backslashes); saving and exiting cleanly persists the result. An unchanged
buffer is a no-op (unless --id/--date is given), an emptied buffer is
refused, and a non-zero editor exit aborts without touching the backlog.

<query> matches an idea (open or done) by its ID or by a case-insensitive
substring of its text. If it matches more than one idea it is refused and the
ambiguous matches are listed, so you can be more specific or use the exact ID.
--id and --date additionally change the matched idea's ID or date. --main
targets the main worktree's backlog, --system targets the system-level backlog
(~/.config/idea/backlog.md), and --file / IDEAS_FILE point elsewhere (see
"idea --help"). Outside a git repo the system backlog is used automatically.

  idea edit a7k2 "wire up dark mode toggle"
  idea edit a7k2                  # open the current text in your editor
  idea edit a7k2 --date 2026-06-01 "backdated rewrite"`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			var newText string
			if len(args) == 2 {
				// Two-arg form: inline replacement, no editor — unchanged behavior.
				newText = args[1]
			} else {
				// One-arg form: resolve the match first so an ambiguous or
				// unmatched query is refused before any editor launches, then
				// round-trip the decoded text through the user's editor.
				current, err := idea.Show(path, args[0])
				if err != nil {
					return err
				}
				edited, unchanged, err := idea.EditInEditor(current.Text)
				if err != nil {
					return err
				}
				if edited == "" {
					return fmt.Errorf("editor buffer is empty — idea unchanged (use \"idea rm\" to delete it)")
				}
				// Unchanged text is a no-op (no rewrite, so no normalize or
				// backfill side effects) — unless --id/--date still has
				// metadata to apply at save.
				if unchanged {
					if newID == "" && newDate == "" {
						fmt.Fprintln(cmd.ErrOrStderr(), "note: text unchanged — nothing to do")
						return nil
					}
					// Metadata-only save: keep the original text verbatim.
					// edited may have lost a trailing LF to the strip-one
					// rule (the unchanged verdict can come from the
					// pre-strip buffer), and a metadata change must never
					// mutate text.
					newText = current.Text
				} else {
					newText = edited
				}
			}

			i, backfilled, err := idea.Edit(path, args[0], newText, newID, newDate)
			if err != nil {
				return err
			}

			printBackfillNotice(cmd, backfilled)
			fmt.Printf("Updated: %s\n", idea.FormatLine(i))
			return nil
		},
	}

	cmd.Flags().StringVar(&newID, "id", "", "Change the idea's ID")
	cmd.Flags().StringVar(&newDate, "date", "", "Change the idea's date")

	return cmd
}
