# Intake: Add CI Workflow

**Change**: 260610-k6pc-add-ci-workflow
**Created**: 2026-06-10
**Status**: Draft

## Origin

> Add a CI workflow (.github/workflows/ci.yml) to idea that runs on pull_request and
> push-to-main, mirroring wt's ci.yml: gofmt -l check, go vet ./..., and go test ./... in
> the src/ working directory using go-version-file src/go.mod. Include concurrency with
> cancel-in-progress keyed on github.ref, and a ci-gate job (needs:[test], if:always()) as
> a single stable required status check. Pin actions to the same SHAs wt uses
> (actions/checkout v4, actions/setup-go v5).

This change originated from a `/fab-discuss` session comparing `idea`'s GitHub Actions setup
against the sibling `wt` repo (`~/code/sahil87/wt/`). The comparison surfaced that `wt` has
**both** `ci.yml` (PR/push tests) and `release.yml` (tag-driven release), while `idea` has
**only** `release.yml`. Pull requests against `idea` therefore run no automated tests, vet, or
gofmt gate before merge — a gap given the constitution leans on tests and gofmt as the quality
bar (Principle V; `stage_directives.apply`).

The user reviewed `wt/.github/workflows/ci.yml` and confirmed it is essentially drop-in for
`idea`: both repos share the identical layout (`src/go.mod`, `src/go.sum`, `working-directory:
src`), and `idea`'s existing `release.yml` already references `src/go.mod` the same way. The
decision was to port it through the fab pipeline rather than hand-edit.

## Why

1. **Problem**: `idea` has no pre-merge CI. The only workflow (`release.yml`) triggers solely on
   `v*` tag push and manual `workflow_dispatch` — nothing validates a pull request. Broken
   formatting, vet failures, or failing tests can land on `main` undetected until a release is
   cut (or never).
2. **Consequence if unfixed**: Regressions merge silently. The constitution's quality
   guarantees (table-driven tests, gofmt-clean source) are aspirational without enforcement.
   Branch protection cannot require a status check that does not exist.
3. **Why this approach**: Mirroring `wt`'s proven `ci.yml` is the lowest-risk option — it is
   already battle-tested on an identically-structured Go repo in the same owner's account, uses
   the same `src/`-rooted module layout, and pins the same action SHAs. Authoring a bespoke
   workflow would reinvent a working pattern and risk drift between the two repos. No new tooling
   or dependencies are introduced.

## What Changes

### New file: `.github/workflows/ci.yml`

A single new workflow file. No existing files are modified. The content mirrors
`wt/.github/workflows/ci.yml` with `idea`-appropriate values (which, given the identical
layout, means no path changes are needed — `src/` and `src/go.mod` apply verbatim).

#### Triggers

```yaml
on:
  push:
    branches:
      - main
  pull_request:
```

Runs on every pull request and on every push to `main`.

#### Permissions

```yaml
permissions:
  contents: read
```

Least-privilege — the workflow only reads source.

#### Concurrency

```yaml
concurrency:
  # Cancel superseded runs on the same ref (PR branch or main).
  group: ci-${{ github.ref }}
  cancel-in-progress: true
```

Keyed on `github.ref` so rapid pushes to the same branch cancel in-flight runs, saving CI
minutes.

#### `test` job

Runs on `ubuntu-latest`. Steps:

1. `actions/checkout` pinned to v4 SHA `34e114876b0b11c390a56381ad16ebd13914f8d5`
2. `actions/setup-go` pinned to v5 SHA `40f1582b2485089dde7abd97c1529aa768e1baff`, configured
   with `go-version-file: src/go.mod` and `cache-dependency-path: src/go.sum` (single source of
   truth for the Go version — same as `release.yml`)
3. **gofmt** (`working-directory: src`):
   ```bash
   unformatted="$(gofmt -l .)"
   if [ -n "$unformatted" ]; then
     echo "The following files are not gofmt-clean:" >&2
     echo "$unformatted" >&2
     echo "Run: (cd src && gofmt -w .)" >&2
     exit 1
   fi
   ```
