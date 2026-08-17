# Plan: Stale-Idea Surfacing on `idea list`

**Change**: 260816-szds-stale-idea-surfacing
**Intake**: `intake.md`

## Requirements

### CLI: `--stale` duration parsing

#### R1: Days-only duration parser
`internal/idea` SHALL provide a duration parser (e.g. `ParseStaleDays(s string) (int, error)`) that accepts a non-negative integer with an optional trailing `d` (`"90d"` and `"90"` both yield 90) and rejects everything else — negative numbers, non-integers, empty string, and any other unit (`"90h"`, `"3w"`) — with a descriptive error. The `cmd/` layer SHALL surface a parse failure as a usage error (exit 2) via the existing `usageArgs`/RunE usage-error convention.

- **GIVEN** the flag value `90d` **WHEN** parsed **THEN** the result is 90 with no error
- **GIVEN** the flag value `90` **WHEN** parsed **THEN** the result is 90 with no error
- **GIVEN** the flag value `-5`, `abc`, `90h`, `3w`, or `""` **WHEN** parsed **THEN** an error is returned **AND** `idea list --stale <that value>` exits 2 with a usage message

### CLI: staleness semantics

#### R2: Strictly-older-than stale predicate, dateless never stale
`internal/idea` SHALL provide a stale predicate (e.g. `IsStale(i Idea, days int, today time.Time) bool`) that is true iff the idea has a non-empty date strictly older than the cutoff `today − days`. An idea dated exactly `today − days` is NOT stale. A dateless idea (`Date == ""`) is never stale. The `today` value is a parameter (the cmd layer passes `time.Now()`) so tests inject fixed dates.

- **GIVEN** today is 2026-08-17 and `days` is 90 **WHEN** an idea is dated 2026-05-18 (89 days old) **THEN** it is not stale
- **GIVEN** the same today/days **WHEN** an idea is dated 2026-05-19 (exactly `today − 90`) **THEN** it is not stale (same-day boundary)
- **GIVEN** the same today/days **WHEN** an idea is dated 2026-05-18 or earlier than `today − 90` **THEN** it is stale
- **GIVEN** an idea with `Date == ""` **WHEN** evaluated with any threshold **THEN** it is not stale

#### R3: `--stale` flag on `list`/`ls` — open-only, mutually exclusive, composing
`idea list` SHALL gain a `--stale <duration>` string flag. When passed, the listing keeps only open ideas that are stale per R1+R2. `--stale` implies open-only: combining it with `--done` or `--all` SHALL be a usage error (exit 2). `--stale` SHALL compose unchanged with `--json` (same `{id,date,status,text}` schema, fewer rows), `--sort`/`--reverse` (applied to the filtered set), `--full`, and the `[id...]` positional filter (intersection; absent-ID stderr warnings unchanged).

