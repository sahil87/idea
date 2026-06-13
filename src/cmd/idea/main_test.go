package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/idea/internal/idea"
)

// buildBinary builds the idea binary to a temp location for integration tests.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "idea")
	cmd := exec.Command("go", "build", "-o", bin, "./")
	cmd.Dir = filepath.Join(findModuleRoot(t), "cmd", "idea")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	// Walk up from cmd/ to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod")
		}
		dir = parent
	}
}

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	backlogDir := filepath.Join(dir, "fab")
	if err := os.MkdirAll(backlogDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backlogDir, "backlog.md"), []byte("# Backlog\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
	return string(out)
}

func TestBareShorthand_AddsIdea(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	cmd := exec.Command(bin, "refactor auth middleware")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bare shorthand failed: %v\n%s", err, out)
	}
	if !strings.HasPrefix(string(out), "Added: [") {
		t.Errorf("expected 'Added: [' prefix, got: %s", out)
	}
	if !strings.Contains(string(out), "refactor auth middleware") {
		t.Errorf("expected idea text in output, got: %s", out)
	}
}

func TestBareShorthand_EmptyTextErrors(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	cmd := exec.Command(bin, "")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	if !strings.Contains(string(out), "text is required") {
		t.Errorf("expected empty text error, got: %s", out)
	}
}

func TestBareShorthand_MultipleArgsJoined(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	cmd := exec.Command(bin, "refactor", "auth", "middleware")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("multi-arg shorthand failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "refactor auth middleware") {
		t.Errorf("expected joined text in output, got: %s", out)
	}
}

func TestSubcommand_AddStillWorks(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	cmd := exec.Command(bin, "add", "via add subcommand")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("add subcommand failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "via add subcommand") {
		t.Errorf("expected idea text in output, got: %s", out)
	}
}

func TestSubcommand_ListStillWorks(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	// Add an idea first
	addCmd := exec.Command(bin, "add", "test idea")
	addCmd.Dir = repo
	if out, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("add failed: %v\n%s", err, out)
	}

	cmd := exec.Command(bin, "list")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list subcommand failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "test idea") {
		t.Errorf("expected idea in list output, got: %s", out)
	}
}

func TestNoArgs_ShowsHelp(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	cmd := exec.Command(bin)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("no-args failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "idea [text]") && !strings.Contains(string(out), "Backlog idea management") {
		t.Errorf("expected help output, got: %s", out)
	}
}

