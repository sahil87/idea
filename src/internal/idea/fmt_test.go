package idea

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// writeTempBacklog writes content to a fresh temp backlog file via the shared
// writeBacklog helper (idea_test.go) and returns its path.
func writeTempBacklog(t *testing.T, content string) string {
	t.Helper()
	return writeBacklog(t, t.TempDir(), content)
}

func readBacklog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// --- Canonicalization (existing normalize-on-write rule set, explicit trigger) ---

func TestFmt_Canonicalize(t *testing.T) {
	today := time.Now().Format("2006-01-02")

	tests := []struct {
		name       string
		in         string
		want       string
		normalized int
		backfilled int
		changed    bool
	}{
		{
			name:       "variant bullet and indentation",
			in:         "  * [x] [e5f6] 2025-06-08: fix bug\n+ [ ] [a7k2] 2025-06-15: add dark mode\n",
			want:       "- [x] [e5f6] 2025-06-08: fix bug\n- [ ] [a7k2] 2025-06-15: add dark mode\n",
			normalized: 2,
			changed:    true,
		},
		{
			name:       "dateless line backfills today",
			in:         "- [ ] [rk7t] tune the reporter\n",
			want:       "- [ ] [rk7t] " + today + ": tune the reporter\n",
			normalized: 1,
			backfilled: 1,
			changed:    true,
		},
		{
			name:       "legacy lone backslash doubles on disk",
			in:         "- [ ] [ab12] 2026-01-01: a\\b\n",
			want:       "- [ ] [ab12] 2026-01-01: a\\\\b\n",
			normalized: 1,
			changed:    true,
		},
		{
			name: "crlf endings normalize to lf",
			in:   "- [ ] [ab12] 2026-01-01: text\r\nprose line\r\n",
			want: "- [ ] [ab12] 2026-01-01: text\nprose line\n",
			// The raw line is stored post-\r-strip, so the per-line count
			// stays 0; the whole-file comparison still flags the change.
			normalized: 0,
			changed:    true,
		},
		{
			name:    "missing trailing newline gains one",
			in:      "- [ ] [ab12] 2026-01-01: text",
			want:    "- [ ] [ab12] 2026-01-01: text\n",
			changed: true,
		},
		{
			name:    "already canonical is untouched",
			in:      "# Backlog\n\n- [ ] [ab12] 2026-01-01: text\n- [x] [e5f6] 2025-06-08: done item\n",
			want:    "# Backlog\n\n- [ ] [ab12] 2026-01-01: text\n- [x] [e5f6] 2025-06-08: done item\n",
			changed: false,
		},
		{
			name:    "non-idea content verbatim",
			in:      "# Backlog\n\nsome prose between items\n\t indented prose\n",
			want:    "# Backlog\n\nsome prose between items\n\t indented prose\n",
			changed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempBacklog(t, tt.in)
			res, err := Fmt(path, false)
			if err != nil {
				t.Fatalf("Fmt failed: %v", err)
			}
			if got := readBacklog(t, path); got != tt.want {
				t.Errorf("file content:\ngot:  %q\nwant: %q", got, tt.want)
			}
			if res.Normalized != tt.normalized {
				t.Errorf("Normalized = %d, want %d", res.Normalized, tt.normalized)
			}
			if res.Backfilled != tt.backfilled {
				t.Errorf("Backfilled = %d, want %d", res.Backfilled, tt.backfilled)
			}
			if res.Changed != tt.changed {
				t.Errorf("Changed = %v, want %v", res.Changed, tt.changed)
			}
			if len(res.Adopted) != 0 {
				t.Errorf("Adopted = %v, want none", res.Adopted)
			}
		})
	}
}

// --- Adoption of bare checkboxes ---

