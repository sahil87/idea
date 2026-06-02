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

Runs the Homebrew upgrade for the installed idea formula, refreshing tap
metadata first so the latest published release is visible. Pass
--skip-brew-update to skip that internal "brew update" refresh when the tap is
already current (faster, but may miss a just-published version).

  idea update
  idea update --skip-brew-update`,
		Args: cobra.NoArgs,
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
