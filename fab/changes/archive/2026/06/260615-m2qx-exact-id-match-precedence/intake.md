# Intake: Exact-ID Match Precedence in RequireSingle

**Change**: 260615-m2qx-exact-id-match-precedence
**Created**: 2026-06-15

## Origin

Originated from backlog item `[m2qx]` (a BUG report), captured after the user hit the bug while running `idea edit --main jznd "<replacement>"`.

> BUG: `idea edit <ID> <new-text>` refuses with "Multiple matches" when the EXACT 4-char ID is passed but that ID string also appears as a SUBSTRING inside another idea's text. EXPECTED: an exact ID match should win over incidental substring matches — passing the canonical ID is the documented escape hatch ("use the exact ID") yet here it's exactly what fails. REPRO: backlog has idea [jznd] (the one to edit) and a second idea [qg64] whose body contains the literal text "[jznd]" (a cross-reference). Running `idea edit --main jznd "<replacement>"` matched BOTH — [jznd] by its ID, [qg64] because "jznd" is a substring of its text — and aborted with "ERROR: Multiple matches:" listing both. WORKAROUND USED: edited backlog.md directly with a text editor.

Interaction mode: one-shot synthesis from the backlog report, with the fix approach pre-agreed with the user. The bug report listed three fix options; **option 1 (exact-ID precedence at the resolver layer)** was selected as cleanest because it preserves substring search for non-ID queries and requires no new CLI surface. The user also agreed that **unit-level testing is sufficient** (no separate CLI integration test) because `edit`/`rm`/`show`/`done`/`reopen` all funnel through the single `RequireSingle` resolver.

## Why

