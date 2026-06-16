# Plan: TTY-Aware Output Rendering

**Change**: 260613-kfcl-tty-aware-output-rendering
**Intake**: `intake.md`

## Requirements

<!-- Derived from the intake's four bundled features (A, B, D, E). All display
     behavior is TTY-gated; piped output stays full/canonical (Constitution VI).
     Logic lives in internal/idea (Constitution IV); cmd/ only wires flags and
     picks the rendering mode. -->

### Terminal Seam: TTY / Width / Color Detection

#### R1: TTY detection
The package SHALL expose `IsTTY(f *os.File) bool` returning whether the given file is a terminal, via `golang.org/x/term.IsTerminal`.

- **GIVEN** stdout connected to an interactive terminal
- **WHEN** `IsTTY(os.Stdout)` is called
- **THEN** it returns true
- **AND** when stdout is a pipe or regular file it returns false

#### R2: Terminal width resolution with fallback
The package SHALL expose `TermWidth(f *os.File) int` returning the terminal column count: `golang.org/x/term.GetSize` first; if that fails or yields a non-positive width, honor `$COLUMNS` when it parses to a positive integer; otherwise fall back to 80.

- **GIVEN** `GetSize` cannot determine width (returns error or width ≤ 0)
- **WHEN** `TermWidth` is called and `$COLUMNS` is set to a positive integer
- **THEN** the `$COLUMNS` value is returned
- **AND** when `$COLUMNS` is unset or invalid, 80 is returned

#### R3: Color helpers honoring NO_COLOR
The package SHALL expose styling helpers that dim the `[id] date:` prefix (ANSI faint `\033[2m`) and color a done `[x]` checkbox green, and a predicate `UseColor(f *os.File) bool` that is true only when `f` is a TTY AND `NO_COLOR` is unset. Color codes SHALL be applied only after width-based truncation so width math counts visible runes, never escape bytes.

- **GIVEN** stdout is a TTY and `NO_COLOR` is unset
- **WHEN** `UseColor(os.Stdout)` is consulted
- **THEN** it returns true and styling helpers wrap text in ANSI codes
- **AND** when `NO_COLOR` is set (to any value, including empty) or stdout is not a TTY, `UseColor` returns false and helpers return their input unchanged

### List Rendering: Truncation, --full, and Display

#### R4: Rune-safe text truncation that never clips the prefix
The package SHALL expose a display-line builder that, given an idea and a target width, renders the canonical escaped line but truncates only the **text portion** to fit the width, appending `…` (U+2026) when clipped. The `- [x] [id] date: ` prefix SHALL NEVER be truncated. Truncation SHALL be rune-safe (operate on `[]rune`, never byte slices). When the escaped text contains a literal `\n` escape sequence (multiline idea), the text SHALL be truncated at the first such newline regardless of width, with `…` appended.

- **GIVEN** an idea whose escaped text exceeds the available width
- **WHEN** the display line is built at that width
- **THEN** the prefix is intact and the text is clipped to fit with a trailing `…`
- **AND** truncation falls on a rune boundary (no broken multibyte sequence)
- **AND** a multiline idea (escaped text containing `\n`) is clipped at the first `\n` with `…` even if it would otherwise fit

#### R5: `--full` flag and TTY gate for `idea list`/`ls`
`idea list` (and its `ls` alias) SHALL accept a `--full` boolean flag. On a TTY with `--full` unset, each listed idea SHALL be truncated per R4 and colored per R3. With `--full` set on a TTY, full text SHALL be emitted (colored per R3, no truncation). When stdout is NOT a TTY, full canonical `FormatLine` output SHALL be emitted regardless of `--full` (no truncation, no color), preserving the pipe contract. `--json` output SHALL be unchanged in all cases.

- **GIVEN** stdout is a TTY and the list contains an over-wide idea
- **WHEN** `idea ls` runs without `--full`
- **THEN** the idea's text is truncated with `…` and the prefix is dimmed
- **AND** with `--full` the full text is shown (no `…`)
- **AND** when stdout is piped, output equals the current `FormatLine` output (no truncation, no ANSI) for both `--full` and default

#### R6: Optional `[id...]` positional filter for `idea list`/`ls`
`idea list`/`ls` SHALL accept zero-or-more positional ID arguments. Each argument MUST be a 4-char lowercase alphanumeric ID (validated via `idea.ValidateID`); a malformed argument is a usage error. When IDs are supplied, only ideas whose ID is in the set are listed, still respecting `--sort`/`--reverse`/truncation/color and the active filter. Unknown (well-formed but absent) IDs SHALL produce a stderr warning naming each missing ID, and the remaining matched ideas SHALL still be listed. When all supplied IDs are unknown, the warning is emitted and the normal empty-result message path applies.

