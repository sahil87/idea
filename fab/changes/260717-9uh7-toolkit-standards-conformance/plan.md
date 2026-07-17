# Plan: Toolkit Standards Conformance

**Change**: 260717-9uh7-toolkit-standards-conformance
**Intake**: `intake.md`

## Requirements

<!-- Requirements are grouped by the standard (or principle) they conform the
     repo to. The audit is runtime-enumerated against `shll v0.0.23`; the runtime
     text of `shll standards` is authoritative over the intake snapshot. -->

### help-dump: Machine-help contract conformance

#### R1: help-dump envelope MUST NOT emit `captured_at`
The `help-dump` JSON envelope SHALL be exactly `{tool, version, schema_version, root}` and MUST NOT carry a `captured_at` field. The capture timestamp is owned by shll.ai's puller — a tool cannot know its own capture time.

- **GIVEN** the `idea help-dump` subcommand
- **WHEN** it emits the envelope JSON
- **THEN** the JSON has keys `tool`, `version`, `schema_version`, `root` in that order
- **AND** no `captured_at` key is present anywhere in the output

#### R2: help-dump verification checklist MUST pass
`idea help-dump` SHALL exit 0, write valid JSON to stdout only with stderr empty, exclude `completion`/`help`/hidden commands from the tree, report `version` from the built binary (ldflags, not a literal), and keep `schema_version` at the integer `1`.

- **GIVEN** a freshly built `idea` binary
- **WHEN** `idea help-dump` runs
- **THEN** exit code is 0, stdout is valid JSON, stderr is empty
- **AND** the command tree contains no `completion`, `help`, or hidden node
- **AND** `version` reflects the ldflags-stamped build value and `schema_version == 1`

### help-dump: Test conformance

#### R3: The pinning test MUST assert `captured_at` is absent
The `help-dump` test SHALL stop asserting that `captured_at` parses as RFC3339 (which pins the violation) and instead assert the field is absent from the emitted JSON, while still pinning the surviving envelope contract (exit 0, valid JSON, `tool`, `schema_version`).

- **GIVEN** the `help_dump_test.go` suite
- **WHEN** it validates the envelope
- **THEN** it asserts `captured_at` does not appear in the raw JSON bytes
- **AND** it no longer references the removed `CapturedAt` struct field

### readme-extraction: docs-site structure conformance

#### R4: README MUST link the command reference at the standard's canonical URL
The README's command-reference link SHALL use the standard's absolute form `https://shll.ai/idea/commands/`, verified live to be the canonical rendered page, rather than the `https://shll.ai/tools/idea/commands/` form (a redirect stub).

- **GIVEN** `README.md`'s "Command reference" section
- **WHEN** it points at the generated command reference
- **THEN** the URL is `https://shll.ai/idea/commands/`
- **AND** no `https://shll.ai/tools/idea/commands/` link remains in the README

### principles №1/№5: Consent flag on destructive writes

#### R5: `rm` and `prune` MUST accept `--yes`/`-y` as a consent alias
`idea rm` and `idea prune` SHALL accept `--yes` (with the `-y` shorthand) as a flag-satisfiable consent alias for their existing `--force` consent semantics. `--force` MUST be retained (public CLI surface is a contract — additive only, no renames/removals). Passing either flag satisfies consent identically.

- **GIVEN** `idea rm <query> --yes` (or `-y`) on a matching idea
- **WHEN** the command runs
- **THEN** the deletion proceeds exactly as with `--force`
- **AND GIVEN** `idea prune --yes` (or `-y`)
- **WHEN** the command runs
- **THEN** all done ideas are removed exactly as with `--force`, printing the count
- **AND** `--force` continues to work unchanged on both commands

### principles №5: Dry-run on destructive writes

