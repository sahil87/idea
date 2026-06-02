# Spec: Import idea command from fab-kit

**Change**: 260504-8vcn-import-idea-from-fab-kit
**Created**: 2026-05-04
**Affected memory**: `docs/memory/release/pipeline.md`, `docs/memory/cli/structure.md`

## Non-Goals

- **Not modifying fab-kit** — removal of `src/go/idea/` from `~/code/sahil87/fab-kit/` is a separate change in that repo. This change leaves fab-kit untouched.
- **Not changing idea's CLI behavior** — same commands, same flags, same exit codes, same backlog file format. This is a structural move, not a redesign.
- **Not preserving per-file commit history** — code lands as a flat snapshot (single import commit). History remains accessible in fab-kit via `git log -- src/go/idea/`.
- **Not introducing new dependencies** — direct deps stay limited to `github.com/spf13/cobra` (per Constitution principle on dependency discipline).
- **Not adding new tests** — existing tests come over and must pass after the module-path rewrite. Test additions are out of scope.
- **Not cutting a release in this change** — first tag push is a separate user action after merge.

## Source Layout: Go module structure

### Requirement: Single repo-level `go.mod`

The repository SHALL contain exactly one Go module declared at `src/go.mod`. The module path SHALL be `github.com/sahil87/idea`. The module SHALL declare `go 1.22` and require `github.com/spf13/cobra v1.8.1` directly (with `pflag` and `mousetrap` carried as indirect deps from `go.sum`).

#### Scenario: go.mod after import
- **GIVEN** the import is complete
- **WHEN** a developer runs `cat src/go.mod`
- **THEN** the first line MUST be `module github.com/sahil87/idea`
- **AND** the file MUST NOT contain any reference to `fab-kit` in its module path

#### Scenario: Module independence
- **GIVEN** the import is complete
- **WHEN** a developer runs `cd src && go build ./...` from a fresh clone
- **THEN** the build MUST succeed without requiring access to the fab-kit repo or any private module proxy

### Requirement: Source files placed under hop's layout

The Go source SHALL be organized as follows under `src/`:
- `cmd/idea/` SHALL contain the cobra entry point and one file per subcommand: `main.go`, `add.go`, `list.go`, `show.go`, `done.go`, `reopen.go`, `edit.go`, `rm.go`, `resolve.go`, plus `main_test.go`.
- `internal/idea/` SHALL contain the package logic: `idea.go` and `idea_test.go`.
- No source files SHALL be placed at any path matching `src/go/idea/...` (the fab-kit nested layout SHALL NOT be carried over).

#### Scenario: Layout matches hop convention
- **GIVEN** the import is complete
- **WHEN** a developer lists `src/`
- **THEN** they SHALL see `go.mod`, `go.sum`, `cmd/`, and `internal/` at the top level
- **AND** `cmd/idea/main.go` SHALL exist
- **AND** `internal/idea/idea.go` SHALL exist
- **AND** no path `src/go/` SHALL exist

### Requirement: Internal import path rewritten

The single internal-package import in `cmd/*.go` SHALL be `github.com/sahil87/idea/internal/idea`. No source file SHALL contain the string `github.com/sahil87/fab-kit` after the import.

#### Scenario: No fab-kit references in source
- **GIVEN** the import is complete
- **WHEN** a developer runs `grep -r "fab-kit" src/ go.mod`
- **THEN** zero matches SHALL be returned

### Requirement: Source content preserved verbatim

Every file copied from `~/code/sahil87/fab-kit/src/go/idea/` SHALL retain its content byte-for-byte except for these four edits, all of which are structural consequences of becoming an independently-released binary:

(a) the `module` line in `go.mod` (`github.com/sahil87/fab-kit/src/go/idea` → `github.com/sahil87/idea`);
(b) the single internal-package import in any `cmd/*.go` file (`"github.com/sahil87/fab-kit/src/go/idea/internal/idea"` → `"github.com/sahil87/idea/internal/idea"`);
(c) hard-coded subpaths of the source tree that must adjust because the cobra entry package moved from `cmd/` (under fab-kit's per-binary go.mod layout) to `cmd/idea/` (under hop's single-go.mod layout). Concretely: `cmd/idea/main_test.go` builds the binary under test by joining a discovered module root with `"cmd"`; this string SHALL change to `"cmd/idea"` to point at the relocated cobra entry package. Any other layout-induced path reference SHALL be edited under this same rule;
(d) version-stamping wiring in `cmd/idea/main.go`, mirroring the pattern established by `wt` (`~/code/sahil87/wt/src/cmd/wt/main.go`). Specifically: declare a package-level `var version = "dev"` (with the comment `// version is the binary version, overridden via -ldflags "-X main.version=..." at build time.`) and add `Version: version,` to the root cobra command's struct literal. This wiring is required because (1) idea is now released independently with `git describe`-derived versions injected via `scripts/build.sh`'s `-ldflags '-X main.version=${VERSION}'`, (2) the imported `.github/formula-template.rb`'s test block invokes `#{bin}/idea --version`, and (3) without this wiring the `-ldflags` injection silently no-ops and `--version` is unrecognized. fab-kit's idea did not need this wiring because it shipped inside fab-kit's archive, where the parent kit owned versioning.

No semantic edits, refactors, formatting changes, comment edits, function renames, or behavior changes SHALL be made beyond the four categories above.

#### Scenario: Functional behavior unchanged
- **GIVEN** the import is complete
- **WHEN** a developer runs `cd src && go test ./...`
- **THEN** all tests SHALL pass
- **AND** the test count SHALL match the test count in fab-kit's `src/go/idea/` at the time of the import

#### Scenario: Diff scope is minimal
- **GIVEN** the import is complete
- **WHEN** a developer compares each ported file to its fab-kit source via `diff`
- **THEN** the only differences SHALL fall under the four categories above (module line in `go.mod`; internal-import line in `cmd/*.go`; layout-induced subpath strings such as `"cmd"` → `"cmd/idea"` in `main_test.go`; version-stamping wiring in `cmd/idea/main.go`)

## Build & Release: Scripts and CI

### Requirement: Build script copied from hop

`scripts/build.sh` SHALL be copied from `~/code/sahil87/hop/scripts/build.sh` with `hop` → `idea` substituted in three locations: the binary output path (`bin/idea`), the cmd path (`./cmd/idea`), and the echoed message. The script's structure (shebang, `set -euo pipefail`, `git describe`-derived `VERSION`, `mkdir -p bin`, `cd src`, `go build` with `-ldflags`) SHALL be preserved verbatim.

#### Scenario: build.sh produces a binary
- **GIVEN** a clean checkout
- **WHEN** a developer runs `./scripts/build.sh`
- **THEN** `bin/idea` SHALL exist and be executable
- **AND** the script's stdout SHALL include `built: bin/idea (version: <version>)`

### Requirement: Install script copied from hop

`scripts/install.sh` SHALL be copied from `~/code/sahil87/hop/scripts/install.sh` with `hop` → `idea` substituted in three locations: the script chain to `./scripts/build.sh`, the destination path (`${HOME}/.local/bin/idea`), and the echoed message. Structure preserved verbatim.

#### Scenario: install.sh installs to ~/.local/bin
- **GIVEN** a successful build
- **WHEN** a developer runs `./scripts/install.sh`
- **THEN** `~/.local/bin/idea` SHALL exist and be executable
- **AND** the script's stdout SHALL include `installed: <DEST>`

### Requirement: Release script copied verbatim

`scripts/release.sh` SHALL be copied from `~/code/sahil87/hop/scripts/release.sh` with no edits. The script is generic — it takes `<patch|minor|major>`, computes the next semver from `git describe --tags --abbrev=0`, validates a clean working tree and named branch, creates the tag, and pushes it. All logic, error messages, and usage text remain identical.

#### Scenario: release.sh validates inputs
- **GIVEN** the script is invoked with no arguments
- **WHEN** the script runs
- **THEN** it SHALL print the usage text and exit 0

#### Scenario: release.sh refuses dirty tree
- **GIVEN** the working tree has uncommitted changes
- **WHEN** the developer runs `./scripts/release.sh patch`
- **THEN** the script SHALL exit non-zero
- **AND** stderr SHALL include `Working tree not clean`

### Requirement: justfile copied from hop

The repository SHALL contain a top-level `justfile` copied from `~/code/sahil87/hop/justfile` with `hop` → `idea` substituted. It SHALL define five recipes: `default` (lists recipes via `just --list`), `build` (delegates to `./scripts/build.sh`), `local-install` (delegates to `./scripts/install.sh`), `test` (runs `cd src && go test ./...`), and `release bump="patch"` (delegates to `./scripts/release.sh {{bump}}`).