// writeRepoBacklog overwrites the repo's fab/backlog.md with the given content.
func writeRepoBacklog(t *testing.T, repo, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, "fab", "backlog.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// runSplit runs the binary capturing stdout and stderr separately. The nil
// environment inherits the process environment.
func runSplit(t *testing.T, bin, repo string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runSplitEnv(t, bin, repo, nil, args...)
}

// readRepoBacklog returns the current contents of the repo's fab/backlog.md.
func readRepoBacklog(t *testing.T, repo string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo, "fab", "backlog.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestRouting_LsAliasAndBareShorthand verifies command routing: "ls" resolves
// to the list subcommand (never the root bare-text add shorthand, which would
// silently append a junk "ls" idea), while bare text starting with a non-alias
// word still routes to add.
func TestRouting_LsAliasAndBareShorthand(t *testing.T) {
	bin := buildBinary(t)

	tests := []struct {
		name       string
		args       []string
		equivalent []string // when non-nil, stdout must match running these args on the same repo state
		wantAdded  string   // when non-empty, the backlog must gain an idea containing this text
	}{
		{
			name:       "ls routes to list",
			args:       []string{"ls"},
			equivalent: []string{"list"},
		},
		{
			name:       "ls --json routes to list --json",
			args:       []string{"ls", "--json"},
			equivalent: []string{"list", "--json"},
		},
		{
			name:      "non-alias bare text routes to add shorthand",
			args:      []string{"lsx", "some", "text"},
			wantAdded: "lsx some text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupGitRepo(t)
			writeRepoBacklog(t, repo, "# Backlog\n\n- [ ] [ab12] 2026-06-01: seeded idea\n")
			before := readRepoBacklog(t, repo)

			stdout, stderr, err := runSplit(t, bin, repo, tt.args...)
			if err != nil {
				t.Fatalf("%v failed: %v\nstdout=%q stderr=%q", tt.args, err, stdout, stderr)
			}

			if tt.equivalent != nil {
				wantOut, wantErr, eqErr := runSplit(t, bin, repo, tt.equivalent...)
				if eqErr != nil {
					t.Fatalf("%v failed: %v\nstdout=%q stderr=%q", tt.equivalent, eqErr, wantOut, wantErr)
				}
				if stdout != wantOut {
					t.Errorf("%v stdout = %q, want %q (output of %v)", tt.args, stdout, wantOut, tt.equivalent)
				}
			}

			after := readRepoBacklog(t, repo)
			if tt.wantAdded != "" {
				if !strings.HasPrefix(stdout, "Added: [") {
					t.Errorf("expected 'Added: [' prefix on stdout, got %q", stdout)
				}
				if !strings.Contains(after, tt.wantAdded) {
					t.Errorf("expected backlog to gain idea %q, got:\n%s", tt.wantAdded, after)
				}
			} else if after != before {
				t.Errorf("backlog changed by %v:\nbefore:\n%s\nafter:\n%s", tt.args, before, after)
			}
		})
	}
}

// TestDone_BackfillNoticeOnStderr verifies the advisory backfill notice goes to
// stderr (with the correct count) while stdout carries only the confirmation.
func TestDone_BackfillNoticeOnStderr(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	// Two dateless ideas; marking one done forces a save that backfills BOTH
	// (normalize-on-write), so the notice should report 2.
	writeRepoBacklog(t, repo, "- [ ] [rk7t] dateless one\n- [ ] [c3d4] dateless two\n")

	stdout, stderr, err := runSplit(t, bin, repo, "done", "rk7t")
	if err != nil {
		t.Fatalf("done failed: %v\nstdout=%q stderr=%q", err, stdout, stderr)
	}
	if !strings.Contains(stderr, "note: stamped today's date on 2 previously-dateless item(s)") {
		t.Errorf("expected backfill notice on stderr, got stderr=%q", stderr)
	}
	if strings.Contains(stdout, "stamped today's date") {
		t.Errorf("backfill notice leaked onto stdout: %q", stdout)
	}
	if !strings.HasPrefix(stdout, "Done: ") {
		t.Errorf("expected confirmation on stdout, got %q", stdout)
	}
}

// TestDone_NoBackfillNoticeWhenDated verifies the notice is suppressed when no
// dateless items are stamped (count == 0).
func TestDone_NoBackfillNoticeWhenDated(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	writeRepoBacklog(t, repo, "- [ ] [rk7t] 2025-06-15: already dated\n")

	stdout, stderr, err := runSplit(t, bin, repo, "done", "rk7t")
	if err != nil {
		t.Fatalf("done failed: %v\nstdout=%q stderr=%q", err, stdout, stderr)
	}
	if strings.Contains(stderr, "stamped today's date") {
		t.Errorf("expected no backfill notice on stderr, got %q", stderr)
	}
	if !strings.HasPrefix(stdout, "Done: ") {
		t.Errorf("expected confirmation on stdout, got %q", stdout)
	}
}

// TestAdd_MultilineTextSingleLine pins the multiline-escape contract end to
// end: adding text with embedded newlines grows the backlog by exactly one
// physical line and the Added: confirmation stays a single escaped line.
func TestAdd_MultilineTextSingleLine(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	text := "first line\n\nsecond paragraph\n- [ ] looks like a task"
	stdout, stderr, err := runSplit(t, bin, repo, "add", "--id", "ab12", text)
	if err != nil {
		t.Fatalf("add failed: %v\nstdout=%q stderr=%q", err, stdout, stderr)
	}
	if got := strings.Count(stdout, "\n"); got != 1 {
		t.Errorf("Added: confirmation spans %d lines, want 1: %q", got, stdout)
	}
	if !strings.HasPrefix(stdout, "Added: [ab12]") {
		t.Errorf("unexpected confirmation prefix: %q", stdout)
	}
	if !strings.Contains(stdout, `first line\n\nsecond paragraph\n- [ ] looks like a task`) {
		t.Errorf("confirmation should carry the escaped text, got %q", stdout)
	}

	data, err := os.ReadFile(filepath.Join(repo, "fab", "backlog.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := strings.TrimSuffix(string(data), "\n")
	if lines := strings.Split(content, "\n"); len(lines) != 2 { // "# Backlog" + 1 idea line
		t.Errorf("backlog has %d physical lines, want 2:\n%q", len(lines), string(data))
	}
}

// TestShow_MultilineRendersRealNewlines verifies plain show unescapes for
// display while --json carries real newlines in the text field.
func TestShow_MultilineRendersRealNewlines(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)
	writeRepoBacklog(t, repo, "- [ ] [ab12] 2026-06-10: first line\\n\\nsecond paragraph\n")

	stdout, stderr, err := runSplit(t, bin, repo, "show", "ab12")
	if err != nil {
		t.Fatalf("show failed: %v\nstdout=%q stderr=%q", err, stdout, stderr)
	}
	want := "- [ ] [ab12] 2026-06-10: first line\n\nsecond paragraph\n"
	if stdout != want {
		t.Errorf("show output:\ngot:  %q\nwant: %q", stdout, want)
	}

	jsonOut, _, err := runSplit(t, bin, repo, "show", "ab12", "--json")
	if err != nil {
		t.Fatalf("show --json failed: %v", err)
	}
	var decoded struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &decoded); err != nil {
		t.Fatalf("decode show --json: %v\noutput=%q", err, jsonOut)
	}
	if decoded.Text != "first line\n\nsecond paragraph" {
		t.Errorf("json text = %q, want real newlines", decoded.Text)
	}
}

// TestList_MultilineStaysEscapedSingleLine verifies list keeps the
// line-per-record guarantee: multiline ideas print in escaped form.
func TestList_MultilineStaysEscapedSingleLine(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)
	writeRepoBacklog(t, repo, "- [ ] [ab12] 2026-06-10: first line\\n\\nsecond paragraph\n")

	stdout, stderr, err := runSplit(t, bin, repo, "list")
	if err != nil {
		t.Fatalf("list failed: %v\nstdout=%q stderr=%q", err, stdout, stderr)
	}
	want := "- [ ] [ab12] 2026-06-10: first line\\n\\nsecond paragraph\n"
	if stdout != want {
		t.Errorf("list output:\ngot:  %q\nwant: %q", stdout, want)
	}
}

