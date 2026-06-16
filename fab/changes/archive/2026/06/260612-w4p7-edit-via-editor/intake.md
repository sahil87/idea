# Intake: Editor-Based Editing for `idea edit`

**Change**: 260612-w4p7-edit-via-editor
**Created**: 2026-06-12

## Origin

One-shot invocation: `/fab-new w4p7` (backlog ID). The raw entry from `fab/backlog.md` `[w4p7]` (2026-06-12), decoded:

> Improve `idea edit` DX — $EDITOR-based editing for the no-text form.
>
> PROBLEM: `idea edit <query> "text"` requires retyping the entire replacement text inline. For long or multiline ideas this is hostile — the escape convention (`\n` sequences) makes hand-editing multiline text on the command line effectively impractical.
>
> BEHAVIOR (decided split):
> - `idea edit <query>` (no text argument) opens $EDITOR on a temp file containing the DECODED idea text (real newlines, real backslashes). On save+exit, re-encode per the escape convention and persist via the existing canonical writer. Pattern precedent: `git commit`, `kubectl edit`.
> - `idea edit <query> "text"` (text argument present) keeps current behavior exactly: replaces the whole text without opening an editor — the quick one-liner and scripting path. No flag needed to choose modes; presence/absence of the text arg is the switch.
> - Query semantics unchanged: substring, case-insensitive, matches ID or text; refuses and lists matches when ambiguous.
> - Editor resolution: $VISUAL then $EDITOR, sensible fallback (vi). stdlib os/exec only — dependency discipline forbids new deps (no survey/tea libs).
> - Save follows existing mutating-command semantics: whole-file normalize-on-write, date backfill stderr notice, non-idea lines preserved verbatim.
>
> OPEN QUESTIONS (solutioning): abort semantics; trailing-newline handling; non-interactive contexts; temp file content/extension; testing per Constitution V; update overview.md command table.

The entry was pre-solutioned at capture time: the two-form split, editor resolution order, and save semantics were decided in the backlog itself. The OPEN QUESTIONS block listed solutioning concerns; this intake resolves each as a graded assumption (see `## Assumptions`) — one was settled directly by the codebase (`idea.Edit` already refuses empty text at `src/internal/idea/idea.go:682`).

## Why

**Pain point.** `idea edit` currently demands the full replacement text inline (`Args: cobra.ExactArgs(2)` in `src/cmd/idea/edit.go`). Tweaking one word in a long idea means retyping the whole thing. For multiline ideas it is worse: the shell argument must contain *real* newline characters (typing literal `\n` in a quoted arg persists as an escaped backslash-n, i.e. the two characters, not a line break), so inline multiline editing requires awkward multi-line shell quoting. In practice long/multiline ideas are append-only.

**Consequence if unaddressed.** Users fall back to hand-editing `fab/backlog.md` for any non-trivial edit — but a multiline idea on disk is one physical line of escaped text, which is exactly the form humans cannot comfortably hand-edit. The tool exists to reduce friction over hand-editing (Constitution I rationale); the edit path currently re-creates that friction.

