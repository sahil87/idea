# Tasks: Import idea command from fab-kit

**Change**: 260504-8vcn-import-idea-from-fab-kit
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

<!-- Create directory structure and copy go.mod/go.sum with the rewritten module path. -->

- [x] T001 Create the source directory tree: `src/cmd/idea/` and `src/internal/idea/` (use `mkdir -p`).
- [x] T002 Copy `~/code/sahil87/fab-kit/src/go/idea/go.mod` to `src/go.mod` and rewrite the module declaration from `module github.com/sahil87/fab-kit/src/go/idea` to `module github.com/sahil87/idea`. Preserve `go 1.22` and the cobra `require` block byte-for-byte.
- [x] T003 [P] Copy `~/code/sahil87/fab-kit/src/go/idea/go.sum` to `src/go.sum` verbatim. After T002 lands, run `cd src && go mod tidy` if `go.sum` becomes inconsistent — only edit it via that command, never by hand.

## Phase 2: Core Implementation

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

## Phase 3: Integration & Edge Cases

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
- [x] T026 [P] Create `docs/specs/overview.md`: derive from `~/code/sahil87/fab-kit/docs/specs/packages.md` lines 94–143 (the `## idea (Backlog Management)` section). Re-frame as standalone idea documentation per spec requirement "CLI overview spec landed at `docs/specs/overview.md`". Required sections in order: overview/purpose, binary location + Homebrew install, worktree behavior (preserve `git rev-parse` resolution rules verbatim), commands table, ID format + query semantics, external-consumer integration (mention `/fab-new` only as one example consumer, not a defining integration). Strip all sentences positioning idea as part of fab-kit's distribution.
- [x] T027 [P] Create `docs/specs/backlog-format.md`: derive from `~/code/sahil87/fab-kit/docs/specs/naming.md` lines 60–73 (the local-backlog Pattern row). Required sections in order: pattern (`- [ ] [{ID}] [{issue_ids}] {YYYY-MM-DD}: {description}`), examples (with-issue-ID + without), component definitions (`[{ID}]`, `[{issue_ids}]`, `{YYYY-MM-DD}`, `{description}`), round-trip preservation guarantee (per Constitution principle I), stability commitment.
- [x] T028 Update `docs/specs/index.md`: add two table rows linking to `overview.md` ("idea CLI overview — purpose, commands, worktree behavior") and `backlog-format.md` ("Backlog file line format — public contract for `fab/backlog.md`"). Preserve the existing header and template comment.

## Phase 4: Polish

<!-- README rewrite. -->

- [x] T029 Rewrite `README.md` at "Standard" depth per spec requirement "README updated to 'Standard' depth". Preserve top-level title and the "Part of @sahil87's open source toolkit" line. Remove the "🚧 Standalone repo in progress. Currently bundled with fab-kit" notice. Add sections in order: one-line description, Install (`brew install sahil87/tap/idea` + `./scripts/install.sh` alternative), Quick Start (3–4 example invocations: `idea "text"`, `idea list`, `idea show <id>`, `idea done <id>`), Commands (brief enumeration or link to `docs/specs/overview.md`), one-line fab-kit `/fab-new` integration mention with link to `docs/specs/backlog-format.md`. Verify: `grep "🚧"` and `grep "Currently bundled with fab-kit"` MUST return zero matches.

## Phase 5: Rework (Cycle 1) — review findings

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

## Phase 6: Rework (Cycle 2) — second-pass review findings

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

<!-- Migrated to plan.md on 2026-06-02 — safe to delete. -->
