# Intake: shll Toolkit Name Conformance

**Change**: 260718-92gj-shll-toolkit-rename-conformance
**Created**: 2026-07-18

## Origin

One-shot `/fab-new` invocation with a fully-specified task. Raw input (abridged to the operative content):

> Task: Conform this repo to the toolkit's standardized name — "shll toolkit".
>
> The toolkit formerly named "sahil87 toolkit" is now the **shll toolkit** (sahil87/shll#56). The readme-extraction standard's canonical README blockquote changed accordingly. This repo's constitution already binds it to revised standards without amendment — this task is the conformance work.
>
> 1. **README blockquote** — replace the toolkit blockquote with this exact line, byte-identical, keeping the mandated head order (H1 -> blockquote -> badges): `> Part of the [shll toolkit](https://shll.ai) — see all projects there.`
> 2. **Prose sweep** — replace remaining `sahil87 toolkit` -> `shll toolkit` and `sahil87 tool(s)` -> `shll tool(s)` wherever they appear as prose: README, `docs/site/**` (including the skill bundle `docs/site/skill.md` if present), CLI help text and user-visible strings (update their test goldens), and `fab/project/` files. If this repo embeds docs in the binary (skill bundle or similar), re-run its sync step so drift-guard tests pass.
> 3. **Constitution (cosmetic, same PR)** — in the Toolkit Standards article, change "part of the sahil87 toolkit" to "part of the shll toolkit" and bump `Last Amended` per the file's governance line. Nothing else in the article changes.
> 4. **Do NOT touch identifiers**: `sahil87/tap` formula names, `github.com/sahil87/…` and `raw.githubusercontent.com/sahil87/…` URLs, the `sahil87/shll` canonical-source reference in the constitution article, and any GitHub-owner constants in code. Historical artifacts (`fab/changes/` archives) stay untouched.
>
> Ship per this repo's normal flow (one fab change -> PR). Tests green; if help text changed, the help-dump JSON shape is unchanged (text-only edits — no `schema_version` bump).

**Precondition verified at intake time** (2026-07-18): `shll standards readme-extraction` runs on this machine and its "README structure" §1 shows the new canonical blockquote verbatim: `> Part of the [shll toolkit](https://shll.ai) — see all projects there.` No `shll update` was needed; the do-not-proceed-from-memory stop condition did not trigger.

**Scope grounding performed at intake time**: a repo-wide grep (`sahil87 toolkit`, `sahil87 tool`, `sahil87`, `toolkit` — excluding `.git/` and treating `fab/changes/` as historical) enumerated every occurrence. The complete edit list is in § What Changes; everything else is an identifier or archive and stays untouched.

## Why

1. **Problem**: The toolkit this repo belongs to was renamed from "sahil87 toolkit" to "shll toolkit" (sahil87/shll#56), and the readme-extraction standard's canonical README blockquote changed with it. This repo still carries the old name in its README blockquote (an older wording variant: `> Part of [@sahil87's open source toolkit](https://shll.ai) — see all projects there.`), in two further README prose spots, in `docs/site/install.md`, and in the constitution's Toolkit Standards article.
2. **Consequence if unfixed**: The README blockquote is a byte-exact cross-repo contract ("this exact line in all seven repos" per the standard) consumed mechanically by shll.ai's pull — a non-canonical blockquote is a standing standards violation on this repo's public page, and the constitution (v1.1.0 § Toolkit Standards) makes revised standards binding "without further amendment", so the repo is out of conformance as of the rename.
3. **Approach**: Pure prose/markdown conformance sweep, no behavior change. The constitution article already binds the repo to revised standards, so no constitutional amendment is required for the obligation — only the cosmetic in-article wording fix, shipped in the same PR.

## What Changes

### 1. README.md — blockquote + two prose spots

Three line edits (line numbers as of intake):

- **Line 3** (the toolkit blockquote — currently the pre-rename wording):
  - Before: `> Part of [@sahil87's open source toolkit](https://shll.ai) — see all projects there.`
  - After (byte-identical to the standard's canonical line): `> Part of the [shll toolkit](https://shll.ai) — see all projects there.`
  - The mandated head order is already in place (line 1 `# idea` H1 → line 3 blockquote → line 5 badges) and MUST be preserved unchanged.
- **Line 15**: `…To install the entire sahil87 toolkit instead:` → `…To install the entire shll toolkit instead:`
- **Line 54**: `> 💡 Have other sahil87 tools? …` → `> 💡 Have other shll tools? …` — the rest of the line, including the `https://github.com/sahil87/shll#…` link target, stays byte-identical.

All other `sahil87` occurrences in README.md (line 5 badge URLs; lines 99/128/136 `github.com/sahil87/…` doc links) are identifiers — untouched.

### 2. docs/site/install.md — one prose spot

- **Line 66**: `> Have other sahil87 tools? …` → `> Have other shll tools? …` — lines 67–68 (including the `github.com/sahil87/shll#…` link) stay untouched, as does the `brew install sahil87/tap/idea` formula on line 10.

`docs/site/workflows.md` contains only a `github.com/sahil87/…` URL (identifier — untouched). `docs/site/skill.md` contains no `sahil87` occurrence at all — no edit, and therefore **no `scripts/sync-skill.sh` re-run is required** (the embedded copy `src/cmd/idea/skill/skill.md` stays byte-identical to its canonical source; the drift-guard test in `src/cmd/idea/skill_test.go` passes by construction).

### 3. fab/project/constitution.md — cosmetic article wording + governance line

In § Toolkit Standards (line 47): `This tool is part of the sahil87 toolkit and MUST conform…` → `This tool is part of the shll toolkit and MUST conform…`. Nothing else in the article changes — in particular the `sahil87/shll` canonical-source reference and the `https://shll.ai` URL stay verbatim.

Governance line (line 51, currently `**Version**: 1.1.0 | **Ratified**: 2026-05-03 | **Last Amended**: 2026-07-18`): bump `Last Amended` to today, 2026-07-18 — which equals the current value, so the line is a no-op edit. `Version` stays 1.1.0 (task directs bumping `Last Amended` only; see Assumptions #4).

### 4. CLI help text / user-visible strings — verified no-op

Repo-wide grep confirms **zero** `sahil87 toolkit` / `sahil87 tool(s)` occurrences in `src/**` or `scripts/**` user-visible strings. The only `sahil87` hits in source are the Homebrew formula constant `sahil87/tap/idea` (`src/internal/idea/update.go:31`, echoed in `update_test.go`) — an identifier, explicitly out of scope — and generic code comments saying "toolkit standard/convention" that never carry the old name. Therefore: no help-text edits, no test-golden updates, no help-dump changes (`schema_version` untouched by construction). The full `go test ./...` suite still runs as verification.

### 5. Out of scope / untouched

- `fab/changes/**` historical artifacts (several carry "sahil87 toolkit" in archived intakes/plans/history) — stay verbatim per task.
- All `sahil87/tap` formula references, `github.com/sahil87/…` and `raw.githubusercontent.com/sahil87/…` URLs, badge URLs, and GitHub-owner constants.
- `docs/memory/**` prose carrying the old name is corrected at the hydrate stage, not during apply (see Affected Memory and Assumptions #5).

## Affected Memory

- `cli/structure`: (modify) § Toolkit-standards conformance prose says "binds this repo to the sahil87 toolkit's published standards" — rename to "shll toolkit"; optionally note this change as the rename-conformance touch.
- `cli/skill`: (modify) line "It adopts the sahil87 toolkit's `skill` standard" — rename to "shll toolkit".
- `release/pipeline`: (modify, optional) if hydrate judges it useful, note that the README blockquote now matches the post-rename canonical line; no old-name prose exists in this file today.

## Impact

- **Files**: `README.md` (3 lines), `docs/site/install.md` (1 line), `fab/project/constitution.md` (1 line + governance-line no-op) — markdown only. Hydrate additionally touches `docs/memory/cli/structure.md` and `docs/memory/cli/skill.md`.
- **Code/tests**: no source or test changes; `go test ./...` must stay green (drift guard, goldens, help-dump all unaffected by construction).
- **External**: shll.ai's daily README/docs-site pull picks up the canonical blockquote on the next pull after merge. No release/tag required — the pull reads the repo, not the binary.
- **Ship**: normal flow — one fab change → PR.

## Open Questions

None — the task specifies the exact replacement line, the sweep scope, the exclusion list, and the ship flow; the precondition gate was verified live.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Replace the README blockquote with `> Part of the [shll toolkit](https://shll.ai) — see all projects there.` byte-identically, preserving the H1 → blockquote → badges head order | Line given verbatim in the task AND verified live against `shll standards readme-extraction` at intake time (precondition gate passed) | S:95 R:90 A:100 D:100 |
| 2 | Certain | Prose sweep is exactly four line edits (README lines 3/15/54, install.md line 66) plus the constitution wording; every other `sahil87` occurrence is an identifier or `fab/changes/` archive and stays untouched | Enumerated by repo-wide grep at intake time, not inferred; task's exclusion list maps 1:1 onto the remaining hits | S:90 R:85 A:95 D:95 |
| 3 | Certain | No CLI help-text/user-visible-string edits, no test-golden updates, no `sync-skill.sh` re-run, no `schema_version` concern | Grep shows zero old-name occurrences in `src/**`/`scripts/**` user-visible strings and none in `docs/site/skill.md`; the task's conditional clauses ("if help text changed", "if this repo embeds docs") resolve to no-ops — verified, with `go test ./...` as the backstop | S:85 R:90 A:95 D:95 |
| 4 | Confident | Governance line: set `Last Amended` to today (2026-07-18 — equal to the current value, a no-op) and leave `Version` at 1.1.0 | Task directs bumping `Last Amended` only and says "Nothing else in the article changes"; no version bump for a cosmetic wording fix. Trivially reversible if the reviewer prefers a patch bump | S:70 R:95 A:75 D:70 |
| 5 | Confident | Old-name prose in `docs/memory/` (cli/structure, cli/skill) is corrected at hydrate, not during apply | Task's sweep list doesn't name memory, but memory is the repo's authoritative post-implementation record and hydrate owns memory maintenance; leaving "sahil87 toolkit" there would be documented drift | S:60 R:90 A:80 D:70 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
