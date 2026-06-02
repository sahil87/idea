# Plan: Add --skip-brew-update flag to update command

**Change**: 260531-t2ov-skip-brew-update-flag
**Status**: In Progress
**Intake**: `intake.md`
**Spec**: `spec.md`

## Requirements

<!-- migrated from spec.md on 2026-06-02 -->

### Non-Goals

- Skipping the `brew info` version check — out of scope; the flag suppresses only the tap-metadata refresh.
- Skipping the up-to-date short-circuit or `brew upgrade` — these always run as today.
- A persistent/global flag — the flag is local to `update` (see Constitution III).
- Refactoring the subprocess invocation convention (no `internal/proc` wrapper, no command-runner interface). Only the minimal seam needed for testing is introduced.
- Auto-detecting "brew was already updated recently" — the flag is an explicit, caller-driven opt-out.

### CLI: `--skip-brew-update` flag

#### Requirement: Flag definition
The `update` subcommand MUST define a local boolean flag named exactly `--skip-brew-update`, defaulting to `false`, with a short usage description stating it skips the internal `brew update` tap-metadata refresh. The flag MUST be a real cobra flag wired on the `update` command (not a root persistent flag). Its value MUST be passed into `idea.Update(...)`.

##### Scenario: Flag is registered on update
- **GIVEN** the compiled `idea` binary
- **WHEN** the user runs `idea update --help`
- **THEN** the help output SHALL list `--skip-brew-update` with its description

##### Scenario: Flag rejects values it shouldn't take
- **GIVEN** a boolean flag
- **WHEN** the user runs `idea update --skip-brew-update`
- **THEN** the flag SHALL parse as `true` with no required value argument

#### Requirement: Default behavior unchanged
When `--skip-brew-update` is absent, `Update` MUST behave exactly as before this change: it runs `brew update --quiet`, then `brew info`, then the up-to-date short-circuit, then `brew upgrade` when an upgrade is available.

##### Scenario: Default path runs brew update
- **GIVEN** a Homebrew-installed binary and the flag absent (`skipBrewUpdate == false`)
- **WHEN** `Update` runs down the brew path
- **THEN** `brew update --quiet` SHALL be invoked
- **AND** `brew info --json=v2 sahil87/tap/idea` SHALL be invoked
- **AND** `brew upgrade sahil87/tap/idea` SHALL be invoked when the versions differ

### internal/idea: `Update` signature and skip behavior

#### Requirement: Threaded boolean parameter
`Update` MUST accept a `skipBrewUpdate bool` parameter threaded from the cobra layer. The signature SHALL be `func Update(currentVersion string, skipBrewUpdate bool, out, errOut io.Writer) error`. The parameter is placed after `currentVersion` and before the writers, grouping inputs ahead of output sinks.

##### Scenario: Single caller updated
- **GIVEN** `Update` has exactly one caller (`src/cmd/idea/update.go`)
- **WHEN** the signature changes
- **THEN** the caller SHALL pass the flag value and the package SHALL compile

#### Requirement: Skip guards only the brew update block
When `skipBrewUpdate` is `true`, `Update` MUST NOT spawn `brew update`. Control MUST flow directly to the `brew info` version check. The `brew info` call, the up-to-date short-circuit, and the `brew upgrade` call MUST be unaffected by the flag. The non-Homebrew-install short-circuit at the top of `Update` MUST be unaffected.

##### Scenario: Skip omits brew update but still upgrades
- **GIVEN** a Homebrew-installed binary and `skipBrewUpdate == true`
- **WHEN** `Update` runs down the brew path with a newer version available
- **THEN** `brew update` SHALL NOT be invoked
- **AND** `brew info --json=v2 sahil87/tap/idea` SHALL be invoked
- **AND** `brew upgrade sahil87/tap/idea` SHALL be invoked

##### Scenario: Skip with up-to-date version
- **GIVEN** a Homebrew-installed binary and `skipBrewUpdate == true`
- **WHEN** `Update` runs and the installed version equals the latest
- **THEN** `brew update` SHALL NOT be invoked
- **AND** `brew info` SHALL be invoked
- **AND** the "Already up to date" short-circuit SHALL fire
- **AND** `brew upgrade` SHALL NOT be invoked

#### Requirement: Output routing preserved
The change MUST NOT alter I/O routing. `brew update` and `brew info` streams remain captured; `brew upgrade` continues to inherit `os.Stdin`/`os.Stdout`/`os.Stderr`. Wrapper messages continue to write to `out`/`errOut`. The `errSilent` mapping in `RunE` is unchanged.

