package main

import (
	"fmt"
	"io"
	"os"

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
// The TTY/width/color decision keys on os.Stdout (the real destination) while
// out is the write target — out is normally os.Stdout in production and a
// buffer under test; both list and prune write their machine-readable lines to
// stdout.
func printIdeaLines(out io.Writer, ideas []idea.Idea, full bool) {
	if idea.IsTTY(os.Stdout) {
		width := idea.TermWidth(os.Stdout)
		color := idea.UseColor(os.Stdout)
		for _, i := range ideas {
			fmt.Fprintln(out, idea.DisplayListLine(i, width, full, color))
		}
		return
	}
	for _, i := range ideas {
		fmt.Fprintln(out, idea.FormatLine(i))
	}
}
