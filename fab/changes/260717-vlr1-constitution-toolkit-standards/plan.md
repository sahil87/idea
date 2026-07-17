# Plan: Constitution Amendment — Bind to sahil87 Toolkit Standards

**Change**: 260717-vlr1-constitution-toolkit-standards
**Intake**: `intake.md`

## Requirements

### Constitution: Toolkit Standards Article

#### R1: New `### Toolkit Standards` article under Additional Constraints
`fab/project/constitution.md` MUST gain a new `### Toolkit Standards` subsection under the existing `## Additional Constraints` section, placed as the fourth article immediately after `### Dependency Discipline`. The article body MUST be the intake's verbatim text (em-dash typography, RFC-2119 `MUST`), with the heading directly followed by prose and NO `**Rationale**` paragraph — matching the other Additional Constraints articles (Test Integrity, Build Reproducibility, Dependency Discipline).

- **GIVEN** the constitution has an `## Additional Constraints` section with three articles ending at `### Dependency Discipline`
- **WHEN** the amendment is applied
- **THEN** a `### Toolkit Standards` article appears after `### Dependency Discipline` and before the `## Governance` section
- **AND** its body is the intake's exact article text with em dashes (not `--`) and no Rationale paragraph

#### R2: Article stays enumeration-free
The `### Toolkit Standards` article MUST NOT copy standard names, counts, or per-standard URLs into the constitution. It MUST reference only the two stable pointers: the `shll standards` command (enumeration + `shll standards <name>` for reading one) and the canonical source location (the sahil87/shll repository's `docs/site/standards/` tree, rendered on https://shll.ai).

- **GIVEN** the toolkit publishes an evolving set of standards enumerated by `shll standards`
- **WHEN** the article is written
- **THEN** it names no individual standard, no count, and no per-standard URL
- **AND** it states that standards added or revised upstream bind this repo without further amendment

#### R3: Governance line version + Last Amended bump
The `## Governance` line MUST be updated to `**Version**: 1.1.0 | **Ratified**: 2026-05-03 | **Last Amended**: 2026-07-18` — a minor version bump (additive new article, no rule changed or removed), the ratification date unchanged, and the amendment date set to 2026-07-18.

- **GIVEN** the current governance line reads `**Version**: 1.0.0 | **Ratified**: 2026-05-03 | **Last Amended**: 2026-05-03`
- **WHEN** the amendment is applied
- **THEN** the line reads `**Version**: 1.1.0 | **Ratified**: 2026-05-03 | **Last Amended**: 2026-07-18`

### Non-Goals

- No conformance fixes — the CLI surface, help output, README.md, and docs/site/ are not audited or changed against the standards. This change only installs the obligation for future changes.
- No file other than `fab/project/constitution.md` is touched.
- No memory hydration of the obligation — the constitution is always-loaded governance; duplicating it into `docs/memory/` would create a second source of truth (handled at the hydrate stage, informational here).

### Design Decisions

1. **Enumeration by pointer, not by copy**: reference `shll standards` and the canonical source rather than listing standards — *Why*: keeps the article evergreen as standards evolve upstream — *Rejected*: inlining standard names/URLs, which would drift and require re-amendment on every upstream change.
2. **No Rationale paragraph**: match the existing Additional Constraints articles' structure (heading + prose only) — *Why*: file-consistency; Core Principles carry Rationale, Additional Constraints do not — *Rejected*: adding a Rationale block, which would diverge from the three sibling articles.

## Tasks

### Phase 1: Amendment

- [x] T001 Add the `### Toolkit Standards` article to `fab/project/constitution.md` immediately after `### Dependency Discipline` (before `## Governance`), using the intake's verbatim article text with em-dash typography and no Rationale paragraph <!-- R1 --> <!-- R2 -->
- [x] T002 Update the `## Governance` line in `fab/project/constitution.md` to `**Version**: 1.1.0 | **Ratified**: 2026-05-03 | **Last Amended**: 2026-07-18` <!-- R3 -->

### Phase 2: Verification

- [x] T003 Re-read the amended `fab/project/constitution.md` and confirm the article text matches the intake exactly (em dashes, no `--`; no Rationale paragraph; enumeration-free — only `shll standards` and the sahil87/shll + shll.ai pointers), placement is correct (after Dependency Discipline, before Governance), and the governance line is exact <!-- R1 --> <!-- R2 --> <!-- R3 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: A `### Toolkit Standards` article exists in `fab/project/constitution.md` as the fourth Additional Constraints subsection, immediately after `### Dependency Discipline` and before `## Governance`, with the intake's verbatim body and no Rationale paragraph
- [x] A-002 R2: The article contains no standard names, counts, or per-standard URLs — only the `shll standards` command reference and the canonical source pointer (sahil87/shll `docs/site/standards/`, rendered on https://shll.ai)
- [x] A-003 R3: The `## Governance` line reads exactly `**Version**: 1.1.0 | **Ratified**: 2026-05-03 | **Last Amended**: 2026-07-18`

### Behavioral Correctness

- [x] A-004 R1: The article uses em-dash (`—`) typography, not double hyphens (`--`), consistent with the constitution's existing prose

### Scenario Coverage

- [x] A-005 R1: No file other than `fab/project/constitution.md` is modified by this change (git diff shows a single touched file)

### Code Quality

- [x] A-006 Pattern consistency: The new article matches the structure of the surrounding Additional Constraints articles (heading directly followed by prose, no Rationale block)
- [x] A-007 No unnecessary duplication: The article points at the live enumeration (`shll standards`) rather than duplicating standard contents into the constitution

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Article placed as the fourth Additional Constraints subsection, after `### Dependency Discipline` and before `## Governance` | Intake specifies placement explicitly; section exists; appending after the last article matches the file's additive structure | S:95 R:90 A:95 D:95 |
| 2 | Confident | Render the intake's `--` as em dashes (`—`) in the article text | Constitution's existing prose uses em dashes; double hyphens are a plain-text transport artifact; trivially reversible | S:60 R:95 A:80 D:75 |
| 3 | Certain | No `**Rationale**` paragraph on the new article | Intake specifies matching the existing Additional Constraints articles, which carry no Rationale (unlike Core Principles) | S:95 R:90 A:95 D:95 |
| 4 | Confident | Version bump 1.0.0 → 1.1.0 (minor) | Additive new article, no rule changed/removed; standard constitution-versioning convention (minor = new article) | S:70 R:90 A:75 D:80 |

4 assumptions (2 certain, 2 confident, 0 tentative).