##### Scenario: Upgrade still inherits stdio
- **GIVEN** the skip flag in either state
- **WHEN** `brew upgrade` is invoked
- **THEN** it SHALL inherit the process stdio exactly as before

### Testing: assert flag behavior without refactoring the convention

#### Requirement: Test the skip behavior via the repo's table-driven pattern
A test in `src/internal/idea/update_test.go` MUST assert that with `skipBrewUpdate == true`, `brew update` is not invoked while `brew info` and `brew upgrade` are; and with `skipBrewUpdate == false`, `brew update` is invoked. The test MUST follow the repo's table-driven style (Constitution V) and use real-process or recorded-argv observation rather than mocking interfaces. It MUST NOT require a real Homebrew installation and MUST be stable in CI.

##### Scenario: Recorded invocations differ by flag
- **GIVEN** a test seam that records the argv of each spawned `brew` subprocess
- **WHEN** `Update` is driven down the brew path with the flag both set and unset
- **THEN** the unset run's recorded argv SHALL contain `brew update` and `brew upgrade`
- **AND** the set run's recorded argv SHALL contain `brew upgrade` but NOT `brew update`

### Design Decisions

1. **Test seam: package-level command + brew-install indirection, exercised via Go's helper-process pattern.**
   - *Why*: The current `Update` shells out to real `brew` at three sites (`exec.CommandContext` for `update`, `upgrade`, `info`) and `isBrewInstalled()` is false under `go test`, so no test can reach the brew path today. To assert which subprocesses run without refactoring the convention, introduce two package-level `var` indirections: `var execCommandContext = exec.CommandContext` (used at all three call sites verbatim) and `var brewInstalled = isBrewInstalled` (so `Update` calls `brewInstalled()`). Tests stub `execCommandContext` with a recorder that captures `name`+`args` and returns a command pointing at the test binary's own helper process (`os.Args[0]` + `-test.run=TestHelperProcess`, the canonical stdlib idiom), and stub `brewInstalled` to return `true`. The helper process emits valid `--json=v2` output for the `info` call and exits 0 for `update`/`upgrade`. This keeps `os/exec` as the mechanism (no wrapper package, no interface, no new dependency) — it is the minimal seam, not a convention refactor.
   - *Rejected*: A `PATH`-shimmed fake `brew` shell script in a temp dir touches zero production code but is heavier, less portable, and off-pattern for this repo's table-driven unit-test style (Constitution V favors direct package unit tests). A full command-runner interface / `internal/proc` wrapper is explicitly forbidden by the contract ("do NOT refactor the subprocess convention") and would over-fragment a one-domain package (Constitution IV note in `update.go`).

2. **Boolean threaded as a positional parameter, not an options struct.**
   - *Why*: The contract says "thread `skipBrewUpdate bool` through `Update()`." `Update` has a single caller, so a positional bool is the minimal, idiomatic change. An options struct would be premature abstraction for one flag.
   - *Rejected*: A functional-options or config-struct signature — unjustified complexity for one boolean with one caller.

## Tasks

### Phase 1: Setup

- [x] T001 Introduce the test seam in `src/internal/idea/update.go`: add package-level `var execCommandContext = exec.CommandContext` and `var brewInstalled = isBrewInstalled` near the top of the file (after the const block). No behavior change yet.

### Phase 2: Core Implementation

- [x] T002 In `src/internal/idea/update.go`, rewrite the three `exec.CommandContext(...)` call sites (the `brew update` at ~L68, the `brew upgrade` at ~L108, and the `brew info` inside `brewLatestVersion` at ~L134) to call `execCommandContext(...)` instead. Replace the direct `isBrewInstalled()` call in `Update` with `brewInstalled()`. Pure indirection — identical runtime behavior.
- [x] T003 In `src/internal/idea/update.go`, change `Update`'s signature to `func Update(currentVersion string, skipBrewUpdate bool, out, errOut io.Writer) error`. Wrap the `brew update --quiet` block (the `ctx`/`updateCmd`/`updateStderr`/`err`/`cancel` section, ~L67–L82) in `if !skipBrewUpdate { ... }`. Everything after it (`brewLatestVersion`, up-to-date short-circuit, `brew upgrade`) stays outside the guard, unchanged. Update the `Update` doc comment to mention the new parameter and that it gates only the `brew update` refresh.
- [x] T004 In `src/cmd/idea/update.go`, register a local bool flag `--skip-brew-update` (default `false`, usage: "skip the internal 'brew update' tap-metadata refresh") on the `update` command, read it in `RunE`, and pass it to `idea.Update(version, skipBrewUpdate, cmd.OutOrStdout(), cmd.ErrOrStderr())`. Keep the existing `errSilent` mapping intact.

