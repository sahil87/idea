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
formatted line. --main targets the main worktree's backlog and --file /
IDEAS_FILE point elsewhere (see "idea --help").

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

			fmt.Println(idea.FormatLine(i))
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}
