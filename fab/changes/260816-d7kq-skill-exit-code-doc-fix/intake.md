# Intake: Skill Bundle Exit-Code Doc Fix

**Change**: 260816-d7kq-skill-exit-code-doc-fix
**Created**: 2026-08-17

## Origin

Backlog item `[d7kq]` (2026-07-20, surfaced by `/docs-distill-memory`), executed autonomously via `/fab-new d7kq` after a validity check confirmed the staleness:

> docs/site/skill.md exit-code section is stale — it still claims usage errors exit 1 and that the toolkit 0/1/2 convention is "not yet implemented (deferred, backlog [xvsj])", but 260717-xvsj shipped the convention (usage errors now exit 2). The bundle now lies to agents branching on exit codes, violating its own never-lie requirement (docs/memory/cli/skill.md). Fix the exit-code bullet in docs/site/skill.md, re-run scripts/sync-skill.sh, and commit the refreshed embed copy src/cmd/idea/skill/skill.md (the drift-guard test enforces the sync).

Validity was re-verified at intake time: `docs/site/skill.md:72-75` still carries the stale claim; the installed binary exits `2` on a bad flag and `1` on a no-match query; change `260717-xvsj-usage-error-exit-codes` merged as PR #35 ("fix: Adopt Toolkit Usage-Error Exit-Code Convention").

## Why

1. **Problem**: The `idea skill` bundle (canonical source `docs/site/skill.md`, embedded verbatim in the binary) documents the tool's exit-code contract for agents. Its exit-code bullet still describes the pre-xvsj world — "`1` for every error including usage/arg errors. Only `shell-init` exits `2` … The toolkit's `0`/`1`/`2` usage-error convention is **not yet implemented** here (deferred, backlog `[xvsj]`)". Since 260717-xvsj shipped, actual behavior is `0` success / `1` operational failure / `2` usage error, tree-wide.
2. **Consequence if unfixed**: The bundle actively lies to agents branching on exit codes — it tells them NOT to treat `2` as "usage error" when that is now exactly what `2` means. This violates the bundle's own binding requirement in `docs/memory/cli/skill.md` ("MUST document idea's **actual** exit-code behavior … never an aspirational or outdated contract, so it never lies to an agent branching on exit codes"). It also dangles a pointer to backlog `[xvsj]` as if still open.
3. **Approach**: Rewrite the one stale bullet in the canonical bundle to state the shipped convention with accurate class membership (sourced from `docs/memory/cli/structure.md` § Exit-code convention), re-run `scripts/sync-skill.sh`, and commit the refreshed embed copy. No code changes — the implementation is already correct; only the doc drifted.

## What Changes

### docs/site/skill.md — replace the stale exit-code bullet

Current text (lines 72–75, in `## Output & exit-code contracts`):

```markdown
- **Exit codes (actual behavior today): `0` on success, `1` for every error including usage/arg
  errors. Only `shell-init` exits `2`** (missing/unsupported shell arg). The toolkit's `0`/`1`/`2`
  usage-error convention is **not yet implemented** here (deferred, backlog `[xvsj]`) — branch on
  `0` vs non-zero, not on `2` meaning "usage error".
```

Replace with a bullet stating the shipped convention. Proposed replacement (wording may be lightly polished at apply, but the factual content is fixed):

```markdown
- **Exit codes follow the toolkit convention: `0` success, `1` operational failure, `2` usage
  error.** Malformed invocations exit `2` — unknown flags, wrong argument counts, a missing or
  unsupported `shell-init` shell, and the `--system`+`--main` conflict. Well-formed invocations
  that fail exit `1` — consent refusals (`rm`/`prune` without `--yes`), no-match or ambiguous
  queries, `fmt --check` on a non-canonical file, and I/O failures. Branching on `2` to detect
  usage errors is supported.
```

Factual class membership (ground truth, from `docs/memory/cli/structure.md` § Exit-code convention, shipped by 260717-xvsj):

