# Tasks: Add --skip-brew-update flag to update command

**Change**: 260531-t2ov-skip-brew-update-flag
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Introduce the test seam in `src/internal/idea/update.go`: add package-level `var execCommandContext = exec.CommandContext` and `var brewInstalled = isBrewInstalled` near the top of the file (after the const block). No behavior change yet.

## Phase 2: Core Implementation

- [x] T002 In `src/internal/idea/update.go`, rewrite the three `exec.CommandContext(...)` call sites (the `brew update` at ~L68, the `brew upgrade` at ~L108, and the `brew info` inside `brewLatestVersion` at ~L134) to call `execCommandContext(...)` instead. Replace the direct `isBrewInstalled()` call in `Update` with `brewInstalled()`. Pure indirection — identical runtime behavior.
- [x] T003 In `src/internal/idea/update.go`, change `Update`'s signature to `func Update(currentVersion string, skipBrewUpdate bool, out, errOut io.Writer) error`. Wrap the `brew update --quiet` block (the `ctx`/`updateCmd`/`updateStderr`/`err`/`cancel` section, ~L67–L82) in `if !skipBrewUpdate { ... }`. Everything after it (`brewLatestVersion`, up-to-date short-circuit, `brew upgrade`) stays outside the guard, unchanged. Update the `Update` doc comment to mention the new parameter and that it gates only the `brew update` refresh.
- [x] T004 In `src/cmd/idea/update.go`, register a local bool flag `--skip-brew-update` (default `false`, usage: "skip the internal 'brew update' tap-metadata refresh") on the `update` command, read it in `RunE`, and pass it to `idea.Update(version, skipBrewUpdate, cmd.OutOrStdout(), cmd.ErrOrStderr())`. Keep the existing `errSilent` mapping intact.

## Phase 3: Integration & Edge Cases

- [x] T005 In `src/internal/idea/update_test.go`, add a `TestHelperProcess` (canonical Go stdlib pattern, guarded by a `GO_WANT_HELPER_PROCESS` env check) that fakes `brew`: for an `info` invocation it prints valid `--json=v2` output with a `formulae[0].versions.stable` value; for `update`/`upgrade` it exits 0. Add a recording stub factory that captures each invocation's name+args into a slice and returns a command pointing at `os.Args[0]` with `-test.run=TestHelperProcess`.
- [x] T006 In `src/internal/idea/update_test.go`, add a table-driven `TestUpdateSkipBrewUpdate` that, for each row, stubs `brewInstalled` to return `true` and `execCommandContext` to the recorder (restoring both via `defer`), calls `Update("v0.0.1", skip, &out, &err)` (using a "stable" version that differs from current so the upgrade path runs), and asserts the recorded brew subcommands: skip=false → contains `update`, `info`, `upgrade`; skip=true → contains `info`, `upgrade` but NOT `update`. Use real temp/`bytes.Buffer` for writers; no interface mocks.

## Phase 4: Polish

- [x] T007 From `src/`, run `go build ./...` and `go test ./internal/idea/...` (and `cd src && go vet ./internal/idea/... ./cmd/idea/...`). Confirm the package builds and the new + existing update tests pass. Fix any fallout (e.g. the `Update` caller in `cmd`).

---

## Execution Order

- T001 (seam vars) blocks T002 (call-site rewrite) and T005/T006 (tests reference the vars).
- T003 (signature + guard) blocks T004 (caller passes the new arg) and T006 (test calls the new signature).
- T002 and T003 both edit `update.go` — apply sequentially, not in parallel.
- T005 blocks T006 (the recorder + helper process are used by the table test).
- T007 runs last, after all code + tests exist.
