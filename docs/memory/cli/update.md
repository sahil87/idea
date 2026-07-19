---
description: "Self-update subcommand (`idea update`): Homebrew-backed upgrade flow, non-brew install fallback hint, and the `--skip-brew-update` flag"
type: memory
---

# `idea update` Subcommand

`idea update` is a self-update command that upgrades the running binary in place via Homebrew. The cobra wrapper lives at `src/cmd/idea/update.go`; the behavior lives at `src/internal/idea/update.go`. The split follows Constitution Principle IV — `cmd/idea/update.go` contains only the cobra factory plus error mapping, and the `internal/idea` package owns all subprocess invocation, version comparison, and Homebrew detection.

## Purpose

Users invoke `idea update` to refresh their local install to the latest release. The command is a thin wrapper around `brew update --quiet` + `brew info --json=v2 sahil87/tap/idea` + `brew upgrade sahil87/tap/idea`. There is no auto-update on every invocation, no scheduled background check, and no telemetry — the user explicitly opts in by running the subcommand. For non-Homebrew installs (manual `scripts/install.sh`, package managers other than Homebrew, raw `go build` outputs), the command prints a two-line hint (the current version note plus a `brew install sahil87/tap/idea` reinstall pointer) and exits 0 without attempting an upgrade.

The behavior lives in `func Update(currentVersion string, skipBrewUpdate bool, out, errOut io.Writer) error`. The `skipBrewUpdate` boolean is threaded down from the cobra layer (see [The `--skip-brew-update` flag](#the---skip-brew-update-flag)); the parameter sits between the version input and the output sinks, grouping inputs ahead of writers. `Update` has exactly one caller (`cmd/idea/update.go`).

## The `--skip-brew-update` flag

`update` registers a single local boolean flag, `--skip-brew-update` (default `false`), via `cmd.Flags().BoolVar(&skipBrewUpdate, "skip-brew-update", false, ...)` in the `updateCmd()` factory. It is a local flag on `update`, not a root persistent flag — the persistent flags (`--file`, `--main`, per Constitution III) are reserved for backlog-wide concerns, and this flag is specific to the update path. `RunE` reads the parsed value and passes it as the second argument to `idea.Update(...)`.

The flag's effect is narrowly scoped: when `skipBrewUpdate` is `true`, `Update` skips ONLY the internal `brew update --quiet` tap-metadata refresh (the `if !skipBrewUpdate { ... }` guard wraps just that block). Control then flows straight to the `brew info` version check. The `brew info` query, the up-to-date short-circuit, and `brew upgrade` are all unaffected by the flag, as is the non-Homebrew-install short-circuit at the top of `Update`. When the flag is absent (`false`), the brew path runs exactly as it otherwise would: `brew update --quiet`, then `brew info`, then the short-circuit, then `brew upgrade` when versions differ.

The flag exists so callers in automated or orchestrated contexts — CI, or a wrapper that has *just* run `brew update` across several brew-installed tools — can suppress the redundant tap-metadata refresh and its per-tap network round-trip. The flag name and semantics are fixed by a cross-toolkit contract shared by sibling tools, so the spelling and behavior are not repo-specific.

## Homebrew detection

`isBrewInstalled()` decides whether to take the brew code path. The check is a substring match against the resolved real path of the running binary:

1. `os.Executable()` returns the path to the current binary.
2. `filepath.EvalSymlinks(self)` resolves through any wrapper symlinks (e.g. `/opt/homebrew/bin/idea` → `.../Cellar/idea/<version>/bin/idea`).
3. The resolved path is checked for the substring `/Cellar/`.

Either step returning an error is treated as "not a Homebrew install" — `isBrewInstalled` returns `false`. This is deliberately defensive: if path resolution is broken we would rather print the manual-update hint than spawn `brew` against a binary whose origin we cannot determine.

## Formula reference

Every `brew` invocation that takes a formula argument uses the fully-qualified tap name `sahil87/tap/idea`, declared as the package-level constant `brewFormula`. Bare `idea` is never used. The fully-qualified form disambiguates against any same-named core formula that could otherwise shadow the tap on `brew info idea`. The same string also appears verbatim in the non-Homebrew hint (`brew install sahil87/tap/idea`).

## Version comparison

`normalizeVersion(v string) string` strips a single leading `"v"` and returns the rest unchanged. There is no semver parsing — equality after a one-character strip is sufficient because both sides of the comparison originate from the same canonical artifact. The release tag is the source of truth: `scripts/release.sh` pushes `vX.Y.Z`, the CI workflow stamps it via `-ldflags "-X main.version=vX.Y.Z"`, and `brew info --json=v2` reports the bare `X.Y.Z` from the published formula. After stripping a leading `v` from each side, byte equality is correct by construction.

The normalization also tolerates `""` and `"v"` (both → `""`) and only ever strips the first `v` (so `"vvv1.0.0"` becomes `"vv1.0.0"`).

## No-deadline brew-safety contract

All three brew subprocesses — `brew update --quiet`, `brew info --json=v2 sahil87/tap/idea`, and `brew upgrade sahil87/tap/idea` — run under `context.Background()`, passed at every call site through the `execCommandContext(ctx, ...)` seam. No `context.WithTimeout` (or any other deadline-carrying context) wraps a brew subprocess, and there are no timeout constants.

The prohibition is a MUST-level clause of the toolkit update standard's brew-handling safety article: no code path may send `SIGKILL` to a package-manager subprocess mid-transaction, and `brew upgrade` may not carry a short hard timeout. `exec.CommandContext`'s default cancel function sends `os.Kill` (SIGKILL) when its context's deadline fires. A SIGKILL landing between brew's `unlink` and `link` steps corrupts the keg (leaving a half-installed binary — `zsh: permission denied: <tool>`), which is the exact failure the standard cites as its motivating incident. With `context.Background()` the cancel path is never armed, so `exec.CommandContext` behaves identically to `exec.Command` on the success path — no bound ever fires.

Ctrl-C is the user's escape hatch: SIGINT reaches the foreground process group, brew traps it and unwinds cleanly. `brew upgrade` inherits the tty (`os.Stdin/Stdout/Stderr`, see I/O routing below), so a slow upgrade is visible rather than a silent hang; `brew update`/`brew info` run after the wrapper prints `Checking for updates...`, so the user knows what is running.

### Design Decisions

**Decision**: Remove deadlines entirely (pass `context.Background()`) rather than replace them with a generous SIGTERM+grace bound.
**Why**: The standard's verification checklist requires "No code path sends SIGKILL to brew," which is only strictly satisfiable by having no kill path at all — Go's graceful pattern (`cmd.Cancel` = SIGTERM + `cmd.WaitDelay`) still escalates to SIGKILL once the grace elapses. The standard itself points away from bounding the call at all, and Ctrl-C remains the escape hatch. All three brew calls are treated uniformly (`brew update` and `brew info` also mutate/read tap state, so the unqualified "no SIGKILL to brew" clause applies to every call, and uniform treatment is simpler than a per-call split).
**Rejected**: A SIGTERM + `WaitDelay` grace bound — still ends in a SIGKILL after the grace, so it fails the checklist's unqualified clause. `HOMEBREW_NO_GITHUB_API=1` — the standard suggests it only for tools that must bound the call; with no bound it is unnecessary.
*Introduced by*: 260719-6gjq-update-version-standards-conformance

## I/O routing

The command makes a deliberate split between small wrapper messages and large tty-aware subprocess streams:

- **Wrapper messages** (`Current version: ...`, `Already up to date (...).`, `Updating ... → v...`, `Updated to v...`, the non-Homebrew hint) are written to the `out io.Writer` parameter.
- **Error hints** (`idea update: brew not found on PATH.`) are written to the `errOut io.Writer` parameter.
- **`brew update` and `brew info` streams** are captured (via `exec.CommandContext.Output()` / `.Run()`) and never routed through `out`/`errOut`. `brew info` output is parsed for `formulae[0].versions.stable`; `brew update` output is discarded.
- **`brew upgrade` streams** inherit `os.Stdin` / `os.Stdout` / `os.Stderr` directly. Brew's tty-aware progress and colored output need a real terminal handle; piping through `out`/`errOut` would defeat that.

The cobra `RunE` passes `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` so production callers route to the real terminal while tests can substitute `bytes.Buffer` for the wrapper messages.

`--skip-brew-update` does not alter any of this routing. When the flag is set, the only difference is that the captured `brew update` subprocess is never spawned; every remaining stream (captured `brew info`, stdio-inheriting `brew upgrade`, wrapper messages to `out`/`errOut`) is routed exactly as above.

## Error mapping: the `errSilent` sentinel

When `brew` is missing on `PATH`, the internal `Update` function writes `idea update: brew not found on PATH.` to `errOut` itself, then returns the original `exec.ErrNotFound`-wrapping error. The cobra `RunE` in `cmd/idea/update.go` detects this case via `errors.Is(err, exec.ErrNotFound)` and returns the `cmd/idea`-package-local sentinel `errSilent` (declared in `main.go`). The top-level error handler in `main()` checks `errors.Is(err, errSilent)` and skips its own `ERROR: ...` line, exiting non-zero without duplicating the message.

The pattern is selective by design: `cmd.SilenceErrors = true` would suppress the cobra error printer for every error returned by `update`, including legitimate ones like `brew upgrade failed`. The sentinel approach silences only the cases where the internal package has already printed a user-facing message. Future cobra subcommands that want the same "internal package owns the error message, cobra stays quiet" pattern should follow the same shape: have the internal package write its hint to its `errOut` parameter, return an error the wrapper can identify (a sentinel or a wrapping error type), and map it to `errSilent` in `RunE`. See change `260508-5bw2-add-update-subcommand` for the originating context.

## Test seam for the brew path

The brew code path is unreachable under `go test` by default: `isBrewInstalled()` returns `false` for the test binary (it lives under a temp dir, not `/Cellar/`), and `Update` would otherwise shell out to real `brew`. To make the brew path testable — specifically to assert *which* `brew` subprocesses run for a given flag value — two package-level indirection vars sit at the top of `internal/idea/update.go`:

```go
var (
	execCommandContext = exec.CommandContext
	brewInstalled      = isBrewInstalled
)
```

`Update` and `brewLatestVersion` call `execCommandContext(...)` at all three brew call sites (`update`, `upgrade`, `info`) and `Update` calls `brewInstalled()` instead of `isBrewInstalled()` directly. In production these resolve to the stdlib function and the real detector, so runtime behavior is identical. These are NOT a command-runner abstraction or an `internal/proc` wrapper — `os/exec` remains the mechanism. The minimal seam is deliberate: refactoring the subprocess convention is explicitly out of scope, and a one-domain package does not warrant a runner interface (the same reasoning that keeps update logic in `internal/idea` rather than a separate `internal/update` package — Constitution Principle IV).

Tests stub both vars (restoring them via `defer`) to force the brew path and record invocations. `TestUpdateSkipBrewUpdate` in `update_test.go` is the table-driven exercise of the flag: it sets `brewInstalled` to return `true` and `execCommandContext` to a recorder that captures a `brewCall{sub, ctx}` per invocation — the brew subcommand plus the `ctx` the call site passed through the seam — and re-executes the test binary with `-test.run=TestHelperProcess` (the canonical stdlib fake-exec idiom, guarded by `GO_WANT_HELPER_PROCESS=1`). The helper process fakes `brew info` with valid `--json=v2` output reporting a stable version (`9.9.9`) that differs from the test's current version so the upgrade path is taken, and exits 0 for `update`/`upgrade`. The table asserts that the flag-absent row records `update`, while both rows record `info` and `upgrade` — proving the skip omits only the tap-metadata refresh. (260531-t2ov-skip-brew-update-flag)

The recorded `ctx` is the assertion surface for the no-deadline brew-safety contract (above): the same loop asserts every recorded brew invocation's `ctx.Deadline()` reports no deadline (`ok == false`) across both table rows and all three subcommands (`update`, `info`, `upgrade`). Reintroducing a `context.WithTimeout` around any brew call site fails this assertion. (260719-6gjq-update-version-standards-conformance)

## Cross-references

- Release pipeline that publishes the Homebrew formula consumed by this command: `../release/pipeline.md`.
- Source-tree placement (`cmd/idea/update.go`, `internal/idea/update.go`, `internal/idea/update_test.go`): `structure.md`.
- Constitution Principle IV (logic in `internal/idea`, not `cmd/`) and Dependency Discipline (stdlib + cobra only): `fab/project/constitution.md`.