**Why this approach.** The $EDITOR temp-file round-trip is the established Unix pattern (`git commit`, `kubectl edit`): the user edits *decoded* text (real newlines, real backslashes) in their own editor, and the tool owns the encode/persist step. It needs zero new dependencies (stdlib `os/exec` only — Constitution's dependency discipline rules out survey/bubbletea-style TUI libs). Using the presence/absence of the text argument as the mode switch avoids a new flag and leaves the existing quick one-liner and scripting path byte-for-byte untouched.

## What Changes

### 1. `cmd/idea/edit.go` — two-form argument contract

- `Args: cobra.ExactArgs(2)` → `cobra.RangeArgs(1, 2)`.
- **Two-arg form (`idea edit <query> "text"`)**: behavior identical to today in every respect — replaces the whole text, never launches an editor, same validation, same output. This is the scripting path; it must not regress.
- **One-arg form (`idea edit <query>`)**: resolves the matched idea (identical query semantics: substring, case-insensitive, matches ID or text, via the existing `RequireSingle` — refuses and lists matches when ambiguous), opens the editor on a temp file containing the **decoded** text (`Idea.Text` — real newlines, real backslashes), and on clean editor exit reads the buffer back, post-processes it (CR-normalize, strip exactly one trailing LF), and persists via the existing `idea.Edit(path, query, newText, newID, newDate)` — inheriting the canonical writer wholesale: whole-file normalize-on-write, date backfill with stderr notice, atomic temp+rename save, non-idea lines preserved verbatim.
- Cobra `Long` help updated to describe both forms with examples (per the repo convention, `Short` stays byte-stable; the help-dump JSON `text` for the edit node updates automatically — schema unchanged).

### 2. `internal/idea` — editor plumbing (Constitution IV seam)

New helpers in `internal/idea` (suggested: a new `editor.go` beside `idea.go`); `cmd/idea/edit.go` stays wiring-only:

- `ResolveEditor() string` — returns `$VISUAL` if non-empty, else `$EDITOR` if non-empty, else `vi`. The returned value is a command *string* that may contain arguments (e.g. `code --wait`).
- `EditInEditor(text string) (string, error)` — the temp-file round trip:
  1. Create a temp file via `os.CreateTemp("", "idea-edit-*.md")` (`.md` extension for editor syntax highlighting); write `text` (raw decoded form); `defer os.Remove`.
  2. Launch the editor through the shell so multi-word values work, passing the path as a positional parameter to sidestep quoting:
     ```go
     cmd := exec.Command("sh", "-c", editor+` "$1"`, "sh", tmpPath)
     cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
     ```
  3. Editor exits non-zero (or fails to launch) → return the error; caller persists nothing.
  4. Editor exits 0 → read the file back, apply `normalizeCR`, strip **exactly one** trailing LF (editors conventionally append a final newline; stripping one means a one-line idea round-trips without gaining a `\n`; deliberate extra trailing blank lines beyond the final LF are kept as content), return the result.

No explicit TTY detection: the editor inherits the process stdio, and a TTY-requiring editor (vi) fails fast on its own with a non-zero exit in non-interactive contexts — satisfying "clear error, never hang" — while keeping `EDITOR=<script>` viable headlessly (which the test strategy depends on).

### 3. Edge semantics (one-arg form)

| Situation | Behavior | Exit |
|-----------|----------|------|
| Editor exits non-zero / fails to launch | No change to the backlog; error surfaced (`ERROR:` line via the existing RunE error path), message states the idea is unchanged | non-zero |
| Buffer unchanged (post-processed buffer == current `Idea.Text`) and no `--id`/`--date` given | No-op: no file rewrite (so no normalize-on-write side effect, no backfill), `note: text unchanged — nothing to do` to **stderr** | 0 |
| Buffer emptied (empty string after post-processing) | Refuse: no change, error to stderr (git's empty-commit-message abort precedent). Pre-check in cmd layer for a clear message; `idea.Edit`'s existing `text is required` guard (`idea.go:682`) is the safety net | non-zero |
| `--id`/`--date` passed with the one-arg form | Editor still opens; flags apply at save (existing `idea.Edit` params). The unchanged-text no-op is **suppressed** when either flag is present, so a metadata-only change still lands <!-- clarified: user confirmed — editor always opens in the no-text form; --id/--date apply at save; no-op suppressed when a flag is present --> | per outcome |

### 4. Output contract

- Success: same confirmation as today — `Updated: <FormatLine(i)>` (escaped single line) on **stdout**; backfill notice on stderr when applicable. stdout stays machine-parseable (Constitution VI).
- All no-op/abort messaging goes to **stderr** only, matching the backfill-notice channel policy (output-channel policy lives in `cmd/`, per Constitution IV).

### 5. Spec/doc updates

- `docs/specs/overview.md` command table: the edit row gains the two-form description (`idea edit <query>` opens $EDITOR; `idea edit <query> "text"` replaces inline).

### 6. Tests (Constitution V: table-driven, no mocks, real temp dirs)

Fake the editor with `EDITOR=<script>` (small shell scripts created in `t.TempDir()`), set via `t.Setenv` for in-process `newRootCmd()` tests or via `cmd.Env` for subprocess `buildBinary` tests. Cases:

1. One-arg form: editor script rewrites the buffer → file updated canonically, `Updated:` on stdout, exit 0.
2. Multiline round-trip: idea stored escaped (`a\nb`) → script asserts it received *decoded* text (records buffer to a side file), writes a new multiline buffer → persisted re-escaped on one physical line.
3. Trailing LF: script appends a final newline to a one-line idea → round-trips without gaining `\n`.
4. Abort: script exits non-zero → backlog byte-identical, non-zero exit, error mentions unchanged.
5. Unchanged buffer: script touches nothing → no file rewrite (byte-identical even with non-canonical lines present in the file — proves no normalize side effect), `note:` on stderr, exit 0.
6. Emptied buffer: script truncates the file → error, backlog unchanged, non-zero exit.
7. Resolution order: `$VISUAL` wins over `$EDITOR`; `$EDITOR` used when `$VISUAL` unset; `vi` fallback asserted via a direct `ResolveEditor` unit test.
8. Two-arg regression: `EDITOR` set to a tripwire script → never invoked; inline behavior unchanged.
9. `--date` with one-arg form and unchanged text → date change persists (no-op suppression).
10. Ambiguous query in one-arg form → refused with match list, editor never launched.

## Affected Memory

- `cli/structure`: (modify) — per-subcommand notes gain the edit two-form contract, editor resolution chain, temp-file round-trip, and edge/exit semantics.

## Impact

- `src/cmd/idea/edit.go` — arg contract (`RangeArgs(1, 2)`), mode switch, edge-case wiring, `Long` help text.
- `src/internal/idea/editor.go` (new) — `ResolveEditor`, `EditInEditor` (temp-file round trip, `sh -c` launch, post-processing). Reuses existing `normalizeCR`.
- `src/internal/idea/idea.go` — no behavioral change expected; `Edit` is reused as-is.
- `src/cmd/idea/main_test.go` + `src/internal/idea/` tests — new table-driven cases above.
- `docs/specs/overview.md` — edit row in the command table.
- No new dependencies. No change to the backlog format contract, other subcommands, or the help-dump JSON schema.

## Open Questions

(none — all decision points from the backlog's OPEN QUESTIONS block are resolved as graded assumptions below)

## Clarifications

### Session 2026-06-12

| Q | A |
|---|---|
| `idea edit <query>` (no text) combined with `--id`/`--date` — editor, metadata-only shortcut, or reject? | Editor opens; flags apply at save; unchanged-text no-op suppressed when a flag is present (assumption #9 → Certain) |

### Session 2026-06-12 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 8 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Mode switch is arg presence: `idea edit <query>` opens $EDITOR; `idea edit <query> "text"` keeps inline behavior exactly; no new flag | Decided in backlog entry at capture ("presence/absence of the text arg is the switch") | S:95 R:70 A:90 D:95 |
| 2 | Certain | Editor resolution `$VISUAL` → `$EDITOR` → `vi`; stdlib `os/exec` only | Decided in backlog; constitution dependency discipline forbids new deps | S:95 R:85 A:95 D:90 |
| 3 | Certain | Persist via existing `idea.Edit` canonical writer (normalize-on-write, backfill stderr notice, atomic save); query semantics unchanged | Decided in backlog; Constitution I/IV make any other path wrong | S:95 R:80 A:95 D:95 |
| 4 | Certain | Emptied buffer → error + no change + non-zero exit (not a silent abort) | Codebase answers it: `idea.Edit` already refuses empty text (`idea.go:682`); matches git's empty-commit-message abort | S:80 R:85 A:95 D:85 |
| 5 | Certain | Post-process buffer: `normalizeCR`, then strip exactly one trailing LF; further trailing blank lines are content | Clarified — user confirmed | S:95 R:90 A:80 D:75 |
| 6 | Certain | Launch via `sh -c '<editor> "$1"' sh <path>` with inherited stdio; no explicit TTY gate (TTY-requiring editors fail fast on their own) | Clarified — user confirmed | S:95 R:85 A:80 D:80 |
| 7 | Certain | Temp file: raw decoded text only, no comment header; `idea-edit-*.md` via `os.CreateTemp`; removed after | Clarified — user confirmed | S:95 R:90 A:75 D:75 |
| 8 | Certain | Unchanged buffer (and no `--id`/`--date`) → no-op: no rewrite, `note:` to stderr, exit 0 | Clarified — user confirmed | S:95 R:90 A:80 D:80 |
| 9 | Certain | `--id`/`--date` with one-arg form: editor still opens, flags apply at save, unchanged-text no-op suppressed when a flag is present | Clarified — user confirmed | S:95 R:85 A:60 D:55 |
| 10 | Certain | Editor helpers (`ResolveEditor`, `EditInEditor`) live in `internal/idea`; `cmd/idea/edit.go` stays wiring-only | Constitution III/IV dictate the seam | S:75 R:85 A:95 D:90 |
| 11 | Certain | Tests: table-driven, fake editor via `EDITOR=<script>`, real temp dirs/repos, no mocks | Constitution V + explicit in backlog entry | S:90 R:90 A:95 D:95 |

11 assumptions (11 certain, 0 confident, 0 tentative, 0 unresolved).
