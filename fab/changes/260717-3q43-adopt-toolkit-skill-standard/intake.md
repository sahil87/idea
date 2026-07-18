# Intake: Adopt Toolkit Skill Standard (`idea skill`)

**Change**: 260717-3q43-adopt-toolkit-skill-standard
**Created**: 2026-07-18

## Origin

One-shot invocation: `/fab-new 3q43` (backlog ID). No prior conversation context. Backlog item verbatim:

> [3q43] 2026-07-18: Adopt the toolkit 'skill' standard for idea: add a hidden-or-visible 'idea skill' subcommand that prints a stable, static, <=150-line agent usage bundle to stdout (exit 0, stderr empty), byte-identical to a canonical docs/site/skill.md, wired via the sync+drift-guard pattern shll standards uses (committed embedded copy + sync script + drift-guard test). Content = usage briefing (when-to-use, capabilities map keyed to subcommands, composition with fab-kit, stdout/stderr + --json + exit-code contracts, gotchas) — NOT a README clone or flag table. docs/site/skill.md also renders at https://shll.ai/idea/skill for free. DEFERRED from 260717-9uh7-toolkit-standards-conformance per the standard's phased per-repo adoption (no tool ships skill yet; principle 10 is a SHOULD, so absence is not yet a violation). Ref: shll standards skill @ shll v0.0.23.

