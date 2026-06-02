# Plan: Import idea command from fab-kit

**Change**: 260504-8vcn-import-idea-from-fab-kit
**Status**: In Progress
**Intake**: `intake.md`
**Spec**: `spec.md`

## Requirements

<!-- migrated from spec.md on 2026-06-02 -->

### Non-Goals

- **Not modifying fab-kit** — removal of `src/go/idea/` from `~/code/sahil87/fab-kit/` is a separate change in that repo. This change leaves fab-kit untouched.
- **Not changing idea's CLI behavior** — same commands, same flags, same exit codes, same backlog file format. This is a structural move, not a redesign.
- **Not preserving per-file commit history** — code lands as a flat snapshot (single import commit). History remains accessible in fab-kit via `git log -- src/go/idea/`.
- **Not introducing new dependencies** — direct deps stay limited to `github.com/spf13/cobra` (per Constitution principle on dependency discipline).
- **Not adding new tests** — existing tests come over and must pass after the module-path rewrite. Test additions are out of scope.
- **Not cutting a release in this change** — first tag push is a separate user action after merge.

### Source Layout: Go module structure

#### Requirement: Single repo-level `go.mod`

The repository SHALL contain exactly one Go module declared at `src/go.mod`. The module path SHALL be `github.com/sahil87/idea`. The module SHALL declare `go 1.22` and require `github.com/spf13/cobra v1.8.1` directly (with `pflag` and `mousetrap` carried as indirect deps from `go.sum`).

##### Scenario: go.mod after import
- **GIVEN** the import is complete
- **WHEN** a developer runs `cat src/go.mod`
- **THEN** the first line MUST be `module github.com/sahil87/idea`
- **AND** the file MUST NOT contain any reference to `fab-kit` in its module path

##### Scenario: Module independence
- **GIVEN** the import is complete
- **WHEN** a developer runs `cd src && go build ./...` from a fresh clone
- **THEN** the build MUST succeed without requiring access to the fab-kit repo or any private module proxy

#### Requirement: Source files placed under hop's layout

The Go source SHALL be organized as follows under `src/`:
- `cmd/idea/` SHALL contain the cobra entry point and one file per subcommand: `main.go`, `add.go`, `list.go`, `show.go`, `done.go`, `reopen.go`, `edit.go`, `rm.go`, `resolve.go`, plus `main_test.go`.
- `internal/idea/` SHALL contain the package logic: `idea.go` and `idea_test.go`.
- No source files SHALL be placed at any path matching `src/go/idea/...` (the fab-kit nested layout SHALL NOT be carried over).

##### Scenario: Layout matches hop convention
- **GIVEN** the import is complete
- **WHEN** a developer lists `src/`
- **THEN** they SHALL see `go.mod`, `go.sum`, `cmd/`, and `internal/` at the top level
- **AND** `cmd/idea/main.go` SHALL exist
- **AND** `internal/idea/idea.go` SHALL exist
- **AND** no path `src/go/` SHALL exist

#### Requirement: Internal import path rewritten

The single internal-package import in `cmd/*.go` SHALL be `github.com/sahil87/idea/internal/idea`. No source file SHALL contain the string `github.com/sahil87/fab-kit` after the import.

##### Scenario: No fab-kit references in source
- **GIVEN** the import is complete
- **WHEN** a developer runs `grep -r "fab-kit" src/ go.mod`
- **THEN** zero matches SHALL be returned

#### Requirement: Source content preserved verbatim

Every file copied from `~/code/sahil87/fab-kit/src/go/idea/` SHALL retain its content byte-for-byte except for these four edits, all of which are structural consequences of becoming an independently-released binary:

(a) the `module` line in `go.mod` (`github.com/sahil87/fab-kit/src/go/idea` → `github.com/sahil87/idea`);
(b) the single internal-package import in any `cmd/*.go` file (`"github.com/sahil87/fab-kit/src/go/idea/internal/idea"` → `"github.com/sahil87/idea/internal/idea"`);
(c) hard-coded subpaths of the source tree that must adjust because the cobra entry package moved from `cmd/` (under fab-kit's per-binary go.mod layout) to `cmd/idea/` (under hop's single-go.mod layout). Concretely: `cmd/idea/main_test.go` builds the binary under test by joining a discovered module root with `"cmd"`; this string SHALL change to `"cmd/idea"` to point at the relocated cobra entry package. Any other layout-induced path reference SHALL be edited under this same rule;
(d) version-stamping wiring in `cmd/idea/main.go`, mirroring the pattern established by `wt` (`~/code/sahil87/wt/src/cmd/wt/main.go`). Specifically: declare a package-level `var version = "dev"` (with the comment `// version is the binary version, overridden via -ldflags "-X main.version=..." at build time.`) and add `Version: version,` to the root cobra command's struct literal. This wiring is required because (1) idea is now released independently with `git describe`-derived versions injected via `scripts/build.sh`'s `-ldflags '-X main.version=${VERSION}'`, (2) the imported `.github/formula-template.rb`'s test block invokes `#{bin}/idea --version`, and (3) without this wiring the `-ldflags` injection silently no-ops and `--version` is unrecognized. fab-kit's idea did not need this wiring because it shipped inside fab-kit's archive, where the parent kit owned versioning.

