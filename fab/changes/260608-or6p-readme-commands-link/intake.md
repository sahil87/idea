# Intake: README link to shll.ai command-reference page

**Change**: 260608-or6p-readme-commands-link
**Created**: 2026-06-08
**Status**: Draft

## Origin

> Also add references to docs/site/* files from README.md - example: the installation section should point to install.md. For the command reference (also) point to https://shll.ai/tools/<tool-name>/commands/ . Add a new PR for these changes.

Follow-up to PR #9 (`260608-3ra7-shll-readme-contract`). The install-guide and workflows `docs/site/*` links the user mentions were already added in #9, so the only remaining item is the command-reference link to the live shll.ai page. Confirmed scope with the user: add **only** the command-reference link, keep the existing absolute `docs/specs/overview.md` link too (the user's "(also)"), and ship as a **new PR** stacked on #9's branch (since #9 is not yet merged).

## Why

shll.ai auto-generates a per-tool command-reference page at `/tools/<slug>/commands/` from the tool's `help-dump` JSON (the `commands` slug is reserved and shll.ai-owned — it is NOT a `docs/site/` page the repo creates). Linking the README's "Command reference" section to `https://shll.ai/tools/idea/commands/` points readers at the always-current, rendered reference without the repo having to maintain a second copy. It complements (does not replace) the existing absolute `docs/specs/overview.md` link.

## What Changes

Single one-line edit to `README.md` (the command-reference pointer line, after the command table):

- **Before**: `Run \`idea <command> --help\` for inline flag details, or see [\`docs/specs/overview.md\`](https://github.com/sahil87/idea/blob/main/docs/specs/overview.md) for the full CLI reference and [\`docs/specs/backlog-format.md\`](…) for the file format contract.`
- **After**: inserts `browse the [full command reference](https://shll.ai/tools/idea/commands/) on shll.ai,` between the inline-flag clause and the `docs/specs/overview.md` clause.

`https://shll.ai/tools/idea/commands/` is an absolute https URL (correct per the contract — it leaves the rendered set, and `commands` is a reserved shll.ai slug, not a repo `docs/site/` page).

No new `docs/site/` files, no other links — the install-guide (`docs/site/install.md`) and workflows (`docs/site/workflows.md`) README links already shipped in PR #9.

## Affected Memory

None. The shll.ai publishing relationship is already documented in `docs/memory/release/pipeline.md` (added in #9); a single additional outbound README link does not change that record.

## Impact

- **Files touched**: `README.md` (one line).
- No code, tests, CI, or `docs/site/` changes.
- Branch is stacked on PR #9's branch (`260608-3ra7-shll-readme-contract`); once #9 merges, this rebases cleanly onto `main`.

## Open Questions

- None. Scope and target confirmed with the user.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Add only the command-reference link; install/workflows links already exist from #9. | User confirmed scope ("Just the command-ref link"). | S:98 R:98 A:98 D:98 |
| 2 | Certain | Link target is `https://shll.ai/tools/idea/commands/`, kept alongside the existing `docs/specs/overview.md` link. | User confirmed ("shll.ai commands page", keep both per "(also)"). | S:97 R:97 A:97 D:97 |
| 3 | Certain | New PR stacked on #9's branch, not branched from `main`. | #9 is unmerged; stacking keeps the diff to one line and rebases cleanly post-merge. | S:96 R:96 A:95 D:96 |
| 4 | Certain | `commands` is a reserved shll.ai-owned slug → hardcoded absolute site URL is correct (not a `docs/site/` page). | shll.ai generates `/tools/<slug>/commands/` from help-dump JSON per the README-extraction contract. | S:96 R:95 A:96 D:95 |

4 assumptions (4 certain, 0 confident, 0 tentative, 0 unresolved).
