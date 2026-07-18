package main

import (
	"errors"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/sahil87/idea/internal/idea"
)

func updateCmd() *cobra.Command {
	var skipBrewUpdate bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "self-update the idea binary via Homebrew",
		Long: `Self-update the idea binary via Homebrew.

Checks the installed idea formula against the latest published release and runs
"brew upgrade" only when a newer version exists; if the binary is already
current it prints "Already up to date" and exits without upgrading. The tap
metadata is refreshed first (via an internal "brew update") so a just-published
release is visible. Pass --skip-brew-update to skip only that tap-metadata
refresh (faster, but may miss a just-published version) — the version check and
any needed upgrade still run. If idea was not installed via Homebrew, the
command explains how to update manually instead.

  idea update
  idea update --skip-brew-update`,
		Args: usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := idea.Update(version, skipBrewUpdate, cmd.OutOrStdout(), cmd.ErrOrStderr())
			// internal/idea writes its own "brew not found" hint to stderr
			// before returning an exec.ErrNotFound-wrapping error. Map it
			// to errSilent so main's top-level error handler does not also
			// print a redundant "ERROR: ..." line.
			if errors.Is(err, exec.ErrNotFound) {
				return errSilent
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&skipBrewUpdate, "skip-brew-update", false, "Skip the internal `brew update` tap-metadata refresh")
	return cmd
}