- **Exit 2 (usage)**: flag-parse errors (via `SetFlagErrorFunc`), arg-count rejections at all 12 wrapped `Args:` sites, `shell-init` missing/unsupported shell, `--system`+`--main` conflict.
- **Exit 1 (operational, deliberately unchanged)**: `rm`/`prune` consent refusals, no-match/ambiguous-match query errors (`RequireSingle`), `fmt --check` on a non-canonical file, file-I/O/editor/git-resolution failures.
- Unknown first words are never errors — the bare-text shorthand captures them as ideas.

No other content in the bundle changes. The `fmt --check` exits-1 line in `## Gotchas` (line 93) remains correct (operational class) and is untouched. The bundle stays well under the 150-line budget (currently 95 lines; the replacement adds ~2 lines).

### scripts/sync-skill.sh + src/cmd/idea/skill/skill.md — re-sync the embed copy

After editing the canonical file, run `scripts/sync-skill.sh` and commit the refreshed byte-identical embed copy `src/cmd/idea/skill/skill.md`. The drift-guard test `TestSkillEmbedMatchesCanonical` and the line-budget guard `TestSkill_LineBudget` (both in `src/cmd/idea/skill_test.go` area) enforce this on `go test ./...`.

### fab/backlog.md — item lifecycle

Backlog item `[d7kq]` is keyed to this change via `--change-id`; its checkbox flips through the normal fab lifecycle (archive-time marking). No manual backlog edit is part of this change.

## Affected Memory

None — no memory file changes. `docs/memory/cli/skill.md` already states the correct contract (its Requirement "Bundle is a bounded usage briefing…" was updated at 260717-xvsj to require documenting the 0/1/2 convention); this change brings the canonical artifact into conformance with existing memory. Hydrate is expected to be a no-op for memory content (index regeneration only if descriptions change, which they should not).

## Impact

- `docs/site/skill.md` — one bullet rewritten (canonical bundle; renders at https://shll.ai/idea/skill).
- `src/cmd/idea/skill/skill.md` — regenerated byte-identical embed copy (changes `idea skill` output in the next release binary).
- No Go source changes, no behavior changes, no new tests. Existing drift-guard + budget tests verify the sync.
- Verification: `go test ./...` (drift guard, line budget), plus a manual read of the new bullet against `docs/memory/cli/structure.md` § Exit-code convention.

## Open Questions

None.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is the single exit-code bullet in `docs/site/skill.md` § Output & exit-code contracts; all other bundle content untouched | Backlog text names exactly this bullet; validity check confirmed the rest of the bundle is accurate | S:90 R:90 A:95 D:90 |
| 2 | Certain | Replacement bullet documents 0/1/2 with class membership sourced verbatim from `docs/memory/cli/structure.md` § Exit-code convention | Memory is the authoritative post-implementation record of xvsj; binary behavior spot-checked (bad flag → 2, no-match → 1) | S:85 R:90 A:95 D:90 |
| 3 | Certain | Re-run `scripts/sync-skill.sh` and commit the refreshed `src/cmd/idea/skill/skill.md`; rely on existing drift-guard + 150-line budget tests for verification | Mechanism documented in `docs/memory/cli/skill.md` and named in the backlog item itself | S:90 R:95 A:95 D:95 |
| 4 | Confident | Affected Memory is empty — `docs/memory/cli/skill.md` already requires the correct contract; hydrate needs no memory edits | Memory was updated at xvsj; this change conforms the artifact to memory, not vice versa | S:75 R:85 A:80 D:75 |
| 5 | Confident | The new bullet keeps agent-facing branching advice but inverts it: branching on `2` = usage error is now supported; the `[xvsj]` deferral reference is dropped | The old bullet's advice existed to warn agents off `2`; post-xvsj the useful advice is the opposite | S:70 R:85 A:80 D:80 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
