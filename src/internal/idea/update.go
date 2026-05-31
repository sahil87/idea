// Self-update support for the idea binary via Homebrew.
//
// This file lives in the existing internal/idea package — idea has only one
// internal domain today, so a separate internal/update package would
// over-fragment the codebase (Constitution Principle IV requires only that
// logic live outside cmd/, not one package per subcommand). Subprocess
// invocations use os/exec directly, matching the existing usage in idea.go;
// no internal/proc wrapper is introduced.
//
// The brew formula is referenced by its fully-qualified name
// (sahil87/tap/idea) to avoid any collision with a same-named core formula.
package idea

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// brewFormula is the fully-qualified tap formula. The fully-qualified form
// disambiguates against any same-named core formula that would otherwise
// shadow it on `brew info idea`.
const brewFormula = "sahil87/tap/idea"

const (
	brewUpdateTimeout  = 30 * time.Second
	brewInfoTimeout    = 30 * time.Second
	brewUpgradeTimeout = 120 * time.Second
)

// commandContext constructs the *exec.Cmd for every brew subprocess. It is a
// package-level seam aliasing exec.CommandContext so unit tests can observe
// which brew subcommands run (e.g. asserting `brew update` is skipped) without
// spawning a real Homebrew. Production always uses the stdlib constructor; the
// subprocess invocation style (.Run()/.Output(), stream wiring) is unchanged.
var commandContext = exec.CommandContext

// brewInstalled reports whether the running binary is a Homebrew install. It is
// a package-level seam aliasing isBrewInstalled so unit tests can exercise the
// brew code path on machines/CI where the test binary does not live under a
// Cellar directory.
var brewInstalled = isBrewInstalled

// Update self-updates the idea binary via Homebrew.
//
// currentVersion is the binary's reported version (e.g. "v0.0.3"). The leading
// "v" is stripped before comparison since `brew info` reports the bare form.
//
// out and errOut receive only the WRAPPER messages this package emits ("Current
// version:", "Already up to date", error hints, etc.). Subprocess stdout/stderr
// from `brew update` and `brew info` is captured for parsing or discarded; the
// foreground `brew upgrade` call inherits the parent's os.Stdin/Stdout/Stderr
// directly so brew's tty-aware progress is visible to the user. The split is
// deliberate: the wrapper messages are small and may be redirected for tests
// or embedding; subprocess streams are large and tty-aware. Callers in
// production should pass os.Stdout / os.Stderr to keep the two consistent.
//
// skipBrewUpdate, when true, skips ONLY the internal `brew update --quiet`
// (tap-metadata refresh) step. Everything else runs unchanged: the `brew info`
// version check, the "already up to date" short-circuit, and `brew upgrade`.
// This is a cross-toolkit contract shared with sibling tools; the flag name is
// always `--skip-brew-update`. Default (false) preserves the original behavior.
//
// Returns nil on success or no-op (not a brew install, already up to date).
// Returns an error wrapping exec.ErrNotFound when brew is missing on PATH
// (callers should map this to errSilent so cobra does not double-print).
// Returns a wrapped error for other brew failures.
func Update(currentVersion string, skipBrewUpdate bool, out, errOut io.Writer) error {
	if !brewInstalled() {
		fmt.Fprintf(out, "idea %s was not installed via Homebrew.\n", currentVersion)
		fmt.Fprintln(out, "Update manually, or reinstall with: brew install "+brewFormula)
		return nil
	}

	fmt.Fprintf(out, "Current version: %s\n", currentVersion)
	fmt.Fprintln(out, "Checking for updates...")

	if !skipBrewUpdate {
		ctx, cancel := context.WithTimeout(context.Background(), brewUpdateTimeout)
		updateCmd := commandContext(ctx, "brew", "update", "--quiet")
		var updateStderr bytes.Buffer
		updateCmd.Stderr = &updateStderr
		err := updateCmd.Run()
		cancel()
		if err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				fmt.Fprintln(errOut, "idea update: brew not found on PATH.")
				return err
			}
			if detail := strings.TrimSpace(updateStderr.String()); detail != "" {
				return fmt.Errorf("brew update failed: %w: %s", err, detail)
			}
			return fmt.Errorf("brew update failed: %w", err)
		}
	}

	latest, err := brewLatestVersion()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			fmt.Fprintln(errOut, "idea update: brew not found on PATH.")
			return err
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if detail := strings.TrimSpace(string(exitErr.Stderr)); detail != "" {
				return fmt.Errorf("could not determine latest version: %w: %s", err, detail)
			}
		}
		return fmt.Errorf("could not determine latest version: %w", err)
	}

	if normalizeVersion(latest) == normalizeVersion(currentVersion) {
		fmt.Fprintf(out, "Already up to date (%s).\n", currentVersion)
		return nil
	}

	fmt.Fprintf(out, "Updating %s → v%s...\n", currentVersion, normalizeVersion(latest))

	upCtx, upCancel := context.WithTimeout(context.Background(), brewUpgradeTimeout)
	defer upCancel()
	cmd := commandContext(upCtx, "brew", "upgrade", brewFormula)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			fmt.Fprintln(errOut, "idea update: brew not found on PATH.")
			return err
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("brew upgrade exited with code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("brew upgrade failed: %w", err)
	}

	fmt.Fprintf(out, "Updated to v%s.\n", normalizeVersion(latest))
	return nil
}

// brewLatestVersion queries Homebrew for the latest stable version of the
// tap formula. Returns the bare version string (e.g. "0.0.3") with no `v`
// prefix — that's how brew reports it in `versions.stable`.
func brewLatestVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), brewInfoTimeout)
	defer cancel()
	out, err := commandContext(ctx, "brew", "info", "--json=v2", brewFormula).Output()
	if err != nil {
		return "", err
	}
	var info struct {
		Formulae []struct {
			Versions struct {
				Stable string `json:"stable"`
			} `json:"versions"`
		} `json:"formulae"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return "", err
	}
	if len(info.Formulae) == 0 || info.Formulae[0].Versions.Stable == "" {
		return "", errors.New("no stable version found in brew info output")
	}
	return info.Formulae[0].Versions.Stable, nil
}

// isBrewInstalled checks whether the running binary lives under a Cellar
// directory, which is the canonical signature of a Homebrew install. The
// symlink at /opt/homebrew/bin/idea (or /usr/local/bin/idea on Intel) resolves
// through to .../Cellar/idea/<version>/bin/idea.
func isBrewInstalled() bool {
	self, err := os.Executable()
	if err != nil {
		return false
	}
	real, err := filepath.EvalSymlinks(self)
	if err != nil {
		return false
	}
	return strings.Contains(real, "/Cellar/")
}

// normalizeVersion strips a single leading "v" so we can compare the binary's
// `git describe`-derived version (e.g. "v0.0.3") against brew's bare report
// ("0.0.3"). It does NOT do semver parsing — string equality after normalize
// is sufficient because both sides come from the same canonical source (the
// release tag).
func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}
