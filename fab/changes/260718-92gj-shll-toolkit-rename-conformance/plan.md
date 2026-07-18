# Plan: shll Toolkit Name Conformance

**Change**: 260718-92gj-shll-toolkit-rename-conformance
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake § What Changes. Pure prose/markdown conformance sweep,
     no behavior change. Requirements use RFC 2119 keywords. -->

### README: Toolkit Naming

#### R1: Canonical toolkit blockquote
The README's toolkit blockquote MUST be byte-identical to the readme-extraction standard's canonical line, and the mandated head order (H1 → blockquote → badges) MUST be preserved.

- **GIVEN** `README.md` line 3 carries the pre-rename wording `> Part of [@sahil87's open source toolkit](https://shll.ai) — see all projects there.`
- **WHEN** the conformance sweep is applied
- **THEN** line 3 reads exactly `> Part of the [shll toolkit](https://shll.ai) — see all projects there.`
- **AND** line 1 (`# idea` H1), the blank line 2, the blockquote (line 3), and the badges (line 5) remain in that order, unchanged apart from the blockquote text

#### R2: README prose old-name replacement
Prose occurrences of the old toolkit name in `README.md` MUST be renamed to the new name, leaving link targets and identifiers untouched.

- **GIVEN** `README.md` line 15 reads `…To install the entire sahil87 toolkit instead:` and line 54 reads `> 💡 Have other sahil87 tools? …`
- **WHEN** the sweep is applied
- **THEN** line 15 reads `…To install the entire shll toolkit instead:` and line 54 reads `> 💡 Have other shll tools? …`
- **AND** every other `sahil87` occurrence in `README.md` (badge URLs on line 5; `github.com/sahil87/…` doc links on lines 54/99/128/136) stays byte-identical

### Docs Site: Toolkit Naming

#### R3: install.md prose old-name replacement
The prose occurrence of the old toolkit name in `docs/site/install.md` MUST be renamed, leaving surrounding lines and identifiers untouched.

- **GIVEN** `docs/site/install.md` line 66 reads `> Have other sahil87 tools? shll shell-install wires up the shell integrations`
- **WHEN** the sweep is applied
- **THEN** line 66 reads `> Have other shll tools? shll shell-install wires up the shell integrations`
- **AND** lines 67–68 (including the `github.com/sahil87/shll#…` link) and the `brew install sahil87/tap/idea` formula (line 10) stay byte-identical

### Constitution: Toolkit Standards Article

#### R4: Constitution article wording + governance line
The Toolkit Standards article MUST use the new toolkit name in its opening sentence, with no other change to the article, and the governance line's `Last Amended` field MUST read today's date with `Version` unchanged.

- **GIVEN** `fab/project/constitution.md` line 47 opens `This tool is part of the sahil87 toolkit and MUST conform…` and line 51 reads `**Version**: 1.1.0 | **Ratified**: 2026-05-03 | **Last Amended**: 2026-07-18`
- **WHEN** the sweep is applied
- **THEN** line 47 opens `This tool is part of the shll toolkit and MUST conform…`
- **AND** the `sahil87/shll` canonical-source reference and the `https://shll.ai` URL later in line 47 stay verbatim
- **AND** line 51 has `Last Amended` = `2026-07-18` (equal to its current value — a no-op) and `Version` stays `1.1.0`

### Non-Goals

- CLI help text / user-visible string edits, test-golden updates, `scripts/sync-skill.sh` re-run, `schema_version` bump — grep confirms zero old-name prose in `src/**`, `scripts/**`, and `docs/site/skill.md`, so all conditional clauses in the task resolve to no-ops.
- Identifiers: `sahil87/tap` formula names, `github.com/sahil87/…` and `raw.githubusercontent.com/sahil87/…` URLs, badge URLs, `sahil87/shll` canonical-source reference, GitHub-owner constants in code.
- `fab/changes/**` historical archives, and `docs/memory/**` prose (corrected at hydrate, not apply).

## Tasks

### Phase 1: Prose Sweep

- [x] T001 [P] Replace `README.md` line 3 blockquote with byte-identical `> Part of the [shll toolkit](https://shll.ai) — see all projects there.`, preserving the H1 → blockquote → badges head order <!-- R1 -->
- [x] T002 [P] In `README.md` line 15, replace `the entire sahil87 toolkit` → `the entire shll toolkit`; in line 54, replace `Have other sahil87 tools?` → `Have other shll tools?` (rest of both lines, incl. link target, byte-identical) <!-- R2 -->
- [x] T003 [P] In `docs/site/install.md` line 66, replace `Have other sahil87 tools?` → `Have other shll tools?` (lines 67–68 and the line-10 formula untouched) <!-- R3 -->
- [x] T004 [P] In `fab/project/constitution.md` line 47, replace `part of the sahil87 toolkit` → `part of the shll toolkit` (nothing else in the article changes); verify line 51 `Last Amended` reads `2026-07-18` and `Version` stays `1.1.0` <!-- R4 -->

