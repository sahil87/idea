# Plan: Escape Multiline Text in Backlog Lines

**Change**: 260610-49mw-escape-multiline-idea-text
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

### CLI: Escape/Unescape Helpers (`internal/idea`)

#### R1: Escaping scheme
`internal/idea` SHALL export `EscapeText(s string) string` that converts real idea text to its persisted single-line form: CR normalization first (CRLF → LF, then any remaining lone CR → LF), then a single-pass replacement of `\` → `\\` (two chars) and LF → `\n` (two chars). The result MUST contain no raw LF and no raw CR, so the persisted idea line is always exactly one physical line. Implementation MUST be stdlib-only (`strings.Replacer` / `strings.ReplaceAll`).

- **GIVEN** the text `first line` + LF + LF + `second paragraph` + LF + `- [ ] looks like a task`
- **WHEN** `EscapeText` is applied
- **THEN** the result is the single-line string `first line\n\nsecond paragraph\n- [ ] looks like a task` (each `\n` being the literal two-character sequence)
- **AND** the result contains no raw LF or CR

- **GIVEN** the text `a` + CRLF + `b` + CR + `c`
- **WHEN** `EscapeText` is applied
- **THEN** the result is `a\nb\nc` (literal sequences) — CRLF and lone CR both normalized to LF before escaping

#### R2: Unescape semantics (lenient)
`internal/idea` SHALL export `UnescapeText(s string) string` performing a left-to-right scan: `\\` → `\`, `\n` → LF, `\` followed by any **other** character → both characters pass through verbatim (e.g. `\b` stays `\b`), and a trailing lone `\` passes through verbatim. Unrecognized escapes MUST never error.

- **GIVEN** the persisted text `C:\\new`
- **WHEN** `UnescapeText` is applied
- **THEN** the result is `C:\new` (literal backslash + `new`, no newline)

- **GIVEN** the legacy persisted text `a\b` and the legacy text `trailing\`
- **WHEN** `UnescapeText` is applied
- **THEN** the results are `a\b` and `trailing\` — verbatim pass-through

#### R3: Round-trip law
`UnescapeText(EscapeText(x))` MUST equal `x` exactly for any CR-free `x` (including backslash-heavy text such as `C:\new`, `a\\b`, trailing `\`, and text that itself looks like an idea line). For `x` containing CR, `UnescapeText(EscapeText(x))` MUST equal the CR-normalized form of `x` — CR→LF normalization is the only deliberate loss.

- **GIVEN** each of: plain text, multi-paragraph text, `C:\new`, `a\\b`, `trailing\`, text containing the literal two chars `\n`, and `- [ ] looks like a task`
- **WHEN** the text is escaped and then unescaped
- **THEN** the original text is recovered byte-for-byte

### CLI: Persistence Seam (`ParseLine`/`FormatLine`)

#### R4: FormatLine escapes on serialization
`FormatLine` MUST serialize `EscapeText(i.Text)` into the canonical `- [%s] [%s] %s: %s` shape. Every write path inherits the guarantee through this single seam: `Add`'s direct append, `SaveFile`'s normalize-on-write rebuild (which `Edit`/`Done`/`Reopen`/`Rm` flow through), `RequireSingle`'s multi-match error listing, and the `Done:`/`Removed:`/`Updated:`/`Reopened:` confirmations. A persisted idea MUST always occupy exactly one physical line and MUST match the existing single-line `lineRegex`. Existing canonical lines containing neither backslash nor newline MUST round-trip byte-identical (no churn).

- **GIVEN** an `Idea` whose `Text` contains embedded newlines
- **WHEN** `FormatLine` serializes it
- **THEN** the output is a single physical line in the canonical shape, with the text in escaped form
- **AND** `ParseLine` accepts that output and recovers the identical `Idea`

#### R5: ParseLine unescapes; in-memory text is real
`ParseLine` MUST apply `UnescapeText` to the captured text group, after the Shape B precision guard (the guard evaluates the raw on-disk text group). `Idea.Text` SHALL always hold the *real* (unescaped) text in memory — consequently `MarshalJSON` and `Match` need no change and operate on real text. `ParseLine` MUST stay pure and line-based (no date stamping, no I/O).

- **GIVEN** the on-disk line `- [ ] [a7k2] 2026-06-10: first line\n\nsecond paragraph`
- **WHEN** `ParseLine` parses it
- **THEN** the returned `Idea.Text` is `first line` + LF + LF + `second paragraph` (real newlines)

#### R6: Add and Edit normalize CR and persist one line
`Add` and `Edit` MUST normalize CR (CRLF → LF, lone CR → LF) on the incoming text before storing it in `Idea.Text`, so the in-memory `Idea` equals what round-trips from disk. `idea add` with multiline text MUST grow the backlog by exactly one physical line; `idea edit` to multiline text MUST keep the idea on exactly one physical line; `idea rm` MUST remove the whole idea with no orphaned residue. Embedded idea-looking lines MUST never parse as separate (phantom) ideas.

- **GIVEN** an empty backlog and `Add` called with `first line` + LF + LF + `second paragraph` + LF + `- [ ] looks like a task`
- **WHEN** the file is reloaded
- **THEN** the file contains exactly one content line, `LoadFile` yields exactly one idea, and that idea's `Text` equals the original multiline input
- **AND** removing that idea via `Rm` leaves no residue of the text in the file

### CLI: Display & Output

#### R7: `show` renders real newlines; JSON carries real newlines
`internal/idea` SHALL export `DisplayLine(i Idea) string` rendering the canonical `- [x] [id] date: ` prefix followed by the **unescaped** (real) text, and `cmd/idea/show.go` MUST use it for plain output so continuation lines appear below the prefix line. JSON output (`list --json`, `show --json`) SHALL carry real newlines in the `text` field via the unchanged `MarshalJSON` (JSON itself encodes them as `\n`).

- **GIVEN** a stored multiline idea
- **WHEN** `idea show <id>` runs without `--json`
- **THEN** stdout shows the prefix + first text line, with subsequent paragraphs on their own physical lines
- **AND** `idea show <id> --json` emits a `text` field whose decoded value contains real newlines

#### R8: `list` stays escaped one-line-per-idea
`idea list` (incl. `--done`, `--all`) MUST keep its shape unchanged: one physical line per idea, printed in canonical **escaped** form via `FormatLine`, preserving the line-per-record guarantee for external pipelines.

- **GIVEN** a backlog containing one multiline idea
- **WHEN** `idea list` runs
- **THEN** stdout contains exactly one idea line, with the text in escaped form

#### R9: Confirmations are escaped single-line
The `Added:` confirmation (`cmd/idea/add.go`) MUST print the escaped single-line form of the text (via `idea.EscapeText`) — never raw newlines. `Updated:` (`cmd/idea/edit.go`) inherits via `FormatLine` and MUST remain single-line; no change beyond verification.

- **GIVEN** `idea add` invoked with multiline text
- **WHEN** the command succeeds
- **THEN** stdout is exactly one line: `Added: [id] date: ` + escaped text

### CLI: Legacy Backslash Policy

#### R10: Lenient pass-through, one-time normalize-on-write
Pre-existing lines whose text contains literal backslashes MUST read verbatim (unrecognized escapes pass through, per R2). On the next mutating save, in-memory `\` re-serializes as `\\` (on-disk `a\b` → `a\\b`) — content unchanged, encoding canonicalized, consistent with the established normalize-on-write precedent. A second save MUST be byte-stable (no further churn). Accepted consequence: a legacy literal two-character `\n` inside text (e.g. `C:\new`) decodes to a real newline on read; re-saving is stable.

- **GIVEN** a legacy file containing `- [ ] [a7k2] 2025-06-15: path a\b here`
- **WHEN** the file is loaded, a mutating command saves it, and it is loaded and saved again
- **THEN** the first read yields `Text == "path a\b here"`; the first save writes `path a\\b here`; the reloaded text is still `path a\b here`; and the second save produces byte-identical file content

### Non-Goals

- Repairing already-mangled multiline entries in existing backlog files — orphaned prose is unrecognizable as idea content; cannot be auto-repaired safely (user-decided exclusion).
- Escaping non-idea (pass-through) lines — Constitution I preserves them verbatim, untouched.
- Updating `docs/specs/backlog-format.md` and `docs/memory/cli/structure.md` — deferred to the hydrate stage per the intake.

### Design Decisions

1. **Transform at the `ParseLine`/`FormatLine` seam**: escape in `FormatLine`, unescape in `ParseLine` — *Why*: single source of truth; every writer/reader (Add append, SaveFile rebuild, confirmations, error listings) inherits automatically; `Idea.Text` always holds real text so `MarshalJSON`/`Match` need no change — *Rejected*: scattering escape calls across `Add`/`Edit`/display call-sites (drift-prone).
2. **`strings.Replacer` pairs**: package-level escaper (`\`→`\\`, LF→`\n`) and unescaper (`\\`→`\`, `\n`→LF) — *Why*: stdlib, single-pass left-to-right with non-overlapping patterns; unmatched bytes copy through, which implements the lenient pass-through semantics of R2 exactly — *Rejected*: hand-rolled scanner (more code for identical semantics).
3. **Shared prefix formatter**: `FormatLine` and the new `DisplayLine` delegate to one private formatter taking the text representation — *Why*: keeps the canonical format string in one place (no duplication) — *Rejected*: duplicating the `fmt.Sprintf` shape in `cmd/show.go` (line-format knowledge would leak out of `internal/idea`, violating Constitution IV).
4. **Escape-on-write / unescape-on-display (intake Option 1)** — *Why*: file stays structurally identical (one physical line per idea); external consumers unaffected — *Rejected* (in intake): continuation-line markers (stateful parsing, contract churn) and newline flattening (lossy).

## Tasks

### Phase 1: Core Helpers

- [x] T001 Implement `normalizeCR`, `EscapeText`, `UnescapeText` (package-level `strings.Replacer` pairs + doc comments) in `src/internal/idea/idea.go` <!-- R1, R2 -->
- [x] T002 Add table-driven unit tests for `EscapeText`/`UnescapeText` plus round-trip property cases (plain, multi-paragraph, `C:\new`, `a\\b`, trailing `\`, literal `\n` text, idea-looking text, CRLF/lone-CR inputs) in `src/internal/idea/idea_test.go` <!-- R1, R2, R3 -->

### Phase 2: Seam Wiring (`internal/idea`)

- [x] T003 Wire the seam in `src/internal/idea/idea.go`: `FormatLine` escapes via a shared private prefix formatter; `ParseLine` unescapes the captured text group after the Shape B guard; add exported `DisplayLine` rendering the unescaped form <!-- R4, R5, R7 -->
- [x] T004 Normalize CR on incoming text in `Add` and `Edit` (`src/internal/idea/idea.go`) so `Idea.Text` matches what round-trips <!-- R6 -->
- [x] T005 Add table-driven tests in `src/internal/idea/idea_test.go`: FormatLine/ParseLine escape round-trip; multiline `Add` → one physical line, one parsed idea, no phantom idea, full text on reload; multiline `Edit` → single-line guarantee; `Rm` of a multiline idea leaves no orphans; CR-input normalization through `Add`; `DisplayLine` renders real newlines; legacy fixtures (lone backslash reads verbatim, doubles on first mutating save, second save byte-stable) <!-- R3, R4, R5, R6, R7, R10 -->

### Phase 3: Command Layer Integration

- [x] T006 Update `Added:` confirmation in `src/cmd/idea/add.go` to print `idea.EscapeText(i.Text)`; verify `Updated:` in `src/cmd/idea/edit.go` inherits via `FormatLine` (no code change expected) <!-- R9 -->
- [x] T007 Switch plain `show` output to `idea.DisplayLine(i)` in `src/cmd/idea/show.go` <!-- R7 -->
- [x] T008 Add end-to-end binary tests in `src/cmd/idea/main_test.go`: `add` with embedded newlines → single-line `Added:` confirmation + backlog grows by exactly one line; `list` shows one escaped line; `show` renders real newlines; `show --json` text field decodes with real newlines; multiline `edit` → single-line `Updated:` <!-- R6, R7, R8, R9 -->

### Phase 4: Polish

- [x] T009 Run `cd src && gofmt -l .` (must be clean), `go vet ./...`, and the full `go test ./...`; fix any fallout <!-- R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `EscapeText` maps `\` → `\\` and LF → `\n`, normalizes CRLF/lone CR to LF first, and its output never contains raw LF or CR
- [x] A-002 R2: `UnescapeText` maps `\\` → `\` and `\n` → LF, passes unrecognized escapes (`\b`) and a trailing lone `\` through verbatim, and never errors
- [x] A-003 R3: round-trip property tests pass — `UnescapeText(EscapeText(x)) == x` for CR-free inputs including backslash-heavy cases; CR inputs recover the CR-normalized form
- [x] A-004 R4: `FormatLine` emits the escaped text; every persisted idea occupies exactly one physical line and re-parses via `lineRegex`
- [x] A-005 R5: `ParseLine` returns real (unescaped) text in `Idea.Text`, applies the Shape B guard on the raw on-disk text, and remains pure (no date stamping, no I/O); `MarshalJSON` and `Match` are unchanged
- [x] A-006 R6: multiline `Add`/`Edit` keep the idea on one physical line with no phantom ideas; `Rm` removes the whole idea with no orphaned residue
- [x] A-007 R7: plain `show` renders the prefix plus real newlines via `DisplayLine`; `show --json`/`list --json` carry real newlines in `text`
- [x] A-008 R8: `list` output remains exactly one escaped canonical line per idea
- [x] A-009 R9: `Added:` prints the escaped single-line form; `Updated:` remains single-line via `FormatLine`

### Behavioral Correctness

- [x] A-010 R4: canonical lines containing neither backslash nor newline round-trip byte-identical — no churn introduced on unrelated content
- [x] A-011 R10: legacy lone-backslash text reads verbatim, re-serializes doubled on the first mutating save, and a second save is byte-stable

### Scenario Coverage

- [x] A-012 R6: the intake's worked example — `add` of `"first line\n\nsecond paragraph\n- [ ] looks like a task"` — persists as one physical line, `show` returns the full multi-paragraph text, and the embedded idea-looking line never parses as a separate idea

### Edge Cases & Error Handling

- [x] A-013 R1: CRLF and lone-CR inputs normalize to LF at the write seam; no raw CR reaches the file
- [x] A-014 R2: unrecognized escape sequences and a trailing lone backslash survive a load/save cycle without corruption

### Code Quality

- [x] A-015 Pattern consistency: new helpers follow existing naming/doc-comment style; tests are table-driven against real temp dirs (Constitution V); gofmt clean and `go vet` clean
- [x] A-016 No unnecessary duplication: the canonical format string lives in one private formatter shared by `FormatLine` and `DisplayLine`; no new dependencies (stdlib only)
- [x] A-017 No magic strings: escape/unescape sequences are defined once as package-level `strings.Replacer` values with explanatory comments

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Exported helper names `EscapeText`/`UnescapeText` in `internal/idea` | Intake names `idea.EscapeText` verbatim; the symmetric counterpart has one obvious name | S:85 R:90 A:90 D:90 |
| 2 | Confident | `EscapeText` performs CR normalization internally, and `Add`/`Edit` also normalize the incoming text before storing it in `Idea.Text` | Intake places normalization "alongside escaping at the write seam" and at `Add`/`Edit`; doing both keeps escape self-contained and keeps the in-memory `Idea` equal to what round-trips | S:70 R:85 A:80 D:70 |
| 3 | Confident | Escape/unescape implemented as package-level `strings.Replacer` pairs; the unescaper's copy-through-unmatched-bytes behavior implements R2's lenient pass-through exactly | Intake explicitly offers `strings.Replacer`; patterns (`\\`, `\n`) cannot both match at one position, so the single pass is deterministic | S:75 R:90 A:85 D:80 |
| 4 | Confident | New exported `DisplayLine(i Idea)` in `internal/idea` renders the unescaped `show` form, sharing a private prefix formatter with `FormatLine` | Keeps line-format knowledge inside `internal/idea` (Constitution IV); the helper name is the agent's choice and trivially renameable | S:60 R:90 A:80 D:65 |
| 5 | Confident | `ParseLine` applies `UnescapeText` after the Shape B precision guard (guard evaluates the raw on-disk text group) | Guard semantics are defined on the persisted form; unescaping cannot create or remove a leading `[`, so ordering is also behavior-neutral — on-disk-first is the conceptually correct reading | S:65 R:85 A:85 D:75 |
| 6 | Certain | `Done:`/`Removed:`/`Reopened:` confirmations need no change — they already print `FormatLine` and inherit escaping | Verified in `cmd/idea/done.go`, `rm.go`, `reopen.go`; intake names only `Added:`/`Updated:` as needing attention | S:85 R:95 A:95 D:90 |

6 assumptions (2 certain, 4 confident, 0 tentative).
