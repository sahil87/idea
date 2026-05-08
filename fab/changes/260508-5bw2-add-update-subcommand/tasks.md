# Tasks: Add `idea update` Subcommand

**Change**: 260508-5bw2-add-update-subcommand
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

(No setup required — no new dependencies, no scaffolding files. The internal package and cmd directory both already exist.)

## Phase 2: Core Implementation

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

## Phase 3: Integration & Edge Cases

- [x] T009 In `src/cmd/idea/main.go`, add `updateCmd()` to the existing `root.AddCommand(...)` call. The list currently contains seven entries; add `updateCmd()` as the eighth (placement at the end of the list is fine — order in `AddCommand` does not affect help-output ordering).

- [x] T010 Run `cd src && go build ./cmd/idea` from the repo root. Verify it compiles with no errors. If imports are missing or unused, fix and re-run until clean.

- [x] T011 Run `cd src && go test ./internal/idea/...` to verify the three new tests pass. If `TestUpdateNonBrewInstall` skips (because the test binary happens to live under `/Cellar/`), that is acceptable — the skip path is intentional. The other two tests MUST pass.

- [x] T012 Run `cd src && go test ./...` to verify the full test suite still passes (no regressions in the existing `cmd/idea/main_test.go` integration suite).

- [x] T013 Smoke-test the binary manually: run `just build` (which calls `scripts/build.sh` and produces `bin/idea` at the repo root via `go build -ldflags "-X main.version=..." -o ../bin/idea ./cmd/idea` from inside `src/`). Then run `./bin/idea update`. Verify stdout contains `was not installed via Homebrew` and `brew install sahil87/tap/idea`, stderr is empty, exit code is 0. <!-- clarified: build command corrected to match scripts/build.sh — output is bin/idea at repo root, not src/idea -->

- [x] T014 Smoke-test help output: run `./bin/idea --help`. Verify `update` is listed under "Available Commands" with the description `"self-update the idea binary via Homebrew"`. <!-- clarified: binary path corrected from ./src/idea to ./bin/idea (matches scripts/build.sh output) -->

## Phase 4: Polish

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
