# Intake: Resilient Backlog Parser

**Change**: 260610-wtmn-resilient-backlog-parser
**Created**: 2026-06-10
**Status**: Draft

## Origin

> Make the `idea` command more resilient. Concrete failure: `idea` can't parse the
> idea list in `fab/backlog.md` in `~/code/sahil87/shll.ai/`.

Originated from a `/fab-discuss` session on 2026-06-10. The user reported that
`idea list` in the shll.ai repo shows zero items despite the backlog being full of
idea-looking lines. Investigation traced the root cause to the parser regex
(`src/internal/idea/idea.go:57`):

```go
var lineRegex = regexp.MustCompile(`^- \[([ x])\] \[([a-z0-9]{4})\] (\d{4}-\d{2}-\d{2}): (.+)$`)
```

The shll.ai backlog uses the format documented in *its own* header —
`- [ ] [{id}] {description}` — with **no date and no colon**. Every line fails the
date-mandatory regex and falls through `LoadFile` (`idea.go:167`) as inert
pass-through content. Result: silent failure — `idea list` prints nothing, no error,
no warning, and the items are invisible to `show`/`done`/`edit`/`rm`.

The discussion surfaced two distinct problems wearing one coat: (1) a format
mismatch (two real backlogs disagree on whether the date is mandatory), and (2) the
deeper *silent-failure* UX gap — a file full of idea-looking lines parses to nothing
with no signal. Decisions below were made interactively with the user during that
session (interaction mode: conversational `/fab-discuss` → `/fab-new`).

## Why

**The pain point.** `idea` is meant to reduce friction over hand-editing a markdown
backlog. But it silently ignores lines that *look exactly like ideas* — same `- [ ]`
checkbox, same 4-char `[id]` — solely because they omit an optional-feeling
`YYYY-MM-DD:` segment. To a user, `idea list` returning nothing on a populated file
reads as "the tool is broken," not "the tool is strict." That silence is the core
resilience defect.

**The consequence of inaction.** Any backlog authored by hand, by an external tool,
or by a slightly different convention (the shll.ai header documents a dateless form)
is invisible to `idea`. Cross-tool sharing of `fab/backlog.md` — an explicit design
goal (see `backlog-format.md`) — breaks the moment the other tool's convention drifts
from idea's strict Shape A. The tool becomes brittle exactly where it should be
forgiving.

**Why this approach (lenient read, canonical write).** Two interpretations were
considered and rejected in favor of a third:

1. *"shll.ai is wrong, migrate it"* — rejected: punts the brittleness; the next
   divergent backlog re-triggers the failure. Doesn't make `idea` more resilient.
2. *"Keep strict format, just warn on near-misses"* — rejected as insufficient on its
   own: surfaces the problem but still refuses to read a perfectly sensible backlog.

The chosen approach makes `idea` **accept messy/variant input but own exactly one
canonical output format**. This is the standard robustness principle (be liberal in
what you accept, strict in what you emit). It fixes the concrete shll.ai case *and*
hardens the parser against the broader class of real-world format drift, while
keeping a single, stable, machine-parseable output contract.

## What Changes

The change is concentrated in `src/internal/idea/idea.go` (the parser core) and its
table-driven test file, plus the two affected spec documents. `add` is explicitly
**out of scope** — it already stamps today's date; this change is purely about
parsing *existing* files more leniently and the resulting save behavior.

### 1. Lenient parser regex (read side)

Relax `lineRegex` so the date segment is optional and common surface-form variants
are accepted. Target shape (final form to be confirmed in plan):

```go
// Accepts: optional leading whitespace; -, *, or + bullet; optional "YYYY-MM-DD: ".
var lineRegex = regexp.MustCompile(`^\s*[-*+] \[([ x])\] \[([a-z0-9]{4})\] (?:(\d{4}-\d{2}-\d{2}): )?(.+)$`)
```

Both of these MUST now parse to the same `Idea` (modulo `Date`):

```
- [ ] [rk7t] 2026-06-10: Tune the README-extraction reporter   → Date="2026-06-10"
- [ ] [rk7t] Tune the README-extraction reporter               → Date="" (dateless)
```

Variant dimensions accepted on **input**:

| Dimension | Accepted on input | Example |
|-----------|-------------------|---------|
| Date | present **or** absent | `[id] 2026-06-10: text` / `[id] text` |
| Bullet marker | `-`, `*`, `+` | `* [ ] [id] text` |
| Leading whitespace | any (indentation, tabs) | `␣␣- [ ] [id] text` |
| Line endings | CRLF or LF | `...text\r\n` |

CRLF handling: `LoadFile` currently splits on `\n` only (`idea.go:160`). It must strip
a trailing `\r` from each line before parsing so CRLF files parse correctly. The
stripped `\r` is part of canonicalization — output is always LF.

