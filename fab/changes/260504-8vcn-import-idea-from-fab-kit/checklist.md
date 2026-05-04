# Quality Checklist: Import idea command from fab-kit

**Change**: 260504-8vcn-import-idea-from-fab-kit
**Generated**: 2026-05-04
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Single repo-level go.mod: `src/go.mod` exists with `module github.com/sahil87/idea` as the first non-empty line, declares `go 1.22`, and requires `github.com/spf13/cobra v1.8.1`.
- [x] CHK-002 Source layout (hop convention): `src/cmd/idea/{main,add,list,show,done,reopen,edit,rm,resolve,main_test}.go` and `src/internal/idea/{idea,idea_test}.go` all exist; no `src/go/` path exists.
- [x] CHK-003 Internal import path rewritten: every `cmd/*.go` that imports the internal package uses `github.com/sahil87/idea/internal/idea`; no source file matches `github.com/sahil87/fab-kit`.
- [x] CHK-004 Source content preserved verbatim: byte-for-byte equal to fab-kit sources except for the go.mod module line and the one internal-import line in `cmd/*.go` (plus the two permitted layout/version edits in main_test.go and main.go documented in the spec).
- [x] CHK-005 build.sh copied with hop→idea substitution: `scripts/build.sh` references `bin/idea`, `./cmd/idea`, and the `built: bin/idea` echo; preserves shebang, `set -euo pipefail`, `git describe` version logic, and `-ldflags` form.
- [x] CHK-006 install.sh copied with hop→idea substitution: `scripts/install.sh` chains `./scripts/build.sh`, sets DEST to `${HOME}/.local/bin/idea`, copies `./bin/idea`, and echoes `installed: $DEST`.
- [x] CHK-007 release.sh copied verbatim: `scripts/release.sh` is byte-for-byte identical to `~/code/sahil87/hop/scripts/release.sh` (no substitutions). Verified via `diff` (exit 0).
- [x] CHK-008 justfile copied with hop→idea substitution: `justfile` contains all five recipes (`default`, `build`, `local-install`, `test`, `release bump="patch"`); test recipe runs `cd src && go test ./...`.
- [x] CHK-009 Release workflow copied with hop→idea substitution: `.github/workflows/release.yml` triggers on `tags: ["v*"]`, declares `permissions: contents: write`, builds for darwin/arm64, darwin/amd64, linux/arm64, linux/amd64, uses `softprops/action-gh-release`, references `Formula/idea.rb`.
- [x] CHK-010 Formula template copied with hop→idea substitution: `.github/formula-template.rb` declares `class Idea`, has `desc` "Capture and manage ideas from the command line", `homepage` `https://github.com/sahil87/idea`, all four download URLs reference `sahil87/idea`, `bin.install "idea"`, test invokes `#{bin}/idea --version`. All five sed placeholders (`VERSION_PLACEHOLDER`, `SHA_DARWIN_ARM64`, `SHA_DARWIN_AMD64`, `SHA_LINUX_ARM64`, `SHA_LINUX_AMD64`) appear exactly once each.
- [x] CHK-011 CLI overview spec landed: `docs/specs/overview.md` exists, contains overview, install, worktree behavior with verbatim `git rev-parse` rules, commands table, ID format/query semantics, and external-consumer integration framing.
- [x] CHK-012 Backlog format spec landed: `docs/specs/backlog-format.md` exists with the literal pattern `- [ ] [{ID}] [{issue_ids}] {YYYY-MM-DD}: {description}`, both example forms, component definitions, round-trip preservation guarantee, stability commitment.
- [x] CHK-013 Specs index updated: `docs/specs/index.md` table contains a row linking to `overview.md` and a row linking to `backlog-format.md`.
- [x] CHK-014 README at Standard depth: `README.md` has Install / Quick Start / Commands sections plus cross-links to both new specs. Strings `🚧` and `Currently bundled with fab-kit` do not appear (grep counts: 0 each).

## Behavioral Correctness

- [x] CHK-015 Tests pass after move: `cd src && go test ./...` exits 0 with `ok` for both `cmd/idea` and `internal/idea` packages.
- [x] CHK-016 Test count parity: number of test functions in new location equals fab-kit `src/go/idea/cmd/main_test.go` plus `src/go/idea/internal/idea/idea_test.go` at the time of import (6 in cmd, 52 in internal — matches fab-kit exactly).
- [x] CHK-017 Build script produces working binary: `./scripts/build.sh` exits 0 and produces an executable `bin/idea`. Stdout: `built: bin/idea (version: 6ec1e94)`.
- [x] CHK-018 Binary --help lists all seven user-facing subcommands: `add`, `list`, `show`, `done`, `reopen`, `edit`, `rm` (verified — also lists cobra-auto `completion` and `help` which are inherent to cobra).
- [x] CHK-019 Binary --version emits a version stamp from `git describe`. Output: `idea version 6ec1e94`.
- [x] CHK-020 release.sh validates dirty tree: with uncommitted changes, `./scripts/release.sh patch` exits 1 and stderr includes `Working tree not clean. Commit or stash changes first.`
- [x] CHK-021 release.sh prints usage with no args: `./scripts/release.sh` exits 0 and prints the usage text (`Usage: release.sh <patch|minor|major>` plus the three bump descriptions).
- [x] CHK-022 just lists all five recipes: `just` (with no args) lists `default`, `build`, `local-install`, `test`, `release` (verified).
- [x] CHK-023 release.yml is valid YAML and contains all four cross-compile targets in any order. The `targets` line shows `darwin/arm64 darwin/amd64 linux/arm64 linux/amd64`.
- [x] **N/A**: CHK-024 Ruby is not installed in this environment (`which ruby` → not found). Spec explicitly permits N/A in this case. File reads as syntactically plausible Ruby (class…end matched, balanced blocks).

