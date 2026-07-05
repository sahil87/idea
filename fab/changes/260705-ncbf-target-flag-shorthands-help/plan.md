# Plan: Target Flag Shorthands + Targets-First Root Help

**Change**: 260705-ncbf-target-flag-shorthands-help
**Intake**: `intake.md`

## Requirements

### CLI: Root Persistent Target Flag Shorthands

#### R1: `-m` shorthand for `--main`
The root persistent `--main` flag SHALL be registered with the single-letter shorthand `-m` via `BoolVarP` on `root.PersistentFlags()` in `src/cmd/idea/main.go`. The usage string SHALL remain byte-identical to the current `--main` usage string. The shorthand SHALL be inherited by every subcommand (persistent flag) with no collision.

- **GIVEN** a git repo with a linked worktree and a main-worktree backlog
- **WHEN** the user runs `idea -m <text>` (or any subcommand with `-m`)
- **THEN** the behavior is identical to passing `--main` — the main worktree's backlog is targeted
- **AND** `idea --main <text>` continues to work unchanged (long form preserved, fully backwards-compatible)

#### R2: `-s` shorthand for `--system`
The root persistent `--system` flag SHALL be registered with the single-letter shorthand `-s` via `BoolVarP` on `root.PersistentFlags()` in `src/cmd/idea/main.go`. The usage string SHALL remain byte-identical to the current `--system` usage string. The shorthand SHALL be inherited by every subcommand with no collision.

- **GIVEN** any working directory (in or out of a git repo)
- **WHEN** the user runs `idea -s <text>` (or any subcommand with `-s`)
- **THEN** the behavior is identical to passing `--system` — the system backlog (`~/.config/idea/backlog.md`) is targeted
- **AND** `idea --system <text>` continues to work unchanged

