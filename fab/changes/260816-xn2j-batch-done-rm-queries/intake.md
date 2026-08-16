# Intake: Batch queries on `idea done` and `idea rm`

**Change**: 260816-xn2j-batch-done-rm-queries
**Created**: 2026-08-17

## Origin

Backlog idea `[xn2j]` (2026-08-17), one-shot via `/fab-new xn2j`:

> FEATURE: batch queries on 'idea done' and 'idea rm' — accept multiple query args, e.g. 'idea done a7k2 x9m1 auth-cleanup'. MOTIVATION: triage sessions mean running the command N times today; 'idea list [id...]' already accepts multiple positionals, so multi-arg is precedented. SEMANTICS: all-or-nothing — resolve EVERY query first (shared matcher, exact-ID precedence; any no-match or ambiguous query aborts the whole invocation with the usual error listing, exit 1, backlog untouched), then mutate once so the file gets a single canonical write and the date-backfill advisory prints at most once. Dedupe: two queries resolving to the same idea act on it once (decide: silent or advisory). For rm, --yes/-y/--force covers the whole batch and --dry-run previews all matches; keep per-item output lines ('Done: [id] text' / 'Removed: [id] text') so stdout stays scriptable. Leave reopen single-query unless it falls out for free from a shared resolveMany helper in internal/idea. Constitution IV: the multi-resolve loop lives in internal/idea, cmd/ stays thin. Table-driven tests incl. the mixed-valid-and-ambiguous batch aborting untouched.

No prior conversation context — the backlog entry is the full design input, and it is detailed enough to pre-answer most decision points (semantics, error contract, output contract, code placement, test expectations).

## Why

1. **Pain point**: triage sessions produce a batch of ideas to close or delete at once. Today `idea done` and `idea rm` take exactly one `<query>` (`cobra.ExactArgs(1)`), so marking N ideas done means N invocations — N process spawns, N file writes, N chances to fat-finger, and up to N repeated date-backfill advisories.
2. **Consequence of not fixing**: the CLI stays awkward for its highest-frequency real-world workflow (post-triage cleanup), and users route around it by hand-editing the backlog — the exact friction the tool exists to remove (Constitution I rationale).
3. **Why this approach**: multi-positional args are already precedented in this CLI (`idea list [id...]` accepts multiple IDs), so the surface change is idiomatic, not novel. All-or-nothing resolve-then-mutate gives one canonical write (one atomic save, one normalize-on-write, one backfill advisory) and makes failures safe: a typo in one query leaves the backlog untouched instead of half-mutated. Alternatives rejected: best-effort per-query processing (partial mutation on failure — surprising and hard to script against) and a separate `done-many` verb (claims bare-text namespace, duplicates surface).

## What Changes

### CLI surface (`src/cmd/idea/done.go`, `src/cmd/idea/rm.go`)

- `idea done <query>...` — `Use: "done <query>..."`, `Args: usageArgs(cobra.MinimumNArgs(1))` (replacing `ExactArgs(1)`; stays wrapped in `usageArgs` so zero args exits 2 per the usage-error convention).
- `idea rm <query>...` — same arg change. `--yes`/`-y`/`--force` consent covers the whole batch (one flag, N deletions). `--dry-run` previews **all** matches (one `idea.FormatLine` line per would-be-removed idea) and still writes nothing and wins over consent flags.
- `idea reopen` is **unchanged** in this change (stays `ExactArgs(1)`) — see Assumptions #6.
- Single-arg invocations behave byte-identically to today — same stdout confirmations, same errors, same exit codes (Constitution VI: the existing surface is a frozen contract; batch is purely additive).
- `Long` help text for both commands gains a sentence describing multi-query behavior (all-or-nothing, dedupe) and a batch example (e.g. `idea done a7k2 x9m1 auth-cleanup`). This flows into `help-dump` JSON automatically (no schema change) and must be checked against the toolkit standards governing CLI surface changes (Constitution § Toolkit Standards — run `shll standards` and read the relevant entries during apply).

### Resolution + mutation semantics (`src/internal/idea/idea.go`)

- **New shared helper** `resolveMany(f *File, queries []string, filter FilterKind) ([]int, error)` (name indicative; exact signature decided at plan): resolves every query with `RequireSingle` semantics against the already-loaded file — same case-insensitive substring matching, same exact-ID-beats-substring precedence, same filter (`FilterOpen` for done, `FilterAll` for rm), evaluated **per query independently**. Any no-match or ambiguous query returns that query's usual error (existing wording from `RequireSingle`, including the `Multiple matches:` listing) and the whole invocation aborts with exit 1, backlog untouched. The multi-resolve loop lives in `internal/idea`, not `cmd/` (Constitution IV).
- **Dedupe**: two queries resolving to the same idea (e.g. its ID and a substring of its text) act on it once. Deduping is keyed on the resolved idea index, preserves first-occurrence order, and emits an **advisory note on stderr** (exact wording decided at plan, e.g. `note: queries 'a7k2' and 'auth' matched the same idea; acted once`) — consistent with the established advisory-notes-to-stderr channel policy (backfill notice, prune hints). stdout stays machine-parseable per-item lines only.
- **`Done`**: signature becomes multi-query (e.g. `Done(path string, queries []string) ([]Idea, int, error)`) — load once, resolve all via the shared helper (`FilterOpen`), flip `Done = true` on each resolved index, **one** `SaveFile`. Returns acted ideas in output order plus the single backfill count.
- **`Rm`**: consent check first (unchanged refusal wording), then load once, resolve all (`FilterAll`), dedupe, remove via the existing `removeIdeaAt` seam — removing in **descending idea-index order** so earlier removals don't invalidate later indices — then one `SaveFile`.
- **`RmPreview`**: becomes multi-query, sharing the exact live match path (`LoadFile` + the shared resolver over `FilterAll`), returning all would-be-removed ideas without writing — preserving the preview-cannot-drift property the toolkit standard requires. A failing query aborts the preview with the identical error the live delete would give.

