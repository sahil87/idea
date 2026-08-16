# Plan: Batch queries on `idea done` and `idea rm`

**Change**: 260816-xn2j-batch-done-rm-queries
**Intake**: `intake.md`

## Requirements

### CLI: Batch query surface

#### R1: `idea done` accepts multiple queries
`idea done` SHALL accept one or more `<query>` positionals (`Use: "done <query>..."`, `Args: usageArgs(cobra.MinimumNArgs(1))`). A single-query invocation MUST behave byte-identically to the current behavior (same stdout confirmation, same errors, same exit codes) — batch is purely additive (Constitution VI).

- **GIVEN** a backlog with open ideas `[a7k2]`, `[x9m1]`, and an idea whose text contains "auth-cleanup"
- **WHEN** `idea done a7k2 x9m1 auth-cleanup`
- **THEN** all three ideas are marked done in one invocation, exit 0
- **AND** `idea done a7k2` alone behaves exactly as today

#### R2: `idea rm` accepts multiple queries with batch consent
`idea rm` SHALL accept one or more `<query>` positionals. A single `--yes`/`-y`/`--force` consent MUST cover the whole batch; without consent the command refuses with the existing wording (`Use --yes (or --force) to confirm deletion`) before resolving anything, exit 1.

- **GIVEN** a backlog with ideas `[a7k2]` and `[x9m1]`
- **WHEN** `idea rm a7k2 x9m1 --yes`
- **THEN** both ideas are removed, exit 0
- **AND** `idea rm a7k2 x9m1` (no consent) removes nothing and exits 1 with the existing refusal

#### R3: `rm --dry-run` previews the whole batch
`idea rm --dry-run` SHALL preview **all** would-be-removed ideas (one `idea.FormatLine` line per match on stdout), write nothing, and win over `--yes`/`--force`. The preview MUST share the live match path so a no-match or ambiguous query aborts the preview with the identical error the live delete would give (preview-cannot-drift, toolkit principle №5).

- **GIVEN** ideas `[a7k2]` and `[x9m1]`
- **WHEN** `idea rm a7k2 x9m1 --dry-run --yes`
- **THEN** two `FormatLine` lines print, the backlog file is byte-identical, exit 0
- **AND** `idea rm a7k2 nosuch --dry-run` exits 1 with the usual no-match error and prints no preview lines

### internal/idea: Multi-resolve semantics

