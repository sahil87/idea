# Plan: Add `idea fmt` — Explicit Canonicalizer with Automatic Checkbox Adoption

**Change**: 260612-4m3a-add-fmt-canonicalizer-adoption
**Intake**: `intake.md`

## Requirements

### CLI: `idea fmt` command surface

#### R1: Subcommand registration and flag surface
A new `fmtCmd() *cobra.Command` factory in `src/cmd/idea/fmt.go` MUST be registered in `newRootCmd()` (`src/cmd/idea/main.go`). The command SHALL take no positional args (`cobra.NoArgs`), define a local `--check` flag, inherit the persistent `--file`/`--main` flags and `IDEAS_FILE` resolution via `resolveFile()`, and carry an enriched `Long` help (terse `Short`, depth in `Long`, inline example) per the repo convention. The cobra file SHALL contain only flag wiring and output formatting (Constitution III/IV).

- **GIVEN** a git repo with a backlog
- **WHEN** `idea fmt` runs
- **THEN** cobra routes to the fmt subcommand (never the root bare-text add shorthand)
- **AND** `idea fmt some text` errors without adding an idea or touching the backlog (namespace claim accepted in the intake)

- **GIVEN** a linked worktree
- **WHEN** `idea fmt --main` runs
- **THEN** the main worktree's backlog is formatted, mirroring every other command's resolution

#### R2: Canonicalization as a thin verb over the existing writer
`internal/idea` SHALL gain an exported `Fmt(path string, check bool) (FmtResult, error)` entry point that composes the **existing** parse/format/save machinery (the same parse walk as `LoadFile`, the same render/serialization used by `SaveFile`/`FormatLine`). There MUST NOT be a second serialization path. A run rewrites recognized idea lines to canonical form: variant bullets (`*`, `+`) → `-`, leading indentation stripped, CRLF → LF, dateless → today's date (counted as backfilled, surfaced via the existing advisory mechanism), legacy lone backslashes in text doubled on disk (decoded content unchanged).

- **GIVEN** a legacy backlog containing `  * [x] [e5f6] fix bug` and `- [ ] [rk7t] a\b` (dateless, CRLF endings)
- **WHEN** `idea fmt` runs on 2026-06-12
- **THEN** the file holds `- [x] [e5f6] 2026-06-12: fix bug` and `- [ ] [rk7t] 2026-06-12: a\\b` with LF endings
- **AND** the result reports the normalized-line and backfilled-date counts

- **GIVEN** a missing backlog file
- **WHEN** `idea fmt` runs
- **THEN** the read error propagates (non-zero exit), consistent with `done`/`edit`

