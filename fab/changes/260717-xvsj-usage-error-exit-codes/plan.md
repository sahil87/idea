# Plan: Adopt Toolkit Usage-Error Exit-Code Convention

**Change**: 260717-xvsj-usage-error-exit-codes
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md. Toolkit principle №4 (Fail fast with actionable errors, a MUST)
     fixes the exit-code convention at 0 success / 1 operational / 2 usage. -->

### CLI: Usage-Error Exit-Code Convention

#### R1: Usage-error sentinel composing with errSilent
The `cmd/idea` package SHALL define a `usageError` wrapper error type that marks an error as
stemming from a malformed invocation, and SHALL implement `Unwrap()` so it composes with the
existing `errSilent` sentinel and keeps `errors.Is`/`errors.As` classification working. Exit-code
policy stays in `cmd/` (Constitution IV); `internal/idea` remains policy-free.

- **GIVEN** an error wrapped as `&usageError{someErr}`
- **WHEN** `main()` classifies it via `errors.As(err, &uerr)`
- **THEN** the wrapper is detected and unwrapping yields the inner error
- **AND** a `&usageError{errSilent}` still satisfies `errors.Is(err, errSilent)` (no extra `ERROR:` line)

#### R2: Flag-error seam via root SetFlagErrorFunc
The root command SHALL register a single `SetFlagErrorFunc` that wraps every flag-parse error as a
`usageError`. Cobra inherits `FlagErrorFunc` from the parent, so subcommands are covered without
per-command wiring. Message text is unchanged.

- **GIVEN** the root command has the flag-error func registered
- **WHEN** the user runs `idea --nope` or `idea list --bogus`
- **THEN** the process exits with code 2
- **AND** the stderr message is byte-identical to today's cobra flag-error output

#### R3: Arg-validation seam via wrapped Args validators
Each subcommand's `Args` positional-args validator SHALL be wrapped by a `usageArgs` helper that
classifies a rejection as a `usageError`. Applied at every wrappable `Args:` site: `add`, `done`,
`reopen`, `rm`, `show` (`cobra.ExactArgs(1)`), `edit` (`cobra.RangeArgs(1,2)`), `fmt`, `prune`,
`update`, `help-dump` (`cobra.NoArgs`), and `list`'s custom validator func. Root and `shell-init`
use `cobra.ArbitraryArgs` — nothing to wrap. The unknown-subcommand class is vacuous for `idea`
(root `ArbitraryArgs` + bare-text shorthand captures unresolved first words by design), so no
command-resolution wrapping is added.

- **GIVEN** a subcommand with a `usageArgs`-wrapped validator
- **WHEN** the user violates its arity (e.g. `idea add` with 0 args, `idea fmt extra`, `idea edit` with 0 args)
- **THEN** the process exits with code 2
- **AND** the arg-validation error message is unchanged

#### R4: main() two-class exit mapping
`main()` SHALL map errors to exit codes: `errors.As` for `*usageError` → exit 2, otherwise exit 1;
success → exit 0. The `errSilent` handling (skip the `ERROR:` line) is preserved. No error wording
changes anywhere — the `ERROR:` prefix, refusal/hint texts, and all stderr composition stay
byte-identical.

- **GIVEN** `Execute()` returns a non-nil error
- **WHEN** the error is (or wraps) a `usageError`
- **THEN** `main()` exits 2
- **AND** WHEN the error is any other operational error THEN `main()` exits 1
- **AND** WHEN the error is (or wraps) `errSilent` THEN no `ERROR:` line is printed

#### R5: shell-init migrates inline os.Exit(2) to the shared path
`shell_init.go`'s two `os.Exit(2)` calls (missing shell, unsupported shell) SHALL become
`return &usageError{errSilent}` after their existing self-printed stderr messages. Observable
behavior is byte-identical (same stderr text, same exit 2), but exit now routes through the single
`main()` seam instead of bypassing deferred functions and killing in-process test runs. The comment
noting "the exact error text (and exit code 2) that the shll meta-CLI expects" remains binding —
those strings MUST NOT change.

- **GIVEN** `idea shell-init` with no shell, or with an unsupported shell
- **WHEN** the command runs
- **THEN** the process exits 2 with stderr byte-identical to today's
  (`idea shell-init: missing shell. Supported: zsh, bash` / `idea shell-init: unsupported shell '<x>'. Supported: zsh, bash`)

#### R6: --system/--main conflict classified as usage error
`internal/idea` SHALL export a sentinel `ErrConflictingTargets` (message text unchanged), which the
existing mutual-exclusion check in `ResolveBacklogPath` returns (directly or wrapped, preserving the
current message). `resolveFile()` in `cmd/idea/resolve.go` SHALL check
`errors.Is(err, idea.ErrConflictingTargets)` and wrap the error into a `usageError` — exit-code
policy stays in `cmd/` (Constitution IV); `internal/idea` only names the condition. The check does
NOT move (colocation with the precedence logic is a documented design decision).

- **GIVEN** the user passes both `--system` and `--main`
- **WHEN** any backlog-touching command resolves its path (e.g. `idea -s -m list`)
- **THEN** the process exits 2 with the unchanged message `--system and --main are mutually exclusive; pass only one`

