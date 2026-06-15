# Plan: Exact-ID Match Precedence in RequireSingle

**Change**: 260615-m2qx-exact-id-match-precedence
**Intake**: `intake.md`

## Requirements

### Resolver: Exact-ID Precedence in `RequireSingle`

#### R1: Exact-ID match wins over incidental substring matches
`RequireSingle` SHALL select a single idea when exactly one of the candidate matches has an `ID` equal to the query (case-insensitive), even when other candidates match only because the query is a substring of their text. The exact-ID owner MUST be returned with its original index and no error.

- **GIVEN** a backlog with idea[0] `{ID: "jznd", Text: "the idea to edit"}` and idea[1] `{ID: "qg64", Text: "see related [jznd] for context"}`, both matching query `"jznd"` via `Match`
- **WHEN** `RequireSingle("jznd", ideas, FilterAll)` is called
- **THEN** it returns idea[0] (ID `"jznd"`) at index 0 with no error
- **AND** the previous "Multiple matches" abort no longer occurs for this input

#### R2: Exact-ID equality is case-insensitive
The exact-ID precedence comparison SHALL use `strings.EqualFold(match.ID, query)` so it aligns with the existing case-insensitive `Match` semantics and introduces no new dependency.

- **GIVEN** a candidate match whose `ID` equals the query under case folding
- **WHEN** the precedence pass scans the match set
- **THEN** that candidate is counted as an exact-ID hit regardless of letter case

#### R3: Zero or multiple exact-ID hits fall through unchanged
When the number of exact-ID hits among the matches is not exactly one, `RequireSingle` MUST preserve its existing behavior: a single overall match returns normally, and two-or-more matches (including the defensive `exactCount > 1` case) return the existing "Multiple matches" ambiguity error with no silent pick.

- **GIVEN** a query that matches multiple ideas where none (or more than one) is an exact-ID hit
- **WHEN** `RequireSingle` is called
- **THEN** it returns the existing "Multiple matches: ... Be more specific or use the exact ID." error

#### R4: Filter semantics are preserved
The exact-ID precedence pass MUST operate over the already-filtered match set (post-`matchesFilter`), so an exact-ID idea excluded by the filter is never force-selected.

- **GIVEN** an exact-ID idea filtered out by the active `FilterKind`
- **WHEN** `RequireSingle` runs
- **THEN** the filtered-out idea is not present in the match set and therefore cannot be selected by the precedence pass

### Non-Goals

- Modifying `Match` or `FindAll` — they keep pure case-insensitive substring semantics for `idea list`/search (Constitution Principle VI; public contract).
- Any CLI/`cmd` changes, new flags, new dependencies, or output/format contract changes.
- A CLI integration test — unit-level coverage at the resolver is sufficient because `edit`/`rm`/`show`/`done`/`reopen` all funnel through `RequireSingle`.

### Design Decisions

1. **Exact-ID precedence at the resolver only**: Add the precedence pass inside `RequireSingle` over the collected `matches`/`indices`. — *Why*: Fixes all five affected commands at the shared seam without touching search/list semantics. — *Rejected*: A new `--id` selector flag (adds CLI surface); changing `Match` (alters `list`/search public contract).

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add an exact-ID precedence pass in `RequireSingle` (`src/internal/idea/idea.go`): before the `len(matches) > 1` ambiguity branch returns, scan `matches` with `strings.EqualFold(m.ID, query)`; if exactly one is an exact-ID hit, return that idea with its original index from `indices`; otherwise fall through to the existing logic unchanged. <!-- R1 R2 R3 R4 -->

### Phase 3: Integration & Edge Cases

- [x] T002 Add regression test `TestRequireSingle_ExactIDBeatsSubstring` to `src/internal/idea/idea_test.go` following the table-driven `TestRequireSingle_*` convention: GIVEN idea[0] `{ID:"jznd", Text:"the idea to edit"}` and idea[1] `{ID:"qg64", Text:"see related [jznd] for context"}` (both open, both match `"jznd"`), WHEN `RequireSingle("jznd", ideas, FilterAll)`, THEN returns idea[0] at index 0 with no error. <!-- R1 -->

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `RequireSingle` returns the exact-ID owner (idea[0], index 0, no error) when one exact-ID hit coexists with substring-only matches.
- [ ] A-002 R2: Exact-ID equality uses `strings.EqualFold(match.ID, query)` (case-insensitive, no new dependency).
- [ ] A-003 R3: Zero or >1 exact-ID hits fall through to existing logic (single match returns; >1 returns the "Multiple matches" error with no silent pick).
- [ ] A-004 R4: The precedence pass iterates the post-`matchesFilter` match set, so filtered-out exact-ID ideas are never force-selected.

### Behavioral Correctness

- [ ] A-005 R1: The input that previously aborted with "Multiple matches" (`"jznd"` against the cross-reference fixture) now resolves to idea[0]; all other inputs behave identically (`TestRequireSingle_OneMatch`, `_NoMatch`, `_MultipleMatches` still pass).

### Scenario Coverage

- [ ] A-006 R1: `TestRequireSingle_ExactIDBeatsSubstring` exists, is table-driven per the project convention, and passes.

### Edge Cases & Error Handling

- [ ] A-007 R3: The defensive `exactCount > 1` case is not silently resolved — it falls through to the ambiguity error (covered by the unchanged `_MultipleMatches` path and code inspection; IDs are unique per Principle VI so this is defensive).

### Code Quality

- [ ] A-008 Pattern consistency: New code follows the naming and structural patterns of the surrounding `RequireSingle` body and the table-driven test convention.
- [ ] A-009 No unnecessary duplication: Reuses `strings.EqualFold` and the already-collected `matches`/`indices`; no new helper or dependency.
- [ ] A-010 No magic strings/numbers: No new magic strings or numbers introduced.
- [ ] A-011 Logic stays in `internal/idea` (Constitution IV): The fix lives in `internal/idea`, not `cmd/`.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Exact-ID precedence lives only in `RequireSingle`; `Match`/`FindAll` untouched | Pre-agreed (intake option 1); Principle VI makes search/output a public contract | S:95 R:80 A:90 D:95 |
| 2 | Certain | Exact-ID equality is case-insensitive via `strings.EqualFold(m.ID, query)` | Matches existing case-insensitive `Match`; IDs are lowercase by Principle VI; no new dependency | S:90 R:85 A:90 D:90 |
| 3 | Certain | `exactCount > 1` falls through to the existing "Multiple matches" error (no silent pick) | Explicitly specified; Principle VI guarantees unique IDs so this is defensive | S:95 R:90 A:95 D:95 |
| 4 | Certain | Precedence scan iterates `matches` (post-`matchesFilter`), preserving filter semantics | Explicitly specified; iterating `matches` achieves this naturally | S:95 R:85 A:90 D:95 |
| 5 | Certain | Regression test uses `FilterAll` (both ideas open) | Prevailing `TestRequireSingle_*` convention uses `FilterAll`; intake permits either as long as both fixtures pass the filter | S:90 R:80 A:90 D:90 |

5 assumptions (5 certain, 0 confident, 0 tentative).
