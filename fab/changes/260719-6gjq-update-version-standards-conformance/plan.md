# Plan: Update & Version Standards Conformance

**Change**: 260719-6gjq-update-version-standards-conformance
**Intake**: `intake.md`

## Requirements

### CLI: Brew-Handling Safety (`idea update`)

#### R1: No SIGKILL path toward brew subprocesses
`src/internal/idea/update.go` MUST NOT impose a deadline on any brew subprocess. The three timeout constants (`brewUpdateTimeout`, `brewInfoTimeout`, `brewUpgradeTimeout`) and every `context.WithTimeout` wrapper SHALL be deleted; all three brew call sites (`brew update --quiet`, `brew info --json=v2`, `brew upgrade`) SHALL pass `context.Background()` through the UNCHANGED `execCommandContext(ctx, ...)` seam. No code path may send SIGKILL to brew (the toolkit update standard's MUST NOT clause — `exec.CommandContext`'s default cancel sends `os.Kill` on deadline, which mid-`brew upgrade` corrupts the keg between unlink and link). Doc comments SHALL state the no-deadline contract and its rationale. Success-path behavior (wrapper messages, I/O routing, `errSilent`/`exec.ErrNotFound` error mapping) MUST be unchanged.

- **GIVEN** a brew-installed binary running `idea update` on a slow network
- **WHEN** any brew subprocess (`update`, `info`, `upgrade`) takes arbitrarily long
- **THEN** no deadline fires and no SIGKILL is ever sent to brew (Ctrl-C/SIGINT remains the user's escape hatch, which brew traps and unwinds cleanly)
- **AND** `exec.CommandContext` with `context.Background()` behaves identically to `exec.Command` on the success path — no behavior change

#### R2: No-deadline contract pinned by test
`src/internal/idea/update_test.go` SHALL extend the existing `execCommandContext` recorder seam (used by `TestUpdateSkipBrewUpdate`) to capture the `ctx` passed at each call site, and SHALL assert that every recorded brew invocation's ctx reports NO deadline (`ctx.Deadline()` ok == false). Reintroducing a `context.WithTimeout` around any brew call MUST fail the test.

- **GIVEN** the brew path forced via the stubbed `brewInstalled`/`execCommandContext` seam vars
- **WHEN** `Update` runs (both skip=false and skip=true rows) and the recorder captures each brew invocation's ctx
- **THEN** for each recorded invocation (`update`, `info`, `upgrade`), `ctx.Deadline()` returns ok == false

### CLI: Version-Shape Conformance (`--version`)

#### R3: Version output shape pinned by test
`src/cmd/idea/main_test.go` SHALL add a conformance test executing the root command with `--version` in-process (via `newRootCmd()` with `SetOut`/`SetErr`/`SetArgs`), asserting: no error (the exit-0 path), output on stdout with stderr empty, and the first line — with nothing preceding it (no banner) — matches `^idea version \S+$` (the `<word> version <rest>` prefix shape shll's `versionPrefixRE` parses; dev builds emit `idea version dev`, release builds `idea version vX.Y.Z`).

- **GIVEN** the root command built by `newRootCmd()` with buffered out/err
- **WHEN** executed with args `["--version"]`
- **THEN** `Execute()` returns nil, stderr is empty, and stdout's line 1 matches `^idea version \S+$`

### Non-Goals

- `HOMEBREW_NO_GITHUB_API=1` — only suggested by the standard for tools that must bound the call; with no bound it is unnecessary
- Any change to `cmd/idea/update.go`, help text, README, or docs/site — timeouts appear nowhere on those surfaces
- Replacing the deadline with a SIGTERM+grace bound — Go's `cmd.Cancel`+`WaitDelay` still SIGKILLs after grace; only "no kill path" strictly satisfies the checklist (intake assumption 1)
- `docs/memory/` updates — hydrate stage's job, not apply's

## Tasks

### Phase 2: Core Implementation

- [x] T001 Remove SIGKILL-bearing deadlines in `src/internal/idea/update.go`: delete the `brewUpdateTimeout`/`brewInfoTimeout`/`brewUpgradeTimeout` const block and all `context.WithTimeout` wrappers; pass `context.Background()` at all three brew call sites through the unchanged `execCommandContext` seam; drop the now-unused `time` import; update doc comments to state the no-deadline brew-safety contract (why no deadline may wrap brew, per the toolkit update standard) <!-- R1 -->
- [x] T002 Extend the recorder in `src/internal/idea/update_test.go`: change `newBrewRecorder` to record `{sub, ctx}` per invocation (a small `brewCall` struct), adapt `TestUpdateSkipBrewUpdate`'s subcommand assertions, and add the no-deadline assertion — every recorded brew invocation's `ctx.Deadline()` must report ok == false <!-- R2 -->
- [x] T003 [P] Add `TestVersionFlag_ShapeConformance` to `src/cmd/idea/main_test.go`: build root via `newRootCmd()`, `SetArgs([]string{"--version"})`, buffered out/err; assert nil error, empty stderr, and stdout line 1 (nothing preceding) matches `^idea version \S+$` <!-- R3 -->

### Phase 3: Integration & Edge Cases

- [x] T004 Verify: `cd src && go test ./internal/idea/ ./cmd/idea/` green, `gofmt -l` clean and `go vet` clean on both touched packages <!-- R1 R2 R3 -->

## Acceptance

### Functional Completeness

- [ ] A-001 R1: No `context.WithTimeout` call and no timeout constant remains in `src/internal/idea/update.go`; all three brew call sites pass `context.Background()`; the `execCommandContext(ctx, ...)` seam signature is unchanged
- [ ] A-002 R2: `update_test.go`'s recorder captures the ctx per brew invocation and asserts `ctx.Deadline()` ok == false for every recorded call (`update`, `info`, `upgrade`) in both table rows
- [ ] A-003 R3: `main_test.go` has a version-shape test asserting nil error, stdout-only output (stderr empty), and first-line match of `^idea version \S+$` with nothing preceding it

### Behavioral Correctness

- [ ] A-004 R1: Success-path behavior is unchanged — wrapper messages, I/O routing (captured `update`/`info`, stdio-inheriting `upgrade`), and `errSilent`/`exec.ErrNotFound` error mapping are untouched; `TestUpdateSkipBrewUpdate`'s skip semantics still pass

### Removal Verification

- [ ] A-005 R1: `brewUpdateTimeout`, `brewInfoTimeout`, `brewUpgradeTimeout`, every `context.WithTimeout`/`cancel` around brew, and the `time` import are gone from `update.go` — no dead code, no remaining kill path toward brew

### Scenario Coverage

- [ ] A-006 R2: Reintroducing a `context.WithTimeout` around any brew call site fails the extended test (the ctx-deadline assertion is exercised on all three subcommands)

### Code Quality

- [ ] A-007 Pattern consistency: New test code follows the file's existing patterns (seam stubbing with `defer` restore in `update_test.go`; in-process `newRootCmd()` execution in `main_test.go`), and touched packages pass `gofmt`/`go vet`
- [ ] A-008 No unnecessary duplication: The existing `execCommandContext` recorder seam and `TestHelperProcess` fake-exec idiom are extended, not duplicated; no new dependencies (stdlib only)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Version test runs in-process via `newRootCmd()` + `SetOut`/`SetErr`/`SetArgs` (the `TestConfirmPrune` pattern), not as a subprocess build | Intake names this pattern verbatim ("build root via `newRootCmd()`, execute with args, capture out/err"); cobra prints the version template to `OutOrStdout()` before RunE, so in-process capture is exact and faster than `buildBinary` | S:90 R:90 A:95 D:90 |
| 2 | Certain | Extend `newBrewRecorder` to record a `{sub, ctx}` struct per invocation and add the deadline assertion inside the existing `TestUpdateSkipBrewUpdate` table loop, rather than adding a parallel recorder/test | Intake prescribes extending the existing recorder seam; one recorder keeps the seam single-sourced (no duplication) and the assertion runs across both table rows automatically | S:85 R:90 A:90 D:85 |
| 3 | Confident | The no-deadline rationale doc comment lives on the seam var block + the brew-invoking code paths (no standalone constant block remains to host it) | The seam comment is where a future editor reaching for `context.WithTimeout` will look; keeps rationale adjacent to the enforced contract | S:70 R:90 A:85 D:75 |

3 assumptions (2 certain, 1 confident, 0 tentative).
