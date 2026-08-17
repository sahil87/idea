# Intake: Stale-Idea Surfacing on `idea list`

**Change**: 260816-szds-stale-idea-surfacing
**Created**: 2026-08-17

## Origin

Backlog idea `[szds]` (2026-08-17), invoked one-shot via `/fab-new szds` — no prior design conversation. Raw backlog text:

> DX: stale-idea surfacing on 'idea list' — turn the stored date from decoration into the review signal it's meant to be. Two independent pieces: (1) 'idea list --stale <duration>' (e.g. --stale 90d; decide accepted units — days only is fine, parse '90d'/'90') filters to open ideas whose date is older than the cutoff; composes with existing --json/--sort/--reverse/--full and the [id...] positional filter. (2) TTY-only age dimming: on a terminal, render ideas past the staleness threshold with the existing dim/color seam (internal/idea term.go render path), gated by NO_COLOR exactly like current coloring, NEVER affecting piped output (pipe contract in docs/memory/cli/list.md keeps piped output canonical). No format change, no new stored fields — Constitution VI schema {id,date,status,text} untouched; --json output unchanged except fewer rows under --stale. Open question for the implementer: should --stale imply open-only (recommended; done ideas are prune's business) and what default threshold, if any, drives the dimming when --stale is not passed (suggest a fixed 90d constant, named per code-quality no-magic-numbers). Table-driven tests over date math incl. same-day boundary.

## Why

1. **Pain point**: every idea line stores a `YYYY-MM-DD` date, but nothing consumes it beyond sorting — it is decoration. Backlogs accumulate months-old open ideas that nobody re-reviews, and `idea list` gives no way to ask "what has been sitting here untouched?" or to see age at a glance.
2. **Consequence of not fixing**: the backlog silently rots. Old ideas drown among new ones, the review loop the tool exists to support (capture → periodically review → do or prune) has no surfacing mechanism, and users fall back to eyeballing dates or shell-piping into `awk`.
3. **Why this approach**: both pieces reuse machinery that already exists — the stored date (no new fields, no format change) and the TTY/color render seam in `internal/idea/term.go` (no new rendering path). A filter flag (`--stale`) serves scripted/deliberate review; passive dimming serves everyday `idea list` glances. They are independent and complementary, and neither touches the machine-readable contracts (Constitution VI: line format and `--json` schema `{id,date,status,text}` untouched; piped output stays canonical).

## What Changes

### 1. `idea list --stale <duration>` filter flag

New string flag `--stale` on `list`/`ls` (`src/cmd/idea/list.go`):

