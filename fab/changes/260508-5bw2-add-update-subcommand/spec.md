# Spec: Add `idea update` Subcommand

**Change**: 260508-5bw2-add-update-subcommand
**Created**: 2026-05-08
**Affected memory**: `docs/memory/cli/update.md` (new), `docs/memory/cli/structure.md` (modify)

## Non-Goals

- Semver-aware version comparison — single-`v`-strip + string equality is sufficient because both sides come from the same release-tag canonical source.
- Auto-update on every CLI invocation, scheduled background checks, or telemetry.
- Non-Homebrew install channel handling beyond the existing "not installed via Homebrew" hint (apt, pacman, manual `scripts/install.sh` users see the hint).
- Rollback or version pinning — Homebrew handles those natively.

## CLI: `idea update` Subcommand

### Requirement: Subcommand Registration

The `idea` root command SHALL register a top-level subcommand named `update`. The subcommand SHALL accept no positional arguments (`cobra.NoArgs`). The subcommand's `Short` description SHALL be `"self-update the idea binary via Homebrew"`. The subcommand SHALL inherit the root's persistent `--file` and `--main` flags but SHALL NOT consume them; both flags are irrelevant to self-update behaviour.

#### Scenario: Subcommand listed in help

- **GIVEN** the binary is built
- **WHEN** the user runs `idea --help`
- **THEN** the output lists `update` as an available command with the short description `"self-update the idea binary via Homebrew"`

#### Scenario: Rejects positional arguments

- **GIVEN** the user runs `idea update foo`
- **WHEN** cobra parses the command line
- **THEN** the command fails with cobra's standard "accepts 0 arg(s), received 1" error
- **AND** the binary exits non-zero

### Requirement: Wiring Site

The `update` subcommand SHALL be registered via the existing `root.AddCommand(...)` call site in `src/cmd/idea/main.go`, alongside the seven existing subcommands (`addCmd`, `listCmd`, `showCmd`, `doneCmd`, `reopenCmd`, `editCmd`, `rmCmd`).

#### Scenario: Single registration site

- **GIVEN** the source tree
- **WHEN** searching for `AddCommand(` in `src/cmd/idea/`
- **THEN** there is exactly one call site (in `main.go`)
- **AND** the call site contains `updateCmd()` as one of its arguments

## CLI: Code Structure

### Requirement: Cobra wrapper in `cmd/idea`

A new file `src/cmd/idea/update.go` SHALL define a factory function `updateCmd() *cobra.Command`. The factory SHALL build the cobra command struct (Use, Short, Args, RunE) and SHALL delegate all behavior to the internal package. The cobra `RunE` body SHALL be no longer than the equivalent of hop's `cmd/hop/update.go` body — flag wiring, error mapping, return.

#### Scenario: cmd file has no business logic

- **GIVEN** `src/cmd/idea/update.go`
- **WHEN** the file is reviewed
- **THEN** it contains only the cobra factory and an error-mapping `RunE`
- **AND** all subprocess invocations, version comparison, and Homebrew detection live in `internal/idea`

### Requirement: Logic in `internal/idea`

A new file `src/internal/idea/update.go` SHALL define an exported function with signature `Update(currentVersion string, out, errOut io.Writer) error`. This function SHALL house all business logic: Homebrew install detection, `brew update` invocation, latest-version query, version comparison, and `brew upgrade` invocation.

#### Scenario: Exported entry point

- **GIVEN** `src/internal/idea/update.go`
- **WHEN** the package is consumed from `cmd/idea/update.go`
- **THEN** only `idea.Update(...)` is called from the cobra wrapper — no other internal symbols leak into `cmd/`

### Requirement: Subprocess invocation via `os/exec`

