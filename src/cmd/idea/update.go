package main

import (
	"errors"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/sahil87/idea/internal/idea"
)

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "self-update the idea binary via Homebrew",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := idea.Update(version, cmd.OutOrStdout(), cmd.ErrOrStderr())
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
}