No semantic edits, refactors, formatting changes, comment edits, function renames, or behavior changes SHALL be made beyond the four categories above.

##### Scenario: Functional behavior unchanged
- **GIVEN** the import is complete
- **WHEN** a developer runs `cd src && go test ./...`
- **THEN** all tests SHALL pass
- **AND** the test count SHALL match the test count in fab-kit's `src/go/idea/` at the time of the import

##### Scenario: Diff scope is minimal
- **GIVEN** the import is complete
- **WHEN** a developer compares each ported file to its fab-kit source via `diff`
- **THEN** the only differences SHALL fall under the four categories above (module line in `go.mod`; internal-import line in `cmd/*.go`; layout-induced subpath strings such as `"cmd"` → `"cmd/idea"` in `main_test.go`; version-stamping wiring in `cmd/idea/main.go`)

### Build & Release: Scripts and CI

#### Requirement: Build script copied from hop

`scripts/build.sh` SHALL be copied from `~/code/sahil87/hop/scripts/build.sh` with `hop` → `idea` substituted in three locations: the binary output path (`bin/idea`), the cmd path (`./cmd/idea`), and the echoed message. The script's structure (shebang, `set -euo pipefail`, `git describe`-derived `VERSION`, `mkdir -p bin`, `cd src`, `go build` with `-ldflags`) SHALL be preserved verbatim.

##### Scenario: build.sh produces a binary
- **GIVEN** a clean checkout
- **WHEN** a developer runs `./scripts/build.sh`
- **THEN** `bin/idea` SHALL exist and be executable
- **AND** the script's stdout SHALL include `built: bin/idea (version: <version>)`

#### Requirement: Install script copied from hop

`scripts/install.sh` SHALL be copied from `~/code/sahil87/hop/scripts/install.sh` with `hop` → `idea` substituted in three locations: the script chain to `./scripts/build.sh`, the destination path (`${HOME}/.local/bin/idea`), and the echoed message. Structure preserved verbatim.

##### Scenario: install.sh installs to ~/.local/bin
- **GIVEN** a successful build
- **WHEN** a developer runs `./scripts/install.sh`
- **THEN** `~/.local/bin/idea` SHALL exist and be executable
- **AND** the script's stdout SHALL include `installed: <DEST>`

#### Requirement: Release script copied verbatim

`scripts/release.sh` SHALL be copied from `~/code/sahil87/hop/scripts/release.sh` with no edits. The script is generic — it takes `<patch|minor|major>`, computes the next semver from `git describe --tags --abbrev=0`, validates a clean working tree and named branch, creates the tag, and pushes it. All logic, error messages, and usage text remain identical.

##### Scenario: release.sh validates inputs
- **GIVEN** the script is invoked with no arguments
- **WHEN** the script runs
- **THEN** it SHALL print the usage text and exit 0

##### Scenario: release.sh refuses dirty tree
- **GIVEN** the working tree has uncommitted changes
- **WHEN** the developer runs `./scripts/release.sh patch`
- **THEN** the script SHALL exit non-zero
- **AND** stderr SHALL include `Working tree not clean`

#### Requirement: justfile copied from hop

The repository SHALL contain a top-level `justfile` copied from `~/code/sahil87/hop/justfile` with `hop` → `idea` substituted. It SHALL define five recipes: `default` (lists recipes via `just --list`), `build` (delegates to `./scripts/build.sh`), `local-install` (delegates to `./scripts/install.sh`), `test` (runs `cd src && go test ./...`), and `release bump="patch"` (delegates to `./scripts/release.sh {{bump}}`).

##### Scenario: just lists recipes
- **GIVEN** `just` is installed
- **WHEN** a developer runs `just` with no arguments
- **THEN** the output SHALL list all five recipes

#### Requirement: GitHub Actions release workflow copied from hop