Subprocess calls SHALL use the standard library `os/exec` package directly. No new internal package (e.g., a `proc` wrapper analogous to hop's `internal/proc`) SHALL be introduced. This decision aligns with the existing `os/exec` usage in `src/internal/idea/idea.go`.

#### Scenario: No new internal packages

- **GIVEN** the change is implemented
- **WHEN** `find src/internal -type d` is run
- **THEN** the only directory listed is `src/internal/idea/`
- **AND** no new sibling package was introduced

## CLI: `idea update` Behaviour

### Requirement: Non-Homebrew install path

When the running binary is NOT installed via Homebrew, `Update` SHALL print to `out` (in order):

1. `idea {currentVersion} was not installed via Homebrew.` (followed by newline)
2. `Update manually, or reinstall with: brew install sahil87/tap/idea` (followed by newline)

The function SHALL return `nil` without invoking `brew`. No output SHALL be written to `errOut` in this case.

The non-Homebrew detection SHALL be: the resolved real path of `os.Executable()` does NOT contain the substring `/Cellar/`. If `os.Executable()` or `filepath.EvalSymlinks` returns an error, the binary SHALL be treated as non-Homebrew (return false from `isBrewInstalled`).

#### Scenario: Locally-built binary

- **GIVEN** a `go build`-produced binary outside any Homebrew Cellar
- **WHEN** `idea update` is invoked
- **THEN** stdout contains `was not installed via Homebrew.`
- **AND** stdout contains `brew install sahil87/tap/idea`
- **AND** stderr is empty
- **AND** the process exits 0
- **AND** no `brew` subprocess is spawned

### Requirement: Homebrew install path — happy upgrade

When the running binary IS installed via Homebrew (real path contains `/Cellar/`) AND `brew` is on `PATH` AND the latest version differs from the current version, `Update` SHALL execute the following sequence, writing wrapper messages to `out`:

1. Print `Current version: {currentVersion}` then `Checking for updates...`
2. Run `brew update --quiet` with a 30-second timeout (`brewUpdateTimeout`). Subprocess output is captured (not streamed) per `os/exec.CommandContext.Output()`. On non-zero exit or context timeout, return a wrapped `brew update failed: ...` error.
3. Run `brew info --json=v2 sahil87/tap/idea` with a 30-second timeout (`brewInfoTimeout`). Parse JSON for `formulae[0].versions.stable`. If parsing fails or the path is empty, return a wrapped `could not determine latest version: ...` error.
4. Compare versions: strip a single leading `v` from each side (`normalizeVersion`), then compare for byte-equality. If equal, print `Already up to date ({currentVersion}).` and return `nil`.
5. Print `Updating {currentVersion} → v{latest}...`
6. Run `brew upgrade sahil87/tap/idea` with a 120-second timeout (`brewUpgradeTimeout`). The child process SHALL inherit the parent's stdin/stdout/stderr (`cmd.Stdin/Stdout/Stderr = os.Stdin/Stdout/Stderr`) so brew's progress output is visible to the user. On non-zero exit or context timeout, return a wrapped `brew upgrade failed: ...` error. If the child exits with a non-zero status code without an error, return an error of the form `brew upgrade exited with code {N}`.
7. Print `Updated to v{latest}.` and return `nil`.

#### Scenario: Brew install with newer version available

- **GIVEN** the binary is installed via Homebrew at `/opt/homebrew/Cellar/idea/0.0.1/bin/idea`
- **AND** `brew info --json=v2 sahil87/tap/idea` reports `versions.stable = "0.0.2"`
- **WHEN** `idea update` is invoked
- **THEN** stdout contains `Current version: v0.0.1`
- **AND** stdout contains `Checking for updates...`
- **AND** stdout contains `Updating v0.0.1 → v0.0.2...`
- **AND** stdout contains `Updated to v0.0.2.`
- **AND** the process exits 0

#### Scenario: Brew install already at latest

- **GIVEN** the binary's `currentVersion` is `v0.0.2`
- **AND** `brew info --json=v2 sahil87/tap/idea` reports `versions.stable = "0.0.2"`
- **WHEN** `idea update` is invoked
- **THEN** stdout contains `Already up to date (v0.0.2).`
- **AND** no `brew upgrade` is invoked
- **AND** the process exits 0

### Requirement: Brew not on PATH

If any subprocess invocation (`brew update`, `brew info`, or `brew upgrade`) returns an error wrapping `exec.ErrNotFound`, `Update` SHALL:

1. Write `idea update: brew not found on PATH.` to `errOut` (followed by newline)
2. Return the original `exec.ErrNotFound`-wrapping error unchanged so the caller can detect it via `errors.Is(err, exec.ErrNotFound)`

The cobra `RunE` in `cmd/idea/update.go` SHALL detect this case via `errors.Is(err, exec.ErrNotFound)` and SHALL return a package-local sentinel error (e.g., `errSilent`) so cobra's default error printer does NOT print a redundant secondary message. The binary SHALL exit with a non-zero status code.

#### Scenario: brew missing on PATH

- **GIVEN** `brew` is not present on `PATH`
- **AND** the binary is installed via Homebrew (so the non-Homebrew shortcut does not trigger)
- **WHEN** `idea update` is invoked
- **THEN** stderr contains `idea update: brew not found on PATH.`
- **AND** the message appears exactly once (no duplication from cobra's default error printer)
- **AND** the process exits non-zero

### Requirement: Constants and timeouts

The `internal/idea/update.go` file SHALL declare the following package-level constants:

```go
const brewFormula = "sahil87/tap/idea"

const (
    brewUpdateTimeout  = 30 * time.Second
    brewInfoTimeout    = 30 * time.Second
    brewUpgradeTimeout = 120 * time.Second
)
```

The fully-qualified formula name (`sahil87/tap/idea`) SHALL be used in every `brew` invocation that takes a formula argument. Bare `idea` SHALL NOT be used, to avoid collision with any same-named core formula.

#### Scenario: Formula reference is fully qualified

- **GIVEN** `src/internal/idea/update.go`
- **WHEN** searching for `brew info` and `brew upgrade` invocation arguments
- **THEN** every formula argument is `sahil87/tap/idea`
- **AND** no invocation passes a bare `idea`

### Requirement: Version-string normalization

The package SHALL define `func normalizeVersion(v string) string` that strips a single leading `"v"` and returns the rest unchanged. The function SHALL NOT perform semver parsing or any other transformation.

#### Scenario: Strip leading v

- **GIVEN** input `"v0.0.3"`
- **WHEN** `normalizeVersion` is called
- **THEN** the return value is `"0.0.3"`

#### Scenario: No leading v

- **GIVEN** input `"0.0.3"`
- **WHEN** `normalizeVersion` is called
- **THEN** the return value is `"0.0.3"`

#### Scenario: Empty string

- **GIVEN** input `""`
- **WHEN** `normalizeVersion` is called
- **THEN** the return value is `""`

#### Scenario: Lone v

- **GIVEN** input `"v"`
- **WHEN** `normalizeVersion` is called
- **THEN** the return value is `""`

#### Scenario: Multiple leading v's

- **GIVEN** input `"vvv1.0.0"`
- **WHEN** `normalizeVersion` is called
- **THEN** the return value is `"vv1.0.0"`
- **AND** only one leading `v` is stripped

### Requirement: I/O writer routing

The wrapper messages emitted by `Update` (e.g., `Current version: ...`, `Already up to date ...`, `Updated to ...`) SHALL be written to the `out io.Writer` parameter. Error hints (e.g., `idea update: brew not found on PATH.`) SHALL be written to the `errOut io.Writer` parameter. Subprocess stdout/stderr from `brew update` and `brew info` SHALL NOT be routed through `out`/`errOut` (they are captured for parsing or discarded). Subprocess streams from the `brew upgrade` foreground call SHALL inherit the parent process's `os.Stdin/Stdout/Stderr` directly, NOT the `out`/`errOut` parameters — this matches `hop update`'s split (small wrapper messages are redirectable for tests; large tty-aware brew progress is not).

#### Scenario: Test buffer captures wrapper messages only

- **GIVEN** a test calls `Update(version, &stdoutBuf, &stderrBuf)` on a non-brew binary
- **WHEN** the call returns
- **THEN** `stdoutBuf.String()` contains the "was not installed via Homebrew" hint
- **AND** `stderrBuf.String()` is empty

## Test Coverage

### Requirement: Unit tests in `internal/idea/update_test.go`

A new file `src/internal/idea/update_test.go` SHALL include the following test functions, all using table-driven style where multiple cases apply:

1. **`TestNormalizeVersion`** — table-driven cases mirroring hop's exact set: `("v0.0.3", "0.0.3")`, `("0.0.3", "0.0.3")`, `("", "")`, `("v", "")`, `("vvv1.0.0", "vv1.0.0")`. Test SHALL assert each call returns the expected string.

2. **`TestUpdateNonBrewInstall`** — assertion that when `isBrewInstalled()` returns false, `Update("v0.0.3", &stdout, &stderr)` returns `nil`, writes both expected hint lines to stdout, and writes nothing to stderr. Test SHALL skip itself with `t.Skip(...)` if the test binary happens to live under a Cellar directory (defensive — covers brew-installed-go developer machines).

3. **`TestIsBrewInstalledReturnsBool`** — smoke test that `isBrewInstalled()` does not panic. The test asserts only non-panicking behavior; the actual bool depends on the test environment.

These three tests mirror hop's `internal/update/update_test.go` 1:1.

#### Scenario: Tests run under existing harness

- **GIVEN** the test files exist
- **WHEN** `cd src && go test ./...` is run from a clean checkout
- **THEN** all tests pass
- **AND** `go test ./internal/idea/...` includes `TestNormalizeVersion`, `TestUpdateNonBrewInstall`, and `TestIsBrewInstalledReturnsBool` in its run set

#### Scenario: No subprocess required for non-brew test

- **GIVEN** the test environment has no `brew` installed and the test binary is outside `/Cellar/`
- **WHEN** `TestUpdateNonBrewInstall` runs
- **THEN** the test passes without spawning any subprocess
- **AND** no test invokes `brew update`, `brew info`, or `brew upgrade`

## Memory: Affected Files

### Requirement: Update `cli/structure.md`

`docs/memory/cli/structure.md` SHALL be updated during hydrate to:

1. Add `update.go` to the Layout listing of `cmd/idea/` files (alongside the existing `add.go list.go show.go done.go reopen.go edit.go rm.go resolve.go`)
2. Add `update.go` and `update_test.go` to the Layout listing of `internal/idea/` files
3. Cross-reference the new `cli/update.md` file from a relevant section (likely "Cross-references" or via inline link)

#### Scenario: Layout listing updated

- **GIVEN** hydrate has run
- **WHEN** `cli/structure.md` is read
- **THEN** the `cmd/idea/` Layout block lists `update.go` among the subcommand files
- **AND** the `internal/idea/` Layout block lists `update.go` and `update_test.go`

### Requirement: New `cli/update.md`

A new memory file SHALL be created at `docs/memory/cli/update.md` covering:

1. **Purpose** — one-paragraph description of the self-update command and when users invoke it
2. **Homebrew detection** — the `/Cellar/` substring rule, the path-resolution chain (`os.Executable` → `filepath.EvalSymlinks`), and the failure-mode (treat errors as non-Homebrew)
3. **Formula reference** — the fully-qualified `sahil87/tap/idea` form and the disambiguation rationale
4. **Version comparison** — single-leading-`v`-strip + string equality; explicit non-use of semver parsing
5. **Timeout constants** — `brewUpdateTimeout` (30s), `brewInfoTimeout` (30s), `brewUpgradeTimeout` (120s)
6. **I/O routing** — wrapper messages to `out`/`errOut` parameters; subprocess streams via inherited stdio for the foreground upgrade call
7. **Cross-reference** — link to `release/pipeline.md` (which owns the formula publication side) and `cli/structure.md` (for code placement)

#### Scenario: Memory file exists post-hydrate

- **GIVEN** hydrate has run
- **WHEN** `ls docs/memory/cli/` is run
- **THEN** `update.md` is listed alongside `structure.md`

### Requirement: Update memory index

`docs/memory/index.md` SHALL be updated during hydrate to add `cli/update.md` to the `cli` domain row's Memory Files column, joining `cli/structure.md`.

#### Scenario: Index lists both cli memory files

- **GIVEN** hydrate has run
- **WHEN** `docs/memory/index.md` is read
- **THEN** the `cli` domain row Memory Files column links to both `cli/structure.md` and `cli/update.md`

## Design Decisions

1. **Use `os/exec` directly, no `internal/proc` package**
   - *Why*: The existing `internal/idea/idea.go` already calls `os/exec` directly. Adding a thin wrapper just to mirror hop's structure adds a package, an indirection, and zero behavior. YAGNI applies. Constitution Dependency Discipline also prefers minimum scope.
   - *Rejected*: Introducing `src/internal/proc/` mirroring hop's wrapper. Hop's wrapper exists because hop has many `os/exec` call sites scattered across `internal/{clone,pull,push,sync,update}`; idea has only two call sites and neither benefits from indirection.

2. **Co-locate update logic in `internal/idea` rather than a new `internal/update` package**
   - *Why*: Hop has multiple internal packages because it has multiple distinct domains (clone, pull, push, sync, update). Idea today has exactly one internal domain (`internal/idea`). Splitting `update` into its own package solely to mirror hop's structure would over-fragment a still-small codebase. Constitution Principle IV is satisfied by "logic lives in `internal/idea`, not in `cmd/`" — it does not require one package per subcommand.
   - *Rejected*: A new `src/internal/update/` package. Adds an import path and a package boundary for ~100 lines of code with no shared API surface.

3. **Single-`v`-strip + string equality, no semver parsing**
   - *Why*: Both sides of the comparison originate from the same canonical artifact — the release tag. The CI workflow extracts the version from `refs/tags/vX.Y.Z` (stripping the `v`) and stamps it via `-ldflags`. `brew info --json=v2` reports the same string from the formula. Equality after one `v`-strip is correct because both sides are by-construction in the same shape.
   - *Rejected*: Semver parsing via `golang.org/x/mod/semver` or similar. Adds a dependency in violation of Constitution Dependency Discipline, and solves a problem (intransitive ordering) that does not exist for an equality check.

4. **Inherit os.Stdio for `brew upgrade` foreground call**
   - *Why*: Brew's upgrade output is large, tty-aware, and includes colored progress and prompt-style stages. Capturing it for replay would lose the live progress feel and force the user to wait silently for up to 120 seconds. Streaming is the right behavior for a long-running interactive subprocess.
   - *Rejected*: Streaming through `out`/`errOut` writers. The writers exist for redirection in tests and embedded callers; routing brew's tty output through them would couple test fixtures to real terminal capabilities and break the small-vs-large stream split that hop's docstring explicitly carves out.

5. **Use a `cmd/idea`-local `errSilent` sentinel for brew-not-found mapping**
   - *Why*: Cobra's default error printer prints `Error: <err>` to stderr automatically when `RunE` returns a non-nil error. The internal package already prints `idea update: brew not found on PATH.` itself. Without intercepting, the user would see the message twice. A package-local sentinel error returned from `RunE` lets `main.go`'s top-level error handler suppress the printer for this specific case.
   - *Rejected*: Setting `cmd.SilenceErrors = true` on the update subcommand only. That suppresses all errors from `update`, including legitimate ones like "brew upgrade failed". The sentinel pattern is selective.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Formula reference is `sahil87/tap/idea` (fully qualified) | Confirmed from intake #1 — verified in `docs/specs/overview.md` and `README.md` | S:95 R:95 A:95 D:95 |
| 2 | Certain | Logic lives in `internal/idea/update.go`, cobra wrapper in `cmd/idea/update.go` | Confirmed from intake #2 — Constitution Principle IV | S:95 R:90 A:95 D:95 |
| 3 | Certain | No new module dependencies — stdlib + cobra only | Confirmed from intake #3 — Constitution Dependency Discipline | S:95 R:95 A:95 D:95 |
| 4 | Certain | Subcommand wiring goes in `main.go`'s existing `root.AddCommand(...)` block | Confirmed from intake #4 | S:95 R:95 A:95 D:95 |
| 5 | Certain | Use `os/exec` directly (no new `internal/proc` package) | Confirmed from intake #5 (clarified — user confirmed) | S:95 R:75 A:80 D:80 |
| 6 | Certain | Strip single leading `v` and use string equality (no semver parsing) | Confirmed from intake #6 (clarified — user confirmed) | S:95 R:80 A:85 D:85 |
| 7 | Certain | Timeout constants copied from hop (30s update, 30s info, 120s upgrade) | Confirmed from intake #7 (clarified — user confirmed) | S:95 R:80 A:85 D:85 |
| 8 | Certain | `cobra.NoArgs`, Short = `"self-update the idea binary via Homebrew"` | Confirmed from intake #8 (clarified — user confirmed) | S:95 R:90 A:85 D:90 |
| 9 | Certain | New memory file `cli/update.md` (rather than appending to `structure.md`) | Confirmed from intake #9 (clarified — user confirmed) | S:95 R:80 A:80 D:75 |
| 10 | Certain | Cobra `RunE` maps brew-not-found error to a `cmd/idea`-local silent-exit sentinel | Confirmed from intake #10 (clarified — user confirmed); naming the sentinel `errSilent` mirrors hop's exact symbol name | S:95 R:75 A:80 D:80 |
| 11 | Certain | Tests mirror hop's `internal/update/update_test.go` (normalizeVersion table, isBrewInstalled smoke, non-brew code path) | Confirmed from intake #11 (clarified — user confirmed) | S:95 R:85 A:85 D:80 |
| 12 | Certain | Co-locate update logic in `internal/idea` (no new `internal/update` package) | Spec-stage analysis: idea has one internal domain today; splitting to mirror hop would over-fragment; Constitution Principle IV requires only that logic live in `internal/idea`, not one package per subcommand | S:90 R:80 A:85 D:85 |
| 13 | Certain | `brew update` and `brew info` use `exec.CommandContext(...).Output()` (captured); `brew upgrade` inherits parent stdio | Spec-stage analysis: matches hop's small-vs-large stream split (codified by hop's `proc.Run` vs `proc.RunForeground` distinction); brew's tty-aware progress requires inherited stdio | S:95 R:80 A:85 D:85 |
| 14 | Certain | Failure to resolve `os.Executable()` or `EvalSymlinks` means treat as non-Homebrew (return false from `isBrewInstalled`) | Spec-stage analysis: matches hop's defensive pattern; safer to skip self-update than to print misleading "brew" output when path resolution is broken | S:95 R:90 A:90 D:90 |
| 15 | Certain | Non-zero exit code from `brew upgrade` returns error of form `brew upgrade exited with code {N}` | Spec-stage analysis: matches hop's `proc.RunForeground` return-code propagation pattern; user gets a specific actionable signal vs. a generic "failed" | S:90 R:85 A:85 D:85 |

15 assumptions (15 certain, 0 confident, 0 tentative, 0 unresolved).
