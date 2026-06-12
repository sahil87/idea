package main

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestFmt_Routing verifies "idea fmt" resolves to the fmt subcommand (never
// the root bare-text add shorthand) and that stray args error instead of
// falling through to add — the namespace claim accepted in the intake.
func TestFmt_Routing(t *testing.T) {
	bin := buildBinary(t)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"fmt routes to subcommand", []string{"fmt"}, false},
		{"stray args error, no add fallthrough", []string{"fmt", "some", "text"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupGitRepo(t)
			writeRepoBacklog(t, repo, "# Backlog\n\n- [ ] [ab12] 2026-06-01: seeded idea\n")
			before := readRepoBacklog(t, repo)

			stdout, stderr, err := runSplit(t, bin, repo, tt.args...)
			if tt.wantErr && err == nil {
				t.Fatalf("%v succeeded, want error\nstdout=%q stderr=%q", tt.args, stdout, stderr)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("%v failed: %v\nstdout=%q stderr=%q", tt.args, err, stdout, stderr)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if after := readRepoBacklog(t, repo); after != before {
				t.Errorf("backlog changed by %v:\nbefore:\n%s\nafter:\n%s", tt.args, before, after)
			}
		})
	}
}

// TestFmt_RewritesAndReportsOnStderr runs the intake's worked example end to
// end: candidates adopted, bracket-guarded line untouched, stdout silent, and
// the full report (adoption lines, backfill note, summary) on stderr.
func TestFmt_RewritesAndReportsOnStderr(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)
	today := time.Now().Format("2006-01-02")

	writeRepoBacklog(t, repo, "# Backlog\n\n"+
		"* [ ] buy milk\n"+
		"- [X] ship the release\n"+
		"- [ ] [DEV-1011] external item\n"+
		"- [ ] [rk7t] dateless managed\n")

	stdout, stderr, err := runSplit(t, bin, repo, "fmt")
	if err != nil {
		t.Fatalf("fmt failed: %v\nstdout=%q stderr=%q", err, stdout, stderr)
	}

	if stdout != "" {
		t.Errorf("stdout = %q, want empty (success is silence)", stdout)
	}
	if got := strings.Count(stderr, "adopted: ["); got != 2 {
		t.Errorf("stderr has %d adoption lines, want 2:\n%s", got, stderr)
	}
	if !strings.Contains(stderr, "buy milk") || !strings.Contains(stderr, "ship the release") {
		t.Errorf("adoption report missing adopted texts:\n%s", stderr)
	}
	if !strings.Contains(stderr, "note: stamped today's date on 1 previously-dateless item(s)") {
		t.Errorf("stderr missing backfill advisory:\n%s", stderr)
	}
	if !strings.Contains(stderr, "fmt: 1 line(s) normalized, 2 line(s) adopted") {
		t.Errorf("stderr missing summary counts:\n%s", stderr)
	}

	after := readRepoBacklog(t, repo)
	if !strings.Contains(after, "- [ ] [DEV-1011] external item\n") {
		t.Errorf("bracket-guarded line was touched:\n%s", after)
	}
	if !strings.Contains(after, ": buy milk\n") || !strings.Contains(after, today+": ship the release\n") {
		t.Errorf("adopted lines not canonical:\n%s", after)
	}
	if strings.Contains(after, "* [ ]") || strings.Contains(after, "[X]") {
		t.Errorf("variant forms survived fmt:\n%s", after)
	}
}

// TestFmt_Idempotent verifies the second run is byte-stable and silent.
func TestFmt_Idempotent(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)
	writeRepoBacklog(t, repo, "* [ ] buy milk\n- [ ] [rk7t] dateless\n")

	if _, _, err := runSplit(t, bin, repo, "fmt"); err != nil {
		t.Fatalf("first fmt failed: %v", err)
	}
	first := readRepoBacklog(t, repo)

	stdout, stderr, err := runSplit(t, bin, repo, "fmt")
	if err != nil {
		t.Fatalf("second fmt failed: %v\nstdout=%q stderr=%q", err, stdout, stderr)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("second run not silent: stdout=%q stderr=%q", stdout, stderr)
	}
	if second := readRepoBacklog(t, repo); second != first {
		t.Errorf("second run changed bytes:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// TestFmt_CheckMode verifies the --check contract: exit 1 + would-be report +
// untouched file when non-canonical; exit 0 + silence when canonical.
func TestFmt_CheckMode(t *testing.T) {
	bin := buildBinary(t)

	tests := []struct {
		name       string
		backlog    string
		wantExit   int
		wantReport bool
	}{
		{
			name:       "non-canonical exits 1 and reports",
			backlog:    "* [ ] buy milk\n- [ ] [rk7t] dateless\n",
			wantExit:   1,
			wantReport: true,
		},
		{
			name:     "canonical exits 0 silently",
			backlog:  "# Backlog\n\n- [ ] [ab12] 2026-06-01: canonical idea\n",
			wantExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupGitRepo(t)
			writeRepoBacklog(t, repo, tt.backlog)

			stdout, stderr, err := runSplit(t, bin, repo, "fmt", "--check")

			exit := 0
			if err != nil {
				ee, ok := err.(*exec.ExitError)
				if !ok {
					t.Fatalf("fmt --check failed to run: %v", err)
				}
				exit = ee.ExitCode()
			}
			if exit != tt.wantExit {
				t.Errorf("exit code = %d, want %d\nstderr=%q", exit, tt.wantExit, stderr)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if tt.wantReport {
				if !strings.Contains(stderr, "adopted: [") || !strings.Contains(stderr, "fmt: ") {
					t.Errorf("expected would-be report on stderr, got %q", stderr)
				}
				if strings.Contains(stderr, "ERROR:") {
					t.Errorf("non-canonical --check must exit silently (no ERROR line), got %q", stderr)
				}
			} else if stderr != "" {
				t.Errorf("stderr = %q, want empty on a canonical file", stderr)
			}

			if after := readRepoBacklog(t, repo); after != tt.backlog {
				t.Errorf("--check modified the file:\nbefore:\n%s\nafter:\n%s", tt.backlog, after)
			}
		})
	}
}