#### R7: Operational-class exits stay 1
Consent refusals (`rm`/`prune` without `--yes`/`--force`), no-match/ambiguous-match query errors
(`RequireSingle`), `fmt --check` on a non-canonical file (exit 1 via `errSilent`), and file
I/O/editor/git-resolution failures SHALL continue to exit 1 — they are outcomes of well-formed
invocations, not malformed ones.

- **GIVEN** a well-formed invocation whose operation fails (no match, declined consent, non-canonical `fmt --check`)
- **WHEN** the command runs
- **THEN** the process exits 1 (unchanged)

#### R8: Documentation — exit-code convention note
`docs/specs/overview.md` SHALL gain a short exit-code convention note (0 success / 1 operational /
2 usage) in the CLI-surface contract. No help-text changes, so no `help-dump` re-run obligation.

- **GIVEN** the specs overview CLI-surface contract
- **WHEN** a reader consults it for exit-code semantics
- **THEN** the 0/1/2 convention is documented

### Non-Goals

- No error-wording changes (exit codes only).
- No JSON/format/schema changes; no new dependencies.
- No command-resolution (unknown-subcommand) wrapping — vacuous for this tree.
- No changes to the `help-dump` JSON envelope.

### Design Decisions

1. **Two seams, one sentinel-based mapping**: root `SetFlagErrorFunc` for flag errors + `usageArgs`-wrapped
   validators for arg-count errors, both routed through one `errors.As(*usageError)` check in `main()` —
   *Why*: a flag-only partial fix would make usage-error classes disagree (flags→2, arg-count→1), worse for an
   exit-code-branching agent than the current uniform 1 — *Rejected*: flag-only `SetFlagErrorFunc`.
2. **`usageError.Unwrap()` is load-bearing**: it composes with `errSilent` (self-printed usage errors return
   `&usageError{errSilent}`) and keeps `errors.Is`/`errors.As` working — *Why*: single mapping seam, no extra
   `ERROR:` line for self-printed cases — *Rejected*: a separate silent-usage sentinel.
3. **Sentinel lives in `internal/idea`, classification in `cmd/`** for R6 — *Why*: Constitution IV keeps
   exit-code policy in `cmd/`; `internal/idea` only names the condition and does not move the check —
   *Rejected*: wrapping inside `ResolveBacklogPath`.

## Tasks

### Phase 1: Sentinel + helper scaffolding

- [x] T001 Add the `usageError` wrapper type (with `Error()` and `Unwrap()`) and the `usageArgs(v cobra.PositionalArgs) cobra.PositionalArgs` helper to `src/cmd/idea/main.go` (colocated with `errSilent`). <!-- R1, R3 -->

### Phase 2: Wire the two usage seams + mapping

- [x] T002 Register `root.SetFlagErrorFunc(func(cmd, err) error { return &usageError{err} })` in `newRootCmd()` in `src/cmd/idea/main.go`. <!-- R2 -->
- [x] T003 Rewrite `main()` in `src/cmd/idea/main.go` to the two-class mapping: keep the `errSilent` `ERROR:`-skip, then `code := 1; var uerr *usageError; if errors.As(err, &uerr) { code = 2 }; os.Exit(code)`. <!-- R4 -->
- [x] T004 Wrap every wrappable `Args:` validator with `usageArgs(...)` in `src/cmd/idea/{add,done,reopen,rm,show}.go` (`ExactArgs(1)`), `edit.go` (`RangeArgs(1,2)`), `fmt.go`/`prune.go`/`update.go`/`help_dump.go` (`NoArgs`), and `list.go` (the custom func). Leave `main.go`/`shell_init.go` `ArbitraryArgs` untouched. <!-- R3 -->

### Phase 3: shell-init + conflict classification

- [x] T005 Migrate `src/cmd/idea/shell_init.go`'s two `os.Exit(2)` calls to `return &usageError{errSilent}` (keeping the self-printed stderr messages and the binding comment); drop the now-unused `os` import if it becomes unused. <!-- R5 -->
- [x] T006 Export `ErrConflictingTargets` (message text unchanged) in `src/internal/idea/idea.go` and have the mutual-exclusion check return it (preserving the exact current message). <!-- R6 -->
- [x] T007 In `src/cmd/idea/resolve.go`, check `errors.Is(err, idea.ErrConflictingTargets)` and wrap into `&usageError{err}`; other errors pass through unchanged. <!-- R6 -->

### Phase 4: Tests + docs

- [x] T008 Add a table-driven subprocess exit-code matrix test in `src/cmd/idea/main_test.go` (reusing `buildBinary`/`setupGitRepo`/`writeRepoBacklog`/`runSplit`): usage→2 (`idea --nope`, `idea list --bogus`, `idea add`, `idea fmt extra`, `idea edit` 0 args, `idea -s -m list`), operational→1 (no-match query, `rm <id>` without consent, `fmt --check` non-canonical), success→0 (`idea list`). Assert exit code via `*exec.ExitError.ExitCode()`. <!-- R2, R3, R4, R6, R7 -->
- [x] T009 Audit `src/cmd/idea/shell_init_test.go` (and other `*_test.go`) for any assertion pinning exit 1 on a usage path; the two shell-init exit-2 tests MUST still pass byte-identically. Update any usage-path exit-1 assertion to 2 (Test Integrity). <!-- R5 -->
- [x] T010 Add a short exit-code convention note (0 success / 1 operational / 2 usage) to the CLI-surface contract in `docs/specs/overview.md`. <!-- R8 -->

