---
description: "`idea list`/`ls` rendering contract: TTY-aware rune-safe text truncation, the `--full` flag, the optional `[id...]` positional filter, the `--stale` age filter and TTY-only age dimming past the staleness threshold, ANSI color (NO_COLOR-gated), and the pipe contract that keeps piped output canonical"
type: memory
---

# `idea list` / `ls` Subcommand

`idea list` (alias `ls`) lists ideas from the backlog. The cobra wrapper lives at `src/cmd/idea/list.go` (`listCmd()` factory); the TTY/width/color/truncation logic lives in `src/internal/idea/term.go` and the staleness parsing/predicate in `src/internal/idea/stale.go` (Constitution IV seams). Open ideas show by default; `--all/-a` adds done ideas, `--done` shows only done, `--stale <N>` keeps only open ideas older than N days, `--json` emits the structured records, `--sort` (`date`|`id`) and `--reverse` order them. The `ls` alias is documented in `structure.md` (§ Command aliases).

Why truncation exists: ideas in this project are frequently paragraph-length, so untruncated terminal output soft-wraps into many visual rows, short ideas drown between long ones, and the scannable `[id] date:` anchor is buried (260613-kfcl-tty-aware-output-rendering).

## TTY-aware rendering (truncation + color)

All display rendering is **TTY-gated** so piped output stays full and canonical (Constitution VI). The single render path is `printIdeaLines` in `cmd/idea/output.go` (see `structure.md` § Shared TTY-aware render path), shared with `prune`:

- **On a terminal** (stdout is a TTY): each idea renders via `idea.DisplayListLine(i, width, full, color, stale)` — text truncated to the terminal width (unless `--full`), prefix dimmed and a done `[x]` greened (unless `NO_COLOR`/non-TTY), and an idea past the effective staleness threshold rendered whole-line faint (see Age dimming below).
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

### Age dimming (stale ideas, TTY-only)

On a color terminal, an **open** idea past the **effective staleness threshold** renders **whole-line faint** — the (already-truncated) text portion joins the dim spans so aged ideas visually recede exactly the way the prefix does. The effective threshold is the `--stale` value when passed, else the fixed named constant `idea.DefaultStaleDimDays` (90) in `internal/idea/stale.go` — one rule, no special case, so a `--stale` result set renders all-dim (everything listed is stale by construction). Details:

