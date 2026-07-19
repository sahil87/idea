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
)

// brewFormula is the fully-qualified tap formula. The fully-qualified form
// disambiguates against any same-named core formula that would otherwise
// shadow it on `brew info idea`.
const brewFormula = "sahil87/tap/idea"

// execCommandContext and brewInstalled are indirection seams for testing.
// Production code uses exec.CommandContext and isBrewInstalled verbatim; tests
// stub these package-level vars to record subprocess invocations and force the
// brew code path. They are NOT a command-runner abstraction — os/exec remains
// the mechanism (see Design Decision 1 in the change spec).
//
// No-deadline brew-safety contract: every brew call site passes
// context.Background() through this seam — never a deadline-carrying ctx. The
// toolkit update standard forbids sending SIGKILL to a package-manager
// subprocess mid-transaction and forbids a short hard timeout on `brew
// upgrade`: exec.CommandContext's default cancel sends os.Kill on deadline,
// and a SIGKILL landing between brew's unlink and link steps corrupts the keg.
// With context.Background() the cancel path is never armed, so the seam
// behaves identically to exec.Command. Ctrl-C remains the user's escape hatch
// (SIGINT reaches the foreground process group; brew traps it and unwinds
// cleanly). Do NOT reintroduce context.WithTimeout here — the ctx-deadline
// assertion in update_test.go pins this contract.
var (
	execCommandContext = exec.CommandContext
	brewInstalled      = isBrewInstalled
)

// Update self-updates the idea binary via Homebrew.
//
// currentVersion is the binary's reported version (e.g. "v0.0.3"). The leading
// "v" is stripped before comparison since `brew info` reports the bare form.
//
// skipBrewUpdate gates ONLY the internal `brew update --quiet` tap-metadata
// refresh. When true, that refresh is skipped and control flows directly to the
// `brew info` version check; the up-to-date short-circuit and `brew upgrade`
// are unaffected. The non-Homebrew-install short-circuit at the top is likewise
// unaffected.
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
		updateCmd := execCommandContext(context.Background(), "brew", "update", "--quiet")
		var updateStderr bytes.Buffer
		updateCmd.Stderr = &updateStderr
		err := updateCmd.Run()
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

	cmd := execCommandContext(context.Background(), "brew", "upgrade", brewFormula)
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
	out, err := execCommandContext(context.Background(), "brew", "info", "--json=v2", brewFormula).Output()
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
