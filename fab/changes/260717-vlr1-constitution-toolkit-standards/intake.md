# Intake: Constitution Amendment — Bind to sahil87 Toolkit Standards

**Change**: 260717-vlr1-constitution-toolkit-standards
**Created**: 2026-07-18

## Origin

One-shot `/fab-new` invocation with a fully specified design. Raw input:

> Task: Amend this repo's fab constitution to bind it to the sahil87 toolkit standards. This repo is part of the sahil87 toolkit. The toolkit publishes binding, producer-facing standards — CLI design principles plus mechanical contracts (machine-readable help output, README/docs-site structure, and others over time). They are canonically authored in the sahil87/shll repository's docs/site/standards/ tree, rendered on https://shll.ai, and readable offline via the `shll standards` command. This change adds a constitution article so every future pipeline run in this repo loads and enforces the obligation.
>
> Make this change:
>
> 1. In fab/project/constitution.md, add a new article under Additional Constraints (create the section if this constitution lacks it, matching the file's existing structure): [article text — reproduced in full under "What Changes" below]
> 2. Bump the constitution's Last Amended date (and version, per this file's own governance line).
> 3. Deliberate constraint: do NOT copy standard names, counts, or per-standard URLs into the constitution — `shll standards` is the enumeration, and the article must stay correct as standards evolve.
>
> Ship per this repo's normal flow (docs-type fab change → PR). Nothing else is in scope — no conformance fixes in this change.

No prior conversation preceded the invocation; the input itself carries all design decisions.

## Why

1. **Problem**: This repo (the `idea` CLI) is part of the sahil87 toolkit, whose binding producer-facing standards — CLI design principles plus mechanical contracts such as machine-readable help output and README/docs-site structure — are authored in the sahil87/shll repository's `docs/site/standards/` tree, rendered on https://shll.ai, and readable offline via `shll standards`. Nothing in this repo's fab governance currently references those standards, so pipeline runs (apply, review) have no loaded obligation to check surface changes against them. The relationship exists de facto (shll.ai already pulls this repo's help-dump JSON and renders README/`docs/site/**` — see `docs/memory/release/pipeline.md`), but it is not enforced de jure.
2. **Consequence of not fixing**: future changes to the CLI surface, help output, README.md, or docs/site/ can silently drift from toolkit standards, since no always-loaded artifact instructs agents to check them. Conformance would depend on the invoking human remembering.
3. **Why this approach**: `fab/project/constitution.md` is in fab's always-load context layer — every pipeline stage (and every dispatched subagent, per the standard subagent context) reads it. A constitution article is therefore the single point that makes the obligation self-enforcing for all future runs. Pointing at the live enumeration (`shll standards`) rather than copying its contents keeps the article evergreen: standards added or revised upstream bind this repo without re-amendment.

## What Changes

### 1. New article in `fab/project/constitution.md`

Add a fourth subsection under the existing `## Additional Constraints` section (the section already exists with Test Integrity, Build Reproducibility, and Dependency Discipline — no need to create it), placed after `### Dependency Discipline`:

```markdown
### Toolkit Standards
This tool is part of the sahil87 toolkit and MUST conform to the toolkit's published standards. The standards are enumerated by running `shll standards` — each entry names what it governs; read one with `shll standards <name>`. Before changing the CLI surface, help output, README.md, or docs/site/, the change MUST be checked against the standards governing that surface. If shll is unavailable, the canonical sources are the sahil87/shll repository's docs/site/standards/ tree (rendered on https://shll.ai). Standards added or revised there bind this repo without further amendment to this constitution.
```

The text is the user-provided article verbatim, with the double hyphens (`--`) of the raw input rendered as em dashes (`—`) to match the constitution's existing typography. <!-- assumed-note: see Assumptions #3 --> Body formatting matches the file's existing articles: heading directly followed by prose (existing Additional Constraints articles carry no **Rationale** paragraph, unlike Core Principles — so none is added here).

### 2. Governance line bump

Update the `## Governance` line:

```markdown
**Version**: 1.1.0 | **Ratified**: 2026-05-03 | **Last Amended**: 2026-07-18
```

- **Version** 1.0.0 → 1.1.0: minor bump — a new article is added, no existing rule is changed or removed. The governance line records only the format (no explicit bump rules), so the standard constitution-versioning convention applies (major = incompatible change/removal, minor = new article/material expansion, patch = wording).
- **Ratified** stays 2026-05-03.
- **Last Amended** → 2026-07-18 (the date this amendment lands).

### 3. Deliberately out of scope

- **No enumeration in the constitution**: no standard names, counts, or per-standard URLs are copied in. `shll standards` is the enumeration; the article must stay correct as standards evolve. Only the two stable pointers appear: the `shll standards` command and the canonical source location (sahil87/shll `docs/site/standards/`, rendered on https://shll.ai).
- **No conformance fixes**: this change does not audit or fix the CLI surface, help output, README.md, or docs/site/ against the standards. It only installs the obligation for future changes.
- **No other file changes**: `fab/project/constitution.md` is the only file touched.

## Affected Memory

None. The constitution is always-loaded fab governance, not spec-level system behavior — the memory domains (ci, cli, release) track how the system behaves, and duplicating a governance obligation into memory would create a second source of truth for something the constitution already carries into every run.

## Impact

- **Files**: `fab/project/constitution.md` only (one new `###` subsection + one governance-line edit).
- **Code/tests**: none. No source, no tests, no build impact. (`fab/` is in `true_impact_exclude`.)
- **Behavioral effect**: every future pipeline run in this repo loads the obligation via the always-load context layer; changes touching the CLI surface, help output, README.md, or docs/site/ must be checked against the governing standards from now on.
- **Change type**: docs.

## Open Questions

None — the input specifies the article text, placement, governance bump, and scope boundaries explicitly.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Add the article under the existing `## Additional Constraints` section, as its fourth subsection after `### Dependency Discipline` | Section exists (no creation needed); appending after the last article matches the file's additive structure; article text provided in full by user | S:95 R:90 A:95 D:95 |
| 2 | Confident | Version bump 1.0.0 → 1.1.0 (minor) | Governance line records format only, no explicit bump rules; additive new article = minor per standard constitution-versioning convention | S:70 R:90 A:75 D:80 |
| 3 | Confident | Render the input's `--` as em dashes (`—`) in the article text | The constitution's existing prose uses em dashes; double hyphens in the raw input read as a plain-text transport artifact; trivially reversible | S:60 R:95 A:80 D:75 |
| 4 | Certain | `change_type` = docs | Explicit in input: "docs-type fab change → PR" | S:95 R:95 A:95 D:95 |
| 5 | Certain | Constitution stays enumeration-free — no standard names, counts, or per-standard URLs | Explicit deliberate constraint in input; keeps the article evergreen as standards evolve | S:100 R:90 A:100 D:100 |
| 6 | Certain | Scope = constitution amendment only; no conformance fixes in this change | Explicit in input: "Nothing else is in scope — no conformance fixes in this change" | S:100 R:85 A:100 D:100 |
| 7 | Confident | Affected Memory: none | Constitution is always-loaded governance, not spec-level system behavior; memory duplication would create a second source of truth | S:65 R:80 A:75 D:70 |
| 8 | Certain | Last Amended = 2026-07-18; Ratified unchanged | Amendment date is today; ratification date is historical fact | S:90 R:95 A:95 D:95 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).
