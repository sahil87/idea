# CI Pipeline

`idea` runs pre-merge continuous integration via `.github/workflows/ci.yml`: every pull request and every push to `main` is gated on a gofmt/vet/test run. This is distinct from the tag-driven *release* pipeline (`../release/pipeline.md`) — CI validates changes **before** merge; release ships artifacts **after** a tag is pushed. The two never overlap (CI does not build binaries, touch tags, or update the Homebrew tap).

The workflow mirrors the sibling `wt` repo's `ci.yml` verbatim — `idea` and `wt` share an identical `src/`-rooted single-`go.mod` layout, so the file is drop-in with no path changes. Reproducing wt's proven workflow (same action SHAs, same `src`-rooted steps) avoids bespoke-workflow drift between the two repos.

## Triggers

```yaml
on:
  push:
    branches:
      - main
  pull_request:
```

Runs on every pull request and on every push to `main`. Pushes to non-`main` branches without an associated PR do not trigger it.

## Permissions & Concurrency

- `permissions: contents: read` — least-privilege; the workflow only reads source (contrast with `release.yml`, which needs `contents: write`).
- `concurrency: { group: ci-${{ github.ref }}, cancel-in-progress: true }` — keyed on the ref, so rapid pushes to the same branch (PR or `main`) cancel in-flight runs and save CI minutes.

## `test` job

Runs on `ubuntu-latest`. Steps:

1. `actions/checkout` pinned to v4 SHA `34e114876b0b11c390a56381ad16ebd13914f8d5`.
2. `actions/setup-go` pinned to v5 SHA `40f1582b2485089dde7abd97c1529aa768e1baff`, configured with `go-version-file: src/go.mod` (single source of truth for the Go version — same convention as `release.yml`) and `cache-dependency-path: src/go.sum`.
3. **gofmt** (`working-directory: src`): runs `gofmt -l .` and exits 1 if any file is unformatted, printing the offending files plus the `Run: (cd src && gofmt -w .)` remediation hint to stderr.
4. **go vet** (`working-directory: src`): `go vet ./...`.
5. **go test** (`working-directory: src`): `go test ./...`.

All three checks run from `working-directory: src` because the module root is `src/` (`src/go.mod`, `src/go.sum`) — see `../cli/structure.md`.

## `ci-gate` job

```yaml
ci-gate:
  needs: [test]
  runs-on: ubuntu-latest
  if: always()
```

A single stable status check to pin branch protection to. It verifies `needs.test.result == 'success'` and exits 1 otherwise (printing the failing result to stderr). `if: always()` guarantees the gate runs — and can fail explicitly — even when `test` failed, cancelled, or was skipped, avoiding the confusing "skipped required check blocks merge" state.

Branch protection SHOULD pin to `ci-gate` rather than `test` directly: the required-check name then stays constant even if `test` is later split or renamed — only `ci-gate`'s `needs:` list tracks that, not the repo ruleset.

## File index

- `.github/workflows/ci.yml` — the CI workflow (gofmt/vet/test `test` job + `ci-gate` aggregation job).

## Design Decisions

- **260610-k6pc — added pre-merge CI by mirroring `wt/.github/workflows/ci.yml` verbatim.** Before this change `idea` had only `release.yml` (tag-driven); pull requests ran no automated tests, vet, or gofmt gate, leaving the constitution's quality guarantees (Principle V tests, gofmt-clean source) unenforced. Porting wt's battle-tested workflow (identical `src/`-rooted layout, same pinned action SHAs) was the lowest-risk option and avoids drift between the two repos. *Rejected*: authoring a bespoke workflow (reinvents a working pattern, risks divergence).
- **260610-k6pc — `ci-gate` is the pinned required check, not `test` directly.** Keeps the required-check name constant across future `test` refactors — only `ci-gate`'s `needs:` list tracks job splits/renames, not the repo ruleset. `if: always()` makes the gate fail explicitly rather than being skipped when `test` does not succeed.
- **260610-k6pc — fixed a pre-existing gofmt violation in `src/internal/idea/idea_test.go` in this same PR.** The struct-field alignment in `TestResolveFilePath` was not gofmt-clean on `main`; without a fix the new gofmt gate would fail on its first run. The fix is a pure `gofmt -w` whitespace re-alignment (no behavioral change), applied so CI is green on arrival.

## Follow-ups (not done by the change)

- **Enable branch protection to require the `ci-gate` check.** This is a repo-settings action (GitHub ruleset), not a code change, so it was out of scope for the change. The workflow now exists and emits a stable `ci-gate` status check; the maintainer must pin the ruleset to it for the gate to actually block merges.

## Cross-references

- Tag-driven release pipeline (distinct concern — ships artifacts after a tag push): `../release/pipeline.md`.
- Source layout the `working-directory: src` steps and `go-version-file: src/go.mod` assume (`src/{go.mod, cmd/idea/, internal/idea/}`): `../cli/structure.md`.
- Constitution Principle V (table-driven tests) and `stage_directives.apply` (gofmt) — the quality bar this CI enforces: `fab/project/constitution.md`, `fab/project/config.yaml`.
