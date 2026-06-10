# Intake: Escape Multiline Text in Backlog Lines

**Change**: 260610-49mw-escape-multiline-idea-text
**Created**: 2026-06-10
**Status**: Draft

## Origin

Synthesized from a completed `/fab-discuss` session (2026-06-10, conversational mode). The user verified the bug empirically (built the binary, tested in /tmp), reviewed three candidate designs, and explicitly chose **Option 1: escape on write, unescape on display**. All major decisions below were made during that session.

> `idea add` and `idea edit` accept text containing embedded newlines and write it raw into `fab/backlog.md`. For input `"first line\n\nsecond paragraph\n- [ ] looks like a task"`: (1) silent truncation — the persisted idea line carries only `first line`; (2) orphaning — the continuation lines land in the file as non-idea pass-through prose that `rm`/`edit` never touch; (3) phantom ideas — pasted content containing a line matching the lenient parser regex would parse as a separate, bogus idea; (4) the `Added:` confirmation echoes the raw newlines. Fix by escaping on write (LF → `\n`, `\` → `\\`), unescaping on display, keeping exactly one physical line per idea.

## Why

**The pain point.** The backlog file format is one-physical-line-per-idea (`- [ ] [id] YYYY-MM-DD: text`), but nothing in the write path enforces it. `Add` builds the line via `FormatLine` and appends with `fmt.Fprintln`; `Edit` rewrites the idea's single line via `SaveFile`. Text containing a raw LF therefore splits across physical lines:

1. **Silent truncation** — only the first physical line carries the checkbox/ID anchors, so `list`, `show`, `edit`, and JSON output see only `first line`. The rest of the user's text is invisible to every command.
2. **Orphaning** — the continuation lines are stored as non-idea pass-through prose (per Constitution I they are preserved verbatim forever). `idea rm <id>` deletes only the idea line and permanently orphans the residue; `edit` likewise rewrites only the first line.
3. **Phantom ideas** — pasted content containing a line that itself matches the lenient parser regex (checkbox + 4-char `[a-z0-9]{4}` id) parses as a separate, bogus idea with its own lifecycle.
4. **Messy confirmation** — `cmd/idea/add.go` prints `Added: [%s] %s: %s` with the raw text, producing multi-line stdout where one machine-parseable line is expected (Constitution VI).

**If we don't fix it**: any multiline paste (the most natural way to capture a rich idea) silently corrupts the backlog — data loss that the user only discovers later, plus unremovable residue in the source-of-truth file.

**Why this approach.** Escape-on-write keeps the file structurally identical — exactly one physical line per idea, canonical format unchanged — so every external line-by-line consumer (fab-kit, shll.ai) keeps working with zero contract churn, while the full text survives round-trip losslessly. Alternatives explicitly rejected during discussion:

- **Continuation-line markers** (e.g. `\`-prefixed follow-on lines): requires stateful multi-line parsing in `LoadFile`, breaks Constitution I ("non-idea lines pass through verbatim" becomes ambiguous), changes the output contract every external consumer must learn, and renders poorly as Markdown.
- **Flattening newlines to spaces at input**: lossy; the user explicitly wants multiline content to survive round-trip (`show` returns the paragraphs).

## What Changes

### Escaping scheme (the core contract)

Two escape sequences, applied to idea text only (never to non-idea lines):

| In-memory character | Persisted sequence (2 chars) |
|---------------------|------------------------------|
| `\` backslash (U+005C) | `\\` |
| LF (U+000A) | `\n` |

- **Escape** (write direction): backslash is escaped first (or equivalently a single-pass replacement) so the encoding is unambiguous. Escaped text contains no raw LF or CR by construction, so the persisted line is always a single physical line and always matches the existing single-line `lineRegex`.
- **Unescape** (read/display direction): left-to-right scan; `\\` → `\`; `\n` → LF; `\` followed by any **other** character → both characters pass through verbatim (e.g. `\b` stays `\b`); a trailing lone `\` passes through verbatim.
- **CR normalization** happens alongside escaping at the write seam: CRLF (`\r\n`) → LF first, then any remaining lone CR → LF. No raw CR ever reaches the file.
- **Round-trip law**: `Unescape(Escape(x)) == x` exactly, for any CR-free `x` (including backslash-heavy text like `C:\new`, `a\\b`, trailing `\`). For text containing CR, `Unescape(Escape(x)) == normalizeCR(x)` — the CR→LF normalization is the only deliberate loss.

Worked example — `idea add "first line\n\nsecond paragraph\n- [ ] looks like a task"` (real newlines in the argument) persists as **one** physical line:

```
- [ ] [a7k2] 2026-06-10: first line\n\nsecond paragraph\n- [ ] looks like a task
```

(each `\n` above is the literal two-character sequence). The embedded `- [ ] looks like a task` never starts a physical line, so the phantom-idea failure mode is structurally eliminated.

### Write path: `internal/idea` seam, both `Add` and `Edit`

Implemented in `src/internal/idea/idea.go` — not in `cmd/` — per Constitution Principles III & IV, so the bare root shorthand (`idea <text>` → `idea add <text>`) inherits it with no extra wiring:

- `Add` (≈ line 388) and `Edit` (≈ line 614) normalize CR and ensure escaping before the text is serialized.
- The natural seam is the existing parse/format pair: `FormatLine` (≈ line 136) escapes `Idea.Text` on serialization; `ParseLine` (≈ line 108) unescapes the captured text group. That single placement covers `Add`'s direct `FormatLine` append, `SaveFile`'s normalize-on-write rebuild (which `Edit`/`Done`/`Reopen`/`Rm` flow through), and `RequireSingle`'s multi-match error listing. `ParseLine` stays **pure and line-based** (no date stamping, no I/O) — unescaping is a pure string transform. <!-- assumed: transform placed at ParseLine/FormatLine rather than scattered across Add/Edit/display call-sites — single source of truth; in-memory Idea.Text always holds real (unescaped) text -->
- **In-memory representation**: `Idea.Text` always holds the *real* text (raw newlines, raw backslashes). Consequences: `MarshalJSON` needs no change (JSON encodes the newlines itself), and `Match`/query semantics operate on the real text the user typed.

### Read/display path

- **`idea list`** (and `list --done`, `--all`): unchanged shape — one line per idea, printed in the canonical **escaped** form via `FormatLine`. External pipelines keep a line-per-record guarantee.
- **`idea show`** (plain): renders the text with **real newlines** (unescaped display). Output keeps the familiar `- [ ] [id] YYYY-MM-DD: ` prefix followed by the unescaped text, so continuation lines appear below:

  ```
  - [ ] [a7k2] 2026-06-10: first line

  second paragraph
  - [ ] looks like a task
  ```

- **JSON output** (`list --json`, `show --json`): the `text` field carries real newlines — JSON itself encodes them as `\n`. With real text in `Idea.Text` this falls out of the existing `MarshalJSON`.

### Confirmation output

- `Added:` (`cmd/idea/add.go`) and `Updated:` (`cmd/idea/edit.go`) print the **escaped single-line form** — never raw newlines. `Updated:` already prints `FormatLine(i)` and inherits this; `Added:` prints `i.Text` directly today and must print the escaped form (cmd-layer output formatting may call an exported escape helper, e.g. `idea.EscapeText` — output formatting in `cmd/` is constitutional).

### Legacy backslash policy (pre-existing files, normalize-on-write)

Pre-existing backlog lines whose text contains literal backslashes were written before any escape convention existed. Policy:

- **Unrecognized escapes pass through verbatim on read** (e.g. `a\b` reads as `a\b` — backslash preserved).
- **One-time normalize-on-write**: when a mutating command next saves the file, in-memory `\` re-serializes as `\\` (e.g. on-disk `a\b` → `a\\b`). Content is unchanged (still reads back as `a\b`); only the on-disk encoding canonicalizes. This matches the project's established normalize-on-write precedent (first mutating save already canonicalizes bullets/indentation/CRLF/dateless lines — a documented, accepted trade-off).
- **Known consequence, accepted**: a legacy line containing the literal two-character sequence `\n` inside its text (e.g. `Fix path C:\new on Windows`) is reinterpreted on read — `\n` decodes to a real newline (`C:` + LF + `ew`). This is unavoidable under any unescape-on-read scheme (the stored bytes are ambiguous with legacy data) and judged rare; re-saving is stable (the newline re-escapes to `\n`, the file does not churn further). <!-- assumed: lenient pass-through for unrecognized escapes + one-time normalize-on-write + accepted legacy `\n` reinterpretation, per the project's normalize-on-write precedent — strict-error and version-marker alternatives rejected as contract churn -->

### Tests

Per Constitution V and the discussion constraints: table-driven tests against real temp dirs (`t.TempDir()`), in `src/internal/idea/idea_test.go` (+ `cmd/idea` in-process coverage where confirmations change). Must include:

- Round-trip property cases — `Unescape(Escape(x)) == x` for: plain text, multi-paragraph text, backslash-heavy inputs (`C:\new`, `a\\b`, trailing `\`, `\\n` vs `\n`), text that *is* an idea-looking line, CRLF/lone-CR inputs (asserting the normalized expectation).
- `add` with embedded newlines → file gains exactly one line; `list` shows one entry; `show` returns the full multi-paragraph text; `rm` removes everything (no orphans).
- `edit` to multiline text → same single-line guarantee.
- Legacy file fixtures: lone-backslash text reads verbatim, re-saves doubled, stable on second save (no further churn).
- Confirmation output: `Added:`/`Updated:` are single-line.

### Out of scope

- **Repairing already-mangled multiline entries** in existing backlog files: the orphaned prose is unrecognizable as idea content; cannot be auto-repaired safely. (User-decided exclusion.)

## Affected Memory

- `cli/structure.md`: (modify) — the "Backlog line lifecycle (lenient read, canonical write)" section gains the escape/unescape convention: the two escape sequences, CR normalization, in-memory-real/on-disk-escaped split, display semantics per command, and the legacy-backslash normalize-on-write note.

## Impact

- **Code**: `src/internal/idea/idea.go` (`ParseLine`, `FormatLine`, `Add`, `Edit`, new exported escape/unescape helpers), `src/internal/idea/idea_test.go`, `src/cmd/idea/add.go` (escaped confirmation), `src/cmd/idea/show.go` (unescaped display), `src/cmd/idea/edit.go` (inherits via `FormatLine`; verify only), `src/cmd/idea/main_test.go` (end-to-end coverage as needed).
- **Public contract**: `docs/specs/backlog-format.md` documents the line format and needs an escape-convention section (with a format-contract change note, mirroring the `260610-wtmn-resilient-backlog-parser` precedent) — expected during hydrate.
- **External consumers** (fab-kit, shll.ai): unaffected structurally — exactly one physical line per idea is preserved; the canonical output shape `- [ ] [id] YYYY-MM-DD: text` is unchanged. Consumers that display raw text will show escaped sequences for multiline ideas (by design — that *is* the canonical form).
- **Constraints honored**: Constitution I (plain-text source of truth; non-idea lines pass through verbatim — untouched), VI (stable output format — structurally unchanged), lenient-read/canonical-write contract from `260610-wtmn-resilient-backlog-parser` preserved, `ParseLine` stays pure and line-based, stdlib only (`strings.Replacer` or a small manual scanner — no new dependencies).

## Open Questions

- None — all major decisions were made in the discussion session; the remaining minor decisions are SRAD-graded below (see #11–#14, especially the Tentative legacy-backslash policy #13, reviewable via `/fab-clarify`).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Escape-on-write / unescape-on-display (Option 1), not continuation-line markers or newline flattening | Discussed — user explicitly chose Option 1 after reviewing three options; alternatives rejected with reasons | S:95 R:90 A:95 D:95 |
| 2 | Certain | Escaping scheme: LF → `\n` (2 chars), `\` → `\\`; escape/unescape exact inverses, lossless round-trip | Discussed — values specified verbatim in the session | S:95 R:85 A:90 D:95 |
| 3 | Certain | CR normalization at the write seam: CRLF → LF, lone CR → LF; no raw CR reaches the file | Discussed — explicitly specified alongside the escaping scheme | S:90 R:85 A:90 D:90 |
| 4 | Certain | Exactly one physical line per idea; canonical output contract `- [ ] [id] YYYY-MM-DD: text` structurally unchanged | Discussed constraint; Constitution I & VI make this non-negotiable | S:95 R:80 A:95 D:95 |
| 5 | Certain | Implemented in `internal/idea` for both `Add` and `Edit`; `cmd/` stays wiring/output-only; bare root shorthand inherits | Discussed; Constitution Principles III & IV mandate the seam | S:90 R:85 A:95 D:90 |
| 6 | Certain | Display semantics: `show` renders real newlines; `list` stays escaped one-line-per-idea; JSON `text` carries real newlines | Discussed — each output explicitly specified by the user | S:90 R:85 A:90 D:90 |
| 7 | Certain | `Added:`/`Updated:` confirmations print the escaped single-line form, never raw newlines | Discussed — explicit decision; Constitution VI (machine-parseable stdout) | S:90 R:90 A:90 D:90 |
| 8 | Certain | Stdlib only (`strings.Replacer` or small manual scanner) — no new dependencies | Constitution Dependency Discipline answers deterministically | S:95 R:95 A:100 D:95 |
| 9 | Certain | Table-driven tests against real temp dirs, incl. round-trip property cases with backslash-heavy inputs | Constitution V + explicit discussion constraint | S:90 R:90 A:95 D:90 |
| 10 | Certain | Out of scope: repairing already-mangled multiline entries in existing backlogs | Discussed — user excluded it; orphaned prose is unrecognizable as idea content | S:90 R:75 A:85 D:90 |
| 11 | Confident | Transform placed at the `ParseLine`/`FormatLine` seam; `Idea.Text` holds real (unescaped) text in memory, so `MarshalJSON` and `Match` need no change | Strongest single-source-of-truth placement; every writer/reader inherits; preserves ParseLine purity (pure string transform) | S:70 R:75 A:80 D:70 |
| 12 | Confident | `show` plain output = canonical `- [x] [id] date: ` prefix + unescaped text (continuation lines render below) | Minimal-delta rendering consistent with current `show`; easily reversed display detail | S:60 R:85 A:75 D:65 |
| 13 | Tentative | Legacy backslash policy: unrecognized escapes (e.g. `\b`) pass through verbatim on read; lone `\` re-serializes as `\\` on first mutating save (one-time normalize-on-write); legacy literal `\n` sequences (e.g. `C:\new`) reinterpret as newlines — accepted | Edge case surfaced in discussion but policy left open; project's normalize-on-write precedent strongly favors lenient pass-through over strict errors, but the legacy `\n` reinterpretation is a user-visible behavior change to existing data not explicitly approved | S:55 R:50 A:70 D:60 |
| 14 | Confident | Update `docs/specs/backlog-format.md` with the escape convention (format-contract change note) during this change's hydrate | Description flags it ("likely during hydrate"); direct precedent: wtmn updated the same spec with a contract-change note | S:65 R:90 A:80 D:75 |

14 assumptions (10 certain, 3 confident, 1 tentative, 0 unresolved).
