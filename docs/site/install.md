# Install

The fastest way to get `idea` is the Homebrew tap. If you'd rather build from
source, a single `just` recipe handles it. This page covers both, plus shell
completion and upgrades.

## Homebrew tap (recommended)

```bash
brew install sahil87/tap/idea
```

The tap gives you a prebuilt `idea` binary on your `$PATH`, managed by Homebrew —
so upgrades, version pinning, and uninstall all go through `brew`. This is the
path most people want: nothing to compile, and `idea update` (see
[Upgrading](#upgrading)) can self-upgrade through the same tap later.

## Manual build from a clean checkout

If you'd rather build it yourself (no Homebrew, or you want to track `main`),
clone the repo and run the install recipe:

```bash
just local-install
```

**Prerequisites**: [Go](https://go.dev/dl/) and [`just`](https://github.com/casey/just).

This builds the binary (stamped with the `git describe` version) and copies it to
`~/.local/bin/idea`. Make sure that directory is on your `$PATH` — for example:

```sh
export PATH="$HOME/.local/bin:$PATH"   # add to ~/.zshrc or ~/.bashrc
```

To build without installing (binary lands at `./bin/idea`), use `just build`.

## Shell completion

`idea shell-init <shell>` emits an eval-safe tab-completion snippet for your
shell. It tab-completes subcommands, flags, and the bare `idea <text>` shorthand.
Supported shells: **zsh, bash, fish, powershell**.

Wire it into your rc file once:

```sh
eval "$(idea shell-init zsh)"   # in ~/.zshrc
eval "$(idea shell-init bash)"  # in ~/.bashrc
```

For fish and powershell, the snippet documents its own install line in a banner
comment at the top of the output:

```sh
idea shell-init fish | source                 # current session
idea shell-init fish > ~/.config/fish/completions/idea.fish   # persistent
```

```powershell
idea shell-init powershell | Out-String | Invoke-Expression   # add to $PROFILE
```

The zsh snippet lazy-loads `compinit` if you haven't initialised the completion
system yet, so it's safe to drop into an rc file unconditionally.

> Have other shll tools? `shll shell-install` wires up the shell integrations
> and autocompletions for **all** of them at once — see
> [shll shell-install](https://github.com/sahil87/shll#shll-shell-install--wire-the-rc-file-recommended).

## Upgrading

If you installed via Homebrew, upgrade in place with:

```bash
idea update
```

This refreshes the tap metadata, checks the installed version against the latest
published release, and runs `brew upgrade` only when a newer version exists — if
you're already current it prints `Already up to date` and exits. (Pass
`--skip-brew-update` to skip only the tap-metadata refresh; faster, but it may
miss a just-published release.) If `idea` was not installed via Homebrew, the
command explains how to update manually instead — for a manual build, re-run
`just local-install` from an updated checkout.

---

Next, see the [workflows](workflows.md) guide for worktree-aware behavior, the
fab-kit loop, and the backlog file format.
