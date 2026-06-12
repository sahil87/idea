# Plan: Add `idea prune` Subcommand

**Change**: 260612-drc1-add-prune-subcommand
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

### internal/idea: Prune Operation

#### R1: Exported `Prune` API
A new exported op `Prune(path string, force bool) ([]Idea, int, error)` MUST live in `src/internal/idea` (Constitution IV). It SHALL return the removed (or would-be-removed) done ideas in file order, the date-backfill count from the save, and an error.

- **GIVEN** a backlog file with open and done ideas
- **WHEN** `Prune(path, true)` is called
- **THEN** it returns every `Done == true` idea in file order, the `SaveFile` backfill count, and a nil error

#### R2: Dry Run Never Writes
With `force == false`, `Prune` MUST NOT write the file: it loads, collects the done ideas, and returns them with backfill count 0 (`SaveFile` never runs).

- **GIVEN** a mixed backlog containing lines that a save would normalize (variant bullet, dateless idea)
- **WHEN** `Prune(path, false)` is called
- **THEN** the returned slice holds the done ideas in file order with count 0
- **AND** the file bytes are unchanged

#### R3: Force Removes All Done Ideas via the SaveFile Seam
With `force == true`, `Prune` MUST remove every `Done == true` idea from the `File` (entries from `lines`/`ideas`/`ideaIndices`, index-adjusted — same bookkeeping as `Rm`), then call `SaveFile`. It thereby inherits the existing save semantics: canonical rewrite of surviving idea lines, date backfill on previously-dateless survivors (count returned), atomic temp-file-plus-rename write, and non-idea lines (headers, blank lines, prose) preserved verbatim (Constitution I).

- **GIVEN** a backlog with headers/prose, open ideas (one dateless), and done ideas
- **WHEN** `Prune(path, true)` is called
- **THEN** only the `[x]` idea lines are removed; surviving idea lines are canonically rewritten
- **AND** non-idea lines pass through verbatim
- **AND** the dateless survivor is stamped with today's date and the returned count reflects it

#### R4: Zero Done Items Is a Successful No-Op
Zero done items MUST NOT be an error: `Prune` returns an empty slice and nil error in both modes. With `force` and zero done items, `SaveFile` MUST NOT run (file untouched) — a no-op mutation would otherwise trigger whole-file normalization/backfill as a surprise side effect.

- **GIVEN** an all-open backlog containing lines a save would normalize
- **WHEN** `Prune(path, force)` is called with either force value
- **THEN** the result is empty with a nil error
- **AND** the file bytes are unchanged

#### R5: Missing File Errors via LoadFile
A missing backlog file MUST error naturally via `LoadFile`'s `os.ReadFile` error, matching every other mutating command (`done`/`reopen`/`edit`/`rm`). No special-casing.

- **GIVEN** no backlog file at the resolved path
- **WHEN** `Prune(path, force)` is called with either force value
- **THEN** the `os.ReadFile` error propagates and the command exits non-zero

### cmd/idea: prune Subcommand

#### R6: Factory Registration and Flag Wiring
A `pruneCmd()` factory MUST exist in a new file `src/cmd/idea/prune.go` (modeled on `rmCmd()` in `rm.go`) and be registered in `newRootCmd()`'s `AddCommand` list (`src/cmd/idea/main.go`). The only new flag is the local `--force`; the command inherits the persistent `--file`/`--main` from root (Constitution III). It SHALL take no positional args (`cobra.NoArgs`), carry an enriched cobra `Long` (worktree-vs-`--main` note, `--force` semantics, inline example) per the repo-wide help-text convention, and define no alias. The command appears in `help-dump` output automatically via the factory tree walk — no JSON schema change.

- **GIVEN** the built binary
- **WHEN** `idea prune -h` runs
- **THEN** the enriched `Long`, the `--force` flag, and the inherited `--file`/`--main` flags are shown

#### R7: Dry-Run Output Contract
`idea prune` (no `--force`) with done items present MUST print one line per done idea via `idea.FormatLine` to **stdout** (pipe-friendly, e.g. `idea prune | wc -l`), the confirm hint `Re-run with --force to confirm.` to **stderr** (advisory, per the `printBackfillNotice` stderr precedent and Constitution VI), and exit 0 — a successful preview, not `rm`'s error-path refusal.

- **GIVEN** a backlog with done ideas
- **WHEN** `idea prune` runs
- **THEN** stdout is exactly the `FormatLine` rendering of each done idea, in file order
- **AND** stderr is exactly `Re-run with --force to confirm.`
- **AND** the exit code is 0 and the file is untouched

