package idea

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// fakeBrewVersion is the stable version the helper process reports via the
// faked `brew info --json=v2` call. It is deliberately different from the
// currentVersion the tests pass to Update so the "already up to date"
// short-circuit is NOT taken and the brew upgrade path runs.
const fakeBrewVersion = "9.9.9"

// TestHelperProcess is not a real test. It is re-executed as a subprocess by
// fakeCommandContext to stand in for `brew`. The standard os/exec testing
// idiom: the parent sets GO_WANT_HELPER_PROCESS=1 and passes the intended brew
// subcommand via GO_BREW_SUBCOMMAND so the helper can emit the right output.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// When impersonating `brew info`, emit JSON that brewLatestVersion can parse.
	if os.Getenv("GO_BREW_SUBCOMMAND") == "info" {
		fmt.Fprintf(os.Stdout, `{"formulae":[{"versions":{"stable":%q}}]}`, fakeBrewVersion)
	}
	os.Exit(0)
}

// fakeCommandContext returns a commandContext replacement that records the
// brew subcommand (args[0] after "brew") into *recorded and returns an
// *exec.Cmd that re-execs the test binary as TestHelperProcess instead of
// spawning a real brew. It preserves the production invocation style — the
// returned value is a real *exec.Cmd whose .Run()/.Output()/stream wiring
// behaves normally.
func fakeCommandContext(recorded *[]string) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		sub := ""
		if len(args) > 0 {
			sub = args[0]
		}
		*recorded = append(*recorded, sub)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess")
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"GO_BREW_SUBCOMMAND="+sub,
		)
		return cmd
	}
}

// withFakeBrew installs the test seams (a brew-installed binary plus a faked
// commandContext) for the duration of one test and restores them afterward.
// It returns a pointer to the slice that records each brew subcommand invoked.
func withFakeBrew(t *testing.T) *[]string {
	t.Helper()
	var recorded []string

	origInstalled := brewInstalled
	origCommand := commandContext
	brewInstalled = func() bool { return true }
	commandContext = fakeCommandContext(&recorded)
	t.Cleanup(func() {
		brewInstalled = origInstalled
		commandContext = origCommand
	})

	return &recorded
}

func TestNormalizeVersion(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"v0.0.3", "0.0.3"},
		{"0.0.3", "0.0.3"},
		{"", ""},
		{"v", ""},
		{"vvv1.0.0", "vv1.0.0"}, // only one leading "v" is stripped
	}
	for _, c := range cases {
		if got := normalizeVersion(c.in); got != c.want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestUpdateNonBrewInstall confirms that when the running binary is NOT
// installed via Homebrew, Update prints a manual-update hint to its `out`
// writer and returns nil without invoking brew. We cannot easily simulate
// "brew install" inside the test process, but we CAN observe that go's
// `go test` binary doesn't live under /Cellar/, so isBrewInstalled returns
// false here — making this assertion stable in CI and on developer machines.
func TestUpdateNonBrewInstall(t *testing.T) {
	if isBrewInstalled() {
		t.Skip("test binary appears to be brew-installed; non-brew code path not exercised")
	}
	var stdout, stderr bytes.Buffer
	if err := Update("v0.0.3", false, &stdout, &stderr); err != nil {
		t.Fatalf("Update on non-brew install returned err: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{
		"v0.0.3 was not installed via Homebrew",
		"brew install sahil87/tap/idea",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected stdout to contain %q, got:\n%s", want, out)
		}
	}
	if got := stderr.String(); got != "" {
		t.Errorf("expected empty stderr, got: %q", got)
	}
}

// contains reports whether s appears in xs.
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// TestUpdateSkipsBrewUpdateWhenFlagSet is the core contract test: with
// skipBrewUpdate=true, the internal `brew update` (tap-metadata refresh) must
// NOT run, but the `brew info` version check AND `brew upgrade` must still run.
func TestUpdateSkipsBrewUpdateWhenFlagSet(t *testing.T) {
	recorded := withFakeBrew(t)

	var stdout, stderr bytes.Buffer
	if err := Update("v0.0.1", true, &stdout, &stderr); err != nil {
		t.Fatalf("Update returned err: %v", err)
	}

	if contains(*recorded, "update") {
		t.Errorf("expected `brew update` NOT to run with --skip-brew-update, got brew calls: %v", *recorded)
	}
	if !contains(*recorded, "info") {
		t.Errorf("expected `brew info` version check to still run, got brew calls: %v", *recorded)
	}
	if !contains(*recorded, "upgrade") {
		t.Errorf("expected `brew upgrade` to still run, got brew calls: %v", *recorded)
	}

	// The version check ran and saw a newer version, so the upgrade message
	// is emitted and the "already up to date" short-circuit is NOT taken.
	if out := stdout.String(); !strings.Contains(out, "Updated to v"+fakeBrewVersion) {
		t.Errorf("expected success message for v%s, got:\n%s", fakeBrewVersion, out)
	}
}

// TestUpdateRunsBrewUpdateByDefault confirms the default (flag absent) behavior
// is unchanged: `brew update`, `brew info`, and `brew upgrade` all run.
func TestUpdateRunsBrewUpdateByDefault(t *testing.T) {
	recorded := withFakeBrew(t)

	var stdout, stderr bytes.Buffer
	if err := Update("v0.0.1", false, &stdout, &stderr); err != nil {
		t.Fatalf("Update returned err: %v", err)
	}

	for _, want := range []string{"update", "info", "upgrade"} {
		if !contains(*recorded, want) {
			t.Errorf("expected `brew %s` to run by default, got brew calls: %v", want, *recorded)
		}
	}
}

// TestUpdateAlreadyUpToDate confirms the short-circuit still works under the
// faked brew: when current version matches the reported stable version, no
// `brew upgrade` runs regardless of the skip flag.
func TestUpdateAlreadyUpToDate(t *testing.T) {
	recorded := withFakeBrew(t)

	var stdout, stderr bytes.Buffer
	if err := Update("v"+fakeBrewVersion, false, &stdout, &stderr); err != nil {
		t.Fatalf("Update returned err: %v", err)
	}

	if contains(*recorded, "upgrade") {
		t.Errorf("expected no `brew upgrade` when already up to date, got brew calls: %v", *recorded)
	}
	if out := stdout.String(); !strings.Contains(out, "Already up to date") {
		t.Errorf("expected 'Already up to date' message, got:\n%s", out)
	}
}

func TestIsBrewInstalledReturnsBool(t *testing.T) {
	// Smoke test: the function must not panic on whatever `os.Executable`
	// returns in the test process. The actual return value depends on the
	// environment — in CI it's false; on a developer machine running `go
	// test` from a brew install of go it's still false (the *go* test binary
	// lives under a temp dir, not /Cellar/). We just assert it doesn't crash.
	_ = isBrewInstalled()
}