- **GIVEN** a backlog containing ideas `oibl` and `rkx4`
- **WHEN** `idea ls oibl rkx4` runs
- **THEN** only those two ideas are listed (filter/sort/reverse still applied)
- **AND** `idea ls oibl zzzz` warns on stderr that `zzzz` was not found and lists `oibl`
- **AND** `idea ls BADID` (malformed) fails with a usage/validation error

### Prune Rendering: Count Header, Color, Interactive Confirm

#### R7: Dry-run count summary header on stderr
`idea prune` without `--force` SHALL, before listing prunable ideas, print to **stderr**: `N done idea(s) would be pruned`. stdout SHALL still carry exactly the removable lines (truncated/colored on a TTY per R4/R3; full canonical `FormatLine` when piped). The header is printed before the list so it is the first thing a human sees.

- **GIVEN** two done ideas exist and stdout is a pipe
- **WHEN** `idea prune` runs
- **THEN** stderr begins with `2 done idea(s) would be pruned` and stdout carries exactly the two `FormatLine` lines

#### R8: Interactive confirm gated on TTY && !force
`idea prune` SHALL prompt for confirmation only when stdout is a TTY AND `--force` is unset. The prompt `Prune N done idea(s)? [y/N] ` SHALL be written to stderr after the count header and list; a line is read from stdin and deletion proceeds only on `y`/`yes` (case-insensitive, trimmed); any other input aborts with no change and an abort message on stderr. When stdout is a TTY in interactive mode, the trailing `Re-run with --force to confirm.` hint SHALL be dropped (the prompt replaces it). When stdout is NOT a TTY and `--force` is unset, the command SHALL NOT prompt and SHALL fall back to the dry run (removable lines on stdout + the trailing `Re-run with --force to confirm.` hint on stderr, in addition to the R7 header). `--force` (TTY or not) SHALL delete immediately with the existing count-only output.

- **GIVEN** stdout is a TTY, `--force` unset, two done ideas
- **WHEN** the prompt fires and the user enters `y`
- **THEN** the two done ideas are removed (same as `--force`) and `Pruned 2 done idea(s).` prints
- **AND** any non-`y`/`yes` answer aborts with no file change and an abort message
- **AND** when stdout is piped and `--force` unset, no prompt is shown and the trailing hint is emitted (current dry-run fallback)

#### R9: `--full` on `idea prune` for symmetry
`idea prune` SHALL accept a `--full` boolean flag with the same TTY-gated semantics as R5 applied to its dry-run / pre-confirm listing.

- **GIVEN** stdout is a TTY with an over-wide done idea
- **WHEN** `idea prune --full` runs
- **THEN** the listed done idea shows full text (no `…`); piped output is unaffected by `--full`

### Non-Goals

- Section grouping via `## H2` headers (item G) — deferred to backlog `ykwp`.
- Paging / `$PAGER` (item C) — decided against.
- Wide-glyph (CJK/emoji) display-width awareness — rune-count against columns is the floor.
- Changing `FormatLine`/`DisplayLine`, the parser, the backlog format, or the `--json` schema.

### Design Decisions

1. **New `internal/idea/term.go` seam**: TTY/width/color/truncation logic lives in `internal/idea`; `cmd/` only asks the seam for the decision. — *Why*: Constitution IV. — *Rejected*: inline logic in `cmd/` (untestable without a PTY, violates IV).
2. **Width injectable for tests**: the display-line builder takes width as a parameter (no real PTY in tests). — *Why*: Constitution V (deterministic, real-temp-dir tests). — *Rejected*: allocating a PTY (flaky, platform-dependent).
3. **Color applied after truncation**: truncate plain text, then wrap with ANSI. — *Why*: width math must count visible runes, not escape bytes (intake open question).
4. **`golang.org/x/term`**: first non-stdlib/cobra direct dep. — *Why*: stdlib has no width primitive; recorded in intake, Constitution Dependency Discipline. — *Rejected*: `os.Stat` char-device check + `$COLUMNS`/`tput` hacks (no clean width).

## Tasks

### Phase 1: Setup

- [x] T001 Add `golang.org/x/term` as a direct dependency: `cd src && go get golang.org/x/term`, then `go mod tidy` (after T002 imports it so tidy keeps it); pin the `go` directive at 1.22 (no incidental toolchain bump) <!-- R1 -->

### Phase 2: Core Implementation (internal/idea seam)

