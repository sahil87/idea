package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sahil87/idea/internal/idea"
	"github.com/spf13/cobra"
)

func showCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "show <query>",
		Short: "Show a single idea",
		Long: `Show a single matching idea from the current worktree's backlog.

<query> matches an idea (open or done) by its ID or by a case-insensitive
substring of its text. If it matches more than one idea it is refused and the
ambiguous matches are listed, so you can be more specific or use the exact ID.
--json emits the structured record (id, date, status, text) instead of the
formatted line. --main targets the main worktree's backlog, --system targets
the system-level backlog ($XDG_CONFIG_HOME/idea/backlog.md, else
~/.config/idea/backlog.md), and --file / IDEAS_FILE point elsewhere (see
"idea --help"). Outside a git repo the system backlog is used automatically.

  idea show a7k2
  idea show "dark mode" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveFile()
			if err != nil {
				return err
			}

			i, err := idea.Show(path, args[0])
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				return enc.Encode(i)
			}

			// DisplayLine renders the real (unescaped) text, so multiline
			// ideas show their continuation lines below the prefix line.
			fmt.Println(idea.DisplayLine(i))
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}
