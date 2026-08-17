# Plan: Promote Idea to Main Backlog

**Change**: 260816-kmti-promote-idea-main-backlog
**Intake**: `intake.md`

## Requirements

### CLI: `idea promote` command surface

#### R1: Visible `promote` subcommand
A new visible subcommand `idea promote <query>` SHALL be registered in `newRootCmd()`'s `AddCommand` roster (`src/cmd/idea/main.go`), implemented as a `promoteCmd()` factory in a new `src/cmd/idea/promote.go` (Constitution III). Its `Args` validator MUST be `usageArgs(cobra.ExactArgs(1))` — an unwrapped validator would regress arg-count errors to exit 1. It SHALL carry an enriched cobra `Long` (what it does, source→dest semantics, collision behavior, an example) with `Short` staying a terse one-liner, per the repo-wide help convention. `help-dump` picks the node up automatically; no schema change.

- **GIVEN** a repo checkout with the new binary
- **WHEN** `idea promote` runs with zero or two args
- **THEN** cobra rejects with a usage error and the process exits 2
- **AND** `idea promote <query>` routes to the subcommand, never to the bare-text add shorthand (namespace claim, same trade as `prune`/`fmt`)

#### R2: Source/destination resolution and flag surface
Promote SHALL resolve source = the **current worktree's** backlog and destination = the **main worktree's** backlog, both via the existing `ResolveBacklogPath` precedence: source with `(systemFlag=false, mainFlag=false, fileFlag)`, destination with `(systemFlag=false, mainFlag=true, fileFlag)` — so `--file`/`IDEAS_FILE` is honored and applied **within each root** (same relative path under both roots; absolute values used verbatim, which collapses source and destination to the same file and takes the R6 no-op). Passing `--main` or `--system` with promote MUST be rejected as a usage error (exit 2, wrapped in `usageError` in the cmd layer per the Constitution IV policy split) — promote defines its own source/dest. Outside a git repository, promote MUST fail operationally (exit 1, `not in a git repository`) — destination resolution is git-only; the system fallback does not apply (a `--to-system` variant is deferred).

- **GIVEN** a linked worktree of a repo
- **WHEN** `idea promote <query>` runs with no root-selector flags
- **THEN** source resolves to `{worktree-root}/fab/backlog.md` and destination to `{main-root}/fab/backlog.md`
- **GIVEN** any invocation with `--main` or `--system`
- **WHEN** promote runs
- **THEN** it exits 2 with a clear conflict message and touches no file
- **GIVEN** a cwd outside any git repository
- **WHEN** promote runs
- **THEN** it exits 1 with the git-resolution error and touches no file

#### R3: Query resolution via the shared matcher
Promote SHALL resolve `<query>` against the **source** backlog using the existing `RequireSingle(query, ideas, FilterAll)` — case-insensitive substring on ID or text, exact-ID precedence, ambiguity refusal listing matches. `FilterAll` because done ideas are promotable (status is preserved). No-match and ambiguous-match are operational errors (exit 1) with the existing error wording; no file is modified.

- **GIVEN** a source backlog with idea `[a7k2]` and another idea whose text contains "a7k2"
- **WHEN** `idea promote a7k2` runs
- **THEN** the exact-ID owner is selected (precedence), not the ambiguity error

#### R4: Move semantics — preserve, destination first, one canonical write per file
The move SHALL preserve the idea's `id`, `date`, and `status` verbatim (a done idea arrives done; a dateless idea gets today's date stamped by the normal save-seam backfill, counted per file). Write ordering MUST be **destination first, then source**: append the idea to the destination `File` and save it; only after that write succeeds, remove the idea from the source (`removeIdeaAt`) and save it — a crash between the writes duplicates the idea, never loses it. Each file SHALL get exactly **one** canonical write through the existing `LoadFile`/`SaveFile`(`render`) seam — normalize-on-write applies to both files as usual. A missing destination file or parent directory is NOT an error: it is treated as an empty backlog and created on write (the `atomicWriteFile` MkdirAll seam).

