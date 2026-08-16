package idea

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// readFile is the promote-test counterpart of the save assertions: raw bytes
// or fail.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestPromote pins the move contract table-driven against real temp dirs
// (Constitution V): id/date/status preserved verbatim on both open and done
// ideas, dateless ideas backfilled by the save seam with per-file counts
// returned, a missing destination file (or parent dir) created, and the
// happy-path source left canonical with the idea removed.
func TestPromote(t *testing.T) {
	tests := []struct {
		name string
		src  string
		// dst is the destination content; "" with dstAbsent means no
		// destination file exists at all.
		dst       string
		dstAbsent bool
		query     string
		wantID    string
		// wantSrc / wantDst are the expected file contents after the call;
		// the literal {TODAY} is replaced with today's date.
		wantSrc         string
		wantDst         string
		wantSrcBackfill int
		wantDstBackfill int
	}{
		{
			name:    "open idea moved with id and date preserved",
			src:     "# Backlog\n\n- [ ] [a7k2] 2026-06-01: move me\n- [ ] [b8k3] 2026-06-02: keep me\n",
			dst:     "# Backlog\n\n- [ ] [z9y8] 2026-05-01: already here\n",
			query:   "a7k2",
			wantID:  "a7k2",
			wantSrc: "# Backlog\n\n- [ ] [b8k3] 2026-06-02: keep me\n",
			wantDst: "# Backlog\n\n- [ ] [z9y8] 2026-05-01: already here\n- [ ] [a7k2] 2026-06-01: move me\n",
		},
		{
			name:    "done idea moved with status preserved",
			src:     "- [x] [d0ne] 2026-06-01: finished elsewhere\n",
			dst:     "# Backlog\n",
			query:   "d0ne",
			wantID:  "d0ne",
			wantSrc: "\n",
			wantDst: "# Backlog\n- [x] [d0ne] 2026-06-01: finished elsewhere\n",
		},
		{
			// The dateless promoted idea is backfilled by the destination
			// save; the dateless source survivor is backfilled by the source
			// save; each count is returned separately.
			name:            "dateless idea backfilled, counts returned per file",
			src:             "- [ ] [a7k2] move me, dateless\n- [ ] [b8k3] 2026-06-02: keep me\n- [ ] [c9l4] keep me too, dateless\n",
			dst:             "- [ ] [z9y8] dateless resident\n",
			query:           "a7k2",
			wantID:          "a7k2",
			wantSrc:         "- [ ] [b8k3] 2026-06-02: keep me\n- [ ] [c9l4] {TODAY}: keep me too, dateless\n",
			wantDst:         "- [ ] [z9y8] {TODAY}: dateless resident\n- [ ] [a7k2] {TODAY}: move me, dateless\n",
			wantSrcBackfill: 1,
			wantDstBackfill: 2,
		},
		{
			name:      "missing destination file is created",
			src:       "- [ ] [a7k2] 2026-06-01: move me\n",
			dstAbsent: true,
			query:     "a7k2",
			wantID:    "a7k2",
			wantSrc:   "\n",
			wantDst:   "- [ ] [a7k2] 2026-06-01: move me\n",
		},
		{
			// Substring resolution through the shared matcher.
			name:    "substring query resolves",
			src:     "- [ ] [a7k2] 2026-06-01: wire up dark mode toggle\n",
			dst:     "",
			query:   "dark mode",
			wantID:  "a7k2",
			wantSrc: "\n",
			wantDst: "- [ ] [a7k2] 2026-06-01: wire up dark mode toggle\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcDir := t.TempDir()
			dstDir := t.TempDir()
			srcPath := writeBacklog(t, srcDir, tt.src)
			dstPath := filepath.Join(dstDir, "backlog.md")
			if !tt.dstAbsent {
				if err := os.WriteFile(dstPath, []byte(tt.dst), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Capture today on both sides of the call: the backfill stamp
			// happens inside Promote, so on a midnight rollover either date
			// is legitimate and the assertion accepts both.
			before := time.Now().Format("2006-01-02")
			got, srcBackfilled, dstBackfilled, err := Promote(srcPath, dstPath, tt.query)
			after := time.Now().Format("2006-01-02")
			if err != nil {
				t.Fatalf("Promote: %v", err)
			}

			if got.ID != tt.wantID {
				t.Errorf("promoted ID = %q, want %q", got.ID, tt.wantID)
			}
			if srcBackfilled != tt.wantSrcBackfill {
				t.Errorf("srcBackfilled = %d, want %d", srcBackfilled, tt.wantSrcBackfill)
			}
			if dstBackfilled != tt.wantDstBackfill {
				t.Errorf("dstBackfilled = %d, want %d", dstBackfilled, tt.wantDstBackfill)
			}

			assertFile := func(path, want string) {
				t.Helper()
				wantBefore := strings.ReplaceAll(want, "{TODAY}", before)
				wantAfter := strings.ReplaceAll(want, "{TODAY}", after)
				if got := readFile(t, path); got != wantBefore && got != wantAfter {
					t.Errorf("%s:\ngot:\n%s\nwant:\n%s", path, got, wantBefore)
				}
			}
			assertFile(srcPath, tt.wantSrc)
			assertFile(dstPath, tt.wantDst)
		})
	}
}

// TestPromote_MissingDestinationDir pins that a missing destination parent
// directory is created on write (the atomicWriteFile MkdirAll seam).
func TestPromote_MissingDestinationDir(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := writeBacklog(t, srcDir, "- [ ] [a7k2] 2026-06-01: move me\n")
	dstPath := filepath.Join(t.TempDir(), "fab", "backlog.md")

	if _, _, _, err := Promote(srcPath, dstPath, "a7k2"); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	want := "- [ ] [a7k2] 2026-06-01: move me\n"
	if got := readFile(t, dstPath); got != want {
		t.Errorf("destination:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestPromote_CollisionRefuses pins the ID-collision refusal: an operational
// error naming the ID, no re-mint, and both files byte-identical afterwards.
func TestPromote_CollisionRefuses(t *testing.T) {
	srcContent := "- [ ] [a7k2] 2026-06-01: source copy\n"
	dstContent := "- [ ] [a7k2] 2026-05-01: destination copy\n"

	srcPath := writeBacklog(t, t.TempDir(), srcContent)
	dstPath := writeBacklog(t, t.TempDir(), dstContent)

	_, _, _, err := Promote(srcPath, dstPath, "a7k2")
	if err == nil {
		t.Fatal("expected collision error, got none")
	}
	if !strings.Contains(err.Error(), "a7k2") {
		t.Errorf("error should name the colliding ID, got: %v", err)
	}
	if got := readFile(t, srcPath); got != srcContent {
		t.Errorf("source modified on refusal:\ngot:\n%s\nwant:\n%s", got, srcContent)
	}
	if got := readFile(t, dstPath); got != dstContent {
		t.Errorf("destination modified on refusal:\ngot:\n%s\nwant:\n%s", got, dstContent)
	}
}

// TestPromote_QueryRefusals pins the shared-matcher failure modes: a no-match
// and an ambiguous match are refused before anything is written, and an exact
// ID wins over a coincidental substring match in another idea's text.
func TestPromote_QueryRefusals(t *testing.T) {
	srcContent := "- [ ] [a7k2] 2026-06-01: the real owner\n- [ ] [b8k3] 2026-06-02: mentions a7k2 in passing\n"

	tests := []struct {
		name      string
		query     string
		wantErr   string
		wantMoved string // ID expected in the destination on success; "" = error case
	}{
		{name: "no match", query: "zzzz", wantErr: "No idea matching 'zzzz'"},
		{name: "ambiguous match lists matches", query: "mentions", wantErr: ""},
		{name: "exact ID precedence over substring", query: "a7k2", wantMoved: "a7k2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ambiguity needs two matching ideas; reuse the shared seed for
			// the exact-ID case and add a second "mentions" idea for the
			// ambiguous one.
			content := srcContent
			if tt.name == "ambiguous match lists matches" {
				content = "- [ ] [b8k3] 2026-06-02: mentions one\n- [ ] [c9l4] 2026-06-03: mentions two\n"
			}
			srcPath := writeBacklog(t, t.TempDir(), content)
			dstPath := filepath.Join(t.TempDir(), "backlog.md")

			_, _, _, err := Promote(srcPath, dstPath, tt.query)

			if tt.wantMoved == "" {
				if err == nil {
					t.Fatalf("expected error for query %q, got none", tt.query)
				}
				if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %v, want substring %q", err, tt.wantErr)
				}
				if tt.wantErr == "" && !strings.Contains(err.Error(), "Multiple matches") {
					t.Errorf("error = %v, want an ambiguity refusal", err)
				}
				if got := readFile(t, srcPath); got != content {
					t.Errorf("source modified on refusal:\ngot:\n%s\nwant:\n%s", got, content)
				}
				if _, statErr := os.Stat(dstPath); !os.IsNotExist(statErr) {
					t.Errorf("destination created on refusal (stat err = %v)", statErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Promote: %v", err)
			}
			if got := readFile(t, dstPath); !strings.Contains(got, "["+tt.wantMoved+"]") {
				t.Errorf("destination missing %q:\n%s", tt.wantMoved, got)
			}
			if got := readFile(t, srcPath); strings.Contains(got, "["+tt.wantMoved+"]") {
				t.Errorf("source still holds %q:\n%s", tt.wantMoved, got)
			}
		})
	}
}

// TestPromote_DestinationWriteFailureLeavesSource pins the destination-first
// ordering: when the destination cannot be written (here the destination path
// is a directory, so loading it fails), the error propagates and the source
// file stays byte-identical.
func TestPromote_DestinationWriteFailureLeavesSource(t *testing.T) {
	srcContent := "- [ ] [a7k2] 2026-06-01: move me\n"
	srcPath := writeBacklog(t, t.TempDir(), srcContent)
	dstPath := t.TempDir() // a directory: the destination can never load/save

	if _, _, _, err := Promote(srcPath, dstPath, "a7k2"); err == nil {
		t.Fatal("expected error for directory destination, got none")
	}
	if got := readFile(t, srcPath); got != srcContent {
		t.Errorf("source modified despite destination failure:\ngot:\n%s\nwant:\n%s", got, srcContent)
	}
}

// TestPromote_MissingSource pins the error path: a missing source backlog
// errors naturally via LoadFile, matching the other mutating commands.
func TestPromote_MissingSource(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "nonexistent.md")
	dstPath := filepath.Join(t.TempDir(), "backlog.md")
	if _, _, _, err := Promote(srcPath, dstPath, "a7k2"); err == nil {
		t.Error("Promote on missing source: expected error")
	}
}
