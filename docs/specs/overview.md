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

The backlog file path can also be overridden globally by `--file <path>` or by setting the `IDEAS_FILE` environment variable.

### System backlog and out-of-git operation

`idea` also works **outside any git repository** and offers a **system-level backlog** for cross-repo idea capture. The system backlog lives at `$XDG_CONFIG_HOME/idea/backlog.md` when `XDG_CONFIG_HOME` is set, and `~/.config/idea/backlog.md` otherwise (resolved via Go's `os.UserConfigDir`). Its parent directory is created on demand on the first mutating write. The file format and all command semantics are identical to a repo backlog — only the path differs.

The backlog path is resolved by this precedence (first match wins):

1. **`--system`** — the system backlog, skipping git entirely (reachable from inside a repo too).
2. **`--file <path>` / `IDEAS_FILE`** — joined to the git root when inside a repo, else to the system config dir (`~/.config/idea/`). An absolute value is used verbatim.
3. **`--main`** — the main worktree root. Git-only: it still errors with `not in a git repository` outside a repo.
4. **In a git repo, no override** — `{worktree-root}/fab/backlog.md` (the default).
5. **Outside a git repo, no override** — the system backlog (the graceful fallback; commands no longer fail with `not in a git repository`).

`--system` and `--main` are mutually exclusive — passing both is a user error and exits non-zero.

## Commands

| Command | Description |
|---------|-------------|
| `idea "text"` | Add a new idea (shorthand for `idea add`) |
| `idea add "text"` | Add a new idea to the backlog |
| `idea list` | List open (uncompleted) ideas |
| `idea show <query>` | Show a single idea matching the query |
| `idea done <query>` | Mark an idea as done |
| `idea reopen <query>` | Reopen a completed idea |
| `idea edit <query>` | Edit an idea's text in your editor (`$VISUAL`, then `$EDITOR`, then `vi`) on the decoded text |
| `idea edit <query> "text"` | Replace an idea's text inline |
| `idea rm <query> --force` | Delete an idea (requires `--force` to confirm) |
| `idea prune [--force]` | Bulk-remove all done ideas (dry run by default; `--force` to delete) |
| `idea fmt` | Rewrite the backlog into canonical form, adopting bare checkbox lines (`--check` reports without writing) |

**Editor form contract** (`idea edit <query>`, no text argument): an unchanged buffer is a no-op — the backlog is untouched, a `note: text unchanged — nothing to do` advisory goes to stderr, and the exit code is 0. An emptied buffer is refused: no change, non-zero exit. A non-zero editor exit aborts: the backlog is untouched, non-zero exit. Passing `--id`/`--date` with the no-text form still opens the editor, applies the metadata at save, and suppresses the unchanged no-op — a metadata-only change lands without mutating the text.

## ID Format & Query Semantics

Each idea gets a short 4-character lowercase alphanumeric ID (e.g., `[a7k2]`) and an ISO date (`YYYY-MM-DD`). IDs are unique within a single backlog file.

Queries (the `<query>` argument on `show`, `done`, `reopen`, `edit`, `rm`) match against either the ID or the description text. Matching is substring, case-insensitive. If a query matches more than one idea, the command refuses to act and lists the matches.

## Parse & Format Behavior (lenient read, canonical write)

`idea` is **liberal in what it accepts and strict in what it emits**:

- **Lenient on read.** The `YYYY-MM-DD:` date segment is **optional** on input, and `idea` also accepts `*`/`+` bullets (in addition to `-`), arbitrary leading whitespace, and CRLF or LF line endings. A line is recognized as an idea by its `[ ]`/`[x]` checkbox plus 4-char `[id]` anchors. This means a hand-edited or externally-authored backlog of dateless `- [ ] [id] text` lines is read correctly rather than silently ignored.
- **Canonical on write.** Every idea line `idea` writes uses one canonical form — `- ` bullet, no indentation, LF endings, and a date that is **always present** (today's date is backfilled when the input had none). A mutating command (`done`/`reopen`/`edit`/`rm`, or `prune --force` when it removes items) normalizes all recognized idea lines in the file at once; non-mutating commands (`list`/`show`) never rewrite the file.
- **Backfill notice.** When a mutating save stamps today's date on one or more previously-dateless items, `idea` prints a brief advisory notice to **stderr** (`note: stamped today's date on N previously-dateless item(s)`), keeping stdout machine-parseable. The notice is suppressed when nothing was backfilled.
- **Explicit canonicalizer.** `idea fmt` rewrites the whole file into canonical form on demand — no semantic change required — and additionally **adopts** bare checkbox lines lacking the `[id]` anchor (fresh unique ID, today's date, checked state preserved), turning an existing markdown task list into a managed backlog in one command. It is idempotent (a second run is byte-stable and skips the write), reports to stderr while stdout stays empty, and `idea fmt --check` writes nothing, prints the would-be report, and exits non-zero when the file is not canonical.

For the full backlog line format — accepted input variants, the canonical output form, date backfill, Shape B pass-through, and the format-contract change note — see [`backlog-format.md`](backlog-format.md).

## External-Consumer Integration

The backlog file is plain Markdown with a stable line format. Any tool that reads `fab/backlog.md` can discover backlog IDs and descriptions without coupling to `idea`'s internals — this is the contract.

One example consumer is fab-kit's `/fab-new`, which can accept a backlog ID and pull the description directly from the file when starting a new change. That integration is illustrative, not defining: the file format is the API, and `idea` is one (canonical) writer of that format.