#### R3: `-f` shorthand for `--file`
The root persistent `--file` flag SHALL be registered with the single-letter shorthand `-f` via `StringVarP` on `root.PersistentFlags()` in `src/cmd/idea/main.go`. The usage string SHALL remain byte-identical to the current `--file` usage string. The shorthand SHALL be inherited by every subcommand with no collision. (Decided-and-recorded from intake Assumption 7 / Open Questions — the intake's default-if-unclarified is to include it; see plan Assumption 3.)

- **GIVEN** any working directory
- **WHEN** the user runs `idea -f <path> <text>`
- **THEN** the behavior is identical to passing `--file <path>` — the given file is targeted with the same rooting rules
- **AND** `idea --file <path> <text>` continues to work unchanged

#### R4: Mutual exclusivity enforcement unchanged
The `--main`/`--system` (now `-m`/`-s`) mutual-exclusion enforcement SHALL remain colocated in `idea.ResolveBacklogPath` (`internal/idea/idea.go`), unchanged. No cobra `MarkFlagsMutuallyExclusive` SHALL be added. Passing both selectors (via any mix of long/short forms) SHALL still yield the existing `--system and --main are mutually exclusive; pass only one` error and a non-zero exit.

- **GIVEN** any working directory
- **WHEN** the user runs `idea -s -m <text>` (or `--system --main`, or any long/short mix)
- **THEN** the command exits non-zero with a message containing `mutually exclusive`
- **AND** no backlog path is resolved and no write occurs

### CLI: Targets-First Root Help

#### R5: Targets-first root `Long` help
The root command's `Long` field in `src/cmd/idea/main.go` SHALL be restructured so a `Targets:` block leads the body immediately after the one-line description. The block SHALL present the three backlog-targeting modes in three rows: the default (current worktree), `-m, --main` (main worktree), and `-s, --system` (`~/.config/idea/backlog.md`). The system path SHALL be rendered as the literal constant `~/.config/idea/backlog.md` (not an XDG-conditional description). The `Long` SHALL state that `--main` and `--system` are mutually exclusive. The existing bare-text shorthand line (`Shorthand: "idea <text>" is equivalent to "idea add <text>".`) SHALL be retained.

- **GIVEN** the user runs `idea -h` (or `idea --help`)
- **WHEN** the root help renders
- **THEN** a `Targets:` section appears at the top of the body listing the three modes with the `-m`/`-s` short forms shown
- **AND** the help states `--main` and `--system` are mutually exclusive
- **AND** the bare-text shorthand line is still present

#### R6: `Short` byte-stability
The root command's `Short` field SHALL remain byte-identical to its current value (`Backlog idea management (current worktree; use --main for main worktree)`). Depth is added in `Long` only, per the `Short` vs `Long` convention (public one-liner used by the `Available Commands` sidebar and help-dump).

- **GIVEN** the help-dump / `Available Commands` sidebar consumers
- **WHEN** they read the root `Short`
- **THEN** the string is unchanged from before this change

### Docs: README Short-Form Mentions

#### R7: Additive README short-form mentions
`README.md` SHALL gain additive mentions of the new short forms at the sites where the long forms are already documented — the feature bullets (lines ~12-13), the targeting table (lines ~103-106), and the "why the default favors the current worktree" paragraph (line ~109). Mentions SHALL be additive only (e.g. `--main`/`-m`), with no restructuring of surrounding prose. *(Rework cycle 1)*: the same additive short-form treatment SHALL apply to `docs/site/workflows.md` (published by shll.ai alongside the README), and the stale `$XDG_CONFIG_HOME` claims on the touched lines (README ~109, workflows.md ~31) SHALL be corrected to the pinned `~/.config/idea/backlog.md`.

- **GIVEN** a reader of `README.md`
- **WHEN** they reach a documented `--main`/`--system`/`--file` site
- **THEN** the corresponding short form is mentioned alongside the long form
- **AND** no section is restructured; only short-form tokens are added

### Non-Goals

- No change to path resolution behavior (`ResolveBacklogPath` precedence, errors, fallback) — only flag registration and help text change.
- No help-dump JSON **schema** change. The dumped `text`/`usage` strings gain the new shorthand rendering automatically via cobra; the frozen cross-repo schema is untouched and no shll.ai action is required (it re-renders on its next pull).
- No `MarkFlagsMutuallyExclusive` — enforcement stays in `ResolveBacklogPath`.
- No broad XDG-prose cleanup. *(Narrowed in rework cycle 1)*: the stale `$XDG_CONFIG_HOME` claims ARE corrected on the two doc lines this change already touches (README ~109, docs/site/workflows.md ~31) — the review found the touched README line contradicting the binary's new help. Broader corrections (e.g. `docs/specs/overview.md:38`) remain out of scope.

### Design Decisions

1. **Use `BoolVarP`/`StringVarP` with unchanged usage strings**: preserves byte-stable usage text while adding the shorthand — *Why*: fully backwards-compatible, minimal diff, matches the existing `list.go` `-a` precedent (`BoolVarP(&all, "all", "a", ...)`) — *Rejected*: reworded usage strings (would churn help-dump `text` unnecessarily).
2. **Include `-f` for `--file`**: intake Assumption 7 (Unresolved, deferred under promptless dispatch) records "default if unclarified: include it". Apply decides-and-records per the SRAD apply contract — INCLUDE, graded Confident — *Why*: `-f` is verified conflict-free, consistent with `-m`/`-s`, purely additive; the intake's own recorded default is inclusion — *Rejected*: deferring it (would leave `--file` as the lone shorthand-less selector, contradicting the change's stated consistency goal).
3. **Test via the existing subprocess seam**: equivalence and conflict cases use `buildBinary`/`setupGitRepo`/`systemEnv`/`runSplitEnv` (as `TestSystem_*` do) — *Why*: `-s`/`--system` target a HOME-isolated backlog best exercised end-to-end; mirrors the existing long-form conflict test `TestSystem_ConflictWithMain` — *Rejected*: in-process `newRootCmd()` only (package-level flag vars persist across `newRootCmd()` calls within a process, risking cross-test state leakage for equivalence assertions; subprocess isolation is cleaner and already the established pattern for these flags).

## Tasks

### Phase 1: Core Implementation