// TestPrune_CLIOutputContract verifies the prune wiring end to end: the bare
// dry run lists the removable lines on stdout with the confirm hint on stderr
// and writes nothing; --force rewrites the file and prints a count only; the
// empty case prints the no-op message and leaves the file untouched in both
// modes. exit 0 on every path is implied by err == nil from runSplit.
func TestPrune_CLIOutputContract(t *testing.T) {
	bin := buildBinary(t)

	mixed := "# Backlog\n\n- [ ] [op3n] 2026-06-01: keep me\n- [x] [d0ne] 2026-06-02: prune me\n- [x] [d1ne] 2026-06-03: prune me too\n"
	allOpen := "- [ ] [op3n] 2026-06-01: keep me\n"

	tests := []struct {
		name        string
		backlog     string
		args        []string
		wantStdout  string
		wantStderr  string
		wantBacklog string // "" means unchanged
	}{
		{
			name:       "dry run lists done ideas on stdout, count header + hint on stderr",
			backlog:    mixed,
			args:       []string{"prune"},
			wantStdout: "- [x] [d0ne] 2026-06-02: prune me\n- [x] [d1ne] 2026-06-03: prune me too\n",
			// Non-TTY (piped) path: the leading count header (feature B) plus
			// the classic trailing fallback hint, in that order.
			wantStderr: "2 done idea(s) would be pruned\nRe-run with --force to confirm.\n",
		},
		{
			name:        "force prints count only and removes the done lines",
			backlog:     mixed,
			args:        []string{"prune", "--force"},
			wantStdout:  "Pruned 2 done idea(s).\n",
			wantStderr:  "",
			wantBacklog: "# Backlog\n\n- [ ] [op3n] 2026-06-01: keep me\n",
		},
		{
			name:       "no done ideas dry run",
			backlog:    allOpen,
			args:       []string{"prune"},
			wantStdout: "No done ideas to prune.\n",
			wantStderr: "",
		},
		{
			name:       "no done ideas force",
			backlog:    allOpen,
			args:       []string{"prune", "--force"},
			wantStdout: "No done ideas to prune.\n",
			wantStderr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupGitRepo(t)
			writeRepoBacklog(t, repo, tt.backlog)

			stdout, stderr, err := runSplit(t, bin, repo, tt.args...)
			if err != nil {
				t.Fatalf("%v failed: %v\nstdout=%q stderr=%q", tt.args, err, stdout, stderr)
			}
			if stdout != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, tt.wantStdout)
			}
			if stderr != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr, tt.wantStderr)
			}

			want := tt.wantBacklog
			if want == "" {
				want = tt.backlog
			}
			if got := readRepoBacklog(t, repo); got != want {
				t.Errorf("backlog:\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// writeEditorScript writes an executable shell script (a fake $EDITOR or
// $VISUAL) into a temp dir and returns its path. The body runs under sh with
// the editor buffer path as $1.
func writeEditorScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-editor.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

// editorEnv returns the inherited environment with any host VISUAL/EDITOR
// scrubbed (so the developer's or CI's editor cannot shadow the per-case
// fakes — resolution order is under test), plus the given overrides.
func editorEnv(overrides ...string) []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "VISUAL=") || strings.HasPrefix(kv, "EDITOR=") {
			continue
		}
		env = append(env, kv)
	}
	return append(env, overrides...)
}

