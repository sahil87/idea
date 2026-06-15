# Intake: Fix exact-ID resolution precedence in single-idea subcommands

**Change**: 260615-m2qx-exact-id-resolution-precedence
**Created**: 2026-06-15

## Origin

This change adopts backlog idea **[m2qx]** (2026-06-15). Initiated via `/fab-new m2qx`, a one-shot creation FROM the existing backlog idea. The change ID `m2qx` is carried over from the backlog so the backlog line is auto-marked done on archive.

Backlog origin (verbatim from `fab/backlog.md`):

> [m2qx] 2026-06-15: BUG: `idea edit <ID> <new-text>` refuses with "Multiple matches" when the EXACT 4-char ID is passed but that ID string also appears as a SUBSTRING inside another idea's text. EXPECTED: an exact ID match should win (or at least be selectable) over incidental substring matches — passing the canonical ID is the documented escape hatch ("use the exact ID") yet here it's exactly what fails. REPRO: backlog has idea [jznd] (the one to edit) and a second idea [qg64] whose body contains the literal text "[jznd]" (a cross-reference). Running `idea edit --main jznd "<replacement>"` matched BOTH — [jznd] by its ID, [qg64] because "jznd" is a substring of its text — and aborted with "ERROR: Multiple matches:" listing both, telling me to "use the exact ID" (which I already was). Same root applies to `rm`/`show`/`done`/`reopen`/`edit` — any command using the shared ID-or-substring matcher. ROOT CAUSE (likely in the matcher in internal/idea, the query resolver used by edit/rm/show/etc.): it ORs an ID-equality check with a case-insensitive text-substring check and treats >1 hit as ambiguous, with no precedence between the two match kinds. FIX OPTIONS: (1) exact-ID match takes precedence — if the query equals some idea's ID exactly, select that idea and ignore substring hits (cleanest; preserves substring-search for non-ID queries); (2) if precedence is undesirable, add an explicit selector (e.g. `--id <ID>` to force ID-only matching) so there's a guaranteed-unambiguous path; (3) at minimum, when ambiguity mixes one exact-ID hit with substring-only hits, prefer the exact-ID hit. WORKAROUND USED: edited backlog.md directly with a text editor, bypassing the CLI. Add a regression test: two ideas where one's ID is a substring of the other's text; `edit <ID>` must target only the ID-owner.

The fix approach (FIX OPTION 1 — exact-ID precedence) was already root-caused and confirmed against the source in conversation. The same defect was independently reproduced via `idea show jznd` failing because idea `qg64`'s text contains the literal `[jznd]`.

## Why

1. **What problem does this solve?** The shared single-idea resolver `RequireSingle()` in `src/internal/idea/idea.go` calls `Match()`, which does case-insensitive *substring* matching against BOTH `idea.ID` AND `idea.Text`, with no exact-ID precedence. When a single-idea subcommand is given an exact, unique 4-char ID, the resolver reports "Multiple matches" whenever that ID string also appears as a substring inside another idea's body text (e.g. a cross-reference like `[jznd]`). The idea becomes **unreachable by its own exact ID** — and the error message tells the user to "use the exact ID," which is precisely what already failed. Confirmed root cause (current code, `idea.go:491-537`):

   ```go
   // Match returns true if query is a case-insensitive substring of id or text.
   func Match(query string, idea Idea) bool {
       q := strings.ToLower(query)
       return strings.Contains(strings.ToLower(idea.ID), q) ||
           strings.Contains(strings.ToLower(idea.Text), q)
   }

   func RequireSingle(query string, ideas []Idea, filter FilterKind) (Idea, int, error) {
       var matches []Idea
       var indices []int
       for i, idea := range ideas {
           if !matchesFilter(idea, filter) {
               continue
           }
           if Match(query, idea) {           // <- ORs ID-substring with text-substring
               matches = append(matches, idea)
               indices = append(indices, i)
           }
       }
       if len(matches) == 0 { ... }
       if len(matches) > 1 { ... "Multiple matches" ... }   // <- exact-ID hit gets buried here
       return matches[0], indices[0], nil
   }
   ```

