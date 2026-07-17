package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// maxSkillLines is the hard line budget for the skill bundle (toolkit skill
// standard, principle №9). Enforced by TestSkill_LineBudget so a bloated bundle
// fails the build rather than silently growing.
const maxSkillLines = 150

// runSkill is driven directly with a bytes.Buffer (the testable seam extracted
// from the cobra factory, mirroring help_dump_test.go's in-process style). No
// subprocess is needed — `idea skill` reads embedded bytes only.

func TestSkill_Contract(t *testing.T) {
	// stdout MUST equal the embedded bundle bytes exactly; stderr is not written
	// by runSkill at all; RunE returns no error on success.
	want, err := skillFS.ReadFile(skillEmbedPath)
	if err != nil {
		t.Fatalf("read embedded %s: %v", skillEmbedPath, err)
	}

	var stdout bytes.Buffer
	if err := runSkill(&stdout); err != nil {
		t.Fatalf("runSkill err = %v, want nil", err)
	}
	if !bytes.Equal(stdout.Bytes(), want) {
		t.Errorf("stdout is not byte-identical to the embedded bundle (len got=%d want=%d)",
			stdout.Len(), len(want))
	}
	if len(want) == 0 {
		t.Error("embedded skill bundle is empty; expected the usage briefing bytes")
	}
}

func TestSkill_CommandStderrEmptyExitZero(t *testing.T) {
	// End-to-end through the cobra tree via newRootCmd(): `idea skill` writes the
	// bundle to stdout, leaves stderr empty, and returns no error (exit 0).
	want, err := skillFS.ReadFile(skillEmbedPath)
	if err != nil {
		t.Fatalf("read embedded %s: %v", skillEmbedPath, err)
	}

	root := newRootCmd()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"skill"})

	if err := root.Execute(); err != nil {
		t.Fatalf("`idea skill` execute failed: %v\nstderr:\n%s", err, stderr.String())
	}
	if !bytes.Equal(stdout.Bytes(), want) {
		t.Errorf("`idea skill` stdout is not byte-identical to the embedded bundle (len got=%d want=%d)",
			stdout.Len(), len(want))
	}
	if stderr.Len() != 0 {
		t.Errorf("`idea skill` wrote to stderr: %q", stderr.String())
	}
}

func TestSkill_RejectsArgs(t *testing.T) {
	// The command takes cobra.NoArgs — a positional argument is a usage error.
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"skill", "extra"})

	if err := root.Execute(); err == nil {
		t.Error("`idea skill extra` should error under cobra.NoArgs, got nil")
	}
}

func TestSkill_LineBudget(t *testing.T) {
	// The bundle MUST be <= maxSkillLines lines (the standard's hard budget).
	data, err := skillFS.ReadFile(skillEmbedPath)
	if err != nil {
		t.Fatalf("read embedded %s: %v", skillEmbedPath, err)
	}
	n := bytes.Count(data, []byte{'\n'})
	// A file ending in a trailing newline has one fewer content line than \n
	// count only when the last line is empty; count \n directly as the line count
	// for a newline-terminated file (each content line contributes one \n).
	if n > maxSkillLines {
		t.Errorf("skill bundle is %d lines, over the %d-line budget (toolkit skill standard) — trim docs/site/skill.md", n, maxSkillLines)
	}
}

// TestSkillEmbedMatchesCanonical is the drift guard: the embedded skill/skill.md
// bytes MUST equal the canonical docs/site/skill.md. This test file lives at
// src/cmd/idea/, so the canonical source is three levels up under docs/site/.
// Runs on every `go test ./...` (and in the existing CI PR workflow) — when the
// canonical doc drifts from the committed copy (someone edits docs/site/skill.md
// without re-running scripts/sync-skill.sh), this fails, naming the drifted file.
func TestSkillEmbedMatchesCanonical(t *testing.T) {
	embedded, err := skillFS.ReadFile(skillEmbedPath)
	if err != nil {
		t.Fatalf("read embedded %s: %v", skillEmbedPath, err)
	}
	canonicalPath := filepath.Join("..", "..", "..", "docs", "site", "skill.md")
	canonical, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("read canonical %s: %v", canonicalPath, err)
	}
	if !bytes.Equal(embedded, canonical) {
		t.Errorf("embedded %s has drifted from canonical docs/site/skill.md — run scripts/sync-skill.sh and commit the refreshed copy",
			skillEmbedPath)
	}
}