### Phase 2: Verification

- [x] T005 Run `go test ./...` from `src/` — must be green (drift guard, goldens, help-dump unaffected by construction) <!-- R1 -->
- [x] T006 Verify byte-exactness of the new blockquote (`grep -Fx '> Part of the [shll toolkit](https://shll.ai) — see all projects there.' README.md`) and that `grep -rn 'sahil87 toolkit\|sahil87 tool' README.md docs/site/ fab/project/` returns zero hits <!-- R2 -->

## Execution Order

- T001–T004 are independent single-line edits across four files; parallelizable.
- T005 and T006 run after T001–T004 complete.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `README.md` line 3 is byte-identical to `> Part of the [shll toolkit](https://shll.ai) — see all projects there.` and the H1 → blockquote → badges order is intact
- [x] A-002 R2: `README.md` lines 15 and 54 use the new name in prose; all `github.com/sahil87/…` links and badge URLs in the file are unchanged
- [x] A-003 R3: `docs/site/install.md` line 66 uses the new name in prose; lines 67–68 and the line-10 formula are unchanged
- [x] A-004 R4: `fab/project/constitution.md` line 47 opens with `part of the shll toolkit`, the `sahil87/shll` canonical reference is intact, and line 51 reads `Last Amended: 2026-07-18` with `Version: 1.1.0`

### Behavioral Correctness

- [x] A-005 R2: `grep -rn 'sahil87 toolkit\|sahil87 tool' README.md docs/site/ fab/project/` returns zero hits after the edits

### Scenario Coverage

- [x] A-006 R1: `grep -Fx '> Part of the [shll toolkit](https://shll.ai) — see all projects there.' README.md` matches exactly one line

### Edge Cases & Error Handling

- [x] A-007 R1: No identifier was altered — `sahil87/tap`, `github.com/sahil87/…`, `raw.githubusercontent.com/sahil87/…`, badge URLs, and the `sahil87/shll` canonical reference remain byte-identical; `fab/changes/**` and `docs/memory/**` untouched

### Code Quality

- [x] A-008 Pattern consistency: Edited prose follows the surrounding markdown style (blockquote form, punctuation, em-dash) of each file
- [x] A-009 No unnecessary duplication: Edits reuse existing lines in place — no new lines, files, or scaffolding introduced

### Build Integrity

- [x] A-010 R1: `go test ./...` from `src/` passes green (no source/test/golden changes; drift guard and help-dump unaffected)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Replace the README blockquote with `> Part of the [shll toolkit](https://shll.ai) — see all projects there.` byte-identically, preserving the H1 → blockquote → badges head order | Line given verbatim in the task AND verified live against `shll standards readme-extraction` at intake time (precondition gate passed) | S:95 R:90 A:100 D:100 |
| 2 | Certain | Prose sweep is exactly four line edits (README lines 3/15/54, install.md line 66) plus the constitution wording; every other `sahil87` occurrence is an identifier or `fab/changes/` archive and stays untouched | Enumerated by repo-wide grep, re-confirmed live at apply time; task's exclusion list maps 1:1 onto the remaining hits | S:90 R:85 A:95 D:95 |
| 3 | Certain | No CLI help-text/user-visible-string edits, no test-golden updates, no `sync-skill.sh` re-run, no `schema_version` concern | Grep shows zero old-name occurrences in `src/**`/`scripts/**` and none in `docs/site/skill.md`; the task's conditional clauses resolve to no-ops — `go test ./...` is the backstop | S:85 R:90 A:95 D:95 |
| 4 | Confident | Governance line: set `Last Amended` to today (2026-07-18 — equal to current value, a no-op) and leave `Version` at 1.1.0 | Task directs bumping `Last Amended` only and says "Nothing else in the article changes"; no version bump for a cosmetic wording fix; trivially reversible | S:70 R:95 A:75 D:70 |
| 5 | Confident | Old-name prose in `docs/memory/` (cli/structure, cli/skill) is corrected at hydrate, not during apply | Task's sweep list doesn't name memory; memory maintenance is hydrate's responsibility; leaving the old name there would be documented drift caught at hydrate | S:60 R:90 A:80 D:70 |

5 assumptions (3 certain, 2 confident, 0 tentative).
