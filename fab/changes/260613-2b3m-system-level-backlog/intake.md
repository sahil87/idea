# Intake: System-Level Backlog & Out-of-Git Operation

**Change**: 260613-2b3m-system-level-backlog
**Created**: 2026-06-13

## Origin

> Allow idea to work even out of git folders. Use a system level backlog.md to track ideas at a system level

Conversational mode. The raw input was a two-part request: (1) make `idea` usable outside a git repository, and (2) introduce a system-level backlog for cross-repo idea capture. The interaction surfaced design tension with **Constitution Principle II** (worktree-aware by default, all resolution via `git rev-parse`) — operating outside git requires a sanctioned non-git path. Three decisions were resolved interactively:

- **Out-of-git default behavior** — user chose *graceful fallback to the system backlog* (over CWD-local `./fab/backlog.md` or erroring with a required flag).
- **Explicit `--system` flag** — user chose *yes, add it* as a persistent peer of `--main`, so the system backlog is reachable from inside a repo too (not implicit-only).
- **System file location** — user chose `~/.config/idea/backlog.md`, explicitly referencing "hop style" (`~/.config/hop/hop.yaml`). hop uses the plain `~/.config` location, which *is* the XDG default; we therefore honor `$XDG_CONFIG_HOME` when set and fall back to `~/.config/idea/backlog.md`.

## Why

1. **Problem**: Today every `idea` command shells out to `git rev-parse --show-toplevel` (or `--git-common-dir`) to find a repo root, and joins the backlog path to it. Outside a git repo both helpers return `not in a git repository`, so **every command fails** — `--file` and `IDEAS_FILE` don't rescue this, because their values are still joined to the (failed) git-resolved root. There is also no place to capture an idea that isn't tied to a specific repo (a cross-cutting task, a personal todo, a "remember to look at X" note).
2. **Consequence if unfixed**: `idea` is unusable in `~`, `/tmp`, plain notes folders, or any non-repo directory — exactly where lightweight idea capture is most natural. Cross-repo ideas have nowhere to live but get wedged into whichever repo happened to be the CWD.
3. **Why this approach**: A single, fixed system-level backlog (XDG-located, mirroring hop) is the lowest-friction model: "outside git ⇒ your personal global list" is predictable and never scatters `fab/` folders across arbitrary directories. The explicit `--system` flag makes the same target reachable from inside a repo without forcing a `cd`. This extends the existing flag-driven resolution model (`--main`, `--file`) rather than inventing a new one, and keeps Principle II intact for the in-git case (git resolution is still the default and the only path used when a repo is present and no override is given).

## What Changes

### 1. Path resolution gains a non-git fallback

`resolveFile()` (in `src/cmd/idea/resolve.go`) currently fails hard when `git rev-parse` errors. New resolution precedence:

1. `--system` flag set → **system backlog** (skip git entirely).
2. `--file <path>` set → joined to the resolved root (git root if in a repo; **system config dir** if outside git — so an out-of-git `--file rel/path` resolves under `~/.config/idea/`). <!-- assumed: out-of-git --file roots at the system config dir rather than CWD — keeps a single non-git anchor consistent with the system-backlog model -->
3. `IDEAS_FILE` env set → same rooting rule as `--file`.
4. `--main` set → main worktree root (requires git; unchanged — errors outside git as today). <!-- assumed: --main remains git-only; it is definitionally about worktrees and has no meaning outside a repo -->
5. In a git repo, no override → `{worktree-root}/fab/backlog.md` (**unchanged default**).
6. **Outside a git repo, no override → system backlog** (the new graceful fallback).

### 2. New `--system` persistent flag

Defined on root alongside `--main`:

```go
root.PersistentFlags().BoolVar(&systemFlag, "system", false, "Operate on the system-level backlog (~/.config/idea/backlog.md) instead of a repo backlog")
```

Forces the system backlog from anywhere, including inside a repo:

```
$ cd ~/my-repo && idea --system "global todo"   # -> ~/.config/idea/backlog.md
$ idea --system list                            # lists the system backlog
```

`--system` and `--main` are mutually exclusive (both pick a root; specifying both is a user error → clear error message, non-zero exit). <!-- assumed: --system + --main is a conflict error rather than a silent precedence rule — two explicit root selectors disagreeing should fail loudly -->

### 3. System backlog location helper

A new `internal/idea` function resolves the system backlog path:

```go
// SystemBacklogPath returns the system-level backlog file path:
//   $XDG_CONFIG_HOME/idea/backlog.md  (when XDG_CONFIG_HOME is set)
//   ~/.config/idea/backlog.md         (otherwise)
func SystemBacklogPath() (string, error)
```

Mirrors hop's convention (`~/.config/hop/hop.yaml`). The parent directory (`~/.config/idea/`) is **created on demand** when a mutating command needs to write and the file/dir does not yet exist (consistent with how the tool expects to write a fresh backlog). <!-- assumed: mkdir -p the config dir on write rather than erroring "no such directory" — matches the zero-setup capture goal -->

### 4. Behavior summary