### Output contract (stdout/stderr)

- Per-item stdout lines, one per acted idea: `Done: {FormatLine}` / `Removed: {FormatLine}` — same format as today, repeated N times, so stdout stays line-per-record scriptable (Constitution VI).
- Item order: argument order (first occurrence for deduped queries) — the command echoes the batch as the user stated it.
- The date-backfill advisory (`note: stamped today's date on N previously-dateless item(s)`) prints **at most once** per invocation (single write ⇒ single count), via the existing `printBackfillNotice`.
- Exit codes unchanged in kind: 0 success, 1 operational (no-match / ambiguous / consent refusal), 2 usage (zero args, bad flags).

### Tests

- Table-driven tests in `src/internal/idea/idea_test.go` for the multi-resolve semantics: happy batch, mixed valid-and-ambiguous batch aborting with backlog byte-untouched, mixed valid-and-no-match batch aborting, dedupe (ID + substring of same idea), filter interaction (done-filter excludes already-done ideas from `done` batch resolution), descending-order removal correctness for `rm` with multiple indices.
- CLI-level coverage in `src/cmd/idea/main_test.go`: multi-arg `done`/`rm` happy path (per-item stdout lines), `rm --dry-run` with multiple queries previewing all and writing nothing, batch consent refusal, single-arg invocations byte-identical to current behavior, zero-arg exit 2.
- Real temp dirs, no mocks (Constitution V).

## Affected Memory

- `cli/structure.md`: (modify) — extend § Query resolution with the shared multi-resolve contract (per-query `RequireSingle` semantics, all-or-nothing, dedupe advisory) and § Consent & dry-run with batch `rm`/`--dry-run` semantics; update the done/rm per-subcommand notes.

## Impact

- **Code**: `src/cmd/idea/done.go`, `src/cmd/idea/rm.go` (Use/Args/Long/RunE), `src/internal/idea/idea.go` (shared resolver + `Done`/`Rm`/`RmPreview` signatures). `reopen.go`, `prune.go`, `list.go` untouched.
- **Tests**: `src/internal/idea/idea_test.go`, `src/cmd/idea/main_test.go`.
- **Docs/contract**: `Long` text changes flow into `help-dump` JSON (schema unchanged — envelope/node shape untouched). `docs/specs/overview.md` documents query semantics for external consumers — flag for human spec update if its wording pins single-query grammar. Toolkit standards check required before changing CLI surface (constitution § Toolkit Standards).
- **No new dependencies** (Dependency Discipline). No changes to line format, IDs, or JSON schemas.

## Open Questions

None — the backlog entry pre-resolved the semantic decisions (all-or-nothing, exact-ID precedence, batch consent, per-item output); the remaining choices were gradeable Confident or better (see Assumptions).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Multi-arg via `usageArgs(cobra.MinimumNArgs(1))`, `Use: "done <query>..."` / `"rm <query>..."`; single-arg behavior byte-identical | Direct consequence of the request + Constitution VI frozen surface; `list [id...]` precedent | S:85 R:80 A:90 D:90 |
| 2 | Certain | All-or-nothing: resolve every query first (per-query `RequireSingle` semantics incl. exact-ID precedence), any failure aborts pre-mutation with exit 1 and a single `SaveFile` on success | Explicitly specified in the backlog entry | S:90 R:75 A:90 D:90 |
| 3 | Confident | Modify existing `Done`/`Rm`/`RmPreview` signatures to accept multiple queries rather than adding parallel `*Many` functions | `internal/` package with a single `cmd/` consumer — no external Go API contract; parallel variants would duplicate utilities (code-quality anti-pattern) | S:60 R:75 A:85 D:70 |
| 4 | Confident | Dedupe is **advisory, not silent**: duplicate queries act once + a stderr note | Backlog left this open; advisory-notes-to-stderr is the established channel policy (backfill notice, prune hints) and silent dedupe hides a probable user mistake | S:55 R:90 A:75 D:60 |
| 5 | Confident | Abort surfaces the **first failing query's** usual error (existing no-match / `Multiple matches:` wording), not an aggregate of all failures | "aborts … with the usual error listing" reads as the existing error; aggregation adds new error-format surface for marginal value | S:65 R:85 A:70 D:60 |
| 6 | Confident | `reopen` stays single-query in this change | Backlog's own tie-breaker ("unless it falls out for free"); the resolver is shared but batch reopen still adds CLI surface, help text, and test matrix — not free. Trivial follow-up if wanted | S:70 R:85 A:75 D:65 |
| 7 | Certain | `rm --dry-run` previews **all** matches via the shared live match path, still writes nothing, still wins over consent; a failing query aborts the preview identically to the live delete | Explicit in backlog + the toolkit preview-cannot-drift standard already governing `RmPreview` | S:80 R:80 A:85 D:85 |
| 8 | Confident | Per-item stdout lines in argument order (first occurrence for dupes); backfill advisory at most once after the single write | Argument order matches the user's stated batch; single-advisory is explicit in the backlog | S:55 R:90 A:70 D:65 |

8 assumptions (3 certain, 5 confident, 0 tentative, 0 unresolved).