The full producer-facing standard was read at intake time via `shll standards skill` (the constitution's Toolkit Standards clause requires checking standards before changing the CLI surface or docs/site/). The reference implementation of the sync+drift-guard mechanism was read from the local `~/code/sahil87/shll` checkout (`scripts/sync-standards.sh`, `src/cmd/shll/standards.go`, `standards_test.go` → `TestStandardsEmbedMatchesCanonical`).

## Why

1. **The gap** (from the standard): an agent operating an *installed* `idea` binary has no offline usage briefing. `-h`/`help-dump` is flag reference (shape, not when-to-reach-for-which); README/`docs/site` requires a repo checkout or a network round-trip to shll.ai; `fab/project` context is contributor-scoped, not caller-scoped. A `<tool> skill` bundle is embedded (offline, present wherever the tool is), and version-locked by construction — the prose ships in the same binary as the flags it describes.
2. **If we don't**: agents keep operating `idea` from flag tables alone. The curated usage knowledge (targets model, exact-ID precedence, pipe contract, gotchas) stays scattered across memory/docs that an installed-tool caller never sees. Principle №10 is a SHOULD today, but the toolkit's forward design (`shll agent-setup` aggregating every installed tool's `skill` output) depends on tools adopting it.
3. **Why this approach**: the standard prescribes the exact mechanism — embedded static bundle, byte-identical to canonical `docs/site/skill.md`, sync script + committed copy + drift-guard test — and names shll's `standards` implementation as the pattern to reuse ("This is the exact mechanism `shll standards` established; reuse it"). No alternatives worth considering: the mechanism is mandated by the standard this change exists to adopt.
4. **Timing**: deferred from 260717-9uh7-toolkit-standards-conformance per phased per-repo adoption. No tool ships `skill` yet (verified: shll's own `docs/site/` has no `skill.md`) — `idea` would be the first adopter, so there is a standard and a mechanism precedent, but no `skill`-bundle content precedent to copy from.

## What Changes

### 1. New canonical bundle: `docs/site/skill.md`

The canonical agent usage briefing, ≤150 lines of raw markdown (hard budget per the standard / principle №9). Renders at `https://shll.ai/idea/skill` for free — `docs/site/**` is already part of the tree shll.ai pulls (alongside existing `install.md`, `workflows.md`).

**Genre rules** (from the standard): usage briefing, NOT a README clone, NOT a flag table. In scope:

- **When to use** — capturing ideas into a per-repo/per-worktree backlog without breaking flow; when it *isn't* the right reach (it is a backlog list, not a task runner or issue tracker).
- **Capabilities map** — one line per capability, keyed to the subcommand: `add` (+ bare-text shorthand `idea <text>`), `list`/`ls`, `show`, `done`, `reopen`, `edit`, `rm`, `prune`, `fmt`, `update`.
- **Composition with fab-kit** — the backlog file (`fab/backlog.md` by default) is shared vocabulary: fab's `/fab-new <4-char-id>` consumes idea's IDs; the `- [ ] [id] YYYY-MM-DD: text` line format is the stable cross-tool contract.
- **Output & exit-code contracts** — stdout-vs-stderr split (stdout is data; advisory notices go to stderr), `--json` availability (`list` and `show` only; schema `{id, date, status, text}` with `status: "open"|"done"`, Constitution VI), and the exit-code reality: **0 success, 1 for all errors (usage errors included); only `shell-init` exits 2**. The toolkit 0/1/2 convention is NOT yet implemented — that is deferred backlog item [xvsj] — so the bundle documents actual behavior, never the aspirational convention.
- **Gotchas** — e.g. the targets model (default = *current worktree's* backlog; `-m/--main` for the main worktree; `-s/--system` for `~/.config/idea/backlog.md`, also the out-of-git fallback); exact-ID precedence in query resolution (a 4-char exact ID wins over incidental substring matches); piped `list` output is canonical and untruncated (TTY-gated truncation/color); multiline idea text is stored escaped (`\n` convention).

The exact prose is authored at apply within these constraints; the bullet contents above are the facts it must draw from (verified against the code/memory at intake time).

### 2. New subcommand: `idea skill` (`src/cmd/idea/skill.go`)

- Cobra factory `skillCmd()` per Constitution III; registered in `newRootCmd()`'s `root.AddCommand(...)` list in `src/cmd/idea/main.go`.
- Command name exactly `skill` (standard: not `agent`, not `context`). **Visible** (not `Hidden: true`) — see Assumptions #2. `Args: cobra.NoArgs`. No flags of its own; no `--json` (stdout IS the raw markdown bytes — "no rendering, no pager, no added framing").
- `RunE` writes the embedded bundle bytes to `cmd.OutOrStdout()` verbatim. stderr empty on success, exit 0.
- Embeds the committed copy:

```go
//go:generate ../../../scripts/sync-skill.sh

//go:embed skill/skill.md
var skillFS embed.FS
```

The Go module root is `src/` and `docs/site/` sits above it, so `//go:embed` cannot reach the canonical file directly — the sync step copies it to `src/cmd/idea/skill/skill.md` first (same reachability constraint and same solution as shll).

### 3. Committed embedded copy: `src/cmd/idea/skill/skill.md`

Byte-identical copy of `docs/site/skill.md`, committed so a clean `go build ./...` (which does not run the sync script) compiles. Kept honest by the drift-guard test.

### 4. Sync script: `scripts/sync-skill.sh`

Mirror of shll's `scripts/sync-standards.sh`, single-file variant: `set -euo pipefail`, `cd "$(dirname "$0")/.."` (repo-root regardless of caller CWD), `cp -f docs/site/skill.md src/cmd/idea/skill/skill.md`, echo a confirmation line.

### 5. Drift-guard + contract tests: `src/cmd/idea/skill_test.go`

Table of guards, mirroring shll's `standards_test.go` shape (buffer-driven testable seam, no subprocess):

- **Drift guard**: embedded `skill/skill.md` bytes MUST equal `../../../docs/site/skill.md` (test file lives at `src/cmd/idea/`, canonical is three levels up). On mismatch, fail naming the drifted file and the fix (`run scripts/sync-skill.sh and commit the refreshed copy`). Runs on every `go test ./...` — CI's existing `ci.yml` PR workflow picks it up with no CI changes.
- **Line budget guard**: the bundle is ≤150 lines (the standard's hard budget, enforced rather than hoped for).
- **Command contract**: stdout equals the embedded bytes exactly; stderr empty; no error returned. (Static-only is guaranteed by construction — embedded bytes, no env lookups — the byte-equality test pins it.)

### 6. Interaction with existing surfaces (no changes needed, verified at intake)

- `help-dump`: a visible `skill` command appears as one more node in the help tree — the frozen envelope/node schema is unaffected. Its `Short`/`Long` flow to shll.ai's command reference automatically.
- No `internal/idea` changes: the command touches no backlog logic (Constitution IV — but note this command has no business logic at all; embedding + printing lives acceptably in `cmd/`, exactly like `help_dump.go`).
- No new dependencies: `embed` is stdlib (Dependency Discipline holds).

## Affected Memory

- `cli/skill`: (new) the `idea skill` subcommand contract (stdout/exit semantics, static-only rule, ≤150-line budget) and the sync+drift-guard mechanics (canonical `docs/site/skill.md` → committed copy `src/cmd/idea/skill/skill.md` via `scripts/sync-skill.sh`, drift-guard test)
- `cli/structure`: (modify) register the new subcommand in the source-tree/command-roster notes; note the first use of `//go:embed` in the repo
- `release/pipeline`: (modify) one-line addition to the shll.ai pull relationship: `docs/site/skill.md` now renders at `/idea/skill`

## Impact

- **New files**: `docs/site/skill.md`, `src/cmd/idea/skill.go`, `src/cmd/idea/skill_test.go`, `src/cmd/idea/skill/skill.md`, `scripts/sync-skill.sh`
- **Modified files**: `src/cmd/idea/main.go` (one line in `root.AddCommand`)
- **CI**: no workflow changes — the drift guard rides the existing `go test ./...` step
- **Public contract surface**: adds a new subcommand (checked against the `skill` standard at intake, per the constitution's Toolkit Standards clause); help-dump tree gains a node; shll.ai gains the `/idea/skill` page on its next pull
- **Out of scope**: exit-code convention adoption ([xvsj], separate change); mirroring the pattern into fab-kit (noted in backlog [e3rk] for `Long` descriptions — a different repo's concern)

## Open Questions

*(none — all decision points graded Certain/Confident; zero Unresolved)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Reuse shll's sync+drift-guard mechanism verbatim: committed embedded copy at `src/cmd/idea/skill/skill.md`, `scripts/sync-skill.sh`, byte-equality drift-guard test | Backlog item and the standard both name this exact mechanism ("reuse it"); reference implementation read from the local shll checkout | S:90 R:85 A:95 D:95 |
| 2 | Confident | Register `skill` as a **visible** command (not `Hidden: true` like `help-dump`) | Backlog explicitly leaves it open ("hidden-or-visible"); standard treats `skill` as a uniform first-class subcommand and agents discover it via `idea -h`; `help-dump` is hidden because it is machine plumbing for shll.ai, which `skill` is not; one-line flip if wrong | S:40 R:90 A:60 D:55 |
| 3 | Certain | Command contract: raw markdown to stdout, stderr empty on success, exit 0, `cobra.NoArgs`, no flags, no `--json` | Mandated verbatim by the standard's Invocation contract ("no rendering, no pager, no added framing") | S:95 R:80 A:95 D:95 |
| 4 | Confident | The bundle documents idea's **actual** exit-code behavior (0 success / 1 all errors; only `shell-init` exits 2) with a note that the toolkit 0/1/2 convention is pending [xvsj] | Documenting the unimplemented convention would make the bundle lie to agents branching on exit codes; the standard requires "the exit-code convention a caller branches on" — for idea today, that is uniform 1 | S:55 R:85 A:80 D:75 |
| 5 | Confident | Drift-guard test additionally pins the ≤150-line budget | The standard states a hard budget ("Bounded — ≤150 lines... rules with teeth"); shll's standards docs carry no budget so there is no test precedent, but enforcing a stated hard rule in the same test file is a cheap, reversible extension | S:60 R:90 A:75 D:70 |
| 6 | Confident | Bundle content outline: when-to-use, capabilities map keyed to the 10 subcommands, fab-kit composition via the shared backlog line format, contracts (`--json` on `list`/`show` only, schema `{id,date,status,text}`), gotchas (targets model, exact-ID precedence, pipe-canonical output, escaped multiline text) | Backlog enumerates the section list; the standard fixes the genre; the specific facts were verified against code/memory at intake — exact prose is authored at apply within the 150-line budget | S:70 R:75 A:70 D:65 |
| 7 | Certain | `//go:generate ../../../scripts/sync-skill.sh` directive lives in `skill.go`, mirroring shll's `standards.go` | Direct pattern reuse; pure convention with an existing reference implementation | S:75 R:95 A:90 D:90 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
