# Plan: Add ls Alias for list Subcommand

**Change**: 260610-04rt-add-ls-alias
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

### CLI: list alias routing

#### R1: `ls` is a cobra alias of `list`
The `cobra.Command` literal in `listCmd()` (`src/cmd/idea/list.go`) SHALL declare `Aliases: []string{"ls"}`. `idea ls` MUST behave identically to `idea list` — same flags (`--all/-a`, `--done`, `--json`, `--sort`, `--reverse`), same inherited persistent flags (`--file`, `--main`), same output. The `list` command's own behavior, output, and JSON schema MUST be otherwise unchanged.

- **GIVEN** a backlog with open ideas
- **WHEN** the user runs `idea ls`
- **THEN** the open ideas are listed exactly as `idea list` would print them

- **GIVEN** any backlog state
- **WHEN** the user runs `idea ls --json`
- **THEN** the structured JSON records (id, date, status, text) are emitted exactly as `idea list --json` would

#### R2: `idea ls` never reaches the bare-text add shorthand
Because cobra resolves command aliases before the root `RunE` fallback fires, `idea ls` MUST NOT append an idea to the backlog. The backlog file SHALL be byte-identical before and after the invocation.

- **GIVEN** a backlog file with known content
- **WHEN** the user runs `idea ls`
- **THEN** no new idea is appended — the backlog file is unchanged

#### R3: Bare-text shorthand preserved for non-alias words
Bare text whose first word is not a registered subcommand or alias (e.g. `idea lsx some text`) SHALL still route to the root bare-text shorthand and append an idea, exactly as before.

- **GIVEN** a backlog file
- **WHEN** the user runs `idea lsx some text`
- **THEN** an idea with text "lsx some text" is appended via the add shorthand

### Non-Goals

- No other aliases — `remove`/`delete` (rm), `upgrade` (update), `cat` (show), `undo` (reopen) were explicitly rejected during intake discussion (each would shadow plausible bare-text idea prose, or reserve wrong semantics)
- No `aliases` field in the help-dump JSON schema (`src/cmd/idea/help_dump.go`) — the schema is a frozen public contract (Constitution VI); cobra's rendered help shows the alias automatically
- No docs/site rendering or release pipeline changes

### Design Decisions

1. **Use cobra's native `Aliases` field**: declare `Aliases: []string{"ls"}` on the `listCmd()` command literal — *Why*: cobra-idiomatic (Constitution III), resolves before the root bare-text fallback with zero custom routing logic, and surfaces automatically in generated help — *Rejected*: intercepting `ls` in the root `RunE` (custom routing logic on root violates Constitution III's "no behavior beyond delegation")

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add `Aliases: []string{"ls"}` to the `cobra.Command` literal in `listCmd()` (`src/cmd/idea/list.go`); gofmt the file <!-- R1 -->
- [x] T002 Add a table-driven routing test to `src/cmd/idea/main_test.go` (reusing `buildBinary`, `setupGitRepo`, `writeRepoBacklog`, `runSplit`) with cases asserting `idea ls` (and `idea ls --json`) stdout is byte-identical to `idea list` (resp. `idea list --json`) on a seeded backlog, and that the backlog file is unchanged after the invocation <!-- R1, R2 -->

### Phase 3: Integration & Edge Cases

- [x] T003 Extend the same routing table with a non-alias bare-text case (`idea lsx some text`) asserting the add shorthand fires (stdout has `Added: [` and the backlog gains the idea line) <!-- R3 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `listCmd()` registers `ls` via the `Aliases` field; `idea ls` produces byte-identical stdout to `idea list` on the same backlog
- [x] A-002 R2: After running `idea ls`, the backlog file is byte-identical to its prior content (no junk "ls" idea appended)
- [x] A-003 R3: `idea lsx some text` (non-alias first word) still appends an idea via the bare-text add shorthand

### Behavioral Correctness

- [x] A-004 R2: The literal invocation `idea ls` no longer triggers the root bare-text fallback (it previously appended an idea with text "ls")

### Scenario Coverage

- [x] A-005 R1: A table-driven routing test in `src/cmd/idea/main_test.go` covers the `ls` → `list` case (Constitution V)
- [x] A-006 R3: The same routing table covers the non-alias bare-text → add case

### Edge Cases & Error Handling

- [x] A-007 R1: `idea ls --json` emits the same structured JSON as `idea list --json`

### Code Quality

- [x] A-008 Pattern consistency: the alias is declared in the `cobra.Command` literal per Constitution III; no custom routing logic added anywhere
- [x] A-009 No unnecessary duplication: new tests reuse the existing `main_test.go` helpers instead of reimplementing setup
- [x] A-010 Formatting/vet clean: touched files pass `gofmt -l` (no output) and `go vet ./...` passes

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Routing tests use the built-binary subprocess pattern (`buildBinary` + `setupGitRepo`) already established in `main_test.go`, not in-process `newRootCmd()` | Pattern extraction: every existing routing test in `main_test.go` builds the binary and execs it; subprocess routing is the seam that exercises cobra's real arg dispatch | S:90 R:90 A:95 D:90 |
| 2 | Confident | Non-alias bare-text case uses `lsx some text` as the probe input | Intake suggested `idea lsx some text` or `idea buy milk` as examples; `lsx` is the stronger probe since it shares the `ls` prefix and proves prefix-matching does not over-trigger | S:80 R:90 A:85 D:80 |
| 3 | Confident | Output equivalence is asserted on stdout only (via `runSplit`), comparing `ls` against a `list` run on the same repo state | stderr is reserved for advisory notices (backfill); stdout is the machine-parseable contract per Constitution VI, so it is the correct equivalence surface | S:75 R:90 A:85 D:85 |

3 assumptions (1 certain, 2 confident, 0 tentative).
