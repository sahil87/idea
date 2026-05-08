# Quality Checklist: Add `idea update` Subcommand

**Change**: 260508-5bw2-add-update-subcommand
**Generated**: 2026-05-08
**Spec**: `spec.md`

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
