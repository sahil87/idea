---
description: "Bulk-remove subcommand (`idea prune`): dry-run-by-default/`--force` contract, stdout/stderr channel split, exit codes, the deliberate non-archival design, and the `removeIdeaAt` seam shared with `Rm`"
---

# `idea prune` Subcommand

`idea prune` bulk-removes all done (`[x]`) ideas from the backlog in one pass. The cobra wrapper lives at `src/cmd/idea/prune.go` (`pruneCmd()` factory, modeled on `rmCmd()`); the behavior lives in `Prune` in `src/internal/idea/idea.go`. The split follows Constitution Principle IV — the `RunE` body contains only `resolveFile()` + `idea.Prune` orchestration and output formatting. Added by `260612-drc1-add-prune-subcommand`.

It fills the bulk-removal gap: `rm` is deliberately single-item (`cobra.ExactArgs(1)` + `RequireSingle`'s single-match-or-refuse contract), and `list` merely hides done items by default — neither clears accumulated done lines.

## Dry-run-by-default / `--force` contract

The bare invocation is a **free dry run** — it mirrors `rm`'s `--force` safety convention but makes the refusal a useful preview that succeeds. The only local flag is `--force`; the command takes no positional args (`cobra.NoArgs`), inherits the persistent `--file`/`--main` from root (Constitution III), and defines no alias.

| Invocation | File state | stdout | stderr | Exit |
|------------|-----------|--------|--------|------|
| `idea prune` (done items exist) | **untouched** | one line per done idea via `FormatLine`, in file order | `Re-run with --force to confirm.` | 0 |
| `idea prune --force` (done items exist) | all `[x]` idea lines removed; survivors canonically rewritten via `SaveFile` | `Pruned N done idea(s).` — count only, no per-line listing | backfill notice when count > 0 | 0 |
| either form (no done items) | untouched — no save | `No done ideas to prune.` | — | 0 |

Per-line listing is **reserved for the dry run**; `--force` output stays quiet for scripting (a user-decided DX trade-off from the intake discussion). After a `--force` run the wrapper calls `printBackfillNotice(cmd, backfilled)` before printing the count, so any `note: stamped today's date on N previously-dateless item(s)` advisory goes to stderr exactly as in `done`/`rm`/`edit`.

## Output channels

stdout carries only the machine-readable result (Constitution VI): the dry run's stdout is exactly the removable lines — pipe-friendly, e.g. `idea prune | wc -l`. The confirm hint is advisory, so it goes to **stderr** via `cmd.ErrOrStderr()`. This is the second deliberate stderr routing in the CLI, after the backfill notice (see `structure.md` § Backfill stderr notice).

## Exit codes

Every designed path exits 0 — including the dry run with items present (a successful preview, unlike `rm` without `--force`, which returns the error `Use --force to confirm deletion` and exits 1). The only error path is a missing backlog file, which errors naturally via `LoadFile`'s `os.ReadFile` error in both modes — matching every other mutating command (`done`/`reopen`/`edit`/`rm`); only `list` special-cases a missing file. No prune-specific special-casing exists.

## Deliberate non-archival design

There is no archive file and no undo mechanism. A `fab/backlog-archive.md` was explicitly rejected in the intake: it would introduce a second file into the deliberately one-file contract (Constitution I, `docs/specs/backlog-format.md`), and every external consumer of the backlog format would have to learn it. The backlog is committed, so **git history is the recovery path**. The command's `Long` states this so users see it in `-h` output.

## Zero-done no-op (no save → no surprise normalization)

Zero done items is not an error: `Prune` returns an empty slice and nil error in both modes, and with `force` it **skips `SaveFile` entirely**. This matters because `SaveFile` regenerates every recognized idea line — a no-op mutation would otherwise normalize the whole file (variant bullets, indentation, date backfill) as a surprise side effect of a command that "did nothing". The guard is the early return `if !force || len(removed) == 0`, which also makes the dry run's no-write guarantee structural (the save call is simply unreachable). Tests prove no-save with byte-identical file content on normalize-bait input (variant bullet + dateless line).

## Internal seam

```go
// src/internal/idea/idea.go (after Rm)
func Prune(path string, force bool) ([]Idea, int, error)
```

Returns the removed (or would-be-removed) done ideas in file order, the `SaveFile` date-backfill count (always 0 when no save ran), and an error. Same `(result, backfillCount, error)` convention as `Done`/`Reopen`/`Edit`/`Rm`.

- **Collection**: `removed := FindAll("", f.ideas, FilterDone)` — the empty query matches everything, so the filter alone selects. `FindAll` copies the matching `Idea` values, so the returned slice is unaffected by the splices that follow.
- **Removal**: with `force` and matches, a **backwards walk** (`idx` from `len(f.ideas)-1` down to 0) calls `removeIdeaAt(f, idx)` for each `Done` idea, so pending removals never shift the indices still to be visited. Then `SaveFile` runs, inheriting the existing save semantics: canonical rewrite of survivors, date backfill (count returned), atomic temp-file-plus-rename write, non-idea lines verbatim (Constitution I).
- **`removeIdeaAt(f *File, idx int)`** is the private helper that removes one idea from the `File`'s bookkeeping — its physical `lines` entry (via `ideaIndices[idx]`), its `ideas` entry, and its `ideaIndices` entry — then decrements the line indices of every idea after the removed line. It is the **single home of the `File` index invariant**, shared by `Rm` and `Prune`; it was extracted from `Rm`'s inline splice bookkeeping during this change's review rework so the invariant is maintained in exactly one place.

## v1 scope

v1 prunes **all** done items. `prune --before YYYY-MM-DD` (prune only old done items) was explicitly deferred — the verb accommodates it later without breaking the contract.

## Tests

- `src/internal/idea/prune_test.go` — `TestPrune`, table-driven against `t.TempDir()` (Constitution V): mixed-file force removes only `[x]` lines; dry run returns done ideas with the file byte-identical; all-open no-op in both modes (byte-identical proves no save); non-idea lines verbatim through force; dateless surviving open item backfilled on the force save with the count reflected. `TestPrune_MissingFile` pins the error path in both modes.
- `src/cmd/idea/main_test.go` — `TestPrune_CLIOutputContract`, subprocess table asserting the exact stdout/stderr split (dry-run listing + stderr hint, force count-only, empty-case message), exit 0 on every path, and resulting backlog content, via the existing `buildBinary`/`setupGitRepo`/`writeRepoBacklog`/`runSplit`/`readRepoBacklog` helpers.

## Cross-references

- Source-tree placement, root command factory registration, display-semantics table, and the bare-text namespace claim of the `prune` verb: `structure.md`.
- Backlog line format contract the prune rewrite preserves: `../../specs/backlog-format.md`; command table: `../../specs/overview.md`.
- Constitution Principles I (one-file plain-text backlog), III (cobra factories, persistent flags on root), IV (logic in `internal/idea`), VI (machine-parseable stdout): `fab/project/constitution.md`.
