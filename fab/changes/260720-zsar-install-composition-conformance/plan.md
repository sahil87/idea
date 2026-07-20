# Plan: Install-Composition Standard Conformance (Policy B)

**Change**: 260720-zsar-install-composition-conformance
**Intake**: `intake.md`

## Requirements

### Docs: install-page Policy B conformance

#### R1: Page intro points at the shll installer
The intro paragraph of `docs/site/install.md` MUST NOT present the Homebrew tap as the primary install path; it SHALL present the shll installer (https://shll.ai) instead, keeping the existing sentence structure (build-from-source alternative, page-scope sentence covering completion and upgrades).

- **GIVEN** the current intro "The fastest way to get `idea` is the Homebrew tap."
- **WHEN** the page is conformed
- **THEN** the intro reads "The fastest way to get `idea` is the shll installer." and still mentions building from source, shell completion, and upgrades

#### R2: Per-formula install section replaced with the shll.ai pointer
The `## Homebrew tap (recommended)` section (heading + `brew install sahil87/tap/idea` fence + follow-on paragraph) MUST be replaced with an `## Install via shll (recommended)` section carrying: the subset curl bootstrap (`curl -fsSL https://shll.ai/install | sh -s -- idea`), the full-toolkit bootstrap (`curl -fsSL https://shll.ai/install | sh`) with an absolute link to https://shll.ai, the `shll install idea` mention, and a closing sentence preserving the Homebrew-managed-lifecycle note plus the forward link to [Upgrading](#upgrading) — the intake's proposed content, mirroring the README's Install section.

- **GIVEN** the conformed `docs/site/install.md`
- **WHEN** grepping `README.md` and `docs/site/` for `brew install`
- **THEN** zero matches remain — no per-formula install instruction is documented anywhere in the rendered set

#### R3: Keep-intact scope — no collateral edits
The Manual build, Shell completion, and Upgrading sections of `docs/site/install.md` MUST remain unchanged; `README.md`, `docs/site/skill.md`, and `docs/site/workflows.md` MUST NOT change; incidental Homebrew-as-mechanism mentions (e.g. `idea update` upgrading via `brew`) MUST be kept.

- **GIVEN** the conformed working tree
- **WHEN** running `git diff --name-only`
- **THEN** the only modified doc-surface file is `docs/site/install.md`, and its diff touches only the intro and the replaced install section

#### R4: Rendering constraints hold
Per memory `release/pipeline.md` (the page renders verbatim at https://shll.ai/idea/install): any external link in the edited content MUST be absolute `https://…`; intra-`docs/site/` links stay natural-relative; the filename stays `install.md` (non-reserved slug); code fences are written at top level (the intake's nested fences are illustrative only); no mermaid fences or theme fragments are introduced.

- **GIVEN** the edited content
- **WHEN** inspecting its links and fences
- **THEN** every external link is absolute `https://…`, intra-site links are relative, and all fences are normal top-level fences

#### R5: No dangling anchors
The retired heading's anchor `#homebrew-tap-recommended` MUST NOT be referenced anywhere in `README.md` or `docs/`; the `#upgrading` anchor used by the new section's forward link MUST still resolve (the `## Upgrading` heading is unchanged).

- **GIVEN** the conformed tree
- **WHEN** grepping `README.md` and `docs/` for `homebrew-tap-recommended`
- **THEN** there are zero references, and `docs/site/install.md` still contains the `## Upgrading` heading

### Non-Goals

- Policy A conformance (formula `depends_on`, runtime probes) — binds the tap formula and binary, neither touched by this change
- Any `README.md` edit — its Install section already carries the conforming shll.ai bootstrap
- `docs/memory/release/pipeline.md` updates — hydrate-stage work, out of apply's scope

## Tasks

### Phase 2: Core Implementation

- [x] T001 Rewrite the intro paragraph of `docs/site/install.md` (lines 3–5) to present the shll installer as the fastest path, per the intake's proposed wording <!-- R1 -->
- [x] T002 Replace the `## Homebrew tap (recommended)` section of `docs/site/install.md` (heading + `brew install sahil87/tap/idea` fence + follow-on paragraph) with the intake's `## Install via shll (recommended)` section, written with top-level `sh` fences <!-- R2 -->

### Phase 3: Integration & Edge Cases

- [x] T003 Verify the conformed tree: `grep -rn 'brew install' README.md docs/site/` returns nothing; `grep -rn 'homebrew-tap-recommended' README.md docs/` returns nothing; `## Upgrading` still present in `docs/site/install.md`; `git diff --name-only` shows only `docs/site/install.md` (plus this change's fab artifacts); edited content's external links are absolute `https://…` and intra-site links relative <!-- R3 R4 R5 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `docs/site/install.md`'s intro presents the shll installer as the fastest way to get `idea`, retaining the build-from-source alternative and the completion/upgrades scope sentence
- [x] A-002 R2: `docs/site/install.md` carries an `## Install via shll (recommended)` section with the subset bootstrap (`curl -fsSL https://shll.ai/install | sh -s -- idea`), the full-toolkit bootstrap, the `shll install idea` mention, the Homebrew-managed-lifecycle note, and the forward link to [Upgrading](#upgrading)

### Removal Verification

- [x] A-003 R2: no per-formula `brew install` line remains anywhere in `README.md` or `docs/site/` (grep returns zero matches)

### Behavioral Correctness

- [x] A-004 R3: only `docs/site/install.md` changed among doc surfaces; its Manual build, Shell completion, and Upgrading sections are intact; `docs/site/skill.md`'s "Self-update the binary via Homebrew" row is kept

### Edge Cases & Error Handling

- [x] A-005 R4: the edited content obeys the rendering constraints — external links absolute `https://…`, intra-`docs/site/` links natural-relative, top-level fences only, filename unchanged, no mermaid/theme fragments
- [x] A-006 R5: `#homebrew-tap-recommended` is referenced nowhere in `README.md`/`docs/`; the `## Upgrading` heading (the `#upgrading` anchor target) is still present

### Code Quality

- [x] A-007 Pattern consistency: the new section mirrors the README Install section's structure and voice (subset bootstrap first, full-toolkit second, `shll install` mention)
- [x] A-008 No unnecessary duplication: the replacement points to the centralized shll.ai install story instead of re-documenting the tap; the only duplication is the deliberate README-mirroring the intake prescribes

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Adopt the intake's proposed replacement content verbatim (intro wording, section heading, bootstrap commands, closing sentence), rendered with top-level `sh` fences | Intake supplies exact text and explicitly marks the nested fences as illustrative; directive calls the change mechanical | S:95 R:95 A:95 D:95 |
| 2 | Certain | Verification is re-read + grep against the documented constraints, not the Go test suite | change_type is docs; no `source_paths`/`test_paths` files are touched, so the test suite cannot exercise this change | S:90 R:95 A:95 D:90 |

2 assumptions (2 certain, 0 confident, 0 tentative).
