package main

import (
	"embed"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

//go:generate ../../../scripts/sync-skill.sh

// skillFS holds the canonical docs/site/skill.md, copied into this package dir by
// scripts/sync-skill.sh and embedded at build time. The Go module root is src/
// and docs/site/ sits above it, so //go:embed cannot reach the canonical file
// directly — the sync step copies it here first (see scripts/sync-skill.sh). The
// committed copy is what a clean `go build ./...` compiles; the drift-guard test
// in skill_test.go keeps it byte-honest against docs/site/skill.md on every
// `go test`.
//
//go:embed skill/skill.md
var skillFS embed.FS

// skillEmbedPath is the path of the embedded bundle within skillFS (matching the
// //go:embed pattern above). Named constant so the read path is single-sourced
// (no magic string — code-quality Anti-Patterns).
const skillEmbedPath = "skill/skill.md"

// skillCmd builds the visible "skill" subcommand — the agent-facing usage bundle
// for callers operating an installed idea binary. It prints the embedded
// docs/site/skill.md verbatim to stdout: raw markdown, no rendering, no framing,
// static across every invocation (the toolkit skill standard). It is visible (not
// hidden like help-dump) so agents discover it via `idea -h`; content is embedded
// at build time so it is offline and versioned with the release.
func skillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "skill",
		Short: "Print the agent usage bundle (offline, embedded)",
		Long: `Print the agent usage bundle for idea to stdout.

A one-page usage briefing for an agent operating an installed idea binary: when
to reach for the tool, its capabilities keyed to each subcommand, how it composes
with fab-kit, the output/exit-code contracts, and gotchas. It is NOT a flag
reference (see "idea <cmd> -h") nor a command tree (see "idea help-dump").

The bundle is raw markdown, printed verbatim with no rendering, no pager, and no
added framing — an agent consumes the bytes directly. It is embedded into the
binary at build time (byte-identical to the repo's docs/site/skill.md, which also
renders at https://shll.ai/idea/skill), so it is offline, static, and versioned
with the release.

  idea skill`,
		Args: usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkill(cmd.OutOrStdout())
		},
	}
}

// runSkill is the implementation seam for `idea skill`, extracted from the cobra
// factory so skill_test.go can drive it with a bytes.Buffer (no subprocess). It
// writes the embedded bundle bytes verbatim to out — the byte-equality tests pin
// that stdout equals the embedded document exactly.
func runSkill(out io.Writer) error {
	data, err := skillFS.ReadFile(skillEmbedPath)
	if err != nil {
		// A missing embed file is a build-integrity bug (the sync step / drift
		// guard should have caught it), not user error.
		return fmt.Errorf("idea skill: read embedded %s: %w", skillEmbedPath, err)
	}
	if _, err := out.Write(data); err != nil {
		return fmt.Errorf("idea skill: write: %w", err)
	}
	return nil
}