- **Gating is identical to prefix coloring** — the same `idea.UseColor(os.Stdout)` check (TTY AND `NO_COLOR` unset) — and dimming is applied **after** truncation, so width math still counts visible runes, never escape bytes.
- **Done ideas are never age-dimmed** — dimming is an open-idea review signal (done ideas are prune's business), so `printIdeaLines` gates the stale flag on `!i.Done` and a done idea listed via `--done`/`--all` keeps its normal rendering however old it is. If a caller ever passes the stale+done combination anyway, the render seam handles it gracefully: a stale done `[x]` keeps its green — the explicit state signal outranks the age hint, and the span structure (dim spans around the checkbox) stays intact.
- **`prune` is unaffected**: `printIdeaLines` threads the threshold as a parameter (`staleDays int` plus the shared `today` clock), and prune's call site passes the named sentinel `idea.NoStaleDim` (-1), which disables age dimming entirely — prune's dry-run listing is a consent surface, not a review surface. 0 cannot be the sentinel because `--stale 0` is a valid threshold ("dated before today").
- **Machine contracts untouched**: piped/redirected output stays full canonical `FormatLine` bytes (no ANSI, no `…`) including under `--stale`, and the `--json` schema is unchanged.
- Dateless ideas (`Date == ""`) are never age-dimmed — their age is uncomputable.

## Optional `[id...]` positional filter

`idea list`/`ls` accepts zero-or-more positional ID arguments (`Use: "list [id...]"`). The behavior:

| Argument | Behavior |
|----------|----------|
| (none) | List all ideas matching the active filter (`--all`/`--done`) + sort. |
| Well-formed IDs present in the backlog | List only those ideas, still respecting filter/`--sort`/`--reverse`/truncation/color. |
| Well-formed but **absent** ID (`zzzz`) | `warning: no idea with ID "zzzz"` on **stderr** (one line per missing ID), and the matched survivors are still listed (warn-and-list-the-rest — pipe-friendly stdout posture). |
| **Malformed** ID (not `[a-z0-9]{4}`) | Usage error via `idea.ValidateID` in the cobra `Args` validator — the command never runs. |

The split is deliberate: a malformed argument is a *usage mistake* (caught up front by `Args`), a well-formed-but-absent ID is a *not-found* condition (warn + continue). The filter lives in the `filterByIDs(cmd, ideas, args)` helper in `list.go`. `idea show <query>` remains the single-idea full-detail command; `ls <id> --full` overlapping `show` is accepted mild redundancy, not a conflict.

## `--stale <duration>` age filter

`--stale <duration>` filters the listing to **open** ideas strictly older than the cutoff (`date < today − N days`):

- **Value syntax**: days only — a non-negative integer with an optional trailing `d`, so `90d` and bare `90` both mean 90 days. Parsing is `idea.ParseStaleDays` in `internal/idea/stale.go` (Constitution IV seam); negative numbers, non-integers, the empty string, and any other unit (`90h`, `3w`) are rejected as usage errors (exit 2), validated up front in `RunE` so a bad value fails even before the backlog is read.
- **Strictly older than**: an idea dated exactly `today − N days` is NOT stale (the same-day boundary). `--stale 0` is valid and means "dated before today".
- **Implies open-only**: done ideas are prune's business, so `--stale` cannot combine with `--done` or `--all` — the combination is a usage error (exit 2) rejected by an explicit `PreRunE` check. Cobra's `MarkFlagsMutuallyExclusive` is also declared, but its group validation runs after `PreRunE` and classifies nothing, so the explicit check is what preserves the repo's 0/1/2 exit-code convention.
- **Dateless ideas** (`Date == ""` — lenient-read lines not yet backfilled by a save) have uncomputable age and are never stale: excluded from `--stale` results.
- **Composition** is unchanged for everything else: `--json` (same `{id,date,status,text}` schema, fewer rows), `--sort`/`--reverse` (applied to the filtered set), `--full`, and the `[id...]` positional filter (intersection — the ID filter runs first, so a requested ID that exists but is fresh drops silently, with stderr warnings reserved for genuinely absent IDs).

The predicate is `idea.IsStale(i Idea, days int, today time.Time)` — `today` is a parameter (the cmd layer passes `time.Now()`) so table-driven tests inject a fixed clock; one clock serves both the filter pass and the dim pass.

## Help text

`Long` documents the truncation/`--full`/`[id...]` behavior, the `--stale` filter (units, open-only implication), the age dimming (effective threshold, TTY/`NO_COLOR` gating), and the pipe contract; `Short` stays the byte-stable one-liner (repo convention, `structure.md` § Command help text). The help-dump JSON schema is unchanged — the list node's `text` updates automatically since it reproduces `-h` output (including cobra's `Aliases: list, ls` line).

## Design Decisions

### Lexicographic ISO-date comparison for the stale predicate
**Decision**: `IsStale` computes the cutoff as `today.AddDate(0, 0, -days).Format("2006-01-02")` and compares `i.Date < cutoff` as strings.
**Why**: Stored dates are validated `YYYY-MM-DD` (zero-padded ISO), where lexicographic order equals chronological order — no per-idea `time.Parse`, no error path on the render/filter hot loop.
**Rejected**: `time.Parse` per idea — introduces a parse-error branch for strings the parser already validated, for zero correctness gain.
*Introduced by*: 260816-szds-stale-idea-surfacing