#### R6: `rm` MUST support `--dry-run` sharing the live match path
`idea rm` SHALL support `--dry-run`: it resolves the match through the same `RequireSingle` path the live delete uses, prints what would be deleted to stdout, and writes nothing. `--dry-run` requires no consent flag (a preview is non-destructive). When `--dry-run` is combined with `--force`/`--yes`, the dry-run preview wins (no deletion).

- **GIVEN** `idea rm <query> --dry-run` matching exactly one idea
- **WHEN** the command runs
- **THEN** the would-be-deleted idea's canonical line is printed to stdout, the backlog file is byte-identical afterward, and exit is 0
- **AND GIVEN** an ambiguous `<query>` under `--dry-run`
- **THEN** the same ambiguity refusal fires as the live path (shared code path), writing nothing

### principles: Audit-recorded verdicts (no code change)

#### R7: The conformance report MUST record every principle's verdict against actual behavior
The report SHALL assess all ten principles against the running binary and disposition each gap as fixed-here (commit pending — filled at ship) or deferred → [backlog-id]. It MUST record the audit judgments the intake called out: `show --json` already ships (PASS, seed corrected); `prune`'s piped free-dry-run satisfies №1's non-TTY refusal contract; the exit-code convention (usage errors → 2) is only partially met and is deferred.

- **GIVEN** the completed audit
- **WHEN** the report is authored
- **THEN** it has one section per standard (`principles`, `help-dump`, `readme-extraction`, `skill`) with PASS or the gaps found
- **AND** each gap names its disposition (fixed here / deferred → backlog id)
- **AND** the report is pinned to the `shll version` row (shll v0.0.23)

### skill: Deferred adoption

#### R8: The `skill` standard MUST be reported deferred, not implemented
`idea skill` SHALL NOT be implemented in this change. The report's skill section reads "deferred, not yet adopted" per the standard's own phased per-repo adoption clause, with a backlog entry as the deferral target.

- **GIVEN** the `skill` standard (binary+repo scope, phased adoption, "no tool ships it yet")
- **WHEN** the report is authored
- **THEN** the skill section reads "deferred, not yet adopted" and references a backlog entry
- **AND** no `skill` subcommand is added to the command tree

### Deferred gaps: Backlog entries in this branch

#### R9: Restructuring-sized gaps MUST be recorded as backlog entries referenced by the report
Gaps too large to fix additively here — the `skill` adoption and the tree-wide usage-error exit-code-2 convention — SHALL be recorded via `idea add` into this worktree's committed `fab/backlog.md`, and their 4-char IDs referenced in the report's disposition column.

- **GIVEN** a deferred gap
- **WHEN** it is recorded
- **THEN** a new line appears in `fab/backlog.md` describing the follow-up work
- **AND** the report's disposition for that gap reads `deferred → [id]`
- **AND** `git diff fab/backlog.md` shows only the intended additive lines

### Verification gate

#### R10: The tree MUST build clean and all tests MUST pass
`go build ./...`, `gofmt -l` (clean), `go vet ./...`, and `go test ./...` SHALL all pass from `src/`. New flags get table-driven tests per Constitution V (real temp dirs, no mocks). If the command tree changed, the help-dump verification checklist MUST be re-run afterward.

- **GIVEN** the completed fixes
- **WHEN** the verification gate runs from `src/`
- **THEN** `go build ./...` succeeds, `gofmt -l` prints nothing, `go vet ./...` is clean, `go test ./...` is green
- **AND** because the command tree changed (new flags → changed `-h` text), `idea help-dump` still passes its checklist (exit 0, valid JSON, no `captured_at`)

### Non-Goals

