# Plan: Remove the help-collection push wiring (shll.ai now pulls)

**Change**: 260603-wtjc-remove-help-push-wiring
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- This is a `change_type: ci` teardown. It removes dead push wiring and updates
     the memory that documents it. No Go source changes (critical invariant: the
     help-dump command and its test are preserved exactly). -->

### CI: Release Workflow Push-Wiring Removal

#### R1: The release workflow MUST NOT contain the help-dump push step
The `Dump help tree and PR to shll.ai` step (the producer + PR-opening + `SHLLAI_TOKEN` usage) SHALL be removed in its entirety from `.github/workflows/release.yml`. After removal, the workflow's last step SHALL be `Update Homebrew tap`.

- **GIVEN** `.github/workflows/release.yml` currently ends with a `continue-on-error: true` step named `Dump help tree and PR to shll.ai` (lines 138–185)
- **WHEN** the step is deleted
- **THEN** no step named `Dump help tree and PR to shll.ai` exists in the file
- **AND** the final step is `Update Homebrew tap`
- **AND** the file ends with a single clean trailing newline

#### R2: No workflow MUST reference `SHLLAI_TOKEN` or `shll`
After R1, the `.github/` tree SHALL contain no reference to `SHLLAI_TOKEN` and no reference to `shll` (no active workflow mentions shll.ai). The repo secret itself is a manual post-merge deletion (out of code scope) and is flagged, not deleted, here.

- **GIVEN** `release.yml:144` was the only workflow reference to `SHLLAI_TOKEN` and the deleted step was the only `shll` mention under `.github/`
- **WHEN** `grep -rn "SHLLAI_TOKEN" .github/` and `grep -rn "shll" .github/` are run after R1
- **THEN** both return nothing (exit non-zero / no matches)

#### R3: Surviving release steps MUST remain byte-for-byte intact and YAML-valid
The cross-compile, "Create GitHub Release", release-notes base-tag, and "Update Homebrew tap" steps SHALL be unchanged. The file SHALL remain valid YAML.

- **GIVEN** `release.yml` has cross-compile, GitHub Release, release-notes base, and Homebrew tap steps above the deleted step
- **WHEN** the file is parsed (e.g. `yaml.safe_load`)
- **THEN** parsing succeeds and the three named steps are present and unchanged

### Docs: Release Pipeline Memory Update

#### R4: The release pipeline memory MUST describe the pull model, not the removed push
`docs/memory/release/pipeline.md` SHALL drop the "Help-dump → shll.ai command reference" subsection (dump→validate→PR, the three auto-merge guards, the PR-not-push rationale), drop the `SHLLAI_TOKEN` paragraph from `## Secrets`, and trim the `release.yml` file-index description. It SHALL add a note that shll.ai now PULLS idea's help via `idea help-dump` on its own schedule (idea's release no longer pushes), and SHALL keep a cross-reference to the surviving `help-dump` command in `../cli/structure.md`.

- **GIVEN** `pipeline.md` documents the now-removed push step, the `SHLLAI_TOKEN` secret, and lists "help-dump PR to shll.ai" in the file index
- **WHEN** the file is updated to the pull model
- **THEN** the "Help-dump → shll.ai command reference" subsection is gone
- **AND** the `SHLLAI_TOKEN` paragraph is gone from `## Secrets`
- **AND** the `release.yml` file-index line reads "cross-compile, GitHub Release, Homebrew tap update" (no shll.ai push)
- **AND** a note states shll.ai pulls via `idea help-dump`, idea no longer pushes
- **AND** a cross-reference to the `help-dump` command in `../cli/structure.md` is present

### Source: Critical Invariant — help-dump Preserved

#### R5: The `help-dump` command and its test MUST NOT be modified
`src/cmd/idea/help_dump.go` and `src/cmd/idea/help_dump_test.go` SHALL be byte-for-byte unchanged. The command SHALL still build, pass its test, and emit valid JSON with `tool: "idea"` and `schema_version: 1`.

- **GIVEN** the help-dump command is the single contract surface shll.ai pulls
- **WHEN** the teardown is applied and `cd src && go build ./... && go test ./cmd/idea/` is run
- **THEN** build and test pass with no changes to `help_dump.go`/`help_dump_test.go`
- **AND** `go run ./cmd/idea help-dump` emits JSON with `tool == "idea"` and `schema_version == 1`

