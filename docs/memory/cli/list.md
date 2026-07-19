---
description: "`idea list`/`ls` rendering contract: TTY-aware rune-safe text truncation, the `--full` flag, the optional `[id...]` positional filter, ANSI color (NO_COLOR-gated), and the pipe contract that keeps piped output canonical"
type: memory
---

# `idea list` / `ls` Subcommand

`idea list` (alias `ls`) lists ideas from the backlog. The cobra wrapper lives at `src/cmd/idea/list.go` (`listCmd()` factory); the TTY/width/color/truncation logic lives in `src/internal/idea/term.go` (Constitution IV seam). Open ideas show by default; `--all/-a` adds done ideas, `--done` shows only done, `--json` emits the structured records, `--sort` (`date`|`id`) and `--reverse` order them. The `ls` alias is documented in `structure.md` (§ Command aliases).

Why truncation exists: ideas in this project are frequently paragraph-length, so untruncated terminal output soft-wraps into many visual rows, short ideas drown between long ones, and the scannable `[id] date:` anchor is buried (260613-kfcl-tty-aware-output-rendering).

## TTY-aware rendering (truncation + color)

All display rendering is **TTY-gated** so piped output stays full and canonical (Constitution VI). The single render path is `printIdeaLines` in `cmd/idea/output.go` (see `structure.md` § Shared TTY-aware render path), shared with `prune`:

- **On a terminal** (stdout is a TTY): each idea renders via `idea.DisplayListLine(i, width, full, color)` — text truncated to the terminal width (unless `--full`), prefix dimmed and a done `[x]` greened (unless `NO_COLOR`/non-TTY).
- **Piped or redirected** (non-TTY): full canonical `FormatLine` output, regardless of `--full` — `--full` is meaningful only on a TTY.
- **`--json`** is unaffected in all cases: structured records (`id`, `date`, `status`, `text`) are emitted unchanged. The display features are display-only — `FormatLine`/`DisplayLine`, the parser, the backlog format, and the `--json` schema are all untouched.

### Truncation (`DisplayListLine` / `truncateText`)

`DisplayListLine` builds the canonical escaped line shape but clips only the **text portion**:

- The `- [done] [id] date: ` prefix is **NEVER** truncated — it is the scannable anchor. The available text width is `width - len([]rune(prefix))`.
- Truncation is **rune-safe**: `truncateText` operates on `[]rune`, never byte slices, so multibyte (CJK/emoji) text is never cut mid-rune. Wide-glyph display-width awareness is an explicit non-goal — rune-count against columns is the floor.
- A single-rune ellipsis `…` (U+2026, the `ellipsis` const) is appended when text is clipped.
- A **multiline** idea (escaped text containing a literal `\n` escape) is always clipped at the first newline with `…` — regardless of width — so a rendered list line is always exactly one physical row.
- **Degenerate width**: when the available text width is non-positive (prefix alone fills/exceeds the terminal) the text reduces to just `…`; when `avail <= 1` only the ellipsis is emitted. The prefix is still never clipped.

### `--full` flag

`--full` (boolean, default false) disables truncation on a TTY: full text is shown (still colored). It has no effect when piped (output is already full canonical there). `prune` carries the same flag for symmetry (see `prune.md`).

### Color (NO_COLOR-gated)

When `idea.UseColor(os.Stdout)` is true (TTY **and** `NO_COLOR` unset — presence disables color regardless of value per the NO_COLOR spec), `DisplayListLine` dims the `- `/`[id] date:` spans (ANSI faint `\033[2m`) and greens a done `[x]` checkbox (`\033[32m`). Color is applied **after** truncation so the width math counts visible runes, never escape bytes (see `structure.md` § term.go seam). The checkbox is rebuilt as its own span between two dim spans so the id/date stay faint while a done `[x]` stays green.

## Optional `[id...]` positional filter

`idea list`/`ls` accepts zero-or-more positional ID arguments (`Use: "list [id...]"`). The behavior:

| Argument | Behavior |
|----------|----------|
| (none) | List all ideas matching the active filter (`--all`/`--done`) + sort. |
| Well-formed IDs present in the backlog | List only those ideas, still respecting filter/`--sort`/`--reverse`/truncation/color. |
| Well-formed but **absent** ID (`zzzz`) | `warning: no idea with ID "zzzz"` on **stderr** (one line per missing ID), and the matched survivors are still listed (warn-and-list-the-rest — pipe-friendly stdout posture). |
| **Malformed** ID (not `[a-z0-9]{4}`) | Usage error via `idea.ValidateID` in the cobra `Args` validator — the command never runs. |

The split is deliberate: a malformed argument is a *usage mistake* (caught up front by `Args`), a well-formed-but-absent ID is a *not-found* condition (warn + continue). The filter lives in the `filterByIDs(cmd, ideas, args)` helper in `list.go`. `idea show <query>` remains the single-idea full-detail command; `ls <id> --full` overlapping `show` is accepted mild redundancy, not a conflict.

## Help text

`Long` documents the truncation/`--full`/`[id...]` behavior and the pipe contract; `Short` stays the byte-stable one-liner (repo convention, `structure.md` § Command help text). The help-dump JSON schema is unchanged — the list node's `text` updates automatically since it reproduces `-h` output (including cobra's `Aliases: list, ls` line).

## Tests

- `src/cmd/idea/main_test.go` — `TestList_IDFilter` (filter to listed IDs; unknown-ID stderr warning naming the missing ID with survivors listed; malformed-ID usage error) and `TestList_PipedOutputIsCanonical` (piped `ls` / `ls --full` is byte-identical to the `FormatLine` listing — no ANSI, no `…`), via the existing `buildBinary`/`setupGitRepo`/`writeRepoBacklog`/`runSplit` helpers.
- `src/internal/idea/term_test.go` — `DisplayListLine`/`truncateText` rune-safety, prefix-never-truncated, ellipsis presence, multiline-at-first-newline, `full` bypasses truncation, and color-applied-after-truncation (see `structure.md` for the term-seam test list).

## Cross-references

- Source-tree placement, the `ls` alias and bare-text namespace rule, the `term.go` TTY/width/color/truncation seam, and the shared `printIdeaLines` render path: `structure.md`.
- The same TTY-aware rendering applied to the prune dry-run, plus the count header and interactive confirm: `prune.md`.
- Command table: `../../specs/overview.md`.
- Constitution Principles IV (logic in `internal/idea`) and VI (machine-parseable stdout): `fab/project/constitution.md`.
- Originating change: `260613-kfcl-tty-aware-output-rendering`.