#### R3: Automatic adoption of bare checkboxes
A line is an adoption candidate iff it does NOT parse as an idea (`ParseLine` returns false) AND matches `^\s*[-*+] \[([ xX])\] (.+)$` with non-blank text. Each adopted line MUST receive: a fresh 4-char `[a-z0-9]{4}` ID unique against both the IDs already in the file and IDs assigned earlier in the same fmt pass; date = today (counted as *adopted*, never double-counted as *backfilled*); preserved checked state (`[x]`/`[X]` adopt as done, canonical `[x]`; `[ ]` as open); and its whitespace-trimmed matched text treated as real text (escaped on write via `FormatLine` — a literal `\` doubles on disk).

- **GIVEN** a backlog containing `* [ ] buy milk` and `- [X] ship the release`
- **WHEN** `idea fmt` runs on 2026-06-12
- **THEN** the lines become `- [ ] [xxxx] 2026-06-12: buy milk` and `- [x] [yyyy] 2026-06-12: ship the release` with distinct fresh IDs that collide with no existing ID
- **AND** the result lists both adopted ideas in file order

#### R4: Precision guards and verbatim preservation
A candidate whose text group — evaluated after trimming surrounding whitespace, so extra spaces between the checkbox and the bracket cannot defeat the guard — begins with a `[...]` bracket MUST NOT be adopted — Shape B lines (`- [ ] [ni3o] [DEV-1011] 2026-02-12: text`), bracket-metadata lines (`- [ ] [DEV-1011] external item`, `- [ ] [TODO] buy milk`, `- [ ] [ab1] text`), and their extra-space variants (`- [ ]  [DEV-1011] x`, `- [ ]  [ab12] 2026-01-01: x`) stay byte-for-byte pass-through. Text-less and whitespace-only checkboxes (`- [ ]`, `- [ ]   `) MUST NOT be adopted. Headers, prose, and blank lines pass through verbatim (Constitution I).

- **GIVEN** a backlog with `- [ ] [DEV-1011] external item`, `# Backlog`, a blank line, and prose
- **WHEN** `idea fmt` runs
- **THEN** every one of those lines is byte-identical afterward

#### R5: Idempotency and skip-write
A second `idea fmt` run MUST be byte-stable. When the rebuilt content is byte-identical to the on-disk file, fmt MUST skip the write entirely (no mtime churn, no atomic rename of identical bytes). A 0-byte file is left untouched (no trailing LF invented).

- **GIVEN** a backlog just written by `idea fmt`
- **WHEN** `idea fmt` runs again
- **THEN** the file bytes and mtime are unchanged and the result reports no changes

#### R6: `--check` mode — unified preview and CI gate
`idea fmt --check` MUST write nothing, print the same stderr report as a real run (what *would* be normalized / adopted / backfilled), and exit non-zero (1) when the file is non-canonical — any normalization, adoption, or backfill would occur — and zero when already canonical. The non-canonical exit uses the `errSilent` sentinel so no extra `ERROR:` line is printed.

- **GIVEN** a non-canonical backlog
- **WHEN** `idea fmt --check` runs
- **THEN** the exit code is 1, the would-be report is on stderr, and the file bytes are unchanged

- **GIVEN** a fully canonical backlog
- **WHEN** `idea fmt --check` runs
- **THEN** the exit code is 0 and nothing is printed

#### R7: Output contract — silent stdout, advisory stderr
stdout MUST stay empty on every fmt path (success is silence + exit 0, the `gofmt -w` precedent; Constitution VI). All human-facing reporting goes to stderr: one `adopted: [id] {escaped text}` line per adopted idea (file order), the existing backfill advisory (`note: stamped today's date on N previously-dateless item(s)` via `printBackfillNotice`, suppressed at 0), and a summary line with the normalized/adopted counts printed only when the run changes (or would change) the file. Zero-activity runs print nothing. `internal/idea` MUST write nothing to stderr — counts flow up in `FmtResult` and `cmd/idea` prints (Constitution IV split).

- **GIVEN** a backlog where fmt adopts 1 line and backfills 1 date
- **WHEN** `idea fmt` runs
- **THEN** stdout is empty and stderr carries the adoption line, the backfill note, and the summary counts

### Specs: contract documentation

#### R8: Spec updates for the widened rewrite contract
`docs/specs/backlog-format.md` MUST document the explicit canonicalizer: a section describing `idea fmt` (canonical rewrite, adoption-candidate shape and bracket precision guard, `--check` contract), a carve-out in Round-Trip Preservation (bare checkbox lines were previously guaranteed non-idea pass-through; after this change `idea fmt` — and only `fmt` — claims them), and a new format-contract change note. Shape B guarantees stay unchanged. `docs/specs/overview.md` MUST gain a command-table row for `idea fmt` and a sentence on the explicit canonicalizer in the parse/format section.

- **GIVEN** an external consumer reading the specs
- **WHEN** they consult Round-Trip Preservation and the command table
- **THEN** the `fmt` adoption carve-out and the new command are documented, and Shape B pass-through guarantees read unchanged

### Non-Goals

- No separate `--dry-run` flag — the user-confirmed single `--check` flag serves both preview and CI gate.
- No adoption by any other command — mutating CRUD commands keep only their incidental normalize-on-write; `list`/`show` stay non-mutating.
- No help-dump JSON schema change — the new node appears via the existing walk.
- No `docs/memory/` edits — `cli/structure.md` is hydrate's responsibility, not apply's.
- No new dependencies (stdlib + cobra only).

### Design Decisions

1. **Internal seams**: split `LoadFile` into `parseContent(content) *File` + thin file-reading wrapper, and `SaveFile` into `render(f) (string, int)` + atomic write — *Why*: `Fmt` needs the original bytes (byte-stability/skip-write) and the rebuilt content without writing (`--check`), while keeping one parse walk and one serialization point — *Rejected*: re-reading the file twice in `Fmt` (racy, double I/O) and duplicating the join/stamp logic (second serialization path, forbidden).
2. **Raw-line retention**: `File.lines` stores each idea line's raw (post-`\r`-strip) text instead of the `""` placeholder — *Why*: per-line "normalized" counting needs the original raw line, which `SaveFile`'s copy-then-overwrite rebuild never reads, so the change is invisible to all existing callers — *Rejected*: a parallel `rawIdea []string` field (must be maintained by `Rm`'s splicing — a drift landmine).
3. **Whole-file `Changed`**: the write/exit verdict compares rebuilt content against the original bytes — *Why*: catches CRLF-on-prose and EOF-newline differences that per-line idea comparison cannot — *Rejected*: deriving `Changed` from the counts (misses non-idea-line byte changes).

## Tasks

### Phase 1: Internal refactors (no behavior change)

- [x] T001 Refactor `src/internal/idea/idea.go`: extract `parseContent(content string) *File` from `LoadFile` (thin wrapper remains), extract `render(f *File) (string, int)` from `SaveFile` (date-stamp + rebuild, no write), and retain each idea line's raw text in its `f.lines` slot (replacing the `""` placeholder; update comments). Run existing `internal/idea` tests to confirm zero behavior change. <!-- R2 R5 -->

### Phase 2: Core implementation (`internal/idea`)

- [x] T002 Create `src/internal/idea/fmt.go`: `adoptRegex` (`^\s*[-*+] \[([ xX])\] (.+)$`), candidate guards (fails `ParseLine`, text not bracket-led via `shapeBPrefixRegex`, text non-blank), `generateUniqueIDInSet` helper (in-memory uniqueness, retry-capped), and `adoptBareCheckboxes(f *File, today string) []Idea` that merges adopted ideas into `f.ideas`/`f.ideaIndices` in file order. <!-- R3 R4 -->
- [x] T003 Add `FmtResult{Adopted []Idea; Normalized int; Backfilled int; Changed bool}` and `Fmt(path string, check bool) (FmtResult, error)` to `src/internal/idea/fmt.go`: read bytes once → `parseContent` → early-return on empty file → count backfill + per-line normalized (canonical vs raw, pre-existing ideas only) → adoption pass → `render` → `Changed` = rebuilt != original → atomic write only when `!check && Changed`. <!-- R2 R3 R5 R6 -->
- [x] T004 Create `src/internal/idea/fmt_test.go`: table-driven tests with `t.TempDir()` covering canonicalization (bullets/indent/CRLF/dateless/legacy backslashes), adoption (bullet variants, indentation, `[X]`→done, fresh in-run-unique IDs, today's date, backslash text escaping), guards (Shape B, `[DEV-1011]`/`[TODO]`/`[ab1]`, text-less and whitespace-only checkboxes), verbatim preservation, count separation (adopted vs backfilled), idempotency + skip-write (mtime unchanged), check mode (no write, `Changed` reported), empty file no-op, missing file error, and the intake worked example. <!-- R2 R3 R4 R5 R6 -->

### Phase 3: CLI integration & e2e

- [x] T005 Create `src/cmd/idea/fmt.go` (`fmtCmd()` with `--check`, stderr report: `adopted: [id] {escaped text}` lines → `printBackfillNotice` → summary counts when changed; `errSilent` on non-canonical `--check`) and register `fmtCmd()` in `newRootCmd()` in `src/cmd/idea/main.go`. <!-- R1 R6 R7 -->
- [x] T006 Create `src/cmd/idea/fmt_test.go` e2e tests (reusing `buildBinary`/`setupGitRepo`/`writeRepoBacklog`/`runSplit`/`readRepoBacklog`): routing (`idea fmt` never falls through to add; `idea fmt some text` errors, backlog untouched), stdout-empty/stderr-report split, `--check` exit codes (1 non-canonical + file unchanged, 0 clean + silent), adoption worked example end-to-end, idempotent second run. Add the `fmt` row to `TestHelpDump_RealSubcommandsPresent` in `src/cmd/idea/help_dump_test.go`. <!-- R1 R6 R7 -->

### Phase 4: Docs & verification

- [x] T007 Update `docs/specs/backlog-format.md`: new "Explicit Canonicalization & Adoption (`idea fmt`)" section, Round-Trip Preservation carve-out, format-contract change note (Shape B unchanged). <!-- R8 -->
- [x] T008 Update `docs/specs/overview.md`: `idea fmt` command-table row + explicit-canonicalizer sentence in the parse/format section. <!-- R8 -->
- [x] T009 Verification: `cd src && go test ./...` (full suite), `gofmt -l` on all changed Go files, `go vet ./...` — all clean. <!-- R1 R2 R3 R4 R5 R6 R7 R8 -->

### Rework (post-review)

- [x] T010 Evaluate the bracket precision guard against the whitespace-trimmed capture and store adopted text trimmed in `src/internal/idea/fmt.go`; add guard-bypass rows (`- [ ]  [DEV-1011] x`, `- [ ]  [ab12] 2026-01-01: x`) and an adopted-text-trim row to `src/internal/idea/fmt_test.go`; mirror the trimmed-guard wording in `docs/specs/backlog-format.md`. <!-- R3 R4 --> <!-- rework: review should-fix — whitespace bypass of bracket guard -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `idea fmt` exists as a registered `fmtCmd()` factory with `--check`, inherits `--file`/`--main`/`IDEAS_FILE`, takes no positional args, and carries enriched `Long` help; stray args error without touching the backlog
- [x] A-002 R2: fmt canonicalizes variant bullets, indentation, CRLF, dateless lines, and legacy backslashes through the existing render machinery — no second serialization path exists in the diff
- [x] A-003 R3: bare checkboxes (`-`/`*`/`+` bullets, indentation, `[ ]`/`[x]`/`[X]`) are adopted with fresh 4-char IDs unique against the file and within the run, today's date, and preserved checked state
- [x] A-004 R4: bracket-led candidates (Shape B, `[DEV-1011]`, `[TODO]`, `[ab1]`) and text-less checkboxes remain byte-for-byte; headers/prose/blank lines verbatim
- [x] A-005 R5: a second fmt run is byte-stable and skips the write entirely (mtime unchanged); a 0-byte file is untouched
- [x] A-006 R6: `--check` writes nothing, prints the would-be report to stderr, exits 1 when non-canonical and 0 when clean
- [x] A-007 R7: stdout is empty on every fmt path; stderr carries the per-line adoption report, the existing backfill advisory wording, and summary counts; zero-activity runs print nothing; `internal/idea` performs no stderr writes
- [x] A-008 R8: both spec files document the new command and the widened-rewrite carve-out; Shape B guarantees read unchanged

### Scenario Coverage

- [x] A-009 R3: the intake worked example (mixed `* [ ]`/`- [X]`/`- [ ] [DEV-1011]` input) is pinned by a test asserting the exact output shape
- [x] A-010 R3: tests assert adopted lines count as adopted but never as backfilled, and pre-existing dateless managed lines count as backfilled — no cross-counting between the two (a backfilled line also counting as normalized is correct: its bytes change)
- [x] A-011 R5: an idempotency test asserts the second run reports `Changed == false` with identical bytes

### Edge Cases & Error Handling

- [x] A-012 R4: whitespace-only checkbox text (`- [ ]   `) is not adopted
- [x] A-013 R3: adopted text containing a literal backslash persists escaped (`a\b` → `a\\b` on disk) with decoded content unchanged
- [x] A-014 R2: fmt on a missing backlog file exits non-zero with the read error

### Code Quality

- [x] A-015 Pattern consistency: new code follows the factory/`RunE`/`resolveFile` and `internal/idea` naming + comment conventions of surrounding code
- [x] A-016 No unnecessary duplication: reuses `ParseLine`, `FormatLine`, `EscapeText`, `shapeBPrefixRegex`, `generateRandomID`, `atomicWriteFile`, and `printBackfillNotice` rather than reimplementing
- [x] A-017 No god functions: `Fmt` and `adoptBareCheckboxes` stay focused (< 50 lines each without clear reason)
- [x] A-018 No magic strings: regexes and report formats live in named package-level vars/clearly-scoped literals consistent with the codebase
- [x] A-019 Tests are table-driven against real temp dirs (`t.TempDir()`), no mocks (Constitution V); gofmt and go vet clean

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant. The only superseded convention (the `""` placeholder for idea slots in `File.lines`, `src/internal/idea/idea.go`) was replaced in place by T001, leaving nothing behind; `generateUniqueID` and `checkIDCollision` retain their call sites in `Add`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | API shape `Fmt(path string, check bool) (FmtResult, error)` with `FmtResult{Adopted []Idea, Normalized, Backfilled int, Changed bool}` | Intake sketched this exact signature ("exact name/shape is a plan decision"); the fields are precisely what the cmd layer's report needs | S:80 R:85 A:85 D:75 |
| 2 | Confident | Internal refactor: `parseContent`/`render` extraction + raw idea lines retained in the `File.lines` placeholder slots | Intake offered "retain raw lines in File or compare whole-file bytes"; both seams are used (per-line counts + byte-stability) with `LoadFile`/`SaveFile` public behavior unchanged | S:75 R:80 A:90 D:70 |
| 3 | Confident | stderr composition: adoption lines, then the existing `printBackfillNotice` line verbatim, then `fmt: N line(s) normalized, M line(s) adopted` printed only when the file changes | Intake fixes channel + advisory tone and delegates composition; reusing `printBackfillNotice` satisfies "wording reused" while the summary carries the remaining two counts | S:70 R:90 A:80 D:60 |
| 4 | Confident | Adoption report line format `adopted: [id] {escaped text}` using `EscapeText` | Intake gives the format as an example; escaping keeps one physical report line per adopted idea, matching the `Added:` confirmation precedent | S:80 R:90 A:85 D:70 |
| 5 | Confident | `--check` prints the identical report including freshly generated would-be IDs (not persisted; a later real run assigns different IDs) | Clarified wording says "prints the same report"; one shared code path is the simplest honest implementation of that contract | S:75 R:90 A:75 D:60 |
| 6 | Confident | Whitespace-only checkbox text is not adopted (TrimSpace guard beyond the regex's `(.+)`) | Extends the intake's non-empty-text rule to its evident intent; errs toward preservation per Constitution I | S:65 R:90 A:85 D:75 |
| 7 | Confident | Missing backlog file → error propagates (like `done`/`edit`); 0-byte file → silent no-op (no trailing LF invented) | Consistent with existing mutating-command behavior; Constitution I forbids inventing content in a file fmt was asked to preserve | S:60 R:90 A:85 D:70 |
| 8 | Confident | `Normalized` counts pre-existing managed lines whose canonical form differs from the raw (post-`\r`-strip) line; CRLF-only/EOF-newline-only differences still trigger the rewrite and non-zero `--check` via whole-file `Changed` but may not increment per-line counts | Counting mechanism explicitly delegated to plan; the whole-file compare guarantees the write/exit contract is exact even where the count is approximate | S:60 R:85 A:75 D:65 |
| 9 | Certain | `fmtCmd()` joins the existing `AddCommand` list, the help-dump node appears via the existing walk (no schema change), and `Args: cobra.NoArgs` makes stray args error rather than fall through | Intake states the namespace note and schema invariance explicitly; cobra mechanics are deterministic | S:90 R:90 A:90 D:90 |
| 10 | Confident | Adopted text is whitespace-trimmed before storage, and the bracket precision guard evaluates the trimmed capture | Rework decision (review should-fix): leading/trailing spaces are checkbox surface formatting, not content — canonical lines use single-space delimiters; trimming both closes the guard bypass (`- [ ]  [DEV-1011] x`) and keeps adopted canonical lines free of leading-space noise | S:65 R:90 A:85 D:75 |

10 assumptions (1 certain, 9 confident, 0 tentative).
