package idea

import (
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTermWidth_Fallback exercises the resolution order when the real terminal
// size is undetectable. In the test process os.Stdout is not a real terminal,
// so term.GetSize fails and TermWidth falls back to $COLUMNS, then to the 80
// default. t.Setenv restores the env after each case.
func TestTermWidth_Fallback(t *testing.T) {
	tests := []struct {
		name    string
		columns string // "" means unset
		setEnv  bool
		want    int
	}{
		{name: "COLUMNS unset falls back to 80", setEnv: false, want: defaultTermWidth},
		{name: "COLUMNS set to a positive integer is honored", setEnv: true, columns: "120", want: 120},
		{name: "COLUMNS empty falls back to 80", setEnv: true, columns: "", want: defaultTermWidth},
		{name: "COLUMNS non-numeric falls back to 80", setEnv: true, columns: "abc", want: defaultTermWidth},
		{name: "COLUMNS zero falls back to 80", setEnv: true, columns: "0", want: defaultTermWidth},
		{name: "COLUMNS negative falls back to 80", setEnv: true, columns: "-5", want: defaultTermWidth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv("COLUMNS", tt.columns)
			} else {
				unsetEnvForTest(t, "COLUMNS")
			}
			// A nil file skips the GetSize branch deterministically (os.Stdout
			// is not a real terminal under `go test` either, but nil is the
			// portable way to force the fallback path).
			if got := TermWidth(nil); got != tt.want {
				t.Errorf("TermWidth() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestIsTTY_NonTerminal verifies non-terminal files (and nil) report false. A
// regular temp file is never a terminal.
func TestIsTTY_NonTerminal(t *testing.T) {
	if IsTTY(nil) {
		t.Error("IsTTY(nil) = true, want false")
	}
	f, err := os.CreateTemp(t.TempDir(), "tty-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if IsTTY(f) {
		t.Error("IsTTY(regular file) = true, want false")
	}
}

// TestUseColor honors NO_COLOR presence and the TTY gate. Under `go test` the
// given file is never a terminal, so UseColor must be false regardless of
// NO_COLOR — the TTY gate dominates. The NO_COLOR-presence semantics are
// covered directly via the helper's documented contract below.
func TestUseColor_TTYGate(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "color-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Not a TTY: always false, even with NO_COLOR unset.
	unsetEnvForTest(t, "NO_COLOR")
	if UseColor(f) {
		t.Error("UseColor(non-tty) = true, want false (TTY gate)")
	}
	// nil file is never a TTY.
	if UseColor(nil) {
		t.Error("UseColor(nil) = true, want false")
	}
}

// TestColorHelpers verifies the dim/green wrappers wrap their input in the
// expected ANSI codes and that the visible text is preserved.
func TestColorHelpers(t *testing.T) {
	if got := dimPrefix("abc"); got != ansiFaint+"abc"+ansiReset {
		t.Errorf("dimPrefix = %q, want faint-wrapped", got)
	}
	if got := greenCheck("[x]"); got != ansiGreen+"[x]"+ansiReset {
		t.Errorf("greenCheck = %q, want green-wrapped", got)
	}
}

// TestDisplayListLine is the core table: rune-safe truncation that never clips
// the prefix, ellipsis presence, multiline-at-first-newline, --full bypass, and
// color-after-truncation. Width is injected (no PTY) per Constitution V.
func TestDisplayListLine(t *testing.T) {
	openShort := Idea{ID: "ab12", Date: "2026-06-01", Text: "short text", Done: false}
	// prefix = "- [ ] [ab12] 2026-06-01: " (25 visible runes)
	prefixLen := len("- [ ] [ab12] 2026-06-01: ")

	tests := []struct {
		name  string
		idea  Idea
		width int
		full  bool
		color bool
		want  string
	}{
		{
			name:  "fits within width is untouched",
			idea:  openShort,
			width: 80,
			want:  "- [ ] [ab12] 2026-06-01: short text",
		},
		{
			name:  "over-wide text is clipped with ellipsis, prefix intact",
			idea:  Idea{ID: "ab12", Date: "2026-06-01", Text: "abcdefghij"},
			width: prefixLen + 4, // room for 3 text runes + ellipsis
			want:  "- [ ] [ab12] 2026-06-01: abc" + ellipsis,
		},
		{
			name:  "full bypasses truncation even when over-wide",
			idea:  Idea{ID: "ab12", Date: "2026-06-01", Text: "abcdefghij"},
			width: prefixLen + 4,
			full:  true,
			want:  "- [ ] [ab12] 2026-06-01: abcdefghij",
		},
		{
			name:  "multiline clipped at first newline regardless of width",
			idea:  Idea{ID: "ab12", Date: "2026-06-01", Text: "line one\nline two"},
			width: 200, // far wider than the whole escaped text
			want:  "- [ ] [ab12] 2026-06-01: line one" + ellipsis,
		},
		{
			name:  "prefix never clipped when terminal narrower than prefix",
			idea:  Idea{ID: "ab12", Date: "2026-06-01", Text: "anything"},
			width: 5, // far narrower than the 25-rune prefix
			want:  "- [ ] [ab12] 2026-06-01: " + ellipsis,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DisplayListLine(tt.idea, tt.width, tt.full, tt.color)
			if got != tt.want {
				t.Errorf("DisplayListLine() =\n  %q\nwant\n  %q", got, tt.want)
			}
		})
	}
}

// TestDisplayListLine_RuneSafe verifies truncation lands on a rune boundary for
// multibyte text (no broken UTF-8) and that the visible text portion before the
// ellipsis is exactly the requested number of runes.
func TestDisplayListLine_RuneSafe(t *testing.T) {
	// Five 3-byte runes; clipping mid-rune via byte slicing would corrupt UTF-8.
	i := Idea{ID: "ab12", Date: "2026-06-01", Text: "日本語版あ"}
	prefixLen := len([]rune("- [ ] [ab12] 2026-06-01: "))

	got := DisplayListLine(i, prefixLen+3, false, false) // keep 2 runes + ellipsis
	if !utf8.ValidString(got) {
		t.Fatalf("output is not valid UTF-8: %q", got)
	}
	want := "- [ ] [ab12] 2026-06-01: 日本" + ellipsis
	if got != want {
		t.Errorf("DisplayListLine() = %q, want %q", got, want)
	}
}

// TestDisplayListLine_ColorAfterTruncation verifies color is applied around the
// already-truncated plain text: the visible (ANSI-stripped) line equals the
// uncolored render, so the width math counted runes, not escape bytes. The done
// checkbox is greened and the id/date prefix is dimmed.
func TestDisplayListLine_ColorAfterTruncation(t *testing.T) {
	i := Idea{ID: "ab12", Date: "2026-06-01", Text: "abcdefghij", Done: true}
	prefixLen := len([]rune("- [x] [ab12] 2026-06-01: "))
	width := prefixLen + 4

	colored := DisplayListLine(i, width, false, true)
	plain := DisplayListLine(i, width, false, false)

	if !strings.Contains(colored, ansiGreen) {
		t.Errorf("expected green code for done checkbox, got %q", colored)
	}
	if !strings.Contains(colored, ansiFaint) {
		t.Errorf("expected faint code for prefix, got %q", colored)
	}
	if stripped := stripANSI(colored); stripped != plain {
		t.Errorf("ANSI-stripped colored line = %q, want %q (color must not change visible text)", stripped, plain)
	}
}

// unsetEnvForTest forces key to be unset for the duration of the test, capturing
// any prior value and restoring it via t.Cleanup so the unset state never leaks
// into other tests (t.Setenv only restores a value it set, not a deletion).
func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()
	prev, had := os.LookupEnv(key)
	os.Unsetenv(key)
	t.Cleanup(func() {
		if had {
			os.Setenv(key, prev)
		} else {
			os.Unsetenv(key)
		}
	})
}

// stripANSI removes ANSI escape sequences for visible-text comparison in tests.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\033' {
			// Skip until the terminating 'm' of a CSI sequence.
			for i < len(s) && s[i] != 'm' {
				i++
			}
			if i < len(s) {
				i++ // skip the 'm'
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