2. **What happens if we don't fix it?** Cross-referencing ideas by ID (a natural, encouraged practice) silently breaks ID-based lookup for **all five** single-idea subcommands: `show`, `done`, `reopen`, `edit`, `rm`. The user cannot reliably address an idea by its primary key. The documented escape hatch ("use the exact ID") is non-functional in exactly the case it exists for. Current workaround is hand-editing `backlog.md`, bypassing the CLI.

3. **Why this approach over alternatives?** FIX OPTION 1 (exact-ID-wins precedence) is the cleanest and the one chosen. It is **sound**: IDs are unique within a backlog file (Constitution Principle VI — "Idea IDs MUST be ... unique within a single backlog file"), so an exact (case-insensitive) ID match can never be genuinely ambiguous and can short-circuit the substring search. It preserves substring search for non-ID queries (no behavior change for `list`-style fuzzy lookups). Option 2 (a new `--id` selector) adds public CLI surface for a problem precedence solves transparently. Option 3 (special-case only the mixed-ambiguity branch) is a strictly weaker subset of Option 1. Fixing the single shared seam (`RequireSingle`) fixes all five subcommands at once.

## What Changes

### 1. `src/internal/idea/idea.go` — exact-ID short-circuit in `RequireSingle`

Add an exact-ID precedence check at the **top** of `RequireSingle(query, ideas, filter)`, before the existing substring-collection loop. Iterate the ideas; if **exactly one** idea passes the active `FilterKind` AND its `ID` equals `query` case-insensitively (`strings.EqualFold(idea.ID, query)`), return that idea and its index immediately. Only when no such exact-ID match is found does control fall through to the existing `Match`-based substring loop (with its unchanged `len == 0` and `len > 1` error branches).

Sketch (final wording at apply time):

```go
func RequireSingle(query string, ideas []Idea, filter FilterKind) (Idea, int, error) {
    // Exact-ID precedence: a case-insensitive exact ID match wins outright.
    // IDs are unique within a backlog (Constitution VI), so this is unambiguous.
    // Respect the active FilterKind so e.g. `done <id>` (FilterOpen) does not
    // resolve a done idea's exact ID.
    exactIdx := -1
    for i, idea := range ideas {
        if !matchesFilter(idea, filter) {
            continue
        }
        if strings.EqualFold(idea.ID, query) {
            if exactIdx != -1 {
                // Defensive: should be impossible given unique IDs; fall through
                // to substring logic rather than guess. (Decision recorded below.)
                exactIdx = -1
                break
            }
            exactIdx = i
        }
    }
    if exactIdx != -1 {
        return ideas[exactIdx], exactIdx, nil
    }

    // Existing substring-collection fallback (unchanged) ...
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
    if len(matches) == 0 { /* unchanged */ }
    if len(matches) > 1 { /* unchanged */ }
    return matches[0], indices[0], nil
}
```

Key properties of the change:
- **FilterKind is respected** by the exact-ID check. `done <id>` uses `FilterOpen`, `reopen <id>` uses `FilterDone`, `show`/`edit`/`rm` use `FilterAll` (per existing call sites at `idea.go:744/760/780/809/862`). A done idea's exact ID MUST NOT be returned by an open-only path.
- **`Match()` is unchanged.** Substring-on-text matching remains available for non-ID queries; the only behavioral change is *which* idea resolves when the query is an exact ID — exact-ID now wins instead of erroring.
- **No `cmd/` changes.** All five subcommands route through `RequireSingle`; the fix is at that single seam (Constitution Principle IV — logic stays in `internal/idea`).

### 2. `src/internal/idea/idea_test.go` — table-driven regression tests

Add table-driven cases (existing style; real temp files per Constitution V) covering at minimum:
- **Primary repro**: two open ideas where idea A's exact ID appears as a substring inside idea B's text (mirroring the real `jznd`/`qg64` + `m2qx` scenario). `RequireSingle("<A-id>", ideas, FilterOpen)` MUST resolve to A — not error with "Multiple matches".
- **Case-insensitivity**: an uppercase/mixed-case query of A's ID still resolves to A via the exact-ID path.
- **FilterKind precedence**: when A's exact ID belongs to a *done* idea and the call uses `FilterOpen`, the exact-ID path does NOT return it (it is filtered out); conversely `FilterDone`/`FilterAll` behave correctly.
- **No regression in substring fallback**: when the query is NOT an exact ID (a genuine text substring), the existing substring behavior still applies — single substring hit resolves; ambiguous substring hits still error with "Multiple matches"; zero hits still error with "No idea matching".

