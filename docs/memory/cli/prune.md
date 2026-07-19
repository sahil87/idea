---
description: "Bulk-remove subcommand (`idea prune`): the TTY × consent (--yes/-y or --force) decision matrix (pipe dry-run vs. interactive [y/N] confirm), the leading stderr count header, TTY-aware truncation/color/--full listing, stdout/stderr channel split, exit codes, the deliberate non-archival design, why no --dry-run alias, and the removeIdeaAt seam shared with Rm"
type: memory
---

# `idea prune` Subcommand

`idea prune` bulk-removes all done (`[x]`) ideas from the backlog in one pass. The cobra wrapper lives at `src/cmd/idea/prune.go` (`pruneCmd()` factory, modeled on `rmCmd()`); the behavior lives in `Prune` in `src/internal/idea/idea.go`. The split follows Constitution Principle IV — the `RunE` body contains only `resolveFile()` + `idea.Prune` orchestration and output formatting. (260612-drc1-add-prune-subcommand)

It fills the bulk-removal gap: `rm` is deliberately single-item (`cobra.ExactArgs(1)` + `RequireSingle`'s single-match-or-refuse contract), and `list` merely hides done items by default — neither clears accumulated done lines.

## Consent flags (`--yes`/`-y` or `--force`)

The three local flags are `--yes`/`-y`, `--force`, and `--full`; the command takes no positional args (`cobra.NoArgs`), inherits the persistent `--file`/`--main`/`--system` from root (Constitution III), and defines no alias. `--yes`/`-y` and `--force` are **equivalent, additive consent aliases** — toolkit principle №1 names `--yes`/`-y` as the canonical flag-satisfiable consent flag, and `--force` is retained alongside it (Constitution VI: the public CLI surface is a growing contract, no renames/removals) (260717-9uh7). The `RunE` computes `consent := force || yes` and passes it into `idea.Prune(path, consent)`; the internal `Prune` takes a single `force bool` meaning "consent given by either flag". Both the immediate-delete branch and the count-only output key on `consent`, not on `force` alone. The `-y` shorthand is collision-free (`-h`/`-v`/`-f`/`-m`/`-s`/`-a` were the pre-existing shorthands).

## TTY-aware decision matrix (consent × TTY)

Behavior is gated on **stdout being a TTY** and consent — the bare invocation is a **free dry run** on a pipe and an **interactive confirm** on a terminal (260613-kfcl), so the un-buryable `[y/N]` prompt is the last line at the cursor instead of a hint that scrolls off-screen.

| stdout TTY? | consent (`--yes`/`-y`/`--force`)? | File state | stdout | stderr | Exit |
|---|---|---|---|---|---|
| any | Yes | all `[x]` lines removed; survivors canonically rewritten via `SaveFile` | `Pruned N done idea(s).` — count only | backfill notice when count > 0 | 0 |
| No | No | **untouched** (dry run) | one line per done idea via `FormatLine`, file order | `N done idea(s) would be pruned` header, then `Re-run with --yes (or --force) to confirm.` trailing hint | 0 |
| Yes | No | removed **iff** the user answers `y`/`yes`; otherwise untouched | the listed (truncated/colored) removable lines, then `Pruned N done idea(s).` on confirm | `N done idea(s) would be pruned` header, the listed lines' channel is stdout, then `Prune N done idea(s)? [y/N] ` prompt; `Aborted — no ideas removed.` on a non-`y` answer | 0 |
| any | (no done items) | untouched — no save | `No done ideas to prune.` | — | 0 |

**Count header (feature B).** Before listing, `idea prune` (without consent) prints `N done idea(s) would be pruned` to **stderr** — the primary signal, printed *before* the list so a human sees the action first regardless of list length. The header carries no call-to-action clause; the action is the interactive prompt (TTY) or the trailing `Re-run with --yes (or --force) to confirm.` hint (non-TTY) — splitting count from action avoids a contradictory re-run-to-confirm line right above a `[y/N]` prompt.

**Interactive confirm (feature E).** On a TTY without consent, after the header + list, `confirmPrune(cmd, n)` writes `Prune N done idea(s)? [y/N] ` to stderr and reads one line from `cmd.InOrStdin()`; deletion proceeds only on `y`/`yes` (case-insensitive, `strings.TrimSpace`'d), and **any other input — including bare Enter and EOF — aborts** with `Aborted — no ideas removed.` and no file change. On confirm the deletion runs through the same consent path (`idea.Prune(path, true)`), so the file outcome is identical to `--yes`/`--force`. The prompt is **never** shown on a non-TTY (it would hang a pipe) — the non-TTY no-consent path falls back to the classic dry run and keeps the trailing `Re-run with --yes (or --force) to confirm.` hint. In TTY mode the prompt **replaces** that trailing hint.

**TTY-aware listing.** The removable-line listing goes through the shared `printIdeaLines` render path (`cmd/idea/output.go`; see `structure.md` and `list.md`): truncated to width and colored on a TTY (unless `--full`), full canonical `FormatLine` when piped — so the dry run stays pipe-friendly (e.g. `idea prune | wc -l`) and a consented (`--yes`/`--force`) run's count-only output is unchanged. `--full` (boolean) disables truncation on a TTY, mirroring `list` for symmetry.

Per-line listing is **reserved for the non-consent paths**; consented output stays quiet for scripting (a user-decided DX trade-off from the intake discussion). After a consented run — and after a confirmed interactive delete — the wrapper calls `printBackfillNotice(cmd, backfilled)` before printing the count, so any `note: stamped today's date on N previously-dateless item(s)` advisory goes to stderr exactly as in `done`/`rm`/`edit`.

## Why no `--dry-run` alias

`prune` deliberately has **no** explicit `--dry-run` flag, even though toolkit principle №5 requires destructive writes to support an accurate preview (audit judgment: 260717-9uh7). Its bare (non-consent) invocation is already a de-facto dry run — a free piped dry run + an interactive pre-confirm listing, both routed through the live `Prune` collection path — so the preview obligation is already satisfied; a redundant `--dry-run` verb would duplicate existing semantics without new capability. Recorded as an audit judgment (see the conformance report), not a gap. Contrast `rm`, whose bare invocation refuses rather than previews, so it carries an explicit `rm --dry-run` via the `RmPreview` seam (see `structure.md` § Consent & dry-run on destructive writes).

## Output channels

stdout carries only the machine-readable result (Constitution VI): the dry run's / pre-confirm's stdout is exactly the removable lines — pipe-friendly, e.g. `idea prune | wc -l`. Everything advisory — the count header, the `[y/N]` prompt, the abort message, the trailing force hint, and the backfill notice — goes to **stderr** via `cmd.ErrOrStderr()`, so `2>/dev/null` suppresses all of it while stdout stays exactly the removable lines. This follows the deliberate advisory-to-stderr channel policy (see `structure.md` § Backfill stderr notice). (The count messages `No done ideas to prune.` / `Pruned N done idea(s).` use `fmt.Println`/`fmt.Printf` to the process stdout.)

## Exit codes

Every designed path exits 0 — including the dry run with items present (a successful preview, unlike `rm` without consent, which returns the error `Use --yes (or --force) to confirm deletion` and exits 1). The only error path is a missing backlog file, which errors naturally via `LoadFile`'s `os.ReadFile` error in both modes — matching every other mutating command (`done`/`reopen`/`edit`/`rm`); only `list` special-cases a missing file. No prune-specific special-casing exists.

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
- **`removeIdeaAt(f *File, idx int)`** is the private helper that removes one idea from the `File`'s bookkeeping — its physical `lines` entry (via `ideaIndices[idx]`), its `ideas` entry, and its `ideaIndices` entry — then decrements the line indices of every idea after the removed line. It is the **single home of the `File` index invariant**, shared by `Rm` and `Prune`, so the invariant is maintained in exactly one place.

## v1 scope

v1 prunes **all** done items. `prune --before YYYY-MM-DD` (prune only old done items) was explicitly deferred — the verb accommodates it later without breaking the contract.

## Tests

- `src/internal/idea/prune_test.go` — `TestPrune`, table-driven against `t.TempDir()` (Constitution V): mixed-file force removes only `[x]` lines; dry run returns done ideas with the file byte-identical; all-open no-op in both modes (byte-identical proves no save); non-idea lines verbatim through force; dateless surviving open item backfilled on the force save with the count reflected. `TestPrune_MissingFile` pins the error path in both modes.
- `src/cmd/idea/main_test.go` — `TestPrune_CLIOutputContract`, subprocess table asserting the exact stdout/stderr split (dry-run listing + stderr hint, force count-only, empty-case message), exit 0 on every path, and resulting backlog content, via the existing `buildBinary`/`setupGitRepo`/`writeRepoBacklog`/`runSplit`/`readRepoBacklog` helpers.
- `src/cmd/idea/main_test.go` — `TestPrune_CountHeaderAndDecisionMatrix` (the `N done idea(s) would be pruned` stderr header text + the non-TTY decision-matrix rows: No/No dry-run fallback with the trailing hint, No/Yes immediate delete, asserting piped stdout stays canonical), `TestConfirmPrune` (the in-process `confirmPrune` y/yes/Y/YES/with-spaces confirm vs. n/no/bare-Enter/EOF/garbage abort), and `TestPrune_ConfirmedDeleteAndAbort` (a `y`/`yes` answer deletes exactly like `--force`; an `n`/EOF answer leaves the backlog byte-identical). (260613-kfcl)

## Cross-references

- Source-tree placement, root command factory registration, display-semantics table, the `term.go` TTY/width/color/truncation seam, the shared `printIdeaLines` render path, and the bare-text namespace claim of the `prune` verb: `structure.md`.
- The same TTY-aware truncation/`--full`/color rendering as it applies to `idea list`/`ls`: `list.md`.
- Backlog line format contract the prune rewrite preserves: `../../specs/backlog-format.md`; command table: `../../specs/overview.md`.
- Constitution Principles I (one-file plain-text backlog), III (cobra factories, persistent flags on root), IV (logic in `internal/idea`), VI (machine-parseable stdout): `fab/project/constitution.md`.
