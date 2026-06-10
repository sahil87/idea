# Plan: Resilient Backlog Parser

**Change**: 260610-wtmn-resilient-backlog-parser
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- RFC-2119. Lenient on read, canonical on write. Derived from intake §What Changes
     and the settled ## Assumptions / ## Clarifications tables. -->

### Parser: Lenient Read

#### R1: Optional date segment
`ParseLine` MUST parse an idea line whether or not the `YYYY-MM-DD:` date segment is present. A dateless line MUST parse to the same `Idea` as its dated counterpart, modulo the `Date` field (which is `""` when absent).

- **GIVEN** the line `- [ ] [rk7t] 2026-06-10: Tune the reporter`
- **WHEN** `ParseLine` is called
- **THEN** it returns an `Idea{ID:"rk7t", Date:"2026-06-10", Text:"Tune the reporter", Done:false}` and `ok=true`
- **AND GIVEN** the line `- [ ] [rk7t] Tune the reporter`
- **WHEN** `ParseLine` is called
- **THEN** it returns `Idea{ID:"rk7t", Date:"", Text:"Tune the reporter", Done:false}` and `ok=true`

#### R2: Variant bullet markers
`ParseLine` MUST accept `-`, `*`, and `+` as the bullet marker on input.

- **GIVEN** the line `* [ ] [a7k2] do a thing` or `+ [x] [a7k2] do a thing`
- **WHEN** `ParseLine` is called
- **THEN** it parses to an `Idea` with `ok=true`, `ID="a7k2"`, correct `Done` flag, and `Text="do a thing"`

#### R3: Leading whitespace tolerance
`ParseLine` MUST accept arbitrary leading whitespace (spaces and tabs) before the bullet.

- **GIVEN** the line `␣␣- [ ] [a7k2] indented idea` (leading spaces) or a tab-indented line
- **WHEN** `ParseLine` is called
- **THEN** it parses to `Idea{ID:"a7k2", Text:"indented idea"}` with `ok=true`

#### R4: CRLF line-ending tolerance
`LoadFile` MUST strip a trailing carriage return from each line before parsing, so lines from a CRLF-terminated file parse identically to LF lines. The stripped `\r` is part of canonicalization — it is never re-emitted.

- **GIVEN** a backlog file whose lines end in `\r\n`
- **WHEN** `LoadFile` parses it
- **THEN** each idea line parses correctly (the trailing `\r` does not leak into `Text`)
- **AND** a subsequent `SaveFile` emits LF-only line endings

#### R5: Canonical line still parses (regression)
The fully-canonical form (`- ` bullet, no indent, date present, LF) MUST continue to parse exactly as before this change. No existing valid line stops parsing.

- **GIVEN** the line `- [x] [e5f6] 2025-06-08: Fix login redirect bug`
- **WHEN** `ParseLine` is called
- **THEN** it returns `Idea{ID:"e5f6", Date:"2025-06-08", Text:"Fix login redirect bug", Done:true}` with `ok=true`

### Parser: Precision Guard

#### R6: Legacy Shape B lines stay inert pass-through
The relaxed regex MUST NOT match a second-bracket "Shape B" line (`- [ ] [id] [DEV-1011] 2026-02-12: text`). Such lines MUST remain unparsed and be preserved verbatim through any mutating save. The `[issue_ids]` slot stays owned by external consumers.

- **GIVEN** the line `- [ ] [ni3o] [DEV-1011] 2026-02-12: Capture more metrics`
- **WHEN** `ParseLine` is called
- **THEN** it returns `ok=false`
- **AND WHEN** the containing file undergoes a mutating save (e.g. `done` on a different idea)
- **THEN** the Shape B line is preserved byte-for-byte

#### R7: Genuine non-idea prose is not parsed
The relaxed regex MUST NOT match headers, blank lines, or prose that lacks the `[ ]`/`[x]` checkbox + 4-char `[id]` anchors. Lines missing those anchors remain non-idea pass-through.

- **GIVEN** lines like `# Backlog`, `Some footer text`, `- a plain bullet`, `- [ ] no id here`, `- [ ] [toolong5] text`
- **WHEN** `ParseLine` is called on each
- **THEN** it returns `ok=false` for every one

### Writer: Canonical Output

#### R8: Canonical serialization on write
`FormatLine` MUST emit the canonical form `- [<check>] [<id>] <date>: <text>` (`- ` bullet, no leading whitespace, date present, single space delimiters). `SaveFile` MUST join lines with LF and end the file with a single trailing LF. A mutating save MUST canonicalize every recognized idea line in the file at once (normalize-on-write).