func TestFmt_AdoptsBareCheckboxes(t *testing.T) {
	today := time.Now().Format("2006-01-02")

	tests := []struct {
		name     string
		in       string // the single candidate line (plus trailing newline)
		wantText string
		wantDone bool
	}{
		{"dash bullet open", "- [ ] buy milk\n", "buy milk", false},
		{"star bullet open", "* [ ] buy milk\n", "buy milk", false},
		{"plus bullet done", "+ [x] ship the release\n", "ship the release", true},
		{"uppercase X adopts as done", "- [X] ship the release\n", "ship the release", true},
		{"indented candidate", "   - [ ] indented task\n", "indented task", false},
		{"surrounding whitespace trims from adopted text", "- [ ]   padded task  \n", "padded task", false},
	}

	lineShape := regexp.MustCompile(`^- \[([ x])\] \[([a-z0-9]{4})\] (\d{4}-\d{2}-\d{2}): (.+)$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempBacklog(t, tt.in)
			res, err := Fmt(path, false)
			if err != nil {
				t.Fatalf("Fmt failed: %v", err)
			}
			if len(res.Adopted) != 1 {
				t.Fatalf("Adopted = %d ideas, want 1", len(res.Adopted))
			}
			a := res.Adopted[0]
			if a.Text != tt.wantText {
				t.Errorf("adopted Text = %q, want %q", a.Text, tt.wantText)
			}
			if a.Done != tt.wantDone {
				t.Errorf("adopted Done = %v, want %v", a.Done, tt.wantDone)
			}
			if a.Date != today {
				t.Errorf("adopted Date = %q, want today %q", a.Date, today)
			}
			if res.Backfilled != 0 {
				t.Errorf("Backfilled = %d, want 0 (adoption is not backfill)", res.Backfilled)
			}

			line := strings.TrimSuffix(readBacklog(t, path), "\n")
			m := lineShape.FindStringSubmatch(line)
			if m == nil {
				t.Fatalf("adopted line %q is not canonical", line)
			}
			if m[2] != a.ID {
				t.Errorf("on-disk ID = %q, want %q (from result)", m[2], a.ID)
			}
			wantCheck := " "
			if tt.wantDone {
				wantCheck = "x"
			}
			if m[1] != wantCheck {
				t.Errorf("on-disk checkbox = %q, want %q", m[1], wantCheck)
			}
		})
	}
}

func TestFmt_AdoptionGuards(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"shape B second bracket", "- [ ] [ni3o] [DEV-1011] 2026-02-12: capture more metrics"},
		{"bracket metadata external id", "- [ ] [DEV-1011] external item"},
		{"bracket metadata word", "- [ ] [TODO] buy milk"},
		{"non-4-char bracket", "- [ ] [ab1] text"},
		{"extra space cannot defeat bracket guard", "- [ ]  [DEV-1011] external item"},
		{"extra space before managed-looking line", "- [ ]  [ab12] 2026-01-01: text"},
		{"textless checkbox", "- [ ]"},
		{"whitespace-only text", "- [ ]   "},
		{"no space after checkbox", "- [ ]glued text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.line + "\n"
			path := writeTempBacklog(t, in)
			res, err := Fmt(path, false)
			if err != nil {
				t.Fatalf("Fmt failed: %v", err)
			}
			if len(res.Adopted) != 0 {
				t.Errorf("Adopted = %v, want none", res.Adopted)
			}
			if res.Changed {
				t.Error("Changed = true, want false")
			}
			if got := readBacklog(t, path); got != in {
				t.Errorf("line not preserved byte-for-byte:\ngot:  %q\nwant: %q", got, in)
			}
		})
	}
}

// TestFmt_WorkedExample pins the intake's worked example: mixed candidates are
// adopted in file order, the bracket-guarded line stays untouched, and
// surrounding non-idea content is verbatim.
func TestFmt_WorkedExample(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	in := "# Backlog\n\n* [ ] buy milk\n- [X] ship the release\n- [ ] [DEV-1011] external item\n"

	path := writeTempBacklog(t, in)
	res, err := Fmt(path, false)
	if err != nil {
		t.Fatalf("Fmt failed: %v", err)
	}
	if len(res.Adopted) != 2 {
		t.Fatalf("Adopted = %d ideas, want 2", len(res.Adopted))
	}
	if res.Adopted[0].Text != "buy milk" || res.Adopted[1].Text != "ship the release" {
		t.Errorf("Adopted order/text = [%q, %q], want file order", res.Adopted[0].Text, res.Adopted[1].Text)
	}

	want := "# Backlog\n\n" +
		"- [ ] [" + res.Adopted[0].ID + "] " + today + ": buy milk\n" +
		"- [x] [" + res.Adopted[1].ID + "] " + today + ": ship the release\n" +
		"- [ ] [DEV-1011] external item\n"
	if got := readBacklog(t, path); got != want {
		t.Errorf("file content:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestFmt_AdoptedIDsUnique asserts fresh IDs are unique against both the IDs
// already in the file and IDs assigned earlier in the same pass.
func TestFmt_AdoptedIDsUnique(t *testing.T) {
	var b strings.Builder
	b.WriteString("- [ ] [ab12] 2026-01-01: existing managed idea\n")
	for i := 0; i < 30; i++ {
		b.WriteString("- [ ] candidate task\n")
	}

	path := writeTempBacklog(t, b.String())
	res, err := Fmt(path, false)
	if err != nil {
		t.Fatalf("Fmt failed: %v", err)
	}
	if len(res.Adopted) != 30 {
		t.Fatalf("Adopted = %d ideas, want 30", len(res.Adopted))
	}

	seen := map[string]bool{"ab12": true}
	for _, a := range res.Adopted {
		if err := ValidateID(a.ID); err != nil {
			t.Errorf("adopted ID %q invalid: %v", a.ID, err)
		}
		if seen[a.ID] {
			t.Errorf("duplicate ID %q assigned", a.ID)
		}
		seen[a.ID] = true
	}
}

// TestFmt_AdoptedBackslashText verifies adopted text is treated as real text:
// a literal backslash doubles on disk and decodes back unchanged.
func TestFmt_AdoptedBackslashText(t *testing.T) {
	path := writeTempBacklog(t, "- [ ] open C:\\new in editor\n")

	res, err := Fmt(path, false)
	if err != nil {
		t.Fatalf("Fmt failed: %v", err)
	}
	if len(res.Adopted) != 1 {
		t.Fatalf("Adopted = %d ideas, want 1", len(res.Adopted))
	}
	if res.Adopted[0].Text != "open C:\\new in editor" {
		t.Errorf("adopted Text = %q, want the raw text", res.Adopted[0].Text)
	}
	if got := readBacklog(t, path); !strings.Contains(got, "open C:\\\\new in editor") {
		t.Errorf("on-disk text should be escaped (doubled backslash), got %q", got)
	}

	// Round-trip: the persisted line decodes back to the original text.
	f, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.ideas) != 1 || f.ideas[0].Text != "open C:\\new in editor" {
		t.Errorf("round-trip Text = %+v, want original text", f.ideas)
	}
}

// --- Counting separation ---

func TestFmt_CountsSeparation(t *testing.T) {
	in := "- [ ] [ab12] dateless managed\n" + // backfilled (and normalized: bytes change)
		"- [ ] bare candidate\n" + // adopted only
		"- [x] [e5f6] 2025-06-08: canonical\n" // untouched

	path := writeTempBacklog(t, in)
	res, err := Fmt(path, false)
	if err != nil {
		t.Fatalf("Fmt failed: %v", err)
	}
	if res.Backfilled != 1 {
		t.Errorf("Backfilled = %d, want 1 (adopted line must not count)", res.Backfilled)
	}
	if len(res.Adopted) != 1 {
		t.Errorf("Adopted = %d, want 1", len(res.Adopted))
	}
	if res.Normalized != 1 {
		t.Errorf("Normalized = %d, want 1 (the backfilled line; not the canonical or adopted ones)", res.Normalized)
	}
	if !res.Changed {
		t.Error("Changed = false, want true")
	}
}

// --- Idempotency & skip-write ---

func TestFmt_IdempotentSecondRun(t *testing.T) {
	path := writeTempBacklog(t, "  * [ ] [ab12] dateless variant\n- [ ] adopt me\n")

	res1, err := Fmt(path, false)
	if err != nil {
		t.Fatalf("first Fmt failed: %v", err)
	}
	if !res1.Changed {
		t.Fatal("first run should report Changed")
	}
	after1 := readBacklog(t, path)

	// Push mtime into the past so an (incorrect) rewrite would be detectable.
	past := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}

	res2, err := Fmt(path, false)
	if err != nil {
		t.Fatalf("second Fmt failed: %v", err)
	}
	if res2.Changed {
		t.Error("second run Changed = true, want false (byte-stable)")
	}
	if res2.Normalized != 0 || res2.Backfilled != 0 || len(res2.Adopted) != 0 {
		t.Errorf("second run counts = %+v, want all zero", res2)
	}
	if after2 := readBacklog(t, path); after2 != after1 {
		t.Errorf("second run changed bytes:\nfirst:  %q\nsecond: %q", after1, after2)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.ModTime().Equal(past) {
		t.Errorf("mtime churned on a byte-stable run: %v, want %v (write must be skipped)", fi.ModTime(), past)
	}
}

// --- Check mode ---

func TestFmt_CheckWritesNothing(t *testing.T) {
	in := "* [ ] [ab12] dateless variant\n- [ ] adopt me\n"
	path := writeTempBacklog(t, in)

	res, err := Fmt(path, true)
	if err != nil {
		t.Fatalf("Fmt --check failed: %v", err)
	}
	if !res.Changed {
		t.Error("Changed = false, want true (file is non-canonical)")
	}
	if res.Normalized != 1 || res.Backfilled != 1 || len(res.Adopted) != 1 {
		t.Errorf("check-mode counts = %+v, want the same report as a real run", res)
	}
	if got := readBacklog(t, path); got != in {
		t.Errorf("check mode modified the file:\ngot:  %q\nwant: %q", got, in)
	}
}

func TestFmt_CheckCleanFile(t *testing.T) {
	in := "# Backlog\n\n- [ ] [ab12] 2026-01-01: canonical\n"
	path := writeTempBacklog(t, in)

	res, err := Fmt(path, true)
	if err != nil {
		t.Fatalf("Fmt --check failed: %v", err)
	}
	if res.Changed {
		t.Error("Changed = true, want false on a canonical file")
	}
	if got := readBacklog(t, path); got != in {
		t.Error("check mode modified a canonical file")
	}
}

// --- Edge cases ---

func TestFmt_EmptyFileUntouched(t *testing.T) {
	path := writeTempBacklog(t, "")

	res, err := Fmt(path, false)
	if err != nil {
		t.Fatalf("Fmt failed: %v", err)
	}
	if res.Changed {
		t.Error("Changed = true, want false on an empty file")
	}
	if got := readBacklog(t, path); got != "" {
		t.Errorf("empty file gained content %q, want untouched", got)
	}
}

func TestFmt_MissingFileErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.md")
	if _, err := Fmt(path, false); err == nil {
		t.Error("expected an error for a missing file")
	}
}