### Backlog: Mark Superseded

#### R6: Backlog item `[nnsn]` MUST be marked superseded by this teardown
`fab/backlog.md` item `[nnsn]` (the original push feature) SHALL carry an appended note that it is superseded by `260603-wtjc-remove-help-push-wiring`, keeping the constitutional line format (`- [ ] [id] YYYY-MM-DD: text`) round-trip-safe.

- **GIVEN** `[nnsn]` is the backlog item that introduced the now-removed push wiring
- **WHEN** the backlog is updated
- **THEN** the `[nnsn]` line text ends with a `(SUPERSEDED by 260603-wtjc-remove-help-push-wiring: shll.ai now pulls; push wiring removed)` note
- **AND** the line still matches `- [ ] [id] YYYY-MM-DD: text` and the rest of the line is intact

### Non-Goals

- Deleting the `SHLLAI_TOKEN` repo secret — that is a manual `gh secret delete` / console action outside a code PR; flagged for post-merge, not performed here.
- Modifying `help-dump`'s output schema (e.g. dropping `captured_at`) — explicitly out-of-scope future work; `schema_version: 1` is preserved as-is.
- Editing `docs/memory/cli/structure.md` — the intake scopes it as "(none)"; the help-dump command/contract it documents is unchanged.

### Design Decisions

1. **Remove the whole step, keep the file**: The push wiring is one self-contained `continue-on-error` step at the end of a multi-purpose release workflow. — *Why*: directive item 5 — remove just the dead step, leave cross-compile/Release/Homebrew intact. — *Rejected*: deleting `release.yml` (it has surviving purpose); commenting the step out (leaves the dead PAT reference and rots).

### Deprecated Requirements

#### Help-dump push transport (from `[nnsn]` / change `260602-nnsn-help-dump-shll-ai`)
**Reason**: shll.ai inverted to a pull model — it `brew install`s idea and runs `idea help-dump` on its own schedule. The release-side push (clone shll.ai, diff, open auto-merged PR via `SHLLAI_TOKEN`) is now dead weight that races/duplicates shll.ai's pull.
**Migration**: shll.ai's live pull workflow already refreshes idea's help. The producer (`help-dump` command) is preserved as the single contract surface shll.ai pulls.

## Tasks

### Phase 1: CI Teardown

- [x] T001 Delete the entire `- name: Dump help tree and PR to shll.ai` step (lines 138–185, including its `continue-on-error`, `env:`/`GH_TOKEN` block, and full `run:` script) from `.github/workflows/release.yml`; ensure the last step becomes `- name: Update Homebrew tap` and the file ends with a single clean trailing newline <!-- R1 -->

### Phase 2: Docs Memory Update

- [x] T002 In `docs/memory/release/pipeline.md`: remove the "**Help-dump → shll.ai command reference**" subsection (the dump→validate→PR block, the three auto-merge guards, the "Why PR, not direct push" paragraph); remove the `SHLLAI_TOKEN` paragraph from `## Secrets`; change the `release.yml` `## File index` line to drop "help-dump PR to shll.ai"; add a note that shll.ai now PULLS idea's help via `idea help-dump` on its own schedule (idea's release no longer pushes) and keep the cross-reference to the surviving `help-dump` command in `../cli/structure.md` <!-- R4 -->

### Phase 3: Backlog

- [x] T003 In `fab/backlog.md`, append ` (SUPERSEDED by 260603-wtjc-remove-help-push-wiring: shll.ai now pulls; push wiring removed)` to the `[nnsn]` line's text, leaving the rest of the line and its `- [ ] [id] YYYY-MM-DD:` prefix intact <!-- R6 -->

### Phase 4: Verification (no source changes)

- [x] T004 Verify `grep -rn "SHLLAI_TOKEN" .github/` and `grep -rn "shll" .github/` both return nothing; confirm `release.yml` still has cross-compile, "Create GitHub Release", and "Update Homebrew tap" steps and parses as valid YAML <!-- R2 R3 -->
- [x] T005 Verify the critical invariant: `src/cmd/idea/help_dump.go` and `help_dump_test.go` are unchanged; run `cd src && go build ./... && go test ./cmd/idea/` (must pass) and `go run ./cmd/idea help-dump` emits JSON with `tool == "idea"` and `schema_version == 1` <!-- R5 -->

