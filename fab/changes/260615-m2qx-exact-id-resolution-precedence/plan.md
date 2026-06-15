# Plan: Fix exact-ID resolution precedence in single-idea subcommands

**Change**: 260615-m2qx-exact-id-resolution-precedence
**Intake**: `intake.md`

## Requirements

### Resolver: Exact-ID Precedence in `RequireSingle`

#### R1: Exact-ID match wins over substring matches
`RequireSingle(query, ideas, filter)` SHALL select an idea whose `ID` equals `query` (case-insensitive) outright, before performing any substring matching, so a unique exact ID is always reachable by its own ID even when that ID string appears as a substring inside another idea's text.

- **GIVEN** two open ideas A and B, where A's exact ID appears verbatim as a substring inside B's text (e.g. a cross-reference like `[jznd]`)
- **WHEN** `RequireSingle("<A-id>", ideas, FilterOpen)` is called
- **THEN** it returns idea A and A's index with no error
- **AND** it does NOT return the "Multiple matches" error

#### R2: Exact-ID match is case-insensitive
The exact-ID precedence check SHALL compare `query` against `idea.ID` using `strings.EqualFold` (case-insensitive), consistent with the existing case-insensitive `Match` semantics.

- **GIVEN** an idea A with ID `a7k2` and another idea whose text contains `a7k2`
- **WHEN** `RequireSingle("A7K2", ideas, FilterOpen)` is called with a mixed/upper-case query
- **THEN** it resolves to idea A via the exact-ID path with no error

#### R3: Exact-ID precedence respects the active FilterKind
The exact-ID check SHALL only consider ideas that pass the active `FilterKind` (via the existing `matchesFilter`), so an exact-ID match on a filtered-out idea (e.g. a done idea under `FilterOpen`) does NOT short-circuit.

- **GIVEN** a done idea A with ID `a7k2`
- **WHEN** `RequireSingle("a7k2", ideas, FilterOpen)` is called
- **THEN** the exact-ID path does NOT return A (it is filtered out)
- **AND** `RequireSingle("a7k2", ideas, FilterDone)` and `RequireSingle("a7k2", ideas, FilterAll)` DO resolve to A

#### R4: Substring fallback behavior is unchanged for non-exact-ID queries
When `query` is not an exact (case-insensitive) ID of any filtered idea, `RequireSingle` SHALL fall through to the existing `Match`-based substring loop with its unchanged `len == 0` ("No idea matching") and `len > 1` ("Multiple matches") error branches. `Match()`, `FindAll()`, `matchesFilter()`, and the `FilterKind` constants SHALL NOT be modified.

- **GIVEN** ideas where the query is a genuine text substring of exactly one idea
- **WHEN** `RequireSingle(query, ideas, filter)` is called
- **THEN** it resolves that single idea
- **AND** an ambiguous substring query still errors with "Multiple matches"
- **AND** a zero-hit query still errors with "No idea matching"