**Precision guard:** the relaxed regex MUST NOT start matching genuine non-idea prose.
The `- [ ]` checkbox + `[a-z0-9]{4}` id anchors keep false-positive risk low, but this
needs explicit negative test cases (e.g. a prose line beginning `- [ ] [abcd] ...`
that is *intended* as prose is an acceptable edge — the anchors mean we treat it as an
idea; the test documents the boundary).

### 2. Canonical output (`FormatLine`, write side)

`FormatLine` (`idea.go:99`) stays the single source of output truth and continues to
emit the canonical form:

```go
return fmt.Sprintf("- [%s] [%s] %s: %s", i.StatusCheck(), i.ID, i.Date, i.Text)
```

Output is always: `-` bullet, no leading whitespace, `YYYY-MM-DD:` present, LF
line endings (via `SaveFile`'s `strings.Join(result, "\n")` at `idea.go:188`). This
is unchanged code — the canonicalization "falls out" of the existing
regenerate-on-save architecture (`SaveFile` rebuilds every idea line from
`FormatLine`, `idea.go:184`).

### 3. Date backfill on save

When a dateless idea (`Date == ""`) is saved, backfill **today's date**
(`time.Now().Format("2006-01-02")`) so every persisted idea has a date. Decision: do
**not** preserve datelessness. The cleanest seam is to stamp the date at save time
(in `SaveFile` or just before it) for any idea whose `Date` is empty — this also keeps
`MarshalJSON` correct, since by marshal time the in-memory `Idea` has a date.

Worked example (`idea done rk7t` on the shll.ai dateless line):

```
in:  - [ ] [rk7t] Tune the README-extraction reporter
out: - [x] [rk7t] 2026-06-10: Tune the README-extraction reporter
```

### 4. Normalize-on-write (accepted trade-off)

Because `SaveFile` regenerates **all** idea lines (not just the edited one), the first
mutating command (`done`/`edit`/`rm`) canonicalizes every recognized idea line in the
file at once: variant bullets → `-`, indentation stripped, CRLF → LF, dateless →
dated. This is a deliberate, accepted trade-off. Consequence to keep visible: a single
`idea done` on one item can produce a large git diff if the file had many
variant/dateless lines. Non-mutating commands (`list`, `show`) never rewrite the file,
so pure reads are diff-free.

### 5. Non-idea lines unchanged

Lines that do not parse as ideas (headers, blank lines, prose, and lines that still
fail the *relaxed* regex) remain verbatim pass-through exactly as today
(`LoadFile` `else` branch, `idea.go:167`). Constitution Principle I (round-trip
preservation of non-idea content) is preserved.

### 6. Spec / docs rewrite (part of this change, not a follow-up)

This change contradicts the *letter* of the current format contract, so the docs must
move with the code:

- **`docs/specs/backlog-format.md`**: Currently declares the date part of the public
  API that "will not change without a major-version bump and a documented migration
  path," and frames the second-bracket `[issue_ids]` form as "Shape B pass-through."
  Rewrite to: date is **optional on input**, output is **canonical** (`-` bullet,
  LF, date always present, backfilled if absent), and document the accepted input
  variants (bullet markers, leading whitespace, CRLF). Note the format-contract change
  explicitly and that output remains a stable machine-parseable contract.
- **`docs/specs/overview.md`**: Update any description of parse/format behavior to
  match lenient-read/canonical-write.

### 7. Tests (table-driven, real temp dirs — Constitution V)

Add table-driven cases in `src/internal/idea/idea_test.go` covering, at minimum:

- Parse: dateless line; CRLF line; leading-whitespace/indented line; `*` and `+`
  bullets; canonical line (regression — still parses).
- Negative/precision: a non-idea prose line is NOT parsed as an idea; a line with the
  second `[issue_ids]` bracket (current "Shape B") — confirm/decide its handling under
  the relaxed regex.
- Round-trip: dateless line → `done`/`edit` → output has backfilled today's date and
  canonical form; variant bullet/indent/CRLF → canonicalized on save.
- Preservation regression: headers, prose, and blank lines survive a mutating save
  byte-for-byte.

## Affected Memory

- `cli/structure.md`: (modify) Update the parser/format description to reflect
  lenient-read/canonical-write, optional-date-on-input, accepted input variants, and
  date-backfill-on-save behavior.

## Impact

- **Code**: `src/internal/idea/idea.go` — `lineRegex` (regex relaxation), `ParseLine`
  (capture-group indices may shift with the optional date group), `LoadFile` (CRLF
  `\r` stripping), date-backfill seam (`SaveFile` or a pre-save normalize step).
  `FormatLine` likely unchanged. `MarshalJSON` unchanged (date populated by save time).
- **Tests**: `src/internal/idea/idea_test.go` — new table-driven cases.
- **Specs**: `docs/specs/backlog-format.md`, `docs/specs/overview.md` — rewritten.
- **Public contract**: Output format unchanged (still canonical). Input contract
  *widened* (strictly more permissive) — no existing valid Shape A line stops parsing.
  This is a backward-compatible read widening + a documented format-doc update.
- **No new dependencies** (Constitution Dependency Discipline): stdlib + cobra only.
- **Constitution**: Principle I (round-trip of non-idea content) preserved; Principle
  VI (stable IDs/output, JSON shape) preserved — JSON `date` stays populated because
  save backfills before marshal. Principle V (table-driven tests, real temp dirs)
  satisfied by section 7.

## Open Questions

All open questions resolved during clarification (2026-06-10) — see `## Clarifications`.

- ~~Should backfill-on-save emit a notice?~~ **Resolved**: yes — emit a brief
  **stderr** notice (e.g. `note: stamped today's date on N previously-dateless
  item(s)`) when a mutating save backfills one or more dates. Notice goes to stderr
  only so stdout stays machine-parseable (Constitution VI); suppress entirely when the
  count is 0.
  <!-- clarified: backfill emits stderr notice (not stdout, not silent) -->
- ~~How should the relaxed regex treat legacy "Shape B" second-bracket lines?~~
  **Resolved**: keep them as **inert pass-through**, exactly as today. The relaxed
  regex MUST NOT match `- [ ] [id] [DEV-1011] date: text`; such lines are preserved
  verbatim and the `[DEV-1011]` slot stays owned by external consumers (fab-kit
  `/fab-new`). A regression test MUST pin this — confirm Shape B lines are neither
  parsed nor rewritten on canonicalize.
  <!-- clarified: Shape B second-bracket lines remain pass-through; add pinning test -->

> **Output-channel note for the plan**: the backfill notice is the first idea command
> output deliberately routed to **stderr** rather than stdout. The plan should confirm
> existing command output (e.g. `done`/`edit` confirmations) and any JSON output remain
> on stdout, and only the advisory backfill notice goes to stderr.

## Clarifications

### Session 2026-06-10

| # | Question | Answer |
|---|----------|--------|
| 9 | Should backfill-on-save announce stamped dates, or stay silent? | Brief **stderr** notice (`note: stamped today's date on N previously-dateless item(s)`); stdout stays machine-parseable; suppressed when count is 0. |
| 10 | How should the relaxed regex treat legacy "Shape B" `[id] [DEV-1011] date: text` lines? | Keep as **inert pass-through** (status quo) — not parsed, preserved verbatim, `[DEV-1011]` owned by external consumers; pin with a regression test. |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Date is optional on input; both dated and dateless lines parse | Explicit user decision in `/fab-discuss` ("Make date optional") | S:98 R:70 A:90 D:95 |
| 2 | Certain | Output is canonical: `-` bullet, LF, date always present, no indentation | Explicit user decision ("Normalize on write"); falls out of existing FormatLine/SaveFile | S:98 R:75 A:92 D:95 |
| 3 | Certain | Dateless ideas get today's date backfilled on save (not kept dateless) | Explicit user decision ("Backfill today's date") | S:98 R:65 A:88 D:95 |
| 4 | Certain | Accept variant input: `-`/`*`/`+` bullets, leading whitespace, CRLF/LF | Explicit user decision ("Common variants too") | S:95 R:75 A:85 D:90 |
| 5 | Certain | Normalize-on-write: first mutating save canonicalizes all recognized idea lines | Direct consequence of regenerate-on-save architecture user chose to keep | S:95 R:60 A:85 D:90 |
| 6 | Certain | Non-idea lines remain verbatim pass-through; `add` unchanged | Explicit scope boundary set in discussion; Constitution I | S:98 R:85 A:95 D:98 |
| 7 | Confident | Spec rewrite (backlog-format.md + overview.md) is part of this change | User agreed docs move with the code; required by Constitution VI framing | S:85 R:80 A:80 D:85 |
| 8 | Confident | Backfill stamped in SaveFile / pre-save normalize step (not in ParseLine) | Keeps ParseLine pure; matches regenerate-on-save seam; MarshalJSON stays correct | S:70 R:70 A:80 D:75 |
| 9 | Certain | Backfill-on-save emits a brief stderr notice (`stamped today's date on N item(s)`), suppressed at count 0; stdout stays clean | Clarified — user confirmed (stderr notice) | S:95 R:85 A:60 D:55 |
| 10 | Certain | Legacy "Shape B" second-bracket lines stay inert pass-through under the relaxed regex; pinned by a regression test | Clarified — user confirmed (keep as pass-through) | S:95 R:75 A:70 D:60 |

10 assumptions (8 certain, 2 confident, 0 tentative, 0 unresolved).
