# Intake: Add ls Alias for list Subcommand

**Change**: 260610-04rt-add-ls-alias
**Created**: 2026-06-10
**Status**: Draft

## Origin

> Alias "idea ls" to "idea list". Any other aliases you think would be useful?
> [after discussion] Agreed, ship just ls.

Conversational origin via `/fab-discuss`. The user asked for an `ls` alias and solicited suggestions for others. During discussion the agent surveyed the full command surface (`add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`, `update`, `shell-init`, `help-dump`) and identified that the bare-text shorthand (`idea <text>` → `idea add <text>`, root `RunE` in `src/cmd/idea/main.go`) means every alias permanently removes a word from the start of bare-text idea capture. Candidate aliases `remove`/`delete` (for `rm`), `upgrade` (for `update`), `cat` (for `show`), and `undo` (for `reopen`) were considered and **rejected** — imperative verbs like "remove" and "upgrade" plausibly start idea text, and `undo` implies revert-last-action semantics worth reserving. The user explicitly agreed: ship just `ls`.

## Why

1. **Pain point**: `idea ls` is near-universal unix muscle memory, but today it does not error — cobra finds no `ls` subcommand, falls through to the root command's bare-text shorthand, and **silently adds an idea with the text "ls"** to the backlog. The user gets polluted data instead of the listing they asked for.
2. **Consequence of not fixing**: every habitual `ls` invocation creates a junk backlog entry that must be manually `rm`'d, eroding trust in the bare-text capture feature.
3. **Why this approach**: cobra has first-class alias support (`Aliases` field on `cobra.Command`), which routes `ls` to `list` *before* the root `RunE` fallback fires. This is the cobra-idiomatic mechanism (Constitution III), requires no custom routing logic, and surfaces automatically in cobra's generated help (`Aliases:` section). "ls" essentially never begins idea prose, so the bare-text namespace cost is nil.

## What Changes

### `src/cmd/idea/list.go` — register the alias

Add the `Aliases` field to the `cobra.Command` literal in `listCmd()`:

```go
cmd := &cobra.Command{
    Use:     "list",
    Aliases: []string{"ls"},
    Short:   "List ideas from the backlog",
    ...
}
```

Behavior after the change:

- `idea ls` behaves identically to `idea list` (same flags: `--all/-a`, `--done`, `--json`, `--sort`, `--reverse`; same persistent flags `--file`, `--main`).
- `idea ls` no longer reaches the root bare-text shorthand, so it no longer adds a junk "ls" idea.
- `idea list` is unchanged. The alias is pure routing — no output, flag, or behavior changes to the list command itself.
- Cobra's generated help for the command shows an `Aliases: list, ls` line automatically.

GIVEN a backlog with open ideas, WHEN the user runs `idea ls`, THEN the open ideas are listed exactly as `idea list` would print them, AND no new idea is appended to the backlog file.

GIVEN any backlog state, WHEN the user runs `idea ls --json`, THEN the structured JSON records (id, date, status, text) are emitted exactly as `idea list --json` would.

### `src/cmd/idea/main_test.go` — table-driven routing test

Add table-driven routing cases (Constitution V) verifying:

1. `ls` routes to the list command (output matches `list`; backlog file unchanged after invocation).
2. Bare text starting with a non-alias word (e.g., `idea lsx some text` or `idea buy milk`) still routes to the bare-text add shorthand and appends an idea.

### Out of scope (explicitly rejected during discussion)

- No other aliases: `remove`/`delete` (rm), `upgrade` (update), `cat` (show), `undo` (reopen) were all considered and rejected — each would shadow a plausible first word of bare-text idea capture, or (for `undo`) reserve the wrong semantics.
- No `aliases` field in the help-dump JSON schema (`src/cmd/idea/help_dump.go`). The schema is a public contract (Constitution VI); an additive field is deferred until an external consumer needs it. Alias visibility via cobra's rendered help text is sufficient for now.
- No changes to docs/site rendering or release pipeline.

## Affected Memory

- `cli/structure`: (modify) note the `ls` alias in the per-subcommand notes for `list`, and the routing rule that command aliases resolve before the root bare-text shorthand.

## Impact

- **Code**: `src/cmd/idea/list.go` (one field added), `src/cmd/idea/main_test.go` (routing test cases).
- **CLI surface**: additive — new invocation spelling `idea ls`; existing invocations unchanged. Behavioral change only for the literal input `idea ls`, which previously added an idea with text "ls" (judged a footgun, not a depended-upon behavior).
- **Docs/specs**: `docs/specs/overview.md` lists commands (human-curated; may be touched at hydrate or left to the human owner).
- **Dependencies**: none — uses existing cobra capability.
- **Help-dump/shll.ai**: unchanged schema; cobra help text gains an `Aliases:` line wherever rendered help is captured.

## Open Questions

(none — scope and mechanism were fully resolved in the originating discussion)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use cobra's native `Aliases: []string{"ls"}` on `listCmd()` | Constitution III mandates cobra-idiomatic surface; cobra aliases are the first-class mechanism and resolve before the root bare-text fallback | S:90 R:90 A:95 D:95 |
| 2 | Certain | Scope is exactly one alias (`ls`); no aliases for rm/update/show/reopen | Discussed — user explicitly agreed "ship just ls" after alternatives were surveyed and rejected | S:95 R:90 A:95 D:95 |
| 3 | Certain | `list` command behavior is otherwise unchanged (flags, output, JSON schema) | Alias is pure routing; Constitution VI stable-output rule makes any output change out of scope | S:90 R:95 A:95 D:95 |
| 4 | Confident | Routing tests live in `src/cmd/idea/main_test.go` as table-driven cases: `ls` → list, non-alias bare text → add | Discussed test shape; Constitution V mandates table-driven tests; main_test.go already covers root routing | S:85 R:85 A:90 D:85 |
| 5 | Confident | No `aliases` field added to help-dump JSON schema | Discussed — schema is public contract (Constitution VI); additive change deferred until a consumer needs it; cobra rendered help shows aliases anyway | S:70 R:80 A:75 D:80 |
| 6 | Confident | Memory hydrate updates `cli/structure` only (no new domain, no spec rewrite) | Single-subcommand surface change; cli/structure.md already holds per-subcommand notes | S:75 R:90 A:85 D:80 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).
