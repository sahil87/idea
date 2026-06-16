# Intake: TTY-Aware Output Rendering

**Change**: 260613-kfcl-tty-aware-output-rendering
**Created**: 2026-06-13

## Origin

<!-- How was this change initiated? Include the user's raw input/prompt, the interaction
     mode (one-shot vs. conversational), and key decisions from the conversation. -->

> Conversational `/fab-discuss` → `/fab-new`. The trigger was a concrete DX pain point reported by the user: running `idea prune` dumps the full (often multi-hundred-word) text of every done idea, and the critical action prompt — `Re-run with --force to confirm.` — gets buried at the bottom of the terminal, off-screen. The same wall-of-text problem afflicts `idea ls`: short one-line ideas are lost between paragraph-length ones.

The discussion enumerated a DX backlog (labelled A–G) and triaged it. This change bundles the four features the user explicitly selected to build now — **A** (truncation + `--full` + `ls [id...]` filter), **B** (prune count header), **D** (color), **E** (interactive prune confirm). Two items were explicitly deferred:

- **G (section grouping via `## H2` headers, read + write side)** → backlogged to the **main** worktree as idea `ykwp` with full hand-off detail.
- **C (paging / `$PAGER`)** → decided against; a `--limit` cap may be revisited later if long lists remain painful after truncation.

Key decisions reached in conversation:
- The pipe-friendliness contract (Constitution VI; `prune.go:44` comment) is sacred — every display change MUST be TTY-gated so piped output stays full/canonical and machine-parseable.
- `golang.org/x/term` is the accepted dependency for isatty + terminal width (user confirmed "1 agreed"), justified under Constitution Dependency Discipline because stdlib has no isatty/width primitive.

## Why

<!-- What problem, what consequence, why this approach. -->

1. **Problem**: `idea ls` and `idea prune` (dry-run) print each idea's full untruncated text via `FormatLine`. Ideas in this project are frequently paragraph-length specs (e.g. `a36m`, `72oq` in the live backlog run >300 words each). On a TTY these soft-wrap into 20+ visual rows apiece, so: (a) the `prune` confirm hint scrolls off-screen and is missed; (b) `ls` is unscannable — short ideas drown between long ones; (c) the 4-char `[id]`/`date` prefix the user actually scans for is visually buried.

2. **Consequence if unfixed**: The reported failure mode persists — users miss the `--force` instruction and assume `prune` did nothing, or cannot find a specific idea in a long `ls`. The tool's whole reason for existing ("reduce friction over hand-editing a markdown file", per Constitution rationale) is undermined when reading the list is itself high-friction.

3. **Why this approach over alternatives**:
   - **Truncation over paging (C)**: paging requires `$PAGER`/`less` TTY-handoff plumbing that is fiddly in Go and overkill for backlogs of dozens (not thousands) of lines. Truncation + a count header + an id filter dissolve most of the need; a `--limit` cap is a cheaper future fallback than a real pager.
   - **Interactive confirm (E) over keeping the two-step `--force` dance**: the user's framing keeps `--force` working for scripts while making the interactive prompt the *last* line at the cursor — inherently un-buryable, which is the most direct fix for the reported pain. The original objection (breaking `--force` muscle memory) is resolved because `--force` still skips the prompt.
   - **`golang.org/x/term` over stdlib-only TTY detection**: stdlib can detect a char device (`os.Stdout.Stat()`) but has no terminal-width primitive; `$COLUMNS`/`tput cols` are hacky and not always available. `x/term` is the idiomatic quasi-official Go package and gives both isatty and width cleanly.

## What Changes

<!-- Be specific. Subsections per change area. Concrete examples and exact behavior. -->

### A. TTY-aware truncation, `--full` flag, and `ls [id...]` filter

**Truncation** (applies to `idea ls` and `idea prune` dry-run output):
- When stdout is a **TTY** and `--full` is not set, truncate each idea's **text portion** so the rendered line fits the terminal width, appending a single-character ellipsis `…` (U+2026) when truncated.
- The `[id] date:` prefix is the scannable anchor and MUST NEVER be truncated — only the trailing text is clipped.
- Multiline ideas (text containing escaped `\n`): truncate at the first newline regardless of width (so display is always one physical row), with the `…` indicating more follows.
- Truncation MUST be **rune-safe** — never cut mid-rune (use `[]rune` slicing or `utf8`-aware logic, not byte slicing). Width-awareness for wide glyphs (CJK/emoji) is a non-goal; rune-count fitting against terminal columns is the floor.
- When stdout is **NOT a TTY** (piped/redirected), emit full canonical text regardless of `--full` — preserves the pipe-friendliness contract. `--full` is only meaningful on a TTY.

**`--full` flag**: add to `idea ls` (and it implicitly disables truncation in `prune` dry-run if applicable — see Open Questions). Disables truncation, emits full text on a TTY.