1. **Problem**: The shared query resolver `RequireSingle` in `src/internal/idea/idea.go` treats *any* query that produces more than one hit as ambiguous and aborts with `Multiple matches: ... Be more specific or use the exact ID.` It has no notion of precedence between match *kinds*. The matching predicate `Match` ORs an ID case-insensitive-substring check with a case-insensitive text-substring check. So when a query equals one idea's exact 4-char ID but that same string also happens to appear as a substring inside another idea's *text* (e.g. a cross-reference `[jznd]` written into idea `[qg64]`'s body), BOTH ideas match and the command aborts — even though the user passed the canonical, documented, unambiguous ID.

2. **Consequence if unfixed**: The error message instructs the user to "use the exact ID" — which is exactly what they already did. There is no in-CLI escape hatch; the only workaround is hand-editing `fab/backlog.md`, bypassing the tool entirely (which defeats the tool's reason to exist per Constitution Principle I). The bug affects every command that shares the matcher: `edit` / `rm` / `show` / `done` / `reopen`.

3. **Why this approach over alternatives**: The bug report offered three options. Option 2 (a new `--id` selector flag) adds CLI surface and burdens the user with knowing about a special flag. Option 3 (prefer the exact-ID hit only when ambiguity mixes exactly one exact-ID hit with substring-only hits) is effectively a subset of option 1. **Option 1** — exact-ID precedence inside `RequireSingle` only — is the cleanest: it fixes all five affected commands at the shared seam, requires no new flags or dependencies, and leaves `idea list`/search substring semantics untouched.

## What Changes

### Resolver: `RequireSingle` exact-ID precedence (`src/internal/idea/idea.go`)

`RequireSingle` is the only function modified. Today its `len(matches) > 1` branch fires before any check for an exact-ID winner:

```go
func RequireSingle(query string, ideas []Idea, filter FilterKind) (Idea, int, error) {
	var matches []Idea
	var indices []int
	for i, idea := range ideas {
		if !matchesFilter(idea, filter) {
			continue
		}
		if Match(query, idea) {
			matches = append(matches, idea)
			indices = append(indices, i)
		}
	}

	if len(matches) == 0 {
		return Idea{}, -1, fmt.Errorf("No idea matching '%s'", query)
	}
	if len(matches) > 1 {
		// ... aborts with "Multiple matches: ..."
	}
	return matches[0], indices[0], nil
}
```

The fix inserts an **exact-ID precedence pass over the already-collected match set** (`matches`/`indices`) *before* the `len(matches) > 1` ambiguity branch fires. Concretely:

- After collecting `matches`/`indices`, scan that match set for ideas whose `ID` equals the query **case-insensitively** (`strings.EqualFold(idea.ID, query)`).
- **If exactly one match has an exact-ID equality with the query**, select that idea (return it with its original index), bypassing the "Multiple matches" error. Substring-only matches in other ideas are ignored in that case.
- **If zero or more-than-one** of the matches are exact-ID hits, fall through to the existing logic unchanged (zero → existing single-match return at the bottom or the >1 ambiguity error as appropriate; two-plus → the existing ambiguity error). This preserves current behavior for every case except the exact-one-ID-among-many scenario.

Illustrative shape (not prescriptive — apply may structure it differently as long as the semantics hold):

```go
// Exact-ID precedence: if exactly one matched idea's ID equals the
// query (case-insensitive), it wins over incidental substring matches.
if len(matches) > 1 {
	exactIdx := -1
	exactCount := 0
	for j, m := range matches {
		if strings.EqualFold(m.ID, query) {
			exactIdx = j
			exactCount++
		}
	}
	if exactCount == 1 {
		return matches[exactIdx], indices[exactIdx], nil
	}
}
```

### Explicitly NOT changed

- **`Match` and `FindAll` are untouched.** They are the search/list predicates; case-insensitive substring matching is the *desired* behavior there. Changing `Match` would alter `idea list`/search semantics (Constitution Principle VI — output formats and search behavior are part of the public contract). The fix lives strictly at the `RequireSingle` resolver layer.
- No CLI/`cmd` changes. No new flags. No new dependencies. No format/output contract changes.

### Edge cases (both already settled)

1. **Query equals two ideas' IDs** — shouldn't happen, since Constitution Principle VI guarantees IDs are unique within a single backlog file. If it somehow does (`exactCount > 1`), the fix deliberately **falls through to the existing ambiguity error** rather than silently picking one. The precedence applies *only* when there is exactly one exact-ID match among the matches.
2. **Filter scoping** — the exact-ID scan operates over the **already-filtered match set** (`matchesFilter` has already been applied during collection). A filtered-out exact-ID idea (e.g. a done idea under `FilterOpen`) is therefore NOT force-selected — filter semantics are preserved. This falls out naturally because the scan iterates `matches`, not the raw `ideas` slice.

### Regression test (`src/internal/idea/idea_test.go`)

Add a focused / table-driven regression test, e.g. `TestRequireSingle_ExactIDBeatsSubstring`:

- GIVEN two ideas where idea[0] has `ID == "jznd"` and idea[1] has text containing the substring `jznd` (e.g. `"see related [jznd] for context"`) — so both match query `"jznd"` via `Match`.
- WHEN calling `RequireSingle("jznd", ideas, FilterOpen)`.
- THEN it returns idea[0] (the exact-ID owner) at index 0 with **no error** (previously this aborted with "Multiple matches").

Unit-level coverage is sufficient and agreed: all five affected commands (`edit`/`rm`/`show`/`done`/`reopen`) funnel through `RequireSingle`, so a resolver-level test exercises the fix for all of them. Follows the project's table-driven convention (Constitution Principle V) alongside the existing `TestRequireSingle_*` tests.

## Affected Memory

- `cli/structure`: (modify) The CLI memory domain documents the resolver / per-subcommand matcher behavior. Record that `RequireSingle` applies exact-ID precedence (an exact case-insensitive ID match among the candidates wins over incidental substring text matches), while `Match`/`FindAll` keep pure substring semantics for `list`/search. Final file placement (`structure.md` vs. a dedicated note) is a hydrate-time decision.

## Impact

- **Code**: `src/internal/idea/idea.go` — single function `RequireSingle` (one added precedence pass). `src/internal/idea/idea_test.go` — one added regression test.
- **APIs / behavior**: `RequireSingle`'s signature is unchanged. Behavior changes only in the previously-aborting "one exact-ID hit + N substring-only hits" case, which now resolves to the exact-ID idea. All other inputs behave identically.
- **Commands affected (improved, no change needed in their code)**: `idea edit`, `idea rm`, `idea show`, `idea done`, `idea reopen` — all consume `RequireSingle`.
- **Dependencies**: none added. `strings.EqualFold` is already in the standard library and `strings` is already imported.
- **Constitution alignment**: Principle IV (logic stays in `internal/idea`), Principle V (table-driven test, real data), Principle VI (IDs unique → the `exactCount > 1` guard is defensive). No format/output contract drift.

## Open Questions

None. The fix approach, the non-goals (don't touch `Match`/`FindAll`), both edge cases, and the test strategy were all pre-agreed in the synthesized description.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Apply exact-ID precedence in `RequireSingle` only; leave `Match`/`FindAll` untouched | Pre-agreed (option 1); Constitution Principle VI makes search/output semantics a public contract — changing `Match` would alter `idea list`/search | S:95 R:80 A:90 D:95 |
| 2 | Certain | Exact-ID equality is case-insensitive (`strings.EqualFold(idea.ID, query)`) | Matches the existing case-insensitive `Match` semantics and IDs are lowercase by Principle VI; no new dependency | S:90 R:85 A:90 D:90 |
| 3 | Certain | When `exactCount > 1`, fall through to the existing "Multiple matches" error (no silent pick) | Explicitly specified; Principle VI guarantees unique IDs so this is a defensive branch | S:95 R:90 A:95 D:95 |
| 4 | Certain | Scope the exact-ID scan to the already-filtered match set so filter semantics are preserved | Explicitly specified; iterating `matches` (post-`matchesFilter`) achieves this naturally | S:95 R:85 A:90 D:95 |
| 5 | Confident | Unit-level regression test in `idea_test.go` is sufficient; no CLI integration test | Pre-agreed — all five affected commands funnel through `RequireSingle`, so a resolver test covers them all | S:85 R:75 A:85 D:80 |
| 6 | Confident | Affected memory is `cli/structure` (modify); exact file placement deferred to hydrate | Memory index lists the resolver/matcher behavior under the cli domain; placement is reversible at hydrate | S:75 R:80 A:80 D:70 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
