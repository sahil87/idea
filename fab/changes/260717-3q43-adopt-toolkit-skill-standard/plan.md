# Plan: Adopt Toolkit Skill Standard (`idea skill`)

**Change**: 260717-3q43-adopt-toolkit-skill-standard
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake § What Changes. RFC 2119 keywords. One R# per change area. -->

### Skill Bundle: Canonical Content (`docs/site/skill.md`)

#### R1: Canonical bundle exists and is a bounded usage briefing
The repo MUST carry a canonical agent usage briefing at `docs/site/skill.md`. It MUST be raw Markdown, ≤150 lines (the standard's hard budget, principle №9), in the usage-briefing genre — NOT a README clone and NOT a flag table. It MUST cover: when-to-use (and when not), a capabilities map keyed to each user-facing subcommand, composition with fab-kit via the shared backlog line format, the output/exit-code contracts documenting idea's **actual** behavior, and gotchas. It MUST be static — no timestamps, environment lookups, or session state.

- **GIVEN** an agent operating an installed `idea` binary with no repo checkout
- **WHEN** it reads the bundle
- **THEN** it learns when to reach for `idea`, what each subcommand does, how `idea` composes with fab-kit, the stdout/stderr/`--json`/exit-code contracts, and the non-obvious gotchas
- **AND** the file is ≤150 lines and contains no dynamic content

#### R2: Exit-code and channel contracts documented match actual behavior
The bundle MUST document idea's actual exit-code reality: **0 on success, 1 for all errors (usage errors included); only `shell-init` exits 2**. It MUST state that the toolkit 0/1/2 convention is not yet implemented (deferred backlog item `[xvsj]`) rather than documenting the aspirational convention. It MUST document the stdout-vs-stderr split (stdout is data; advisory notices go to stderr) and that `--json` exists only on `list` and `show`, with schema `{id, date, status, text}` and `status: "open"|"done"` (Constitution VI).

- **GIVEN** an agent that branches on `idea`'s exit codes
- **WHEN** it reads the exit-code contract in the bundle
- **THEN** it sees `0`/`1` (and `shell-init`'s `2`) as the real behavior, with a note that the 0/1/2 convention is pending `[xvsj]`
- **AND** it never sees a claim that usage errors return `2` today

### Skill Subcommand (`src/cmd/idea/skill.go`)

#### R3: `idea skill` command prints the embedded bundle to stdout
There MUST be a cobra factory `skillCmd()` (Constitution III) registered in `newRootCmd()`. The command name MUST be exactly `skill`. It MUST be **visible** (not `Hidden: true`). It MUST take `cobra.NoArgs`, define no flags of its own, and have no `--json`. Its `RunE` MUST write the embedded bundle bytes verbatim to `cmd.OutOrStdout()`, leaving stderr empty and exiting 0 on success.

- **GIVEN** an installed `idea` binary
- **WHEN** `idea skill` runs
- **THEN** stdout is the embedded bundle bytes verbatim, stderr is empty, exit code is 0
- **AND** `idea skill anything` errors (NoArgs) and the command appears in `idea -h`

#### R4: Bundle is embedded via `//go:embed` with a `//go:generate` sync directive
`skill.go` MUST embed the committed copy `src/cmd/idea/skill/skill.md` via `//go:embed skill/skill.md` into an `embed.FS` (the module root is `src/`, so `//go:embed` cannot reach `docs/site/skill.md` directly). It MUST carry a `//go:generate ../../../scripts/sync-skill.sh` directive, mirroring shll's `standards.go`.

- **GIVEN** a clean checkout (where `go build ./...` does not run the sync script)
- **WHEN** the package is built
- **THEN** the committed `skill/skill.md` copy compiles into the binary via the embed directive
- **AND** `go generate ./...` runs the sync script through the directive

### Committed Embedded Copy (`src/cmd/idea/skill/skill.md`)

#### R5: A committed byte-identical copy of the canonical bundle exists in the embed dir
`src/cmd/idea/skill/skill.md` MUST exist and be byte-identical to `docs/site/skill.md`, committed so a clean `go build ./...` compiles without running the sync script.

- **GIVEN** the canonical `docs/site/skill.md`
- **WHEN** the committed copy is compared to it
- **THEN** the two files are byte-for-byte identical

### Sync Script (`scripts/sync-skill.sh`)

#### R6: Sync script copies the canonical bundle into the embed dir, repo-root-relative
`scripts/sync-skill.sh` MUST exist, be executable, and mirror shll's `scripts/sync-standards.sh` as a single-file variant: `set -euo pipefail`, `cd "$(dirname "$0")/.."` (so it runs from the repo root regardless of caller CWD), `cp -f docs/site/skill.md src/cmd/idea/skill/skill.md`, and echo a confirmation line.

- **GIVEN** an edited `docs/site/skill.md`
- **WHEN** `scripts/sync-skill.sh` runs from any working directory
- **THEN** `src/cmd/idea/skill/skill.md` is overwritten to match, and a confirmation line is printed

### Drift-Guard & Contract Tests (`src/cmd/idea/skill_test.go`)

#### R7: Drift-guard test fails when the embedded copy diverges from canonical
`src/cmd/idea/skill_test.go` MUST assert the embedded `skill/skill.md` bytes equal `../../../docs/site/skill.md` (the canonical file, three levels up from the test's package dir). On mismatch it MUST fail naming the drifted file and the fix (`run scripts/sync-skill.sh and commit the refreshed copy`). It runs on every `go test ./...`, so the existing CI PR workflow picks it up with no CI changes.

- **GIVEN** an edit to `docs/site/skill.md` without re-running the sync script
- **WHEN** `go test ./...` runs
- **THEN** the drift-guard test fails, naming the drifted file and the remedial command

#### R8: Line-budget guard test enforces the ≤150-line bundle budget
`skill_test.go` MUST assert the bundle is ≤150 lines (the standard's hard budget, enforced rather than hoped for).

- **GIVEN** a bundle that grows past 150 lines
- **WHEN** `go test ./...` runs
- **THEN** the budget guard test fails, reporting the actual line count against the 150 limit

#### R9: Command-contract test pins stdout/stderr/exit behavior
`skill_test.go` MUST drive the command through a testable in-process seam (buffer-driven, no subprocess) and assert: stdout equals the embedded bytes exactly; stderr is empty; no error is returned.

- **GIVEN** the `skill` command driven with capture buffers
- **WHEN** it runs with no args
- **THEN** stdout equals the embedded bundle bytes, stderr is empty, and RunE returns nil

### Registration (`src/cmd/idea/main.go`)

#### R10: `skillCmd()` is registered in the root command
`newRootCmd()`'s `root.AddCommand(...)` list MUST include `skillCmd()`, so `idea skill` is dispatchable and appears as a node in the help tree. Because it is visible, its `Short`/`Long` flow to the help-dump JSON automatically (no help-dump schema change).

- **GIVEN** the built binary
- **WHEN** `idea help-dump` runs
- **THEN** a `skill` node appears in the command tree with the correct path `idea skill`

### Non-Goals

- Adopting the toolkit 0/1/2 usage-error exit-code convention — deferred backlog item `[xvsj]`, a separate change.
- Mirroring the sync+drift-guard pattern into fab-kit — a different repo's concern (backlog `[e3rk]`).
- Adding `--json` or any flag to `idea skill` — the standard mandates raw markdown to stdout with no framing.
- Any `internal/idea` change — the command touches no backlog logic.

### Design Decisions

1. **Visible `skill` command, not `Hidden: true`**: register `skill` as a first-class user-facing subcommand — *Why*: the standard treats `skill` as a uniform first-class subcommand agents discover via `idea -h`; `help-dump` is hidden only because it is machine plumbing for shll.ai, which `skill` is not — *Rejected*: hidden like `help-dump` (a one-line flip if wrong; Assumption #2).
2. **Reuse shll's sync+drift-guard mechanism verbatim**: committed embed copy + `sync-skill.sh` + byte-equality drift-guard test — *Why*: the standard names this exact mechanism ("reuse it") and shll's `standards.go`/`standards_test.go`/`sync-standards.sh` are the reference — *Rejected*: embedding `docs/site/skill.md` directly (impossible — it sits above the `src/` module root, the same reachability constraint shll solved this way).
3. **Buffer-driven testable seam** (`runSkill(stdout io.Writer) error`): extract the write logic from the cobra factory so tests drive it with `bytes.Buffer` and no subprocess — *Why*: mirrors shll's `runStandards` seam and idea's own in-process test style (`help_dump_test.go` runs via `newRootCmd()`); the command reads embedded bytes only, so no fake proc runner is needed — *Rejected*: subprocess test (unnecessary; the bytes are static).
4. **Document actual exit-code behavior, not the aspirational convention**: the bundle states 0 success / 1 all errors, only `shell-init` exits 2, with a pending-`[xvsj]` note — *Why*: documenting the unimplemented 0/1/2 convention would make the bundle lie to agents branching on exit codes (Assumption #4).

## Tasks

### Phase 1: Setup

- [x] T001 Create `docs/site/skill.md` — the canonical ≤150-line usage-briefing bundle: when-to-use, capabilities map keyed to the 10 user-facing subcommands (add + bare-text shorthand, list/ls, show, done, reopen, edit, rm, prune, fmt, update), fab-kit composition via the shared `- [ ] [id] YYYY-MM-DD: text` backlog line format, output/exit contracts (stdout=data / stderr=advisory; `--json` on list/show only; 0 success / 1 all errors / only `shell-init` exits 2 with the pending-`[xvsj]` note), and gotchas (targets model default=current worktree with `-m`/`-s`; exact-ID precedence; pipe-canonical untruncated output; escaped multiline `\n`) <!-- R1 R2 -->

### Phase 2: Core Implementation

- [x] T002 Create `scripts/sync-skill.sh` (executable, `chmod +x`) — single-file mirror of shll's `sync-standards.sh`: `set -euo pipefail`, `cd "$(dirname "$0")/.."`, `cp -f docs/site/skill.md src/cmd/idea/skill/skill.md`, echo a confirmation line <!-- R6 -->
- [x] T003 Run `scripts/sync-skill.sh` to produce the committed byte-identical copy `src/cmd/idea/skill/skill.md` <!-- R5 -->
- [x] T004 Create `src/cmd/idea/skill.go` — cobra factory `skillCmd()` (visible, `Use: "skill"`, `cobra.NoArgs`, no flags, enriched `Long`), `//go:generate ../../../scripts/sync-skill.sh` and `//go:embed skill/skill.md` directives with an `embed.FS` var, and a buffer-driven `runSkill(out io.Writer) error` seam that writes the embedded bytes verbatim <!-- R3 R4 -->
- [x] T005 Register `skillCmd()` in `newRootCmd()`'s `root.AddCommand(...)` list in `src/cmd/idea/main.go` <!-- R10 -->

### Phase 3: Integration & Edge Cases

- [x] T006 Create `src/cmd/idea/skill_test.go` (table/seam-driven, no subprocess) with: the drift guard (`embedded == ../../../docs/site/skill.md`, failing with the fix hint), the ≤150-line budget guard, and the command contract (stdout == embedded bytes, stderr empty, RunE nil) <!-- R7 R8 R9 -->

### Phase 4: Polish

- [x] T007 Verify the whole suite: `cd src && go test ./...`, `gofmt -l .` (must be empty), `go vet ./...` (clean); confirm the built `idea skill` prints the bundle and `idea help-dump` shows the `skill` node <!-- R1 R2 R3 R4 R5 R6 R7 R8 R9 R10 -->

## Execution Order

- T001 blocks T002/T003 (sync copies the canonical file; test compares against it).
- T003 (produces `skill/skill.md`) blocks T004 (embed target must exist to compile) and T006 (embed + canonical both read).
- T004 blocks T005 (registration references `skillCmd()`).
- T007 runs last (validates the whole change).

## Acceptance

### Functional Completeness

- [x] A-001 R1: `docs/site/skill.md` exists, is ≤150 lines of raw Markdown, in the usage-briefing genre, and covers when-to-use, the capabilities map keyed to subcommands, fab-kit composition, output/exit contracts, and gotchas
- [x] A-002 R2: the bundle documents actual exit behavior (0 success / 1 all errors; only `shell-init` exits 2) with the pending-`[xvsj]` note, the stdout/stderr split, and `--json` on `list`/`show` only with schema `{id, date, status, text}`
- [x] A-003 R3: `idea skill` is a visible cobra command (`skillCmd()`), `cobra.NoArgs`, no flags, writing the embedded bundle verbatim to stdout with empty stderr and exit 0
- [x] A-004 R4: `skill.go` embeds `skill/skill.md` via `//go:embed` into an `embed.FS` and carries the `//go:generate ../../../scripts/sync-skill.sh` directive
- [x] A-005 R5: `src/cmd/idea/skill/skill.md` exists and is byte-identical to `docs/site/skill.md`
- [x] A-006 R6: `scripts/sync-skill.sh` exists, is executable, runs repo-root-relative, copies canonical → embed, and echoes a confirmation
- [x] A-007 R7: `skill_test.go`'s drift guard fails on divergence, naming the drifted file and the remedial command
- [x] A-008 R8: `skill_test.go`'s budget guard enforces ≤150 lines
- [x] A-009 R9: `skill_test.go`'s command-contract test asserts stdout==embedded, stderr empty, no error
- [x] A-010 R10: `skillCmd()` is registered in `newRootCmd()` and `idea help-dump` shows a `skill` node at path `idea skill`

### Behavioral Correctness

- [x] A-011 R3: `idea skill extra-arg` errors under `cobra.NoArgs`; `idea skill` with no args succeeds
- [x] A-012 R5: after editing `docs/site/skill.md`, running `scripts/sync-skill.sh` restores byte-identity and the drift guard passes again

### Scenario Coverage

- [x] A-013 R9: a buffer-driven test exercises the `runSkill` seam in-process (no subprocess), matching idea's existing in-process test style

### Code Quality

- [x] A-014 Pattern consistency: `skill.go` follows the cobra-factory + enriched-`Long` + `cmd.OutOrStdout()` conventions of surrounding `cmd/idea/*.go` files (e.g. `help_dump.go`, `show.go`); `skill_test.go` mirrors idea's in-process/table-driven test style (Constitution V)
- [x] A-015 No unnecessary duplication: the write logic lives in a single `runSkill` seam reused by both the cobra `RunE` and the tests; no second serialization or read path; the embed path uses a named constant rather than a repeated string literal (code-quality Anti-Patterns: no magic strings)
- [x] A-016 No god functions: `skillCmd()` and `runSkill` stay small (well under the 50-line anti-pattern threshold)
- [x] A-017 Dependency discipline: no new dependencies — `embed` is stdlib (Constitution Dependency Discipline)
- [x] A-018 gofmt/vet clean: `gofmt -l .` reports nothing and `go vet ./...` passes (CI enforces both)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant.

## Assumptions

<!-- Graded SRAD decisions made while co-generating ## Requirements. The intake
     already recorded the substantive design decisions (7 rows); these are the
     apply-time restatements plus the implementation-seam decisions. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Reuse shll's sync+drift-guard mechanism verbatim (committed embed copy at `src/cmd/idea/skill/skill.md`, `scripts/sync-skill.sh`, byte-equality drift-guard test) | Standard and backlog both name this exact mechanism ("reuse it"); shll reference read at intake (intake Assumption #1) | S:90 R:85 A:95 D:95 |
| 2 | Confident | Register `skill` as a **visible** command (not `Hidden: true`) | Standard treats `skill` as a first-class subcommand agents discover via `idea -h`; `help-dump` is hidden only as shll.ai plumbing; one-line flip if wrong (intake Assumption #2) | S:40 R:90 A:60 D:55 |
| 3 | Certain | Command contract: raw markdown to stdout, stderr empty on success, exit 0, `cobra.NoArgs`, no flags, no `--json` | Mandated verbatim by the standard's Invocation contract (intake Assumption #3) | S:95 R:80 A:95 D:95 |
| 4 | Confident | Bundle documents idea's actual exit-code behavior (0/1; only `shell-init` exits 2) with a pending-`[xvsj]` note | Documenting the unimplemented 0/1/2 convention would make the bundle lie to agents (intake Assumption #4) | S:55 R:85 A:80 D:75 |
| 5 | Confident | Drift-guard test additionally pins the ≤150-line budget | The standard states a hard ≤150-line budget; enforcing it in the same test file is a cheap, reversible extension (intake Assumption #5) | S:60 R:90 A:75 D:70 |
| 6 | Confident | Bundle content outline: when-to-use, capabilities map keyed to the 10 subcommands, fab-kit composition via the shared backlog line format, contracts, gotchas | Backlog enumerates the section list; standard fixes the genre; facts verified against code/memory at intake (intake Assumption #6) | S:70 R:75 A:70 D:65 |
| 7 | Certain | `//go:generate ../../../scripts/sync-skill.sh` directive in `skill.go`, mirroring shll's `standards.go` | Direct pattern reuse with an existing reference implementation (intake Assumption #7) | S:75 R:95 A:90 D:90 |
| 8 | Confident | Extract a buffer-driven `runSkill(out io.Writer) error` seam from the cobra factory for in-process testing | Mirrors shll's `runStandards` seam and idea's in-process test style (`help_dump_test.go` via `newRootCmd()`); no subprocess needed since the command reads embedded bytes only | S:75 R:90 A:85 D:80 |

8 assumptions (3 certain, 5 confident, 0 tentative).