- **GIVEN** a file containing `*` bullets, indented lines, and CRLF endings among recognized idea lines
- **WHEN** any mutating command saves the file
- **THEN** every recognized idea line is rewritten canonical: `- ` bullet, no indent, LF ending, date present
- **AND** non-idea lines are untouched

#### R9: Date backfill on save
When a recognized idea has `Date == ""` at save time, `SaveFile` MUST backfill today's date (`time.Now().Format("2006-01-02")`) before serializing, so every persisted idea line carries a date. `ParseLine` stays pure (it does not stamp), keeping `MarshalJSON` correct because the in-memory `Idea` has a date by the time it is marshaled after a save.

- **GIVEN** the dateless line `- [ ] [rk7t] Tune the reporter` and today is `2026-06-10`
- **WHEN** `idea done rk7t` runs
- **THEN** the persisted line is `- [x] [rk7t] 2026-06-10: Tune the reporter`

#### R10: Backfill emits a single stderr notice
`SaveFile` MUST report the count of dates it backfilled. The `cmd/idea` layer MUST print a brief notice to **stderr** — `note: stamped today's date on N previously-dateless item(s)` — when `N > 0`, and MUST suppress it entirely when `N == 0`. stdout MUST remain the machine-parseable confirmation only (Constitution VI). Existing `done`/`edit`/`reopen`/`rm` confirmations stay on stdout.

- **GIVEN** a file with 2 dateless recognized ideas
- **WHEN** a mutating command saves it
- **THEN** stderr receives exactly `note: stamped today's date on 2 previously-dateless item(s)`
- **AND** stdout receives only the normal command confirmation
- **AND GIVEN** a file with 0 dateless ideas
- **WHEN** a mutating command saves it
- **THEN** stderr receives no notice

### Round-Trip Preservation

#### R11: Non-idea content preserved verbatim
Headers, prose, and blank lines MUST survive a mutating save byte-for-byte (Constitution I). `add` is out of scope and unchanged.

- **GIVEN** a file with a `# Backlog` header, a blank line, prose, and idea lines
- **WHEN** a mutating command saves it
- **THEN** the header, blank line, and prose are unchanged (only recognized idea lines are canonicalized)

### Docs

#### R12: Spec docs reflect lenient-read / canonical-write
`docs/specs/backlog-format.md` and `docs/specs/overview.md` MUST be rewritten so the date is documented as optional on input, output is documented as canonical (`-` bullet, LF, date always present, backfilled if absent), the accepted input variants (bullet markers, leading whitespace, CRLF) are listed, and the format-contract change is noted explicitly. Shape B remains documented as inert pass-through.

- **GIVEN** a reader of `backlog-format.md`
- **WHEN** they read the contract
- **THEN** they learn that input is liberal (optional date, variant bullets/whitespace/CRLF) and output is a single canonical, machine-parseable form

### Non-Goals

- `add` command behavior — already stamps today's date; out of scope.
- Parsing or assigning semantics to the Shape B `[issue_ids]` bracket — stays owned by external consumers.
- `docs/memory/` updates — handled in the later hydrate stage, not here.
- Changing the JSON output schema — unchanged (`MarshalJSON` stays correct via save-time backfill).

### Design Decisions

1. **stderr-notice seam — count surfaced up to cmd layer**: `SaveFile` returns `(int, error)` (count backfilled); the mutating internal ops (`Done`, `Reopen`, `Edit`, `Rm`) return that count to `cmd/idea`, which prints the stderr notice. — *Why*: Constitution IV (logic in `internal/idea`, output formatting in `cmd/`); keeps the package free of direct `os.Stderr` writes and keeps output channel choice in the command layer. — *Rejected*: writing to `os.Stderr` from inside the package (couples I/O policy to the logic layer, harder to test the channel).
2. **Backfill at the save seam, not in ParseLine**: stamp `Date==""` ideas inside `SaveFile` just before serialization. — *Why*: keeps `ParseLine` pure so `MarshalJSON` and dateless round-trips stay correct; matches the existing regenerate-on-save architecture. — *Rejected*: stamping in `ParseLine` (would lose the dateless signal and pollute pure reads like `list`).
3. **Regex anchored on checkbox + 4-char id, optional non-capturing date group**: `^\s*[-*+] \[([ x])\] \[([a-z0-9]{4})\] (?:(\d{4}-\d{2}-\d{2}): )?(.+)$`. — *Why*: the `[ ]`/`[x]` + `[a-z0-9]{4}` anchors keep false-positive risk low; the optional non-capturing group makes the date optional without a second regex. A Shape B line fails because after `[id] ` the next token is `[DEV-...]`, which matches neither the date group nor is consumed as text in a way that yields a valid date — it falls into `Text`, but the precision guard test pins that Shape B is NOT treated as an idea via the second-bracket structure. — *Note*: see Assumption 1 for the Shape B exclusion mechanism.

