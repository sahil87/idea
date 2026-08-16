# Intake: Promote Idea to Main Backlog

**Change**: 260816-kmti-promote-idea-main-backlog
**Created**: 2026-08-17

## Origin

One-shot invocation from backlog idea `[kmti]` (2026-08-17) via `/fab-new kmti`:

> FEATURE: 'idea promote <query>' — move an idea from the CURRENT worktree's backlog to the MAIN worktree's backlog. MOTIVATION: the tool is worktree-aware by design (Constitution II) because capture happens in linked worktrees, but worktrees are ephemeral — ideas captured there die with worktree cleanup unless manually re-added with --main. Promote closes the loop: resolve <query> against the current backlog (shared ID-or-substring matcher, exact-ID precedence), remove it there, append it to the main backlog preserving id/date/status; refuse with a clear error on ID collision in the destination (do NOT silently re-mint the ID — external references may exist). No-op with an advisory when already in the main worktree (current == main). Both files get one canonical write each (lenient-read/canonical-write contract as usual). Design decisions for whoever picks this up: (a) flag surface — how does promote interact with --main (probably error: promote defines its own source/dest) and --system (consider a --to-system variant later, ship repo->main first); (b) failure atomicity — write destination first, then remove from source, so a crash duplicates rather than loses. Logic in internal/idea, thin cobra wiring per Constitution III/IV; table-driven tests with real temp dirs + real git worktrees per Constitution V.

No prior conversation context — the backlog entry itself carries the design decisions (flag surface, failure atomicity, code placement), which are encoded as assumptions below.

## Why

1. **Pain point**: `idea` is worktree-aware by default (Constitution II) because idea capture happens during exploratory work in linked worktrees. But linked worktrees are ephemeral — `wt` worktrees get cleaned up when a branch merges. An idea captured in a worktree's `fab/backlog.md` dies with the worktree unless the user remembers to manually re-add it with `--main` (retyping the text, losing the original ID and date).
2. **Consequence if unfixed**: silent idea loss on worktree cleanup, or manual copy friction that defeats the tool's purpose (reducing friction over hand-editing markdown).
3. **Why this approach**: a single `promote` verb closes the capture→retain loop using machinery that already exists — the shared query resolver (`RequireSingle`, exact-ID precedence), the lenient-read/canonical-write file contract, and the `MainRepoRoot()` resolution already used by `--main`. Alternatives rejected: (a) auto-promoting on worktree deletion — `idea` has no hook into `wt`/git lifecycle and shouldn't grow one; (b) capturing to main by default — would reverse Constitution II's deliberate default.

## What Changes

### New subcommand: `idea promote <query>`

A new visible subcommand moving one idea from the **current worktree's** backlog (source) to the **main worktree's** backlog (destination):

```
idea promote <query>
```

