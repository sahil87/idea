# Plan: Editor-Based Editing for `idea edit`

**Change**: 260612-w4p7-edit-via-editor
**Intake**: `intake.md`

## Requirements

### CLI: Two-Form Argument Contract

#### R1: Arg-presence mode switch
`idea edit` SHALL accept one or two positional args (`cobra.RangeArgs(1, 2)`). The two-arg form (`idea edit <query> "text"`) MUST keep current behavior byte-for-byte: inline replacement, no editor launch, same validation, same output. The one-arg form (`idea edit <query>`) SHALL open the resolved editor on the idea's decoded text.

- **GIVEN** a backlog with idea `[ab12] hello`
- **WHEN** `idea edit ab12 "new text"` runs with `$EDITOR` set to a tripwire script
- **THEN** the text is replaced inline, `Updated: <FormatLine>` prints on stdout, and the tripwire never fires

- **GIVEN** the same backlog
- **WHEN** `idea edit ab12` runs with `$EDITOR` set to a script that rewrites the buffer
- **THEN** the editor opens on the decoded text and the rewritten buffer is persisted canonically

### internal/idea: Editor Plumbing

#### R2: Editor resolution chain
A new `ResolveEditor() string` in `src/internal/idea/editor.go` SHALL return `$VISUAL` if non-empty, else `$EDITOR` if non-empty, else `vi`. The returned value is a command string that MAY contain arguments (e.g. `code --wait`). Only stdlib (`os/exec`) is used — no new dependencies.

- **GIVEN** `VISUAL=code --wait` and `EDITOR=vim`
- **WHEN** `ResolveEditor()` is called
- **THEN** it returns `code --wait`
- **AND** with `VISUAL` empty it returns `vim`, and with both empty it returns `vi`

#### R3: Temp-file round trip with post-processing
A new `EditInEditor(text string) (string, error)` SHALL: create a temp file via `os.CreateTemp("", "idea-edit-*.md")` seeded with the raw decoded text; launch the editor through the shell as `sh -c '<editor> "$1"' sh <tmpPath>` with inherited stdio (no TTY gate); remove the temp file on return. On editor exit 0 it SHALL read the buffer back, apply `normalizeCR`, strip **exactly one** trailing LF (further trailing blank lines are content), and return the result. On non-zero exit or launch failure it SHALL return an error and the caller MUST persist nothing.

- **GIVEN** an idea whose stored text is `a\nb` (escaped on disk)
- **WHEN** the one-arg form opens the editor
- **THEN** the editor buffer contains the decoded text `a` LF `b` (real newline, no escape sequences)

- **GIVEN** an editor script that writes `changed` plus a final newline
- **WHEN** the buffer is post-processed
- **THEN** the returned text is `changed` with no trailing LF

#### R4: Persistence via existing canonical writer, resolve-before-launch
The one-arg form SHALL resolve the matched idea via the existing query semantics (substring, case-insensitive, ID or text, `RequireSingle`) **before** launching the editor — an ambiguous or unmatched query is refused with the match list and the editor never opens. Persistence SHALL go through the existing `idea.Edit(path, query, newText, newID, newDate)`, inheriting normalize-on-write, date backfill with stderr notice, atomic save, and verbatim non-idea-line preservation.

- **GIVEN** two ideas whose text contains `hello`
- **WHEN** `idea edit hello` runs with a tripwire `$EDITOR`
- **THEN** the command refuses with `Multiple matches`, exits non-zero, and the tripwire never fires

### CLI: Edge Semantics (one-arg form)

#### R5: Abort, unchanged, and emptied buffers
The one-arg form SHALL implement: (a) editor non-zero exit / launch failure → no change to the backlog, error via the `RunE` error path mentioning the idea is unchanged, non-zero exit; (b) post-processed buffer equal to the current text with no `--id`/`--date` given → no-op with **no file rewrite** (no normalize/backfill side effects), `note: text unchanged — nothing to do` on stderr, exit 0; (c) post-processed buffer empty → refuse with a clear error in the cmd layer (the `idea.Edit` `text is required` guard is the safety net), no change, non-zero exit.

- **GIVEN** a backlog containing a non-canonical (dateless, star-bullet) line alongside the target idea
- **WHEN** the editor exits 0 without touching the buffer
- **THEN** the backlog file is byte-identical (proving no rewrite) and the note appears on stderr with exit 0