#### R8: Force Output Contract
`idea prune --force` with done items present MUST print `Pruned N done idea(s).` — count only, no per-line listing — to stdout and exit 0. The wrapper SHALL call `printBackfillNotice(cmd, backfilled)` so the backfill advisory goes to stderr exactly as in `done`/`rm`/`edit`.

- **GIVEN** a backlog with 2 done ideas
- **WHEN** `idea prune --force` runs
- **THEN** stdout is exactly `Pruned 2 done idea(s).` and the done lines are gone from the file
- **AND** any backfill advisory appears on stderr only (suppressed at count 0)

#### R9: Empty-Case Output Contract
With no done items, both `idea prune` and `idea prune --force` MUST print `No done ideas to prune.` to stdout, exit 0, and leave the file untouched (no save).

- **GIVEN** an all-open backlog
- **WHEN** `idea prune` or `idea prune --force` runs
- **THEN** stdout is exactly `No done ideas to prune.`, the exit code is 0, and the file bytes are unchanged

### Tests

#### R10: Table-Driven Internal Coverage
Table-driven tests against real temp dirs (`t.TempDir()`, Constitution V) MUST cover the five intake cases in `src/internal/idea`: (1) mixed file `--force` removes only `[x]` lines, (2) all-open file is a no-op in both modes, (3) dry run on a mixed file returns the done ideas with file bytes unchanged, (4) `--force` preserves non-idea lines verbatim, (5) a previously-dateless surviving open item is stamped on the `--force` save with the count reflected.

- **GIVEN** the prune test table
- **WHEN** `go test ./internal/idea/ -run Prune` runs
- **THEN** all five intake cases (plus the missing-file error path) pass

#### R11: CLI-Level Output-Channel Coverage
A subprocess test in `src/cmd/idea/main_test.go` SHOULD verify the command wiring end to end: dry-run stdout/stderr split, force count-only output, empty-case message, exit codes, and file state — reusing the existing `buildBinary`/`setupGitRepo`/`writeRepoBacklog`/`runSplit`/`readRepoBacklog` helpers.

- **GIVEN** a seeded repo backlog
- **WHEN** the prune CLI test table runs against the built binary
- **THEN** stdout, stderr, exit code, and resulting file content match the R7/R8/R9 contracts

### Non-Goals

- `prune --before YYYY-MM-DD` (prune only old done items) — explicitly deferred; v1 prunes **all** done items
- Any archive/undo mechanism — rejected in intake (git history is the recovery path; a second file breaks the one-file contract)
- An alias for `prune` — aliases are namespace decisions per the `ls` precedent; none was discussed

### Design Decisions

1. **New `prune` verb, not a `--done` bulk mode on `rm`**: preserves `rm`'s `ExactArgs(1)` single-match-or-refuse safety contract — *Why*: strong CLI precedent (`git remote prune`, `docker system prune`) — *Rejected*: overloading `rm`
2. **Bare invocation is a free dry run**: mirrors `rm`'s `--force` convention but makes the refusal a useful preview that exits 0 — *Why*: user-approved DX — *Rejected*: erroring without `--force`
3. **Count-only `--force` stdout**: per-line listing is reserved for the dry run — *Why*: user decided the one open question; keeps `--force` quiet for scripting — *Rejected*: listing on both paths

## Tasks

### Phase 1: Core Implementation (internal/idea)

- [x] T001 Implement `Prune(path string, force bool) ([]Idea, int, error)` in `src/internal/idea/idea.go` (after `Rm`): load, collect `Done == true` ideas in file order; return early (count 0, no save) when force is false or nothing matched; otherwise remove the done entries from `lines`/`ideas`/`ideaIndices` with `Rm`-style index adjustment and call `SaveFile` <!-- R1 R2 R3 R4 R5 -->
- [x] T002 Add table-driven tests in new `src/internal/idea/prune_test.go` covering the five intake cases (mixed force, all-open both modes with byte-identical no-save proof, dry-run-untouched, verbatim non-idea lines, backfill on dateless survivor) plus a missing-file error test, reusing `writeBacklog` <!-- R10 -->

### Phase 2: CLI Wiring (cmd/idea)