- **GIVEN** a done, dateless idea `[a7k2]` in the source backlog and no destination file
- **WHEN** `idea promote a7k2` runs
- **THEN** the destination file is created containing the idea with the same ID, `[x]` state, and today's date backfilled
- **AND** the idea is removed from the source backlog
- **AND** both files are in canonical form

#### R5: Destination ID collision refusal
Before writing, promote MUST check the destination's parsed ideas for the same ID. On collision it SHALL refuse with a clear operational error (exit 1) naming the ID — following the existing what/why/next error style — and MUST NOT re-mint the ID or modify either file. (Parsed-ideas-only scope: the same accepted blind spot as `checkIDCollision` — a 4-char bracket inside an unparseable line is invisible.)

- **GIVEN** idea `[a7k2]` present in both source and destination backlogs
- **WHEN** `idea promote a7k2` runs
- **THEN** it exits 1 with an error naming `[a7k2]` and both files are byte-identical to before

#### R6: Already-in-main no-op
When the resolved source and destination paths are the same file (main worktree, or an absolute `--file` override), promote SHALL be a no-op: nothing written, a `note:`-prefixed advisory on **stderr** (established advisory channel), exit 0 — matching the `idea edit` unchanged-buffer precedent. Detection compares the two resolved paths in the cmd layer.

- **GIVEN** a cwd in the main worktree
- **WHEN** `idea promote <query>` runs
- **THEN** exit code is 0, stdout is empty, stderr carries a one-line `note:` advisory, and the backlog is untouched

#### R7: Output contract
On success, stdout SHALL carry exactly the machine-parseable confirmation `Promoted: {FormatLine(idea)}` (escaped single line, matching the `Done:`/`Removed:` shape — Constitution VI). Date-backfill counts from **both** saves SHALL surface via the existing stderr notice path (`printBackfillNotice`), and `internal/idea` writes nothing to stderr (Constitution IV split).

- **GIVEN** a successful promote that backfilled dates in the source file
- **WHEN** output is inspected
- **THEN** stdout is exactly the `Promoted:` line and the backfill `note:` appears on stderr

### Docs: public-surface documentation

#### R8: Spec, skill bundle, and README updated
The change SHALL add the `promote` row/semantics to `docs/specs/overview.md` (commands table + a short semantics note), to the agent usage bundle `docs/site/skill.md` (synced to `src/cmd/idea/skill/skill.md` via `scripts/sync-skill.sh`, keeping the drift guard and 150-line budget guard green), and to the `README.md` command list.

- **GIVEN** the completed change
- **WHEN** `cd src && go test ./...` runs
- **THEN** the skill drift guard and budget guard pass with promote documented

### Non-Goals

- `--to-system` promote variant (repo→system) — deliberately deferred; rejecting `--system` now keeps the surface open.
- Auto-promotion on worktree cleanup — `idea` has no hook into `wt`/git lifecycle.
- Bulk promote (multiple queries / `--all`) — one idea per invocation.
- Consent flag / `--dry-run` — promote is a move, not a destructive delete; the datum exists in at least one file at all times.

### Design Decisions

#### Destination-first write ordering
**Decision**: Save the destination backlog before removing from the source; the two saves are not atomic as a pair.
**Why**: A crash between the writes leaves the idea duplicated (visible, trivially fixed with `rm`) instead of lost — the failure mode the backlog entry explicitly chose.
**Rejected**: Source-first (a crash silently loses the idea); a two-file transactional scheme (needless complexity for a plain-text tool — Constitution I keeps files independently hand-editable).
*Introduced by*: 260816-kmti-promote-idea-main-backlog

#### Whole-file canonical write for the destination append
**Decision**: Append into the destination via `LoadFile` + append to `File.ideas` + `SaveFile`, not via `Add`'s raw byte-append path.
**Why**: The backlog contract asks for one canonical write per file; the `SaveFile`/`render` seam gives status preservation, date backfill counting, and dir creation for free.
**Rejected**: Extending `Add` with a status parameter (churns a stable exported signature for one caller); raw append (skips backfill counting and canonicalization).
*Introduced by*: 260816-kmti-promote-idea-main-backlog

