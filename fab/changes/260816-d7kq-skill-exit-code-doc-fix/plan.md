# Plan: Skill Bundle Exit-Code Doc Fix

**Change**: 260816-d7kq-skill-exit-code-doc-fix
**Intake**: `intake.md`

## Requirements

### Docs: Skill bundle exit-code contract

#### R1: Bundle documents the shipped 0/1/2 exit-code convention
The exit-code bullet in `docs/site/skill.md` § Output & exit-code contracts MUST document idea's actual shipped behavior: `0` success, `1` operational failure, `2` usage error (the toolkit convention adopted tree-wide by 260717-xvsj). It MUST name the usage-error class accurately (unknown flags, wrong argument counts, `shell-init` missing/unsupported shell, the `--system`+`--main` conflict) and the operational class accurately (`rm`/`prune` consent refusals, no-match/ambiguous queries, `fmt --check` on a non-canonical file, I/O failures), per `docs/memory/cli/structure.md` § Exit-code convention. It MUST NOT retain the stale claims that usage errors exit `1`, that only `shell-init` exits `2`, or that the convention is "not yet implemented (deferred, backlog `[xvsj]`)". No other bundle content changes.

- **GIVEN** an agent reading `idea skill` (or `docs/site/skill.md`) to decide how to branch on exit codes
- **WHEN** it reads the exit-code bullet
- **THEN** the bullet states `0` success / `1` operational failure / `2` usage error with accurate class membership
- **AND** no reference to a deferred `[xvsj]` backlog item or "not yet implemented" remains

#### R2: Embed copy re-synced, drift guard green
After the canonical edit, `scripts/sync-skill.sh` MUST be re-run so `src/cmd/idea/skill/skill.md` is byte-identical to `docs/site/skill.md`, and the refreshed copy MUST be committed alongside. `go test ./...` MUST pass, including the drift guard `TestSkillEmbedMatchesCanonical`.

- **GIVEN** the edited `docs/site/skill.md`
- **WHEN** `scripts/sync-skill.sh` runs and `go test ./...` executes
- **THEN** the drift-guard test passes (the two files are byte-identical) and all other tests remain green

#### R3: Line budget preserved
The edited bundle MUST stay within the 150-line hard budget (`TestSkill_LineBudget`).

- **GIVEN** the edited `docs/site/skill.md`
- **WHEN** the line-budget test runs
- **THEN** it passes (bundle is ≤150 lines; pre-edit it is 95, the replacement adds ~2)

### Non-Goals

- No Go source changes — the exit-code implementation shipped at 260717-xvsj is correct; only the doc drifted.
- No memory or spec edits — `docs/memory/cli/skill.md` already requires the correct contract; this change conforms the artifact to memory.
- No other bundle edits — the `fmt --check` exits-1 gotcha line is correct (operational class) and stays.

## Tasks

### Phase 1: Core Implementation

- [x] T001 Replace the stale exit-code bullet in `docs/site/skill.md` (§ Output & exit-code contracts, lines 72–75) with the shipped 0/1/2 convention text from intake § What Changes, keeping accurate class membership <!-- R1 -->
- [x] T002 Run `scripts/sync-skill.sh` to refresh the embed copy `src/cmd/idea/skill/skill.md` <!-- R2 -->
- [x] T003 Run `go test ./...` — verify drift guard (`TestSkillEmbedMatchesCanonical`), line budget (`TestSkill_LineBudget`), and the rest of the suite pass <!-- R2 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: The exit-code bullet states `0` success / `1` operational failure / `2` usage error, with usage class (flags, arg counts, shell-init shell, `--system`+`--main`) and operational class (consent refusals, no-match/ambiguous queries, `fmt --check`, I/O) matching `docs/memory/cli/structure.md` § Exit-code convention
- [x] A-002 R2: `src/cmd/idea/skill/skill.md` is byte-identical to `docs/site/skill.md`

### Behavioral Correctness

- [x] A-003 R1: The stale claims are gone — no "usage/arg errors exit 1", no "Only `shell-init` exits 2", no "not yet implemented (deferred, backlog `[xvsj]`)" text remains anywhere in the bundle

### Scenario Coverage

- [x] A-004 R2: `go test ./...` passes with the refreshed embed copy
- [x] A-005 R3: `TestSkill_LineBudget` passes (bundle ≤150 lines)

### Edge Cases & Error Handling

- [x] A-006 R1: Bullet content is verifiable against the real binary (usage error → exit 2, operational error → exit 1) — spot-checked at intake, re-checkable via the repo build

### Code Quality

- [x] A-007 Pattern consistency: The replacement bullet matches the bundle's existing style (bold lead-in, backticked literals, wrapped prose)
- [x] A-008 No unnecessary duplication: The bullet documents the contract without duplicating `docs/memory/cli/structure.md`'s implementation detail (seams, sentinels)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use the intake's proposed replacement bullet wording (lightly polishable), keeping the agent-facing note that branching on `2` is supported | Intake fixed the factual content; wording latitude was explicitly granted there | S:85 R:95 A:90 D:90 |
| 2 | Certain | Verification is the existing test suite only — no new tests | Drift guard + budget guard already cover exactly this change's failure modes | S:85 R:95 A:95 D:90 |

2 assumptions (2 certain, 0 confident, 0 tentative).