## Execution Order

- T001 → T004 (the grep/YAML checks verify T001)
- T005 is independent of T001–T003 (asserts no Go change); run after edits to confirm the invariant held

## Acceptance

### Functional Completeness

- [x] A-001 R1: `.github/workflows/release.yml` contains no `Dump help tree and PR to shll.ai` step and its last step is `Update Homebrew tap`
- [x] A-002 R4: `docs/memory/release/pipeline.md` no longer documents the push step or `SHLLAI_TOKEN`, the file-index `release.yml` line drops "help-dump PR to shll.ai", and it notes the pull model with a cross-reference to `help-dump` in `../cli/structure.md`
- [x] A-003 R6: `fab/backlog.md` `[nnsn]` line carries the SUPERSEDED note and remains format-valid

### Removal Verification

- [x] A-004 R2: `grep -rn "SHLLAI_TOKEN" .github/` and `grep -rn "shll" .github/` both return nothing
- [x] A-005 R5: `src/cmd/idea/help_dump.go` and `src/cmd/idea/help_dump_test.go` are byte-for-byte unchanged (no Go source modified)

### Behavioral Correctness

- [x] A-006 R3: `release.yml` parses as valid YAML and still contains the cross-compile, "Create GitHub Release", and "Update Homebrew tap" steps, byte-for-byte intact
- [x] A-007 R5: `cd src && go build ./... && go test ./cmd/idea/` passes, and `go run ./cmd/idea help-dump` emits JSON with `tool == "idea"` and `schema_version == 1`

### Scenario Coverage

- [x] A-008 R5: The existing `help_dump_test.go` (envelope/`schema_version`/filtering contract) passes unmodified, proving the contract surface is protected independent of the removed push CI

### Code Quality

- [x] A-009 Pattern consistency: Edits to `release.yml`, `pipeline.md`, and `backlog.md` follow the surrounding YAML/Markdown style; no stray whitespace or broken structure
- [x] A-010 No unnecessary duplication: No commented-out dead wiring left behind; the step is fully removed, not disabled

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Manual post-merge follow-up (flagged, not done here): `gh secret delete SHLLAI_TOKEN --repo sahil87/idea` — no workflow references it after this change.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Delete the entire `Dump help tree and PR to shll.ai` step (release.yml:138–185), keep the rest of the file | Verified by reading release.yml — one self-contained `continue-on-error` step at the end; sole workflow user of `SHLLAI_TOKEN`/`shll`; directive item 5 says remove just the step | S:95 R:80 A:95 D:95 |
| 2 | Certain | Do NOT touch `help_dump.go`/`help_dump_test.go` (byte-for-byte) | Directive's explicit critical invariant; help-dump is the single contract surface shll.ai pulls; SHAs recorded pre-change to confirm no drift | S:98 R:90 A:98 D:98 |
| 3 | Certain | No auto-merge wiring exists locally to delete (directive item 3) | Auto-merge lives in shll.ai's `help-automerge.yml`; release.yml:185 documents "NO gh pr merge here"; grep finds no local automerge logic | S:95 R:85 A:95 D:90 |
| 4 | Confident | Update `docs/memory/cli/structure.md` to the pull model (lines ~60, 79, 85, 116) | Originally scoped `(none)` at intake on the assumption structure.md documented only the command/contract. Review found three lines that *did* describe the now-removed push transport and contradicted the freshly-updated `pipeline.md`, so the correction was folded into this change to keep the two memory files consistent. Transport phrasing only — the `help-dump` command/contract description is unchanged | S:85 R:80 A:85 D:80 |
| 5 | Confident | Repo-secret deletion is a manual post-merge action; remove usage + flag deletion, do not assume done | Deleting a GitHub repo secret needs `gh secret delete`/console, outside a code PR; directive says remove usage then delete secret only after confirming no other use | S:80 R:70 A:90 D:85 |
| 6 | Confident | Mark backlog `[nnsn]` superseded via an appended text note, preserving the `- [ ] [id] YYYY-MM-DD: text` format | This teardown removes the push feature `nnsn` introduced; constitution Principle I requires round-trip-safe line format, so append to text rather than restructure the line | S:80 R:85 A:85 D:82 |

6 assumptions (3 certain, 3 confident, 0 tentative).