- [x] T002 Create `src/internal/idea/term.go` with `IsTTY(f *os.File) bool` (wraps `term.IsTerminal`) and `TermWidth(f *os.File) int` (`term.GetSize` → `$COLUMNS` → 80) <!-- R1 R2 -->
- [x] T003 Add color helpers to `src/internal/idea/term.go`: `UseColor(f *os.File) bool` (TTY && NO_COLOR unset), `dimPrefix(s string) string` (ANSI faint), `greenCheck(s string) string`; package-level ANSI consts <!-- R3 -->
- [x] T004 Add the rune-safe display-line builder to `src/internal/idea/term.go`: `DisplayListLine(i Idea, width int, full, color bool) string` — builds the escaped canonical line, truncates only the text portion at the first `\n` escape or to width with `…`, never clips the prefix, applies color after truncation <!-- R4 -->

### Phase 3: Integration & Edge Cases (cmd wiring)

- [x] T005 Wire `src/cmd/idea/list.go`: add `--full` flag; change `Args` to a validator allowing zero-or-more 4-char IDs; filter ideas by the supplied ID set with a stderr warning for unknown IDs; render each idea via `idea.DisplayListLine` using `IsTTY`/`TermWidth`/`UseColor(os.Stdout)`; `--json` and non-TTY paths unchanged <!-- R5 R6 -->
- [x] T006 Wire `src/cmd/idea/prune.go`: add `--full` flag; print the R7 stderr count header before the listing; render dry-run/pre-confirm lines via `idea.DisplayListLine`; implement the R8 TTY&&!force interactive `[y/N]` prompt (read from `cmd.InOrStdin()`, prompt on stderr) calling `idea.Prune(path, true)` on confirm; drop the trailing hint in TTY mode, keep it in the non-TTY no-force fallback; `--force` path unchanged except header <!-- R7 R8 R9 -->

### Phase 4: Tests

- [x] T007 [P] Add `src/internal/idea/term_test.go`: table-driven tests for `TermWidth` fallback (GetSize-fail injected via the non-TTY path + `$COLUMNS` set/unset/invalid), `UseColor`/color helpers honoring `NO_COLOR`, and `DisplayListLine` (multibyte rune-safety, prefix-never-truncated, ellipsis presence, multiline-at-first-newline, `full` bypasses truncation, color applied after truncation) <!-- R2 R3 R4 -->
- [x] T008 [P] Extend `src/cmd/idea/main_test.go`: `ls [id...]` filter incl. unknown-id stderr warning and malformed-id error; prune count-header text and the non-TTY decision-matrix rows (No/No fallback hint, No/Yes immediate delete); assert piped output stays canonical (no ANSI, no truncation) <!-- R5 R6 R7 R8 R9 -->

### Phase 5: Review Rework

