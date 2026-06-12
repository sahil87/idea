# Intake: Add `idea prune` Subcommand

**Change**: 260612-drc1-add-prune-subcommand
**Created**: 2026-06-12
**Status**: Draft

## Origin

Synthesized from a completed `/fab-discuss` design session on 2026-06-12. The user asked whether a command exists to remove completed backlog items; a survey confirmed none does (`rm` is single-item with single-match-or-refuse semantics; `list` merely hides done items by default). A DX proposal was made, reviewed interactively, and explicitly approved by the user, who also decided the one open question (output verbosity on `--force`) as: **count only**. All major decisions below were made interactively — SRAD signal strength is high; this intake was generated without further prompting.

> Add an `idea prune` subcommand — bulk-remove all done (`[x]`) ideas from the backlog in one pass. Bare `idea prune` is a free dry run (lists each done idea that would be removed, then a confirm hint); `idea prune --force` performs the deletion and prints a count only.

## Why

1. **Pain point**: Done ideas accumulate in `fab/backlog.md` forever. The only removal path today is `idea rm <query> --force`, which is deliberately single-item — `cobra.ExactArgs(1)` plus `RequireSingle`'s single-match-or-refuse contract. Clearing 20 finished items means 20 separate `rm` invocations, each needing a unique query. `idea list` hiding done items by default only masks the clutter; the file keeps growing.
2. **Consequence of not fixing**: The backlog file bloats with dead lines, normalize-on-write diffs grow noisier, and users fall back to hand-editing the markdown — exactly the friction the tool exists to remove (Constitution rationale under Principle I).
3. **Why this approach**: A new verb, `prune`, with strong CLI precedent for "remove things no longer needed" (`git remote prune`, `docker system prune`). Bolting a `--done` bulk mode onto `rm` was rejected because it muddles `rm`'s safety contract (single-match-or-refuse). A non-destructive archive file (e.g. `fab/backlog-archive.md`) was rejected because it introduces a second file into the deliberately one-file contract (Constitution I, `docs/specs/backlog-format.md`) and every external consumer would need to learn it — git history already provides recovery since the backlog is committed.

## What Changes

### New subcommand: `idea prune`

A new cobra subcommand registered in `newRootCmd()` (`src/cmd/idea/main.go`), implemented as a factory function `pruneCmd()` in a new file `src/cmd/idea/prune.go`, following the existing pattern — `rmCmd()` in `src/cmd/idea/rm.go` is the closest template (including the `--force` flag wiring and the `printBackfillNotice` call).

**Behavior matrix (user-approved DX):**

| Invocation | File state | stdout | Exit |
|------------|-----------|--------|------|
| `idea prune` (done items exist) | **untouched** | One line per done idea via `idea.FormatLine` (same display formatting as `list`/`rm` confirmations), then the confirm hint `Re-run with --force to confirm.` | 0 |
| `idea prune --force` (done items exist) | All `[x]` idea lines removed; survivors canonically rewritten | `Pruned N done idea(s).` — count only, no per-line listing | 0 |
| `idea prune` / `idea prune --force` (no done items) | untouched | `No done ideas to prune.` | 0 |

- The bare invocation is a **free dry run**: it mirrors `rm`'s existing `--force` safety convention but makes the refusal useful as a preview. The user decision: per-line listing is reserved for the dry run, keeping `--force` output quiet for scripting.
- The confirm hint line (`Re-run with --force to confirm.`) is advisory, not part of the machine-readable result, so it goes to **stderr** — keeping the dry run's stdout exactly the removable lines (pipe-friendly, e.g. `idea prune | wc -l`), per Constitution VI and the `printBackfillNotice` stderr precedent (Confident assumption — see Assumptions #15).
- The dry run with items exits **0** — the "free dry run" framing makes it a successful preview, not `rm`'s error-path refusal (which returns an error and exits 1).
- Inherits the persistent flags `--file` / `--main` from root, like every backlog command (Constitution III). No new flags beyond `--force`. No alias (aliases are namespace decisions per the `ls` precedent; none was discussed).
- Carries an enriched cobra `Long` (worktree-vs-`--main` note, `--force` semantics, inline example) per the repo-wide help-text convention; the command appears in `help-dump` output automatically via the factory tree walk — no JSON schema change.