**`ls [id...]` positional filter**: `idea list`/`idea ls` gains an optional variadic positional argument of idea IDs. When provided, only ideas whose ID is in the set are listed (still list-formatted, still respects `--sort`/`--reverse`/truncation/color). Invalid/unknown IDs: see Open Questions. `idea show <id>` remains the full-detail single-idea command; `ls <id> --full` overlapping with `show` is acceptable mild redundancy, not a conflict.

Example (TTY, 80-col terminal):
```
$ idea ls
- [ ] [oibl] 2026-03-14: unable to scroll terminal on mobile
- [ ] [rkx4] 2026-03-20: The text input dialog - show buttons at the bottom of t…
$ idea ls --full
- [ ] [rkx4] 2026-03-20: The text input dialog - show buttons at the bottom of that dialog in a section that represent 'most used commands' ...(full text)...
$ idea ls oibl rkx4        # filter to two ideas
```

### B. `idea prune` dry-run count summary header (stderr)

Before listing the prunable ideas, print a one-line summary to **stderr**:
```
2 done idea(s) would be pruned — re-run with --force to confirm.
```
- Goes to **stderr** (like the existing hint and the backfill notice), so `2>/dev/null` suppresses it consistently and **stdout still carries exactly the removable lines** (pipe-friendly, Constitution VI).
- Printed **before** the list so the action is the first thing a human sees, regardless of list length.
- Replaces / subsumes the trailing `Re-run with --force to confirm.` line, OR complements it — the header is the primary signal. (The trailing hint may be kept or dropped; leading header is the fix.)

### D. Color / styling (TTY-gated, `NO_COLOR`-aware)

When stdout is a **TTY** AND the `NO_COLOR` env var is **unset**:
- Dim (ANSI faint / `\033[2m`) the `[id] date:` prefix so the eye lands on the text.
- Color the done checkbox `[x]` green.
- Open `[ ]` checkboxes and the idea text render in the default color.

When stdout is **NOT a TTY** OR `NO_COLOR` is set: emit plain, uncolored output (current behavior). Applies to `idea ls` and `idea prune` dry-run. Color and truncation are independent but share the same TTY gate.

### E. Interactive confirm for `idea prune`

Decision matrix for `idea prune`:

