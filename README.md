# idea

> Part of [@sahil87's open source toolkit](https://ai.shll.in) — see all projects there.

[![Latest release](https://img.shields.io/github/v/release/sahil87/idea)](https://github.com/sahil87/idea/releases) [![Downloads](https://img.shields.io/github/downloads/sahil87/idea/total)](https://github.com/sahil87/idea/releases) [![Stars](https://img.shields.io/github/stars/sahil87/idea?style=social)](https://github.com/sahil87/idea/stargazers)

Capture and manage ideas from the command line. A worktree-aware backlog tracker that keeps `fab/backlog.md` as the source of truth.

## Install

Homebrew tap:

```bash
brew install sahil87/tap/idea
```

Or build and install manually from a clean checkout:

```bash
./scripts/install.sh
```

## Quick Start

```bash
idea "refactor auth middleware"   # capture an idea
idea list                         # see what's open
idea show <id>                    # inspect a single idea
idea done <id>                    # mark an idea complete
```

## Commands

`add`, `list`, `show`, `done`, `reopen`, `edit`, `rm` — plus the bare-text shorthand (`idea "text"` is equivalent to `idea add "text"`). See [docs/specs/overview.md](docs/specs/overview.md) for the full reference.

- **Worktree behavior**: commands target the current worktree's `fab/backlog.md` by default; pass `--main` to target the main worktree's backlog, override the path with `--file`, or set `IDEAS_FILE` in the environment. See [docs/specs/overview.md](docs/specs/overview.md) for resolution details.

## Integration

`idea` integrates with fab-kit's `/fab-new` via the `fab/backlog.md` file format — see [docs/specs/backlog-format.md](docs/specs/backlog-format.md) for the public contract.