### Phase 3: Integration & Edge Cases

- [x] T005 In `src/internal/idea/update_test.go`, add a `TestHelperProcess` (canonical Go stdlib pattern, guarded by a `GO_WANT_HELPER_PROCESS` env check) that fakes `brew`: for an `info` invocation it prints valid `--json=v2` output with a `formulae[0].versions.stable` value; for `update`/`upgrade` it exits 0. Add a recording stub factory that captures each invocation's name+args into a slice and returns a command pointing at `os.Args[0]` with `-test.run=TestHelperProcess`.
- [x] T006 In `src/internal/idea/update_test.go`, add a table-driven `TestUpdateSkipBrewUpdate` that, for each row, stubs `brewInstalled` to return `true` and `execCommandContext` to the recorder (restoring both via `defer`), calls `Update("v0.0.1", skip, &out, &err)` (using a "stable" version that differs from current so the upgrade path runs), and asserts the recorded brew subcommands: skip=false → contains `update`, `info`, `upgrade`; skip=true → contains `info`, `upgrade` but NOT `update`. Use real temp/`bytes.Buffer` for writers; no interface mocks.

### Phase 4: Polish

- [x] T007 From `src/`, run `go build ./...` and `go test ./internal/idea/...` (and `cd src && go vet ./internal/idea/... ./cmd/idea/...`). Confirm the package builds and the new + existing update tests pass. Fix any fallout (e.g. the `Update` caller in `cmd`).

---

## Execution Order

- T001 (seam vars) blocks T002 (call-site rewrite) and T005/T006 (tests reference the vars).
- T003 (signature + guard) blocks T004 (caller passes the new arg) and T006 (test calls the new signature).
- T002 and T003 both edit `update.go` — apply sequentially, not in parallel.
- T005 blocks T006 (the recorder + helper process are used by the table test).
- T007 runs last, after all code + tests exist.

## Acceptance

## Functional Completeness
- [ ] CHK-001 Flag definition: `--skip-brew-update` is a real local cobra bool flag on `update`, default `false`, with usage text mentioning the `brew update` tap-metadata refresh; appears in `idea update --help`.
- [ ] CHK-002 Threaded parameter: `Update` signature is `Update(currentVersion string, skipBrewUpdate bool, out, errOut io.Writer) error` and the cobra `RunE` passes the parsed flag value.
- [ ] CHK-003 Skip guards only brew update: when `skipBrewUpdate == true`, only the `brew update` block is skipped; `brew info`, the up-to-date short-circuit, and `brew upgrade` are unaffected.

## Behavioral Correctness
- [ ] CHK-004 Default unchanged: with the flag absent (`false`), `brew update --quiet` runs exactly as before — no behavioral or output difference from pre-change.
- [ ] CHK-005 Output routing preserved: `brew update`/`brew info` remain captured; `brew upgrade` still inherits `os.Stdin/Stdout/Stderr`; wrapper messages still go to `out`/`errOut`; `errSilent` mapping intact.
- [ ] CHK-006 No convention refactor: no `internal/proc` wrapper or command-runner interface introduced; `os/exec` remains the mechanism; only the minimal `execCommandContext`/`brewInstalled` `var` seam added.

## Scenario Coverage
- [ ] CHK-007 "Skip omits brew update but still upgrades": test asserts skip=true → recorded argv has `info` + `upgrade`, NOT `update`.
- [ ] CHK-008 "Default path runs brew update": test asserts skip=false → recorded argv has `update` + `info` + `upgrade`.
- [ ] CHK-009 Help lists the flag: `--skip-brew-update` is discoverable (verifiable via cobra flag registration / help).

## Edge Cases & Error Handling
- [ ] CHK-010 Up-to-date short-circuit with skip: when skip=true and versions match, `brew upgrade` is NOT invoked and "Already up to date" fires (covered by spec scenario; verify behavior preserved).
- [ ] CHK-011 Single caller compiles: the only `Update` caller (`src/cmd/idea/update.go`) is updated and the module builds.

## Code Quality
- [ ] CHK-012 Pattern consistency: new code follows the file's naming/structure (package-level `var` seam near consts, doc-comment style on `Update`, table-driven test like existing tests).
- [ ] CHK-013 No unnecessary duplication: the helper-process / recorder pattern is the standard stdlib idiom; no duplicated brew-invocation logic.
- [ ] CHK-014 No magic strings: brew subcommand names compared in tests reference clear literals; formula stays the `brewFormula` constant (Constitution: no magic strings, anti-pattern list).
- [ ] CHK-015 Function size: `Update` stays focused; the skip guard does not turn it into a god function.

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