- [x] T003 Create `src/cmd/idea/prune.go` with `pruneCmd()` factory modeled on `rmCmd()`: enriched `Long`, `cobra.NoArgs`, local `--force` flag, `resolveFile()` + `idea.Prune` orchestration, empty-case message, dry-run `FormatLine` listing + stderr confirm hint, force count-only output + `printBackfillNotice` <!-- R6 R7 R8 R9 -->
- [x] T004 Register `pruneCmd()` in `newRootCmd()`'s `AddCommand` list in `src/cmd/idea/main.go` <!-- R6 -->
- [x] T005 Add subprocess test `TestPrune_CLIOutputContract` in `src/cmd/idea/main_test.go` (table-driven: dry run, force, empty case both modes) asserting exact stdout/stderr and resulting backlog content via the existing helpers <!-- R11 -->

### Phase 3: Validation

- [x] T006 Run scoped tests (`cd src && go test ./internal/idea/ -run Prune` and `go test ./cmd/idea/ -run Prune`), then the full suite (`go test ./...`), `gofmt -l .`, and `go vet ./...` — all clean <!-- R10 R11 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `Prune(path string, force bool) ([]Idea, int, error)` is exported from `internal/idea`; the cmd layer contains no pruning logic beyond flag wiring and output formatting
- [x] A-002 R6: `pruneCmd()` exists in `src/cmd/idea/prune.go`, is registered in `newRootCmd()`, defines local `--force`, takes no positional args, inherits `--file`/`--main`, and carries an enriched `Long` with the worktree note and an inline example

### Behavioral Correctness

- [x] A-003 R2: `force == false` returns the done ideas in file order with backfill count 0 and leaves the file byte-identical (no `SaveFile` call)
- [x] A-004 R3: `force == true` removes every `[x]` idea line; survivors are canonically rewritten via the `SaveFile` seam and the backfill count is returned
- [x] A-005 R4: zero done items returns an empty slice and nil error in both modes; with force, no save occurs (file byte-identical — no normalization side effect)
- [x] A-006 R7: dry-run stdout is exactly one `FormatLine` line per done idea; `Re-run with --force to confirm.` goes to stderr; exit code 0
- [x] A-007 R8: force stdout is exactly `Pruned N done idea(s).` (count only); the backfill advisory goes to stderr via `printBackfillNotice` and is suppressed at count 0
- [x] A-008 R9: `No done ideas to prune.` on stdout with exit 0 in both modes, file untouched

### Edge Cases & Error Handling

- [x] A-009 R5: a missing backlog file surfaces `LoadFile`'s `os.ReadFile` error and the command exits non-zero (no special-casing)
- [x] A-010 R3: non-idea lines (headers, blank lines, prose between items) are preserved verbatim through a force prune

### Scenario Coverage

- [x] A-011 R10: table-driven tests against `t.TempDir()` cover all five intake cases plus the missing-file path, and pass
- [x] A-012 R11: the CLI subprocess test verifies the stdout/stderr split, exit codes, and file state for dry-run, force, and empty cases

### Code Quality

- [x] A-013 Pattern consistency: `prune.go` mirrors `rm.go`'s factory shape; `Prune` mirrors the existing op conventions (path-first signature, `(result, backfillCount, error)` return, doc comment referencing `Done` for the count)
- [x] A-014 No unnecessary duplication: reuses `LoadFile`/`SaveFile`/`FormatLine`/`printBackfillNotice`/`resolveFile` — no parallel helpers introduced
- [x] A-015 No god functions: `Prune` and the `pruneCmd` `RunE` body each stay focused and under ~50 lines
- [x] A-016 `gofmt -l` reports nothing and `go vet ./...` is clean over `src/`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | `Args: cobra.NoArgs` on prune | Intake is silent on positional-arg handling; prune takes no query, and rejecting stray args loudly beats silently ignoring them; trivially reversible | S:45 R:90 A:80 D:75 |
| 2 | Confident | `Prune` goes in `idea.go` (after `Rm`); tests in a new `prune_test.go` sibling | Intake offered "idea.go or a sibling file" for each; keeping all ops in idea.go matches the existing layout, while idea_test.go at 1700+ lines makes a test sibling the maintainable choice | S:55 R:95 A:85 D:70 |
| 3 | Confident | CLI-level subprocess test is added (intake said MAY) | The stdout/stderr channel split is central to this change's DX contract and the repo has direct precedent (`TestDone_BackfillNoticeOnStderr`); low cost, reuses existing helpers | S:55 R:95 A:90 D:80 |

3 assumptions (0 certain, 3 confident, 0 tentative).