#### R4: All-or-nothing resolution with per-query RequireSingle semantics
A shared multi-resolver in `internal/idea` SHALL resolve every query independently with `RequireSingle` semantics (case-insensitive substring on ID/text, exact-ID precedence over substring matches, the command's filter — `FilterOpen` for done, `FilterAll` for rm). Any no-match or ambiguous query MUST abort the whole invocation with that query's existing error (including the `Multiple matches:` listing), exit 1, backlog byte-untouched — no partial mutation ever.

- **GIVEN** a batch where the second query is ambiguous (matches two ideas)
- **WHEN** `idea done a7k2 <ambiguous> x9m1`
- **THEN** the existing `Multiple matches:` error prints, exit 1
- **AND** the backlog file is byte-identical (idea `a7k2` was NOT marked done)

#### R5: Dedupe with advisory
Two queries resolving to the same idea SHALL act on it once. Deduping is keyed on the resolved idea index and preserves first-occurrence order. The command layer MUST emit a one-line advisory on **stderr** when deduping occurred (stdout stays per-item machine-parseable lines only, per the advisory-notes-to-stderr channel policy).

- **GIVEN** idea `[a7k2]` whose text contains "auth"
- **WHEN** `idea done a7k2 auth`
- **THEN** the idea is marked done once, one `Done:` line prints on stdout, exit 0
- **AND** a `note: …` advisory prints on stderr

#### R6: Single canonical write
Each successful batch invocation SHALL perform exactly one `SaveFile` (one atomic write, one whole-file normalization), so the date-backfill advisory prints **at most once** via the existing `printBackfillNotice`.

- **GIVEN** a backlog with two dateless ideas being marked done in one batch
- **WHEN** `idea done <q1> <q2>`
- **THEN** the file is written once and exactly one `note: stamped today's date on 2 previously-dateless item(s)` prints on stderr

### CLI: Output contract

#### R7: Per-item stdout lines in argument order
Each acted idea SHALL print its own confirmation line — `Done: {FormatLine}` / `Removed: {FormatLine}` — in argument order (first occurrence for deduped queries), keeping stdout line-per-record scriptable. Exit codes stay 0 success / 1 operational / 2 usage (zero positionals exits 2 via the `usageArgs` wrap).

- **GIVEN** a valid three-query batch
- **WHEN** `idea done q1 q2 q3`
- **THEN** three `Done:` lines print on stdout in the order q1, q2, q3
- **AND** `idea done` (zero args) exits 2

### Non-Goals

- `idea reopen` stays single-query — the backlog's own tie-breaker; batch reopen adds CLI surface + test matrix and is a trivial follow-up if wanted.
- No changes to `list`/`prune`/`edit`/`show`, the line format, IDs, or JSON schemas.
- No aggregate multi-error reporting — the first failing query's usual error is the abort message.

### Design Decisions

#### Modify existing signatures instead of adding *Many variants
**Decision**: `Done`, `Rm`, and `RmPreview` change signature to accept `queries []string` and return `[]Idea` (e.g. `Done(path string, queries []string) ([]Idea, int, error)`).
**Why**: `internal/` package with a single `cmd/` consumer — no external Go API contract; one code path keeps single-query and batch behavior provably identical.
**Rejected**: parallel `DoneMany`/`RmMany` functions — duplicated utilities (code-quality anti-pattern) that would let the two paths drift.
*Introduced by*: 260816-xn2j-batch-done-rm-queries

#### Dedupe advisory computed at the cmd layer
**Decision**: the resolver returns deduped indices; the cmd layer computes the dedupe count as `len(queries) − len(actedIdeas)` (valid because all-or-nothing guarantees every query resolved to exactly one idea) and prints the stderr advisory itself.
**Why**: keeps `internal/idea` stderr-free (Constitution IV output-channel split) and avoids growing the return signature with dedupe metadata.
**Rejected**: returning a dedupe report struct from `internal/idea` — extra API surface for information the caller can derive.
*Introduced by*: 260816-xn2j-batch-done-rm-queries

#### Descending-index removal inside Rm
**Decision**: `Rm` removes resolved ideas via the existing `removeIdeaAt` seam in **descending idea-index order**.
**Why**: `removeIdeaAt` shifts the indices of every idea after the removed one; descending order keeps the remaining resolved indices valid without recomputation.
**Rejected**: re-resolving after each removal — N loads/scans and re-introduces mid-batch ambiguity windows.
*Introduced by*: 260816-xn2j-batch-done-rm-queries

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add unexported `resolveMany(f *File, queries []string, filter FilterKind) ([]int, error)` to `src/internal/idea/idea.go`: per-query `RequireSingle` call (reusing it verbatim for exact-ID precedence + error wording), abort on first failing query, dedupe on resolved index preserving first-occurrence order <!-- R4 -->
- [x] T002 Change `Done` in `src/internal/idea/idea.go` to `Done(path string, queries []string) ([]Idea, int, error)`: load once, `resolveMany` with `FilterOpen`, flip `Done = true` on each index, single `SaveFile`, return acted ideas in resolution order <!-- R6 -->
- [x] T003 Change `Rm` to `Rm(path string, queries []string, force bool) ([]Idea, int, error)` (consent check first, `resolveMany` with `FilterAll`, capture removed ideas in first-occurrence order, remove via `removeIdeaAt` in descending index order, single `SaveFile`) and `RmPreview` to `RmPreview(path string, queries []string) ([]Idea, error)` sharing the same load+resolve path <!-- R4 -->
- [x] T004 Update `src/cmd/idea/done.go`: `Use: "done <query>..."`, `Args: usageArgs(cobra.MinimumNArgs(1))`, `RunE` passes `args`, prints one `Done: {FormatLine}` line per acted idea, prints the dedupe stderr advisory when `len(args) > len(acted)`, extends `Long` with batch semantics + example <!-- R1 -->
- [x] T005 Update `src/cmd/idea/rm.go`: same surface change (`rm <query>...`), batch `--dry-run` printing all previews, batch consent pass-through, per-item `Removed:` lines, dedupe advisory, extended `Long` <!-- R2 -->

### Phase 3: Integration & Edge Cases

- [x] T006 [P] Table-driven tests in `src/internal/idea/idea_test.go`: happy multi-query `Done`/`Rm`, mixed valid+ambiguous batch aborts with file byte-untouched, mixed valid+no-match batch aborts, dedupe (exact ID + substring of same idea acts once), `FilterOpen` interaction (done idea not resolvable by `done`), multi-index `Rm` removal correctness (non-adjacent indices), multi-query `RmPreview` returns all without writing <!-- R4 -->
- [x] T007 [P] CLI tests in `src/cmd/idea/main_test.go`: multi-arg `done`/`rm` per-item stdout lines in argument order, batch `rm --dry-run` previews all + file byte-identical + wins over `--yes`, batch consent refusal exits 1, single-arg invocations byte-identical to current output, zero-arg `done`/`rm` exit 2, single backfill advisory on a dateless batch <!-- R7 -->

### Phase 4: Polish

- [x] T008 Check the changed CLI surface against toolkit standards (`shll standards` — principles entry at minimum), verify `-h` output for both commands renders the new `<query>...` usage, run the full test suite (`cd src && go test ./...`) and `gofmt` <!-- R1 -->

## Execution Order

- T001 blocks T002/T003; T002/T003 block T004/T005; T006/T007 [P] after T004/T005; T008 last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `idea done` accepts 1+ queries; multi-query batch marks all matched open ideas done in one invocation
- [x] A-002 R2: `idea rm` accepts 1+ queries; one consent flag covers the batch; no-consent refusal unchanged and removes nothing
- [x] A-003 R3: batch `rm --dry-run` prints one `FormatLine` per match, writes nothing, wins over consent flags
- [x] A-004 R4: a shared `internal/idea` multi-resolver gives every query `RequireSingle` semantics incl. exact-ID precedence; the multi-resolve loop is not in `cmd/`

### Behavioral Correctness

- [x] A-005 R1: single-query `done`/`rm` invocations are byte-identical to pre-change behavior (stdout, stderr, exit codes)
- [x] A-006 R6: a successful batch performs exactly one `SaveFile`; the backfill advisory prints at most once
- [x] A-007 R5: duplicate queries resolving to one idea act once and emit a stderr advisory; stdout carries only per-item confirmation lines

### Scenario Coverage

- [x] A-008 R4: test proves a mixed valid+ambiguous batch aborts with the existing `Multiple matches:` error and a byte-untouched backlog
- [x] A-009 R4: test proves a mixed valid+no-match batch aborts with the existing no-match error and a byte-untouched backlog
- [x] A-010 R7: test proves per-item output lines print in argument order for a 3-query batch <!-- review: ordering proven at both layers — the CLI test asserts exact stdout lines for a reversed-input 2-query batch, and internal TestRm_Batch asserts first-occurrence order for a 3-query batch (with dedupe) -->

### Edge Cases & Error Handling

- [x] A-011 R7: zero positionals on `done`/`rm` exit 2 (usage) via `usageArgs(cobra.MinimumNArgs(1))`
- [x] A-012 R4: `done` batch resolution honors `FilterOpen` (an already-done idea is not resolvable, and its exact ID is not force-selected past the filter)
- [x] A-013 R4: multi-index `rm` with non-adjacent indices removes the correct lines (descending-order removal; verified against file content)

### Code Quality

- [x] A-014 Pattern consistency: new code follows existing `cmd/` thin-wiring + `internal/idea` logic split (Constitution III/IV); table-driven tests with real temp dirs (Constitution V)
- [x] A-015 No unnecessary duplication: `RequireSingle` is reused by the multi-resolver (not reimplemented); `removeIdeaAt` reused for removal; no parallel `*Many` copies of mutation logic
- [x] A-016 No magic strings: advisory wording and any new constants named per surrounding conventions

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant (`RequireSingle`, `removeIdeaAt`, and the single-query callers `show`/`reopen`/`edit` all remain live; the batch path reuses them rather than replacing them)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Dedupe count derived at cmd layer as `len(queries) − len(acted)`; advisory wording finalized at apply (e.g. `note: N duplicate query(ies) matched an already-selected idea; acted once`) | All-or-nothing guarantees 1:1 query→idea resolution, so the derivation is sound; keeps internal stderr-free | S:60 R:90 A:80 D:70 |
| 2 | Confident | Keep `([]Idea, int, error)` return shape (slice replaces single `Idea`) rather than introducing a result struct | Minimal diff against the established `(Idea, int, error)` convention shared by Done/Reopen/Edit/Rm | S:60 R:85 A:80 D:70 |
| 3 | Certain | `Rm` removes in descending resolved-index order via existing `removeIdeaAt` | Index-shift semantics of `removeIdeaAt` make this the only correct single-pass order | S:75 R:80 A:95 D:90 |
| 4 | Confident | Resolver is unexported (`resolveMany`) — exported surface stays `Done`/`Rm`/`RmPreview` | Only in-package consumers exist; smallest API growth | S:55 R:90 A:85 D:75 |

4 assumptions (1 certain, 3 confident, 0 tentative).