## Affected Memory

- `cli/structure.md`: (modify) documents the backlog line lifecycle and the query/resolution seam. Add or update a resolver query-semantics note recording **exact-ID-wins precedence** in `RequireSingle` (exact case-insensitive ID match short-circuits the substring search and respects the active FilterKind). Hydrate confirms exact placement.

## Impact

- **Code**: `src/internal/idea/idea.go` (`RequireSingle` only — `Match`, `FindAll`, `matchesFilter`, and the `FilterKind` constants are untouched). `src/internal/idea/idea_test.go` (new table-driven cases).
- **Subcommands fixed (all via the one seam)**: `show` (FilterAll), `done` (FilterOpen), `reopen` (FilterDone), `edit` (FilterAll), `rm` (FilterAll).
- **Public contract**: unchanged (Constitution Principle VI). Output formats, JSON schema, and CLI flags are untouched — only *which* idea resolves changes, not output shape. `Match()` / `FindAll()` (used by `list`) keep their current substring semantics.
- **Dependencies**: none. `strings.EqualFold` is stdlib; `strings` is already imported.
- **Docs/specs**: possibly `docs/specs/backlog-format.md` or `docs/specs/overview.md` may gain a one-line note on ID-resolution semantics (exact-ID wins over substring text match for single-idea subcommands). Specs are human-curated; flagged for consideration, not auto-required.

## Open Questions

- Should `docs/specs/overview.md` / `docs/specs/backlog-format.md` formally document the exact-ID-wins resolution rule, or is the memory note (`cli/structure.md`) sufficient? (Specs are human-curated — left to hydrate/author judgment; non-blocking.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Adopt FIX OPTION 1 (exact-ID-wins precedence) over Option 2 (`--id` selector) or Option 3 (mixed-ambiguity special case) | Backlog idea names Option 1 as cleanest; root-caused and confirmed against source in conversation; soundness rests on Constitution VI (IDs unique per backlog). Option 2 adds public CLI surface; Option 3 is a subset of Option 1 | S:95 R:80 A:95 D:90 |
| 2 | Certain | Implement the fix in `RequireSingle` (the single shared seam) in `internal/idea`, no `cmd/` changes | All five subcommands route through `RequireSingle` (verified at idea.go:744/760/780/809/862); Constitution IV mandates logic in `internal/idea` | S:95 R:85 A:95 D:95 |
| 3 | Certain | Exact-ID check uses `strings.EqualFold` (case-insensitive) and is gated by the active `FilterKind` | Backlog/context specify case-insensitive exact match; existing call sites pass FilterOpen/FilterDone/FilterAll and a done idea's ID must not resolve on an open-only path | S:90 R:85 A:95 D:90 |
| 4 | Certain | Leave `Match()` / substring fallback behavior unchanged; only exact-ID precedence is added | Minimal correct fix per backlog "Deferred (NOT in scope)"; preserves `list` fuzzy search and avoids touching the public contract (Constitution VI) | S:95 R:80 A:90 D:90 |
| 5 | Confident | Add table-driven regression tests in `idea_test.go` with real temp files: primary repro, case-insensitivity, FilterKind precedence, substring no-regression | Constitution V mandates table-driven tests with real temp dirs; backlog explicitly requests a regression test for the ID-substring scenario | S:85 R:90 A:90 D:85 |
| 6 | Confident | Affected memory = `cli/structure.md` (modify) to record exact-ID-wins resolver semantics | Memory index lists `cli` as the domain covering CLI structure + backlog line lifecycle + resolution; structure.md is the resolver/query-semantics home | S:80 R:90 A:80 D:80 |
| 7 | Confident | Defensive handling if two ideas somehow share an exact ID under the active filter: fall through to substring logic rather than silently pick the first | Constitution VI guarantees ID uniqueness so this branch is unreachable; "fall through, don't guess" is the one obvious safe default and trivially reversible | S:75 R:85 A:85 D:80 |
| 8 | Confident | Record exact-ID-wins semantics in the `cli/structure.md` memory note; treat a spec note in `overview.md`/`backlog-format.md` as optional author discretion at hydrate | Memory is the documented home for resolution semantics; specs are human-curated and non-blocking, with the memory note as the clear front-runner path | S:70 R:85 A:80 D:75 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