- `Args: usageArgs(cobra.ExactArgs(1))` — one query argument, wrapped per the tree-wide usage-error convention (an unwrapped validator would regress this command's arg-count errors to exit 1).
- Query resolution via the shared `RequireSingle(query, ideas, FilterAll)` — substring match on ID or text, exact-ID precedence, ambiguity refusal. `FilterAll` because done ideas are promotable too (the contract preserves `status`).
- On match: the idea is **appended** to the destination backlog preserving `id`, `date`, and `status` verbatim, then **removed** from the source backlog. Confirmation to stdout in the established escaped single-line shape: `Promoted: {FormatLine(idea)}`.

### Move semantics and failure atomicity

Write ordering is **destination first, then source** (backlog decision (b)): append to the main backlog and save it; only after that write succeeds, remove from the current backlog and save it. A crash between the two writes leaves the idea duplicated (present in both files), never lost. Each file gets exactly **one canonical write** through the existing `SaveFile`/`render` seam — normal normalize-on-write applies to both files, and date-backfill counts from **both** saves flow up so the `note: stamped today's date on N previously-dateless item(s)` stderr advisory prints per the existing `printBackfillNotice` channel policy.

If the destination file or its directory does not exist, it is created (same posture as `Add`'s append path — `MkdirAll` + create). A missing destination is an empty backlog, not an error.

### ID collision in destination

Before writing, check the destination for an existing idea with the same ID. On collision, **refuse with a clear operational error** (exit 1) naming the ID and the destination file — do NOT silently re-mint the ID, because external references to the ID may exist (e.g. fab change folders embed backlog IDs). Neither file is modified on refusal. Suggested wording shape (final wording at implementation, following the existing what/why/next error style): `ID [kmti] already exists in the main backlog — resolve the collision manually (edit or rm one side), then retry.`

### Already-in-main no-op

When the resolved source and destination paths are the same file (running from the main worktree, or outside any worktree situation that collapses the two), promote is a **no-op with an advisory**: nothing is written, a `note:` advisory goes to **stderr** (established advisory channel), exit 0 (matching the `idea edit` unchanged-buffer no-op precedent). Detection compares the two **resolved absolute backlog paths** (not just worktree roots) so `--file` overrides are covered.

### Flag surface

- **`--main` with promote: usage error (exit 2).** Promote defines its own source (current) and destination (main); `--main` would make the source ambiguous. Same classification as the existing `--system`+`--main` conflict — a malformed invocation.
- **`--system` with promote: usage error (exit 2).** A `--to-system` variant is deliberately deferred (backlog decision (a): ship repo→main first). Rejecting now keeps the surface open for a later `--to-system` without a behavior change.
- **`--file` / `IDEAS_FILE`: honored, applied within each root** — consistent with the documented model where `--main`/`--system` select a *root* and `--file` applies *within* it. Source = current worktree root + file override; destination = main worktree root + the same file override. With no override, both default to `fab/backlog.md` under their respective roots.
- **Outside a git repository: operational error** (`not in a git repository`, exit 1) — main-worktree resolution is git-only, same as `--main` today. No graceful system fallback for promote (that's the deferred `--to-system`).

### Code placement (Constitution III/IV)

- `src/internal/idea/`: new exported `Promote(srcPath, dstPath, query string) (Idea, srcBackfilled, dstBackfilled int, err error)` (exact signature at implementation) owning load-resolve-collision-check-append-save-remove-save. Collision check and the dest-first ordering live here. No stderr output from `internal/idea`.
- `src/cmd/idea/promote.go`: new `promoteCmd()` factory — flag conflict checks (`--main`/`--system` → `usageError`), source/dest path resolution via the existing resolution helpers, orchestration call, stdout confirmation, stderr advisories/backfill notices. Registered in `newRootCmd()`'s `AddCommand` roster.
- Enriched cobra `Long` per the repo-wide convention (what it does, source→dest semantics, collision behavior, an example); `Short` stays a terse one-liner. `help-dump` picks the new node up automatically (no schema change).

### Namespace claim

`promote` as a subcommand name claims the word from the bare-text shorthand: `idea promote the blog post` routes to the subcommand (and errors under `ExactArgs(1)`... actually resolves `the` as query and errors on arg count) instead of capturing an idea beginning with "promote". Same accepted trade as `prune` and `fmt` — the verb is the natural, standard name for this operation and the claim is inherent to shipping it as a subcommand.

### Docs and toolkit-standards surface

- `docs/specs/overview.md` commands table: add the `promote` row (human-curated spec; updated as part of the change since it documents the public command surface).
- `docs/site/skill.md` + committed copy `src/cmd/idea/skill/skill.md`: add promote to the agent usage bundle via `scripts/sync-skill.sh` (drift guard + 150-line budget guard must stay green).
- `README.md` command list: add the promote row.
- Toolkit standards check (constitution § Toolkit Standards): CLI-surface change — conforms to principles №1/№4/№5 as follows: promote is a *move*, not a destructive delete (data preserved in one of the two files at all times), so no `--yes` consent gate and no `--dry-run` are required; exit codes follow the 0/1/2 convention.

## Affected Memory

- `cli/promote`: (new) `idea promote` contract — source/dest resolution, dest-first atomicity, collision refusal, no-op advisory, flag-surface conflicts
- `cli/structure`: (modify) roster addition in `newRootCmd()`, 13th `usageArgs` wrap site, `promote` namespace claim, cross-reference to `cli/promote`

## Impact

- **New files**: `src/cmd/idea/promote.go`, promote logic + tests in `src/internal/idea/` (either in `idea.go` or a new `promote.go` + `promote_test.go`, matching how `fmt.go`/`prune_test.go` split).
- **Modified files**: `src/cmd/idea/main.go` (roster), `src/cmd/idea/main_test.go` (subprocess exit-code matrix rows + routing coverage), `docs/specs/overview.md`, `docs/site/skill.md` + `src/cmd/idea/skill/skill.md` (synced), `README.md`.
- **Tests**: table-driven with real temp dirs and **real git repos + linked worktrees** created in test setup (Constitution V) — promote from linked worktree to main, collision refusal, already-in-main no-op, dest-missing creation, dateless-idea backfill on both sides, done-idea promotion preserving `[x]`, `--main`/`--system` usage errors, outside-git operational error.
- **No new dependencies** — stdlib + existing cobra/x-term only.
- **External contract**: adds one command; no change to line format, JSON schema, help-dump envelope, or existing command behavior (Constitution VI additive-only).

## Open Questions

- None — the backlog entry pre-resolved the two flagged design decisions (flag surface, atomicity), and the remainder follows established repo contracts.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Command surface is `idea promote <query>`, `ExactArgs(1)` wrapped in `usageArgs` | Explicit in backlog; arg-wrap mandated by the tree-wide exit-code convention in cli/structure memory | S:95 R:90 A:95 D:95 |
| 2 | Certain | Query resolves via shared `RequireSingle` with `FilterAll` (done ideas promotable) | Backlog names the shared matcher + exact-ID precedence; "preserving status" implies both states move | S:80 R:85 A:85 D:75 |
| 3 | Certain | `--main` and `--system` with promote are usage errors (exit 2); `--to-system` deferred | Backlog decision (a) — "probably error" + "ship repo->main first"; classification mirrors the existing `--system`+`--main` conflict | S:80 R:85 A:80 D:80 |
| 4 | Confident | `--file`/`IDEAS_FILE` honored, applied within each root (source = current root + override, dest = main root + same override) | Backlog silent; consistent with the documented "selectors pick the root, --file applies within it" model; easily changed | S:40 R:80 A:55 D:40 |
| 5 | Certain | Destination ID collision refuses with operational error (exit 1), no re-mint, neither file modified | Explicit in backlog (external ID references may exist) | S:95 R:90 A:90 D:95 |
| 6 | Certain | Write destination first, then remove from source — crash duplicates, never loses | Explicit backlog decision (b) | S:95 R:85 A:90 D:95 |
| 7 | Certain | Logic in `internal/idea` (`Promote` op), thin cobra wiring in `cmd/idea/promote.go` | Explicit in backlog; Constitution III/IV | S:95 R:90 A:100 D:95 |
| 8 | Confident | current==main detected by comparing resolved absolute backlog paths; no-op prints stderr `note:` advisory, exit 0 | Backlog mandates the no-op+advisory; path-compare (vs root-compare) covers `--file` overrides; exit 0 matches the edit no-op precedent | S:70 R:85 A:75 D:70 |
| 9 | Confident | stdout confirmation `Promoted: {FormatLine}` (escaped single line) | Matches Done:/Removed:/Reopened: confirmation shape; stdout stays machine-parseable (Constitution VI) | S:55 R:90 A:85 D:75 |
| 10 | Confident | No consent flag, no `--dry-run` — promote is a move, not a destructive delete | Toolkit principle №5 targets destructive writes; a move preserves the data in one file at all times; collision/no-match paths refuse before writing | S:60 R:85 A:70 D:65 |
| 11 | Confident | Missing destination file/dir created on demand (Add's append posture) | Fresh main worktree may predate any backlog; mirrors existing Add + lazy-MkdirAll behavior | S:65 R:85 A:80 D:80 |
| 12 | Confident | Backfill from BOTH saves surfaces via the existing stderr notice; each file gets exactly one canonical write | Backlog mandates one canonical write per file; notice channel policy is established (`printBackfillNotice`) | S:55 R:90 A:80 D:75 |
| 13 | Certain | Outside a git repo, promote errors operationally (exit 1) — no system fallback | Main-worktree resolution is git-only today (`--main` errors identically); fallback belongs to the deferred `--to-system` | S:80 R:85 A:85 D:85 |

13 assumptions (7 certain, 6 confident, 0 tentative, 0 unresolved).
