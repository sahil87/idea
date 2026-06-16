# Plan: Add CI Workflow

**Change**: 260610-k6pc-add-ci-workflow
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- Requirements derived from intake.md, which mirrors wt/.github/workflows/ci.yml
     verbatim. idea's repo layout is identical to wt's (src/go.mod, src/go.sum),
     so the workflow is drop-in with no path changes. -->

### CI: Workflow Triggers & Metadata

#### R1: Workflow runs on pull requests and pushes to main
The workflow SHALL be defined in `.github/workflows/ci.yml` and MUST trigger on every
`pull_request` and on every `push` to the `main` branch.

- **GIVEN** the `ci.yml` workflow exists
- **WHEN** a pull request is opened or a commit is pushed to `main`
- **THEN** the workflow is triggered
- **AND** pushes to non-`main` branches (without an associated PR) do NOT trigger it

#### R2: Workflow declares least-privilege permissions
The workflow MUST declare `permissions: contents: read` — the workflow only reads source.

- **GIVEN** the `ci.yml` workflow
- **WHEN** any job runs
- **THEN** the granted token has only `contents: read` scope

#### R3: Concurrency cancels superseded runs on the same ref
The workflow MUST declare a `concurrency` group keyed on `ci-${{ github.ref }}` with
`cancel-in-progress: true`, so rapid pushes to the same branch cancel in-flight runs.

- **GIVEN** a run is in progress for a given ref
- **WHEN** a newer push lands on the same ref
- **THEN** the older run is cancelled and the newer one proceeds

### CI: Test Job

#### R4: `test` job checks out and sets up Go from the module file
The `test` job MUST run on `ubuntu-latest`, check out the repo using
`actions/checkout` pinned to v4 SHA `34e114876b0b11c390a56381ad16ebd13914f8d5`, and set up Go
using `actions/setup-go` pinned to v5 SHA `40f1582b2485089dde7abd97c1529aa768e1baff` with
`go-version-file: src/go.mod` and `cache-dependency-path: src/go.sum`.

- **GIVEN** the `test` job
- **WHEN** it starts
- **THEN** it checks out the repo at the pinned checkout SHA
- **AND** sets up Go using the version declared in `src/go.mod`, caching modules per `src/go.sum`

#### R5: `test` job enforces gofmt, vet, and tests in `src`
The `test` job MUST run three checks, each with `working-directory: src`:
(1) a gofmt step that runs `gofmt -l .` and exits non-zero if any file is unformatted,
printing the offending files and remediation hint to stderr;
(2) `go vet ./...`;
(3) `go test ./...`.

- **GIVEN** the `test` job has set up Go
- **WHEN** the source under `src/` is gofmt-clean, vet-clean, and tests pass
- **THEN** the `test` job succeeds
- **AND** if any file is not gofmt-clean, the gofmt step prints the files and the
  `Run: (cd src && gofmt -w .)` hint to stderr and exits 1
- **AND** a `go vet` or `go test` failure fails the job

### Repo: gofmt Cleanliness

#### R7: Source under `src/` is gofmt-clean so the new gofmt gate passes
The source tree under `src/` MUST be gofmt-clean so the gofmt step added in R5 passes on the
first CI run. A pre-existing violation in `src/internal/idea/idea_test.go` (struct field
alignment in `TestResolveFilePath`, present on `main`) MUST be corrected via `gofmt -w` as part
of this change.

- **GIVEN** the new `ci.yml` gofmt gate
- **WHEN** CI runs on this change's PR
- **THEN** `gofmt -l .` (in `src`) produces no output and the gate passes
- **AND** the only source edit is the gofmt re-formatting (no behavioral change)

### CI: Gate Job

#### R6: `ci-gate` job is the single stable required status check
The workflow MUST define a `ci-gate` job with `needs: [test]`, `runs-on: ubuntu-latest`, and
`if: always()`, which verifies `needs.test.result == 'success'` and exits 1 otherwise. This
provides a single, stable required-check name for branch protection that stays constant even if
`test` is later split or renamed.

- **GIVEN** the `test` job has finished (any result)
- **WHEN** `ci-gate` runs (it always runs due to `if: always()`)
- **THEN** it succeeds only when `needs.test.result == 'success'`
- **AND** when `test` failed, cancelled, or was skipped, `ci-gate` prints the failing result to
  stderr and exits 1 (failing explicitly rather than being skipped)

### Non-Goals

- Enabling branch protection to require the `ci-gate` check — that is a repo-settings action, not
  a code change. Noted for the user to action separately.
- Modifying `release.yml` or any source behavior. The only source edit is the gofmt re-formatting
  of `idea_test.go` (R7) — a no-op whitespace change, not a behavioral one.
- Adding or changing dependencies.

### Design Decisions

1. **Mirror `wt/.github/workflows/ci.yml` verbatim**: reproduce wt's proven CI file unchanged —
   *Why*: identical repo layout (`src/go.mod`, `src/go.sum`) makes it drop-in; it is battle-tested
   on a sibling Go repo and pins the same action SHAs, avoiding bespoke-workflow drift — *Rejected*:
   authoring a bespoke workflow (reinvents a working pattern, risks divergence between the two repos).
2. **`ci-gate` as the pinned required check (not `test` directly)**: keeps the required-check name
   constant across future `test` refactors — *Why*: only `ci-gate`'s `needs:` list tracks job
   splits/renames, not the repo ruleset — *Rejected*: pinning `test` directly (brittle to renames).

