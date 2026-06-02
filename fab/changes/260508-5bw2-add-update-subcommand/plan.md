# Plan: Add `idea update` Subcommand

**Change**: 260508-5bw2-add-update-subcommand
**Status**: In Progress
**Intake**: `intake.md`
**Spec**: `spec.md`

## Requirements

<!-- migrated from spec.md on 2026-06-02 -->

### Non-Goals

- Semver-aware version comparison — single-`v`-strip + string equality is sufficient because both sides come from the same release-tag canonical source.
- Auto-update on every CLI invocation, scheduled background checks, or telemetry.
- Non-Homebrew install channel handling beyond the existing "not installed via Homebrew" hint (apt, pacman, manual `scripts/install.sh` users see the hint).
- Rollback or version pinning — Homebrew handles those natively.

### CLI: `idea update` Subcommand

#### Requirement: Subcommand Registration

The `idea` root command SHALL register a top-level subcommand named `update`. The subcommand SHALL accept no positional arguments (`cobra.NoArgs`). The subcommand's `Short` description SHALL be `"self-update the idea binary via Homebrew"`. The subcommand SHALL inherit the root's persistent `--file` and `--main` flags but SHALL NOT consume them; both flags are irrelevant to self-update behaviour.

##### Scenario: Subcommand listed in help

- **GIVEN** the binary is built
- **WHEN** the user runs `idea --help`
- **THEN** the output lists `update` as an available command with the short description `"self-update the idea binary via Homebrew"`

##### Scenario: Rejects positional arguments

- **GIVEN** the user runs `idea update foo`
- **WHEN** cobra parses the command line
- **THEN** the command fails with cobra's standard "accepts 0 arg(s), received 1" error
- **AND** the binary exits non-zero

#### Requirement: Wiring Site

The `update` subcommand SHALL be registered via the existing `root.AddCommand(...)` call site in `src/cmd/idea/main.go`, alongside the seven existing subcommands (`addCmd`, `listCmd`, `showCmd`, `doneCmd`, `reopenCmd`, `editCmd`, `rmCmd`).

##### Scenario: Single registration site

- **GIVEN** the source tree
- **WHEN** searching for `AddCommand(` in `src/cmd/idea/`
- **THEN** there is exactly one call site (in `main.go`)
- **AND** the call site contains `updateCmd()` as one of its arguments

### CLI: Code Structure

#### Requirement: Cobra wrapper in `cmd/idea`

A new file `src/cmd/idea/update.go` SHALL define a factory function `updateCmd() *cobra.Command`. The factory SHALL build the cobra command struct (Use, Short, Args, RunE) and SHALL delegate all behavior to the internal package. The cobra `RunE` body SHALL be no longer than the equivalent of hop's `cmd/hop/update.go` body — flag wiring, error mapping, return.

##### Scenario: cmd file has no business logic

- **GIVEN** `src/cmd/idea/update.go`
- **WHEN** the file is reviewed
- **THEN** it contains only the cobra factory and an error-mapping `RunE`
- **AND** all subprocess invocations, version comparison, and Homebrew detection live in `internal/idea`

#### Requirement: Logic in `internal/idea`

A new file `src/internal/idea/update.go` SHALL define an exported function with signature `Update(currentVersion string, out, errOut io.Writer) error`. This function SHALL house all business logic: Homebrew install detection, `brew update` invocation, latest-version query, version comparison, and `brew upgrade` invocation.

##### Scenario: Exported entry point

- **GIVEN** `src/internal/idea/update.go`
- **WHEN** the package is consumed from `cmd/idea/update.go`
- **THEN** only `idea.Update(...)` is called from the cobra wrapper — no other internal symbols leak into `cmd/`

#### Requirement: Subprocess invocation via `os/exec`