`.github/workflows/release.yml` SHALL be copied from `~/code/sahil87/hop/.github/workflows/release.yml` with `hop` → `idea` substituted in all binary-name and tarball-name positions (e.g., `hop-${os}-${arch}` → `idea-${os}-${arch}`, `./cmd/hop` → `./cmd/idea`, `Formula/hop.rb` → `Formula/idea.rb`). The workflow SHALL preserve verbatim: trigger (`tags: ["v*"]`), permissions, the four cross-compile targets (darwin/arm64, darwin/amd64, linux/arm64, linux/amd64), the release-base detection logic for minor releases, the use of `softprops/action-gh-release`, and the homebrew-tap update flow.

##### Scenario: Workflow yaml is valid
- **GIVEN** `.github/workflows/release.yml` exists
- **WHEN** a developer parses the file as YAML
- **THEN** parsing SHALL succeed
- **AND** the `on.push.tags` array SHALL contain `"v*"`

##### Scenario: All four platform builds present
- **GIVEN** `.github/workflows/release.yml` exists
- **WHEN** a developer inspects the cross-compile step
- **THEN** the `targets` variable SHALL list `darwin/arm64 darwin/amd64 linux/arm64 linux/amd64` (in any order)

#### Requirement: Homebrew formula template copied from hop

`.github/formula-template.rb` SHALL be copied from `~/code/sahil87/hop/.github/formula-template.rb` with substitutions: class name `Hop` → `Idea`, `desc` updated to "Capture and manage ideas from the command line", `homepage` updated to `https://github.com/sahil87/idea`, all four download URLs updated to point at `sahil87/idea` releases with `idea-${os}-${arch}.tar.gz` filenames, the `bin.install` line updated to `bin.install "idea"`, and the `test` block updated to invoke `#{bin}/idea --version`. The four `VERSION_PLACEHOLDER`, `SHA_DARWIN_ARM64`, `SHA_DARWIN_AMD64`, `SHA_LINUX_ARM64`, `SHA_LINUX_AMD64` placeholders SHALL be preserved verbatim — they are sed targets in the workflow.

##### Scenario: Formula template is valid Ruby
- **GIVEN** `.github/formula-template.rb` exists
- **WHEN** a developer runs `ruby -c .github/formula-template.rb` (if Ruby is available)
- **THEN** the syntax check SHALL pass

##### Scenario: Placeholders intact
- **GIVEN** `.github/formula-template.rb` exists
- **WHEN** a developer greps for placeholders
- **THEN** all five placeholders (`VERSION_PLACEHOLDER`, `SHA_DARWIN_ARM64`, `SHA_DARWIN_AMD64`, `SHA_LINUX_ARM64`, `SHA_LINUX_AMD64`) SHALL be present exactly once each

### Specs: Documentation deliverables

#### Requirement: CLI overview spec landed at `docs/specs/overview.md`