- **GIVEN** a backlog with open ideas of mixed ages **WHEN** `idea list --stale 90d` runs **THEN** only open ideas strictly older than 90 days are listed
- **GIVEN** the same backlog **WHEN** `idea list --stale 90 --json` runs **THEN** output is a JSON array with the unchanged per-record schema and only the stale rows
- **GIVEN** any backlog **WHEN** `idea list --stale 90d --done` (or `--all`) runs **THEN** the command exits 2 with a flags-mutually-exclusive usage error
- **GIVEN** IDs `a7k2` (stale) and `b3c9` (fresh) **WHEN** `idea ls a7k2 b3c9 --stale 90d` runs **THEN** only `a7k2` is listed (intersection) and no warning is emitted for `b3c9` (it exists; it just isn't stale)

### Rendering: TTY-only age dimming

#### R4: Whole-line faint rendering past the effective threshold
On a terminal with color enabled, `idea list` SHALL render ideas past the **effective staleness threshold** with the existing ANSI faint style extended to the whole line (the text portion joins the already-dim prefix). The effective threshold is the `--stale` value when passed, else a fixed named default constant of 90 days (no magic numbers). Dimming is applied AFTER truncation (width math counts visible runes, never escape bytes). The threshold is threaded through `printIdeaLines` as a parameter with a named no-dimming sentinel; `prune`'s call site passes the sentinel so prune output is byte-identical to today.

- **GIVEN** stdout is a TTY, `NO_COLOR` unset, and an open idea dated 100 days ago **WHEN** `idea list` runs (no `--stale`) **THEN** that idea's whole line renders faint while ideas ≤ 90 days old render as today
- **GIVEN** the same terminal **WHEN** `idea list --stale 30d` runs **THEN** every listed idea (all >30 days old by construction) renders faint
- **GIVEN** `idea prune` in dry-run **WHEN** its listing renders **THEN** output is unchanged from today (no age dimming)

#### R5: Gating identical to existing color; machine contracts untouched
Age dimming SHALL key on the existing `UseColor(os.Stdout)` gate (TTY AND `NO_COLOR` unset — presence disables). Piped/redirected output SHALL remain full canonical `FormatLine` bytes (no ANSI, no ellipsis) including under `--stale`. The backlog line format, parser, `FormatLine`/`DisplayLine`, and the `--json` schema SHALL be untouched — no new stored fields.

- **GIVEN** `idea list --stale 90d | cat` **WHEN** output is inspected **THEN** it is byte-identical to the canonical `FormatLine` rendering of the filtered set (no escape codes)
- **GIVEN** `NO_COLOR=1 idea list` on a TTY **WHEN** a stale idea renders **THEN** no ANSI codes are emitted (truncation still applies)

### Docs: help text and toolkit doc surfaces

#### R6: Help text and doc surfaces updated
`list.go`'s `Long` help SHALL document `--stale` (units, open-only implication) and the age dimming (threshold, TTY/NO_COLOR gating, pipe contract unchanged); `Short` stays byte-stable. The README command-table row for `idea list` and the `docs/site/skill.md` list line SHALL mention `--stale`; `scripts/sync-skill.sh` SHALL be re-run so the embedded copy matches (drift-guard test stays green, 150-line budget respected). The help-dump JSON needs no manual edit (it reproduces `-h` output).

- **GIVEN** `idea list -h` **WHEN** read **THEN** `--stale` and the dimming behavior are documented and `Short` is unchanged
- **GIVEN** the skill drift-guard test **WHEN** `go test` runs after the docs edit **THEN** it passes (canonical and embedded copies match)

### Non-Goals

- No staleness on `prune`, `show`, or any other subcommand — `list`/`ls` only
- No new stored fields, no line-format or `--json` schema change (Constitution VI)
- No wall-clock display of age (e.g. "90d old" column) — dimming and filtering only
- No configurable default threshold (no env var, no config file) — a fixed named constant

### Design Decisions

#### Lexicographic ISO-date comparison for the stale predicate
**Decision**: Compute the cutoff as `today.AddDate(0, 0, -days).Format("2006-01-02")` and compare `i.Date < cutoff` as strings.
**Why**: Stored dates are validated `YYYY-MM-DD` (zero-padded ISO), where lexicographic order equals chronological order — no per-idea `time.Parse`, no error path on the render/filter hot loop.
**Rejected**: `time.Parse` per idea — introduces a parse-error branch for strings the parser already validated, for zero correctness gain.
*Introduced by*: 260816-szds-stale-idea-surfacing

#### No-dimming sentinel threaded through the shared render path
**Decision**: `printIdeaLines` gains a stale-days parameter with a named sentinel constant (e.g. `NoStaleDim = -1`) meaning "no age dimming"; `prune` passes the sentinel, `list` passes the effective threshold.
**Why**: Keeps the single shared render path (one home for the list/prune rendering policy) while scoping dimming to `list`; `--stale 0` remains a valid threshold ("older than today"), so 0 cannot be the sentinel.
**Rejected**: A separate list-only render function — duplicates the TTY/width/color mode selection the shared path exists to centralize.
*Introduced by*: 260816-szds-stale-idea-surfacing

#### Dimming keys on the effective threshold (intake assumption #5, carried forward)
**Decision**: One rule — "past the effective threshold ⇒ dim" — where the effective threshold is the `--stale` value when passed, else the 90-day default constant. A `--stale` result set therefore renders all-dim.
**Why**: One rule, no special case; all-dim under `--stale` is consistent (everything listed is, by construction, stale).
**Rejected**: Suppressing dimming when `--stale` is passed — a special case whose only benefit is cosmetic.
*Introduced by*: 260816-szds-stale-idea-surfacing

#### Done-checkbox green outranks age faint
**Decision**: If a done idea ever renders stale (unreachable via `list` today — `--stale` implies open-only and the default filter is open-only), the `[x]` keeps its green while the rest of the line dims.
**Why**: Explicit state signal beats an age hint; also keeps `DisplayListLine`'s span structure (dim spans around the checkbox) intact.
**Rejected**: Fainting the checkbox too — loses the done signal for zero simplification (the spans already exist).
*Introduced by*: 260816-szds-stale-idea-surfacing

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add `src/internal/idea/stale.go`: `DefaultStaleDimDays = 90` and `NoStaleDim = -1` named constants, `ParseStaleDays(s string) (int, error)`, `IsStale(i Idea, days int, today time.Time) bool` (lexicographic cutoff compare, dateless never stale); plus table-driven `src/internal/idea/stale_test.go` covering parse cases (`90d`, `90`, `0`, `-5`, `abc`, `90h`, `3w`, `""`) and predicate date math incl. the same-day boundary and dateless exclusion <!-- R1, R2 -->
- [x] T002 Extend `DisplayListLine` in `src/internal/idea/term.go` with a `stale bool` parameter: when true and color is on, the text portion also renders faint (whole line dim; a done `[x]` keeps green); applied after truncation; update `src/internal/idea/term_test.go` with stale-render cases (dim-after-truncation, NO_COLOR/no-color path emits no codes) <!-- R4, R5 -->
- [x] T003 Thread the threshold through `printIdeaLines` in `src/cmd/idea/output.go` (new `staleDays int` param + `today`; compute per-idea staleness via `idea.IsStale`, pass `stale` into `DisplayListLine`); update the `prune.go` call site to pass `idea.NoStaleDim` (output byte-identical to today) <!-- R4 -->
- [x] T004 Wire `--stale` in `src/cmd/idea/list.go`: `StringVar` flag, parse via `idea.ParseStaleDays` (usage error on bad value), `MarkFlagsMutuallyExclusive("stale", "done")` + `("stale", "all")`, filter the open list through `IsStale` with `time.Now()`, pass the effective threshold (flag value when `Changed`, else `DefaultStaleDimDays`) to `printIdeaLines`; update `Long` help text (`Short` unchanged) <!-- R3, R6 -->
- [x] T005 Binary tests in `src/cmd/idea/main_test.go` (existing `buildBinary`/`setupGitRepo`/`writeRepoBacklog`/`runSplit` helpers): `--stale` filters rows against a fixed-date backlog (both `90d` and bare `90`), `--json` row count with unchanged schema, `--stale --done` / `--stale --all` exit 2, composition with `[id...]` (intersection, no warning for existing-but-fresh ID) and `--sort`/`--reverse`, piped `--stale` output is canonical `FormatLine` bytes, invalid `--stale` values exit 2 <!-- R3, R5 -->

### Phase 3: Integration & Edge Cases

- [x] T006 Docs surfaces: update README.md `idea list` table row and `docs/site/skill.md` list line to mention `--stale`; run `scripts/sync-skill.sh` to refresh the embedded copy (drift guard + 150-line budget stay green); run `gofmt`, `go vet ./...`, and the full `go test ./...` from `src/` <!-- R6 -->

## Execution Order

- T001 blocks T002–T004 (constants/predicate are consumed downstream)
- T002 blocks T003 (signature change); T003 blocks T004 (threshold param)
- T005 follows T004; T006 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `ParseStaleDays` accepts `90d`/`90` → 90 and rejects negative, non-integer, empty, and non-`d`-unit values with errors; table-driven tests cover all cases
- [x] A-002 R2: `IsStale` implements strictly-older-than semantics — exactly-`today − N` is not stale, `Date == ""` is never stale — verified by table-driven tests with injected `today`
- [x] A-003 R3: `idea list --stale <N>` lists only open ideas strictly older than the cutoff; `--stale` + `--done`/`--all` exits 2 as a usage error
- [x] A-004 R4: on a color TTY, ideas past the effective threshold render whole-line faint (default 90d constant when `--stale` absent; the `--stale` value when passed); prune output is byte-identical to before
- [x] A-005 R5: piped `idea list --stale <N>` output is canonical `FormatLine` bytes (no ANSI, no `…`); `--json` schema `{id,date,status,text}` unchanged
- [x] A-006 R6: `list -h` documents `--stale` + dimming with `Short` byte-stable; README and `docs/site/skill.md` mention `--stale`; skill sync/drift guard green

### Behavioral Correctness

- [x] A-007 R3: `--stale` composes with `--sort`/`--reverse`/`--full` and the `[id...]` positional filter (intersection; stderr warnings only for absent IDs, not fresh ones)

### Scenario Coverage

- [x] A-008 R3: a binary test exercises `--stale 90d` and bare `--stale 90` end-to-end against a fixed-date backlog written by the test

### Edge Cases & Error Handling

- [x] A-009 R1/R2: edge inputs covered by tests — `--stale 0` (older than today), a dateless idea, an idea dated today, an idea dated exactly at the cutoff

### Code Quality

- [x] A-010 Pattern consistency: new code follows the term.go seam conventions (injected width/color/today, doc comments explaining contract rationale) and the cobra factory pattern
- [x] A-011 No unnecessary duplication: reuses `UseColor`/`TermWidth`/`truncateText`/`printIdeaLines`; no parallel render path
- [x] A-012 No magic numbers: 90-day default and the no-dimming sentinel are named constants; no god functions introduced
- [x] A-013 Constitution IV: parsing/predicate/dim logic lives in `internal/idea`; `cmd/list.go` contains only flag wiring, validation, and orchestration

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Lexicographic string compare of ISO dates for the stale predicate (no `time.Parse` per idea) | Stored dates are validated `YYYY-MM-DD` where lexicographic = chronological; avoids a dead error path | S:70 R:85 A:85 D:75 |
| 2 | Confident | `NoStaleDim = -1` sentinel threaded through `printIdeaLines`; prune passes it | `--stale 0` is a valid threshold so 0 can't be the sentinel; keeps the single shared render path | S:60 R:85 A:80 D:70 |
| 3 | Confident | New `stale.go`/`stale_test.go` files rather than growing `term.go` | term.go is the render seam; date math is filter logic — separate concern, same package | S:55 R:90 A:80 D:75 |
| 4 | Confident | `--stale` is a `StringVar` (accepts `90d`/`90`), presence detected via `Flags().Changed("stale")` | An IntVar can't accept the `d` suffix; `Changed` is the idiomatic cobra presence probe | S:65 R:85 A:85 D:80 |
| 5 | Confident | Whole-line dim = text portion joins the already-faint prefix; a done `[x]` keeps green (state > age) | Minimal delta to `DisplayListLine`'s existing span structure; done+stale unreachable via list anyway | S:55 R:85 A:75 D:70 |
| 6 | Confident | Mutual exclusion via cobra `MarkFlagsMutuallyExclusive` (flows through the existing usage-error exit-2 convention) | Idiomatic cobra; matches the repo's 0/1/2 exit-code convention for usage mistakes | S:60 R:85 A:85 D:80 |
| 7 | Confident | The mutual-exclusion conflict is rejected in a `PreRunE` returning `&usageError{...}`, in addition to `MarkFlagsMutuallyExclusive` | Cobra v1.8.1 runs `ValidateFlagGroups` after `PreRunE` but classifies nothing, so its group error would exit 1; the PreRunE check runs first and preserves the exit-2 convention (refines assumption 6) | S:60 R:85 A:80 D:75 |

7 assumptions (0 certain, 7 confident, 0 tentative).
