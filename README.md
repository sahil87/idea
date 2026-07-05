# idea

> Part of [@sahil87's open source toolkit](https://shll.ai) — see all projects there.

[![Latest release](https://img.shields.io/github/v/release/sahil87/idea)](https://github.com/sahil87/idea/releases) [![Downloads](https://img.shields.io/github/downloads/sahil87/idea/total)](https://github.com/sahil87/idea/releases) [![Stars](https://img.shields.io/github/stars/sahil87/idea?style=social)](https://github.com/sahil87/idea/stargazers)

Capture and manage ideas from the command line. A worktree-aware backlog tracker that keeps `fab/backlog.md` as the source of truth — plain Markdown, queryable from the CLI, shareable with the rest of your team via git.

## Why idea?

- **Plain Markdown, not a database** — your backlog is a checked-in `fab/backlog.md` file. Hand-edit it, grep it, diff it, review it in PRs. `idea` is one (canonical) writer of the format; the file is the contract.
- **Per-repo by default, with a global escape hatch** — every repo has its own backlog, so there's no cross-project noise. When you're not in a repo (or want a personal, cross-cutting list), `idea` falls back to a system-level backlog at `~/.config/idea/backlog.md` — reachable from anywhere with `--system` (`-s`).
- **Worktree-aware** — by default `idea` reads/writes the *current* worktree's backlog, so parallel changes don't step on each other. `--main` (`-m`) opts into the shared backlog, and `--system` (`-s`) targets the global one.
- **Short, addressable IDs** — every idea gets a 4-character ID like `[qu1d]` you can type into any command. Queries also match free-text substrings.
- **Hooks into fab-kit** — `fab/backlog.md` is the same file fab-kit's `/fab-new` reads, so capturing an idea today and starting a change from it tomorrow is one command.

## Install

Homebrew tap:

```bash
brew install sahil87/tap/idea
```

Or build and install manually from a clean checkout (requires Go and `just`):

```bash
just local-install
```

Builds the binary and copies it to `~/.local/bin/idea`. Make sure that directory is on your `$PATH`.

To upgrade later: `idea update` (self-upgrades via Homebrew).

For Homebrew details, manual-build prerequisites, shell completion, and upgrades, see the full [install guide](docs/site/install.md).

## Shell completion

`idea shell-init <shell>` emits eval-safe tab-completion for your shell. Add this line to your rc file:

```sh
eval "$(idea shell-init zsh)"   # in ~/.zshrc
eval "$(idea shell-init bash)"  # in ~/.bashrc
```

Supports `zsh`, `bash`, `fish`, and `powershell`. Tab-completes subcommands, flags, and the `idea <text>` shorthand.

> 💡 Have other sahil87 tools? [`shll shell-install`](https://github.com/sahil87/shll#shll-shell-install--wire-the-rc-file-recommended) handles all of their shell integrations and autocompletions at once.

## Quick Start

A typical capture-and-triage session:

```text
$ idea "refactor auth middleware to use JWT"
Added: [qu1d] 2026-05-11: refactor auth middleware to use JWT

$ idea "add rate limiting to public endpoints"
Added: [dpr1] 2026-05-11: add rate limiting to public endpoints

$ idea "update README with new setup steps"
Added: [xumo] 2026-05-11: update README with new setup steps

$ idea list
- [ ] [qu1d] 2026-05-11: refactor auth middleware to use JWT
- [ ] [dpr1] 2026-05-11: add rate limiting to public endpoints
- [ ] [xumo] 2026-05-11: update README with new setup steps

$ idea done dpr1
Done: - [x] [dpr1] 2026-05-11: add rate limiting to public endpoints

$ idea list                 # open items only
- [ ] [qu1d] 2026-05-11: refactor auth middleware to use JWT
- [ ] [xumo] 2026-05-11: update README with new setup steps
```

Queries (the `<id>` arg on `show`, `done`, `reopen`, `edit`, `rm`) match against either the ID or the description text — substring, case-insensitive. So `idea done auth` would also have closed `qu1d`.

## Command reference

| Command | Summary |
|---------|---------|
| `idea "text"` | Add a new idea (shorthand for `idea add`). |
| `idea add "text"` | Add a new idea to the backlog. |
| `idea list` | List open ideas. `--all` includes done items, `--done` only done, `--json` for scripting, `--sort id\|date`, `--reverse`. |
| `idea show <query>` | Show a single idea matching the query (by ID or substring). |
| `idea done <query>` | Mark an idea as done. |
| `idea reopen <query>` | Reopen a completed idea. |
| `idea edit <query> "text"` | Replace an idea's description. |
| `idea rm <query> --force` | Delete an idea (requires `--force` to confirm). |
| `idea update` | Self-update via Homebrew. |

Run `idea <command> --help` for inline flag details, browse the [full command reference](https://shll.ai/tools/idea/commands/) on shll.ai, or see [`docs/specs/overview.md`](https://github.com/sahil87/idea/blob/main/docs/specs/overview.md) for the full CLI reference and [`docs/specs/backlog-format.md`](https://github.com/sahil87/idea/blob/main/docs/specs/backlog-format.md) for the file format contract.

### Worktree-aware by default

This is the one behavior worth knowing in detail. `idea` resolves the backlog file based on where you run it:

| Where you are | What you type | Which file `idea` touches |
|---------------|---------------|----------------------------|
| Main repo | `idea add "..."` | `<main>/fab/backlog.md` |
| Linked worktree | `idea add "..."` | `<worktree>/fab/backlog.md` (local to this worktree) |
| Linked worktree | `idea --main add "..."` (`-m`) | `<main>/fab/backlog.md` (shared with the team) |
| Outside any git repo | `idea add "..."` | `~/.config/idea/backlog.md` (the system backlog — automatic fallback) |
| Anywhere | `idea --system add "..."` (`-s`) | `~/.config/idea/backlog.md` (the system backlog, even inside a repo) |
| Anywhere | `idea --file path/to/file.md ...` (`-f`) | that file (relative to the git root, or to `~/.config/idea` outside a repo; absolute paths as-is) |
| Anywhere | `IDEAS_FILE=... idea ...` | the env-var path (same rooting as `--file`) |

Why the default favors the current worktree: when you're heads-down on a change and capture a thought, you usually mean "for *this* branch." `--main` (`-m`) is the explicit opt-in for "add this to the shared roadmap." In the main worktree, `--main` and the default behave identically. `--system` (`-s`) is the opt-in for a personal, cross-repo list — and it's also what you get automatically when you run `idea` outside any git repo, so capture works everywhere. (The system path is always `~/.config/idea/backlog.md` on every platform — `$XDG_CONFIG_HOME` is ignored. `--system` and `--main` are mutually exclusive.)

## Integration with fab-kit

`fab/backlog.md` is the same file fab-kit reads, so the loop closes naturally:

1. **Capture** — `idea "add retry logic to API client"`
2. **Triage** — `idea list` to review what's open
3. **Start work** — `/fab-new <id>` (in your AI agent) pulls the description from the backlog and spins up a change folder + branch
4. **Close** — `idea done <id>` after the change ships

For bulk work, `fab batch new` reads every open idea and spawns a worktree + Claude session per item — the whole backlog becomes a parallel work queue in one command.

The backlog format is a stable public contract — any tool that follows [`backlog-format.md`](https://github.com/sahil87/idea/blob/main/docs/specs/backlog-format.md) can read or write the file without coupling to `idea`'s internals.

For a deeper walkthrough of worktree resolution, the fab-kit loop, and the backlog format, see [workflows](docs/site/workflows.md).

## Gotchas

- **`idea rm` requires `--force`.** This is intentional — deletes are destructive and there's no undo. Use `idea done` if you just want the item out of `idea list`.
- **Ambiguous queries refuse to act.** If your query matches more than one idea, `idea done`, `edit`, and `rm` print the matches and exit without changing anything. Disambiguate with the 4-char ID.
- **Lines with extra brackets are invisible to queries.** fab-kit's `/fab-new` writes issue IDs into a second bracket (e.g., `- [ ] [qu1d] [DEV-1011] 2026-05-11: ...`). Those lines are preserved verbatim in the file but won't appear in `idea list` or match `idea show` — they're treated as inert pass-through content. See [`backlog-format.md`](https://github.com/sahil87/idea/blob/main/docs/specs/backlog-format.md) for the Shape A vs. Shape B distinction.