- **Value syntax**: days only. Accepts `90d` or bare `90` (both mean 90 days). Parsing lives in `internal/idea` (e.g. `ParseStaleDuration(s string) (int, error)`); rejects negative numbers, non-integer values, and any unit other than a trailing `d` (`90h`, `3w` → usage error via the existing usage-error convention, exit 2).
- **Semantics**: keeps only ideas whose date is **strictly older** than the cutoff (`date < today − N days`). An idea dated exactly `today − N days` is NOT stale — "older than the cutoff", with a table-driven same-day boundary test.
- **Implies open-only**: `--stale` filters open ideas (done ideas are prune's business, per the backlog recommendation). Combining `--stale` with `--done` or `--all` is a usage error (cobra `MarkFlagsMutuallyExclusive` or equivalent explicit check) — an explicit conflict beats a silent override.
- **Composes** with everything else unchanged: `--json` (same schema, fewer rows), `--sort`/`--reverse` (ordering applied to the filtered set), `--full`, and the `[id...]` positional filter (intersection: requested IDs that are also stale; absent-ID stderr warnings unchanged).
- **Dateless ideas** (`Date == ""` — lenient-parsed lines not yet backfilled by a save): age is uncomputable, so they are never stale — excluded from `--stale` results and never age-dimmed.
- Filtering logic lives in `internal/idea` (Constitution IV); `list.go` only wires the flag and passes the parsed threshold down. The stale predicate takes `today` as a parameter (cmd layer passes `time.Now()`) so tests inject fixed dates.

Example:

```
idea list --stale 90d        # open ideas older than 90 days
idea list --stale 90 --json  # same, structured output (schema unchanged)
idea ls a7k2 --stale 30d     # ID filter ∩ stale filter
```

### 2. TTY-only age dimming on `idea list`

On a terminal, ideas past the staleness threshold render **entirely dim** — the existing ANSI faint style (`ansiFaint` in `src/internal/idea/term.go`) extends from the prefix to the whole line (text included), so stale ideas visually recede exactly the way the `[id] date:` prefix already does:

- **Gating is identical to current coloring**: applied only when `idea.UseColor(os.Stdout)` is true (TTY AND `NO_COLOR` unset — presence disables, per the NO_COLOR spec). Piped/redirected output stays full canonical `FormatLine` bytes — the pipe contract in `docs/memory/cli/list.md` is untouched. `--json` unaffected.
- **Threshold**: when `--stale` is not passed, a fixed named constant (e.g. `const defaultStaleDimDays = 90` in `internal/idea`, per the code-quality no-magic-numbers rule) drives dimming. When `--stale N` IS passed, the effective threshold is N — one rule ("past the effective threshold → dim"), no special case; a `--stale` result set therefore renders all-dim, which is consistent (everything listed is, by construction, stale). <!-- assumed: dimming keys on the effective threshold (--stale value when passed, else the 90d constant) rather than suppressing dimming under --stale — one rule beats a special case; display-only and trivially reversible -->
- **Style mechanics**: dimming is applied after truncation (existing rule — width math counts visible runes, never escape bytes). A stale done `[x]` interaction cannot occur in practice (`--stale` implies open-only; default list is open-only), but the render seam handles the combination gracefully if a caller ever passes it.
- **Scope**: `idea list`/`ls` only. `prune`'s dry-run listing shares `printIdeaLines` (`src/cmd/idea/output.go`), so the staleness threshold is threaded as a parameter that `prune` does not pass — prune output is unchanged.
- **No new stored fields, no format change**: `FormatLine`, `DisplayLine`, the parser, the backlog line format, and the `--json` schema are all untouched.

### 3. Help text + docs surface

- `list.go` `Long` gains the `--stale`/dimming description; `Short` stays byte-stable (repo convention). The help-dump JSON updates automatically (it reproduces `-h` output).
- CLI-surface change ⇒ check against toolkit standards (`shll standards`: `help-dump`, `readme-extraction`, `skill`) — README / `docs/site/` / the embedded skill bundle mention `list` behavior and may need the flag added where the list surface is documented.

### 4. Tests

Table-driven (Constitution V), real temp dirs:

- `internal/idea`: duration parsing (`90d`, `90`, `0`, negative, garbage, `90h`), stale predicate date math including the same-day boundary (`today − N` exactly ⇒ not stale), dateless-idea exclusion, dim-rendering with injected width/color/threshold.
- `cmd/idea` (binary tests, existing `buildBinary`/`writeRepoBacklog`/`runSplit` helpers): `--stale` filters rows (incl. `--json` row count), `--stale` + `--done`/`--all` usage error, composition with `[id...]`/`--sort`/`--reverse`, and piped output stays canonical (no ANSI, no `…`) with `--stale`.

## Affected Memory

- `cli/list`: (modify) add the `--stale` filter contract (units, strict-older-than boundary, open-only implication, mutual exclusion with `--done`/`--all`, composition rules, dateless exclusion) and the age-dimming section (effective threshold, whole-line faint, UseColor gating, prune unaffected)
- `cli/structure`: (modify) light touch — the term.go render-seam summary and shared `printIdeaLines` note gain the staleness-threshold parameter

## Impact

- `src/cmd/idea/list.go` — flag wiring, mutual-exclusion validation, help text
- `src/cmd/idea/output.go` — `printIdeaLines` signature gains the staleness threshold (prune call site passes the no-dimming value)
- `src/internal/idea/term.go` (or a sibling `stale.go`) — duration parsing, stale predicate, whole-line dim rendering
- `src/cmd/idea/main_test.go`, `src/internal/idea/term_test.go` (+ possibly a new `stale_test.go`) — tests above
- Help-dump JSON output changes implicitly (list node `-h` text); `docs/site/` / README / skill bundle wherever the list surface is enumerated
- No dependency changes; no format or schema changes

## Open Questions

None — the backlog entry raised two implementer questions (open-only implication; default dim threshold) and supplied recommendations for both; they are graded as assumptions below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Duration units are days only; `90d` and bare `90` both parse as 90 days | Backlog states this verbatim ("days only is fine, parse '90d'/'90'") | S:90 R:85 A:90 D:85 |
| 2 | Confident | `--stale` implies open-only filtering | Backlog recommends it explicitly ("done ideas are prune's business") | S:80 R:75 A:80 D:75 |
| 3 | Confident | `--stale` + `--done`/`--all` is a usage error, not a silent override | Backlog silent on the combination; explicit conflict is the idiomatic cobra posture and matches the repo's usage-error (exit 2) convention | S:50 R:80 A:60 D:55 |
| 4 | Certain | Default dim threshold is a fixed named constant, 90 days | Backlog suggests exactly this ("a fixed 90d constant, named per code-quality no-magic-numbers") | S:85 R:85 A:85 D:80 |
| 5 | Tentative | Dimming keys on the effective threshold (`--stale` value when passed, else the 90d constant) — no dim-suppression special case under `--stale` | Backlog doesn't address it; "one rule" vs "suppress when filtered" are both defensible; display-only and trivially reversible | S:30 R:80 A:45 D:30 |
| 6 | Confident | Staleness is strictly-older-than: an idea dated exactly `today − N` is NOT stale | Backlog wording "older than the cutoff" plus its explicit same-day-boundary test callout | S:75 R:85 A:80 D:70 |
| 7 | Confident | Dateless ideas (`Date == ""`) are never stale — excluded from `--stale`, never dimmed | Age is uncomputable; lenient-read lines are backfilled at first save anyway, so the safe default costs nothing | S:60 R:85 A:80 D:75 |
| 8 | Certain | Parsing/predicate/dim logic lives in `internal/idea`; `cmd/list.go` wires flags only | Constitution IV mandates the split; term.go is the named seam | S:85 R:80 A:95 D:90 |
| 9 | Certain | Dimming gated by `UseColor` (TTY + NO_COLOR presence); piped output canonical; `--json` schema untouched | Backlog states all three; Constitution VI and the existing pipe contract lock them in | S:95 R:85 A:95 D:95 |
| 10 | Confident | Dimming scoped to `idea list` only; `printIdeaLines` threads the threshold as a parameter prune does not pass | Backlog titles the feature "on 'idea list'"; prune's dry-run is a consent surface, not a review surface | S:50 R:80 A:60 D:50 |
| 11 | Confident | Stale predicate takes `today` as an injected parameter; cmd layer passes `time.Now()` | Constitution V table-driven date-math tests require a fixed clock; matches the render seam's inject-width pattern | S:55 R:85 A:80 D:80 |
| 12 | Confident | `--stale` value validation: non-negative integer with optional `d`; negative/garbage/other units are usage errors (exit 2) | Repo has an established 0/1/2 usage-error convention; days-only is decided by #1 | S:55 R:85 A:75 D:70 |

12 assumptions (4 certain, 7 confident, 1 tentative, 0 unresolved).
