# Intake: Enrich Cobra `Long` Help for idea's Commands

**Change**: 260602-s73u-enrich-command-long-help
**Created**: 2026-06-02
**Status**: Draft

## Origin

> Enrich the cobra `Long` descriptions for idea's commands so that `idea <cmd> -h` carries the human explanation, not just a one-line `Short` + flag list. MOTIVATION: shll.ai's command-reference pages render each tool's RAW -h output verbatim (single-sourced from the binary, no hand-written site copy — see the help-collection contract + nnsn). Today 8 of idea's 10 commands set only `Short` (just `main` and `shell-init` have `Long`), so the generated reference is thin and the curated prose that used to live in the site's hand-written commands.md has no home in the binary. Fix at the source: flesh out `Long` (in src/cmd/idea/{add,list,done,edit,rm,show,reopen,update}.go) with the explanatory text a user wants from `idea add -h` — what the command does, key flags/behaviors, the worktree-vs-`--main` resolution where relevant, a short example. This is the canonical place for that prose: it helps real terminal users AND flows automatically into help/idea.json -> the shll.ai reference (nnsn captures Long+UsageString as `text`), so the explanation is single-sourced and never drifts. Keep `Short` as the terse sidebar/`Available Commands` one-liner; put the depth in `Long`. Mirror this in fab-kit (same pattern, same motivation) if it lands there too. Constitutional fit: the site stays a directory pointing at the binary's own help, not a second copy to maintain. Verify with `go build` + `idea add -h` showing the enriched output, and that the help-dump (nnsn) picks it up.

**Mode**: One-shot. The input is a single, highly detailed specification — no prior `/fab-discuss` turns. Scope, files, motivation, and verification criteria were all supplied up front.

## Why

**The problem.** `idea`'s `-h` output is the *single source* for the shll.ai command-reference page (per the `[nnsn]` help-dump contract — the site renders the binary's raw `-h` byte-for-byte, with no hand-written site copy). The help-dump captures each command's `Long` + `UsageString` as the node's `text`. But today only 2 of idea's 10 commands set `Long` (`main` and `shell-init`); the other 8 (`add`, `list`, `done`, `edit`, `rm`, `show`, `reopen`, `update`) set only a terse one-line `Short`. So `idea add -h` shows nothing more than that one line plus the auto-generated flag list — and the generated reference page is correspondingly thin.

**The consequence if unfixed.** The curated explanatory prose that historically lived in the site's hand-written `commands.md` has nowhere to live in the binary. Either it gets re-added as a second, hand-maintained copy on the site (violating the single-source contract and guaranteeing drift), or it is simply lost and both terminal users (`idea <cmd> -h`) and site readers get a thin, unhelpful reference.

**Why this approach (enrich `Long` at the source) over alternatives.** The binary's help is the canonical, single-sourced home for command prose: it serves real terminal users *and* flows automatically into `help/idea.json` → the shll.ai reference via `nnsn`. Prose written once in `Long` never drifts from what the site shows, because the site is a directory *pointing at* the binary's own help, not a second copy. The rejected alternative — keeping the prose in a hand-written site file — reintroduces exactly the drift the help-collection contract was designed to eliminate. This is the constitutionally aligned fix: fix at the source, keep one writer of the text.

## What Changes

Add a multi-paragraph `Long` field to each of the 8 commands that currently lack one. `Short` is left unchanged (it remains the terse one-liner for the `Available Commands` sidebar and `idea -h` root listing); the explanatory depth goes in `Long`.

Target files (all under `src/cmd/idea/`): `add.go`, `list.go`, `done.go`, `edit.go`, `rm.go`, `show.go`, `reopen.go`, `update.go`.

### `Long` content contract (per command)

Each `Long` should carry the explanation a user actually wants from `idea <cmd> -h`:

1. **What the command does** — one or two sentences of plain-language behavior, beyond the `Short` one-liner.
2. **Key flags / behaviors** — what the non-obvious flags do (e.g., `list --json`/`--all`/`--done`/`--sort`/`--reverse`; `rm --force` as a required confirmation; `edit`/`add` `--id`/`--date`; `show --json`; `update --skip-brew-update`).
3. **Worktree-vs-`--main` resolution** — for the commands that read/write the backlog (all 8 except `update`), a short note that the command operates on the **current worktree's** `fab/backlog.md` by default and that `--main` (plus `--file`/`IDEAS_FILE`) selects a different backlog. Phrase this once, consistently, per command — not a verbatim wall of text duplicated identically (the persistent flags are documented on root, so each command's `Long` should *reference* the behavior, not re-explain the whole resolution algorithm).
4. **Query semantics** — for `show`/`done`/`reopen`/`edit`/`rm`, a note that `<query>` matches an idea by ID or substring of its text (case-insensitive), and that an ambiguous query that matches more than one idea is refused with the list of matches.
5. **A short example** — one representative invocation (e.g., `idea add "wire up dark mode"`, `idea list --json`, `idea rm a7k2 --force`).

### Style

Match the existing `Long` prose style already used in `main.go` and `shell_init.go`: a raw Go string literal (backtick-quoted), short paragraphs, blank-line separation, an inline indented example block where helpful. Keep it tight — explanatory, not exhaustive. Do not restate the cobra-auto-generated `Usage:` / flag list (cobra appends that automatically below `Long`).

### Example (illustrative — `add`)

```go
Short: "Add a new idea to the backlog",
Long: `Add a new idea to the current worktree's backlog (fab/backlog.md).