- [x] T009 Extract the duplicated TTY-aware render loop (`list.go` RunE branch + `prune.go` `printPruneLines`) into one shared `cmd/idea/output.go` helper `printIdeaLines(out io.Writer, ideas []idea.Idea, full bool)`; call it from both sites; remove `printPruneLines` (review should-fix #1) <!-- R5 R7 R9 -->
- [x] T010 Add in-process tests for the interactive confirm (review should-fix #2): `TestConfirmPrune` (y/yes/Y/YES/spaces confirm; n/no/bare-enter/EOF/garbage abort) and `TestPrune_ConfirmedDeleteAndAbort` (y/yes delete like `--force`; n/EOF leave the backlog byte-identical), in `src/cmd/idea/main_test.go` <!-- R8 -->

## Execution Order

- T002 blocks T003, T004 (same file, shared package decls)
- T002–T004 block T005, T006 (cmd wiring consumes the seam)
- T007, T008 follow their respective implementation tasks; [P] relative to each other
- T009, T010 are review-rework follow-ups (post-implementation)

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `IsTTY` correctly reports terminal vs. non-terminal for a given `*os.File`
- [ ] A-002 R2: `TermWidth` returns `GetSize` width when available, else `$COLUMNS`, else 80
- [ ] A-003 R3: color helpers dim the prefix and green the `[x]`; `UseColor` is true only on a TTY with `NO_COLOR` unset
- [ ] A-004 R4: the display-line builder truncates only the text portion rune-safely with `…`, never the prefix
- [ ] A-005 R5: `idea ls --full` shows full text on a TTY; default truncates; piped output is unchanged canonical
- [ ] A-006 R6: `idea ls oibl rkx4` lists only those ideas; unknown IDs warn on stderr; malformed IDs error
- [ ] A-007 R7: `idea prune` dry run prints the `N done idea(s) would be pruned` header on stderr with removable lines on stdout
- [ ] A-008 R8: prune prompts only when TTY && !force; non-TTY no-force falls back to dry run with the trailing hint; `--force` deletes immediately. Confirm logic and confirmed-delete are test-backed by `TestConfirmPrune` + `TestPrune_ConfirmedDeleteAndAbort`
- [ ] A-009 R9: `idea prune --full` shows full done-idea text on a TTY; piped output unaffected

### Behavioral Correctness

- [ ] A-010 R5: a piped `idea ls` / `idea ls --full` emits byte-identical output to today's `FormatLine` listing (no ANSI, no `…`)
- [ ] A-011 R8: an aborted interactive prune (`n`/EOF answer) leaves the backlog file byte-identical, proven by `TestPrune_ConfirmedDeleteAndAbort`

### Scenario Coverage

- [ ] A-012 R4: multiline idea (escaped `\n` in text) renders as one physical row truncated at the first newline with `…`
- [ ] A-013 R6: an unknown-ID warning is exercised by a test asserting the stderr message and the listed survivors

### Edge Cases & Error Handling

- [ ] A-014 R4: truncation on multibyte (non-ASCII) text falls on a rune boundary (no broken UTF-8), covered by a test
- [ ] A-015 R2: `TermWidth` handles `$COLUMNS` unset and invalid (`$COLUMNS=abc`) by returning 80

### Code Quality

- [ ] A-016 Pattern consistency: new code follows the naming, error-handling, and cobra-factory patterns of surrounding `cmd/idea` and `internal/idea` files
- [ ] A-017 No unnecessary duplication: reuses `FormatLine`/`EscapeText`/`ValidateID` and the existing `printBackfillNotice`/`resolveFile` helpers rather than reimplementing them; the TTY-aware render loop is a single shared `printIdeaLines` helper (`cmd/idea/output.go`) called by both `list` and `prune` (no duplicated render path)
- [ ] A-018 Readability over cleverness: the truncation helper is straightforward `[]rune` slicing, not byte arithmetic (code-quality.md Principles)
- [ ] A-019 No magic strings/numbers: the 80-col fallback and ANSI codes are named constants (code-quality.md Anti-Patterns)
- [ ] A-020 Logic placement: all TTY/width/color/truncation logic lives in `internal/idea`; `cmd/` only wires flags and picks the rendering mode (Constitution IV)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New `internal/idea/term.go` holds TTY/width/color/truncation; `cmd/` only wires flags | Constitution IV mandate; intake Design decision | S:92 R:80 A:95 D:88 |
| 2 | Certain | Width is a parameter to the display-line builder so tests inject it (no real PTY) | Constitution V; intake explicitly requires injectable seam | S:90 R:82 A:95 D:90 |
| 3 | Certain | `TermWidth` order: `GetSize` → `$COLUMNS` → 80 | Intake assumption #12, explicit | S:85 R:85 A:90 D:85 |
| 4 | Certain | Color applied after truncation so width counts visible runes | Intake open-question resolution, explicit | S:88 R:80 A:92 D:88 |
| 5 | Confident | Count header text: `N done idea(s) would be pruned` (drop the trailing `— re-run...` clause from the header since the interactive prompt / fallback hint carries the call-to-action) | Intake shows `N done idea(s) would be pruned — re-run with --force to confirm.` but assumption #13 drops the force-hint in TTY mode where the prompt fires; splitting the header (count) from the action (prompt or trailing hint) avoids a contradictory "re-run with --force" line right above a `[y/N]` prompt. Reversible wording choice. | S:70 R:88 A:78 D:72 |
| 6 | Confident | Malformed `ls` ID args (not `[a-z0-9]{4}`) are a usage error via `ValidateID`; only well-formed-but-absent IDs get the warn-and-list-rest treatment | Intake assumption #11 covers "unknown IDs" (warn+skip); malformed vs. unknown is a natural split — a typo'd long string is a usage mistake, a 4-char absent ID is "not found". Reversible. | S:68 R:85 A:78 D:70 |
| 7 | Confident | Add `--full` to both `ls` and `prune` | Intake assumption #13 (symmetry, cheap) | S:72 R:85 A:80 D:75 |
| 8 | Confident | `NO_COLOR` set to ANY value (incl. empty string) disables color, per the NO_COLOR spec (presence, not truthiness) | Standard NO_COLOR convention; intake says "unset" gate. Reversible. | S:78 R:85 A:82 D:80 |

8 assumptions (4 certain, 4 confident, 0 tentative).
