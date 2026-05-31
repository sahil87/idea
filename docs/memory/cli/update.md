# `idea update` Subcommand

`idea update` is a self-update command that upgrades the running binary in place via Homebrew. The cobra wrapper lives at `src/cmd/idea/update.go`; the behavior lives at `src/internal/idea/update.go`. The split follows Constitution Principle IV — `cmd/idea/update.go` contains only the cobra factory plus error mapping, and the `internal/idea` package owns all subprocess invocation, version comparison, and Homebrew detection.

## Purpose

Users invoke `idea update` to refresh their local install to the latest release. The command is a thin wrapper around `brew update --quiet` + `brew info --json=v2 sahil87/tap/idea` + `brew upgrade sahil87/tap/idea`. There is no auto-update on every invocation, no scheduled background check, and no telemetry — the user explicitly opts in by running the subcommand. For non-Homebrew installs (manual `scripts/install.sh`, package managers other than Homebrew, raw `go build` outputs), the command prints a two-line hint (the current version note plus a `brew install sahil87/tap/idea` reinstall pointer) and exits 0 without attempting an upgrade.

## `--skip-brew-update` flag

`idea update --skip-brew-update` (boolean, defaults `false`) skips ONLY the internal `brew update --quiet` tap-metadata refresh. Everything else runs unchanged: the `brew info` version check, the "already up to date" short-circuit, and `brew upgrade`. The default (flag absent) preserves the original behavior exactly.

The flag name is a **cross-toolkit contract** shared with five sibling tools — it must remain exactly `--skip-brew-update`. It lets a caller that has already refreshed brew's tap metadata (or wants to avoid the network round-trip) skip the redundant refresh while still performing the upgrade.

Wiring: the cobra factory `updateCmd()` in `cmd/idea/update.go` declares a local `skipBrewUpdate bool`, registers it via `cmd.Flags().BoolVar(&skipBrewUpdate, "skip-brew-update", false, …)`, and threads it as the second positional argument of `Update(version string, skipBrewUpdate bool, out, errOut io.Writer)`. In `internal/idea/update.go`, `Update` wraps the entire `brew update` block (timeout context, `commandContext(...)` construction, stderr capture, error mapping) in `if !skipBrewUpdate { … }`; the version check and upgrade live outside that guard and always run.

## Homebrew detection

`isBrewInstalled()` decides whether to take the brew code path. The check is a substring match against the resolved real path of the running binary:

1. `os.Executable()` returns the path to the current binary.
2. `filepath.EvalSymlinks(self)` resolves through any wrapper symlinks (e.g. `/opt/homebrew/bin/idea` → `.../Cellar/idea/<version>/bin/idea`).
3. The resolved path is checked for the substring `/Cellar/`.

Either step returning an error is treated as "not a Homebrew install" — `isBrewInstalled` returns `false`. This is deliberately defensive: if path resolution is broken we would rather print the manual-update hint than spawn `brew` against a binary whose origin we cannot determine.

`Update` calls this check indirectly through the package-level `brewInstalled` var (which aliases `isBrewInstalled` in production). The indirection is a test seam: because a `go test` binary never lives under `/Cellar/`, unit tests stub `brewInstalled` to return `true` so the brew code path can be exercised without a real Homebrew install. See the Testing section.

## Formula reference

Every `brew` invocation that takes a formula argument uses the fully-qualified tap name `sahil87/tap/idea`, declared as the package-level constant `brewFormula`. Bare `idea` is never used. The fully-qualified form disambiguates against any same-named core formula that could otherwise shadow the tap on `brew info idea`. The same string also appears verbatim in the non-Homebrew hint (`brew install sahil87/tap/idea`).

## Version comparison

`normalizeVersion(v string) string` strips a single leading `"v"` and returns the rest unchanged. There is no semver parsing — equality after a one-character strip is sufficient because both sides of the comparison originate from the same canonical artifact. The release tag is the source of truth: `scripts/release.sh` pushes `vX.Y.Z`, the CI workflow stamps it via `-ldflags "-X main.version=vX.Y.Z"`, and `brew info --json=v2` reports the bare `X.Y.Z` from the published formula. After stripping a leading `v` from each side, byte equality is correct by construction.

The normalization also tolerates `""` and `"v"` (both → `""`) and only ever strips the first `v` (so `"vvv1.0.0"` becomes `"vv1.0.0"`).

## Timeout constants

Three package-level constants govern subprocess wait time:

| Constant | Duration | Subprocess |
|----------|----------|------------|
| `brewUpdateTimeout` | 30s | `brew update --quiet` |
| `brewInfoTimeout` | 30s | `brew info --json=v2 sahil87/tap/idea` |
| `brewUpgradeTimeout` | 120s | `brew upgrade sahil87/tap/idea` |