### New internal op: `idea.Prune`

Per Constitution IV (logic in `internal/idea`, thin cmd wrapper):

```go
// src/internal/idea (idea.go or a sibling file)
func Prune(path string, force bool) ([]Idea, int, error)
```

- Returns the removed (or would-be-removed) ideas slice in file order, the backfill count, and an error.
- **`force == false` MUST NOT write the file** — it loads, collects the done ideas, and returns them (backfill count 0, since `SaveFile` never runs).
- **`force == true`** removes every `Done == true` idea from the `File` (entries from `lines`/`ideas`/`ideaIndices`, index-adjusted — same bookkeeping as `Rm`), then calls `SaveFile`. This inherits the existing save semantics: canonical rewrite of surviving idea lines, date backfill on previously-dateless survivors (count returned), atomic temp-file-plus-rename write, and **non-idea lines (headers, blank lines, prose) preserved verbatim** per Constitution I.
- Zero done items is not an error: returns an empty slice, nil error, in both modes; the cmd layer prints `No done ideas to prune.`. With `force` and zero done items, no save occurs (file untouched) — a no-op mutation would otherwise trigger whole-file normalization/backfill as a surprise side effect (Confident assumption — see Assumptions #16).
- A missing backlog file errors naturally via `LoadFile`'s `os.ReadFile` error, matching every other mutating command (`done`/`reopen`/`edit`/`rm`); only `list` special-cases a missing file.
- The cmd wrapper calls `printBackfillNotice(cmd, backfilled)` after a `--force` run, so the advisory `note: stamped today's date on N previously-dateless item(s)` goes to stderr exactly as in `done`/`rm`/`edit`.

### Tests

Table-driven, against real temp dirs (`t.TempDir()`), per Constitution V — in `src/internal/idea` (e.g. extend `idea_test.go` or a `prune_test.go` sibling). Cases to cover (from the discussion):

1. Mixed open/done file: `--force` removes only the `[x]` lines; open lines survive.
2. All-open file: nothing to prune — empty result, file untouched, both modes.
3. Dry run (force=false) on a mixed file: returns the done ideas, file bytes unchanged.
4. `--force` preserves non-idea lines (headers, prose, blank lines) verbatim.
5. Backfill notice path: a previously-dateless **surviving** open item gets today's date stamped on the `--force` save, and the returned backfill count reflects it.

CLI-level coverage (command wiring, output strings, exit codes) MAY extend `src/cmd/idea/main_test.go`'s existing subprocess helpers if needed.

### Out of scope (v1)

- `prune --before YYYY-MM-DD` (prune only old done items) — explicitly deferred; the verb accommodates it later. v1 prunes **all** done items.
- Any archive/undo mechanism — rejected (see Why §3).

## Affected Memory

- `cli/structure`: (modify) — add `prune` to the subcommand roster in the root command factory listing and per-subcommand notes (dry-run-by-default convention, count-only `--force` output).
- `cli/prune`: (new) — per-subcommand note (precedent: `cli/update.md`) covering the dry-run/`--force` contract, output channels, and the deliberate non-archival design.

## Impact

- `src/cmd/idea/prune.go` — new file: `pruneCmd()` factory (`--force` flag, output formatting, `printBackfillNotice` call).
- `src/cmd/idea/main.go` — register `pruneCmd()` in `newRootCmd()`'s `AddCommand` list.
- `src/internal/idea/` — new exported op `Prune(path string, force bool) ([]Idea, int, error)` plus its table-driven tests.
- `help-dump` JSON — gains the `prune` node automatically (frozen schema unchanged; no `aliases`/new fields).
- `docs/specs/overview.md` — human-curated command list may warrant an update post-implementation (spec maintenance is human-owned; noted, not tasked here).
- Dependencies: none added (stdlib + cobra only, per Dependency Discipline).

## Open Questions

None — all design decisions (verb choice, dry-run default, `--force` output verbosity, empty-case behavior, rejection of the archive alternative, v1 scope) were resolved interactively in the /fab-discuss session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New `prune` verb, not a `--done` flag on `rm` | Discussed — user approved; preserves `rm`'s ExactArgs(1) single-match-or-refuse safety contract; CLI precedent (git remote prune, docker system prune) | S:95 R:75 A:90 D:90 |
| 2 | Certain | Bare `idea prune` = free dry run: lists each done idea via `FormatLine`, prints confirm hint, never writes the file | Discussed — user explicitly approved this DX, mirroring `rm`'s --force convention as a useful preview | S:95 R:85 A:90 D:90 |
| 3 | Certain | `--force` stdout is count only: `Pruned N done idea(s).` — no per-line listing | Discussed — user decided the one open question (count only; per-line reserved for dry run, keeping --force quiet for scripting) | S:95 R:90 A:90 D:95 |
| 4 | Certain | No done items: `No done ideas to prune.`, exit 0, both with and without --force | Discussed — specified verbatim in the approved DX | S:95 R:90 A:90 D:90 |
| 5 | Certain | Inherits persistent `--file` / `--main` from root; only new flag is `--force`; no alias | Constitution III (persistent flags on root); alias scope decision per `ls` precedent (memory cli/structure) | S:90 R:90 A:95 D:95 |
| 6 | Certain | Mutation rides the existing LoadFile/SaveFile seam: canonical rewrite of survivors, atomic write, non-idea lines verbatim, date backfill with stderr notice via `printBackfillNotice` | Discussed + Constitution I/IV; the save semantics are inherited, not redesigned | S:90 R:80 A:95 D:90 |
| 7 | Certain | No archive file — git history is the recovery path | Discussed — rejected alternative; a second file breaks the one-file contract (Constitution I, docs/specs/backlog-format.md) and burdens external consumers | S:90 R:70 A:85 D:85 |
| 8 | Certain | v1 prunes all done items; `--before YYYY-MM-DD` deferred as future extension | Discussed — verb accommodates it; explicitly out of v1 scope | S:90 R:85 A:85 D:85 |
| 9 | Certain | API shape: `idea.Prune(path string, force bool) ([]Idea, int, error)` in internal/idea; thin `pruneCmd()` wrapper in cmd/idea/prune.go modeled on `rmCmd()` | Discussed (implementation shape given) + Constitution IV logic/wiring split | S:90 R:75 A:90 D:85 |
| 10 | Certain | Dry run performs no write: force=false returns the would-be-removed slice without calling SaveFile | Discussed — explicit requirement ("must not write the file") | S:95 R:80 A:90 D:90 |
| 11 | Certain | Tests are table-driven against `t.TempDir()`, covering the five discussed cases (mixed file, all-open, dry-run-untouched, verbatim non-idea lines, backfill on dateless survivors) | Constitution V + case list given in discussion | S:90 R:90 A:95 D:90 |
| 12 | Certain | Change type: feat | Declared in the synthesis | S:95 R:95 A:95 D:95 |
| 13 | Certain | New command carries an enriched cobra `Long`; appears in help-dump automatically with no schema change | Repo-wide convention (memory cli/structure: `Short` vs `Long`, frozen help-dump contract) | S:85 R:90 A:95 D:90 |
| 14 | Confident | Dry run with done items exits 0 (successful preview), not `rm`'s non-zero refusal | "Free dry run" framing implies success; exit code not explicitly stated for this row — clear front-runner, trivially reversible | S:55 R:90 A:65 D:65 |
| 15 | Confident | Confirm hint `Re-run with --force to confirm.` goes to stderr; dry-run stdout carries only the removable lines | Synthesis says "prints" without naming a stream; Constitution VI machine-parseable stdout + printBackfillNotice stderr precedent point one way | S:50 R:90 A:80 D:70 |
| 16 | Confident | Missing backlog file errors via LoadFile, matching every other mutating command; zero-done `--force` skips SaveFile (no surprise whole-file normalization) | Not discussed; codebase pattern is unambiguous for mutating commands (`done`/`rm`/`edit`), and the approved DX's "file untouched" empty-case row implies no no-op save | S:40 R:90 A:80 D:75 |

16 assumptions (13 certain, 3 confident, 0 tentative, 0 unresolved).
