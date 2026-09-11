# Plan: HexoKit Banner Sweep

**Change**: 260911-jmkb-hexokit-banner-sweep
**Intake**: `intake.md`

## Requirements

### README: Toolkit blockquote

#### R1: Canonical HexoKit blockquote
`README.md` line 3 MUST be exactly `> Part of [HexoKit](https://hexokit.com) — see all projects there.` — the line the `readme-extraction` standard mandates for all seven toolkit repos as revised by shll PR #98 (C1). The em-dash (U+2014), the absence of the article ("Part of", not "Part of the"), the link text `HexoKit`, and the trailing period are all part of the byte-exact contract.

- **GIVEN** the README currently carries `> Part of the [shll toolkit](https://shll.ai) — see all projects there.` on line 3
- **WHEN** the change is applied
- **THEN** line 3 reads `> Part of [HexoKit](https://hexokit.com) — see all projects there.` byte-for-byte
- **AND** no other line in `README.md` changes

#### R2: Head order preserved
The README head MUST keep the standard's mandated order: line 1 `# idea` H1 → blank → line 3 blockquote → blank → line 5 the contiguous three-badge run → blank → line 7 tagline prose. Nothing may be introduced above the H1.

- **GIVEN** the head order is already conformant before the change
- **WHEN** line 3 is replaced
- **THEN** `head -7 README.md` shows H1, blank, blockquote, blank, badge line, blank, tagline — unchanged apart from the blockquote text

#### R3: Phase-1 scope boundary — nothing else changes
No other file and no other README line SHALL change. In particular every `shll.ai` URL (install one-liners on README lines 12/18 and `docs/site/install.md` lines 10/17; the command-reference link on README line 100), the "shll toolkit" / "shll installer" family-name prose (README line 15; `docs/site/install.md` lines 3/14), the "Have other shll tools?" hint, the constitution's § Toolkit Standards article, badge and GitHub URLs, `docs/site/skill.md` and its embedded copy, all Go source, and all `docs/memory/**` narrative MUST remain byte-identical at apply. (Memory present-truth is corrected at hydrate, not apply.)

- **GIVEN** the full working tree before the change
- **WHEN** apply completes
- **THEN** `git diff --stat` lists exactly `README.md` (1 insertion, 1 deletion) plus this change's own `fab/changes/260911-jmkb-hexokit-banner-sweep/` artifacts
- **AND** `go test ./...` is green (nothing in `src/` changed; the skill-bundle drift guard passes by construction)

### Non-Goals

- Flipping "shll toolkit" family-name prose or `shll.ai` URLs to HexoKit / hexokit.com — deferred to the X2/X4-era pass while shll.ai remains the consuming site.
- Any `rk` / `run-kit` substrate or product-mention edit — grep-verified zero occurrences in this repo (intake § What Changes 2).
- Constitution amendment — D14 keeps `shll standards` and the `sahil87/shll` canonical-source reference; the article's toolkit-name wording waits for X4.
- Repo/badge link changes — R2 of the plan (repo rename bundle); `idea`'s own repo is not being renamed.

### Design Decisions

#### Take the revised blockquote from the C1 PR diff, not the installed shll binary
**Decision**: The replacement line is sourced from shll PR #98's diff to `docs/site/standards/readme-extraction.md`, verified at intake time; the installed `shll standards readme-extraction` still prints the pre-C1 line and is not the oracle for this change.
**Why**: The plan's cutover order is C1 → (C3 ∥ C7); C7 is gated on C1's PR being up and reviewed, not on shll's release. Waiting for a shll release would serialize seven satellite PRs behind an unrelated release cadence.
**Rejected**: Waiting for `shll` to ship C1 so the installed standard matches — adds latency with no correctness gain, since the line text is fixed in the merged-intent PR; running a conformance check against the installed binary — would flag the new line as non-conformant for the duration of the two-site window and is documented in the PR body as the expected state.
*Introduced by*: 260911-jmkb-hexokit-banner-sweep

## Tasks

### Phase 2: Core Implementation

- [x] T001 Replace `README.md` line 3 `> Part of the [shll toolkit](https://shll.ai) — see all projects there.` with `> Part of [HexoKit](https://hexokit.com) — see all projects there.` (byte-exact: em-dash, no article, trailing period); touch no other line <!-- R1, R2 -->

### Phase 3: Integration & Edge Cases

- [x] T002 Verify scope: `git diff --stat` shows only `README.md` (+1/−1) beyond `fab/changes/260911-jmkb-hexokit-banner-sweep/`; `head -7 README.md` shows the H1 → blockquote → badges → tagline order; run `go test ./...` from `src/` and confirm green <!-- R3 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `README.md` line 3 is byte-identical to `> Part of [HexoKit](https://hexokit.com) — see all projects there.` (`grep -c` for the exact string returns 1; the old `shll toolkit](https://shll.ai) — see` string returns 0)
- [x] A-002 R2: `head -7 README.md` is `# idea`, blank, the new blockquote, blank, the three-badge line, blank, the tagline — no frontmatter, HTML, or comment above the H1
- [x] A-003 R3: The apply diff touches only `README.md` (+1/−1) and the change's own `fab/changes/…` artifacts; `docs/site/**`, `src/**`, `fab/project/**`, `docs/memory/**` are unchanged at apply

### Behavioral Correctness

- [x] A-004 R1: The em-dash is U+2014 (not `--` or `-`), the article "the" is absent, and the trailing period is present — a byte-level comparison against the standard's fenced line in shll#98 matches

### Scenario Coverage

- [x] A-005 R3: `go test ./...` passes (skill-bundle drift guard included), confirming no source or embedded-docs change rode along

### Edge Cases & Error Handling

- [x] A-006 R3: No `shll.ai` URL, "shll toolkit"/"shll installer" phrase, or "Have other shll tools?" hint was altered — `grep -c 'shll.ai' README.md docs/site/install.md` reads README 3 (down from 4: the only removed occurrence is the replaced blockquote's own link) and install.md 4 (unchanged)

### Code Quality

- [x] A-007 Pattern consistency: The edit mirrors the `260718-92gj` blockquote-flip precedent — one head-line replacement, head order intact, identifiers untouched
- [x] A-008 No unnecessary duplication: No new files, sections, or duplicated prose introduced

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Conformance note for the PR body: the blockquote matches the `readme-extraction` standard as revised in shll#98 (C1); the installed `shll` binary reports the pre-C1 line until shll releases C1 — the intended C1 → C7 ordering, not a defect.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Exactly one file line changes at apply; the run-kit product-mention clause needs no task | Intake grep-verified zero occurrences; the standard's line is fixed by shll#98 | S:95 R:100 A:100 D:100 |
| 2 | Certain | `go test ./...` is the only verification beyond the byte checks — no new tests | Docs-only change; no source touched; existing drift guard covers the embedded skill bundle | S:90 R:100 A:100 D:100 |

2 assumptions (2 certain, 0 confident, 0 tentative).
