# Intake: Install-Composition Standard Conformance (Policy B)

**Change**: 260720-zsar-install-composition-conformance
**Created**: 2026-07-20

## Origin

One-shot `/fab-new` invocation with a detailed directive:

> Conform this repo's install documentation to the shll toolkit's install-composition standard, Policy B. Read the authoritative standard first: /home/sahil/code/sahil87/shll/docs/site/standards/install-composition.md (rendered on https://shll.ai). Policy B: per-tool READMEs and doc pages must not carry per-formula "brew install sahil87/tap/&lt;tool&gt;" install instructions; installation points to https://shll.ai (curl bootstrap: `curl -fsSL https://shll.ai/install | sh`; subset installs remain supported via `shll install <tool>`). Task: audit README.md and docs/site/ for per-formula install instructions and replace them with the shll.ai pointer. IMPORTANT distinction: replace install *instructions* (sections telling the user how to install), but KEEP incidental mentions such as actionable error-hint examples in standards/conformance text (Policy A mandates those hints) and historical/changelog references. Mechanical docs-only change; keep all usage and feature content intact.

The authoritative standard was read at intake time. The full audit (grep for `brew install|shll.ai|shll install|homebrew` across `README.md` and `docs/site/`) was also performed at intake time; its findings are encoded in **What Changes** below, so the apply agent inherits a completed audit, not an instruction to audit.

## Why

1. **Pain point**: Policy B of the toolkit's install-composition standard centralizes install documentation on shll.ai. Seven per-repo copies of the install dance drift — every change to the install story (a tap-trust requirement, a bootstrap change) must otherwise be chased across every repo plus the tap. This repo's `docs/site/install.md` still opens with a per-formula "Homebrew tap (recommended)" section carrying `brew install sahil87/tap/idea` — a direct Policy B violation, rendered publicly at `https://shll.ai/idea/install`.
2. **Consequence of not fixing**: the constitution's Toolkit Standards section binds this repo to published shll standards; leaving the violation means the rendered install page contradicts the toolkit's own install story and silently drifts as the bootstrap evolves.
3. **Why this approach**: replace only the per-formula install *instructions* with the shll.ai pointer (curl bootstrap + `shll install` mention), exactly as the standard's "Verifying conformance" checklist prescribes. Individual formula installs remain *supported* — what is unsupported is *documenting* them per-repo — so no functional or usage content changes.

## What Changes

### Audit result (complete — performed at intake)

| Location | Finding | Action |
|----------|---------|--------|
| `README.md` §Install (lines 9–19) | Already conforms: carries the shll.ai curl bootstrap (`curl -fsSL https://shll.ai/install \| sh -s -- idea` subset form + full-toolkit form), no per-formula `brew install` | **No change** |
| `README.md` line 41 | Links to the full install guide at `docs/site/install.md` | **No change** (link target is being conformed, not removed) |
| `docs/site/install.md` §"Homebrew tap (recommended)" (lines 7–16) + page intro (lines 3–5) | **Policy B violation**: `brew install sahil87/tap/idea` documented as the recommended install path | **Replace** with the shll.ai pointer (see below) |
| `docs/site/install.md` §Manual build, §Shell completion, §Upgrading | Manual-build path, completion wiring, `idea update` behavior — mentions Homebrew as the *managed mechanism*, carries no per-formula install instruction | **Keep intact** |
| `docs/site/skill.md` line 35 | `idea update` described as "Self-update the binary via Homebrew" — incidental mechanism mention, not an install instruction | **Keep** |
| `docs/site/workflows.md` | No install content | **No change** |

There are no Policy A error-hint examples (`<tool> is not installed. Install it: brew install sahil87/tap/<tool>`) in this repo's README or docs/site/ — the KEEP carve-out for such hints applies vacuously here.

### docs/site/install.md — replace the per-formula section with the shll.ai pointer

1. **Page intro (lines 3–5)**: currently "The fastest way to get `idea` is the Homebrew tap." Rewrite to point at the shll installer / https://shll.ai, keeping the sentence structure ("...is the shll installer. If you'd rather build from source, a single `just` recipe handles it. This page covers both, plus shell completion and upgrades.").
2. **Replace `## Homebrew tap (recommended)` section** (heading + `brew install sahil87/tap/idea` block + follow-on paragraph) with a shll.ai-pointing section mirroring the README's Install section. Proposed content:

   ```markdown
   ## Install via shll (recommended)

   ```sh
   curl -fsSL https://shll.ai/install | sh -s -- idea
   ```

   Installs idea (plus the shll meta-CLI) via Homebrew, handling tap trust
   automatically. To install the entire [shll toolkit](https://shll.ai) instead:

   ```sh
   curl -fsSL https://shll.ai/install | sh
   ```

   Already have `shll`? `shll install idea` does the same. Either way the binary
   is managed by Homebrew — upgrades, version pinning, and uninstall all go
   through `brew`, and `idea update` (see [Upgrading](#upgrading)) can
   self-upgrade later.
   ```

   The closing sentence preserves the current section's useful content (Homebrew-managed lifecycle, forward link to #upgrading) without the per-formula instruction.
3. **Check internal anchors**: the old heading's anchor (`#homebrew-tap-recommended`) changes. Grep `README.md` and `docs/site/` for references to that anchor and fix any found (none surfaced in the intake audit, but verify at apply).

### Rendering constraints (binding on the edit)

`docs/site/install.md` is pulled and rendered verbatim at `https://shll.ai/idea/install` (see memory `release/pipeline.md`):
- External links MUST be absolute `https://…`; intra-`docs/site/` links stay natural-relative (e.g. `[workflows](workflows.md)`).
- No reserved slugs are involved (file name unchanged).
- Nested fenced code blocks in the proposed content above are illustrative — write the actual file with normal top-level fences.

## Affected Memory

- `release/pipeline`: (modify) add a change-history entry recording the install-composition Policy B conformance (install.md's recommended path is now the shll.ai bootstrap; per-formula `brew install` documentation removed) — mirrors the precedent of `260717-9uh7`'s conformance entry. No structural/contract content changes.

## Impact

- **Files**: `docs/site/install.md` (edit), `docs/memory/release/pipeline.md` (hydrate-stage history entry). `README.md`, `docs/site/skill.md`, `docs/site/workflows.md` audited — no changes.
- **No code, tests, CLI behavior, CI, or release pipeline changes.** Docs-only; `source_paths` untouched.
- **Public surface**: the rendered `https://shll.ai/idea/install` page changes on shll.ai's next daily pull.
- **Standards**: satisfies install-composition Policy B's "Verifying conformance" checklist item — "The README's install section ... links to https://shll.ai instead of carrying per-formula `brew install` lines" (README already did; install.md will after this change). Policy A (formula `depends_on`, runtime probes) is out of scope — it binds the tap formula and binary, neither of which this change touches.

## Open Questions

None — the directive plus the completed audit resolve all decision points.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only `docs/site/install.md` violates Policy B; `README.md` already conforms (its curl bootstrap *is* the shll.ai pointer), so the README audit outcome is no-change | Standard names the curl bootstrap as the conforming form; grep audit of README + docs/site completed at intake | S:90 R:90 A:95 D:90 |
| 2 | Confident | Replacement section mirrors README's Install section (subset bootstrap first, full-toolkit second, `shll install idea` mentioned) under heading "Install via shll (recommended)" | Keeps the two doc surfaces consistent; standard allows curl bootstrap or `shll install` as the pointed-to forms; exact heading wording is easily adjusted | S:80 R:85 A:85 D:75 |
| 3 | Confident | Keep Manual build, Shell completion, and Upgrading sections intact — Homebrew as the managed *mechanism* (e.g. `idea update` runs `brew upgrade`) is not a per-formula install instruction | Policy B's supported-vs-unsupported line bans documenting `brew install sahil87/tap/<tool>`, not mentioning Homebrew; upgrade docs are usage content the directive says to keep | S:85 R:85 A:85 D:80 |
| 4 | Confident | Keep `docs/site/skill.md`'s "Self-update the binary via Homebrew" row — incidental mechanism mention, not an install instruction | Directive's explicit KEEP carve-out for incidental mentions | S:85 R:90 A:85 D:85 |
| 5 | Confident | Memory touch limited to a `release/pipeline` change-history entry at hydrate | Docs-only change alters no contracts; precedent: 260717-9uh7 conformance entries are history-only | S:70 R:90 A:80 D:75 |

5 assumptions (1 certain, 4 confident, 0 tentative, 0 unresolved).