#### Scenario: just lists recipes
- **GIVEN** `just` is installed
- **WHEN** a developer runs `just` with no arguments
- **THEN** the output SHALL list all five recipes

### Requirement: GitHub Actions release workflow copied from hop

`.github/workflows/release.yml` SHALL be copied from `~/code/sahil87/hop/.github/workflows/release.yml` with `hop` → `idea` substituted in all binary-name and tarball-name positions (e.g., `hop-${os}-${arch}` → `idea-${os}-${arch}`, `./cmd/hop` → `./cmd/idea`, `Formula/hop.rb` → `Formula/idea.rb`). The workflow SHALL preserve verbatim: trigger (`tags: ["v*"]`), permissions, the four cross-compile targets (darwin/arm64, darwin/amd64, linux/arm64, linux/amd64), the release-base detection logic for minor releases, the use of `softprops/action-gh-release`, and the homebrew-tap update flow.

#### Scenario: Workflow yaml is valid
- **GIVEN** `.github/workflows/release.yml` exists
- **WHEN** a developer parses the file as YAML
- **THEN** parsing SHALL succeed
- **AND** the `on.push.tags` array SHALL contain `"v*"`

#### Scenario: All four platform builds present
- **GIVEN** `.github/workflows/release.yml` exists
- **WHEN** a developer inspects the cross-compile step
- **THEN** the `targets` variable SHALL list `darwin/arm64 darwin/amd64 linux/arm64 linux/amd64` (in any order)

### Requirement: Homebrew formula template copied from hop

`.github/formula-template.rb` SHALL be copied from `~/code/sahil87/hop/.github/formula-template.rb` with substitutions: class name `Hop` → `Idea`, `desc` updated to "Capture and manage ideas from the command line", `homepage` updated to `https://github.com/sahil87/idea`, all four download URLs updated to point at `sahil87/idea` releases with `idea-${os}-${arch}.tar.gz` filenames, the `bin.install` line updated to `bin.install "idea"`, and the `test` block updated to invoke `#{bin}/idea --version`. The four `VERSION_PLACEHOLDER`, `SHA_DARWIN_ARM64`, `SHA_DARWIN_AMD64`, `SHA_LINUX_ARM64`, `SHA_LINUX_AMD64` placeholders SHALL be preserved verbatim — they are sed targets in the workflow.

#### Scenario: Formula template is valid Ruby
- **GIVEN** `.github/formula-template.rb` exists
- **WHEN** a developer runs `ruby -c .github/formula-template.rb` (if Ruby is available)
- **THEN** the syntax check SHALL pass

#### Scenario: Placeholders intact
- **GIVEN** `.github/formula-template.rb` exists
- **WHEN** a developer greps for placeholders
- **THEN** all five placeholders (`VERSION_PLACEHOLDER`, `SHA_DARWIN_ARM64`, `SHA_DARWIN_AMD64`, `SHA_LINUX_ARM64`, `SHA_LINUX_AMD64`) SHALL be present exactly once each

## Specs: Documentation deliverables

### Requirement: CLI overview spec landed at `docs/specs/overview.md`