## Tasks

### Phase 2: Core Implementation

- [x] T001 Relax `lineRegex` in `src/internal/idea/idea.go` to `^\s*[-*+] \[([ x])\] \[([a-z0-9]{4})\] (?:(\d{4}-\d{2}-\d{2}): )?(.+)$` and update `ParseLine` capture-group handling so the optional date group (m[3]) yields `Date==""` when absent; keep `ParseLine` pure (no date stamping). <!-- R1 R2 R3 R5 R7 -->
- [x] T002 Add a Shape B exclusion guard in `ParseLine` (`src/internal/idea/idea.go`): after the relaxed regex matches, reject the line when the captured text begins with a `[...]` second bracket immediately following the id (i.e. `- [ ] [id] [DEV-1011] ...`), returning `ok=false` so Shape B stays inert pass-through. <!-- R6 -->
- [x] T003 Update `LoadFile` in `src/internal/idea/idea.go` to strip a trailing `\r` from each split line before `ParseLine`, so CRLF files parse and non-idea CRLF lines are preserved as their LF form is not required (only recognized idea lines canonicalize; non-idea lines keep their original bytes minus the line split). <!-- R4 R11 -->
- [x] T004 Change `SaveFile` in `src/internal/idea/idea.go` to return `(int, error)`: before serializing, for each idea with `Date==""` stamp `time.Now().Format("2006-01-02")` and increment a counter; return the counter. `FormatLine` stays canonical and unchanged. <!-- R8 R9 -->
- [x] T005 Update the internal mutating ops `Done`, `Reopen`, `Edit`, `Rm` in `src/internal/idea/idea.go` to return the backfill count from `SaveFile` up to their callers (new signature `(Idea, int, error)`), preserving existing error semantics. <!-- R9 R10 -->

### Phase 3: Integration & Edge Cases

- [x] T006 Update `src/cmd/idea/done.go`, `reopen.go`, `edit.go`, `rm.go` to consume the new `(Idea, int, error)` signatures and, when count > 0, print `note: stamped today's date on N previously-dateless item(s)` to `cmd.ErrOrStderr()`; keep the existing confirmation on stdout. <!-- R10 -->

### Phase 4: Tests & Docs

- [x] T007 [P] Add table-driven parse tests in `src/internal/idea/idea_test.go`: dateless parse, CRLF parse, leading-whitespace/indented parse, `*` and `+` bullet parse, canonical-still-parses regression; negative cases: Shape B stays `ok=false`, non-idea prose (`# Backlog`, footer, `- [ ] no id`, bad id length) stays `ok=false`. <!-- R1 R2 R3 R4 R5 R6 R7 -->
- [x] T008 [P] Add table-driven save/round-trip tests in `src/internal/idea/idea_test.go` (real `t.TempDir()`): dateless→`Done`/`Edit` backfills today's date + canonicalizes and returns count; variant bullet/indent/CRLF → canonicalized on save; Shape B line preserved byte-for-byte through a mutating save; headers/prose/blank lines survive a mutating save byte-for-byte; `SaveFile` returns correct backfill count (incl. 0). <!-- R8 R9 R10 R11 -->
- [x] T009 [P] Add cmd-layer tests in `src/cmd/idea/` (or extend `main_test.go`) asserting the stderr notice appears with the correct count when dateless items are stamped and is absent when count is 0, while stdout carries only the confirmation. <!-- R10 -->
- [x] T010 [P] Rewrite `docs/specs/backlog-format.md`: date optional on input; canonical output (`-` bullet, LF, date always present, backfilled if absent); accepted input variants table (bullet markers, leading whitespace, CRLF); explicit format-contract change note; Shape B remains inert pass-through. <!-- R12 -->
- [x] T011 [P] Update `docs/specs/overview.md` parse/format description to lenient-read / canonical-write (optional date on input, backfill-on-save, stderr notice). <!-- R12 -->

## Execution Order

- T001 → T002 (Shape B guard depends on the relaxed regex existing).
- T004 → T005 → T006 (signature change cascades cmd-ward).
- T007–T011 run after their respective implementation tasks; all `[P]` relative to each other.

## Acceptance