#### No-op detection in the cmd layer
**Decision**: `cmd/idea/promote.go` compares the two resolved backlog paths and short-circuits to the advisory no-op before calling `internal/idea`.
**Why**: Path resolution already lives at the cmd seam (`resolveFile` pattern); the advisory is stderr channel policy, which is a `cmd/` concern (Constitution IV). `Promote` stays a pure two-distinct-paths operation.
**Rejected**: Detecting inside `Promote` (would put output-channel policy or a sentinel error in `internal/idea` for no gain).
*Introduced by*: 260816-kmti-promote-idea-main-backlog

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add `Promote(srcPath, dstPath, query string) (Idea, srcBackfilled int, dstBackfilled int, err error)` in new `src/internal/idea/promote.go`: load source, resolve via `RequireSingle(query, f.ideas, FilterAll)`, check destination for ID collision (parsed ideas; reuse/mirror `checkIDCollision`), append the idea to the destination `File` and `SaveFile` it (dest-first), then `removeIdeaAt` + `SaveFile` the source; missing destination file loads as empty. No stderr output. <!-- R3 R4 R5 -->
- [x] T002 Add table-driven tests in new `src/internal/idea/promote_test.go` (real temp dirs, Constitution V): open + done idea moved with id/date/status preserved; dateless idea backfilled with per-file counts returned; missing destination file/dir created; collision refusal leaves both files byte-identical; no-match and ambiguous-match errors; exact-ID precedence via the shared resolver; source untouched when the destination save fails (e.g. destination path is a directory). <!-- R4 -->
- [x] T003 Add `promoteCmd()` in new `src/cmd/idea/promote.go`: `Use: "promote <query>"`, `Args: usageArgs(cobra.ExactArgs(1))`, enriched `Long` + terse `Short`; `RunE` rejects `mainFlag`/`systemFlag` with `&usageError{...}`, resolves source `ResolveBacklogPath(false, false, fileFlag)` and destination `ResolveBacklogPath(false, true, fileFlag)`, short-circuits the same-path no-op with a stderr `note:` advisory (exit 0), calls `idea.Promote`, prints backfill notices for both files via `printBackfillNotice`, and prints `Promoted: {FormatLine}` to stdout. <!-- R1 R2 R6 R7 -->
- [x] T004 Register `promoteCmd()` in `newRootCmd()`'s `AddCommand` roster in `src/cmd/idea/main.go` (alongside the other verb factories). <!-- R1 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Add subprocess CLI tests in `src/cmd/idea/main_test.go` (reusing `buildBinary`/`setupGitRepo`/`runSplit` helpers; add a linked-worktree helper via `git worktree add` if none exists): promote from a linked worktree lands the idea in the main worktree's backlog and removes it from the source; already-in-main no-op (exit 0, stderr note, file untouched); `promote --main` and `promote --system` exit 2; outside-git exit 1; destination collision exits 1; done idea arrives `[x]`; stdout carries exactly the `Promoted:` line. <!-- R2 -->
- [x] T006 Extend the exit-code matrix rows in `src/cmd/idea/main_test.go` for promote: usage→2 (arg count, `--main`, `--system`), operational→1 (no match, collision, outside git), success→0. <!-- R2 -->

### Phase 4: Polish

- [x] T007 [P] Document promote in `docs/specs/overview.md`: add the commands-table row and a short "Promote" semantics note (source→dest, collision refusal, no-op, flag conflicts, exit codes). <!-- R8 -->
- [x] T008 [P] Add promote to `docs/site/skill.md` and run `scripts/sync-skill.sh` to refresh `src/cmd/idea/skill/skill.md`; keep the 150-line budget green. <!-- R8 -->
- [x] T009 [P] Add the promote row to the `README.md` command list. <!-- R8 -->
- [x] T010 Run `cd src && gofmt -l . && go vet ./... && go test ./...` — all green, including the skill drift guard. <!-- R8 -->