#### R5: Defensive handling of impossible duplicate exact IDs
If two or more filtered ideas somehow share the same exact ID under the active filter (impossible given Constitution VI's uniqueness guarantee), `RequireSingle` SHALL fall through to the substring logic rather than guess which to return.

- **GIVEN** two filtered ideas that both match `query` exactly by ID (a contrived violation of ID uniqueness)
- **WHEN** `RequireSingle(query, ideas, filter)` is called
- **THEN** the exact-ID short-circuit is skipped and control falls through to the substring loop (which reports them as multiple matches)

### Design Decisions

1. **Exact-ID precedence (FIX OPTION 1)**: short-circuit at the top of `RequireSingle` — *Why*: backlog idea names it the cleanest fix; soundness rests on Constitution VI (IDs unique per backlog), so an exact ID match can never be genuinely ambiguous; preserves substring search for non-ID queries — *Rejected*: a new `--id` selector (adds public CLI surface); special-casing only the mixed-ambiguity branch (a strict subset of Option 1).
2. **Single shared seam**: implement only in `RequireSingle` in `internal/idea`, no `cmd/` changes — *Why*: all five single-idea subcommands (`show`/`done`/`reopen`/`edit`/`rm`) route through `RequireSingle`; Constitution IV mandates logic in `internal/idea` — *Rejected*: per-subcommand fixes (duplication, drift risk).

### Non-Goals

- Changing `Match()` / `FindAll()` substring semantics (the `list` fuzzy search keeps current behavior).
- Adding any new CLI flag or subcommand.
- Changing output formats or JSON schema (public contract per Constitution VI is untouched).

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add an exact-ID precedence short-circuit at the top of `RequireSingle` in `src/internal/idea/idea.go`, before the substring-collection loop: iterate ideas, skip those failing `matchesFilter(idea, filter)`, track a single index where `strings.EqualFold(idea.ID, query)`; if exactly one filtered idea matches, return it and its index immediately; on a second exact match, abandon the short-circuit (set index to -1) and fall through. Leave `Match`, `FindAll`, `matchesFilter`, and the substring branches unchanged. <!-- R1 R2 R3 R5 -->

### Phase 3: Integration & Edge Cases

- [x] T002 Add table-driven regression tests in `src/internal/idea/idea_test.go` (GIVEN/WHEN/THEN scenario names, in-memory `[]Idea`) covering: primary repro (A's ID is a substring of B's text, FilterOpen resolves A), case-insensitive exact-ID query, FilterKind precedence (done idea's ID not returned under FilterOpen; returned under FilterDone/FilterAll), and substring no-regression (single substring hit resolves, ambiguous errors "Multiple matches", zero hits errors "No idea matching"). <!-- R1 R2 R3 R4 R5 -->

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `RequireSingle` returns the exact-ID owner (and correct index) when that ID is a substring of another filtered idea's text, with no error.
- [ ] A-002 R2: A mixed/upper-case query of an idea's ID resolves to that idea via the exact-ID path.
- [ ] A-003 R3: The exact-ID check respects `FilterKind` — a done idea's ID is not returned under `FilterOpen` but is under `FilterDone`/`FilterAll`.
- [ ] A-004 R4: Non-exact-ID substring queries retain prior behavior; `Match`/`FindAll`/`matchesFilter`/`FilterKind` are byte-for-byte unchanged.
- [ ] A-005 R5: Duplicate exact-ID under the active filter falls through to the substring loop rather than guessing.

### Behavioral Correctness

- [ ] A-006 R1: The primary repro (jznd/qg64/m2qx-style scenario) that previously errored "Multiple matches" now resolves the exact-ID owner.

### Scenario Coverage

- [ ] A-007 R1: A table-driven regression test exercises the primary repro and asserts the resolved idea and index.
- [ ] A-008 R3: Tests exercise FilterOpen/FilterDone/FilterAll precedence for the same exact ID.

### Edge Cases & Error Handling

- [ ] A-009 R4: Tests assert ambiguous substring still errors "Multiple matches" and zero hits still errors "No idea matching".

### Code Quality

- [ ] A-010 Pattern consistency: New code follows surrounding naming/structure; tests are table-driven with GIVEN/WHEN/THEN names per stage directives.
- [ ] A-011 No unnecessary duplication: Reuses existing `matchesFilter`/`Match`; no logic added to `cmd/`.
- [ ] A-012 Formatting & vet: `gofmt -l internal/idea/` reports no files and `go vet ./...` is clean.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Adopt exact-ID-wins precedence (FIX OPTION 1) in `RequireSingle` only | Named cleanest in backlog; root-caused against source; Constitution VI guarantees ID uniqueness; all five subcommands share this seam (Constitution IV) | S:95 R:85 A:95 D:92 |
| 2 | Certain | Use `strings.EqualFold` and gate the exact-ID check by the active `FilterKind` | Case-insensitive exact match per intake; existing call sites pass FilterOpen/Done/All and a done idea's ID must not resolve on an open-only path | S:92 R:85 A:95 D:90 |
| 3 | Certain | Leave `Match`/`FindAll`/`matchesFilter`/`FilterKind` unchanged | Minimal correct fix; preserves `list` fuzzy search and the public contract (Constitution VI) | S:95 R:80 A:90 D:90 |
| 4 | Confident | On a second exact-ID match under the filter, abandon the short-circuit and fall through to substring logic | Constitution VI makes this unreachable; "fall through, don't guess" is the safe default and trivially reversible | S:80 R:85 A:85 D:82 |
| 5 | Confident | Add the new regression tests as table-driven with GIVEN/WHEN/THEN names (existing per-case `TestRequireSingle_*` left intact) | Constitution V + apply stage_directives mandate table-driven + GIVEN/WHEN/THEN; `RequireSingle` operates on in-memory `[]Idea` so no temp files needed | S:85 R:88 A:88 D:85 |

5 assumptions (3 certain, 2 confident, 0 tentative).