// runSplitEnv is runSplit with an explicit environment.
func runSplitEnv(t *testing.T, bin, repo string, env []string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = repo
	cmd.Env = env
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// TestEdit_EditorForm covers the one-arg editor-based form of `idea edit`
// end to end with fake $EDITOR/$VISUAL shell scripts, plus the two-arg
// tripwire regression. The side file (exposed to scripts as $SIDE) doubles
// as the editor-invocation tripwire and the captured-buffer recorder.
func TestEdit_EditorForm(t *testing.T) {
	bin := buildBinary(t)

	const seed = "# Backlog\n\n- [ ] [ab12] 2026-06-10: hello\n"

	tests := []struct {
		name        string
		backlog     string // initial backlog; defaults to seed when ""
		editor      string // $EDITOR fake script body ("" = EDITOR unset)
		visual      string // $VISUAL fake script body ("" = VISUAL unset)
		args        []string
		wantErr     bool
		wantStdout  string   // exact stdout
		wantStderr  []string // required stderr substrings
		noStderr    []string // forbidden stderr substrings
		wantBacklog string   // expected final backlog; "" = byte-identical to initial
		wantSide    string   // required side-file content (checked when non-empty)
		sideAbsent  bool     // assert the side file was never created (tripwire)
	}{
		{
			name:        "editor rewrite persists canonically",
			editor:      `printf 'rewritten by editor\n' > "$1"`,
			args:        []string{"edit", "ab12"},
			wantStdout:  "Updated: - [ ] [ab12] 2026-06-10: rewritten by editor\n",
			wantBacklog: "# Backlog\n\n- [ ] [ab12] 2026-06-10: rewritten by editor\n",
		},
		{
			name:        "multiline round-trip: decoded buffer in, re-escaped line out",
			backlog:     "# Backlog\n\n- [ ] [ab12] 2026-06-10: a\\nb\n",
			editor:      `cat "$1" > "$SIDE"; printf 'x\ny\n' > "$1"`,
			args:        []string{"edit", "ab12"},
			wantStdout:  "Updated: - [ ] [ab12] 2026-06-10: x\\ny\n",
			wantBacklog: "# Backlog\n\n- [ ] [ab12] 2026-06-10: x\\ny\n",
			wantSide:    "a\nb", // the editor saw DECODED text: real newline, no escapes
		},
		{
			name:       "trailing LF appended by editor does not change the idea",
			editor:     `printf '\n' >> "$1"`,
			args:       []string{"edit", "ab12"},
			wantStdout: "",
			wantStderr: []string{"note: text unchanged — nothing to do"},
		},
		{
			name:       "editor non-zero exit aborts without changes",
			editor:     `printf 'discarded edit\n' > "$1"; exit 3`,
			args:       []string{"edit", "ab12"},
			wantErr:    true,
			wantStdout: "",
			wantStderr: []string{"ERROR:", "idea unchanged"},
		},
		{
			name:       "unchanged buffer is a no-op with no normalize side effect",
			backlog:    "# Backlog\n\n- [ ] [ab12] 2026-06-10: hello\n* [ ] [zz99] dateless passenger\n",
			editor:     `exit 0`,
			args:       []string{"edit", "ab12"},
			wantStdout: "",
			wantStderr: []string{"note: text unchanged — nothing to do"},
		},
		{
			// The stored text itself ends in an LF (escaped `\n` on disk).
			// An untouched session must be a byte-identical no-op: the
			// pre-strip buffer equals the text even though the strip-one
			// rule would eat the trailing LF.
			name:       "untouched LF-terminated text is a byte-identical no-op",
			backlog:    "# Backlog\n\n- [ ] [ab12] 2026-06-10: hello\\n\n",
			editor:     `exit 0`,
			args:       []string{"edit", "ab12"},
			wantStdout: "",
			wantStderr: []string{"note: text unchanged — nothing to do"},
		},
		{
			// Metadata-only save on unchanged LF-terminated text: the date
			// changes but the text (including its trailing LF) is preserved
			// verbatim — the forced save must not apply the strip-one rule.
			name:        "--date on unchanged LF-terminated text preserves the text verbatim",
			backlog:     "# Backlog\n\n- [ ] [ab12] 2026-06-10: hello\\n\n",
			editor:      `exit 0`,
			args:        []string{"edit", "ab12", "--date", "2026-01-01"},
			wantStdout:  "Updated: - [ ] [ab12] 2026-01-01: hello\\n\n",
			noStderr:    []string{"unchanged"},
			wantBacklog: "# Backlog\n\n- [ ] [ab12] 2026-01-01: hello\\n\n",
		},
		{
			name:       "emptied buffer is refused",
			editor:     `: > "$1"`,
			args:       []string{"edit", "ab12"},
			wantErr:    true,
			wantStdout: "",
			wantStderr: []string{"ERROR:", "empty"},
		},
		{
			name:        "VISUAL wins over EDITOR",
			editor:      `touch "$SIDE"`, // tripwire: must not run
			visual:      `printf 'via visual\n' > "$1"`,
			args:        []string{"edit", "ab12"},
			wantStdout:  "Updated: - [ ] [ab12] 2026-06-10: via visual\n",
			wantBacklog: "# Backlog\n\n- [ ] [ab12] 2026-06-10: via visual\n",
			sideAbsent:  true,
		},
		{
			name:        "two-arg form never launches the editor",
			editor:      `touch "$SIDE"`, // tripwire: must not run
			args:        []string{"edit", "ab12", "inline replacement"},
			wantStdout:  "Updated: - [ ] [ab12] 2026-06-10: inline replacement\n",
			wantBacklog: "# Backlog\n\n- [ ] [ab12] 2026-06-10: inline replacement\n",
			sideAbsent:  true,
		},
		{
			name:        "--date with unchanged text suppresses the no-op",
			editor:      `exit 0`,
			args:        []string{"edit", "ab12", "--date", "2026-01-01"},
			wantStdout:  "Updated: - [ ] [ab12] 2026-01-01: hello\n",
			noStderr:    []string{"unchanged"},
			wantBacklog: "# Backlog\n\n- [ ] [ab12] 2026-01-01: hello\n",
		},
		{
			name:       "ambiguous query refused before editor launch",
			backlog:    "# Backlog\n\n- [ ] [ab12] 2026-06-10: hello one\n- [ ] [cd34] 2026-06-10: hello two\n",
			editor:     `touch "$SIDE"`, // tripwire: must not run
			args:       []string{"edit", "hello"},
			wantErr:    true,
			wantStdout: "",
			wantStderr: []string{"Multiple matches"},
			sideAbsent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupGitRepo(t)
			initial := tt.backlog
			if initial == "" {
				initial = seed
			}
			writeRepoBacklog(t, repo, initial)

			side := filepath.Join(t.TempDir(), "side")
			env := editorEnv("SIDE=" + side)
			if tt.editor != "" {
				env = append(env, "EDITOR="+writeEditorScript(t, tt.editor))
			}
			if tt.visual != "" {
				env = append(env, "VISUAL="+writeEditorScript(t, tt.visual))
			}

			stdout, stderr, err := runSplitEnv(t, bin, repo, env, tt.args...)
			if tt.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr = %v\nstdout=%q stderr=%q", err, tt.wantErr, stdout, stderr)
			}
			if stdout != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, tt.wantStdout)
			}
			for _, want := range tt.wantStderr {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr = %q, want substring %q", stderr, want)
				}
			}
			for _, banned := range tt.noStderr {
				if strings.Contains(stderr, banned) {
					t.Errorf("stderr = %q, must not contain %q", stderr, banned)
				}
			}

			wantBacklog := tt.wantBacklog
			if wantBacklog == "" {
				wantBacklog = initial
			}
			if got := readRepoBacklog(t, repo); got != wantBacklog {
				t.Errorf("backlog:\ngot:  %q\nwant: %q", got, wantBacklog)
			}

			if tt.sideAbsent {
				if _, statErr := os.Stat(side); !os.IsNotExist(statErr) {
					t.Errorf("editor tripwire fired: side file exists (stat err = %v)", statErr)
				}
			}
			if tt.wantSide != "" {
				b, readErr := os.ReadFile(side)
				if readErr != nil {
					t.Fatalf("read side file: %v", readErr)
				}
				if string(b) != tt.wantSide {
					t.Errorf("editor buffer = %q, want %q", string(b), tt.wantSide)
				}
			}
		})
	}
}