```
$ cd /tmp && idea "buy milk"          # outside git -> ~/.config/idea/backlog.md
$ cd ~/my-repo && idea "fix bug"      # in git      -> ~/my-repo/fab/backlog.md   (unchanged)
$ cd ~/my-repo && idea --system "x"   # in git      -> ~/.config/idea/backlog.md
$ idea --system list                  # anywhere    -> ~/.config/idea/backlog.md
$ idea --main "y"                      # outside git -> ERROR: not in a git repository  (unchanged)
```

The backlog **file format is entirely unchanged** — the system backlog is the same canonical Markdown checklist, just at a different path. All CRUD/`fmt`/`list`/`show` semantics carry over verbatim.

### 5. Docs / help text

Per-command `Long` help and `docs/specs/overview.md` "Worktree Behavior" section gain a "system backlog" note; the resolution-precedence change is documented in `overview.md`.

## Affected Memory

- `cli/structure`: (modify) Document the path-resolution precedence (system fallback when outside git, `--system` flag, `~/.config/idea/backlog.md` XDG location) alongside the existing worktree-resolution notes.

## Impact

- **Code**:
  - `src/cmd/idea/resolve.go` — resolution precedence rewrite (the core change).
  - `src/cmd/idea/main.go` — register `--system` persistent flag; `--system`/`--main` conflict check.
  - `src/internal/idea/idea.go` — add `SystemBacklogPath()`; adjust `ResolveFilePath` rooting for the out-of-git `--file`/`IDEAS_FILE` case; on-demand dir creation on write.
  - Per-command `Long` help strings (`add.go`, `list.go`, `show.go`, `done.go`, `reopen.go`, `edit.go`, `rm.go`, `prune.go`, `fmt.go`).
  - `help_dump.go` output picks up the new flag automatically (walks the live tree).
- **Tests**: `t.TempDir()`-based tests for: outside-git fallback (point `$HOME`/`$XDG_CONFIG_HOME` at a temp dir), `--system` inside a repo, `--system`+`--main` conflict, on-demand dir creation. Git-dependent cases use a real temp repo per Constitution V.
- **Specs**: `docs/specs/overview.md` (Worktree Behavior + resolution precedence).
- **Constitution**: Principle II ("Worktree-Aware by Default, Main-Worktree Opt-In") narrows in scope — it still governs the in-git case, but a sanctioned non-git path now exists. May warrant a one-line amendment noting the system-backlog escape hatch. <!-- assumed: a constitution note is advisable but the change does not violate Principle II (git resolution remains default + only in-repo path); flagged for review, not blocking -->
- **No new dependencies** — uses `os.UserConfigDir`/stdlib only (stays within stdlib + cobra per Dependency Discipline).

## Open Questions

- Should the constitution's Principle II be formally amended to mention the system-backlog escape hatch, or is the spec/overview note sufficient? (Flagged in Impact; non-blocking.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Outside git, the default backlog is the system-level file (graceful fallback, no error) | Decided by the user in conversation — system-level chosen over CWD-local and required-flag; no remaining alternative | S:90 R:80 A:90 D:90 |
| 2 | Certain | Add a persistent `--system` flag (peer of `--main`) forcing the system backlog from anywhere | Decided by the user in conversation — "yes, add --system" over implicit-only; mirrors the existing `--main` pattern exactly | S:90 R:80 A:90 D:90 |
| 3 | Certain | System backlog lives at `$XDG_CONFIG_HOME/idea/backlog.md`, default `~/.config/idea/backlog.md` | Decided by the user — specified `~/.config/idea/backlog.md`, "hop style"; XDG-respect is the standard reading and verified against hop's own layout | S:88 R:75 A:88 D:85 |
| 4 | Confident | `--system` and `--main` together are a conflict error (non-zero exit), not silent precedence | Two explicit root selectors disagreeing should fail loudly; config gives no signal for a silent winner; cheap to change | S:55 R:75 A:65 D:70 |
| 5 | Confident | Out-of-git `--file`/`IDEAS_FILE` relative values root at the system config dir, not CWD | Keeps a single consistent non-git anchor; absolute paths are unaffected; reversible if it surprises users | S:55 R:70 A:65 D:65 |
| 6 | Certain | `--main` stays git-only (errors outside a repo, unchanged) | Constitution II + the definition of a worktree fix this — `--main` has no coherent out-of-git meaning; behavior is unchanged | S:85 R:80 A:92 D:90 |
| 7 | Confident | The system config dir is created on demand (`mkdir -p`) on first mutating write | Matches the zero-setup capture goal; erroring "no such directory" would defeat the feature. Trivially reversible | S:60 R:80 A:70 D:75 |
| 8 | Certain | File format, ID rules, and all CRUD/`fmt` semantics are unchanged — only the path differs | Constitution I + backlog-format spec fix the format; this change touches resolution only | S:90 R:75 A:95 D:95 |
| 9 | Certain | No new dependencies; use `os.UserConfigDir` / stdlib for the XDG path | Dependency Discipline (stdlib + cobra only) decides it; stdlib `UserConfigDir` already covers the XDG path | S:85 R:80 A:95 D:90 |
| 10 | Tentative | Constitution Principle II gets a one-line note about the system-backlog escape hatch; spec/overview updated regardless | Reasonable to keep the constitution accurate, but whether to amend vs. spec-note-only is a judgment call — flagged, non-blocking | S:50 R:65 A:55 D:50 |

10 assumptions (6 certain, 3 confident, 1 tentative, 0 unresolved).