## Tasks

### Phase 1: Core Implementation

- [x] T001 Create `.github/workflows/ci.yml` mirroring `wt/.github/workflows/ci.yml` verbatim: `name: CI`; `on: { push: { branches: [main] }, pull_request: {} }`; `permissions: { contents: read }`; `concurrency: { group: ci-${{ github.ref }}, cancel-in-progress: true }` <!-- R1, R2, R3 -->
- [x] T002 In `.github/workflows/ci.yml`, add the `test` job (`runs-on: ubuntu-latest`) with checkout (v4 SHA `34e114876b0b11c390a56381ad16ebd13914f8d5`) and setup-go (v5 SHA `40f1582b2485089dde7abd97c1529aa768e1baff`, `go-version-file: src/go.mod`, `cache-dependency-path: src/go.sum`) <!-- R4 -->
- [x] T003 In `.github/workflows/ci.yml` `test` job, add the gofmt step (`working-directory: src`, `gofmt -l .` with non-empty-output failure + remediation hint), the `go vet ./...` step, and the `go test ./...` step (both `working-directory: src`) <!-- R5 -->
- [x] T004 In `.github/workflows/ci.yml`, add the `ci-gate` job (`needs: [test]`, `runs-on: ubuntu-latest`, `if: always()`) that exits 1 unless `needs.test.result == 'success'` <!-- R6 -->

- [x] T005 Run `(cd src && gofmt -w .)` to make the source tree gofmt-clean — corrects the pre-existing `src/internal/idea/idea_test.go` struct field alignment so the new gofmt gate passes on first CI run <!-- R7 -->

### Phase 2: Validation

- [x] T006 Validate `.github/workflows/ci.yml` is well-formed YAML (`python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"`); confirm `src/go.mod` + `src/go.sum` exist; and confirm `(cd src && gofmt -l . && go vet ./... && go test ./...)` is fully clean <!-- R1, R4, R5, R7 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `.github/workflows/ci.yml` exists with `on.push.branches: [main]` and `on.pull_request` triggers
- [x] A-002 R2: The workflow declares `permissions: contents: read`
- [x] A-003 R3: The workflow declares `concurrency` with `group: ci-${{ github.ref }}` and `cancel-in-progress: true`
- [x] A-004 R4: The `test` job runs on `ubuntu-latest`, uses `actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5`, and `actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff` with `go-version-file: src/go.mod` and `cache-dependency-path: src/go.sum`
- [x] A-005 R5: The `test` job has gofmt (`working-directory: src`, fails on `gofmt -l .` output), `go vet ./...`, and `go test ./...` steps, all in `working-directory: src`
- [x] A-006 R6: The `ci-gate` job has `needs: [test]`, `runs-on: ubuntu-latest`, `if: always()`, and exits 1 unless `needs.test.result == 'success'`

### Scenario Coverage

- [x] A-007 R5: The gofmt step prints offending files plus `Run: (cd src && gofmt -w .)` to stderr and exits 1 when any file is unformatted
- [x] A-008 R6: When `test` is non-success, `ci-gate` fails explicitly (does not skip), keeping a skipped-required-check merge block from occurring

### Code Quality

- [x] A-009 Pattern consistency: `ci.yml` matches the structure/conventions of the existing `release.yml` (SHA-pinned actions, `src/go.mod` references) and mirrors wt's source verbatim
- [x] A-010 No unnecessary duplication: No bespoke reimplementation — the workflow reuses wt's proven pattern; only `ci.yml` (new) and the gofmt re-format of `idea_test.go` are touched
- [x] A-011 R7: `(cd src && gofmt -l .)` produces no output (source tree is gofmt-clean), and `go vet ./...` + `go test ./...` remain green after the re-format

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- **Pre-existing gofmt violation — fixed in this PR (user decision)**: `src/internal/idea/idea_test.go`
  was not gofmt-clean on `main` (struct field alignment in `TestResolveFilePath`). Without a fix, the
  new gofmt gate would fail on the first CI run. The user opted to fix it in this same PR so CI is green
  on arrival (T005). The fix is a pure `gofmt -w` whitespace re-alignment (4 lines) — no behavioral
  change. See revised Assumption 3.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Reproduce `wt/.github/workflows/ci.yml` byte-for-byte (same SHAs, same `src`-rooted steps, same comments) | Intake assumptions 1-7 are all Certain and specify the content verbatim; identical repo layout confirmed (`src/go.mod`, `src/go.sum` present) | S:98 R:85 A:95 D:98 |
| 2 | Certain | New `ci.yml` plus a gofmt-only re-format of `idea_test.go`; no other source or `release.yml` changes | Intake scoped this to one additive file; user expanded scope to include the gofmt fix (see #3). The fix is whitespace-only | S:98 R:90 A:95 D:95 |
| 3 | Certain | Fix the pre-existing `idea_test.go` gofmt violation in THIS PR via `gofmt -w` so CI is green on arrival | Clarified — user explicitly chose "Fix it in this PR" when asked. Overrides the intake's original "no source changes" scope. Violation predates this branch (exists on `main`); fix is a 4-line whitespace re-alignment | S:95 R:85 A:90 D:95 |

3 assumptions (3 certain, 0 confident, 0 tentative).
