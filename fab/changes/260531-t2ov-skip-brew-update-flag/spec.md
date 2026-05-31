# Spec: Add --skip-brew-update flag to update command

**Change**: 260531-t2ov-skip-brew-update-flag
**Created**: 2026-05-31
**Affected memory**: `docs/memory/cli/update.md`

## Non-Goals

- Skipping the `brew info` version check — out of scope; the flag suppresses only the tap-metadata refresh.
- Skipping the up-to-date short-circuit or `brew upgrade` — these always run as today.
- A persistent/global flag — the flag is local to `update` (see Constitution III).
- Refactoring the subprocess invocation convention (no `internal/proc` wrapper, no command-runner interface). Only the minimal seam needed for testing is introduced.
- Auto-detecting "brew was already updated recently" — the flag is an explicit, caller-driven opt-out.

## CLI: `--skip-brew-update` flag

### Requirement: Flag definition
The `update` subcommand MUST define a local boolean flag named exactly `--skip-brew-update`, defaulting to `false`, with a short usage description stating it skips the internal `brew update` tap-metadata refresh. The flag MUST be a real cobra flag wired on the `update` command (not a root persistent flag). Its value MUST be passed into `idea.Update(...)`.

#### Scenario: Flag is registered on update
- **GIVEN** the compiled `idea` binary
- **WHEN** the user runs `idea update --help`
- **THEN** the help output SHALL list `--skip-brew-update` with its description

#### Scenario: Flag rejects values it shouldn't take
- **GIVEN** a boolean flag
- **WHEN** the user runs `idea update --skip-brew-update`
- **THEN** the flag SHALL parse as `true` with no required value argument

### Requirement: Default behavior unchanged
When `--skip-brew-update` is absent, `Update` MUST behave exactly as before this change: it runs `brew update --quiet`, then `brew info`, then the up-to-date short-circuit, then `brew upgrade` when an upgrade is available.

#### Scenario: Default path runs brew update
- **GIVEN** a Homebrew-installed binary and the flag absent (`skipBrewUpdate == false`)
- **WHEN** `Update` runs down the brew path
- **THEN** `brew update --quiet` SHALL be invoked
- **AND** `brew info --json=v2 sahil87/tap/idea` SHALL be invoked
- **AND** `brew upgrade sahil87/tap/idea` SHALL be invoked when the versions differ

## internal/idea: `Update` signature and skip behavior

### Requirement: Threaded boolean parameter
`Update` MUST accept a `skipBrewUpdate bool` parameter threaded from the cobra layer. The signature SHALL be `func Update(currentVersion string, skipBrewUpdate bool, out, errOut io.Writer) error`. The parameter is placed after `currentVersion` and before the writers, grouping inputs ahead of output sinks.

#### Scenario: Single caller updated
- **GIVEN** `Update` has exactly one caller (`src/cmd/idea/update.go`)
- **WHEN** the signature changes
- **THEN** the caller SHALL pass the flag value and the package SHALL compile

### Requirement: Skip guards only the brew update block
When `skipBrewUpdate` is `true`, `Update` MUST NOT spawn `brew update`. Control MUST flow directly to the `brew info` version check. The `brew info` call, the up-to-date short-circuit, and the `brew upgrade` call MUST be unaffected by the flag. The non-Homebrew-install short-circuit at the top of `Update` MUST be unaffected.

#### Scenario: Skip omits brew update but still upgrades
- **GIVEN** a Homebrew-installed binary and `skipBrewUpdate == true`
- **WHEN** `Update` runs down the brew path with a newer version available
- **THEN** `brew update` SHALL NOT be invoked
- **AND** `brew info --json=v2 sahil87/tap/idea` SHALL be invoked
- **AND** `brew upgrade sahil87/tap/idea` SHALL be invoked

#### Scenario: Skip with up-to-date version
- **GIVEN** a Homebrew-installed binary and `skipBrewUpdate == true`
- **WHEN** `Update` runs and the installed version equals the latest
- **THEN** `brew update` SHALL NOT be invoked
- **AND** `brew info` SHALL be invoked
- **AND** the "Already up to date" short-circuit SHALL fire
- **AND** `brew upgrade` SHALL NOT be invoked

### Requirement: Output routing preserved
The change MUST NOT alter I/O routing. `brew update` and `brew info` streams remain captured; `brew upgrade` continues to inherit `os.Stdin`/`os.Stdout`/`os.Stderr`. Wrapper messages continue to write to `out`/`errOut`. The `errSilent` mapping in `RunE` is unchanged.

#### Scenario: Upgrade still inherits stdio
- **GIVEN** the skip flag in either state
- **WHEN** `brew upgrade` is invoked
- **THEN** it SHALL inherit the process stdio exactly as before

## Testing: assert flag behavior without refactoring the convention

