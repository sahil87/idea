---
description: "Move subcommand (`idea promote <query>`): current-to-main worktree backlog move — fixed source/dest resolution with --file applying within each root, shared RequireSingle/FilterAll query semantics, verbatim id/date/status preservation, destination-first writes (a crash duplicates, never loses), destination ID-collision refusal (never re-minted), same-path no-op advisory, --main/--system usage errors, and the stdout/stderr channel split"
type: memory
---

# `idea promote` Subcommand

`idea promote <query>` moves one idea from the **current worktree's** backlog to the **main worktree's** backlog — the retain step for ideas captured in an ephemeral linked worktree, which would otherwise die with worktree cleanup. The cobra wrapper lives at `src/cmd/idea/promote.go` (`promoteCmd()` factory, `Use: "promote <query>"`, enriched `Long` + terse `Short`); the behavior lives in `Promote` in `src/internal/idea/promote.go` (Constitution III/IV split — the `RunE` holds only flag-conflict checks, path resolution, and output). (260816-kmti)

## Source/destination resolution and flag surface

Promote defines its own source and destination, resolved in the cmd layer via the shared `idea.ResolveBacklogPath(systemFlag, mainFlag bool, fileFlag string)` (see [structure](/cli/structure.md) § Backlog path resolution):

- **Source** — `ResolveBacklogPath(false, false, fileFlag)`: the current worktree root.
- **Destination** — `ResolveBacklogPath(false, true, fileFlag)`: the main worktree root. This leg is git-only, so outside a git repository promote fails operationally (exit 1, `not in a git repository`) — there is no system-backlog fallback.

`--file`/`IDEAS_FILE` applies **within each root**: the same relative override resolves under both roots; an absolute value is honored verbatim on both sides, which collapses source and destination to the same file and takes the no-op path below.

`--main` and `--system` with promote are **usage errors (exit 2)**, returned as `&usageError{...}` from the cmd layer — a malformed invocation, classified like the existing `--system`+`--main` conflict (exit-code policy is a `cmd/` concern, Constitution IV). Rejecting `--system` — rather than interpreting it — keeps the surface open for a deferred `--to-system` variant without a later behavior change.

## Query resolution

`<query>` resolves against the **source** backlog via the shared `RequireSingle(query, f.ideas, FilterAll)` — case-insensitive substring on ID or text, exact-ID precedence over incidental substring matches, ambiguity refusal listing the matches (see [structure](/cli/structure.md) § Query resolution). `FilterAll` because done ideas are promotable — status is part of what is preserved. No-match and ambiguous-match are operational errors (exit 1) with the existing `RequireSingle` wording; neither file is modified.

## Move semantics

`Promote(srcPath, dstPath, query string) (Idea, int, int, error)` owns the whole operation — load, resolve, collision-check, append, save, remove, save:

1. Load the source; resolve the idea via `RequireSingle`.
2. Load the destination — `fs.ErrNotExist` loads as an empty backlog (`&File{}`), not an error; a missing destination file or parent directory is created on write via the `atomicWriteFile` MkdirAll seam (same posture as `Add`'s append path).
3. Collision-check the destination (below).
4. Append `FormatLine(idea)` to the destination `File` (its `lines`, `ideaIndices`, and `ideas` kept in step) and `SaveFile` it.
5. Only after the destination write succeeds: `removeIdeaAt(src, idx)` — the shared File-index-invariant seam (see [prune](/cli/prune.md) § Internal seam) — and `SaveFile` the source.

The move preserves the idea's `id`, `date`, and `status` **verbatim** — a done idea arrives `[x]`. Each file gets exactly **one** canonical write through the `LoadFile`/`SaveFile`(`render`) seam, so normalize-on-write and date backfill apply to both files as usual; a previously-dateless promoted idea gets today's date stamped by the destination save. The two returned ints are the per-file backfill counts (source, destination), and the returned `Idea` is the idea as it landed in the destination (post-backfill). A missing **source** file errors naturally via `LoadFile`, matching the other mutating commands.

## Destination ID-collision refusal

Before anything is written, the destination's parsed ideas are scanned for the same ID. On collision the move is refused with an operational error (exit 1) following the what/why/next style:

```
ID '<id>' already exists in <dstPath> — resolve the collision manually (edit or rm one side), then retry
```

The ID is never silently re-minted — external references (e.g. fab change folders embed backlog IDs) may key on it — and neither file is modified on refusal. The check loop deliberately mirrors `checkIDCollision` over the already-parsed destination rather than calling that helper (it re-loads from a path; `Promote` already holds the parsed `File`), and shares its accepted parsed-ideas-only blind spot: a 4-char bracket inside an unparseable line is invisible.

## Already-in-main no-op

When the two resolved backlog paths are the same file — running from the main worktree, or an absolute `--file` override collapsing both sides — promote is a **no-op**: nothing is written, stderr carries `note: already in the main worktree — nothing to promote`, and the exit code is 0 (the `idea edit` unchanged-buffer precedent — see [edit](/cli/edit.md)). Detection is a plain compare of the two resolved paths in the cmd layer, short-circuiting before `Promote` is called; `Promote` stays a pure two-distinct-paths operation.

## Output contract

On success, stdout carries exactly `Promoted: {FormatLine(idea)}` — the escaped single-line confirmation matching the `Done:`/`Removed:`/`Reopened:` shape (Constitution VI). Advisories stay on stderr: the no-op note, and the date-backfill notice. The cmd layer sums the two per-file backfill counts into a single `printBackfillNotice` call, so the `note: stamped today's date on N previously-dateless item(s)` advisory reports the combined total across both files without saying which file(s) got stamped. `internal/idea` writes nothing to stderr (Constitution IV channel split).

## Namespace claim

`promote` claims the word from the bare-text shorthand: `idea promote the blog post` routes to the subcommand (and errors under its `usageArgs(cobra.ExactArgs(1))`) instead of capturing an idea beginning with "promote" — the same accepted namespace trade as `prune` and `fmt` (see [structure](/cli/structure.md) § Command aliases and the bare-text shorthand).

## Design Decisions

### Destination-first write ordering
**Decision**: Save the destination backlog before removing from the source; the two saves are not atomic as a pair.
**Why**: A crash between the writes leaves the idea duplicated (visible, trivially fixed with `rm`) instead of lost.
**Rejected**: Source-first (a crash silently loses the idea); a two-file transactional scheme (needless complexity for a plain-text tool — Constitution I keeps files independently hand-editable).
*Introduced by*: 260816-kmti-promote-idea-main-backlog

### Whole-file canonical write for the destination append
**Decision**: Append into the destination via `LoadFile` + append to `File.ideas` + `SaveFile`, not via `Add`'s raw byte-append path.
**Why**: The backlog contract asks for one canonical write per file; the `SaveFile`/`render` seam gives status preservation, date backfill counting, and dir creation for free.
**Rejected**: Extending `Add` with a status parameter (churns a stable exported signature for one caller); raw append (skips backfill counting and canonicalization).
*Introduced by*: 260816-kmti-promote-idea-main-backlog

### No-op detection in the cmd layer
**Decision**: `cmd/idea/promote.go` compares the two resolved backlog paths and short-circuits to the advisory no-op before calling `internal/idea`.
**Why**: Path resolution already lives at the cmd seam (`resolveFile` pattern); the advisory is stderr channel policy, which is a `cmd/` concern (Constitution IV). `Promote` stays a pure two-distinct-paths operation.
**Rejected**: Detecting inside `Promote` (would put output-channel policy or a sentinel error in `internal/idea` for no gain).
*Introduced by*: 260816-kmti-promote-idea-main-backlog

## Tests

- `src/internal/idea/promote_test.go` — table-driven against real temp dirs (Constitution V): `TestPromote` (open and done ideas moved with id/date/status preserved; dateless ideas backfilled with per-file counts returned; missing destination file created; substring resolution), `TestPromote_MissingDestinationDir` (missing parent dir created via the `atomicWriteFile` MkdirAll seam), `TestPromote_CollisionRefuses` (error names the ID; both files byte-identical), `TestPromote_QueryRefusals` (no-match and ambiguity refused pre-write; exact-ID precedence over a coincidental substring), `TestPromote_DestinationWriteFailureLeavesSource` (a directory as destination path pins the destination-first ordering — the source stays byte-identical), `TestPromote_MissingSource` (natural `LoadFile` error).
- `src/cmd/idea/main_test.go` — subprocess coverage over a real git repo plus a `git worktree add` linked worktree: `TestPromote_LinkedWorktreeCLI` (happy path by ID and by substring; done idea arrives `[x]`; no-match exit 1; `--main`/`--system` exit 2), `TestPromote_DestinationCollisionCLI`, `TestPromote_AlreadyInMainNoOp` (exit 0, stderr note, file untouched), `TestPromote_OutsideGitFails` (exit 1), `TestPromote_FileFlagAppliesWithinEachRoot` (`--file custom.md` resolves under both roots), plus promote rows in the tree-wide exit-code matrix (usage→2, operational→1, success→0).

## Cross-references

- Source-tree placement, the `newRootCmd()` roster, backlog path resolution precedence, query resolution, the 0/1/2 exit-code convention (promote's `ExactArgs(1)` is the 13th `usageArgs` wrap site), and the bare-text namespace-claim rule: [structure](/cli/structure.md).
- The shared `removeIdeaAt` File-index-invariant seam and the advisory-to-stderr channel policy: [prune](/cli/prune.md).
- The unchanged-buffer no-op precedent (stderr `note:` advisory, exit 0): [edit](/cli/edit.md).
- Public behavior contract (commands table + Promote semantics note): `../../specs/overview.md`; the line-format contract both writes preserve: `../../specs/backlog-format.md`.
- Constitution Principles II (worktree-aware default, git-only main-worktree resolution), III/IV (cobra factory + `internal/idea` split), VI (machine-parseable stdout): `fab/project/constitution.md`.