### Functional Completeness

- [ ] A-001 R1: Dateless and dated lines both parse to the same `Idea` modulo `Date` (`Date==""` when absent).
- [ ] A-002 R2: `-`, `*`, `+` bullets all parse.
- [ ] A-003 R3: Leading whitespace (spaces/tabs) before the bullet parses.
- [ ] A-004 R4: CRLF lines parse; trailing `\r` never leaks into `Text`; output is LF.
- [ ] A-005 R5: Canonical line still parses identically (regression green).
- [ ] A-006 R8: `FormatLine` emits canonical form; `SaveFile` canonicalizes all recognized idea lines on a mutating save and ends file with single LF.
- [ ] A-007 R9: Dateless idea gets today's date on `done`/`edit`; `MarshalJSON` stays correct (date populated post-save).
- [ ] A-008 R12: Both spec docs rewritten to lenient-read / canonical-write with the variant table and the format-contract change note.

### Behavioral Correctness

- [ ] A-009 R10: stderr notice `note: stamped today's date on N previously-dateless item(s)` printed when N>0, suppressed when N==0; stdout carries only the confirmation.

### Scenario Coverage

- [ ] A-010 R9: Worked example holds — `- [ ] [rk7t] Tune the reporter` → `idea done rk7t` → `- [x] [rk7t] <today>: Tune the reporter`.

### Edge Cases & Error Handling

- [ ] A-011 R6: Shape B `- [ ] [id] [DEV-1011] date: text` returns `ok=false` and is preserved byte-for-byte across a mutating save (regression-pinned).
- [ ] A-012 R7: Non-idea prose / headers / blanks / bad-id lines all return `ok=false`.
- [ ] A-013 R11: Headers, prose, and blank lines survive a mutating save byte-for-byte.

### Code Quality

- [ ] A-014 Pattern consistency: New code follows surrounding `internal/idea` naming/structure and thin-cobra conventions (Constitution III/IV).
- [ ] A-015 No unnecessary duplication: Reuses existing helpers (`FormatLine`, `SaveFile`, `time.Now().Format`) rather than reimplementing.
- [ ] A-016 No god functions / magic strings: backfill date format reuses the existing `"2006-01-02"` layout already used by `Add`; notice string is a single clear literal.
- [ ] A-017 Tests are table-driven against real temp dirs (`t.TempDir()`), per Constitution V and code-quality test-alongside strategy.

## Notes

- Check items as you review: `- [x]`
- Module root is `src/` (`src/go.mod`); run `cd src && go test ./...` and `cd src && go build ./...`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Shape B exclusion is enforced by an explicit guard in `ParseLine` (reject when the captured text begins with a `[...]` bracket immediately after the id), not solely by regex shape — because the relaxed `(.+)` text group would otherwise swallow `[DEV-1011] date: text` into `Text` and wrongly treat it as a dateless idea. | The relaxed regex alone cannot distinguish Shape B from a dateless idea whose text legitimately starts with a bracket; an explicit, well-tested guard is the clearest correct mechanism and is pinned by a regression test. | S:80 R:75 A:80 D:70 |
| 2 | Confident | Backfill count is surfaced up via new `(Idea, int, error)` return signatures on `Done`/`Reopen`/`Edit`/`Rm` and `(int, error)` on `SaveFile`; cmd layer prints to `cmd.ErrOrStderr()`. | Intake explicitly preferred surfacing the count to `cmd/idea` for printing (Constitution IV). Signature change is mechanical and fully internal (no external API). | S:85 R:70 A:85 D:80 |
| 3 | Certain | Notice wording is exactly `note: stamped today's date on N previously-dateless item(s)` on stderr, suppressed at N==0. | Verbatim from intake Clarification #9 / Assumption 9. | S:95 R:85 A:80 D:85 |
| 4 | Confident | `reopen` is included in the backfill-notice plumbing even though it operates on already-dated done items, because normalize-on-write may backfill OTHER dateless lines in the same file. | Direct consequence of the regenerate-on-save / normalize-on-write architecture (Assumption 5). | S:80 R:75 A:80 D:75 |
| 5 | Certain | Non-idea lines that contained a CRLF keep their original text (minus the line-split `\n`); only recognized idea lines are canonicalized to LF. Whole-file output is LF-joined. | Matches existing `SaveFile` join-on-`\n` behavior and Constitution I verbatim preservation of non-idea content; canonicalization scope is the recognized idea lines. | S:90 R:75 A:85 D:80 |

5 assumptions (2 certain, 3 confident, 0 tentative).
