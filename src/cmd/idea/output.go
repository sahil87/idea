package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sahil87/idea/internal/idea"
)

// printIdeaLines renders a slice of ideas one-per-line to out, TTY-aware. It is
// the single home of the list/prune rendering policy (Constitution IV's split
// keeps the truncation/color logic in internal/idea; this just picks the mode):
// when stdout is a terminal each line is truncated to the width and colored
// (unless full); when piped or redirected, full canonical FormatLine output is
// emitted regardless of full, preserving the machine-parseable pipe contract
// (Constitution VI).
//
// staleDays is the age-dimming threshold: on a color terminal, OPEN ideas
// stale per idea.IsStale (strictly older than today − staleDays) render
// whole-line faint. Done ideas are never age-dimmed — the dimming is an
// open-idea review signal (done ideas are prune's business), so a done idea
// listed via --done/--all keeps its normal rendering however old it is. Pass
// idea.NoStaleDim to disable dimming entirely (prune does — its listing is a
// consent surface, not a review surface). Piped output is never dimmed either
// way. today is passed explicitly so the staleness clock matches the caller's
// filter pass.
//
// The TTY/width/color decision keys on os.Stdout (the real destination) while
// out is the write target — out is normally os.Stdout in production and a
// buffer under test; both list and prune write their machine-readable lines to
// stdout.
func printIdeaLines(out io.Writer, ideas []idea.Idea, full bool, staleDays int, today time.Time) {
	if idea.IsTTY(os.Stdout) {
		width := idea.TermWidth(os.Stdout)
		color := idea.UseColor(os.Stdout)
		for _, i := range ideas {
			stale := staleDays != idea.NoStaleDim && !i.Done && idea.IsStale(i, staleDays, today)
			fmt.Fprintln(out, idea.DisplayListLine(i, width, full, color, stale))
		}
		return
	}
	for _, i := range ideas {
		fmt.Fprintln(out, idea.FormatLine(i))
	}
}