## Scenario Coverage

- [x] CHK-025 No fab-kit references in source tree: `grep -r "fab-kit" src/ src/go.mod` returns zero matches.
- [x] CHK-026 Module independence: `cd src && go build ./...` succeeds without access to fab-kit (fab-kit is not a dependency of go.mod and tests pass).
- [x] CHK-027 Specs are self-contained: `docs/specs/overview.md` is readable in isolation; the only `fab-kit` reference is on line 54 in the External-Consumer Integration section, framed as one example consumer (not as a defining integration).
- [x] CHK-028 README is self-contained: a fresh reader can install (Homebrew tap or `./scripts/install.sh`) and run their first command (`idea "..."`) without leaving the README.

## Edge Cases & Error Handling

- [x] CHK-029 Diff scope is minimal: `diff` against fab-kit source for each ported file shows only: (a) module line in go.mod, (b) one internal-import line in each cmd/*.go (8 files), (c) layout-induced `"cmd"` → `"cmd", "idea"` path edit in main_test.go, (d) version wiring (`var version = "dev"` + comment + `Version: version,`) in main.go. internal/idea/{idea.go,idea_test.go} are byte-identical (zero diff).
- [x] CHK-030 Placeholders intact in formula template: all five sed placeholders (`VERSION_PLACEHOLDER`, `SHA_DARWIN_ARM64`, `SHA_DARWIN_AMD64`, `SHA_LINUX_ARM64`, `SHA_LINUX_AMD64`) are present exactly once.

## Code Quality

- [x] CHK-031 Readability and maintainability: ported code retains its original structure; the only edits are the four spec-permitted categories — no inline refactors or comment tweaks beyond the documented version-stamping comment.
- [x] CHK-032 Follow existing project patterns: layout matches hop's convention (`src/{go.mod, cmd/<bin>/, internal/<bin>/}`); version-stamping wiring mirrors wt's pattern verbatim (declaration comment, `var version = "dev"`, `Version: version,` field placement).
- [x] CHK-033 No god functions introduced: this change does not add new code, so no functions exceed 50 lines as a result of this change.
- [x] CHK-034 No duplicating existing utilities: this change does not add new code, so no utility is duplicated.
- [x] CHK-035 No magic strings or numbers introduced: all string substitutions in scripts/CI/formula are at well-defined replacement points (binary name, URL, package path, message text).
- [x] CHK-036 Pattern consistency: copy/substitute operations are mechanical; no semantic edits to ported sources.
- [x] CHK-037 No unnecessary duplication: `scripts/release.sh` is shared verbatim with hop's pattern (one-source-of-truth concept retained at the convention level even though the file is duplicated across repos by design).

## Rework Cycle 1 Deliverables

<!-- Added during cycle 2 review to cover items introduced by Phase 5 (T030–T035). -->

- [x] CHK-038 `.gitignore` ignores Go build artifacts: file exists at repo root, contains entries for `bin/`, `dist/`, `*.test`, `*.out`, `coverage.*`, `*.coverprofile`, `profile.cov`, `go.work`, `go.work.sum`, `*.exe`, `*.dll`, `*.so`, `*.dylib`. Verified `git check-ignore bin/idea` exits 0 (the built binary is now ignored).
- [x] CHK-039 Stray `scratch_dirty_marker.txt` removed from repo root: `ls scratch_dirty_marker.txt` exits non-zero ("No such file or directory").
- [x] CHK-040 `LICENSE` file exists at repo root: MIT license text matching hop's pattern; copyright line reads exactly `Copyright (c) 2026 Sahil Ahuja`.
- [x] CHK-041 `docs/specs/backlog-format.md` issue-ID clarification: a dedicated "Opaque Pass-Through Fields" section now distinguishes the components `idea` parses (`[ID]`, date, description) from the slot it preserves verbatim without parsing (`[issue_ids]`); the with-issue-ID example annotates that `idea` passes the bracket through unchanged; the Stability Commitment explicitly excludes `[issue_ids]` semantics from `idea`'s contract; cross-references Constitution principle I via a link in the closing paragraph.
- [x] CHK-042 `README.md` worktree-behavior line: a sub-bullet under Commands describes default-current-worktree behavior, the `--main` flag, the `--file` override, and the `IDEAS_FILE` environment variable, with a cross-link to `docs/specs/overview.md` for full resolution rules.

## Rework Cycle 2 Deliverables

<!-- Added during cycle 3 review to cover items introduced by Phase 6 (T036–T038). -->

- [x] CHK-043 `docs/specs/backlog-format.md` accurately represents that lines containing the optional `[issue_ids]` bracket are invisible to ALL idea operations (not just the slot). Verified: doc now uses explicit Shape A (idea-managed, no second bracket — parsed/queried/round-tripped) vs Shape B (any line with extra content between `[{ID}]` and the date — inert pass-through, invisible to `list`/`show`/`done`/`edit`/`rm`, round-trip preserved per Constitution principle I) framing. The with-issue-ID example annotation explicitly states "this form is preserved verbatim by `idea` on round-trip, but is NOT visible to any `idea` query — it is treated as inert content." The Stability Commitment distinguishes Shape A's full parsing contract from Shape B's round-trip-only contract. Live-tested: a backlog with one Shape A and one Shape B line — `idea list` shows only the Shape A line; `idea show <Shape-B-id>` exits 1; `idea show <Shape-A-id>` exits 0.
