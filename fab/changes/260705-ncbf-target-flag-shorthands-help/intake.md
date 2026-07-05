# Intake: Target Flag Shorthands + Targets-First Root Help

**Change**: 260705-ncbf-target-flag-shorthands-help
**Created**: 2026-07-05

## Origin

Dispatched promptless via `/fab-proceed` from a live conversation (synthesized description; no direct user Q&A in this intake session).

> Add single-letter shorthand aliases for the root persistent target flags of the `idea` CLI and make the three backlog-targeting modes prominent in the root help text.
>
> Decisions made in conversation: add `-m` for `--main` and `-s` for `--system` via `BoolVarP` on the root command's persistent flags; restructure the root `Long` help around a Targets section (sketch below); state `--main`/`--system` mutual exclusivity in that section. Verified no shorthand conflicts. Open point: also adding `-f` for `--file` was leaned toward but not user-confirmed — deferred as an assumption.

## Why

1. **Pain point**: The three backlog-targeting modes — current worktree (default), main worktree (`--main`), and system-level (`--system`) — are the CLI's core mental model, yet the root help barely surfaces them. The current `Long` (src/cmd/idea/main.go:31-33) mentions only the bare-text shorthand; `--system` appears nowhere outside the auto-generated flags block. And the target flags are typed constantly (`idea --main add ...`) with no short forms, unlike `list`'s `-a`.
2. **Consequence if unfixed**: Users discover `--system` late or not at all (it is the newest selector), and the most-typed flags stay the most verbose. The shll.ai command reference — which renders the root help verbatim via `help-dump` — inherits the same thin framing.
3. **Why this approach**: Additive shorthands (`BoolVarP`) are fully backwards-compatible — long forms keep working, no behavior changes. A "Targets:" section in `Long` makes the three modes the first thing read at `idea -h`, and becomes the canonical statement of the `--main`/`--system` mutual exclusion (currently documented only in the README; enforced at runtime in `ResolveBacklogPath`).

## What Changes

### 1. Shorthand registration on root persistent flags (`src/cmd/idea/main.go`)

Currently (main.go:52-54) the three persistent flags are registered with `StringVar`/`BoolVar` — no shorthands:

```go
root.PersistentFlags().StringVar(&fileFlag, "file", "", "Override backlog file path (relative to the git root, or to ~/.config/idea when outside a repo)")
root.PersistentFlags().BoolVar(&mainFlag, "main", false, "Operate on the main worktree's backlog instead of the current worktree")
root.PersistentFlags().BoolVar(&systemFlag, "system", false, "Operate on the system-level backlog (~/.config/idea/backlog.md) instead of a repo backlog")
```

Switch `--main` and `--system` to the `P` variants with shorthands, keeping the usage strings unchanged:

```go
root.PersistentFlags().BoolVarP(&mainFlag, "main", "m", false, "Operate on the main worktree's backlog instead of the current worktree")
root.PersistentFlags().BoolVarP(&systemFlag, "system", "s", false, "Operate on the system-level backlog (~/.config/idea/backlog.md) instead of a repo backlog")
```

**Conflict check (verified against the live tree)**: the only shorthands in use are cobra's default `-h` (help, all commands), `-v` (version, root only — materialized via `InitDefaultVersionFlag`), and the list-local `-a` for `--all` (src/cmd/idea/list.go:106). `-m` and `-s` are free everywhere; as root persistent shorthands they are inherited by every subcommand without collision. (`-f` is also free — see Assumption 7 / Open Questions.)

### 2. Targets-first root `Long` (`src/cmd/idea/main.go`)

Restructure the root command's `Long` around the three targets. Sketch agreed in conversation (exact prose has agent latitude; the three-row Targets block and the mutual-exclusion statement are required):

```
Targets:
  (default)      current worktree's backlog
  -m, --main     main worktree's backlog (shared)
  -s, --system   ~/.config/idea/backlog.md (cross-repo, also the default outside a repo)
```

Requirements for the new `Long`:

- The Targets block above is prominent (leading the body after the one-line description).
- It states that `--main` and `--system` are mutually exclusive — the help becomes the canonical place people read this (README already documents it; runtime enforcement lives in `ResolveBacklogPath`, internal/idea/idea.go:449-450, and is **unchanged** by this change).
- The existing bare-text shorthand line (`Shorthand: "idea <text>" is equivalent to "idea add <text>".`) is retained.
- The system path is rendered as the constant `~/.config/idea/backlog.md` (per memory `cli/structure`: the path is pinned on every platform, `$XDG_CONFIG_HOME` ignored) — do not import the README's stale XDG claim.
- `Short` stays byte-stable (it is the public one-liner used by the `Available Commands` sidebar and help-dump; per the `Short` vs `Long` convention, depth goes in `Long` only).