| stdout TTY? | `--force`? | Behavior |
|---|---|---|
| Yes | No | List prunable ideas (with B's header + A's truncation + D's color), then prompt `Prune N done idea(s)? [y/N] ` on stderr; read a line from stdin; delete only on `y`/`yes` (case-insensitive); anything else aborts with no change. |
| Yes | Yes | Skip the prompt entirely; delete immediately (current `--force` behavior). |
| No | No | Do NOT prompt (would hang a pipe). Fall back to today's dry-run: list removable lines on stdout + stderr hint. |
| No | Yes | Delete immediately (current behavior). |

The prompt is the last line written, at the cursor — inherently visible. `--force` preserves script behavior unchanged.

### Shared internal helper

A small new internal seam (e.g. in `internal/idea`, or a tiny `internal/idea/term.go`) provides: `IsTTY(f *os.File) bool`, `TermWidth(f *os.File) int` (fallback width when undetectable — see Open Questions), and styling helpers (dim/green honoring `NO_COLOR`). `golang.org/x/term` is the single new direct dependency. Per Constitution IV, all logic lives in `internal/idea`; `cmd/` only wires flags and chooses TTY vs. plain rendering by asking the helper.

## Affected Memory

<!-- Spec-level behavior changes → memory updates. -->

- `cli/list`: (modify) `idea ls` gains TTY-aware truncation, `--full` flag, optional `[id...]` positional filter, and color output.
- `cli/prune`: (modify) `idea prune` dry-run gains a leading stderr count header and TTY-gated interactive `[y/N]` confirm; non-TTY/`--force` paths unchanged.
- `cli/index` (or the relevant cli sub-domain index): (modify) note the new `golang.org/x/term` direct dependency and the TTY/width/color rendering seam.

<!-- assumed: exact memory file paths under cli/ — the cli domain has 4 files (structure, line lifecycle, per-subcommand notes); the per-subcommand file(s) and dependency note are the likely targets. Hydrate will resolve precise paths. -->

## Impact

<!-- Affected code areas, APIs, dependencies, systems. -->

- **`src/cmd/idea/list.go`**: add `--full` flag and `[id...]` positional arg (`Args` changes from none to variadic id validation); branch rendering between TTY (truncate+color) and plain.
- **`src/cmd/idea/prune.go`**: add leading stderr count header; add TTY-gated `[y/N]` prompt; keep `--force` and non-TTY paths intact; preserve stdout = removable lines.
- **`src/internal/idea/idea.go`**: `FormatLine`/`DisplayLine` unchanged (machine/canonical); add a new display/truncation/styling helper consumed by the cmd layer.
- **New**: small `internal/idea` TTY/width/color seam wrapping `golang.org/x/term`.
- **Dependencies**: `go.mod`/`go.sum` gain `golang.org/x/term` (first non-stdlib/cobra direct dep) — justified per Constitution Dependency Discipline.
- **Tests**: table-driven tests for truncation (rune-safety, multiline-at-first-newline, prefix-never-truncated, ellipsis), id filter, count-header text, and the prune decision matrix. TTY-dependent behavior is tested by injecting the TTY/width decision (the helper seam) rather than allocating a real PTY — keeps tests `t.TempDir()`-style and deterministic per Constitution V.
- **No change** to: the backlog file format, the parser, `FormatLine` output, the `--json` schema (Constitution VI untouched — display-only change).

## Open Questions

<!-- SRAD prioritizes at apply entry. Just list. -->

- **Fallback terminal width** when `x/term.GetSize` fails (e.g. width 0 / not a real terminal but isatty true): default to 80 columns? Honor `$COLUMNS` first?
- **Unknown IDs in `ls [id...]`**: error out, or silently skip with a stderr warning, or print nothing? (Lean: warn-and-skip on stderr, list the rest — consistent with the pipe-friendly stdout posture.)
- **Does `--full` belong on `prune`** too, or only `ls`? The `prune` dry-run reuses the same rendering; a `--full` on prune would let users see full done-idea text before confirming. (Lean: add to both for symmetry, cheap.)
- **Truncation reserve for color codes**: ANSI codes are zero-width but counted as bytes — ensure width math counts *visible* runes, not escape sequences (apply color AFTER truncation, or measure pre-color).
- **Keep or drop the trailing `Re-run with --force to confirm.`** line once the leading header (B) exists? (Lean: drop it in TTY mode where the interactive prompt replaces it; keep it in non-TTY no-force fallback.)

## Assumptions

<!-- All four SRAD grades. Scores required. Unresolved → status context in Rationale. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `golang.org/x/term` for isatty + terminal width; first non-stdlib/cobra direct dep, justified per Dependency Discipline | User explicitly confirmed ("1 agreed"); stdlib has no width primitive; quasi-official package | S:95 R:80 A:90 D:90 |
| 2 | Certain | All display changes (truncate, color, confirm) are TTY-gated; piped output stays full/canonical to preserve Constitution VI pipe contract | Discussed and agreed as the recurring guardrail; constitution-mandated | S:95 R:75 A:95 D:90 |
| 3 | Certain | Logic lives in a new `internal/idea` TTY/width/color seam; `cmd/` only wires flags and picks rendering mode | Constitution IV mandates logic in internal/idea; matches existing structure | S:90 R:80 A:95 D:85 |
| 4 | Confident | Truncate only the text portion, never the `[id] date:` prefix; truncate multiline at first `\n`; rune-safe | Discussed explicitly; prefix is the scannable anchor, rune-safety is the correctness floor — minor latitude in ellipsis char/exact column budget | S:90 R:80 A:88 D:82 |
| 5 | Certain | B's count summary goes to stderr, printed before the list; stdout stays exactly the removable lines | Explicitly agreed in discussion + mandated by Constitution VI pipe contract; no alternative | S:92 R:85 A:92 D:88 |
| 6 | Certain | E: prompt only when TTY && !force; never prompt on non-TTY (would hang a pipe); --force always skips prompt | User authored this exact decision matrix in conversation; nothing left to decide | S:95 R:80 A:90 D:90 |
| 7 | Confident | D: dim `[id] date:` prefix, green `[x]`; gate on TTY && unset NO_COLOR | Discussed; NO_COLOR is the standard opt-out — exact ANSI shades have minor latitude | S:88 R:85 A:88 D:80 |
| 8 | Certain | Add `--full` to `ls`; add optional `[id...]` positional filter to `ls`; `show` stays the single-idea full-detail command | User explicitly requested both in conversation; `show` already exists so roles are fixed | S:92 R:80 A:88 D:85 |
| 9 | Certain | Skip paging/`$PAGER` (item C); a `--limit` cap may come later if needed | User explicitly decided against paging in conversation | S:95 R:90 A:88 D:90 |
| 10 | Certain | Section grouping (item G) is out of scope — deferred to main backlog `ykwp` (read + write side) | User explicitly instructed to backlog G separately and scope this change to A/B/D/E | S:95 R:90 A:90 D:92 |
| 11 | Confident | Unknown IDs in `ls [id...]` → warn on stderr and list the rest (rather than hard error) | Clear front-runner: consistent with the pipe-friendly stdout posture; trivially reversible | S:70 R:85 A:75 D:70 |
| 12 | Confident | Fallback terminal width = 80 cols (honoring `$COLUMNS` if set) when width is undetectable | 80 cols is the universal terminal default and `$COLUMNS`-first is the standard convention | S:75 R:85 A:80 D:75 |
| 13 | Confident | Add `--full` to `prune` dry-run too (symmetry with `ls`), and drop the trailing force-hint in TTY mode where the prompt replaces it | Cheap symmetric choice with an obvious front-runner; trivially reversible | S:70 R:85 A:75 D:72 |

13 assumptions (8 certain, 5 confident, 0 tentative, 0 unresolved).