`docs/specs/overview.md` SHALL be created as a standalone specification of the `idea` CLI, derived from the "## idea (Backlog Management)" section of `~/code/sahil87/fab-kit/docs/specs/packages.md` (lines 94–143). The spec SHALL be re-framed as idea-as-its-own-tool — fab-kit-context references SHALL be removed or rewritten:
- The "Binary: src/go/idea/ ..." line SHALL be rewritten to reference `src/cmd/idea/` and Homebrew installation via `sahil87/tap/idea`.
- The Distribution paragraph (about fab-kit's per-platform release archive) SHALL be removed.
- The "Integration with Fab" subsection SHALL be retained but rewritten to describe a generic external-consumer contract: any tool can read `fab/backlog.md` to discover backlog IDs; `/fab-new` from fab-kit is mentioned as one example consumer (not the defining integration).

The spec SHALL contain (in order): an overview/purpose paragraph, binary location and installation, worktree behavior (current worktree default, `--main` opt-in, `git rev-parse` resolution rules — preserved verbatim), the commands table, ID format and query semantics, and the external-consumer integration section.

##### Scenario: Spec is self-contained
- **GIVEN** `docs/specs/overview.md` exists
- **WHEN** a developer reads the file in isolation
- **THEN** they SHALL be able to understand what idea does, how to install it, and how to use it
- **AND** they SHALL NOT need to read any fab-kit doc to follow it

##### Scenario: No fab-kit dependency framing
- **GIVEN** `docs/specs/overview.md` exists
- **WHEN** a developer greps for "fab-kit"
- **THEN** matches SHALL appear only in the external-consumer subsection (e.g., "fab-kit's `/fab-new` is one example consumer")
- **AND** no match SHALL describe idea as part of, included in, or distributed by fab-kit

#### Requirement: Backlog format spec landed at `docs/specs/backlog-format.md`

`docs/specs/backlog-format.md` SHALL be created from the relevant rows of `~/code/sahil87/fab-kit/docs/specs/naming.md` (lines 60–73). The spec SHALL document the `fab/backlog.md` line format as idea's primary public contract. Sections in order: pattern (`- [ ] [{ID}] [{issue_ids}] {YYYY-MM-DD}: {description}` — issue IDs optional), examples (one with issue ID, one without), component definitions, round-trip preservation guarantee (per Constitution principle I — non-idea lines preserved verbatim), and a stability commitment statement that the format is the public API for external consumers.

##### Scenario: Pattern is documented exactly
- **GIVEN** `docs/specs/backlog-format.md` exists
- **WHEN** a developer reads the pattern definition
- **THEN** the literal pattern string `- [ ] [{ID}] [{issue_ids}] {YYYY-MM-DD}: {description}` SHALL appear verbatim
- **AND** both example forms (with and without issue IDs) SHALL appear

#### Requirement: Specs index updated

`docs/specs/index.md` SHALL be updated to include rows for both new specs in the `| Spec | Description |` table. Each row SHALL link to the spec file and provide a one-line description.

##### Scenario: Index lists both specs
- **GIVEN** `docs/specs/index.md` is updated
- **WHEN** a developer reads the table
- **THEN** they SHALL find a row linking to `overview.md` describing the idea CLI
- **AND** a row linking to `backlog-format.md` describing the backlog file format

### README

#### Requirement: README updated to "Standard" depth

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

##### Scenario: README is self-contained
- **GIVEN** `README.md` is updated
- **WHEN** a developer lands on the repo for the first time
- **THEN** they SHALL be able to install the tool and run their first command without leaving the README

##### Scenario: No "in progress" notice
- **GIVEN** `README.md` is updated
- **WHEN** a developer reads the file
- **THEN** the string "🚧" SHALL NOT appear
- **AND** the string "Currently bundled with fab-kit" SHALL NOT appear

### Memory: Hydrate deliverables

#### Requirement: Release pipeline memory file created

At hydrate, `docs/memory/release/pipeline.md` SHALL be created. It SHALL document: the tag-driven release flow (`scripts/release.sh` cuts the tag, GitHub Actions takes over from the tag push), the cross-compile matrix (darwin/linux × arm64/amd64), the GitHub Release creation step, and the contract with `sahil87/homebrew-tap` (which the workflow updates by sed-substituting `formula-template.rb`). It SHALL note that the `HOMEBREW_TAP_TOKEN` secret must be configured on this repo and that the same token already powers hop and fab-kit releases.

##### Scenario: Memory file describes the full pipeline
- **GIVEN** the change has reached hydrate
- **WHEN** a developer reads `docs/memory/release/pipeline.md`
- **THEN** the file SHALL describe each stage from `release.sh` invocation through homebrew-tap update
- **AND** the file SHALL reference the `HOMEBREW_TAP_TOKEN` secret requirement

#### Requirement: CLI structure memory file created

At hydrate, `docs/memory/cli/structure.md` SHALL be created. It SHALL document the source layout decision: `src/cmd/idea/` for the cobra entry point, `src/internal/idea/` for the logic package, single `src/go.mod` with module path `github.com/sahil87/idea`. It SHALL note this follows hop's convention (single repo-level go.mod) rather than fab-kit's per-binary go.mod layout, and explain when each pattern is appropriate (single-binary repo → hop layout; multi-binary monorepo → fab-kit layout).

##### Scenario: Memory file documents the layout
- **GIVEN** the change has reached hydrate
- **WHEN** a developer reads `docs/memory/cli/structure.md`
- **THEN** the file SHALL describe the `src/{go.mod, cmd/idea/, internal/idea/}` layout
- **AND** the file SHALL state the module path
- **AND** the file SHALL contrast with the alternative (fab-kit's per-binary layout)

#### Requirement: Memory index updated

`docs/memory/index.md` SHALL be updated at hydrate to include rows for the two new memory domains (`release` and `cli`) and pointers to the new files.

##### Scenario: Memory index lists new domains
- **GIVEN** the change has reached hydrate
- **WHEN** a developer reads `docs/memory/index.md`
- **THEN** the table SHALL contain a row for the `release` domain pointing at `pipeline.md`
- **AND** a row for the `cli` domain pointing at `structure.md`

### Validation

#### Requirement: Tests pass after the move

After the import, all tests in the new location SHALL pass via `cd src && go test ./...`. The set of test cases SHALL match the set in fab-kit's `src/go/idea/` at the time of the import (no tests added, no tests removed).

##### Scenario: Test count parity
- **GIVEN** the import is complete
- **WHEN** a developer counts test functions in `src/cmd/idea/` and `src/internal/idea/`
- **THEN** the count SHALL equal the count in `~/code/sahil87/fab-kit/src/go/idea/cmd/` and `~/code/sahil87/fab-kit/src/go/idea/internal/idea/` at the time of the import

##### Scenario: Tests pass in the new repo
- **GIVEN** the import is complete
- **WHEN** a developer runs `cd src && go test ./...`
- **THEN** the exit code SHALL be 0
- **AND** stdout SHALL show `ok` for both `cmd/idea` and `internal/idea` packages

#### Requirement: Binary builds and runs

After the import, the `idea` binary SHALL build via `./scripts/build.sh` and respond to `--help` and `--version` invocations.

##### Scenario: Binary responds to --help
- **GIVEN** a successful build
- **WHEN** a developer runs `./bin/idea --help`
- **THEN** the exit code SHALL be 0
- **AND** stdout SHALL list the seven subcommands: `add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`

##### Scenario: Binary responds to --version
- **GIVEN** a successful build via `./scripts/build.sh`
- **WHEN** a developer runs `./bin/idea --version`
- **THEN** the exit code SHALL be 0
- **AND** stdout SHALL include the version stamp from `git describe`

### Design Decisions

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

## Tasks

### Phase 1: Setup

<!-- Create directory structure and copy go.mod/go.sum with the rewritten module path. -->

- [x] T001 Create the source directory tree: `src/cmd/idea/` and `src/internal/idea/` (use `mkdir -p`).
- [x] T002 Copy `~/code/sahil87/fab-kit/src/go/idea/go.mod` to `src/go.mod` and rewrite the module declaration from `module github.com/sahil87/fab-kit/src/go/idea` to `module github.com/sahil87/idea`. Preserve `go 1.22` and the cobra `require` block byte-for-byte.
- [x] T003 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/go.sum` to `src/go.sum` verbatim. After T002 lands, run `cd src && go mod tidy` if `go.sum` becomes inconsistent — only edit it via that command, never by hand.

### Phase 2: Core Implementation

<!-- Port Go source files. Each cmd/*.go file is independent of the others (they all import `internal/idea`); internal/idea/*.go has no internal deps. So all source-file copies can parallelize. -->

- [x] T004 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/internal/idea/idea.go` to `src/internal/idea/idea.go` byte-for-byte (no edits — this file has no fab-kit-prefixed imports).
- [x] T005 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/internal/idea/idea_test.go` to `src/internal/idea/idea_test.go` byte-for-byte.
- [x] T006 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/cmd/main.go` to `src/cmd/idea/main.go`; rewrite the import line `"github.com/sahil87/fab-kit/src/go/idea/internal/idea"` to `"github.com/sahil87/idea/internal/idea"` if present.
- [x] T007 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/cmd/add.go` to `src/cmd/idea/add.go`; apply the same import rewrite if present.
- [x] T008 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/cmd/list.go` to `src/cmd/idea/list.go`; apply the same import rewrite if present.
- [x] T009 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/cmd/show.go` to `src/cmd/idea/show.go`; apply the same import rewrite if present.
- [x] T010 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/cmd/done.go` to `src/cmd/idea/done.go`; apply the same import rewrite if present.
- [x] T011 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/cmd/reopen.go` to `src/cmd/idea/reopen.go`; apply the same import rewrite if present.
- [x] T012 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/cmd/edit.go` to `src/cmd/idea/edit.go`; apply the same import rewrite if present.
- [x] T013 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/cmd/rm.go` to `src/cmd/idea/rm.go`; apply the same import rewrite if present.
- [x] T014 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/cmd/resolve.go` to `src/cmd/idea/resolve.go`; apply the same import rewrite if present.
- [x] T015 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/cmd/main_test.go` to `src/cmd/idea/main_test.go`; apply the same import rewrite if present.
- [x] T015b Layout-induced path edit in `src/cmd/idea/main_test.go`: change the `filepath.Join(findModuleRoot(t), "cmd")` occurrence to `filepath.Join(findModuleRoot(t), "cmd", "idea")` (or equivalent — the literal `"cmd"` string at line ~16 becomes `"cmd/idea"` joined appropriately). This is the only edit in `main_test.go` beyond the import rewrite, and is permitted by the spec's "Source content preserved verbatim" requirement category (c). Verify by `diff` that the file differs from fab-kit's version only at the import line and this path string.
- [x] T015c Version-stamping wiring in `src/cmd/idea/main.go`, mirroring `~/code/sahil87/wt/src/cmd/wt/main.go`. Two edits: (1) at package scope (after the existing `var fileFlag string` / `var mainFlag bool` block, around line 13), insert: `// version is the binary version, overridden via -ldflags "-X main.version=..." at build time.` followed by `var version = "dev"`. (2) Inside the `root := &cobra.Command{...}` struct literal, add the field `Version: version,` (placement: alongside `Use:`, `Short:`, etc. — wt places it after the Long block). Run `cd src && go build ./...` and `cd src && go test ./...` after the edit to confirm the source still compiles and tests still pass. Permitted by the spec's "Source content preserved verbatim" requirement category (d).

### Phase 3: Integration & Edge Cases

<!-- Build/release machinery, validation, and the no-fab-kit-references guarantee. -->

- [x] T016 Verify no source file under `src/` contains `github.com/sahil87/fab-kit`: run `grep -r "fab-kit" src/`. The expected match count is 0. If matches appear, fix the offending imports before proceeding.
- [x] T017 Build verification: run `cd src && go build ./...` from a clean state. Build MUST succeed. Then run `cd src && go test ./...`. All tests MUST pass with exit code 0 and `ok` for both `cmd/idea` and `internal/idea` packages.
- [x] T018 [P] Copy `~/code/sahil87/hop/scripts/release.sh` to `scripts/release.sh` verbatim (no substitutions). Set executable bit (`chmod +x scripts/release.sh`).
- [x] T019 [P] Copy `~/code/sahil87/hop/scripts/build.sh` to `scripts/build.sh`, substituting `hop` → `idea` at three locations: `bin/idea`, `./cmd/idea`, and the echoed `built: bin/idea (version: ...)` message. Set executable bit.
- [x] T020 [P] Copy `~/code/sahil87/hop/scripts/install.sh` to `scripts/install.sh`, substituting `hop` → `idea` at three locations: the `DEST="${HOME}/.local/bin/idea"`, the binary copy `cp -f ./bin/idea "$DEST"`, and the echoed `installed: $DEST` message. Set executable bit.
- [x] T021 [P] Copy `~/code/sahil87/hop/justfile` to `./justfile`, substituting `hop` → `idea` at every occurrence (binary name in build/install/test/release recipes and the comment header). The five recipes (`default`, `build`, `local-install`, `test`, `release bump="patch"`) MUST be preserved.
- [x] T022 [P] Create `.github/workflows/` directory and copy `~/code/sahil87/hop/.github/workflows/release.yml` to `.github/workflows/release.yml`, substituting `hop` → `idea` everywhere it appears as a binary or tarball name (including `hop-${os}-${arch}` → `idea-${os}-${arch}`, `./cmd/hop` → `./cmd/idea`, `Formula/hop.rb` → `Formula/idea.rb`, the `Building ${output}...` echoes, and the commit message `hop v${version}` → `idea v${version}`). Preserve verbatim: trigger, permissions, all four cross-compile targets, the release-base detection logic, and the `softprops/action-gh-release` action pin.
- [x] T023 [P] Copy `~/code/sahil87/hop/.github/formula-template.rb` to `.github/formula-template.rb`, substituting: class name `Hop` → `Idea`; `desc` → `"Capture and manage ideas from the command line"`; `homepage` → `"https://github.com/sahil87/idea"`; all four download URLs to `https://github.com/sahil87/idea/releases/download/v#{version}/idea-{os}-{arch}.tar.gz`; `bin.install "hop"` → `bin.install "idea"`; the test block to invoke `#{bin}/idea --version`. Preserve verbatim: all five sed placeholders (`VERSION_PLACEHOLDER`, `SHA_DARWIN_ARM64`, `SHA_DARWIN_AMD64`, `SHA_LINUX_ARM64`, `SHA_LINUX_AMD64`).
- [x] T024 Verify build script: run `./scripts/build.sh`. The script MUST exit 0, produce `bin/idea`, and stdout MUST include `built: bin/idea (version: ...)`.
- [x] T025 Verify the binary runs: run `./bin/idea --help`. Exit code MUST be 0 and stdout MUST list at least the seven user-facing subcommands (`add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`). Then run `./bin/idea --version`. Exit code MUST be 0 and stdout MUST include the `git describe`-derived version stamp.
- [x] T026 [P] Create `docs/specs/overview.md`: derive from `~/code/sahil87/fab-kit/docs/specs/packages.md` lines 94–143 (the `## idea (Backlog Management)` section). Re-frame as standalone idea documentation per spec requirement "CLI overview spec landed at `docs/specs/overview.md`". Required sections in order: overview/purpose, binary location + Homebrew install, worktree behavior (preserve `git rev-parse` resolution rules verbatim), commands table, ID format + query semantics, external-consumer integration (mention `/fab-new` only as one example consumer, not a defining integration).
- [x] T027 [P] Create `docs/specs/backlog-format.md`: derive from `~/code/sahil87/fab-kit/docs/specs/naming.md` lines 60–73 (the local-backlog Pattern row). Required sections in order: pattern (`- [ ] [{ID}] [{issue_ids}] {YYYY-MM-DD}: {description}`), examples (with-issue-ID + without), component definitions (`[{ID}]`, `[{issue_ids}]`, `{YYYY-MM-DD}`, `{description}`), round-trip preservation guarantee (per Constitution principle I), stability commitment.
- [x] T028 Update `docs/specs/index.md`: add two table rows linking to `overview.md` ("idea CLI overview — purpose, commands, worktree behavior") and `backlog-format.md` ("Backlog file line format — public contract for `fab/backlog.md`"). Preserve the existing header and template comment.

### Phase 4: Polish

<!-- README rewrite. -->

- [x] T029 Rewrite `README.md` at "Standard" depth per spec requirement "README updated to 'Standard' depth". Preserve top-level title and the "Part of @sahil87's open source toolkit" line. Remove the "🚧 Standalone repo in progress. Currently bundled with fab-kit" notice. Add sections in order: one-line description, Install (`brew install sahil87/tap/idea` + `./scripts/install.sh` alternative), Quick Start (3–4 example invocations: `idea "text"`, `idea list`, `idea show <id>`, `idea done <id>`), Commands (brief enumeration or link to `docs/specs/overview.md`), one-line fab-kit `/fab-new` integration mention with link to `docs/specs/backlog-format.md`. Verify: `grep "🚧"` and `grep "Currently bundled with fab-kit"` MUST return zero matches.

### Phase 5: Rework (Cycle 1) — review findings

<!-- Added after outward review identified must-fix and should-fix items. -->

- [x] T030 [P] Replace `.gitignore` with the Go-flavored pattern set from `~/code/sahil87/hop/.gitignore`. Preserve any existing entries that are not in conflict (review the current file first); but the core requirement is that `bin/`, `dist/`, `*.test`, `*.out`, `coverage.*`, `*.coverprofile`, `profile.cov`, `go.work`, `go.work.sum`, `*.exe`, `*.dylib`, `*.so`, `*.dll` are present. After writing, verify `git check-ignore bin/idea` exits 0.
- [x] T031 [P] Remove the stray `scratch_dirty_marker.txt` from repo root (`rm scratch_dirty_marker.txt`). Verify with `ls scratch_dirty_marker.txt 2>&1` that the file is gone.
- [x] T032 Clarify the `[issue_ids]` slot in `docs/specs/backlog-format.md`. Resolution: the format permits an optional `[issue_ids]` bracket, but idea does NOT parse, index, or expose it via any query — it is **opaque pass-through content** preserved verbatim per Constitution principle I. External consumers (e.g., fab-kit's `/fab-new`) parse the issue ID independently. Edits required:
  - Move the `[issue_ids]` slot description out of the "components idea parses" framing into a new subsection titled "Opaque pass-through fields" (or equivalent) that explains: idea preserves these brackets verbatim during round-trip but does not match them in `idea show`/`done`/`edit`/`rm` queries (queries match against `[ID]` and description text only).
  - Update the stability commitment to clarify what idea guarantees: round-trip preservation of any bracket-slot content between `[ID]` and the date, but no parsing semantics for those slots.
  - Keep both example forms (with and without `[issue_ids]`); annotate the with-issue-ID example noting that idea passes the bracket through unchanged.
  - Cross-reference the Constitution principle I about non-idea-line preservation extending to within-line opaque fields.
- [x] T033 [P] Add a "Worktree behavior" mention to `README.md` after the Quick Start section (or as a sub-bullet under Commands). One-line summary: idea operates on the current worktree's `fab/backlog.md` by default; pass `--main` to target the main worktree's backlog; override the file path with `--file` or set `IDEAS_FILE` in the environment. Cross-link to `docs/specs/overview.md` for full details.
- [x] T034 [P] Add a `LICENSE` file at repo root containing the MIT license text. Source: copy `~/code/sahil87/hop/LICENSE` verbatim except for the copyright line, which SHALL read `Copyright (c) 2026 Sahil Ahuja` (matching hop's pattern, year aligned with the create-date in `.status.yaml`).
- [x] T035 Final verification after rework: run `cd src && go test ./...` (must pass), `./scripts/build.sh` (must exit 0), `./bin/idea --help` and `./bin/idea --version` (both must exit 0). Run `git status --short` and confirm `scratch_dirty_marker.txt` is absent and `bin/idea` is no longer listed (it's now ignored).

### Phase 6: Rework (Cycle 2) — second-pass review findings

<!-- The cycle 1 fix to backlog-format.md improved framing but still misrepresented behavior. Live test confirmed: lines containing the optional [issue_ids] bracket fail the parser regex entirely and become INVISIBLE to idea (not just the slot — the whole line). The line is treated like a comment/header, preserved as inert content. Plus a footer count mismatch in spec.md. -->

- [x] T036 Correct `docs/specs/backlog-format.md` to reflect actual parser behavior (verified live: a line `- [ ] [a7k2] [DEV-1011] 2026-02-12: text` is rejected by the `lineRegex` at `src/internal/idea/idea.go:57` — `^- \[([ x])\] \[([a-z0-9]{4})\] (\d{4}-\d{2}-\d{2}): (.+)$` — so the entire line is invisible to `idea list`, `show`, `done`, `edit`, `rm`). Required edits:
  - The "Opaque Pass-Through Fields" subsection must be rewritten so it does NOT imply that the rest of the line (ID, date, description) is parsed when the optional `[issue_ids]` slot is present. The accurate framing is: lines whose shape exactly matches `- [ ] [{ID}] {YYYY-MM-DD}: {description}` (no second bracket) are idea-managed; lines whose shape includes the optional `[issue_ids]` bracket are NOT matched by the parser and are treated as **inert pass-through content**, equivalent to a header or prose line — round-trip preserved verbatim per Constitution principle I, but invisible to all idea operations (`list`, `show`, `done`, `edit`, `rm`).
  - Update the with-issue-ID example annotation to make this explicit: "this form is preserved verbatim by `idea` on round-trip but is not visible to any `idea` query — it is treated as inert content. External consumers like fab-kit's `/fab-new` parse the issue-ID bracket independently."
  - In the stability commitment, distinguish: (a) the format spec describes both shapes, (b) idea's stability commitment covers the no-second-bracket shape (parsing, query, round-trip), (c) for the with-second-bracket shape, idea's stability commitment is round-trip preservation only (the line stays as-is), (d) any query semantics for the issue-ID bracket are owned by external consumers, not idea.
  - Cross-reference: explicitly state that lines with the optional `[issue_ids]` slot fall under Constitution principle I's "non-idea lines preserved verbatim" rule — they are non-idea-managed lines from idea's perspective even though they look superficially similar to idea-managed lines.
- [x] T037 Fix the assumptions footer in `fab/changes/260504-8vcn-import-idea-from-fab-kit/spec.md`. The current line says "21 assumptions (17 certain, 4 confident, 0 tentative, 0 unresolved)." The actual table has 18 Certain + 3 Confident (verified by `awk -F'|' '/^\| [0-9]+ \|/ {print $3}' spec.md | sort | uniq -c`). Update the footer to "21 assumptions (18 certain, 3 confident, 0 tentative, 0 unresolved)."
- [x] T038 Final verification after cycle 2 rework. Re-run the live invisible-line test on a temporary repo:
  ```bash
  TMPDIR=$(mktemp -d)
  cd "$TMPDIR" && git init -q && mkdir -p fab && cat > fab/backlog.md <<EOF
  # Backlog

  - [ ] [a7k2] [DEV-1011] 2026-02-12: With issue id
  - [ ] [b8l3] 2026-02-23: Without issue id
  EOF
  /home/sahil/code/sahil87/idea.worktrees/handy-rhino/bin/idea --file fab/backlog.md list
  /home/sahil/code/sahil87/idea.worktrees/handy-rhino/bin/idea --file fab/backlog.md show a7k2; echo "show-issue-form exit: $?"
  /home/sahil/code/sahil87/idea.worktrees/handy-rhino/bin/idea --file fab/backlog.md show b8l3; echo "show-no-issue-form exit: $?"
  cd / && rm -rf "$TMPDIR"
  ```
  Expected: list shows ONLY `[b8l3]`; `show a7k2` exits non-zero (no match); `show b8l3` exits 0. Then re-run `cd src && go test ./...` and `./scripts/build.sh`. Confirm everything still passes.

---

## Execution Order

- **Phase 1** before all later phases. T002 must complete before T003 if `go.sum` ends up inconsistent (re-run via `go mod tidy`); otherwise T003 is independent and can parallelize with T002.
- **Phase 2 source ports (T004–T015)** can all run in parallel — each is a different file. They depend on Phase 1 (directories must exist).
- **T016 (no-fab-kit-references grep)** depends on all of T004–T015 completing.
- **T017 (build + tests)** depends on T016 (so the build doesn't fail on a missed import).
- **Phase 3 build/release-machinery copies (T018–T023)** are independent of Phase 2 source ports — they can run in parallel with Phase 2 if convenient. T024 depends on T017 (Go must compile) and T019 (build.sh must exist).
- **T025** depends on T024.
- **T026, T027** are independent file creates — can run in parallel and are independent of Go work.
- **T028** depends on T026 and T027 (links target those files).
- **T029** is independent of Go work but depends on T026 and T027 existing if it cross-links them.

## Acceptance

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
