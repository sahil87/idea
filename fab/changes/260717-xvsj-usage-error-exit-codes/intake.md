# Intake: Adopt Toolkit Usage-Error Exit-Code Convention

**Change**: 260717-xvsj-usage-error-exit-codes
**Created**: 2026-07-18

## Origin

> xvsj

One-shot `/fab-new xvsj` invocation resolving backlog item `[xvsj]` (2026-07-18). No conversational refinement was needed — the backlog entry plus the `260717-9uh7-toolkit-standards-conformance` conformance report (which deferred this item) carry the design decisions. Backlog text, verbatim:

> Adopt the toolkit exit-code convention tree-wide (principle 4): usage errors should exit 2, operational failures exit 1, success 0. Today only 'idea shell-init' returns 2 (it does its own os.Exit(2)); every OTHER usage error — unknown flag, wrong arg count (cobra.ExactArgs), unknown subcommand — exits 1 via main()'s uniform os.Exit(1). Verified: cobra's SetFlagErrorFunc catches FLAG errors but NOT arg-count/unknown-command errors, so a complete fix must tag both seams: a root SetFlagErrorFunc for flag errors PLUS wrapping the Args validators / command-resolution path to mark usage errors, then map that to exit 2 in main(). DEFERRED from 260717-9uh7-toolkit-standards-conformance because a flag-only partial fix would make usage-error classes DISAGREE (flags->2, arg-count->1), which is worse for an agent branching on exit codes than the current uniform 1; the intake scoped exit-code fixes to local-only and deferred structural ones. Do it consistently in its own change (an errUsage/errExitCode sentinel is the idiomatic vehicle; principle 4's 'Enforced by' names the errSilent/errExitCode pattern). Ref: shll standards principles no.4 @ shll v0.0.23.

## Why

1. **The pain point.** Toolkit principle №4 (*Fail fast with actionable errors*, a MUST) fixes the exit-code convention at `0` success / `1` operational failure / `2` usage error. `idea` today collapses both failure classes to exit `1`: `main()` runs `os.Exit(1)` on any error from `Execute()` (`src/cmd/idea/main.go:100`), so an unknown flag, a wrong arg count, and a genuine operational failure (file I/O, no match) are indistinguishable to a caller. Only `shell-init` exits `2`, via its own inline `os.Exit(2)` calls — a one-off, not a convention.

2. **The consequence.** An agent (or script) branching on exit codes cannot tell "I invoked this wrong — fix the invocation and retry" from "the operation failed — different remedy". The 9uh7 audit marked principle №4 **Gap → deferred** for exactly this; leaving it means `idea` stays non-conformant with a MUST obligation of the toolkit standards the constitution binds this repo to (§ Toolkit Standards, v1.1.0).

3. **Why this approach.** A flag-only fix (`SetFlagErrorFunc` alone) was explicitly rejected at 9uh7: it would make usage-error classes *disagree* (flag errors → 2, arg-count errors → 1), which is worse for an exit-code-branching agent than the current uniform 1. The complete fix tags **both** seams in one change — the flag-error seam and the arg-validation seam — and routes both through one sentinel-based mapping in `main()`, the `errSilent`/`errExitCode` pattern principle №4's "Enforced by" names as the toolkit-idiomatic vehicle.

## What Changes

### 1. Usage-error sentinel in `cmd/idea`

A wrapper error type in the command layer (exit-code policy is a `cmd/` concern per Constitution IV — `internal/idea` stays policy-free):

```go
// usageError marks an error as stemming from a malformed invocation
// (bad flag, wrong arg count, conflicting target flags) so main()
// maps it to exit 2 per the toolkit exit-code convention.
type usageError struct{ err error }

func (u *usageError) Error() string { return u.err.Error() }
func (u *usageError) Unwrap() error { return u.err }
```

`Unwrap()` is load-bearing: it lets a usage error **compose with the existing `errSilent` sentinel** (a self-printed usage error like shell-init's returns `&usageError{errSilent}` — exit 2, no extra `ERROR:` line), and keeps `errors.Is`/`errors.As` classification working in `main()`.

### 2. Flag-error seam: root `SetFlagErrorFunc`

One registration on the root command wraps every flag-parse error as a usage error — cobra inherits `FlagErrorFunc` from the parent, so subcommands are covered without per-command wiring:

```go
root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
    return &usageError{err}
})
```

After this, `idea --nope` and `idea list --bogus` exit **2** (message unchanged).

### 3. Arg-validation seam: wrap the `Args` validators

Verified at 9uh7: `SetFlagErrorFunc` does **not** catch arg-count errors. Each subcommand's `Args` validator is wrapped by a small helper:

```go
// usageArgs wraps a cobra positional-args validator so its rejection
// is classified as a usage error (exit 2).
func usageArgs(v cobra.PositionalArgs) cobra.PositionalArgs {
    return func(cmd *cobra.Command, args []string) error {
        if err := v(cmd, args); err != nil {
            return &usageError{err}
        }
        return nil
    }
}
```

Applied at every `Args:` site: `add`/`done`/`reopen`/`rm`/`show` (`cobra.ExactArgs(1)`), `edit` (`cobra.RangeArgs(1, 2)`), `fmt`/`prune`/`update`/`help-dump` (`cobra.NoArgs`), and `list`'s custom validator func. Root and `shell-init` use `cobra.ArbitraryArgs` — nothing to wrap.

**Unknown-subcommand class is vacuous for `idea`**: the root is `ArbitraryArgs` with the bare-text shorthand (`idea <text>` → `idea add <text>`), so an unresolved first word is *captured as an idea*, never an error (the documented routing rule in `docs/memory/cli/structure.md`). No command-resolution wrapping is needed; the backlog's "unknown subcommand" case does not arise in this tree.

### 4. `main()` exit mapping

`main()` grows from a uniform `os.Exit(1)` to the two-class mapping (wording untouched):

```go
func main() {
    if err := newRootCmd().Execute(); err != nil {
        if !errors.Is(err, errSilent) {
            fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
        }
        code := 1
        var uerr *usageError
        if errors.As(err, &uerr) {
            code = 2
        }
        os.Exit(code)
    }
}
```

**No error-wording changes anywhere.** The 9uh7 audit already assessed the what/why/next wording as conformant; this change alters exit codes only. In particular the `ERROR:` prefix, the refusal/hint texts, and all stderr composition stay byte-identical.

### 5. `shell-init`: migrate inline `os.Exit(2)` to the shared path

`shell_init.go`'s two `os.Exit(2)` calls (missing shell at `shell_init.go:43`, unsupported shell at `shell_init.go:60`) become `return &usageError{errSilent}` after their existing self-printed stderr messages. Observable behavior is byte-identical (same stderr text — `idea shell-init: missing shell. Supported: zsh, bash` / the unsupported-shell variant — same exit 2); what changes is that the exit now routes through the single `main()` seam instead of bypassing deferred functions and killing in-process test runs. The comment noting "the exact error text (and exit code 2) that the shll meta-CLI expects" remains binding — those strings MUST NOT change.

### 6. `--system`/`--main` conflict classified as usage error

`ResolveBacklogPath`'s first-line mutual-exclusion check (`--system and --main are mutually exclusive; pass only one`) is a usage error in substance — a malformed invocation — but surfaces today as a plain error → exit 1. It gets classified without moving the check (its colocation with the precedence logic is a documented design decision):

- `internal/idea` exports a sentinel, e.g. `var ErrConflictingTargets = errors.New(...)`, which the existing conflict return wraps (`fmt.Errorf("...: %w", ErrConflictingTargets)` or returning the sentinel-based error directly, preserving the current message text).
- `resolveFile()` in `cmd/idea/resolve.go` (the one-line forwarder) checks `errors.Is(err, idea.ErrConflictingTargets)` and wraps into `usageError` — exit-code *policy* stays in `cmd/` (Constitution IV); `internal/idea` only names the condition.

After this, `idea -s -m list` exits **2** with the unchanged message.

### 7. Exits that deliberately stay `1` (operational class)

- `rm`/`prune` consent refusals (`Use --yes (or --force) to confirm ...`) — a declined mutation, not a malformed invocation (and №1's non-TTY refusal contract was assessed conformant at 9uh7 with exit 1).
- No-match / ambiguous-match query errors from `RequireSingle`.
- `fmt --check` on a non-canonical file (the documented CI-gate contract: exit 1 via `errSilent`).
- File I/O, editor, git-resolution failures.

### 8. Tests

Table-driven subprocess coverage in `cmd/idea/main_test.go` (reusing `buildBinary`/`setupGitRepo`/`runSplit`), one exit-code matrix test:

- **Usage → 2**: root unknown flag (`idea --nope`), subcommand unknown flag (`idea list --bogus`), arg-count violations (`idea add` with 0 args, `idea fmt extra`, `idea edit` with 0 args), target conflict (`idea -s -m list`), `idea shell-init` (0 args) and `idea shell-init nope` — the latter two also asserting the stderr text is byte-identical to today's.
- **Operational → 1**: a no-match query (e.g. `idea done zzzz` on a seeded backlog), `idea rm <id>` without consent, `idea fmt --check` on a non-canonical file.
- **Success → 0**: one representative happy path (`idea list`).

Existing tests asserting error paths keep passing unless they pin exit 1 on a usage path — any such assertion is updated to 2 (Test Integrity: tests conform to the spec).

### 9. Documentation

`docs/specs/overview.md` gains a short exit-code convention note (0 success / 1 operational / 2 usage) in the CLI-surface contract — principle №4's "Enforced by" names per-subcommand exit-code documentation in each tool's CLI-surface spec. Memory updates land at hydrate (below). No help-text changes, so no `help-dump` re-run obligation (the tree and flags are unchanged).

## Affected Memory

- `cli/structure`: (modify) — document the tree-wide exit-code convention: the `usageError` sentinel composing with `errSilent`, the root `SetFlagErrorFunc`, the wrapped `Args` validators, `main()`'s 0/1/2 mapping, shell-init's migration off inline `os.Exit(2)`, the `ErrConflictingTargets` classification seam, and the operational-class exits that stay 1; update the "main() is a four-line wrapper … `os.Exit(1)` on error" description and close the `[xvsj]` deferral note in the toolkit-standards-conformance section.

## Impact

- `src/cmd/idea/main.go` — sentinel type + `usageArgs` helper (or a small sibling `errors.go`), `SetFlagErrorFunc` registration, exit mapping in `main()`.
- All 11 `Args:` sites across `src/cmd/idea/{add,done,reopen,rm,show,edit,fmt,prune,update,help_dump,list}.go` — mechanical wrap.
- `src/cmd/idea/shell_init.go` — two `os.Exit(2)` calls → sentinel returns.
- `src/internal/idea/idea.go` — exported `ErrConflictingTargets` sentinel (message text unchanged); `src/cmd/idea/resolve.go` — `errors.Is` classification.
- `src/cmd/idea/main_test.go` — exit-code matrix; audit `shell_init_test.go` for assertions on the exit mechanics.
- **External contract change**: usage errors move 1 → 2. Callers branching `0` vs non-zero are unaffected; callers pattern-matching exit 1 for usage cases will see 2 — this is the point of the convention and is the same migration every toolkit tool makes. Constitution VI freezes stdout schemas and IDs; exit codes are governed by the toolkit standard the constitution now binds (§ Toolkit Standards), which mandates this change.
- No new dependencies; no format/JSON changes; `go build` + `cd src && go test ./...` verify.

## Open Questions

None — the backlog item and the 9uh7 conformance report resolve the design; the remaining judgment calls are recorded as graded assumptions below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Tree-wide 0/1/2 convention: usage errors exit 2, operational failures 1, success 0 | Mandated by toolkit principle №4 (MUST) and the backlog item; exactly what 9uh7 deferred | S:95 R:85 A:100 D:95 |
| 2 | Certain | Both seams fixed in one change — root `SetFlagErrorFunc` for flag errors + wrapped `Args` validators for arg-count; no partial fix | Backlog explicitly rejects the flag-only partial (usage classes would disagree, worse than uniform 1) | S:95 R:80 A:95 D:90 |
| 3 | Certain | Vehicle is a `usageError` sentinel in `cmd/idea` composing with `errSilent` via `Unwrap`, mapped once in `main()` | Backlog names the errUsage/errExitCode sentinel as the idiomatic vehicle; principle №4's "Enforced by" cites the same pattern | S:90 R:75 A:85 D:85 |
| 4 | Confident | shell-init's two inline `os.Exit(2)` calls migrate to `return &usageError{errSilent}` with byte-identical stderr and exit code | Unifies the single exit seam and restores in-process testability; backlog is silent but the code signal is clear and behavior is provably unchanged | S:70 R:80 A:80 D:60 |
| 5 | Confident | `--system`/`--main` conflict is classified usage (exit 2) via an exported `ErrConflictingTargets` sentinel checked at the `resolveFile` seam | Squarely a malformed invocation under №4; small extension beyond the backlog's named seams; classification stays in `cmd/` per Constitution IV | S:50 R:75 A:55 D:45 |
| 6 | Confident | No error-wording changes — exit codes only; `ERROR:` prefix and all stderr texts stay byte-identical | 9uh7 already assessed wording conformant (what/why/next); minimal diff protects shll's pinned shell-init strings | S:75 R:85 A:80 D:75 |
| 7 | Certain | Table-driven subprocess exit-code matrix in `main_test.go` (usage→2 incl. shell-init byte-identical stderr, operational→1, success→0) | Constitution V pattern; existing `buildBinary`/`runSplit` infrastructure covers it | S:80 R:85 A:90 D:85 |
| 8 | Certain | Unknown-subcommand class is vacuous — root `ArbitraryArgs` + bare-text shorthand captures unresolved names by design; no command-resolution wrapping | Documented routing rule in `cli/structure`; changing it would break the shorthand contract (Constitution III) | S:85 R:90 A:95 D:90 |
| 9 | Confident | Consent refusals, no-match/ambiguous queries, `fmt --check`, and I/O failures stay exit 1 (operational class) | They are outcomes of well-formed invocations, not usage mistakes; 9uh7 assessed the refusal contract conformant at 1 | S:70 R:80 A:85 D:70 |
| 10 | Confident | `docs/specs/overview.md` gains the 0/1/2 convention note in the CLI-surface contract | Principle №4's "Enforced by" names per-subcommand exit-code docs in each tool's CLI-surface spec | S:60 R:85 A:75 D:65 |

10 assumptions (5 certain, 5 confident, 0 tentative, 0 unresolved).
