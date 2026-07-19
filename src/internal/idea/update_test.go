package idea

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

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

func TestIsBrewInstalledReturnsBool(t *testing.T) {
	// Smoke test: the function must not panic on whatever `os.Executable`
	// returns in the test process. The actual return value depends on the
	// environment — in CI it's false; on a developer machine running `go
	// test` from a brew install of go it's still false (the *go* test binary
	// lives under a temp dir, not /Cellar/). We just assert it doesn't crash.
	_ = isBrewInstalled()
}

// TestHelperProcess is the canonical Go stdlib fake-exec target. It is not a
// real test: when invoked normally it returns immediately. The recorder stub
// re-executes the test binary with `-test.run=TestHelperProcess` and
// GO_WANT_HELPER_PROCESS=1 so this body runs as a stand-in for `brew`.
//
// It inspects the original brew args (appended after a "--" separator) to fake
// each subcommand: `brew info` prints valid `--json=v2` output with a stable
// version that differs from the test's currentVersion (so the upgrade path is
// taken); `brew update` / `brew upgrade` simply exit 0.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Args after "--" are the original brew args the recorder forwarded.
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	isInfo := false
	for _, a := range args {
		if a == "info" {
			isInfo = true
			break
		}
	}
	if isInfo {
		// Stable version 9.9.9 differs from the test's "v0.0.1" so the
		// up-to-date short-circuit does NOT fire and the upgrade path runs.
		os.Stdout.WriteString(`{"formulae":[{"versions":{"stable":"9.9.9"}}]}`)
	}
	os.Exit(0)
}

// brewCall records one brew invocation seen by the recorder: the brew
// subcommand (the first arg after "brew") plus the ctx the call site passed
// through the execCommandContext seam. Capturing the ctx is what lets the
// tests pin the no-deadline brew-safety contract (see update.go's seam
// comment): a reintroduced context.WithTimeout shows up as a ctx deadline.
type brewCall struct {
	sub string
	ctx context.Context
}

// newBrewRecorder returns a stub matching exec.CommandContext's signature. It
// records each invocation's brew subcommand and ctx into *recorded and returns
// a command that re-runs the test binary's TestHelperProcess, forwarding the
// original brew args after a "--" separator so the helper can see which
// subcommand was requested.
func newBrewRecorder(recorded *[]brewCall) func(context.Context, string, ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// args[0] is the brew subcommand (update / upgrade / info).
		if name == "brew" && len(args) > 0 {
			*recorded = append(*recorded, brewCall{sub: args[0], ctx: ctx})
		}
		helperArgs := append([]string{"-test.run=TestHelperProcess", "--", name}, args...)
		cmd := exec.Command(os.Args[0], helperArgs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	}
}

// TestUpdateSkipBrewUpdate drives Update down the brew path with the seam vars
// stubbed and asserts which brew subcommands are spawned. With skip=false the
// `brew update` refresh runs; with skip=true it is omitted, while `brew info`
// and `brew upgrade` run in both cases.
//
// It also pins the no-kill-path brew-safety contract (toolkit update
// standard): every recorded brew invocation's ctx must be non-nil, report NO
// deadline, and be non-cancellable (Done() == nil), so no code path can ever
// SIGKILL a brew subprocess mid-transaction. Reintroducing a
// context.WithTimeout OR context.WithCancel around any brew call site fails
// this test.
func TestUpdateSkipBrewUpdate(t *testing.T) {
	tests := []struct {
		name       string
		skip       bool
		wantUpdate bool
	}{
		{name: "flag absent runs brew update", skip: false, wantUpdate: true},
		{name: "flag set skips brew update", skip: true, wantUpdate: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origBrewInstalled := brewInstalled
			origExecCommandContext := execCommandContext
			defer func() {
				brewInstalled = origBrewInstalled
				execCommandContext = origExecCommandContext
			}()

			var recorded []brewCall
			brewInstalled = func() bool { return true }
			execCommandContext = newBrewRecorder(&recorded)

			var stdout, stderr bytes.Buffer
			if err := Update("v0.0.1", tc.skip, &stdout, &stderr); err != nil {
				t.Fatalf("Update returned err: %v\nstderr: %s", err, stderr.String())
			}

			has := func(sub string) bool {
				for _, r := range recorded {
					if r.sub == sub {
						return true
					}
				}
				return false
			}

			// No-kill-path contract: no brew subprocess may run under a
			// deadline-carrying OR cancellable ctx (either arms
			// exec.CommandContext's SIGKILL cancel path, which the toolkit
			// update standard forbids). The nil check runs first so a
			// regression fails cleanly instead of panicking.
			for _, r := range recorded {
				if r.ctx == nil {
					t.Errorf("brew %s invoked with a nil ctx; brew subprocesses must run with context.Background()", r.sub)
					continue
				}
				if deadline, ok := r.ctx.Deadline(); ok {
					t.Errorf("brew %s invoked with a ctx deadline (%v); brew subprocesses must run with no deadline", r.sub, deadline)
				}
				if r.ctx.Done() != nil {
					t.Errorf("brew %s invoked with a cancellable ctx; brew subprocesses must run with a non-cancellable ctx (no kill path)", r.sub)
				}
			}

			if !has("info") {
				t.Errorf("expected brew info to be invoked; recorded: %v", recorded)
			}
			if !has("upgrade") {
				t.Errorf("expected brew upgrade to be invoked; recorded: %v", recorded)
			}
			if got := has("update"); got != tc.wantUpdate {
				t.Errorf("brew update invoked = %v, want %v; recorded: %v", got, tc.wantUpdate, recorded)
			}
		})
	}
}
