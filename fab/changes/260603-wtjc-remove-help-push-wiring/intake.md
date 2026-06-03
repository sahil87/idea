# Intake: Remove the help-collection push wiring (shll.ai now pulls)

**Change**: 260603-wtjc-remove-help-push-wiring
**Created**: 2026-06-03
**Status**: Draft

## Origin

<!-- One-shot invocation via /fab-new. The directive to implement is hosted in the
     shll.ai repo and was read at intake time. -->

> There's an update in the way we integrate with shll.ai. To understand it read
> https://github.com/sahil87/shll.ai/blob/main/docs/specs/help-dump-contract.md#teardown-directive-paste-to-a-tool-repo-agent .
> Implement the change.

The referenced section is a ready-to-paste task block, **"Task: Remove the help-collection push wiring (shll.ai now pulls)"**. Its verbatim instructions drive this change:

> shll.ai now **pulls** the data itself — a scheduled job there `brew install`s the tool, runs `<tool> help-dump`, and commits the captured JSON. **This repo no longer pushes anything.** Your job in this change is to remove the now-dead push wiring, in a single PR.
>
> 1. **Delete the producer CI** — the workflow step(s)/job that walk the command tree and write the `help/*.json` file.
> 2. **Delete the PR-opening step** that opened a pull request into `sahil87/shll.ai`.
> 3. **Delete the auto-merge wiring** for that PR.
> 4. **Remove `SHLLAI_TOKEN` usage** from the workflows. Then remove the `SHLLAI_TOKEN` repo secret itself **only after confirming it is not used anywhere else** (grep the whole repo for `SHLLAI_TOKEN` first; if anything other than the help-push wiring references it, leave the secret in place and flag it).
> 5. If deleting the above leaves a whole workflow file with no remaining purpose, delete the file; if the help-push was one job inside a larger workflow, remove just that job and leave the rest intact.
>
> **Critical invariant — do NOT touch the `help-dump` command.** … Keep it working exactly as-is — this change removes only the *transport* (the CI that pushed the output), never the command that *produces* it. … keep emitting `schema_version: 1` as today.

