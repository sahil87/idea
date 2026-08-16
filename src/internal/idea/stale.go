package idea

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// This file is the staleness seam (Constitution IV): the duration parsing and
// the stale predicate behind `idea list --stale` and TTY age dimming live here
// in internal/idea; the cmd/ layer only wires the flag and passes time.Now().

const (
	// DefaultStaleDimDays is the staleness threshold used for TTY age dimming
	// when --stale is not passed: open ideas older than this many days render
	// faint on a color terminal. Fixed by design — no env var or config knob.
	DefaultStaleDimDays = 90

	// NoStaleDim is the sentinel stale-days value meaning "no age dimming".
	// It cannot be 0 because --stale 0 is a valid threshold ("older than
	// today"); prune passes this so its listing is never age-dimmed.
	NoStaleDim = -1
)

// ParseStaleDays parses a --stale duration value into a day count. Days only:
// a non-negative integer with an optional trailing "d" ("90d" and "90" both
// yield 90). Everything else — negative numbers, non-integers, empty strings,
// and any other unit ("90h", "3w") — is rejected with a descriptive error the
// cmd/ layer surfaces as a usage error (exit 2).
func ParseStaleDays(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("stale duration must be a non-negative number of days (e.g. \"90d\" or \"90\")")
	}
	digits := strings.TrimSuffix(s, "d")
	days, err := strconv.Atoi(digits)
	if err != nil || days < 0 {
		return 0, fmt.Errorf("invalid stale duration %q: must be a non-negative number of days (e.g. \"90d\" or \"90\")", s)
	}
	return days, nil
}

// IsStale reports whether idea i is stale: true iff it has a non-empty date
// strictly older than the cutoff today − days. An idea dated exactly
// today − days is NOT stale; a dateless idea (Date == "") is never stale —
// its age is uncomputable. The comparison is lexicographic on the stored
// YYYY-MM-DD strings, which is exact because validated ISO dates are
// zero-padded (lexicographic order equals chronological order) — no per-idea
// time.Parse on the filter/render hot path. today is a parameter (the cmd
// layer passes time.Now()) so tests inject a fixed clock.
func IsStale(i Idea, days int, today time.Time) bool {
	if i.Date == "" {
		return false
	}
	cutoff := today.AddDate(0, 0, -days).Format("2006-01-02")
	return i.Date < cutoff
}