Each idea is stored as a Markdown checklist line with a generated 4-char ID
and today's date. Use --main to target the main worktree's backlog instead,
or --file / IDEAS_FILE to point at a different file.

  idea add "wire up dark mode"
  idea add --id a7k2 --date 2026-06-01 "backdated idea"`,
```

### fab-kit mirror

The motivation notes "Mirror this in fab-kit (same pattern, same motivation) if it lands there too." This `idea` change is scoped to the `idea` repo only. The fab-kit mirror is **out of scope for this change** — it is a separate codebase with its own pipeline. Track it as a follow-up rather than spanning two repos in one change (constitution favors small, focused changes; cross-repo edits aren't supported by this worktree).

## Affected Memory

- `cli/structure.md`: (modify) The per-subcommand notes section may gain a line noting that each command carries an enriched `Long` consumed by the help-dump. Minor — implementation-adjacent, decided at hydrate.

<!-- This is primarily a documentation/prose change inside existing cobra factories; no new package, no behavior change. The bulk is help text, which memory does not track verbatim. -->

## Impact

- **Code**: 8 files under `src/cmd/idea/` (`add`, `list`, `done`, `edit`, `rm`, `show`, `reopen`, `update`). Each gains a `Long:` field in its `*cobra.Command` literal. No `RunE`, flag wiring, or `internal/idea` logic changes — purely additive help text.
- **APIs / behavior**: No runtime behavior change. `Short` is the public sidebar string and stays byte-stable. `Long` is new output surfaced only via `-h`/`--help` and the help-dump.
- **Downstream consumer**: `[nnsn]` (the build-time help-dump → `help/idea.json` → shll.ai reference). `nnsn` captures `Long+UsageString` as node `text`; enriching `Long` is what makes that capture worth rendering. The two are complementary — `nnsn` is the mechanism, this change is the content.
- **Dependencies**: None added. Pure cobra struct-literal text.
- **Tests**: No logic to unit-test, and no regression test added. Help text is not behavior, so the constitution does not require coverage here; the change stays purely additive prose. <!-- clarified: no Long-presence regression test — user chose to keep the change purely additive; accepts that a future command could ship Short-only without a CI signal -->. The existing `main_test.go` continues to build/exercise the binary unchanged.
- **Constitution**: Aligns with Principle III (cobra-idiomatic surface) and IV (cmd/ holds only wiring/formatting — help text qualifies). No principle is stretched.

## Open Questions

- ~~Should a regression test be added asserting every non-hidden subcommand has a non-empty `Long`?~~ **Resolved (clarify 2026-06-02):** No test. Help text is not behavior; the constitution doesn't require it. The change stays purely additive prose, accepting that a future command could ship `Short`-only without a CI signal.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly the 8 `Short`-only commands (`add`, `list`, `done`, `edit`, `rm`, `show`, `reopen`, `update`); `main` and `shell-init` already have `Long` and are untouched. | Enumerated verbatim in the input and confirmed by `grep` of the source (only `main.go`/`shell_init.go` set `Long`). | S:98 R:90 A:95 D:95 |
| 2 | Certain | `Short` stays unchanged; depth goes in `Long` only. | Explicit in the input ("Keep `Short` as the terse one-liner; put the depth in `Long`"). `Short` is also the public sidebar contract. | S:98 R:85 A:95 D:95 |
| 3 | Certain | `Long` style matches existing `main.go`/`shell_init.go` (backtick raw string, short paragraphs, inline example). | Existing in-repo prose is the established pattern; constitution says follow existing patterns. | S:90 R:85 A:95 D:90 |
| 4 | Confident | Each backlog-touching command's `Long` mentions worktree-vs-`--main` resolution by reference (not a duplicated full algorithm). | Input asks for "the worktree-vs-`--main` resolution where relevant"; persistent flags are root-documented, so per-command text should reference, not re-derive. One obvious reading. | S:80 R:80 A:80 D:75 |
| 5 | Confident | `show`/`done`/`reopen`/`edit`/`rm` `Long` documents query semantics (ID-or-substring, case-insensitive, ambiguity refused). | Verified against `RequireSingle`/`Match` in `internal/idea/idea.go`; this is the user-facing behavior a `-h` reader needs. | S:82 R:82 A:88 D:80 |
| 6 | Confident | The fab-kit mirror is out of scope for this change (follow-up only). | Input hedges it ("if it lands there too"); fab-kit is a separate repo/pipeline and cross-repo edits don't fit one worktree/change. Easily revisited. | S:70 R:75 A:80 D:78 |
| 7 | Certain | No regression test — the change stays purely additive prose. | Clarified — user chose no test; help text is not behavior, so the constitution doesn't require it. | S:95 R:75 A:65 D:55 |
| 8 | Certain | Verification = `go build` succeeds + `idea add -h` (and peers) shows enriched output + the help-dump (`nnsn`) captures it. | Acceptance criteria stated verbatim in the input. | S:95 R:80 A:90 D:90 |

8 assumptions (6 certain, 2 confident, 0 tentative, 0 unresolved).

## Clarifications

### Session 2026-06-02

| # | Question | Answer |
|---|----------|--------|
| 7 | Add a regression test asserting subcommands carry an enriched `Long`? | No test — keep the change purely additive prose. Help text is not behavior; not constitution-required. Accepts that a future command could ship `Short`-only without a CI signal. |