- **GIVEN** an editor script that rewrites the buffer but exits 3
- **WHEN** `idea edit ab12` runs
- **THEN** the backlog is byte-identical, the exit code is non-zero, and the error mentions the idea is unchanged

- **GIVEN** an editor script that truncates the buffer to empty
- **WHEN** `idea edit ab12` runs
- **THEN** the command errors, the backlog is unchanged, and the exit code is non-zero

#### R6: `--id`/`--date` interaction
With the one-arg form, `--id`/`--date` SHALL still open the editor; the flags apply at save via the existing `idea.Edit` params. The unchanged-text no-op MUST be suppressed when either flag is present, so a metadata-only change still lands.

- **GIVEN** idea `[ab12] 2026-06-10: hello` and an editor script that leaves the buffer untouched
- **WHEN** `idea edit ab12 --date 2026-01-01` runs
- **THEN** the date change persists, `Updated:` prints on stdout, and no "unchanged" note appears

#### R7: Output channels
Success output SHALL be unchanged: `Updated: <FormatLine(i)>` (escaped single line) on **stdout**; the backfill notice on stderr when applicable. All no-op/abort messaging SHALL go to **stderr** only (Constitution VI: stdout stays machine-parseable; channel policy lives in `cmd/` per Constitution IV).

- **GIVEN** any no-op or abort outcome of the one-arg form
- **WHEN** the command finishes
- **THEN** stdout is empty and the message appears on stderr

### Docs: Help and Spec

#### R8: Help text and overview spec
The edit command's cobra `Long` SHALL describe both forms with examples; `Short` stays byte-stable; the `Use` line reflects the now-optional text arg (`edit <query> [new-text]`). `docs/specs/overview.md`'s command table SHALL document both forms (one-arg opens `$VISUAL`/`$EDITOR`/`vi`; two-arg replaces inline). The help-dump JSON schema is unchanged (node `text` updates automatically).

- **GIVEN** the updated binary
- **WHEN** `idea edit -h` runs
- **THEN** the help describes both the inline and the editor-based form

### Tests

#### R9: Table-driven coverage per intake §6
Tests SHALL be table-driven, with fake editors as small shell scripts in `t.TempDir()` (set via `t.Setenv` for in-process tests, `cmd.Env` for subprocess tests — with host `VISUAL`/`EDITOR` scrubbed), real temp dirs/repos, no mocks (Constitution V). The ten intake cases MUST be covered: editor rewrite, multiline round-trip with decoded-buffer assertion, trailing-LF, abort, unchanged no-op with byte-identical file, emptied buffer, `$VISUAL`/`$EDITOR`/`vi` resolution, two-arg tripwire regression, `--date` no-op suppression, ambiguous query.

- **GIVEN** the new test tables in `src/cmd/idea/main_test.go` and `src/internal/idea/editor_test.go`
- **WHEN** `cd src && go test ./...` runs
- **THEN** all ten intake cases pass alongside the existing suite

### Non-Goals