// TestEdit_MultilineSingleLineConfirmation verifies the Updated: confirmation
// (via FormatLine) stays a single escaped line for multiline replacement text.
func TestEdit_MultilineSingleLineConfirmation(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)
	writeRepoBacklog(t, repo, "- [ ] [ab12] 2026-06-10: old text\n")

	stdout, stderr, err := runSplit(t, bin, repo, "edit", "ab12", "new first\nnew second")
	if err != nil {
		t.Fatalf("edit failed: %v\nstdout=%q stderr=%q", err, stdout, stderr)
	}
	want := "Updated: - [ ] [ab12] 2026-06-10: new first\\nnew second\n"
	if stdout != want {
		t.Errorf("edit confirmation:\ngot:  %q\nwant: %q", stdout, want)
	}
}

// TestList_IDFilter covers the optional [id...] positional filter: listing only
// the requested IDs, the warn-and-list-the-rest behavior for well-formed-but-
// absent IDs, and the usage error for a malformed ID. The subprocess pipes its
// output, so list emits canonical FormatLine lines (the non-TTY path) — which
// also pins the pipe contract (no ANSI, no truncation).
func TestList_IDFilter(t *testing.T) {
	bin := buildBinary(t)

	const seed = "# Backlog\n\n" +
		"- [ ] [ab12] 2026-06-01: first idea\n" +
		"- [ ] [cd34] 2026-06-02: second idea\n" +
		"- [ ] [ef56] 2026-06-03: third idea\n"

	tests := []struct {
		name       string
		args       []string
		wantStdout string
		wantStderr []string // required stderr substrings
		noStdout   []string // forbidden stdout substrings
		wantErr    bool
	}{
		{
			name:       "filter to a subset by id",
			args:       []string{"ls", "ab12", "ef56", "--sort", "id"},
			wantStdout: "- [ ] [ab12] 2026-06-01: first idea\n- [ ] [ef56] 2026-06-03: third idea\n",
			noStdout:   []string{"cd34"},
		},
		{
			name:       "unknown id warns on stderr and lists the rest",
			args:       []string{"ls", "ab12", "zzzz"},
			wantStdout: "- [ ] [ab12] 2026-06-01: first idea\n",
			wantStderr: []string{`no idea with ID "zzzz"`},
			noStdout:   []string{"zzzz"},
		},
		{
			name:       "all-unknown ids warn and fall through to empty message",
			args:       []string{"ls", "yyyy", "zzzz"},
			wantStdout: "No ideas found.\n",
			wantStderr: []string{`no idea with ID "yyyy"`, `no idea with ID "zzzz"`},
		},
		{
			name:    "malformed id is a usage error",
			args:    []string{"ls", "TOOLONG"},
			wantErr: true,
			wantStderr: []string{
				"invalid ID",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupGitRepo(t)
			writeRepoBacklog(t, repo, seed)

			stdout, stderr, err := runSplit(t, bin, repo, tt.args...)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none\nstdout=%q stderr=%q", stdout, stderr)
				}
			} else if err != nil {
				t.Fatalf("%v failed: %v\nstdout=%q stderr=%q", tt.args, err, stdout, stderr)
			}

			if tt.wantStdout != "" && stdout != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, tt.wantStdout)
			}
			for _, sub := range tt.wantStderr {
				if !strings.Contains(stderr, sub) {
					t.Errorf("stderr %q missing %q", stderr, sub)
				}
			}
			for _, sub := range tt.noStdout {
				if strings.Contains(stdout, sub) {
					t.Errorf("stdout %q should not contain %q", stdout, sub)
				}
			}
		})
	}
}

