package idea

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestPrune pins the prune contract table-driven against real temp dirs:
// force removes every done idea while preserving non-idea lines verbatim, the
// dry run (force=false) never writes, the zero-done case skips the save
// entirely in both modes, and a dateless surviving open item is backfilled on
// the force save with the count reflected.
func TestPrune(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		force        bool
		wantIDs      []string // expected returned idea IDs, in file order
		wantBackfill int
		// wantFile is the expected file content after the call; the literal
		// {TODAY} is replaced with today's date. Equal to content for the
		// no-write cases.
		wantFile string
	}{
		{
			name: "force removes only done lines on a mixed file",
			content: "- [ ] [op3n] 2026-06-01: keep me\n" +
				"- [x] [d0ne] 2026-06-02: prune me\n" +
				"- [ ] [op4n] 2026-06-03: keep me too\n" +
				"- [x] [d1ne] 2026-06-04: prune me too\n",
			force:   true,
			wantIDs: []string{"d0ne", "d1ne"},
			wantFile: "- [ ] [op3n] 2026-06-01: keep me\n" +
				"- [ ] [op4n] 2026-06-03: keep me too\n",
		},
		{
			// The variant bullet and missing date would be canonicalized if
			// SaveFile ran; byte-identical content proves it did not.
			name:    "dry run returns done ideas and leaves the file byte-identical",
			content: "- [ ] [op3n] 2026-06-01: keep me\n* [x] [d0ne] dateless variant done\n",
			force:   false,
			wantIDs: []string{"d0ne"},
			wantFile: "- [ ] [op3n] 2026-06-01: keep me\n" +
				"* [x] [d0ne] dateless variant done\n",
		},
		{
			name:     "all-open dry run is a no-op",
			content:  "* [ ] [op3n] dateless open survivor\n",
			force:    false,
			wantIDs:  nil,
			wantFile: "* [ ] [op3n] dateless open survivor\n",
		},
		{
			// Same normalize-bait content as above: byte-identical output
			// proves force with zero done items skips the save entirely.
			name:     "all-open force skips the save entirely",
			content:  "* [ ] [op3n] dateless open survivor\n",
			force:    true,
			wantIDs:  nil,
			wantFile: "* [ ] [op3n] dateless open survivor\n",
		},
		{
			name: "force preserves non-idea lines verbatim",
			content: "# Backlog\n\nSome prose between items\n" +
				"- [x] [d0ne] 2026-06-02: prune me\n" +
				"- [ ] [op3n] 2026-06-01: keep me\n\nFooter\n",
			force:   true,
			wantIDs: []string{"d0ne"},
			wantFile: "# Backlog\n\nSome prose between items\n" +
				"- [ ] [op3n] 2026-06-01: keep me\n\nFooter\n",
		},
		{
			name: "force backfills a dateless surviving open item",
			content: "- [ ] [op3n] keep me, stamp me\n" +
				"- [x] [d0ne] 2026-06-02: prune me\n",
			force:        true,
			wantIDs:      []string{"d0ne"},
			wantBackfill: 1,
			wantFile:     "- [ ] [op3n] {TODAY}: keep me, stamp me\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeBacklog(t, dir, tt.content)

			// Capture today before the code under test stamps the backfill
			// date, so a midnight rollover cannot flake the test.
			today := time.Now().Format("2006-01-02")
			pruned, backfilled, err := Prune(path, tt.force)
			if err != nil {
				t.Fatalf("Prune: %v", err)
			}

			var gotIDs []string
			for _, i := range pruned {
				gotIDs = append(gotIDs, i.ID)
			}
			if !slices.Equal(gotIDs, tt.wantIDs) {
				t.Errorf("pruned IDs = %v, want %v", gotIDs, tt.wantIDs)
			}

			if backfilled != tt.wantBackfill {
				t.Errorf("backfilled = %d, want %d", backfilled, tt.wantBackfill)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			want := strings.ReplaceAll(tt.wantFile, "{TODAY}", today)
			if string(data) != want {
				t.Errorf("file content:\ngot:\n%s\nwant:\n%s", data, want)
			}
		})
	}
}

// TestPrune_MissingFile pins the error path: a missing backlog file errors
// naturally via LoadFile in both modes, matching the other mutating commands.
func TestPrune_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.md")
	for _, force := range []bool{false, true} {
		if _, _, err := Prune(path, force); err == nil {
			t.Errorf("Prune(force=%v) on missing file: expected error", force)
		}
	}
}