### No-dimming sentinel threaded through the shared render path
**Decision**: `printIdeaLines` takes a `staleDays` parameter with the named sentinel `idea.NoStaleDim` (-1) meaning "no age dimming"; `prune` passes the sentinel, `list` passes the effective threshold.
**Why**: Keeps the single shared render path (one home for the list/prune rendering policy) while scoping dimming to `list`; `--stale 0` is a valid threshold ("older than today"), so 0 cannot be the sentinel.
**Rejected**: A separate list-only render function — duplicates the TTY/width/color mode selection the shared path exists to centralize.
*Introduced by*: 260816-szds-stale-idea-surfacing

### Dimming keys on the effective threshold
**Decision**: One rule — "past the effective threshold ⇒ dim" — where the effective threshold is the `--stale` value when passed, else the 90-day default constant. A `--stale` result set renders all-dim.
**Why**: One rule, no special case; all-dim under `--stale` is consistent (everything listed is, by construction, stale).
**Rejected**: Suppressing dimming when `--stale` is passed — a special case whose only benefit is cosmetic.
*Introduced by*: 260816-szds-stale-idea-surfacing

### Done-checkbox green outranks age faint
**Decision**: If a done idea ever renders stale (unreachable via `list` — `--stale` implies open-only and the default filter is open-only), the `[x]` keeps its green while the rest of the line dims.
**Why**: Explicit state signal beats an age hint; also keeps `DisplayListLine`'s span structure (dim spans around the checkbox) intact.
**Rejected**: Fainting the checkbox too — loses the done signal for zero simplification (the spans already exist).
*Introduced by*: 260816-szds-stale-idea-surfacing

## Tests

- `src/cmd/idea/main_test.go` — `TestList_IDFilter` (filter to listed IDs; unknown-ID stderr warning naming the missing ID with survivors listed; malformed-ID usage error), `TestList_PipedOutputIsCanonical` (piped `ls` / `ls --full` is byte-identical to the `FormatLine` listing — no ANSI, no `…`), `TestList_Stale` (`--stale 90d` and bare `--stale 90` against a fixed-date backlog; same-day boundary not stale; `--stale --done`/`--stale --all` and invalid values exit 2; composition with `[id...]`, `--sort`, `--reverse`; piped `--stale` output stays canonical) and `TestList_StaleJSON` (`--json` row count with the unchanged schema), via the existing `buildBinary`/`setupGitRepo`/`writeRepoBacklog`/`runSplit` helpers.
- `src/internal/idea/stale_test.go` — `TestParseStaleDays` (accepts `90d`/`90`/`0`; rejects `-5`, `abc`, `90h`, `3w`, empty) and `TestIsStale` (strictly-older-than date math with an injected `today`, the same-day boundary, dateless-idea exclusion).
- `src/internal/idea/term_test.go` — `DisplayListLine`/`truncateText` rune-safety, prefix-never-truncated, ellipsis presence, multiline-at-first-newline, `full` bypasses truncation, and color-applied-after-truncation, plus `TestDisplayListLine_Stale` (whole-line faint when stale, stale done `[x]` keeps green, no-color path emits no codes) (see `structure.md` for the term-seam test list).

## Cross-references

- Source-tree placement, the `ls` alias and bare-text namespace rule, the `term.go` TTY/width/color/truncation seam, the `stale.go` staleness seam, and the shared `printIdeaLines` render path: `structure.md`.
- The same TTY-aware rendering applied to the prune dry-run (which passes the no-dimming sentinel), plus the count header and interactive confirm: `prune.md`.
- Command table: `../../specs/overview.md`.
- Constitution Principles IV (logic in `internal/idea`) and VI (machine-parseable stdout): `fab/project/constitution.md`.
- Originating changes: TTY-aware rendering — `260613-kfcl-tty-aware-output-rendering`; stale-idea surfacing (`--stale` + age dimming) — `260816-szds-stale-idea-surfacing`.