### 3. README touch-up (`README.md`, cosmetic, in scope because cheap)

Mention the new short forms where the long forms are documented: the feature bullets (lines 12-13), the targeting table (lines 103-106), and the "why the default favors the current worktree" paragraph (line 109). Additive mentions only (e.g., `--main`/`-m`) — no restructuring.

### 4. Tests

- Add table-driven CLI coverage asserting `-m` ≡ `--main` and `-s` ≡ `--system` (e.g., extend `src/cmd/idea/main_test.go` using the existing in-process `newRootCmd()` or subprocess helpers), and that `-s -m` together still yields the `mutually exclusive` error (the existing long-form conflict test is at main_test.go:1089).
- `help_dump_test.go` assertions (`-h, --help` / `-v, --version` presence) are unaffected; the dumped `text` gains the new shorthand rendering automatically.

### Non-changes (explicit)

- No behavior change to path resolution (`ResolveBacklogPath` untouched apart from nothing — precedence, errors, mutual-exclusion enforcement all unchanged).
- No help-dump JSON schema change: the schema is a frozen cross-repo contract; the shorthand/help changes ride inside the existing `text`/`usage` strings, and shll.ai re-renders on its next pull after release — no action needed.
- No cobra `MarkFlagsMutuallyExclusive` addition — enforcement stays colocated with the precedence in `ResolveBacklogPath` (deliberate prior design, memory `cli/structure`).

## Affected Memory

- `cli/structure`: (modify) — the "Three persistent root selectors" table gains the `-m`/`-s` short forms; the root-command-factory and `Short` vs `Long` sections reflect the Targets-first root `Long`.

## Impact

- `src/cmd/idea/main.go` — flag registration (2 lines → `BoolVarP`) + `Long` rewrite. Root command surface only; all subcommands inherit the shorthands automatically.
- `src/cmd/idea/main_test.go` — new equivalence/conflict test cases.
- `README.md` — cosmetic short-form mentions.
- Help-dump JSON content (`idea help-dump` output `text`/`usage` fields) changes automatically on next release; schema untouched; shll.ai pull relationship unaffected.
- Constitution III alignment: persistent flags remain defined on root and inherited; root adds no behavior beyond delegation.

## Open Questions

- Should `-f` be added as a shorthand for `--file` in the same change? It is conflict-free and consistent with `-m`/`-s`; the conversation leaned yes ("cheap") but the user did not explicitly confirm. Deferred — see Assumption 7. <!-- assumed: default-include -f unless clarified otherwise; lowest-risk consistent reading of the conversation -->

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Add `-m` shorthand for `--main` via `BoolVarP` on root persistent flags | Discussed — user decided; conflict check verified `-m` free | S:90 R:70 A:90 D:90 |
| 2 | Certain | Add `-s` shorthand for `--system` via `BoolVarP` on root persistent flags | Discussed — user decided; conflict check verified `-s` free | S:90 R:70 A:90 D:90 |
| 3 | Certain | Restructure root `Long` around a leading Targets block (three-row sketch above); keep the bare-text shorthand line; `Short` unchanged | Discussed — sketch agreed; exact prose is agent latitude; `Short` byte-stability is established convention | S:85 R:90 A:85 D:80 |
| 4 | Certain | Targets section states `--main`/`--system` mutual exclusivity | Discussed — help becomes the canonical statement; already true in code and README | S:85 R:90 A:95 D:90 |
| 5 | Certain | Mutual-exclusion **enforcement** unchanged — stays in `ResolveBacklogPath`, no `MarkFlagsMutuallyExclusive` | Conversation scoped this as help-text-only; memory records the colocated-check design as deliberate | S:75 R:85 A:90 D:85 |
| 6 | Certain | No help-dump schema change / no shll.ai action — new help rides existing `text`/`usage` fields, re-rendered on next release | Stated as a constraint in conversation; matches the frozen-contract memory | S:80 R:90 A:95 D:90 |
| 7 | Unresolved | Also add `-f` shorthand for `--file` in this change | Deferred — promptless dispatch. Assistant leaned yes (cheap, consistent, `-f` verified conflict-free) but user did not explicitly confirm; shipped shorthands are permanent public contract. Default if unclarified: include it | S:40 R:40 A:55 D:60 |
| 8 | Confident | README gets additive short-form mentions at the documented `--main`/`--system` sites (lines 12-13, 103-109) | Conversation marked it "in scope if cheap"; it is cheap and purely additive | S:60 R:85 A:80 D:75 |

8 assumptions (6 certain, 1 confident, 0 tentative, 1 unresolved).
