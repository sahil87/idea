package idea

import (
	"testing"
	"time"
)

// TestParseStaleDays covers the --stale duration grammar: a non-negative
// integer with an optional trailing "d" parses; everything else (negative,
// non-integer, empty, other units) is an error.
func TestParseStaleDays(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "bare days", input: "90", want: 90},
		{name: "days with d suffix", input: "90d", want: 90},
		{name: "zero is a valid threshold", input: "0", want: 0},
		{name: "zero with d suffix", input: "0d", want: 0},
		{name: "one day", input: "1d", want: 1},
		{name: "empty is an error", input: "", wantErr: true},
		{name: "negative is an error", input: "-5", wantErr: true},
		{name: "negative with d suffix is an error", input: "-5d", wantErr: true},
		{name: "non-integer is an error", input: "abc", wantErr: true},
		{name: "hours unit is an error", input: "90h", wantErr: true},
		{name: "weeks unit is an error", input: "3w", wantErr: true},
		{name: "double suffix is an error", input: "90dd", wantErr: true},
		{name: "bare d is an error", input: "d", wantErr: true},
		{name: "decimal is an error", input: "1.5d", wantErr: true},
		{name: "whitespace is an error", input: " 90", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStaleDays(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseStaleDays(%q) = %d, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseStaleDays(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseStaleDays(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// TestIsStale pins the strictly-older-than semantics against a fixed clock:
// an idea dated exactly today − days is NOT stale (same-day boundary), older
// is stale, and a dateless idea is never stale. Dates are compared
// lexicographically — the cutoff is today.AddDate(0, 0, -days) formatted
// YYYY-MM-DD, and stored dates are validated zero-padded ISO.
func TestIsStale(t *testing.T) {
	// Fixed clock: today is 2026-08-17 (a Monday), so today − 90 = 2026-05-19.
	today := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		idea Idea
		days int
		want bool
	}{
		{name: "younger than cutoff is not stale", idea: Idea{ID: "ab12", Date: "2026-08-01"}, days: 90, want: false},
		{name: "exactly at cutoff is not stale (same-day boundary)", idea: Idea{ID: "ab12", Date: "2026-05-19"}, days: 90, want: false},
		{name: "one day older than cutoff is stale", idea: Idea{ID: "ab12", Date: "2026-05-18"}, days: 90, want: true},
		{name: "much older is stale", idea: Idea{ID: "ab12", Date: "2025-01-01"}, days: 90, want: true},
		{name: "dated today is not stale", idea: Idea{ID: "ab12", Date: "2026-08-17"}, days: 0, want: false},
		{name: "zero threshold: yesterday is stale", idea: Idea{ID: "ab12", Date: "2026-08-16"}, days: 0, want: true},
		{name: "dateless is never stale", idea: Idea{ID: "ab12", Date: ""}, days: 90, want: false},
		{name: "dateless is never stale at zero threshold", idea: Idea{ID: "ab12", Date: ""}, days: 0, want: false},
		{name: "done status does not affect the predicate", idea: Idea{ID: "ab12", Date: "2026-05-18", Done: true}, days: 90, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStale(tt.idea, tt.days, today); got != tt.want {
				t.Errorf("IsStale(%+v, %d, %v) = %v, want %v", tt.idea, tt.days, today, got, tt.want)
			}
		})
	}
}