// TestList_PipedOutputIsCanonical pins the pipe contract: piped `ls` and
// `ls --full` emit byte-identical canonical FormatLine output (no ANSI, no
// ellipsis truncation) regardless of --full, since the subprocess is not a TTY.
func TestList_PipedOutputIsCanonical(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)
	// A long idea that WOULD truncate on a narrow terminal.
	long := strings.Repeat("x", 300)
	writeRepoBacklog(t, repo, "- [ ] [ab12] 2026-06-01: "+long+"\n")

	plain, _, err := runSplit(t, bin, repo, "ls")
	if err != nil {
		t.Fatalf("ls failed: %v", err)
	}
	full, _, err := runSplit(t, bin, repo, "ls", "--full")
	if err != nil {
		t.Fatalf("ls --full failed: %v", err)
	}
	want := "- [ ] [ab12] 2026-06-01: " + long + "\n"
	if plain != want {
		t.Errorf("piped ls stdout:\ngot:  %q\nwant: %q", plain, want)
	}
	if full != want {
		t.Errorf("piped ls --full stdout:\ngot:  %q\nwant: %q", full, want)
	}
	if strings.Contains(plain, "\033[") || strings.Contains(plain, "…") {
		t.Errorf("piped output leaked ANSI/ellipsis: %q", plain)
	}
}

