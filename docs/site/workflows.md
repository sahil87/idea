# Workflows

`idea` is small, but two of its behaviors are worth understanding in depth:
**how it resolves which backlog file to touch**, and **how it plugs into the
fab-kit change loop**. This page also documents the backlog file format, which is
a stable public contract you can build your own tooling against.

New here? Start with the [install guide](install.md), then come back.

## Worktree-aware resolution

`idea` resolves the backlog file based on *where you run it* — not on an
environment default or a global config. By default it reads and writes the
**current worktree's** `fab/backlog.md`.

| Where you are | What you type | Which file `idea` touches |
|---------------|---------------|----------------------------|
| Main repo | `idea add "..."` | `<main>/fab/backlog.md` |
| Linked worktree | `idea add "..."` | `<worktree>/fab/backlog.md` (local to this worktree) |
| Linked worktree | `idea --main add "..."` | `<main>/fab/backlog.md` (shared with the team) |
| Anywhere | `idea --file path/to/file.md ...` | that file (relative to the git root) |
| Anywhere | `IDEAS_FILE=... idea ...` | the env-var path |

How resolution actually works under the hood:

- **Current worktree** (the default): `idea` runs `git rev-parse --show-toplevel`
  and uses `<toplevel>/fab/backlog.md`.
- **`--main`**: `idea` runs `git rev-parse --path-format=absolute --git-common-dir`
  and takes that directory's parent as the main worktree root, then uses
  `<main>/fab/backlog.md`. In the *main* worktree, `--main` and the default
  behave identically.
- **`--file <path>`**: overrides the path entirely (resolved relative to the git
  root).
- **`IDEAS_FILE`**: same override via environment variable.

**Why the default favors the current worktree.** When you're heads-down on a
change in a linked worktree and capture a thought, you almost always mean "for
*this* branch." Defaulting to the current worktree keeps parallel changes from
stepping on each other's backlogs. `--main` is the explicit opt-in for "add this
to the shared roadmap." Resolution never reads heuristics or stray environment
state — it asks git directly, so behavior is predictable from your current
directory alone.

## fab-kit integration loop

`fab/backlog.md` is the same file fab-kit's `/fab-new` reads, so capture and
execution close into a single loop:

1. **Capture** — `idea "add retry logic to API client"` drops a new item into the
   current worktree's backlog.
2. **Triage** — `idea list` shows what's open; narrow with substring queries or
   the 4-char ID.
3. **Start work** — `/fab-new <id>` (in your AI agent) pulls the description
   straight from the backlog and spins up a change folder plus branch.
4. **Close** — `idea done <id>` after the change ships.

For bulk work, `fab batch new` reads every open idea and spawns a worktree plus a
Claude session per item — the whole backlog becomes a parallel work queue in one
command.

## Backlog format as a public contract

The line format in `fab/backlog.md` is the public API between `idea` and any
external consumer — `idea` is just one (canonical) writer of it. There are two
shapes:

- **Shape A — idea-managed**: `- [ ] [{ID}] {YYYY-MM-DD}: {description}` (no
  second bracket). `idea` parses, queries, edits, and round-trips these lines.

  ```text
  - [ ] [ngaw] 2026-02-23: Quality gate
  ```

- **Shape B — pass-through**: any line with extra content between `[{ID}]` and
  the date (for example the optional `[{issue_ids}]` bracket that fab-kit's
  `/fab-new` writes). `idea` does **not** match these against its parser regex,
  so they are treated as inert pass-through content — invisible to `idea list`,
  `show`, `done`, `edit`, and `rm` — but preserved byte-for-byte on every
  round-trip.

  ```text
  - [ ] [ni3o] [DEV-1011] 2026-02-12: Capture more metrics
  ```

The semantics of the second bracket are owned by external consumers, not by
`idea` — which is exactly what lets `idea` and other tools share one backlog file
without either needing to understand the other's metadata. The Shape A parsing
contract will not change without a major-version bump and a documented migration.

For the full field-by-field specification, see the format spec at
[backlog-format.md](https://github.com/sahil87/idea/blob/main/docs/specs/backlog-format.md).