- No TTY detection — a TTY-requiring editor fails fast on its own in non-interactive contexts (intake assumption #6)
- No new flags, no TUI/survey dependencies, no change to query semantics, the backlog format contract, other subcommands, or the help-dump JSON schema

### Design Decisions

1. **Resolve-before-launch via `idea.Show`**: the one-arg form reuses `idea.Show` (LoadFile + `RequireSingle`, `FilterAll`) to fetch the current decoded text before opening the editor — *Why*: identical match semantics to `idea.Edit`'s own resolution, refuses ambiguity before any editor opens, zero new resolver code — *Rejected*: a new `internal/idea` resolve helper (duplicates `Show`).
2. **Shell launch with positional param**: `sh -c '<editor> "$1"' sh <tmpPath>` — *Why*: multi-word `$EDITOR` values work and the path needs no quoting — *Rejected*: splitting the editor string in Go (fragile quoting).

## Tasks

### Phase 1: Core Implementation (internal/idea)

- [x] T001 Create `src/internal/idea/editor.go` with `ResolveEditor()` ($VISUAL → $EDITOR → `vi` fallback as a named constant; value may carry args) <!-- R2 -->
- [x] T002 Add `EditInEditor(text string) (string, error)` to `src/internal/idea/editor.go`: `os.CreateTemp("", "idea-edit-*.md")` seeded with decoded text, `sh -c '<editor> "$1"' sh <tmpPath>` launch with inherited stdio, deferred temp removal, non-zero-exit error mentioning the idea is unchanged, post-processing via existing `normalizeCR` + strip exactly one trailing LF <!-- R3 -->
- [x] T003 Add `src/internal/idea/editor_test.go`: table-driven `TestResolveEditor` (VISUAL wins / EDITOR fallback / `vi` default via `t.Setenv`) and `TestEditInEditor` (rewrite + trailing-LF strip, exactly-one-LF semantics, untouched buffer, CRLF normalization, multi-word editor value, non-zero exit) plus temp-file removal/naming check <!-- R2 R3 R9 -->

### Phase 2: Command Wiring (cmd/idea)

- [x] T004 Rework `src/cmd/idea/edit.go` `RunE`: `Args: cobra.RangeArgs(1, 2)`; two-arg path unchanged; one-arg path = resolve via `idea.Show` → `idea.EditInEditor` → empty-buffer refusal → unchanged no-op (`note: text unchanged — nothing to do` to stderr, exit 0) suppressed when `--id`/`--date` present → persist via existing `idea.Edit`; success output unchanged <!-- R1 R4 R5 R6 R7 -->
- [x] T005 Update `src/cmd/idea/edit.go` `Use` line to `edit <query> [new-text]` and `Long` help to describe both forms with an editor-form example (`Short` byte-stable) <!-- R8 -->

### Phase 3: Integration Tests & Docs

- [x] T006 Add to `src/cmd/idea/main_test.go`: helpers (`writeEditorScript`, `editorEnv` scrubbing host VISUAL/EDITOR, `runSplitEnv`) and table-driven `TestEdit_EditorForm` covering the intake's subprocess cases — editor rewrite, multiline round-trip with `$SIDE` decoded-buffer capture, trailing-LF no-op, abort, unchanged-buffer no-op with byte-identical non-canonical file, emptied buffer, VISUAL-wins-over-EDITOR, two-arg tripwire regression, `--date` no-op suppression, ambiguous query with tripwire <!-- R1 R4 R5 R6 R7 R9 -->
- [x] T007 Update `docs/specs/overview.md` command table: edit rows for both forms <!-- R8 -->

### Phase 4: Polish

- [x] T008 Run `gofmt -l` on touched Go files, `go vet ./...`, and the full `cd src && go test ./...`; fix anything surfaced <!-- R9 -->
- [x] T009 Close the trailing-LF no-op defeat: `EditInEditor` returns `(edited, unchanged, error)` where `unchanged` also compares the pre-strip (CR-normalized) buffer, so an untouched LF-terminated text is a byte-identical no-op; `edit.go` uses the verdict for the no-op branch and passes the original `Idea.Text` (not the stripped buffer) on a flag-forced metadata-only save; spec prose + memory reconciled; unit and subprocess tests added <!-- rework: post-pass outward-review should-fix absorption --> <!-- R5 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `idea edit` accepts 1 or 2 args; the two-arg form's behavior is byte-for-byte unchanged (tripwire test proves no editor launch); the one-arg form opens the editor on decoded text
- [x] A-002 R2: `ResolveEditor` returns `$VISUAL`, else `$EDITOR`, else `vi`; multi-word values pass through intact
- [x] A-003 R3: `EditInEditor` uses `idea-edit-*.md` temp files seeded with raw decoded text, launches via `sh -c '<editor> "$1"' sh <path>` with inherited stdio, removes the temp file, and post-processes with `normalizeCR` + exactly-one-trailing-LF strip
- [x] A-004 R4: persistence goes through the existing `idea.Edit` (normalize-on-write, backfill notice, atomic save); the match is resolved before the editor launches

### Behavioral Correctness

- [x] A-005 R5: editor non-zero exit → backlog byte-identical, non-zero exit, error mentions the idea is unchanged
- [x] A-006 R5: unchanged buffer without `--id`/`--date` → no file rewrite (byte-identical even with non-canonical lines present), `note: text unchanged — nothing to do` on stderr, exit 0
- [x] A-007 R5: emptied buffer → error, backlog unchanged, non-zero exit
- [x] A-008 R6: `--id`/`--date` with the one-arg form opens the editor, applies the flags at save, and suppresses the unchanged-text no-op
- [x] A-009 R7: stdout carries only `Updated: <FormatLine>` on success; all no-op/abort/backfill messaging is stderr-only

### Scenario Coverage

- [x] A-010 R9: all ten intake §6 cases exist as table-driven tests (fake editor scripts in `t.TempDir()`, host VISUAL/EDITOR scrubbed, no mocks) and pass via `cd src && go test ./...`
- [x] A-011 R3: the multiline round-trip test asserts the editor buffer received *decoded* text (side-file capture) and the rewritten multiline buffer persists re-escaped on one physical line

### Edge Cases & Error Handling

- [x] A-012 R4: ambiguous one-arg query is refused with the match list and the editor never launches (tripwire)
- [x] A-018 R5: an untouched session on LF-terminated text is a byte-identical no-op (pre-strip comparison; stderr note, exit 0); a flag-forced save on unchanged text preserves the original text verbatim; a deliberate edit of `a\n` to `a` still registers as a change

### Code Quality

- [x] A-013 Pattern consistency: new code follows the cobra-factory / `internal/idea`-seam patterns (Constitution III/IV); `cmd/idea/edit.go` stays wiring-only; gofmt-clean, `go vet` clean
- [x] A-014 No unnecessary duplication: reuses `normalizeCR`, `idea.Show`, `idea.Edit`, `printBackfillNotice`, and existing test helpers (`buildBinary`, `setupGitRepo`, `writeRepoBacklog`, `readRepoBacklog`)
- [x] A-015 No god functions: `RunE` stays a short orchestration body; `EditInEditor` is a single focused helper
- [x] A-016 No magic strings: editor fallback and temp-file pattern are named constants

### Docs

- [x] A-017 R8: `Long` help describes both forms (Short byte-stable, `Use` shows `[new-text]`); `docs/specs/overview.md` command table documents both forms

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | One-arg pre-resolve reuses existing `idea.Show` (LoadFile + `RequireSingle`, `FilterAll`) — no new resolver helper | Identical semantics to `Edit`'s own resolution; pure reuse; Constitution IV seam already in place | S:85 R:90 A:95 D:90 |
| 2 | Confident | Exact wording of abort/empty error strings (intake fixes only the unchanged-note verbatim): abort error appends `— idea unchanged`; empty-buffer error says `editor buffer is empty — idea unchanged (use "idea rm" to delete it)`; tests assert substrings | Intake mandates "mentions unchanged"/clear message, not exact strings; trivially reversible | S:70 R:92 A:85 D:75 |
| 3 | Confident | `Use` line becomes `edit <query> [new-text]` | RangeArgs makes the second arg optional; usage string must reflect it; help-dump schema unaffected | S:65 R:92 A:88 D:82 |
| 4 | Confident | `docs/specs/overview.md` edit row becomes two rows (one per form) matching the table's one-command-per-row style | Intake says "the edit row gains the two-form description"; two rows is the clearest fit for the existing table format; trivially reversible | S:70 R:95 A:80 D:68 |
| 5 | Certain | Intake test case 3 (editor appends one trailing LF to an unchanged one-liner) lands on the unchanged no-op path (exit 0, stderr note, byte-identical file) — that *is* the "round-trips without gaining `\n`" proof | Deterministic consequence of intake-decided rules #5 + #8 (strip one LF, then unchanged → no-op) | S:85 R:90 A:90 D:85 |
| 6 | Certain | Subprocess tests scrub host `VISUAL`/`EDITOR` from the inherited env before applying per-case overrides | Test correctness: a developer/CI `VISUAL` would otherwise shadow the fake `EDITOR` (resolution order is the feature under test) | S:80 R:95 A:95 D:90 |
| 7 | Confident | Extra unit cases beyond the intake's ten (CRLF normalization, multi-word editor value, temp-file removal/naming) added in `editor_test.go` | Test-alongside strategy; pins intake-decided behavior (#6, #7) without changing any contract | S:72 R:95 A:90 D:85 |

7 assumptions (3 certain, 4 confident, 0 tentative).