Subprocess calls SHALL use the standard library `os/exec` package directly. No new internal package (e.g., a `proc` wrapper analogous to hop's `internal/proc`) SHALL be introduced. This decision aligns with the existing `os/exec` usage in `src/internal/idea/idea.go`.

##### Scenario: No new internal packages

- **GIVEN** the change is implemented
- **WHEN** `find src/internal -type d` is run
- **THEN** the only directory listed is `src/internal/idea/`
- **AND** no new sibling package was introduced

### CLI: `idea update` Behaviour

#### Requirement: Non-Homebrew install path

When the running binary is NOT installed via Homebrew, `Update` SHALL print to `out` (in order):

1. `idea {currentVersion} was not installed via Homebrew.` (followed by newline)
2. `Update manually, or reinstall with: brew install sahil87/tap/idea` (followed by newline)

The function SHALL return `nil` without invoking `brew`. No output SHALL be written to `errOut` in this case.

The non-Homebrew detection SHALL be: the resolved real path of `os.Executable()` does NOT contain the substring `/Cellar/`. If `os.Executable()` or `filepath.EvalSymlinks` returns an error, the binary SHALL be treated as non-Homebrew (return false from `isBrewInstalled`).

##### Scenario: Locally-built binary

- **GIVEN** a `go build`-produced binary outside any Homebrew Cellar
- **WHEN** `idea update` is invoked
- **THEN** stdout contains `was not installed via Homebrew.`
- **AND** stdout contains `brew install sahil87/tap/idea`
- **AND** stderr is empty
- **AND** the process exits 0
- **AND** no `brew` subprocess is spawned

#### Requirement: Homebrew install path — happy upgrade

When the running binary IS installed via Homebrew (real path contains `/Cellar/`) AND `brew` is on `PATH` AND the latest version differs from the current version, `Update` SHALL execute the following sequence, writing wrapper messages to `out`:

1. Print `Current version: {currentVersion}` then `Checking for updates...`
2. Run `brew update --quiet` with a 30-second timeout (`brewUpdateTimeout`). Subprocess output is captured (not streamed) per `os/exec.CommandContext.Output()`. On non-zero exit or context timeout, return a wrapped `brew update failed: ...` error.
3. Run `brew info --json=v2 sahil87/tap/idea` with a 30-second timeout (`brewInfoTimeout`). Parse JSON for `formulae[0].versions.stable`. If parsing fails or the path is empty, return a wrapped `could not determine latest version: ...` error.
4. Compare versions: strip a single leading `v` from each side (`normalizeVersion`), then compare for byte-equality. If equal, print `Already up to date ({currentVersion}).` and return `nil`.
5. Print `Updating {currentVersion} → v{latest}...`
6. Run `brew upgrade sahil87/tap/idea` with a 120-second timeout (`brewUpgradeTimeout`). The child process SHALL inherit the parent's stdin/stdout/stderr (`cmd.Stdin/Stdout/Stderr = os.Stdin/Stdout/Stderr`) so brew's progress output is visible to the user. On non-zero exit or context timeout, return a wrapped `brew upgrade failed: ...` error. If the child exits with a non-zero status code without an error, return an error of the form `brew upgrade exited with code {N}`.
7. Print `Updated to v{latest}.` and return `nil`.

##### Scenario: Brew install with newer version available

- **GIVEN** the binary is installed via Homebrew at `/opt/homebrew/Cellar/idea/0.0.1/bin/idea`
- **AND** `brew info --json=v2 sahil87/tap/idea` reports `versions.stable = "0.0.2"`
- **WHEN** `idea update` is invoked
- **THEN** stdout contains `Current version: v0.0.1`
- **AND** stdout contains `Checking for updates...`
- **AND** stdout contains `Updating v0.0.1 → v0.0.2...`
- **AND** stdout contains `Updated to v0.0.2.`
- **AND** the process exits 0

##### Scenario: Brew install already at latest

- **GIVEN** the binary's `currentVersion` is `v0.0.2`
- **AND** `brew info --json=v2 sahil87/tap/idea` reports `versions.stable = "0.0.2"`
- **WHEN** `idea update` is invoked
- **THEN** stdout contains `Already up to date (v0.0.2).`
- **AND** no `brew upgrade` is invoked
- **AND** the process exits 0

#### Requirement: Brew not on PATH

If any subprocess invocation (`brew update`, `brew info`, or `brew upgrade`) returns an error wrapping `exec.ErrNotFound`, `Update` SHALL:

1. Write `idea update: brew not found on PATH.` to `errOut` (followed by newline)
2. Return the original `exec.ErrNotFound`-wrapping error unchanged so the caller can detect it via `errors.Is(err, exec.ErrNotFound)`

The cobra `RunE` in `cmd/idea/update.go` SHALL detect this case via `errors.Is(err, exec.ErrNotFound)` and SHALL return a package-local sentinel error (e.g., `errSilent`) so cobra's default error printer does NOT print a redundant secondary message. The binary SHALL exit with a non-zero status code.

##### Scenario: brew missing on PATH

- **GIVEN** `brew` is not present on `PATH`
- **AND** the binary is installed via Homebrew (so the non-Homebrew shortcut does not trigger)
- **WHEN** `idea update` is invoked
- **THEN** stderr contains `idea update: brew not found on PATH.`
- **AND** the message appears exactly once (no duplication from cobra's default error printer)
- **AND** the process exits non-zero

#### Requirement: Constants and timeouts

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

##### Scenario: Formula reference is fully qualified

- **GIVEN** `src/internal/idea/update.go`
- **WHEN** searching for `brew info` and `brew upgrade` invocation arguments
- **THEN** every formula argument is `sahil87/tap/idea`
- **AND** no invocation passes a bare `idea`

#### Requirement: Version-string normalization

The package SHALL define `func normalizeVersion(v string) string` that strips a single leading `"v"` and returns the rest unchanged. The function SHALL NOT perform semver parsing or any other transformation.

##### Scenario: Strip leading v

- **GIVEN** input `"v0.0.3"`
- **WHEN** `normalizeVersion` is called
- **THEN** the return value is `"0.0.3"`

##### Scenario: No leading v

- **GIVEN** input `"0.0.3"`
- **WHEN** `normalizeVersion` is called
- **THEN** the return value is `"0.0.3"`

##### Scenario: Empty string

- **GIVEN** input `""`
- **WHEN** `normalizeVersion` is called
- **THEN** the return value is `""`

##### Scenario: Lone v

- **GIVEN** input `"v"`
- **WHEN** `normalizeVersion` is called
- **THEN** the return value is `""`

##### Scenario: Multiple leading v's

- **GIVEN** input `"vvv1.0.0"`
- **WHEN** `normalizeVersion` is called
- **THEN** the return value is `"vv1.0.0"`
- **AND** only one leading `v` is stripped

#### Requirement: I/O writer routing

The wrapper messages emitted by `Update` (e.g., `Current version: ...`, `Already up to date ...`, `Updated to ...`) SHALL be written to the `out io.Writer` parameter. Error hints (e.g., `idea update: brew not found on PATH.`) SHALL be written to the `errOut io.Writer` parameter. Subprocess stdout/stderr from `brew update` and `brew info` SHALL NOT be routed through `out`/`errOut` (they are captured for parsing or discarded). Subprocess streams from the `brew upgrade` foreground call SHALL inherit the parent process's `os.Stdin/Stdout/Stderr` directly, NOT the `out`/`errOut` parameters — this matches `hop update`'s split (small wrapper messages are redirectable for tests; large tty-aware brew progress is not).

##### Scenario: Test buffer captures wrapper messages only

- **GIVEN** a test calls `Update(version, &stdoutBuf, &stderrBuf)` on a non-brew binary
- **WHEN** the call returns
- **THEN** `stdoutBuf.String()` contains the "was not installed via Homebrew" hint
- **AND** `stderrBuf.String()` is empty

### Test Coverage

#### Requirement: Unit tests in `internal/idea/update_test.go`

A new file `src/internal/idea/update_test.go` SHALL include the following test functions, all using table-driven style where multiple cases apply:

1. **`TestNormalizeVersion`** — table-driven cases mirroring hop's exact set: `("v0.0.3", "0.0.3")`, `("0.0.3", "0.0.3")`, `("", "")`, `("v", "")`, `("vvv1.0.0", "vv1.0.0")`. Test SHALL assert each call returns the expected string.

2. **`TestUpdateNonBrewInstall`** — assertion that when `isBrewInstalled()` returns false, `Update("v0.0.3", &stdout, &stderr)` returns `nil`, writes both expected hint lines to stdout, and writes nothing to stderr. Test SHALL skip itself with `t.Skip(...)` if the test binary happens to live under a Cellar directory (defensive — covers brew-installed-go developer machines).

3. **`TestIsBrewInstalledReturnsBool`** — smoke test that `isBrewInstalled()` does not panic. The test asserts only non-panicking behavior; the actual bool depends on the test environment.

These three tests mirror hop's `internal/update/update_test.go` 1:1.

##### Scenario: Tests run under existing harness

- **GIVEN** the test files exist
- **WHEN** `cd src && go test ./...` is run from a clean checkout
- **THEN** all tests pass
- **AND** `go test ./internal/idea/...` includes `TestNormalizeVersion`, `TestUpdateNonBrewInstall`, and `TestIsBrewInstalledReturnsBool` in its run set

##### Scenario: No subprocess required for non-brew test

- **GIVEN** the test environment has no `brew` installed and the test binary is outside `/Cellar/`
- **WHEN** `TestUpdateNonBrewInstall` runs
- **THEN** the test passes without spawning any subprocess
- **AND** no test invokes `brew update`, `brew info`, or `brew upgrade`

### Memory: Affected Files

#### Requirement: Update `cli/structure.md`

`docs/memory/cli/structure.md` SHALL be updated during hydrate to:

1. Add `update.go` to the Layout listing of `cmd/idea/` files (alongside the existing `add.go list.go show.go done.go reopen.go edit.go rm.go resolve.go`)
2. Add `update.go` and `update_test.go` to the Layout listing of `internal/idea/` files
3. Cross-reference the new `cli/update.md` file from a relevant section (likely "Cross-references" or via inline link)

##### Scenario: Layout listing updated

- **GIVEN** hydrate has run
- **WHEN** `cli/structure.md` is read
- **THEN** the `cmd/idea/` Layout block lists `update.go` among the subcommand files
- **AND** the `internal/idea/` Layout block lists `update.go` and `update_test.go`

#### Requirement: New `cli/update.md`

A new memory file SHALL be created at `docs/memory/cli/update.md` covering:

1. **Purpose** — one-paragraph description of the self-update command and when users invoke it
2. **Homebrew detection** — the `/Cellar/` substring rule, the path-resolution chain (`os.Executable` → `filepath.EvalSymlinks`), and the failure-mode (treat errors as non-Homebrew)
3. **Formula reference** — the fully-qualified `sahil87/tap/idea` form and the disambiguation rationale
4. **Version comparison** — single-leading-`v`-strip + string equality; explicit non-use of semver parsing
5. **Timeout constants** — `brewUpdateTimeout` (30s), `brewInfoTimeout` (30s), `brewUpgradeTimeout` (120s)
6. **I/O routing** — wrapper messages to `out`/`errOut` parameters; subprocess streams via inherited stdio for the foreground upgrade call
7. **Cross-reference** — link to `release/pipeline.md` (which owns the formula publication side) and `cli/structure.md` (for code placement)

##### Scenario: Memory file exists post-hydrate

- **GIVEN** hydrate has run
- **WHEN** `ls docs/memory/cli/` is run
- **THEN** `update.md` is listed alongside `structure.md`

#### Requirement: Update memory index

`docs/memory/index.md` SHALL be updated during hydrate to add `cli/update.md` to the `cli` domain row's Memory Files column, joining `cli/structure.md`.

##### Scenario: Index lists both cli memory files

- **GIVEN** hydrate has run
- **WHEN** `docs/memory/index.md` is read
- **THEN** the `cli` domain row Memory Files column links to both `cli/structure.md` and `cli/update.md`

### Design Decisions

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

## Tasks

### Phase 1: Setup

(No setup required — no new dependencies, no scaffolding files. The internal package and cmd directory both already exist.)

### Phase 2: Core Implementation

- [x] T001 Create `src/internal/idea/update.go` with the package-level `brewFormula` constant (`sahil87/tap/idea`) and the three timeout constants (`brewUpdateTimeout`, `brewInfoTimeout`, `brewUpgradeTimeout` — all `time.Duration`, values 30s/30s/120s). Add the file header comment describing the package's responsibility and the formula-name disambiguation rationale (mirror hop's `internal/update/update.go` header). Include imports for `context`, `encoding/json`, `errors`, `fmt`, `io`, `os`, `os/exec`, `path/filepath`, `strings`, `time`.

- [x] T002 In `src/internal/idea/update.go`, implement `func normalizeVersion(v string) string` that strips a single leading `"v"` using `strings.TrimPrefix(v, "v")`. No semver parsing.

- [x] T003 In `src/internal/idea/update.go`, implement `func isBrewInstalled() bool`. Logic: call `os.Executable()`; if error, return false. Call `filepath.EvalSymlinks(self)`; if error, return false. Return `strings.Contains(real, "/Cellar/")`.

- [x] T004 In `src/internal/idea/update.go`, implement `func brewLatestVersion() (string, error)`. Build a 30-second timeout context (`brewInfoTimeout`). Run `exec.CommandContext(ctx, "brew", "info", "--json=v2", brewFormula).Output()`. On error, return the error unchanged (callers will check `errors.Is(err, exec.ErrNotFound)`). Parse the captured stdout as JSON into a struct with shape `{ Formulae []{ Versions struct { Stable string `json:"stable"` } `json:"versions"` } `json:"formulae"` }`. If the formulae array is empty or `Stable` is empty, return `errors.New("no stable version found in brew info output")`. Otherwise return `info.Formulae[0].Versions.Stable`.

- [x] T005 In `src/internal/idea/update.go`, implement the public function `func Update(currentVersion string, out, errOut io.Writer) error` per the spec's "Homebrew install path — happy upgrade" and "Non-Homebrew install path" requirements. Sequence:
  1. If `!isBrewInstalled()`: print the two-line non-Homebrew hint to `out` (`idea {currentVersion} was not installed via Homebrew.` then `Update manually, or reinstall with: brew install sahil87/tap/idea`), return nil.
  2. Print `Current version: {currentVersion}` and `Checking for updates...` to `out`.
  3. Run `exec.CommandContext(ctx, "brew", "update", "--quiet").Run()` with `brewUpdateTimeout` context. If error wraps `exec.ErrNotFound`, write `idea update: brew not found on PATH.` to `errOut` and return the original error. Otherwise on error return `fmt.Errorf("brew update failed: %w", err)`.
  4. Call `brewLatestVersion()`. Same `exec.ErrNotFound` handling. Other errors → `fmt.Errorf("could not determine latest version: %w", err)`.
  5. If `normalizeVersion(latest) == normalizeVersion(currentVersion)`: print `Already up to date ({currentVersion}).` to `out`, return nil.
  6. Print `Updating {currentVersion} → v{normalizeVersion(latest)}...` to `out`.
  7. Build a 120-second context (`brewUpgradeTimeout`). Construct `cmd := exec.CommandContext(upCtx, "brew", "upgrade", brewFormula)`. Set `cmd.Stdin = os.Stdin`, `cmd.Stdout = os.Stdout`, `cmd.Stderr = os.Stderr`. Call `cmd.Run()`. Same `exec.ErrNotFound` handling. Other errors: if the error is an `*exec.ExitError` with a non-zero exit code, return `fmt.Errorf("brew upgrade exited with code %d", exitErr.ExitCode())`; otherwise return `fmt.Errorf("brew upgrade failed: %w", err)`.
  8. Print `Updated to v{normalizeVersion(latest)}.` to `out`, return nil.

- [x] T006 [P] Create `src/internal/idea/update_test.go` with three test functions:
  1. `TestNormalizeVersion` — table-driven, cases `("v0.0.3", "0.0.3")`, `("0.0.3", "0.0.3")`, `("", "")`, `("v", "")`, `("vvv1.0.0", "vv1.0.0")`. Mirror hop's exact set verbatim.
  2. `TestUpdateNonBrewInstall` — call `Update("v0.0.3", &stdout, &stderr)`. Skip via `t.Skip(...)` if `isBrewInstalled()` returns true. Assert no error returned, stdout contains both `"v0.0.3 was not installed via Homebrew"` and `"brew install sahil87/tap/idea"`, stderr is empty.
  3. `TestIsBrewInstalledReturnsBool` — smoke test that just calls `_ = isBrewInstalled()` and asserts it doesn't panic.

  Use `package idea` (not `idea_test`). No mocks. No subprocess invocations beyond the trivial ones already exercised by stdlib `os/exec` (none of these tests should actually spawn `brew`).

- [x] T007 Create `src/cmd/idea/update.go` defining `func updateCmd() *cobra.Command`. Imports: `errors`, `os/exec`, `github.com/spf13/cobra`, and `github.com/sahil87/idea/internal/idea` (the existing internal package). The command struct has `Use: "update"`, `Short: "self-update the idea binary via Homebrew"`, `Args: cobra.NoArgs`, and a `RunE` body that:
  1. Calls `idea.Update(version, cmd.OutOrStdout(), cmd.ErrOrStderr())` (where `version` is the package-level `var version` from `main.go`).
  2. If the returned error is non-nil and `errors.Is(err, exec.ErrNotFound)`, returns `errSilent` (the package-local sentinel — see T008).
  3. Otherwise returns `err` unchanged.

- [x] T008 In `src/cmd/idea/update.go` (or `main.go` if more idiomatic), declare a package-level sentinel `var errSilent = errors.New("silent")`. In `main.go`'s top-level error handler at the bottom of `main()`, intercept the sentinel: if `errors.Is(err, errSilent)`, exit with status 1 WITHOUT printing the `ERROR: %s` line. The current handler is `fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)` followed by `os.Exit(1)` — restructure to a switch / `if errors.Is(...)` branch that suppresses printing for the sentinel. Ensure other errors continue to be printed exactly as today.

### Phase 3: Integration & Edge Cases

- [x] T009 In `src/cmd/idea/main.go`, add `updateCmd()` to the existing `root.AddCommand(...)` call. The list currently contains seven entries; add `updateCmd()` as the eighth (placement at the end of the list is fine — order in `AddCommand` does not affect help-output ordering).

- [x] T010 Run `cd src && go build ./cmd/idea` from the repo root. Verify it compiles with no errors. If imports are missing or unused, fix and re-run until clean.

- [x] T011 Run `cd src && go test ./internal/idea/...` to verify the three new tests pass. If `TestUpdateNonBrewInstall` skips (because the test binary happens to live under `/Cellar/`), that is acceptable — the skip path is intentional. The other two tests MUST pass.

- [x] T012 Run `cd src && go test ./...` to verify the full test suite still passes (no regressions in the existing `cmd/idea/main_test.go` integration suite).

- [x] T013 Smoke-test the binary manually: run `just build` (which calls `scripts/build.sh` and produces `bin/idea` at the repo root via `go build -ldflags "-X main.version=..." -o ../bin/idea ./cmd/idea` from inside `src/`). Then run `./bin/idea update`. Verify stdout contains `was not installed via Homebrew` and `brew install sahil87/tap/idea`, stderr is empty, exit code is 0. <!-- clarified: build command corrected to match scripts/build.sh — output is bin/idea at repo root, not src/idea -->

- [x] T014 Smoke-test help output: run `./bin/idea --help`. Verify `update` is listed under "Available Commands" with the description `"self-update the idea binary via Homebrew"`. <!-- clarified: binary path corrected from ./src/idea to ./bin/idea (matches scripts/build.sh output) -->

### Phase 4: Polish

(Memory updates are handled by hydrate via `/fab-continue`, not by tasks. The spec's "Memory: Affected Files" requirements drive that step.)

---

## Execution Order

- T001 (file scaffolding + constants) → blocks T002, T003, T004, T005 (all add functions to the same file).
- T002, T003, T004 are sequential within `update.go` because they all touch the same file (no `[P]` between them — but they are independent in semantics; the constraint is purely the single-file edit serialization).
- T005 depends on T002, T003, T004 (uses `normalizeVersion`, `isBrewInstalled`, `brewLatestVersion`).
- T006 (test file) depends on T001-T005 being committed because the tests reference exported (`Update`) and unexported (`normalizeVersion`, `isBrewInstalled`) symbols. Marked `[P]` because it can be drafted alongside T005 if the developer is comfortable; in practice, write T006 last.
- T007 (cmd wrapper) depends on T005 (calls `idea.Update`).
- T008 (sentinel + main.go interception) is independent of T007 file-wise but is needed before T007's `RunE` can compile (T007 references `errSilent`). Implement T008 first or together with T007.
- T009 (wire into AddCommand) depends on T007.
- T010 (compile) depends on T001-T009.
- T011, T012 (test) depend on T010.
- T013, T014 (smoke) depend on T010.

## Acceptance

## Functional Completeness

- [ ] CHK-001 Subcommand Registration: `idea --help` lists `update` with the exact short text `"self-update the idea binary via Homebrew"`.
- [ ] CHK-002 Subcommand Registration: `idea update foo` exits non-zero with cobra's "accepts 0 arg(s)" error.
- [ ] CHK-003 Wiring Site: `grep -r "AddCommand(" src/cmd/idea/` returns exactly one match (in `main.go`) and that line includes `updateCmd()`.
- [ ] CHK-004 Cobra wrapper in `cmd/idea`: `src/cmd/idea/update.go` exists; the `RunE` body contains only the call to `idea.Update`, the `errors.Is(err, exec.ErrNotFound)` mapping to `errSilent`, and a return.
- [ ] CHK-005 Logic in `internal/idea`: `src/internal/idea/update.go` exists and exports `Update(currentVersion string, out, errOut io.Writer) error`.
- [ ] CHK-006 Subprocess invocation via `os/exec`: `find src/internal -mindepth 1 -maxdepth 1 -type d` lists only `src/internal/idea` (no new sibling package).
- [ ] CHK-007 Non-Homebrew install path: `Update` returns nil and writes the two-line hint to `out` (no `errOut`, no `brew` subprocess) when `isBrewInstalled()` is false.
- [ ] CHK-008 Homebrew install path — happy upgrade: when `isBrewInstalled()` is true and latest != current, `Update` runs `brew update --quiet`, calls `brewLatestVersion()`, runs `brew upgrade sahil87/tap/idea` with inherited stdio, and prints `Updating ... → v...` and `Updated to v...`.
- [ ] CHK-009 Homebrew install path — already current: when latest == current after `normalizeVersion`, `Update` prints `Already up to date (...)` and returns nil without spawning `brew upgrade`.
- [ ] CHK-010 Brew not on PATH: when any subprocess returns an `exec.ErrNotFound`-wrapping error, `Update` writes `idea update: brew not found on PATH.` to `errOut` exactly once and returns the original error; the cobra `RunE` maps it to `errSilent` and `main.go` exits non-zero without printing `ERROR:`.
- [ ] CHK-011 Constants and timeouts: `update.go` declares `brewFormula = "sahil87/tap/idea"`, `brewUpdateTimeout = 30*time.Second`, `brewInfoTimeout = 30*time.Second`, `brewUpgradeTimeout = 120*time.Second` as package-level constants, and every `brew info`/`brew upgrade` call uses `brewFormula`.
- [ ] CHK-012 Version-string normalization: `normalizeVersion` strips at most one leading `v` and performs no other transformation.
- [ ] CHK-013 I/O writer routing: subprocess streams from `brew update` and `brew info` are not routed through `out`/`errOut`; `brew upgrade` inherits parent `os.Stdin/Stdout/Stderr`; only wrapper messages reach `out`/`errOut`.

## Behavioral Correctness

- [ ] CHK-014 The existing `main.go` top-level error handler suppresses the `ERROR: %s` print only for `errors.Is(err, errSilent)` — every other error type continues to print exactly as before.

## Scenario Coverage

- [ ] CHK-015 Scenario "Subcommand listed in help": exercised by manual smoke (T014) — confirms `update` appears in `idea --help` with the exact short text.
- [ ] CHK-016 Scenario "Rejects positional arguments": coverable by integration test or smoke; the cobra `NoArgs` declaration plus standard cobra behavior is sufficient verification.
- [ ] CHK-017 Scenario "Locally-built binary": exercised by `TestUpdateNonBrewInstall` (which only runs when the test binary is outside `/Cellar/`).
- [ ] CHK-018 Scenario "Brew install with newer version available": NOT exercised by automated tests (would require live Homebrew). Mark as **N/A** if smoke-tested manually on a brew-installed binary, or document why deferred.
- [ ] CHK-019 Scenario "Brew install already at latest": NOT exercised by automated tests (live Homebrew dependency). Same handling as CHK-018.
- [ ] CHK-020 Scenario "brew missing on PATH": NOT exercised by automated tests (would require manipulating `PATH` mid-process). Document as out-of-scope-for-tests but verifiable manually.
- [ ] CHK-021 Scenario "Strip leading v" / "No leading v" / "Empty string" / "Lone v" / "Multiple leading v's": all five cases exercised by `TestNormalizeVersion`.
- [ ] CHK-022 Scenario "Test buffer captures wrapper messages only": exercised by `TestUpdateNonBrewInstall`.
- [ ] CHK-023 Scenario "Tests run under existing harness": `cd src && go test ./...` passes from a clean checkout.
- [ ] CHK-024 Scenario "No subprocess required for non-brew test": `TestUpdateNonBrewInstall` does not spawn `brew` (verifiable by running with `brew` absent from `PATH`).

## Edge Cases & Error Handling

- [ ] CHK-025 `os.Executable()` error path: `isBrewInstalled` returns false (treats binary as non-Homebrew) — covered by the early-return in T003.
- [ ] CHK-026 `filepath.EvalSymlinks` error path: `isBrewInstalled` returns false — same defensive pattern.
- [ ] CHK-027 `brew info` returns empty `formulae` array: `brewLatestVersion` returns `errors.New("no stable version found in brew info output")`.
- [ ] CHK-028 `brew info` returns non-empty `formulae` but empty `versions.stable`: same error as CHK-027.
- [ ] CHK-029 `brew upgrade` exits non-zero (e.g., conflict): error returned matches `brew upgrade exited with code {N}` format.
- [ ] CHK-030 `brew update`/`brew info`/`brew upgrade` context timeout: caller sees a wrapped error containing the originating subcommand name.

## Code Quality

- [ ] CHK-031 Pattern consistency: New code in `cmd/idea/update.go` and `internal/idea/update.go` follows the naming, struct-literal, and error-wrapping style of the surrounding files (compare to `cmd/idea/add.go` and `internal/idea/idea.go`).
- [ ] CHK-032 No unnecessary duplication: The implementation reuses stdlib `os/exec`, does not introduce a new `internal/proc`-style wrapper, and does not duplicate the `version` package-level variable from `main.go`.
- [ ] CHK-033 Readability and maintainability over cleverness (Project principle): function bodies are linear, with clear early-return guards; no nested closures or reflection.
- [ ] CHK-034 Follow existing project patterns (Project principle): the cobra factory `updateCmd()` mirrors the shape of `addCmd()`/`listCmd()` etc; the `internal/idea` exported function mirrors how existing exported APIs in the package are shaped.
- [ ] CHK-035 No god functions (>50 lines without clear reason): `Update`'s body is sequential by design; if it exceeds 50 lines, split into `runBrewUpdate`, `runBrewUpgrade` helpers — only if needed.
- [ ] CHK-036 No magic strings or numbers: all literals (formula name, timeouts, hint text) are package-level constants or single-use literals with self-explanatory inline values.
- [ ] CHK-037 Constitution Principle III (Cobra-Idiomatic CLI Surface): `updateCmd()` is a `*cobra.Command` factory; root command's bare-text shorthand and persistent flags untouched.
- [ ] CHK-038 Constitution Principle IV (Logic in `internal/idea`): the cobra `RunE` body contains only flag wiring, error mapping, and a return — no parsing, no subprocess invocation, no version comparison.
- [ ] CHK-039 Constitution Principle V (Table-driven tests, no filesystem mocks): `TestNormalizeVersion` is table-driven; `TestUpdateNonBrewInstall` uses real I/O writers (no mocks).
- [ ] CHK-040 Constitution Test Integrity: tests conform to spec — when adjusting tests, the change is "test now matches spec" (not "spec changed to match test").
- [ ] CHK-041 Constitution Build Reproducibility: no new build-time codegen, no embedded timestamps, no env-var-dependent behavior at build time.
- [ ] CHK-042 Constitution Dependency Discipline: `go.mod` shows no new direct dependencies after this change.

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-018 **N/A**: live Homebrew unavailable in CI`