This teardown is the inverse of backlog item `[nnsn]` (the original push feature, shipped in PR #5 / change `260602-nnsn-help-dump-shll-ai`). `nnsn` should be marked superseded by this change.

## Why

1. **Problem**: shll.ai inverted its data-collection model. It used to *receive* `idea`'s command-reference JSON via a push: this repo's release CI walked the Cobra tree, wrote `help/idea.json`, and opened an auto-merged PR into `sahil87/shll.ai` using the `SHLLAI_TOKEN` PAT. shll.ai now **pulls** instead — a scheduled job there `brew install`s `idea`, runs `idea help-dump`, and commits the JSON on its own schedule (and on-demand via `workflow_dispatch`). The push step in `idea`'s `release.yml` is now dead weight: every release still clones shll.ai and (when content changed) opens a PR that races against / duplicates shll.ai's own pull.

2. **Consequence of inaction**: We keep a cross-repo write path and a `sahil87` PAT (`SHLLAI_TOKEN`) live for no reason — an unnecessary credential and blast-radius surface. Releases keep doing useless work (clone shll.ai, diff, maybe open a PR) that can produce confusing duplicate/competing PRs alongside shll.ai's pull. Dead wiring also rots: it implies a contract that no longer holds.

3. **Why this approach (full removal of the push step) over alternatives**: The directive is explicit — remove the transport, keep the producer. The push step is entirely self-contained (one named step at the end of `release.yml`, `continue-on-error: true`), so deleting it is clean and low-risk. We do **not** disable-but-keep (commenting it out leaves the dead PAT reference and rots); we do **not** touch `help-dump` (it is now the single contract surface shll.ai pulls). Safe to do now because shll.ai's pull workflow is already live and proven — there is no refresh gap.

## What Changes

### 1. Remove the push step from `.github/workflows/release.yml`

Delete the **entire** final step, `- name: Dump help tree and PR to shll.ai` (currently lines 138–185 of `release.yml`). This single step is the complete push wiring and covers directive items 1, 2, and 4:

- **Producer (item 1)**: `dist/idea-linux-amd64/idea help-dump > help/idea.json` + the `python3` JSON-validation guard.
- **PR-opening (item 2)**: the shll.ai clone, the per-version branch `help-dump/idea-${version}`, the `captured_at`-stripped no-op guard, the commit (authored as `sahil87`), the `git push`, and the `gh pr create --repo sahil87/shll.ai`.
- **`SHLLAI_TOKEN` usage (item 4, workflow side)**: the step's only env line, `GH_TOKEN: ${{ secrets.SHLLAI_TOKEN }}` (line 144). After deletion, `SHLLAI_TOKEN` is referenced by **no** workflow.

The step is `continue-on-error: true` and ordered last (after Cross-compile, GitHub Release, and Homebrew tap update), so removing it cannot affect any other step. Nothing above it depends on `help/idea.json` or on the shll.ai clone.

**Auto-merge wiring (item 3)**: there is **nothing to delete in this repo**. Auto-merge was never owned here — it lives in shll.ai's `.github/workflows/help-automerge.yml`. The current step even documents this (line 185: `# NO gh pr merge here — shll.ai's help-automerge.yml owns merging.`). Item 3 is satisfied by removing the PR-opening step (item 2); no further local change.

**File-vs-job decision (item 5)**: `release.yml` is a larger workflow (cross-compile → GitHub Release → release-notes base tag → Homebrew tap → help-push). The help-push is one step inside it, not the whole file. Per item 5, remove **just that step** and leave the rest of `release.yml` intact. Do **not** delete the file.

After removal, `release.yml`'s last step is **"Update Homebrew tap"**.

### 2. `SHLLAI_TOKEN` repo secret (item 4, secret side) — flag, do not silently drop

Grep across the whole repo for `SHLLAI_TOKEN` before/after the change. Findings (already verified at intake):

- `.github/workflows/release.yml:144` — the push step being deleted. **Only live workflow use.** Gone after change 1.
- `docs/memory/release/pipeline.md` (3 mentions) — **documentation only**, describing the now-removed step. Updated in change 3 below, not a blocker.
- `fab/changes/260602-nnsn-...` and `fab/backlog.md` — historical change artifacts / backlog text. Not executable references.

Conclusion: after change 1, no **workflow** references `SHLLAI_TOKEN`. The repo secret can be safely retired. **However, deleting a GitHub repo secret is a console/CLI action outside the code change** (`gh secret delete SHLLAI_TOKEN --repo sahil87/idea`), not something a PR can do. So this change:
- Removes all in-code/workflow usage (change 1).
- **Flags** the manual follow-up in the PR description and ship notes: "After merge, delete the `SHLLAI_TOKEN` repo secret — no workflow references it anymore." We do not assume it's deleted; we surface it.

### 3. Update `docs/memory/release/pipeline.md`

The memory file documents the deleted step in detail. Update it to reflect the pull model:

- Remove the **"Help-dump → shll.ai command reference"** subsection (currently the block describing the dump → validate → PR-to-shll.ai step, the three auto-merge guards, and the PR-not-push rationale).
- Remove the `SHLLAI_TOKEN` paragraph from the **## Secrets** section (it documents a secret no longer used by any workflow).
- Update the **## File index** line for `release.yml` to drop "help-dump PR to shll.ai" from its description (it becomes: cross-compile, GitHub Release, Homebrew tap update).
- Add a short note that shll.ai now **pulls** `idea`'s help via `idea help-dump` on its own schedule — `idea`'s release no longer pushes. Cross-reference the still-present `help-dump` command in `../cli/structure.md` so the surviving contract surface stays documented.

### 4. Preserve `help-dump` command and its test (critical invariant — NO change)

- `src/cmd/idea/help_dump.go` — the hidden `help-dump` subcommand. **Untouched.** Still emits `{tool, version, schema_version: 1, root}` (plus `captured_at` as today — the directive's note about dropping `captured_at` is an explicitly **out-of-scope future** schema enrichment; this teardown keeps `schema_version: 1` exactly as today).
- `src/cmd/idea/help_dump_test.go` — already a comprehensive contract test (envelope incl. `schema_version: 1` and non-empty `version`, root fields, completion/help/help-dump filtering, leaf `commands: []` serialization, default-flags text). The directive says "if you have a test, keep it" — we keep it as-is. **No new test needed**; the contract surface is already protected independent of the (now-removed) push CI. This satisfies the directive's test requirement without modification.

## Affected Memory

- `release/pipeline.md`: (modify) Remove the help-dump→shll.ai step subsection and the `SHLLAI_TOKEN` secret paragraph; note that shll.ai now pulls via `idea help-dump`; trim the `release.yml` file-index description.
- `cli/structure.md`: (modify) Correct the push-model phrasing to the pull model (lines ~60, 79, 85, 116): help-dump is now *pulled* by shll.ai (`brew install` + `idea help-dump` on its own schedule), not published by a release-side CI step. The `help-dump` command/contract itself is unchanged — only the transport description. <!-- Originally scoped (none) at intake; the review stage found these lines contradicted the updated pipeline.md, so the fix was folded into this change. -->

> **Scope note**: `cli/structure.md` was originally scoped `(none)` at intake (on the assumption it documented only the command/contract, not the transport). Review found three lines that *did* describe the now-removed push transport and contradicted the updated `pipeline.md`, so the correction was made within this change. This note reconciles the artifact with the actual diff.

## Impact

- **CI**: `.github/workflows/release.yml` — one step removed (~48 lines). Remaining steps (cross-compile, GitHub Release, release-notes base, Homebrew tap) unaffected; they have no dependency on the deleted step.
- **Secrets**: `SHLLAI_TOKEN` becomes unreferenced by any workflow. Repo-secret deletion flagged as a manual post-merge step (cannot be done from the PR).
- **Source code**: none. `help-dump` command and test are explicitly preserved (critical invariant).
- **Docs**: `docs/memory/release/pipeline.md` and `docs/memory/cli/structure.md` updated to the pull model.
- **Downstream (shll.ai)**: none from this repo — shll.ai's live pull workflow already refreshes `idea`'s help independently, so retiring the push leaves no gap.
- **Backlog**: mark `[nnsn]` superseded by this teardown (the push feature it tracked is being removed).

## Open Questions

<!-- None blocking. The directive is explicit and the wiring is self-contained. -->

- (none)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The push wiring is exactly the single `Dump help tree and PR to shll.ai` step (release.yml:138–185); remove the whole step | Verified by reading release.yml — one self-contained `continue-on-error` step at the end; grep confirms it is the sole workflow user of `SHLLAI_TOKEN`/`shll.ai` | S:95 R:80 A:95 D:95 |
| 2 | Certain | Do NOT touch `help-dump` command or `help_dump.go`/`help_dump_test.go` | Directive's explicit critical invariant; `help-dump` is now the single contract surface shll.ai pulls | S:98 R:90 A:98 D:98 |
| 3 | Certain | No auto-merge wiring exists to delete in this repo (item 3) | Auto-merge lives in shll.ai's `help-automerge.yml`; release.yml:185 documents "NO gh pr merge here"; grep finds no automerge logic locally | S:95 R:85 A:95 D:90 |
| 4 | Certain | Remove just the step, keep `release.yml` (item 5) | help-push is one step inside a multi-purpose release workflow; cross-compile/Release/Homebrew remain | S:95 R:80 A:95 D:95 |
| 5 | Confident | Existing `help_dump_test.go` already satisfies the directive's "add a test if missing" — add nothing | Test asserts exit 0, valid JSON, `tool`/`schema_version`/`version`, filtering, leaf arrays — exceeds the directive's minimal bar | S:85 R:90 A:90 D:85 |
| 6 | Confident | Repo-secret deletion is a manual post-merge action; this change removes usage + flags the deletion, does not assume it done | Deleting a GitHub repo secret needs `gh secret delete`/console, outside a code PR; directive says remove usage then delete secret only after confirming no other use | S:80 R:70 A:90 D:85 |
| 7 | Confident | `captured_at` stays emitted; keep `schema_version: 1` | Directive marks dropping `captured_at` / schema enrichment as explicit future out-of-scope work — "keep emitting schema_version: 1 as today" | S:90 R:75 A:90 D:90 |
| 8 | Confident | Update `docs/memory/release/pipeline.md` to remove the step + `SHLLAI_TOKEN` docs and note the pull model | Memory must track reality; pipeline.md currently documents the deleted step and secret in detail | S:80 R:75 A:85 D:85 |
| 9 | Confident | Mark backlog `[nnsn]` superseded | This teardown removes the push feature `nnsn` introduced; leaving it open misrepresents state | S:75 R:85 A:80 D:80 |

9 assumptions (4 certain, 5 confident, 0 tentative, 0 unresolved).