`docs/specs/overview.md` SHALL be created as a standalone specification of the `idea` CLI, derived from the "## idea (Backlog Management)" section of `~/code/sahil87/fab-kit/docs/specs/packages.md` (lines 94–143). The spec SHALL be re-framed as idea-as-its-own-tool — fab-kit-context references SHALL be removed or rewritten:
- The "Binary: src/go/idea/ ..." line SHALL be rewritten to reference `src/cmd/idea/` and Homebrew installation via `sahil87/tap/idea`.
- The Distribution paragraph (about fab-kit's per-platform release archive) SHALL be removed.
- The "Integration with Fab" subsection SHALL be retained but rewritten to describe a generic external-consumer contract: any tool can read `fab/backlog.md` to discover backlog IDs; `/fab-new` from fab-kit is mentioned as one example consumer (not the defining integration).

The spec SHALL contain (in order): an overview/purpose paragraph, binary location and installation, worktree behavior (current worktree default, `--main` opt-in, `git rev-parse` resolution rules — preserved verbatim), the commands table, ID format and query semantics, and the external-consumer integration section.

#### Scenario: Spec is self-contained
- **GIVEN** `docs/specs/overview.md` exists
- **WHEN** a developer reads the file in isolation
- **THEN** they SHALL be able to understand what idea does, how to install it, and how to use it
- **AND** they SHALL NOT need to read any fab-kit doc to follow it

#### Scenario: No fab-kit dependency framing
- **GIVEN** `docs/specs/overview.md` exists
- **WHEN** a developer greps for "fab-kit"
- **THEN** matches SHALL appear only in the external-consumer subsection (e.g., "fab-kit's `/fab-new` is one example consumer")
- **AND** no match SHALL describe idea as part of, included in, or distributed by fab-kit

### Requirement: Backlog format spec landed at `docs/specs/backlog-format.md`

`docs/specs/backlog-format.md` SHALL be created from the relevant rows of `~/code/sahil87/fab-kit/docs/specs/naming.md` (lines 60–73). The spec SHALL document the `fab/backlog.md` line format as idea's primary public contract. Sections in order: pattern (`- [ ] [{ID}] [{issue_ids}] {YYYY-MM-DD}: {description}` — issue IDs optional), examples (one with issue ID, one without), component definitions, round-trip preservation guarantee (per Constitution principle I — non-idea lines preserved verbatim), and a stability commitment statement that the format is the public API for external consumers.

#### Scenario: Pattern is documented exactly
- **GIVEN** `docs/specs/backlog-format.md` exists
- **WHEN** a developer reads the pattern definition
- **THEN** the literal pattern string `- [ ] [{ID}] [{issue_ids}] {YYYY-MM-DD}: {description}` SHALL appear verbatim
- **AND** both example forms (with and without issue IDs) SHALL appear

### Requirement: Specs index updated

`docs/specs/index.md` SHALL be updated to include rows for both new specs in the `| Spec | Description |` table. Each row SHALL link to the spec file and provide a one-line description.

#### Scenario: Index lists both specs
- **GIVEN** `docs/specs/index.md` is updated
- **WHEN** a developer reads the table
- **THEN** they SHALL find a row linking to `overview.md` describing the idea CLI
- **AND** a row linking to `backlog-format.md` describing the backlog file format

## README

### Requirement: README updated to "Standard" depth

`README.md` SHALL be rewritten to follow the "Standard" depth pattern (mirroring hop's README depth). The "🚧 Standalone repo in progress. Currently bundled with fab-kit" notice SHALL be removed. The existing top-level header ("# idea" and "Part of @sahil87's open source toolkit" line) SHALL be preserved.

The rewritten README SHALL contain (in order):
1. Title and toolkit pointer (preserved).
2. One-line description (preserved or refined).
3. **Install** section: Homebrew tap command (`brew install sahil87/tap/idea`) and the manual `./scripts/install.sh` alternative.
4. **Quick Start** section: 3–4 example commands showing `idea "text"`, `idea list`, `idea show <id>`, `idea done <id>`.
5. **Commands** section: a brief enumeration of all subcommands (or a link to `docs/specs/overview.md`).
6. A one-line mention of fab-kit `/fab-new` integration via `fab/backlog.md`, with a link to `docs/specs/backlog-format.md` for the format contract.
7. Cross-links to `docs/specs/overview.md` and `docs/specs/backlog-format.md`.

The README SHALL NOT include badges, screenshots, contributing notes, or a license badge (those belong to the "Full" depth, which is out of scope).

#### Scenario: README is self-contained
- **GIVEN** `README.md` is updated
- **WHEN** a developer lands on the repo for the first time
- **THEN** they SHALL be able to install the tool and run their first command without leaving the README

#### Scenario: No "in progress" notice
- **GIVEN** `README.md` is updated
- **WHEN** a developer reads the file
- **THEN** the string "🚧" SHALL NOT appear
- **AND** the string "Currently bundled with fab-kit" SHALL NOT appear

## Memory: Hydrate deliverables

### Requirement: Release pipeline memory file created

At hydrate, `docs/memory/release/pipeline.md` SHALL be created. It SHALL document: the tag-driven release flow (`scripts/release.sh` cuts the tag, GitHub Actions takes over from the tag push), the cross-compile matrix (darwin/linux × arm64/amd64), the GitHub Release creation step, and the contract with `sahil87/homebrew-tap` (which the workflow updates by sed-substituting `formula-template.rb`). It SHALL note that the `HOMEBREW_TAP_TOKEN` secret must be configured on this repo and that the same token already powers hop and fab-kit releases.

#### Scenario: Memory file describes the full pipeline
- **GIVEN** the change has reached hydrate
- **WHEN** a developer reads `docs/memory/release/pipeline.md`
- **THEN** the file SHALL describe each stage from `release.sh` invocation through homebrew-tap update
- **AND** the file SHALL reference the `HOMEBREW_TAP_TOKEN` secret requirement

### Requirement: CLI structure memory file created

At hydrate, `docs/memory/cli/structure.md` SHALL be created. It SHALL document the source layout decision: `src/cmd/idea/` for the cobra entry point, `src/internal/idea/` for the logic package, single `src/go.mod` with module path `github.com/sahil87/idea`. It SHALL note this follows hop's convention (single repo-level go.mod) rather than fab-kit's per-binary go.mod layout, and explain when each pattern is appropriate (single-binary repo → hop layout; multi-binary monorepo → fab-kit layout).

#### Scenario: Memory file documents the layout
- **GIVEN** the change has reached hydrate
- **WHEN** a developer reads `docs/memory/cli/structure.md`
- **THEN** the file SHALL describe the `src/{go.mod, cmd/idea/, internal/idea/}` layout
- **AND** the file SHALL state the module path
- **AND** the file SHALL contrast with the alternative (fab-kit's per-binary layout)

### Requirement: Memory index updated

`docs/memory/index.md` SHALL be updated at hydrate to include rows for the two new memory domains (`release` and `cli`) and pointers to the new files.

#### Scenario: Memory index lists new domains
- **GIVEN** the change has reached hydrate
- **WHEN** a developer reads `docs/memory/index.md`
- **THEN** the table SHALL contain a row for the `release` domain pointing at `pipeline.md`
- **AND** a row for the `cli` domain pointing at `structure.md`

## Validation

### Requirement: Tests pass after the move

After the import, all tests in the new location SHALL pass via `cd src && go test ./...`. The set of test cases SHALL match the set in fab-kit's `src/go/idea/` at the time of the import (no tests added, no tests removed).

#### Scenario: Test count parity
- **GIVEN** the import is complete
- **WHEN** a developer counts test functions in `src/cmd/idea/` and `src/internal/idea/`
- **THEN** the count SHALL equal the count in `~/code/sahil87/fab-kit/src/go/idea/cmd/` and `~/code/sahil87/fab-kit/src/go/idea/internal/idea/` at the time of the import

#### Scenario: Tests pass in the new repo
- **GIVEN** the import is complete
- **WHEN** a developer runs `cd src && go test ./...`
- **THEN** the exit code SHALL be 0
- **AND** stdout SHALL show `ok` for both `cmd/idea` and `internal/idea` packages

### Requirement: Binary builds and runs

After the import, the `idea` binary SHALL build via `./scripts/build.sh` and respond to `--help` and `--version` invocations.

#### Scenario: Binary responds to --help
- **GIVEN** a successful build
- **WHEN** a developer runs `./bin/idea --help`
- **THEN** the exit code SHALL be 0
- **AND** stdout SHALL list the seven subcommands: `add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`

#### Scenario: Binary responds to --version
- **GIVEN** a successful build via `./scripts/build.sh`
- **WHEN** a developer runs `./bin/idea --version`
- **THEN** the exit code SHALL be 0
- **AND** stdout SHALL include the version stamp from `git describe`

## Design Decisions

1. **Layout follows hop, not fab-kit**: Use `src/{go.mod, cmd/idea/, internal/idea/}` (single repo-level go.mod).
   - *Why*: idea is a single binary; hop's layout is the proven shape for single-binary Go repos in this account. Every piece of release machinery (`scripts/`, justfile, CI workflow, formula template) is built around this layout — copying hop's machinery requires hop's layout.
   - *Rejected*: fab-kit's per-binary `src/go/<bin>/{cmd,internal,go.mod}` layout. Justified in fab-kit because fab-kit ships four binaries from one monorepo with shared kit/templates dirs; that justification doesn't apply here.

2. **Flat copy, not history preservation**: Code lands as a single import commit; no `git filter-repo --subdirectory-filter`.
   - *Why*: idea has a small file count (~12 files) and shallow history in fab-kit. The blame-continuity gain from `filter-repo` doesn't justify its complexity. History remains accessible in fab-kit via `git log -- src/go/idea/`.
   - *Rejected*: `git filter-repo --subdirectory-filter src/go/idea` against a fab-kit clone, then merge into this repo with path rewrites. Higher complexity (filter-repo + path rewrites in same pass + cross-repo merge) for small marginal gain.

3. **Two-spec split, not one combined CLI doc**: `docs/specs/overview.md` for tool behavior, `docs/specs/backlog-format.md` for the file format contract.
   - *Why*: The file format is the durable public API — external consumers (fab-kit's `/fab-new`, future tools) depend on its stability. Separating it into its own spec signals that durability and lets the format spec evolve on a different cadence than the tool spec.
   - *Rejected*: A single `docs/specs/cli.md` mixing both. Loses the API/implementation separation; format-stability commitments get buried inside tool-behavior prose.

4. **Constitution unchanged**: `fab/project/constitution.md` is preserved as-is.
   - *Why*: The constitution (dated 2026-05-03) already encodes idea's six design principles correctly. No principle changes as a result of the structural move.
   - *Rejected*: Refreshing the "Last Amended" date or adding a "II.5 Standalone Repo" principle. The principles aren't changing, so neither edit is warranted.

5. **No memory files generated until hydrate**: Memory files appear only after the apply stage, written by the hydrate sub-agent.
   - *Why*: Per the fab pipeline, memory captures *what actually happened*; writing it at spec time would invert that ordering.
   - *Rejected*: Pre-creating empty memory files at spec stage as scaffolding. Adds no value; pollutes the change.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Module path is `github.com/sahil87/idea` (matches repo URL) | Confirmed from intake #1; verified against the convention used by hop (`github.com/sahil87/hop`). | S:95 R:80 A:95 D:95 |
| 2 | Certain | Source layout: `src/{go.mod, cmd/idea/, internal/idea/}` (single repo-level go.mod, hop convention) | Confirmed from intake #2; verified against `~/code/sahil87/hop/src/` structure. | S:100 R:75 A:95 D:100 |
| 3 | Certain | `scripts/release.sh` copied verbatim from hop (no edits) | Confirmed from intake #3; verified by reading hop's release.sh — fully generic, no hop-name references. | S:90 R:85 A:100 D:100 |
| 4 | Certain | `scripts/build.sh` and `scripts/install.sh` copied with `hop` → `idea` substitution at three locations each | Confirmed from intake #4; substitution targets enumerated in spec (`bin/idea`, `./cmd/idea`, echoed message; `${HOME}/.local/bin/idea` for install). | S:90 R:85 A:100 D:100 |
| 5 | Certain | `.github/workflows/release.yml` and `.github/formula-template.rb` are landed alongside the scripts | Confirmed from intake #5; user explicitly answered "yes" to including the CI workflow. Formula template referenced by the workflow. | S:100 R:75 A:95 D:100 |
| 6 | Certain | Two specs: `docs/specs/overview.md` (CLI behavior) + `docs/specs/backlog-format.md` (file format) | Upgraded from intake #9 (was Certain) — confirmed by user's "ok" on the two-file split. Maps directly to user's "idea section + naming conventions". | S:95 R:90 A:80 D:95 |
| 7 | Certain | Constitution stays as-is — no edits to `fab/project/constitution.md` | Confirmed from intake #10; verified by re-reading constitution. | S:95 R:85 A:90 D:95 |
| 8 | Certain | Only internal cross-package import in idea's code is its own `internal/idea`; module-path rewrite is the only required source edit | Confirmed from intake #7; verified by full-tree grep of fab-kit `src/go/idea/`. | S:100 R:90 A:100 D:100 |
| 9 | Certain | fab-kit will not be modified in this change | Confirmed from intake #8; explicit Non-Goal in this spec. | S:100 R:80 A:100 D:100 |
| 10 | Certain | Code lands as a flat copy (single import commit), not via `git filter-repo` | Upgraded from intake #13; user confirmed flat copy. | S:95 R:55 A:60 D:95 |
| 11 | Certain | README rewrite at "Standard" depth: Install / Quick Start / Commands sections + cross-links to specs + one-line fab-kit `/fab-new` integration mention. No badges, no contributing guide. | Upgraded from intake #14; user confirmed "Standard" over "Minimum"/"Full". Mirrors hop's README depth. | S:90 R:90 A:75 D:90 |
| 12 | Certain | `HOMEBREW_TAP_TOKEN` secret and `sahil87/homebrew-tap` repo already exist (same infra as hop and fab-kit) | Upgraded from intake #15; user confirmed both exist. | S:95 R:60 A:90 D:95 |
| 13 | Confident | Two memory files at hydrate: `docs/memory/release/pipeline.md` and `docs/memory/cli/structure.md` | Confirmed from intake #11; user kept the two-file plan. The release pipeline involves an external system (homebrew-tap) and a token, which warrants memory; CLI source layout records the chosen convention for future reference. | S:75 R:70 A:75 D:75 |
| 14 | Confident | First-version tag is `v0.1.0` | Confirmed from intake #12; defensible default. release.sh computes the next bump from current state, so this is easily overridden at release time and does not affect any code in this change. | S:55 R:90 A:75 D:60 |
| 15 | Certain | No new dependencies introduced; direct deps stay limited to `cobra` (per Constitution dependency-discipline principle) | Verified by reading fab-kit's `src/go/idea/go.mod` — only direct dep is cobra. Constitution principle prohibits new deps without justification. | S:100 R:85 A:95 D:100 |
| 16 | Certain | No CLI behavior changes — same commands, flags, exit codes, file format | Stated in Non-Goals; reinforced by intake's "no behavior changes" language. | S:100 R:75 A:95 D:100 |
| 17 | Confident | All seven idea subcommands (`add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`) come over plus `resolve` (visible in `cmd/`) | Verified by `ls ~/code/sahil87/fab-kit/src/go/idea/cmd/` — files: add, done, edit, list, main, reopen, resolve, rm, show. The `resolve` subcommand wasn't enumerated in the user-facing commands table in `packages.md`; treating it as an internal-helper subcommand. May or may not be exposed in `--help`. | S:70 R:85 A:75 D:80 |
| 18 | Certain | Test count parity: same number of test functions in new location as in fab-kit `src/go/idea/` at import time | Stated as a validation requirement; enforces the "no test additions/removals" Non-Goal. | S:100 R:85 A:95 D:100 |
| 19 | Certain | The `--help` output enumerates the seven user-facing subcommands (matches the table in fab-kit's packages.md) | Verified expectation from the imported source — cobra auto-generates `--help` from registered subcommands; behavior unchanged. | S:90 R:80 A:90 D:95 |
| 20 | Certain | The "Source content preserved verbatim" requirement permits layout-induced path edits in test helpers — specifically `cmd/idea/main_test.go`'s hard-coded `"cmd"` subpath becomes `"cmd/idea"` because the cobra entry package moved when adopting hop's single-go.mod layout. | Discovered during apply (T017): the in-test `go build` step in `main_test.go` references `filepath.Join(findModuleRoot(t), "cmd")` which was correct under fab-kit's per-binary layout (`src/go/idea/{go.mod, cmd/}`) but resolves to a non-package directory under hop's layout (`src/{go.mod, cmd/idea/, internal/idea/}`). User confirmed amendment over alternatives (refactoring main_test.go to self-locate via `runtime.Caller`, or flattening to `src/cmd/main.go` and breaking hop-layout). One-character path edit, semantically identical to the import-line rewrite already permitted. | S:95 R:80 A:90 D:95 |
| 21 | Certain | The "Source content preserved verbatim" requirement also permits version-stamping wiring in `cmd/idea/main.go`: declare `var version = "dev"` plus a comment, and add `Version: version,` to the root cobra command. Pattern copied from `~/code/sahil87/wt/src/cmd/wt/main.go`. | Discovered during apply (T025): `./bin/idea --version` exited with `unknown flag: --version` because fab-kit's idea source has no version wiring. Build script's `-ldflags '-X main.version=${VERSION}'` was a no-op against the unmodified source, and the imported `.github/formula-template.rb`'s `#{bin}/idea --version` test would fail Homebrew's install-time check. User directed: "copy what wt has done." Verified wt's pattern by reading its main.go. Three-line edit: the var declaration + comment + the `Version:` field. | S:100 R:80 A:95 D:100 |

21 assumptions (18 certain, 3 confident, 0 tentative, 0 unresolved).

<!-- Merged into plan.md ## Requirements on 2026-06-02 — safe to delete. -->