- Implementing `idea skill` (deferred — phased adoption, R8).
- A tree-wide exit-code-2 usage-error convention (deferred — a partial fix would create a worse inconsistency than the current uniform exit-1; see Design Decisions).
- Any `show --json` work — it already ships and conforms (intake seed #4 was stale; runtime is authoritative).
- An explicit `--dry-run` alias on `prune` — its piped free-dry-run already satisfies the obligation (audit judgment recorded in the report).
- Any dependency changes or restructuring of the query matcher / bare-text shorthand.

### Design Decisions

1. **Exit-code-2 for usage errors → deferred, not partially fixed**: The toolkit convention (Principle №4) is `0` success / `1` operational / `2` usage error; today only `shell-init` returns 2, while all other usage errors (unknown flag, wrong arg count, unknown subcommand) return 1 via `main()`. — *Why deferred*: `SetFlagErrorFunc` cleanly tags flag errors, but cobra does NOT route arg-count/unknown-command errors through it (verified empirically), so a complete fix must tag the `Args` validator seam across the tree plus the command-resolution path — that is restructuring, which the intake scopes out of this change. — *Rejected*: a flag-errors-only partial fix (flags→2, arg errors→1) — rejected because it makes two usage-error classes disagree, which is worse for an agent branching on exit codes than the current uniform 1.
2. **`--yes`/`-y` as an alias, `--force` retained**: Principle №1 names `--yes`/`-y` as the canonical consent flag; the existing `--force` is a public-contract string. — *Why*: additive alias keeps the surface a growing contract (Constitution VI). — *Rejected*: renaming `--force` to `--yes` — breaks the contract.
3. **`rm --dry-run` routes through the shared `RequireSingle` path**: Principle №5 requires a dry-run that shares the real code path. — *Why*: resolving the match with the same function the live delete uses guarantees the preview never drifts from the live behavior (same ambiguity/no-match refusals). — *Rejected*: a separate preview matcher — the exact drift the standard warns against.

## Tasks

### Phase 1: Runtime enumeration & per-standard audit

- [x] T001 Re-run the authoritative enumeration and pin the version: `shll version` (record the `shll` row), `shll standards`, and `shll standards <name>` for every listed entry; if the list or any text differs from the intake snapshot, the runtime text wins and downstream tasks adjust. <!-- R7 -->
- [x] T002 Audit `help-dump` against its "Verifying conformance" checklist verbatim using a freshly built binary (`go build -o … ./cmd/idea` from `src/`): exit 0, valid JSON to stdout only, stderr empty, envelope shape, `completion`/`help`/hidden absent, `version` from ldflags, `schema_version == 1`, and confirm the `captured_at` violation. <!-- R1 -->
- [x] T003 Audit `readme-extraction` against its checklist verbatim over `README.md` + `docs/site/` (head order, tail-heading closure, relative-link/image rules, reserved names, mermaid/theme fragments) and verify the live command-reference URL (`curl` both forms; confirm `https://shll.ai/idea/commands/` is canonical vs. the `/tools/idea/` redirect stub). <!-- R4 -->
- [x] T004 Assess all ten `principles` against actual behavior of every subcommand (`add`, `list`/`ls`, `show`, `done`, `reopen`, `edit`, `rm`, `prune`, `fmt`, `update`, `shell-init`, `help-dump`, root shorthand): non-interactive/TTY handling, stdout/stderr split, `--json`, `--dry-run`, exit codes, error wording, idempotency, output volume — recording each verdict for the report (incl. the `show --json`-already-ships correction, the `prune` non-TTY refusal judgment, and the exit-code-2 gap). <!-- R7 -->

### Phase 2: help-dump fix

- [x] T005 Remove `captured_at` from `src/cmd/idea/help_dump.go`: delete the `CapturedAt string` field from the `helpDump` struct and its `CapturedAt: time.Now()...` population; drop the now-unused `time` import. Envelope stays `{tool, version, schema_version, root}`. <!-- R1 -->
- [x] T006 Invert the test in `src/cmd/idea/help_dump_test.go`: replace the `time.Parse(time.RFC3339, dump.CapturedAt)` assertion with an assertion that `captured_at` is absent from the raw JSON bytes; remove the `CapturedAt` reference and the now-unused `time` import; keep the surviving envelope assertions (exit 0, valid JSON, `tool`, `schema_version`). <!-- R3 -->

### Phase 3: principle-gap additive fixes

- [x] T007 Add `--yes`/`-y` consent alias to `idea rm` in `src/cmd/idea/rm.go`: register a `BoolVarP(&yes, "yes", "y", …)` local flag and treat `force || yes` as consent when calling `idea.Rm`. Keep `--force`. Update the command `Long` to mention `--yes`/`-y`. <!-- R5 -->
- [x] T008 Add `--yes`/`-y` consent alias to `idea prune` in `src/cmd/idea/prune.go`: register `BoolVarP(&yes, "yes", "y", …)` and treat `force || yes` as the immediate-delete path. Keep `--force` and `--full`. Update the `Long`. <!-- R5 -->
- [x] T009 Add `--dry-run` to `idea rm` in `src/cmd/idea/rm.go` + `src/internal/idea/idea.go`: add an `RmPreview(path, query string) (Idea, error)` (or a `dryRun` param on `Rm`) that resolves the match via the same `RequireSingle`/`FilterAll` path and returns the would-be-removed idea WITHOUT saving; the `rm` RunE, when `--dry-run`, prints the canonical `FormatLine` to stdout and writes nothing (dry-run wins over `--force`/`--yes`). Update the `Long`. <!-- R6 -->
- [x] T010 [P] Add table-driven tests (real temp dirs, no mocks — Constitution V): in `src/internal/idea/idea_test.go` cover the `rm` dry-run seam (match found → returns idea, file byte-identical; ambiguous → same refusal as live; no match → error); in `src/cmd/idea/main_test.go` (or the in-process `newRootCmd()` harness) cover `rm --yes`/`-y` deleting like `--force`, `prune --yes`/`-y` deleting like `--force`, and `rm --dry-run` printing to stdout with the backlog unchanged and exit 0. <!-- R5 --> <!-- R6 -->

### Phase 4: README fix

- [x] T011 Fix the command-reference URL in `README.md` (line ~99): change `https://shll.ai/tools/idea/commands/` to `https://shll.ai/idea/commands/`. <!-- R4 -->

### Phase 5: Deferred-gap backlog entries

- [x] T012 From this worktree's repo root, add a backlog entry for `skill`-standard adoption via `idea add "<text>"` (use the installed `idea` binary, or `go run ./cmd/idea` from `src/` if it misbehaves); capture the 4-char ID for the report. Verify with `git diff fab/backlog.md` that only the intended line was added. <!-- R8 --> <!-- R9 -->
- [x] T013 From this worktree's repo root, add a backlog entry for the tree-wide usage-error exit-code-2 convention via `idea add "<text>"`; capture the 4-char ID for the report. Verify with `git diff fab/backlog.md` that only the intended line was added. <!-- R9 -->

### Phase 6: Conformance report

- [x] T014 Author `fab/changes/260717-9uh7-toolkit-standards-conformance/conformance-report.md` per the intake § 7 structure: pinned to `shll v0.0.23`; one section per standard (`principles` with a per-numbered-principle verdict table, `help-dump`, `readme-extraction`, `skill`); each gap dispositioned "fixed here (commit pending — filled at ship)" or "deferred → [backlog-id]" (using the IDs from T012/T013). <!-- R7 --> <!-- R8 -->

### Phase 7: Verification gate

- [x] T015 From `src/`, run the gate: `go build ./...`, `gofmt -l .` (must print nothing), `go vet ./...`, `go test ./...` (green). Fix any failures. <!-- R10 -->
- [x] T016 Because the command tree changed (new flags → changed `-h` text), rebuild the binary and re-run the help-dump verification checklist (exit 0, valid JSON to stdout only, stderr empty, no `captured_at`, `schema_version == 1`). <!-- R2 --> <!-- R10 -->

## Execution Order

- T001–T004 (audit) precede all fixes; T001's runtime enumeration is authoritative and may adjust downstream scope.
- T005 blocks T006 (test asserts the removed field's absence); T006 blocks nothing else.
- T007/T008 (consent alias) and T009 (dry-run) precede T010 (their tests).
- T012/T013 (backlog IDs) block T014 (report references the IDs).
- T015 and T016 run last, after all code and doc changes.

## Acceptance

### Functional Completeness

- [x] A-001 R1: The `idea help-dump` envelope is exactly `{tool, version, schema_version, root}` with no `captured_at` field anywhere in the output.
- [x] A-002 R2: `idea help-dump` passes its verification checklist — exit 0, valid JSON to stdout only, stderr empty, no `completion`/`help`/hidden nodes, `version` from ldflags, `schema_version == 1`.
- [x] A-003 R4: The README's command-reference link is `https://shll.ai/idea/commands/` and no `https://shll.ai/tools/idea/commands/` link remains.
- [x] A-004 R5: `idea rm <query> --yes` and `-y` delete exactly like `--force`; `idea prune --yes` and `-y` prune exactly like `--force`; `--force` still works on both.
- [x] A-005 R6: `idea rm <query> --dry-run` prints the would-be-deleted line to stdout, leaves the backlog byte-identical, and exits 0.
- [x] A-006 R7: `conformance-report.md` has one section per standard, each gap dispositioned, pinned to shll v0.0.23.
- [x] A-007 R8: The report's skill section reads "deferred, not yet adopted" with a backlog reference; no `skill` subcommand exists in the tree.
- [x] A-008 R9: `fab/backlog.md` gained the deferred-gap entries (skill adoption + exit-code-2), their IDs are referenced in the report, and `git diff fab/backlog.md` shows only the intended additive lines.

### Behavioral Correctness

- [x] A-009 R3: `help_dump_test.go` asserts `captured_at` is absent from the raw JSON (no longer parses it as RFC3339) and no longer references the removed `CapturedAt` field.
- [x] A-010 R6: `rm --dry-run` shares the `RequireSingle` match path — an ambiguous query under `--dry-run` produces the same refusal as the live delete, writing nothing.
- [x] A-011 R5: The `--yes`/`-y` flags are additive — `--force` is not renamed or removed on either `rm` or `prune`.

### Scenario Coverage

- [x] A-012 R5: Table-driven tests (real temp dirs) cover `rm --yes`/`-y` and `prune --yes`/`-y` deleting like `--force`.
- [x] A-013 R6: Table-driven tests cover the `rm` dry-run seam (match → file unchanged; ambiguous → refusal; no match → error) and the `rm --dry-run` CLI output contract.

### Edge Cases & Error Handling

- [x] A-014 R6: `rm --dry-run` combined with `--force`/`--yes` still performs no deletion (dry-run wins).
- [x] A-015 R7: The report records the audit judgments — `show --json` already ships (PASS), `prune` piped free-dry-run satisfies №1's non-TTY refusal, and the exit-code-2 gap is deferred with rationale.

### Code Quality

- [x] A-016 Pattern consistency: New flag wiring follows the existing cobra factory / `BoolVarP` / `RunE` patterns in `src/cmd/idea/`; the dry-run seam follows the `internal/idea` logic-vs-`cmd/` split (Constitution IV).
- [x] A-017 No unnecessary duplication: `rm --dry-run` reuses `RequireSingle`/`FormatLine` rather than reimplementing matching or formatting; consent handling reuses the existing `idea.Rm`/`idea.Prune` force path.
- [x] A-018 Readability over cleverness: Consent (`force || yes`) and dry-run branching stay simple and explicit; no magic strings — flag names/usage strings are literals in the factory as elsewhere.
- [x] A-019 Constitution V: All new tests are table-driven against real temp dirs with no filesystem mocks; Constitution VI: stdout stays machine-parseable (dry-run line via `FormatLine`), advisories on stderr.

### Verification Gate

- [x] A-020 R10: `go build ./...`, `gofmt -l .` (empty), `go vet ./...`, and `go test ./...` all pass from `src/`, and `idea help-dump` re-passes its checklist after the tree change.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (Noted, not candidates: `--force` on `rm`/`prune` is now a semantic duplicate of `--yes`, but it is deliberately retained as public CLI contract per R5/Constitution VI — additive only, no removals; the `captured_at` code and its `time` imports were already deleted by this change itself.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Audit target set = the four entries `shll standards` lists at apply (principles, help-dump, readme-extraction, skill @ shll v0.0.23); runtime text is authoritative and matched the intake snapshot exactly | Re-ran `shll version`/`shll standards`/`shll standards <name>` at apply; all matched intake | S:95 R:90 A:95 D:95 |
| 2 | Certain | Remove `captured_at` from the help-dump envelope + invert the pinning test to assert absence | Standard's "rule with teeth"; violation confirmed in the running binary (`captured_at` present); source locations confirmed | S:95 R:85 A:95 D:95 |
| 3 | Certain | README commands URL → `https://shll.ai/idea/commands/` (standard's form); the `/tools/idea/commands/` form is a live redirect stub, standard's form is the canonical 134KB rendered page | Verified live with `curl`: standard form 200 + full content + `<title>Commands</title>`; old form 200 but `<title>Redirecting to…</title>` 345 bytes | S:90 R:90 A:90 D:95 |
| 4 | Certain | `show --json` already ships and conforms (stable `{id,date,status,text}`) — intake seed #4 was stale; NO `show --json` work | Runtime authoritative: `idea show <q> --json` emits the record; `show.go` has had the `--json` flag since the initial port | S:95 R:95 A:95 D:95 |
| 5 | Confident | Add `--yes`/`-y` as a consent alias on `rm` and `prune`, keeping `--force` (additive, no renames) | Principle №1 names `--yes`/`-y`; task lists "a missing flag" as in-scope; `-y` shorthand is free (no collision with `-h`/`-v`/`-f`/`-m`/`-s`/`-a`) | S:75 R:80 A:80 D:70 |
| 6 | Confident | Add `--dry-run` to `idea rm` via a new `internal/idea` preview seam that shares `RequireSingle`; dry-run wins over `--force`/`--yes` | Principle №5 MUST for destructive writes + shared-live-path requirement; the `internal/idea` seam preserves Constitution IV | S:75 R:75 A:80 D:70 |
| 7 | Confident | Exit-code-2 for usage errors is a real gap but DEFERRED to backlog (not fixed here) — a partial flag-only fix would worsen the inconsistency | Verified `SetFlagErrorFunc` catches flag errors but NOT arg-count/unknown-command errors; full fix touches the Args seam tree-wide = structural, which the intake scopes out; partial fix makes usage-error classes disagree | S:70 R:70 A:75 D:65 |
| 8 | Confident | `prune`'s piped free-dry-run satisfies №1's non-TTY refusal contract (no hang, names `--force` on stderr, exits without deleting) — documented design, recorded as an audit PASS, no code change | Behavior verified deliberate + documented (docs/memory/cli/prune.md); №1's intent (no hang, flag named) is met | S:65 R:75 A:65 D:60 |
| 9 | Confident | No explicit `--dry-run` alias on `prune` — its existing piped free-dry-run + interactive confirm already satisfy the obligation; recorded as an audit judgment | Adding a redundant `--dry-run` verb to prune would duplicate the existing dry-run semantics without new capability; intake asked to assess and record either way | S:65 R:80 A:70 D:65 |
| 10 | Confident | Deferred gaps (skill adoption + exit-code-2) recorded via `idea add` into this worktree's committed `fab/backlog.md`; IDs referenced in the report | Repo's demonstrated convention (e3rk, ykwp); entries ride this PR; task says "draft change or issue per this repo's convention" | S:70 R:80 A:80 D:70 |

10 assumptions (4 certain, 6 confident, 0 tentative).
