# idea CLI Overview

The `idea` command manages a per-repo backlog stored in `fab/backlog.md`. It is a lightweight CRUD tool for capturing, triaging, and tracking ideas — small enough to live alongside hand-edited markdown, structured enough to be queryable from the command line.

## Binary & Installation

**Binary**: `src/cmd/idea/` (Go binary, distributed as `idea`).

Install via Homebrew tap:

```bash
brew install sahil87/tap/idea
```

Or install manually from a clean checkout:

```bash
./scripts/install.sh
```

The manual installer builds the binary via `./scripts/build.sh` and copies it to `~/.local/bin/idea`.

## Worktree Behavior

By default, `idea` operates on the **current worktree's** `fab/backlog.md` (resolved via `git rev-parse --show-toplevel`). Pass `--main` to target the main worktree's backlog instead; internally, `idea` resolves the main worktree root by running `git rev-parse --path-format=absolute --git-common-dir` and taking its parent directory. In the main worktree, both behave identically. This ensures that users in a linked worktree get predictable local behavior unless they explicitly opt into the shared backlog.

The backlog file path can also be overridden globally by `--file <path>` (relative to the resolved git root) or by setting the `IDEAS_FILE` environment variable.

## Commands

| Command | Description |
|---------|-------------|
| `idea "text"` | Add a new idea (shorthand for `idea add`) |
| `idea add "text"` | Add a new idea to the backlog |
| `idea list` | List open (uncompleted) ideas |
| `idea show <query>` | Show a single idea matching the query |
| `idea done <query>` | Mark an idea as done |
| `idea reopen <query>` | Reopen a completed idea |
| `idea edit <query> "text"` | Modify an idea's text |
| `idea rm <query> --force` | Delete an idea (requires `--force` to confirm) |
| `idea prune [--force]` | Bulk-remove all done ideas (dry run by default; `--force` to delete) |

## ID Format & Query Semantics

Each idea gets a short 4-character lowercase alphanumeric ID (e.g., `[a7k2]`) and an ISO date (`YYYY-MM-DD`). IDs are unique within a single backlog file.

Queries (the `<query>` argument on `show`, `done`, `reopen`, `edit`, `rm`) match against either the ID or the description text. Matching is substring, case-insensitive. If a query matches more than one idea, the command refuses to act and lists the matches.

## Parse & Format Behavior (lenient read, canonical write)

`idea` is **liberal in what it accepts and strict in what it emits**:

- **Lenient on read.** The `YYYY-MM-DD:` date segment is **optional** on input, and `idea` also accepts `*`/`+` bullets (in addition to `-`), arbitrary leading whitespace, and CRLF or LF line endings. A line is recognized as an idea by its `[ ]`/`[x]` checkbox plus 4-char `[id]` anchors. This means a hand-edited or externally-authored backlog of dateless `- [ ] [id] text` lines is read correctly rather than silently ignored.
- **Canonical on write.** Every idea line `idea` writes uses one canonical form — `- ` bullet, no indentation, LF endings, and a date that is **always present** (today's date is backfilled when the input had none). A mutating command (`done`/`reopen`/`edit`/`rm`) normalizes all recognized idea lines in the file at once; non-mutating commands (`list`/`show`) never rewrite the file.
- **Backfill notice.** When a mutating save stamps today's date on one or more previously-dateless items, `idea` prints a brief advisory notice to **stderr** (`note: stamped today's date on N previously-dateless item(s)`), keeping stdout machine-parseable. The notice is suppressed when nothing was backfilled.

For the full backlog line format — accepted input variants, the canonical output form, date backfill, Shape B pass-through, and the format-contract change note — see [`backlog-format.md`](backlog-format.md).

## External-Consumer Integration

The backlog file is plain Markdown with a stable line format. Any tool that reads `fab/backlog.md` can discover backlog IDs and descriptions without coupling to `idea`'s internals — this is the contract.

One example consumer is fab-kit's `/fab-new`, which can accept a backlog ID and pull the description directly from the file when starting a new change. That integration is illustrative, not defining: the file format is the API, and `idea` is one (canonical) writer of that format.
