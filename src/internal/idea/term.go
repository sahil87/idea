package idea

import (
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// This file is the TTY/width/color/truncation seam (Constitution IV): all
// terminal-aware rendering logic lives here in internal/idea, and the cmd/
// layer only asks this seam for a decision (is this a TTY? how wide? use
// color?) and for a ready-to-print display line. FormatLine/DisplayLine stay
// the machine/canonical renderers and are intentionally untouched.

// defaultTermWidth is the column count assumed when the real width cannot be
// detected (GetSize fails and $COLUMNS is unset/invalid). 80 is the universal
// terminal default.
const defaultTermWidth = 80

// ANSI styling codes. Color is only ever emitted on a TTY with NO_COLOR unset
// (see UseColor); these are applied AFTER width-based truncation so the width
// math counts visible runes, never escape bytes.
const (
	ansiReset = "\033[0m"
	ansiFaint = "\033[2m"  // dim — used for the [id] date: prefix
	ansiGreen = "\033[32m" // used for a done [x] checkbox
)

// ellipsis is the single-rune marker appended to a clipped text portion.
const ellipsis = "…" // U+2026

// IsTTY reports whether f is connected to an interactive terminal. It is the
// single gate every display change (truncation, color, the prune prompt) keys
// on so that piped/redirected output stays full and canonical (Constitution VI).
func IsTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// TermWidth returns the terminal column count for f. Resolution order:
// term.GetSize first; if that fails or yields a non-positive width, honor
// $COLUMNS when it parses to a positive integer; otherwise fall back to 80.
func TermWidth(f *os.File) int {
	if f != nil {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 {
			return w
		}
	}
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n > 0 {
			return n
		}
	}
	return defaultTermWidth
}

// UseColor reports whether styled output should be emitted to f: true only when
// f is a TTY AND the NO_COLOR environment variable is unset. Per the NO_COLOR
// convention, presence disables color regardless of value (including an empty
// string), so the check is os.LookupEnv presence, not a truthiness test.
func UseColor(f *os.File) bool {
	if !IsTTY(f) {
		return false
	}
	_, noColor := os.LookupEnv("NO_COLOR")
	return !noColor
}

// dimPrefix wraps s in the ANSI faint code so the scannable [id] date: prefix
// recedes and the eye lands on the idea text.
func dimPrefix(s string) string {
	return ansiFaint + s + ansiReset
}

// greenCheck wraps s (a done "[x]" checkbox) in the ANSI green code.
func greenCheck(s string) string {
	return ansiGreen + s + ansiReset
}

// DisplayListLine renders an idea as a single physical list line for terminal
// display. It builds the canonical escaped line shape — the same prefix and
// escaped text that FormatLine produces — but, when full is false, truncates
// only the text portion to fit width (appending the ellipsis when clipped) and,
// when the escaped text contains a literal "\n" escape (a multiline idea),
// clips at the first newline regardless of width so the line is always one
// physical row. The "- [done] [id] date: " prefix is NEVER truncated.
//
// width is the terminal column count (a parameter so tests inject it rather
// than allocating a PTY). When color is true the prefix is dimmed and a done
// [x] checkbox is greened — applied AFTER truncation so the width math counts
// visible runes, not escape bytes. When stale is also true the whole line
// (text included) renders faint so aged ideas visually recede; a done [x]
// keeps its green — the explicit state signal outranks the age hint.
func DisplayListLine(i Idea, width int, full, color, stale bool) string {
	check := i.StatusCheck()
	// The prefix mirrors formatLineWith up to (and including) the "text"
	// position: "- [<check>] [<id>] <date>: ".
	prefix := "- [" + check + "] [" + i.ID + "] " + i.Date + ": "
	text := EscapeText(i.Text)

	if !full {
		text = truncateText(text, width-len([]rune(prefix)))
	}

	if !color {
		return prefix + text
	}

	styledCheck := "[" + check + "]"
	if i.Done {
		styledCheck = greenCheck(styledCheck)
	}
	// Dim the prefix EXCEPT the checkbox, which carries its own (green) color
	// when done. Rebuild the prefix in two dim spans around the checkbox so the
	// id/date stay faint while the [x] stays green.
	styledPrefix := dimPrefix("- ") + styledCheck + dimPrefix(" ["+i.ID+"] "+i.Date+": ")
	if stale {
		// Whole-line faint: the (already-truncated) text joins the dim spans.
		// A done [x] is untouched above, so it stays green — state outranks age.
		text = dimPrefix(text)
	}
	return styledPrefix + text
}

// truncateText returns text clipped to fit avail visible columns, appending the
// ellipsis when it had to clip. It is rune-safe (operates on []rune, never byte
// slices) so multibyte text is never cut mid-rune. A multiline idea — escaped
// text containing a literal "\n" — is always clipped at the first newline (with
// an ellipsis) so the rendered line is one physical row, regardless of width.
//
// When avail is non-positive (the prefix alone already fills or exceeds the
// terminal) the text is reduced to just the ellipsis, since the prefix is the
// scannable anchor and is never itself clipped.
func truncateText(text string, avail int) string {
	clipMultiline := false
	if idx := strings.Index(text, `\n`); idx >= 0 {
		text = text[:idx]
		clipMultiline = true
	}

	runes := []rune(text)
	if !clipMultiline && len(runes) <= avail {
		return text
	}

	// Need to clip: reserve one column for the ellipsis.
	if avail <= 1 {
		return ellipsis
	}
	keep := avail - 1
	if keep >= len(runes) {
		// The text fits but a multiline marker forced a clip; keep all of the
		// first-line runes and mark that more follows.
		return string(runes) + ellipsis
	}
	return string(runes[:keep]) + ellipsis
}