Each subprocess runs under a `context.WithTimeout` derived from these constants. The 120s upgrade budget reflects that `brew upgrade` may need to download a tarball, run the formula's `test do` block, and update its own metadata. The two 30s budgets cover the metadata-only operations.

## I/O routing

The command makes a deliberate split between small wrapper messages and large tty-aware subprocess streams:

- **Wrapper messages** (`Current version: ...`, `Already up to date (...).`, `Updating ... → v...`, `Updated to v...`, the non-Homebrew hint) are written to the `out io.Writer` parameter.
- **Error hints** (`idea update: brew not found on PATH.`) are written to the `errOut io.Writer` parameter.
- **`brew update` and `brew info` streams** are captured (via `.Output()` / `.Run()` on the `*exec.Cmd`) and never routed through `out`/`errOut`. `brew info` output is parsed for `formulae[0].versions.stable`; `brew update` output is discarded.
- **`brew upgrade` streams** inherit `os.Stdin` / `os.Stdout` / `os.Stderr` directly. Brew's tty-aware progress and colored output need a real terminal handle; piping through `out`/`errOut` would defeat that.

The cobra `RunE` passes `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` so production callers route to the real terminal while tests can substitute `bytes.Buffer` for the wrapper messages.

## Error mapping: the `errSilent` sentinel

When `brew` is missing on `PATH`, the internal `Update` function writes `idea update: brew not found on PATH.` to `errOut` itself, then returns the original `exec.ErrNotFound`-wrapping error. The cobra `RunE` in `cmd/idea/update.go` detects this case via `errors.Is(err, exec.ErrNotFound)` and returns the `cmd/idea`-package-local sentinel `errSilent` (declared in `main.go`). The top-level error handler in `main()` checks `errors.Is(err, errSilent)` and skips its own `ERROR: ...` line, exiting non-zero without duplicating the message.

The pattern is selective by design: `cmd.SilenceErrors = true` would suppress the cobra error printer for every error returned by `update`, including legitimate ones like `brew upgrade failed`. The sentinel approach silences only the cases where the internal package has already printed a user-facing message. Future cobra subcommands that want the same "internal package owns the error message, cobra stays quiet" pattern should follow the same shape: have the internal package write its hint to its `errOut` parameter, return an error the wrapper can identify (a sentinel or a wrapping error type), and map it to `errSilent` in `RunE`. See change `260508-5bw2-add-update-subcommand` for the originating context.

## Testing

Two package-level seams in `internal/idea/update.go` make the brew code path unit-testable without a real Homebrew install (Constitution Principle IV calls for exactly this kind of testable seam):

- `var commandContext = exec.CommandContext` — every brew `*exec.Cmd` is built through this alias. Production uses the stdlib constructor unchanged (same `.Run()` / `.Output()` calls, same stream wiring); tests swap it for a fake that records each brew subcommand and returns a `*exec.Cmd` re-execing the test binary.
- `var brewInstalled = isBrewInstalled` — tests stub it to `func() bool { return true }` to reach the brew path.

`update_test.go` uses the standard `os/exec` helper-process idiom: `fakeCommandContext` re-execs the test binary with `-test.run=TestHelperProcess`, passing `GO_WANT_HELPER_PROCESS=1` and `GO_BREW_SUBCOMMAND=<sub>`. `TestHelperProcess` emits parseable `versions.stable` JSON only when impersonating `brew info`, and exits 0 otherwise. `withFakeBrew(t)` installs both seams and restores them via `t.Cleanup`, returning a `*[]string` of the brew subcommands invoked.

Behavior tests assert against that recorded slice:

- `TestUpdateSkipsBrewUpdateWhenFlagSet` — with `skipBrewUpdate=true`, `update` is NOT recorded but `info` and `upgrade` are (the core `--skip-brew-update` contract).
- `TestUpdateRunsBrewUpdateByDefault` — with the flag `false`, all three (`update`, `info`, `upgrade`) run.
- `TestUpdateAlreadyUpToDate` — when the current version equals the reported stable version, no `upgrade` runs and the "Already up to date" message is emitted.

## Cross-references

- Release pipeline that publishes the Homebrew formula consumed by this command: `../release/pipeline.md`.
- Source-tree placement (`cmd/idea/update.go`, `internal/idea/update.go`, `internal/idea/update_test.go`): `structure.md`.
- Constitution Principle IV (logic in `internal/idea`, not `cmd/`) and Dependency Discipline (stdlib + cobra only): `fab/project/constitution.md`.