## Execution Order

- T001 blocks T002 and T003; T003 blocks T004; T004 blocks T005/T006.
- T007/T008/T009 are independent [P] docs tasks; T010 runs last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `idea promote <query>` is a registered visible subcommand with wrapped `ExactArgs(1)`, enriched `Long`, and appears in `help-dump` output automatically
- [x] A-002 R3: Query resolution uses the shared `RequireSingle` with `FilterAll` — exact-ID precedence and ambiguity refusal behave identically to the other resolver commands
- [x] A-003 R4: A promoted idea lands in the main backlog with `id`, `date`, and `status` preserved verbatim, and is removed from the source
- [x] A-004 R5: Destination ID collision refuses with exit 1, no ID re-mint, both files unmodified
- [x] A-005 R6: Running from the main worktree is a no-op — exit 0, stderr `note:` advisory, no write
- [x] A-006 R2: `--main`/`--system` with promote exit 2; outside a git repo promote exits 1; `--file`/`IDEAS_FILE` applies within each root

### Behavioral Correctness

- [x] A-007 R4: Destination is written before the source removal (verified by code inspection + the destination-save-failure test leaving the source untouched)
- [x] A-008 R7: stdout is exactly `Promoted: {FormatLine}`; backfill advisories from both saves appear on stderr; `internal/idea` writes nothing to stderr

### Scenario Coverage

- [x] A-009 R4: Table-driven tests exist with real temp dirs AND a real git repo + linked worktree (Constitution V) covering the happy path, done-idea promotion, and dateless backfill
- [x] A-010 R5: Collision, no-match, and ambiguous-match scenarios are covered by tests asserting exit codes and file immutability

### Edge Cases & Error Handling

- [x] A-011 R4: Missing destination file and missing parent directory are created on promote; the result is canonical
- [x] A-012 R2: Exit-code matrix covers promote: usage→2, operational→1, success→0

### Code Quality

- [x] A-013 Pattern consistency: promote.go files mirror the surrounding factory/seam patterns (cmd factory + internal op, `resolveFile`-style resolution, established error wording style)
- [x] A-014 No unnecessary duplication: reuses `RequireSingle`, `removeIdeaAt`, `SaveFile`/`render`, `FormatLine`, `printBackfillNotice`, and the collision-check pattern instead of new variants
- [x] A-015 No magic strings/numbers: advisory and error texts follow existing constants/conventions; no god functions (>50 lines) in the new code
- [x] A-016 Constitution IV split holds: no business logic in `RunE` beyond orchestration; no output-channel policy in `internal/idea`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. The collision check in `src/internal/idea/promote.go:54-58` deliberately mirrors `checkIDCollision` over already-parsed ideas rather than replacing it (that helper re-loads from a path; `Promote` already holds the parsed destination).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | `Promote` lives in a new `src/internal/idea/promote.go` (+ `promote_test.go`) rather than growing `idea.go` | Mirrors the `fmt.go`/`prune_test.go` per-feature file split already in the package | S:60 R:90 A:85 D:80 |
| 2 | Confident | Destination append via `LoadFile` + `SaveFile`, not `Add`'s raw byte-append | One-canonical-write contract from the intake; preserves status and counts backfill for free; `Add`'s signature stays untouched | S:70 R:85 A:85 D:75 |
| 3 | Confident | Same-path no-op detected in the cmd layer by comparing the two resolved paths | Keeps `Promote` pure and stderr policy in `cmd/` (Constitution IV); covers absolute `--file` collapse | S:65 R:85 A:80 D:70 |
| 4 | Confident | Destination-save-failure test uses a directory-as-destination-path to prove source is untouched | Deterministic way to fail the first write without mocks (Constitution V forbids filesystem mocks) | S:50 R:90 A:75 D:70 |

4 assumptions (0 certain, 4 confident, 0 tentative).