### Requirement: Test the skip behavior via the repo's table-driven pattern
A test in `src/internal/idea/update_test.go` MUST assert that with `skipBrewUpdate == true`, `brew update` is not invoked while `brew info` and `brew upgrade` are; and with `skipBrewUpdate == false`, `brew update` is invoked. The test MUST follow the repo's table-driven style (Constitution V) and use real-process or recorded-argv observation rather than mocking interfaces. It MUST NOT require a real Homebrew installation and MUST be stable in CI.

#### Scenario: Recorded invocations differ by flag
- **GIVEN** a test seam that records the argv of each spawned `brew` subprocess
- **WHEN** `Update` is driven down the brew path with the flag both set and unset
- **THEN** the unset run's recorded argv SHALL contain `brew update` and `brew upgrade`
- **AND** the set run's recorded argv SHALL contain `brew upgrade` but NOT `brew update`

## Design Decisions

1. **Test seam: package-level command + brew-install indirection, exercised via Go's helper-process pattern.**
   - *Why*: The current `Update` shells out to real `brew` at three sites (`exec.CommandContext` for `update`, `upgrade`, `info`) and `isBrewInstalled()` is false under `go test`, so no test can reach the brew path today. To assert which subprocesses run without refactoring the convention, introduce two package-level `var` indirections: `var execCommandContext = exec.CommandContext` (used at all three call sites verbatim) and `var brewInstalled = isBrewInstalled` (so `Update` calls `brewInstalled()`). Tests stub `execCommandContext` with a recorder that captures `name`+`args` and returns a command pointing at the test binary's own helper process (`os.Args[0]` + `-test.run=TestHelperProcess`, the canonical stdlib idiom), and stub `brewInstalled` to return `true`. The helper process emits valid `--json=v2` output for the `info` call and exits 0 for `update`/`upgrade`. This keeps `os/exec` as the mechanism (no wrapper package, no interface, no new dependency) — it is the minimal seam, not a convention refactor.
   - *Rejected*: A `PATH`-shimmed fake `brew` shell script in a temp dir touches zero production code but is heavier, less portable, and off-pattern for this repo's table-driven unit-test style (Constitution V favors direct package unit tests). A full command-runner interface / `internal/proc` wrapper is explicitly forbidden by the contract ("do NOT refactor the subprocess convention") and would over-fragment a one-domain package (Constitution IV note in `update.go`).

2. **Boolean threaded as a positional parameter, not an options struct.**
   - *Why*: The contract says "thread `skipBrewUpdate bool` through `Update()`." `Update` has a single caller, so a positional bool is the minimal, idiomatic change. An options struct would be premature abstraction for one flag.
   - *Rejected*: A functional-options or config-struct signature — unjustified complexity for one boolean with one caller.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Flag name exactly `--skip-brew-update`, bool, default `false` | Confirmed from intake #1 — fixed by cross-toolkit contract | S:100 R:90 A:95 D:100 |
| 2 | Certain | Skip guards ONLY the `brew update` block; info / short-circuit / upgrade always run | Confirmed from intake #2 — explicit in contract; single-branch guard in `Update` | S:100 R:80 A:95 D:100 |
| 3 | Certain | Default (flag absent) preserves current behavior byte-for-byte | Confirmed from intake #3 — explicit in contract | S:100 R:85 A:95 D:100 |
| 4 | Certain | Output routing unchanged; no subprocess-convention refactor | Confirmed from intake #4 — explicit in contract; matches `cli/update.md` I/O split | S:95 R:75 A:95 D:95 |
| 5 | Certain | Signature `Update(currentVersion string, skipBrewUpdate bool, out, errOut io.Writer)` | Upgraded from intake #5 (Confident→Certain) — single caller, contained; param order is inputs-before-sinks; spec-level analysis confirms no ambiguity | S:90 R:80 A:90 D:85 |
| 6 | Certain | Local flag on `update`, not a root persistent flag | Upgraded from intake #6 (Confident→Certain) — Constitution III reserves persistent flags for `--file`/`--main`; deterministic | S:90 R:85 A:95 D:90 |
| 7 | Certain | Table-driven test in `update_test.go`; run via `cd src && go test ./internal/idea/...` | Upgraded from intake #7 (Confident→Certain) — Constitution V + config directives + justfile; deterministic | S:90 R:85 A:95 D:90 |
| 8 | Confident | Test seam = two `var` indirections (`execCommandContext`, `brewInstalled`) + Go helper-process pattern | Upgraded from intake #8 (Tentative→Confident) — Design Decision 1 selects the minimal idiomatic seam over PATH-shim/interface; canonical stdlib pattern, zero deps. Remains Confident (not Certain) because the exact recorder shape is an implementation detail settled at apply | S:75 R:65 A:80 D:75 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
