# Intake: Import idea command from fab-kit

**Change**: 260504-8vcn-import-idea-from-fab-kit
**Created**: 2026-05-04
**Status**: Draft

## Origin

User-initiated `/fab-new` invocation. Raw input:

> Copy the idea command's implementation from ~/code/sahil87/fab-kit/ from there to this new repo. Copy its specs (from docs/specs) also. For the folder structure, use the conventions from ~/code/sahil87/hop/. The release script can also be copied from hop. We will later be removing idea from fab-kit. First lets discuss any hard coupling between fab-kit and idea or if its an easy move.

The session opened with a coupling analysis (this assistant's pre-flight before creating the change). Findings, agreed in conversation:

- The idea binary in fab-kit (`src/go/idea/`) has zero cross-package coupling to fab/wt/kit. Its only external dep is `cobra`; its only internal import is `fab-kit/src/go/idea/internal/idea` (its own internal package). The module path needs to change but the code does not.
- Idea's integration with `/fab-new` is via the on-disk format of `fab/backlog.md` only — the `[xxxx]` ID + optional `[ISSUE_ID]` second bracket convention. That contract lives in fab-kit's skill, not in idea's code, so the move does not break fab-kit's consumers.
- Idea has no dedicated spec in fab-kit; documentation lives as a section in `docs/specs/packages.md` plus an entry in `docs/specs/naming.md` (the local-backlog item format).
- The only practical coupling is build/release: today fab-kit's release archive ships the `idea` binary alongside `fab`, `wt`, `fab-kit`. After this move, idea will release independently using hop's release machinery.

User decisions captured during the discussion:

1. Folder structure follows hop's convention (`src/{cmd/<bin>,internal,go.mod}`) — not fab-kit's per-binary-go.mod layout.
2. Both `scripts/release.sh` and `.github/workflows/release.yml` come from hop verbatim; the Homebrew formula template (`hop/.github/formula-template.rb`) comes too.
3. Specs to copy: the idea section from `packages.md` AND the idea-relevant naming convention from `naming.md`. Idea must become an independent command/repo — its spec set should stand alone, no inbound references to fab-kit.
4. fab-kit will keep idea in place for now; removal from fab-kit is a separate change (out of scope here).

## Why

**Problem**: idea is a general-purpose backlog CLI but currently lives inside fab-kit's monorepo. This couples its release cadence, versioning, Homebrew distribution, and visibility to fab-kit. Users who want a lightweight backlog tool inherit the entire fab-kit footprint.

**Consequence if we don't move it**: idea cannot be released, versioned, or installed independently. The fab-kit ↔ idea contract (the `fab/backlog.md` line format) is invisible — the integration looks like one tool when it is really two. This blocks promoting idea on its own terms (e.g., on `ai.shll.in`) and forces fab-kit users to upgrade fab-kit just to get idea changes.

**Why now**: This repo (`idea`) already exists with `fab/project/{config.yaml,constitution.md,context.md,...}` initialized. The constitution already encodes idea's design principles (plain-text backlog, worktree-aware, cobra-idiomatic, internal/idea separation, table-driven tests, stable IDs). The receiving structure is ready; we just need to land the code.

**Why hop's conventions over fab-kit's**: hop is a single-binary Go repo released independently via Homebrew tap — exactly the shape idea needs to become. Reusing hop's `scripts/`, `justfile`, `.github/workflows/release.yml`, and formula template is faster, more proven, and more consistent than inventing a new pattern. fab-kit's per-binary `src/go/<bin>/go.mod` layout makes sense for a multi-binary monorepo; idea is a single binary and should look like hop.

## What Changes

### 1. Add Go source under `src/`

Port `~/code/sahil87/fab-kit/src/go/idea/` into `src/` using hop's layout. Single repo-root `src/go.mod`; binary entry under `src/cmd/idea/`; package logic under `src/internal/idea/`.

Source mapping (fab-kit path → new path):

| fab-kit path | new path |
|---|---|
| `src/go/idea/cmd/main.go` | `src/cmd/idea/main.go` |
| `src/go/idea/cmd/add.go` | `src/cmd/idea/add.go` |
| `src/go/idea/cmd/list.go` | `src/cmd/idea/list.go` |
| `src/go/idea/cmd/show.go` | `src/cmd/idea/show.go` |
| `src/go/idea/cmd/done.go` | `src/cmd/idea/done.go` |
| `src/go/idea/cmd/reopen.go` | `src/cmd/idea/reopen.go` |
| `src/go/idea/cmd/edit.go` | `src/cmd/idea/edit.go` |
| `src/go/idea/cmd/rm.go` | `src/cmd/idea/rm.go` |
| `src/go/idea/cmd/resolve.go` | `src/cmd/idea/resolve.go` |
| `src/go/idea/cmd/main_test.go` | `src/cmd/idea/main_test.go` |
| `src/go/idea/internal/idea/idea.go` | `src/internal/idea/idea.go` |
| `src/go/idea/internal/idea/idea_test.go` | `src/internal/idea/idea_test.go` |
| `src/go/idea/go.mod` | `src/go.mod` (rewritten — see below) |
| `src/go/idea/go.sum` | `src/go.sum` (carry over, regenerate if needed) |

### 2. Rewrite the Go module path

`src/go.mod` (single-line change vs. fab-kit's):

```go
// before (fab-kit)
module github.com/sahil87/fab-kit/src/go/idea
go 1.22
require github.com/spf13/cobra v1.8.1

// after (idea)
module github.com/sahil87/idea
go 1.22
require github.com/spf13/cobra v1.8.1
```

The single internal-package import inside `cmd/*.go` changes from:

```go
import "github.com/sahil87/fab-kit/src/go/idea/internal/idea"
```

to:

```go
import "github.com/sahil87/idea/internal/idea"
```

### 3. Build & install scripts (copied verbatim from hop, s/hop/idea/g)

Copy `~/code/sahil87/hop/scripts/{build.sh,install.sh,release.sh}` to `scripts/`. Substitute `hop` → `idea` in `build.sh` and `install.sh` only; `release.sh` is generic and lands unchanged.

`scripts/build.sh` after substitution:

```bash
#!/usr/bin/env bash
set -euo pipefail

VERSION="$(git describe --tags --always 2>/dev/null || echo dev)"
mkdir -p bin
cd src
go build -ldflags "-X main.version=${VERSION}" -o ../bin/idea ./cmd/idea
echo "built: bin/idea (version: ${VERSION})"
```

`scripts/install.sh` after substitution:

```bash
#!/usr/bin/env bash
set -euo pipefail

./scripts/build.sh

DEST="${HOME}/.local/bin/idea"
mkdir -p "$(dirname "$DEST")"
cp -f ./bin/idea "$DEST"
echo "installed: $DEST"
```

`scripts/release.sh`: copy verbatim — no substitutions needed.

### 4. justfile

Copy hop's `justfile` and substitute `hop` → `idea`. Recipes: `default` (list), `build`, `local-install`, `test`, `release bump="patch"`.

### 5. CI workflow + Homebrew formula template

Copy `~/code/sahil87/hop/.github/workflows/release.yml` to `.github/workflows/release.yml` with `hop` → `idea` substitution throughout. Tag-driven release: cross-compile darwin/linux × arm64/amd64, GitHub Release with auto-generated notes, then sed the formula template into `homebrew-tap` and push.

Copy `~/code/sahil87/hop/.github/formula-template.rb` to `.github/formula-template.rb` with substitutions:

```ruby
class Idea < Formula
  desc "Capture and manage ideas from the command line"
  homepage "https://github.com/sahil87/idea"
  ...
  url "https://github.com/sahil87/idea/releases/download/v#{version}/idea-darwin-arm64.tar.gz"
  ...
  def install
    bin.install "idea"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/idea --version")
  end
end
```

The Homebrew tap repo (`sahil87/homebrew-tap`) and the `HOMEBREW_TAP_TOKEN` secret already exist (used by hop and fab-kit). No infra setup needed beyond pushing the formula on tag.

### 6. Specs (`docs/specs/`)

Two new spec files derived from fab-kit:

#### `docs/specs/overview.md`

Standalone spec of the `idea` CLI, derived from the "## idea (Backlog Management)" section of `~/code/sahil87/fab-kit/docs/specs/packages.md` (lines 94–143). Strip fab-kit-context sentences ("...alongside fab/wt/kit binaries...", "Distribution: Go binaries are included in fab-kit's per-platform release archive..."); replace with idea-as-its-own-tool framing. Sections:

- Overview & purpose (per-repo plain-text backlog, lightweight CRUD)
- Binary location (`src/cmd/idea/`) and installation (Homebrew tap, direct build)
- Worktree behavior (current worktree default; `--main` opt-in; `git rev-parse` resolution rules — preserve verbatim)
- Commands table (`idea`, `idea add`, `idea list`, `idea show`, `idea done`, `idea reopen`, `idea edit`, `idea rm`)
- ID format (4-char alphanumeric, unique within file) and query semantics (substring, case-insensitive, ID match)
- Integration with external pipelines (described as a generic "consumers can read `fab/backlog.md` by ID"; mention `/fab-new` only as one example consumer, not a coupling)

#### `docs/specs/backlog-format.md`

The backlog file's line format — derived from the relevant rows of `~/code/sahil87/fab-kit/docs/specs/naming.md` (lines 60–73). This is idea's primary public contract. Sections:

- Pattern: `- [ ] [{ID}] [{issue_ids}] {YYYY-MM-DD}: {description}` (issue IDs optional)
- Examples (one with issue ID, one without)
- Component definitions (`[{ID}]` 4-char alphanumeric, `[{issue_ids}]` optional Linear-style, `{YYYY-MM-DD}` ISO date, `{description}` free-form)
- Round-trip preservation guarantee (constitution principle I — non-idea lines preserved verbatim)
- Stability commitment (this format is the public API for external consumers)

#### Update `docs/specs/index.md`

Add table rows for the two new specs. The existing index template currently has just an empty header.

### 7. README update

Replace the "🚧 Standalone repo in progress. Currently bundled with fab-kit" notice in `README.md` with installation and usage. Keep the "Part of @sahil87's open source toolkit" header. Add: install via `brew install sahil87/tap/idea` or `./scripts/install.sh`; quick-start `idea "my idea"`, `idea list`, `idea done <id>`; pointer to `docs/specs/overview.md` and `docs/specs/backlog-format.md`.

### 8. What does NOT change in this scope

- **fab-kit is not modified**. Removing idea from fab-kit (deleting `src/go/idea/`, updating fab-kit's release archive, deprecating the bundled `idea` binary) is a follow-up change in the fab-kit repo. This change focuses entirely on landing idea in its own repo.
- **No behavior changes** to the idea CLI. Same commands, same flags, same backlog format, same exit codes. This is a structural move, not a redesign.
- **No new tests** beyond what comes over from fab-kit. Existing tests in `cmd/main_test.go` and `internal/idea/idea_test.go` should pass after the module path rewrite.

## Affected Memory

This change is implementation-only — it ports existing code into the new repo without changing behavior. The behavior memory will accumulate organically as future changes land. For this change, two domains warrant new memory files (capturing facts that aren't derivable from the code at a glance):

- `release/pipeline`: (new) Tag-driven release flow — `release.sh` cuts the tag, GitHub Actions cross-compiles for darwin/linux × arm64/amd64, publishes the GitHub Release, updates `sahil87/homebrew-tap` formula. Captures the contract with the tap repo + `HOMEBREW_TAP_TOKEN` secret.
- `cli/structure`: (new) Source layout — `src/cmd/idea/` for cobra entry, `src/internal/idea/` for logic, single `src/go.mod` with module path `github.com/sahil87/idea`. Records the convention so future binaries (if any) follow the same shape.

## Impact

**Touched in this repo (creates):**
- `src/go.mod`, `src/go.sum`
- `src/cmd/idea/{main,add,list,show,done,reopen,edit,rm,resolve,main_test}.go`
- `src/internal/idea/{idea,idea_test}.go`
- `scripts/{build,install,release}.sh`
- `justfile`
- `.github/workflows/release.yml`, `.github/formula-template.rb`
- `docs/specs/{overview,backlog-format}.md`
- `docs/specs/index.md` (modify — add rows)
- `README.md` (modify — replace "in progress" notice)

**External systems:**
- `sahil87/homebrew-tap`: a new `Formula/idea.rb` will be created on first tag push (pre-existing tap repo, pre-existing CI token).
- GitHub Actions: requires `HOMEBREW_TAP_TOKEN` repo secret to be configured before the first release. Same secret is already used by hop and fab-kit; the user owns this and will set it on the new repo.

**Dependencies**: cobra v1.8.1 (and transitively pflag, mousetrap). No new dependencies introduced.

**Out of scope (separate follow-up):** removing `src/go/idea/` from fab-kit, removing the idea-bundled-in-fab-kit references from fab-kit's own specs (`packages.md`, `naming.md`, `index.md`, `user-flow.md`), updating fab-kit's release archive, and the user-facing announcement that idea now installs separately.

## Open Questions

- Should the first release be `v0.1.0` or `v1.0.0`? The CLI is mature and used in production by the user, so `v1.0.0` is defensible; but per common Go-CLI practice, `v0.x` until API stability is asserted is also reasonable. Default assumption: `v0.1.0`. Not blocking — the release script computes the next bump, so the user picks at release time.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Module path becomes `github.com/sahil87/idea` (matches repo URL convention; replaces fab-kit-prefixed path) | Discussed — user said "Idea needs to become an independent command / repo." Repo URL is the canonical Go module path. | S:95 R:80 A:95 D:95 |
| 2 | Certain | Source layout follows hop's convention: `src/{go.mod, cmd/idea/, internal/idea/}` (single repo-level go.mod, not per-binary as in fab-kit) | Discussed — user explicitly chose hop's convention. fab-kit uses per-binary go.mod because it's a multi-binary monorepo; idea is single-binary. | S:100 R:75 A:95 D:100 |
| 3 | Certain | `scripts/release.sh` copied verbatim from hop (no substitutions) | Verified by reading hop's release.sh — fully generic, references `git describe`, no hop-specific identifiers. | S:90 R:85 A:100 D:100 |
| 4 | Certain | `scripts/build.sh` and `scripts/install.sh` copied from hop with `hop` → `idea` substitution | Verified — only hop-name occurrences are the binary name (`bin/hop`, `./cmd/hop`, `~/.local/bin/hop`). Mechanical rename. | S:90 R:85 A:100 D:100 |
| 5 | Certain | `.github/workflows/release.yml` and `.github/formula-template.rb` are copied alongside `release.sh` (CI is what makes the script useful) | Discussed — user answered "yes" to including the CI workflow. Verified `formula-template.rb` is referenced by the workflow. | S:100 R:75 A:95 D:100 |
| 6 | Certain | Specs to land: idea section from `packages.md` + idea-relevant rows from `naming.md` (the local-backlog format). No other fab-kit spec files copied. | Discussed — user said "idea section + any naming conventions that impact idea." Verified other fab-kit spec files (`index.md`, `user-flow.md`) reference idea only as a fab-kit consumer, not as an idea-internal concern. | S:100 R:80 A:90 D:95 |
| 7 | Certain | Only internal cross-package import in idea's code is its own `internal/idea`; module path rewrite is the only required source edit | Verified by grepping all imports in `~/code/sahil87/fab-kit/src/go/idea/` — every external import is stdlib or cobra; only one project-internal import line. | S:100 R:90 A:100 D:100 |
| 8 | Certain | fab-kit will not be modified in this change; removing idea from fab-kit is a separate change (separate repo, separate PR) | Discussed — user said "We will later be removing idea from fab-kit." Explicit separation. | S:100 R:80 A:100 D:100 |
| 9 | Certain | Specs land as two files: `docs/specs/overview.md` (idea CLI overview) and `docs/specs/backlog-format.md` (line format spec) | Clarified — user confirmed two-file split. Maps directly to user's guidance ("idea section + naming conventions that impact idea") and preserves clean separation between tool behavior and file-format public contract. | S:95 R:90 A:80 D:95 |
| 10 | Certain | Constitution stays as-is — already encodes idea's design principles (plain-text backlog, worktree-aware, cobra-idiomatic, internal/idea separation, table-driven tests, stable IDs) | Clarified — user confirmed no constitution changes. Verified by reading `fab/project/constitution.md`: already idea-focused, dated 2026-05-03. | S:95 R:85 A:90 D:95 |
| 11 | Confident | Memory files: two new (`release/pipeline`, `cli/structure`) — captures facts not obvious from code | Default — release pipeline involves an external system (homebrew-tap) and a token, which warrants memory. CLI source layout records the chosen convention. Could alternatively defer all memory to first behavior change. | S:60 R:70 A:75 D:60 |
| 12 | Confident | First-version tag is `v0.1.0` (not `v1.0.0`) | Default — Go CLI convention favors `v0.x` until API stability is asserted; release.sh script computes the next bump from current tag, so this is easy to override at release time. | S:55 R:90 A:75 D:60 |
| 13 | Certain | Code lands as a flat copy from `~/code/sahil87/fab-kit/src/go/idea/` — single import commit, no `git filter-repo` history preservation | Clarified — user confirmed flat copy. History remains accessible in fab-kit via `git log -- src/go/idea/`. | S:95 R:55 A:60 D:95 |
| 14 | Certain | `README.md` follows hop's "Standard" depth: Install / Quick Start / Commands sections, cross-links to `docs/specs/overview.md` and `docs/specs/backlog-format.md`, one-line mention of fab-kit `/fab-new` integration via `fab/backlog.md`. No badges, no contributing guide. | Clarified — user chose "Standard" over "Minimum" or "Full". Mirrors hop's README depth. | S:90 R:90 A:75 D:90 |
| 15 | Certain | `HOMEBREW_TAP_TOKEN` repo secret on this repo exists; the `sahil87/homebrew-tap` repo exists at `~/code/sahil87/homebrew-tap/`. Out-of-scope for code changes. | Clarified — user confirmed both the tap repo and the secret already exist (same infra powers hop and fab-kit). Documented in Impact for visibility. | S:95 R:60 A:90 D:95 |

15 assumptions (13 certain, 2 confident, 0 tentative, 0 unresolved).