- [x] T001 Switch the three root persistent flag registrations in `src/cmd/idea/main.go` from `StringVar`/`BoolVar` to `StringVarP`/`BoolVarP`: `--file` gains `-f`, `--main` gains `-m`, `--system` gains `-s`; keep all three usage strings byte-identical. <!-- R1 --> <!-- R2 --> <!-- R3 -->
- [x] T002 Rewrite the root command's `Long` in `src/cmd/idea/main.go` with a leading `Targets:` block (three rows: default / `-m, --main` / `-s, --system` with the literal `~/.config/idea/backlog.md`), a `--main`/`--system` mutual-exclusion statement, and the retained bare-text shorthand line; leave `Short` byte-unchanged. <!-- R5 --> <!-- R6 --> <!-- rework: must-fix — the Long's --file/-f line claims it "overrides the backlog path within the selected target", but ResolveBacklogPath short-circuits to SystemBacklogPath() before consulting fileFlag, so --file is silently ignored under -s/--system (empirically confirmed). Qualify the claim, e.g. "--file/-f overrides the backlog path within the selected root (ignored with --system)". Help-text-only fix — no behavior change. -->

### Phase 2: Tests

- [x] T003 Add a table-driven equivalence test in `src/cmd/idea/main_test.go` asserting `-m` ≡ `--main` and `-s` ≡ `--system` (and, additively, `-f <path>` ≡ `--file <path>`) — same targeted backlog / same output — using the existing subprocess helpers (`buildBinary`/`setupGitRepo`/`systemEnv`/`runSplitEnv`). <!-- R1 --> <!-- R2 --> <!-- R3 --> <!-- rework: should-fix — current equivalence cases run where the flag is behaviorally identical to the default (-m from the main worktree; -s outside any repo), so they cannot detect a mis-wired shorthand. Strengthen: exercise -m from a linked worktree (git worktree add, per R1's GIVEN) and run the -s case inside a repo (mirroring TestSystem_FlagInsideRepo) so short-vs-long actually diverges from the default path. -->
- [x] T004 Add a conflict test in `src/cmd/idea/main_test.go` asserting `-s -m` together still yields the `mutually exclusive` error and a non-zero exit (mirroring `TestSystem_ConflictWithMain`). <!-- R4 -->

### Phase 3: Docs

- [x] T005 [P] Add additive short-form mentions to `README.md` at the feature bullets (~12-13), the targeting table (~103-106), and the current-worktree-default paragraph (~109) — `-m`/`-s`/`-f` alongside their long forms, no restructuring. <!-- R7 --> <!-- rework: should-fix ×2 — (a) the already-edited README line ~109 still carries the false claim "The system path honors $XDG_CONFIG_HOME"; the path is pinned to ~/.config/idea/backlog.md (XDG ignored, per docs/memory/cli/structure.md) and the line now contradicts the binary's new help — fix the few words on that touched line. (b) docs/site/workflows.md (~lines 18-54) mirrors the same targeting table/prose and is published by shll.ai alongside README — give it the same additive short-form mentions, and fix its stale "$XDG_CONFIG_HOME/idea/backlog.md" claim (~line 31). docs/specs/overview.md:38 stays out of scope (hydrate/separate change). -->

## Execution Order

- T001 blocks T003/T004 (tests exercise the registered shorthands).
- T002 is independent of the tests (help text; no test asserts the new `Long` prose — help_dump_test only checks `-h`/`-v` presence, which is unaffected).
- T005 is independent and parallelizable.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `idea -m` targets the main worktree's backlog identically to `idea --main`; long form still works.
- [x] A-002 R2: `idea -s` targets the system backlog identically to `idea --system`; long form still works.
- [x] A-003 R3: `idea -f <path>` targets the given file identically to `idea --file <path>`; long form still works.
- [x] A-004 R5: `idea -h` root help leads with a `Targets:` block showing the three modes with `-m`/`-s` short forms, states `--main`/`--system` mutual exclusion, renders the system path as `~/.config/idea/backlog.md`, retains the bare-text shorthand line, and makes no false claim about `--file` under `--system`.
- [x] A-005 R7: `README.md` and `docs/site/workflows.md` mention `-m`/`-s`/`-f` additively at the documented long-form sites with no restructuring; the stale XDG claims on the touched lines are corrected.

### Behavioral Correctness

- [x] A-006 R4: `idea -s -m` (and long/short mixes) still exits non-zero with a `mutually exclusive` message; `ResolveBacklogPath` and its enforcement are unchanged (no `MarkFlagsMutuallyExclusive`).
- [x] A-007 R6: root `Short` is byte-identical to its prior value.

### Scenario Coverage

- [x] A-008 R1 R2 R3: a table-driven equivalence test asserts `-m`≡`--main`, `-s`≡`--system`, `-f`≡`--file` via the subprocess seam and passes — with the `-m` case exercised from a linked worktree and the `-s` case inside a repo, so a mis-wired shorthand is detectable.
- [x] A-009 R4: a conflict test asserts the `-s -m` mutual-exclusion error and passes.

### Edge Cases & Error Handling

- [x] A-010 R1 R2 R3: the three usage strings are byte-identical to their pre-change values (only the shorthand column is added by cobra), so help-dump `text` gains only the rendered `-x, --long` shorthand form.

### Code Quality

- [x] A-011 Pattern consistency: new flag registrations follow the existing `BoolVarP`/`StringVarP` precedent (`list.go` `-a`); the new test follows the table-driven / real-temp-dir / subprocess conventions (Constitution V).
- [x] A-012 No unnecessary duplication: the equivalence/conflict tests reuse existing helpers (`buildBinary`/`setupGitRepo`/`systemEnv`/`runSplitEnv`) rather than reimplementing setup.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `None — this change adds new functionality without making existing code redundant` — the root `Long` was rewritten in place (no leftover string/symbol); `resolveFile()` and `ResolveBacklogPath` are untouched; `TestSystem_ConflictWithMain` (`src/cmd/idea/main_test.go:1080`) is NOT made redundant by the new mixed-form conflict test — it covers the long/long pair the new table deliberately omits, completing the 2×2 flag-form matrix.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Register `-m`/`-s`/`-f` via `BoolVarP`/`StringVarP` with byte-unchanged usage strings | Intake Assumptions 1-2 are Certain for `-m`/`-s`; the `P`-variant with unchanged usage is the established `list.go -a` pattern; conflict check verified all three free | S:90 R:75 A:90 D:90 |
| 2 | Certain | Targets-first root `Long` (three-row block, mutual-exclusion statement, literal `~/.config/idea/backlog.md`, retained bare-text line), `Short` byte-stable | Intake Assumptions 3-4 Certain; exact prose is agent latitude; `Short` byte-stability is established convention (memory `cli/structure`) | S:85 R:90 A:85 D:85 |
| 3 | Confident | Include `-f` shorthand for `--file` in this change (decides intake Assumption 7 / Open Question) | Apply decides-and-records the intake's deferred Unresolved point; the intake's own recorded default is include; `-f` is verified conflict-free, purely additive, and consistent with `-m`/`-s`. Confident (not Certain) because it was never user-confirmed and shipped shorthands are permanent public contract | S:60 R:55 A:75 D:70 |
| 4 | Confident | Correct the stale `$XDG_CONFIG_HOME` claim only on lines this change already touches (README ~109, workflows.md ~31); broader XDG corrections stay out of scope | Revised in rework cycle 1: review found the already-edited README line contradicting the new help text this change ships; a few-word fix on touched lines is not "restructuring". `docs/specs/overview.md:38` remains untouched (hydrate/separate change) | S:70 R:85 A:80 D:75 |
| 5 | Certain | Test equivalence/conflict via the existing subprocess seam (`buildBinary`/`systemEnv`/`runSplitEnv`), mirroring `TestSystem_*` | Constitution V (table-driven, real temp dirs, real git repos); subprocess isolation avoids package-level flag-var leakage across in-process `newRootCmd()` calls; matches the existing long-form conflict test's shape | S:80 R:85 A:90 D:85 |
| 6 | Confident | Rework cycle 1 nice-to-have: make `TestTargetFlagShorthands_ConflictWithShortForms` table-driven and add the mixed long/short conflict pairs (`-s --main`, `--system -m`); do NOT fold it with `TestSystem_ConflictWithMain` | Directly covers R4's WHEN ("any long/short mix"); cheap and low-risk. Folding was declined — `TestSystem_ConflictWithMain` (T004, already `[x]`) is a passing long-form test outside the rework annotations, so merging would churn out-of-scope code against the minimal-diff preference | S:70 R:85 A:85 D:80 |

6 assumptions (3 certain, 3 confident, 0 tentative).