// TestPrune_CountHeaderAndDecisionMatrix covers feature B (the leading stderr
// count header) and the non-TTY rows of the feature E decision matrix. The
// subprocess pipes stdout (never a TTY), so: no-force prints the header + the
// removable lines on stdout + the trailing fallback hint and never prompts;
// --force deletes immediately. The interactive TTY prompt is verified at the
// seam level (internal/idea term tests) since it requires a real terminal.
func TestPrune_CountHeaderAndDecisionMatrix(t *testing.T) {
	bin := buildBinary(t)

	mixed := "# Backlog\n\n" +
		"- [ ] [op3n] 2026-06-01: keep me\n" +
		"- [x] [d0ne] 2026-06-02: prune me\n" +
		"- [x] [d1ne] 2026-06-03: prune me too\n"

	tests := []struct {
		name        string
		args        []string
		wantStdout  string
		wantStderr  []string // required stderr substrings (in addition to header)
		noStderr    []string // forbidden stderr substrings
		wantBacklog string   // "" means unchanged
	}{
		{
			name:       "non-tty no-force: header + lines on stdout + fallback hint, no prompt",
			args:       []string{"prune"},
			wantStdout: "- [x] [d0ne] 2026-06-02: prune me\n- [x] [d1ne] 2026-06-03: prune me too\n",
			wantStderr: []string{"2 done idea(s) would be pruned", "Re-run with --force to confirm."},
			noStderr:   []string{"[y/N]"},
		},
		{
			name:        "non-tty force: deletes immediately, count only",
			args:        []string{"prune", "--force"},
			wantStdout:  "Pruned 2 done idea(s).\n",
			noStderr:    []string{"would be pruned", "[y/N]"},
			wantBacklog: "# Backlog\n\n- [ ] [op3n] 2026-06-01: keep me\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := setupGitRepo(t)
			writeRepoBacklog(t, repo, mixed)

			stdout, stderr, err := runSplit(t, bin, repo, tt.args...)
			if err != nil {
				t.Fatalf("%v failed: %v\nstdout=%q stderr=%q", tt.args, err, stdout, stderr)
			}
			if stdout != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, tt.wantStdout)
			}
			for _, sub := range tt.wantStderr {
				if !strings.Contains(stderr, sub) {
					t.Errorf("stderr %q missing %q", stderr, sub)
				}
			}
			for _, sub := range tt.noStderr {
				if strings.Contains(stderr, sub) {
					t.Errorf("stderr %q should not contain %q", stderr, sub)
				}
			}

			want := tt.wantBacklog
			if want == "" {
				want = mixed
			}
			if got := readRepoBacklog(t, repo); got != want {
				t.Errorf("backlog:\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// TestConfirmPrune covers the [y/N] confirm logic in-process (feature E). It
// reads cmd.InOrStdin(), so it is unit-testable without a PTY by feeding a
// buffer: only "y"/"yes" (case-insensitive, trimmed) confirm; everything else —
// including bare Enter and EOF (no input) — aborts. The TTY gating that decides
// whether confirmPrune is even reached is exercised by the seam tests; here we
// pin the decision itself.
func TestConfirmPrune(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "y confirms", input: "y\n", want: true},
		{name: "yes confirms", input: "yes\n", want: true},
		{name: "uppercase Y confirms", input: "Y\n", want: true},
		{name: "YES confirms", input: "YES\n", want: true},
		{name: "y with surrounding spaces confirms", input: "  y  \n", want: true},
		{name: "n aborts", input: "n\n", want: false},
		{name: "no aborts", input: "no\n", want: false},
		{name: "bare enter aborts", input: "\n", want: false},
		{name: "EOF (no input) aborts", input: "", want: false},
		{name: "garbage aborts", input: "yep\n", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newRootCmd()
			cmd.SetIn(strings.NewReader(tt.input))
			cmd.SetErr(&bytes.Buffer{}) // swallow the prompt
			if got := confirmPrune(cmd, 2); got != tt.want {
				t.Errorf("confirmPrune(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestPrune_ConfirmedDeleteAndAbort proves the two file-state outcomes of the
// interactive confirm (closing acceptance A-008's TTY rows and A-011 with real
// tests rather than code inspection): a "y"/"yes" answer deletes exactly like
// --force, while any abort answer leaves the backlog byte-identical. It mirrors
// the prune RunE guard — idea.Prune(path, true) runs only when confirmPrune
// returns true — using the real internal/idea seam, so the confirm logic and
// the resulting write (or no-write) are tested together without a PTY.
func TestPrune_ConfirmedDeleteAndAbort(t *testing.T) {
	const mixed = "# Backlog\n\n" +
		"- [ ] [op3n] 2026-06-01: keep me\n" +
		"- [x] [d0ne] 2026-06-02: prune me\n" +
		"- [x] [d1ne] 2026-06-03: prune me too\n"
	const pruned = "# Backlog\n\n- [ ] [op3n] 2026-06-01: keep me\n"

	tests := []struct {
		name        string
		input       string
		wantConfirm bool
		wantBacklog string
	}{
		{name: "y deletes the done ideas", input: "y\n", wantConfirm: true, wantBacklog: pruned},
		{name: "yes deletes the done ideas", input: "yes\n", wantConfirm: true, wantBacklog: pruned},
		{name: "n leaves the backlog byte-identical", input: "n\n", wantConfirm: false, wantBacklog: mixed},
		{name: "EOF leaves the backlog byte-identical", input: "", wantConfirm: false, wantBacklog: mixed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "backlog.md")
			if err := os.WriteFile(path, []byte(mixed), 0644); err != nil {
				t.Fatal(err)
			}

			cmd := newRootCmd()
			cmd.SetIn(strings.NewReader(tt.input))
			cmd.SetErr(&bytes.Buffer{})

			// Mirror the prune RunE guard exactly: dry run first (never writes),
			// then delete only on a confirmed answer.
			if _, _, err := idea.Prune(path, false); err != nil {
				t.Fatalf("dry-run prune: %v", err)
			}
			confirmed := confirmPrune(cmd, 2)
			if confirmed != tt.wantConfirm {
				t.Fatalf("confirmPrune = %v, want %v", confirmed, tt.wantConfirm)
			}
			if confirmed {
				if _, _, err := idea.Prune(path, true); err != nil {
					t.Fatalf("confirmed prune: %v", err)
				}
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.wantBacklog {
				t.Errorf("backlog:\ngot:\n%s\nwant:\n%s", got, tt.wantBacklog)
			}
		})
	}
}

// --- System Backlog & Out-of-Git Operation ---

// systemEnv builds a minimal environment that isolates HOME/XDG_CONFIG_HOME at
// the given config dir while preserving PATH (git lookups inside the binary
// need it). Returns the env slice and the resolved system backlog path.
func systemEnv(t *testing.T) (env []string, configDir, backlogPath string) {
	t.Helper()
	configDir = t.TempDir()
	env = []string{
		"HOME=" + configDir,
		"XDG_CONFIG_HOME=" + configDir,
		"PATH=" + os.Getenv("PATH"),
	}
	backlogPath = filepath.Join(configDir, "idea", "backlog.md")
	return env, configDir, backlogPath
}

// TestSystem_OutOfGitFallback verifies that, outside any git repo, add/list
// gracefully fall back to the system backlog instead of failing with
// "not in a git repository".
func TestSystem_OutOfGitFallback(t *testing.T) {
	bin := buildBinary(t)
	nonGit := t.TempDir() // not a git repo
	env, _, backlogPath := systemEnv(t)

	stdout, stderr, err := runSplitEnv(t, bin, nonGit, env, "add", "buy milk")
	if err != nil {
		t.Fatalf("add outside git failed: %v\nstdout=%q stderr=%q", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "buy milk") {
		t.Errorf("expected idea text in output, got: %q", stdout)
	}
	b, readErr := os.ReadFile(backlogPath)
	if readErr != nil {
		t.Fatalf("system backlog not written at %s: %v", backlogPath, readErr)
	}
	if !strings.Contains(string(b), "buy milk") {
		t.Errorf("system backlog missing idea, got: %q", b)
	}

	// list outside git reads back the same system backlog.
	stdout, stderr, err = runSplitEnv(t, bin, nonGit, env, "list")
	if err != nil {
		t.Fatalf("list outside git failed: %v\nstdout=%q stderr=%q", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "buy milk") {
		t.Errorf("list outside git missing idea, got: %q", stdout)
	}
}

// TestSystem_FlagInsideRepo verifies that --system targets the system backlog
// even when run inside a git repo, leaving the repo backlog untouched.
func TestSystem_FlagInsideRepo(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)
	env, _, backlogPath := systemEnv(t)

	stdout, stderr, err := runSplitEnv(t, bin, repo, env, "--system", "global todo")
	if err != nil {
		t.Fatalf("--system add in repo failed: %v\nstdout=%q stderr=%q", err, stdout, stderr)
	}

	// The system backlog gets the idea...
	b, readErr := os.ReadFile(backlogPath)
	if readErr != nil {
		t.Fatalf("system backlog not written at %s: %v", backlogPath, readErr)
	}
	if !strings.Contains(string(b), "global todo") {
		t.Errorf("system backlog missing idea, got: %q", b)
	}
	// ...and the repo backlog stays empty of it (only the seed header).
	if repoContent := readRepoBacklog(t, repo); strings.Contains(repoContent, "global todo") {
		t.Errorf("repo backlog should not contain the --system idea, got: %q", repoContent)
	}
}

// TestSystem_ConflictWithMain verifies --system + --main is a user error.
func TestSystem_ConflictWithMain(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)
	env, _, _ := systemEnv(t)

	stdout, stderr, err := runSplitEnv(t, bin, repo, env, "--system", "--main", "x")
	if err == nil {
		t.Fatalf("expected non-zero exit for --system --main, got success\nstdout=%q", stdout)
	}
	if !strings.Contains(stderr, "mutually exclusive") {
		t.Errorf("expected conflict message on stderr, got: %q", stderr)
	}
}

// TestSystem_OnDemandDirCreation verifies the config dir is created on the
// first mutating write when it does not yet exist.
func TestSystem_OnDemandDirCreation(t *testing.T) {
	bin := buildBinary(t)
	nonGit := t.TempDir()
	env, configDir, backlogPath := systemEnv(t)

	// Precondition: the idea config dir does not exist yet.
	ideaDir := filepath.Join(configDir, "idea")
	if _, err := os.Stat(ideaDir); !os.IsNotExist(err) {
		t.Fatalf("precondition: %s should not exist, stat err=%v", ideaDir, err)
	}

	if _, stderr, err := runSplitEnv(t, bin, nonGit, env, "add", "first idea"); err != nil {
		t.Fatalf("first add failed: %v\nstderr=%q", err, stderr)
	}
	if _, err := os.Stat(backlogPath); err != nil {
		t.Fatalf("system backlog not created on first write: %v", err)
	}
}