4. **go vet** (`working-directory: src`): `go vet ./...`
5. **go test** (`working-directory: src`): `go test ./...`

#### `ci-gate` job

```yaml
ci-gate:
  needs: [test]
  runs-on: ubuntu-latest
  if: always()
  steps:
    - name: Verify CI passed
      run: |
        if [ "${{ needs.test.result }}" != "success" ]; then
          echo "CI gate failed: test job result = ${{ needs.test.result }}" >&2
          exit 1
        fi
        echo "CI gate passed: all required jobs succeeded."
```

A single stable status check to pin branch protection to. Pinning the ruleset to this job
(rather than `test` directly) keeps the required-check name constant even if `test` is later
split or renamed — only this job's `needs:` list tracks that, not the repo ruleset. `if:
always()` ensures the gate runs (and can fail explicitly) even when `test` failed, avoiding the
confusing "skipped required check blocks merge" state.

## Affected Memory

The `release` memory domain documents the release pipeline. CI is a distinct, new concern. A
small memory addition recording the CI workflow's existence and purpose is warranted during
hydrate, but no existing memory file changes behavior.

- `release/pipeline.md`: (modify) Add a note that CI (`ci.yml`) runs gofmt/vet/test on PRs and
  push-to-main, distinct from the tag-driven release pipeline — OR a new `ci/` domain if the
  reviewer prefers separation. Defer the exact placement to hydrate.

## Impact

- **New file**: `.github/workflows/ci.yml`
- **No source code changes** — purely additive CI config.
- **No dependencies** added.
- **Downstream (out of scope here)**: enabling branch protection to require the `ci-gate` check
  is a repo-settings action, not a code change. Noted for the user to action separately if
  desired.
- **GitHub Actions billing**: adds CI minutes per PR/push, mitigated by `cancel-in-progress`.

## Open Questions

None blocking. One deferred-to-hydrate detail: whether the CI note lives under the existing
`release/` memory domain or a new `ci/` domain.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Mirror `wt/.github/workflows/ci.yml` rather than author a bespoke workflow | Discussed — user reviewed wt's ci.yml and chose to port it; identical repo layout makes it drop-in | S:95 R:80 A:90 D:95 |
| 2 | Certain | Triggers are `pull_request` + push to `main` | Explicitly specified in the request and matches wt | S:98 R:75 A:95 D:98 |
| 3 | Certain | Three checks: `gofmt -l`, `go vet ./...`, `go test ./...` in `working-directory: src` | Explicitly specified; `src/` is the confirmed module root (`src/go.mod`) | S:98 R:70 A:95 D:95 |
| 4 | Certain | `concurrency` keyed on `ci-${{ github.ref }}` with `cancel-in-progress: true` | Explicitly specified and matches wt | S:95 R:90 A:90 D:90 |
| 5 | Certain | `ci-gate` job (`needs:[test]`, `if: always()`) as the single stable required status check | Explicitly specified and matches wt's pattern verbatim | S:95 R:80 A:90 D:90 |
| 6 | Certain | Pin actions to wt's exact SHAs (checkout v4 `34e1148...`, setup-go v5 `40f1582...`) | Explicitly specified; SHAs read directly from wt's ci.yml | S:98 R:85 A:95 D:95 |
| 7 | Certain | `go-version-file: src/go.mod`, `cache-dependency-path: src/go.sum`, `permissions: contents: read` | Matches wt and idea's own release.yml; least-privilege is the obvious default | S:90 R:85 A:95 D:90 |
| 8 | Confident | Enabling branch protection to require `ci-gate` is out of scope (repo-settings action, not code) | CI file creation is the code change; protection rules are a separate manual repo action | S:75 R:85 A:80 D:80 |
| 9 | Tentative | Memory note placement (existing `release/` domain vs. new `ci/` domain) deferred to hydrate | Both are reasonable; low blast radius, easily decided at hydrate | S:50 R:90 A:70 D:55 |

9 assumptions (7 certain, 1 confident, 1 tentative, 0 unresolved).