## Execution Order

- T001 blocks T002–T007 (they reference `usageError`/`usageArgs`).
- T006 blocks T007 (resolve.go references the exported sentinel).
- T008–T010 run after Phases 1–3.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `usageError` type exists in `cmd/idea` with `Error()`+`Unwrap()`; `&usageError{errSilent}` satisfies `errors.Is(_, errSilent)`.
- [x] A-002 R2: `root.SetFlagErrorFunc` is registered once; `idea --nope` and `idea list --bogus` exit 2 with unchanged stderr.
- [x] A-003 R3: every wrappable `Args:` site is wrapped with `usageArgs`; arg-count violations exit 2 with unchanged messages.
- [x] A-004 R4: `main()` maps `*usageError`→2, else→1, success→0, preserving `errSilent` behavior.
- [x] A-005 R5: `shell-init` missing/unsupported-shell paths return `&usageError{errSilent}`, still exit 2 with byte-identical stderr.
- [x] A-006 R6: `ErrConflictingTargets` is exported (message unchanged); `idea -s -m list` exits 2 via the `resolveFile` classification seam.
- [x] A-007 R8: `docs/specs/overview.md` documents the 0/1/2 exit-code convention.

### Behavioral Correctness

- [x] A-008 R4: usage errors move 1→2 tree-wide while operational errors stay 1 — the two classes no longer collapse.
- [x] A-009 R5: shell-init exit now routes through `main()` (no inline `os.Exit`), so in-process test runs are not killed.

### Scenario Coverage

- [x] A-010 R2 R3 R4 R6 R7: the `main_test.go` exit-code matrix exercises usage→2, operational→1, success→0 and passes.
- [x] A-011 R5: the existing shell-init exit-2 tests pass unchanged (byte-identical stderr + exit 2).

### Edge Cases & Error Handling

- [x] A-012 R1: a self-printed usage error (`&usageError{errSilent}`) exits 2 with NO extra `ERROR:` line.
- [x] A-013 R7: consent refusal, no-match query, and `fmt --check` non-canonical all still exit 1.

### Code Quality

- [x] A-014 Pattern consistency: new code follows the `errSilent` sentinel style and cobra factory patterns of surrounding code.
- [x] A-015 No unnecessary duplication: the single `usageArgs` helper and single `main()` mapping seam are reused; no per-command exit logic.
- [x] A-016 No magic numbers/strings: exit codes 1/2 map through the sentinel classification, not scattered literals; no error wording duplicated.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/cmd/idea/shell_init_test.go:78-84, 104-110` — the two inline `*exec.ExitError` type-assertion + `ExitCode()` blocks duplicate what the new `exitCodeOf` helper (`main_test.go`) does; consolidating onto the helper would delete ~14 lines of test boilerplate.
- `src/cmd/idea/fmt_test.go:151-155` — same inline exit-code extraction pattern, also consolidatable onto `exitCodeOf`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `usageError` type + `usageArgs` helper live in `main.go` alongside `errSilent` (not a separate `errors.go` sibling) | Intake offers "or a small sibling `errors.go`" as optional; colocation with `errSilent` matches the existing single-file convention and is trivially reversible | S:85 R:90 A:85 D:80 |
| 2 | Certain | The 11 wrapped `Args:` sites are `add`/`done`/`reopen`/`rm`/`show`/`edit`/`fmt`/`prune`/`update`/`help-dump`/`list`; `main`/`shell-init` (`ArbitraryArgs`) are not wrapped | Verified by grepping `Args:` across `cmd/idea/*.go`; intake enumerates the same set (its "help-dump" list-item plus `update` = the 4 `NoArgs` sites) | S:90 R:85 A:95 D:90 |
| 3 | Certain | `ErrConflictingTargets` returned directly (not wrapped) by the mutual-exclusion check, preserving the exact message `--system and --main are mutually exclusive; pass only one` | Simplest form; `errors.Is` matches the sentinel identity directly; intake permits either direct return or `%w` wrapping | S:85 R:85 A:90 D:80 |
| 4 | Confident | The exit-code matrix reads the exit code via `(*exec.ExitError).ExitCode()` on the `err` returned by `runSplit` | Matches the existing pattern in `shell_init_test.go`/`fmt_test.go`; `runSplit` already returns `err` from `cmd.Run()` | S:75 R:85 A:85 D:75 |
| 5 | Confident | The `docs/specs/overview.md` note is a brief additive paragraph in the CLI-surface contract, wording only (no restructure) | Intake scopes it as "a short exit-code convention note"; minimal additive edit is lowest-risk and matches principle №4's "Enforced by" | S:70 R:85 A:75 D:70 |

5 assumptions (3 certain, 2 confident, 0 tentative).
