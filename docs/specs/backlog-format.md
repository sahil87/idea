# Backlog File Format

The line format used in `fab/backlog.md` is the contract between `idea` and any external consumer. `idea` follows the robustness principle: **lenient on read, canonical on write.** It accepts a range of surface-form variants on input, but owns exactly one canonical output form.

## Canonical Output Form

Every idea line that `idea` writes (or rewrites) uses this single canonical shape:

```
- [ ] [{ID}] {YYYY-MM-DD}: {description}
```

- `- ` bullet (hyphen + single space), no leading whitespace.
- `[ ]` for open, `[x]` for completed.
- `[{ID}]` — the 4-character lowercase alphanumeric backlog ID.
- `{YYYY-MM-DD}` — ISO date, **always present** in output (backfilled if the input had none; see [Date backfill](#date-backfill-on-write)).
- `{description}` — free-form description text, in **escaped form** (see [Escaped Text in the Description](#escaped-text-in-the-description)).
- LF line endings; the file ends with a single trailing LF.

This canonical form is the stable, machine-parseable output contract. External tooling that reads `fab/backlog.md` can rely on it.

## Escaped Text in the Description

The `{description}` field of a canonical line is **escaped**, so an idea always occupies exactly one physical line — even when its text contains newlines. Exactly two escape sequences exist in canonical output:

| Real character | Persisted sequence (2 chars) |
|----------------|------------------------------|
| `\` backslash (U+005C) | `\\` |
| newline (LF, U+000A) | `\n` |

Carriage returns never appear in output: CRLF and lone CR in input text are normalized to LF *before* escaping, so no raw CR — and, after escaping, no raw LF — is ever written into an idea line.

**Recovering the real text (consumers).** To decode a description, scan left-to-right: `\\` → `\` and `\n` → LF. A backslash followed by any **other** character (e.g. `\b`) passes through verbatim, as does a trailing lone `\` — unrecognized escapes are never an error; decoding is lenient by design.

Worked example — adding an idea whose real text is `first line`, a blank line, then `second paragraph` persists as **one** physical line (each `\n` below is the literal two-character sequence):

```
- [ ] [a7k2] 2026-06-10: first line\n\nsecond paragraph
```

The **one-physical-line-per-idea guarantee is unchanged** by the escape convention — line-by-line consumers keep working with zero structural churn. `idea list` prints the escaped canonical form (preserving the line-per-record guarantee), plain `idea show` renders the real newlines, and JSON output (`--json`) carries real newlines in the `text` field (JSON applies its own encoding).

## Accepted Input Variants (lenient read)

`idea` parses a recognized idea line whether or not it is canonical. The following variants are all accepted on **input** and parse to the same idea:

| Dimension | Accepted on input | Example |
|-----------|-------------------|---------|
| Date | present **or** absent | `[id] 2026-06-10: text` / `[id] text` |
| Bullet marker | `-`, `*`, or `+` | `* [ ] [id] text` |
| Leading whitespace | any indentation (spaces or tabs) | `␣␣- [ ] [id] text` |
| Line endings | CRLF or LF | `...text\r\n` |

The anchors that make a line *recognizable as an idea* are the `[ ]`/`[x]` checkbox plus the 4-character `[a-z0-9]{4}` ID. These keep false-positive matching of genuine prose low — a line missing either anchor is treated as non-idea pass-through.

The relaxed parser regex is:

```
^\s*[-*+] \[([ x])\] \[([a-z0-9]{4})\] (?:(\d{4}-\d{2}-\d{2}): )?(.+)$
```

A dateless line and its dated counterpart parse to the same `Idea` modulo the `Date` field (which is empty when the date segment is absent):

```
- [ ] [rk7t] 2026-06-10: Tune the README-extraction reporter   → Date="2026-06-10"
- [ ] [rk7t] Tune the README-extraction reporter               → Date="" (dateless)
```

> **Boundary note**: because the date segment is optional, a line whose description merely *begins* with a malformed date (e.g. `- [ ] [a7k2] 2025-6-15: Text`) is not rejected — it parses as a dateless idea whose text is `2025-6-15: Text`. Only a well-formed `YYYY-MM-DD:` prefix is lifted into the `Date` field.

## Date Backfill on Write

`idea` does not preserve datelessness. When a recognized idea has no date and the file is saved by a mutating command (`done`, `reopen`, `edit`, `rm`) or by `idea fmt`, `idea` backfills **today's date** before writing, so every persisted idea line carries a date.

Worked example (`idea done rk7t` on a dateless line, run on 2026-06-10):

```
in:  - [ ] [rk7t] Tune the README-extraction reporter
out: - [x] [rk7t] 2026-06-10: Tune the README-extraction reporter
```

When one or more dates are backfilled by a mutating command, `idea` prints a brief advisory notice to **stderr** (never stdout, which stays machine-parseable):

```
note: stamped today's date on N previously-dateless item(s)
```

The notice is suppressed entirely when no dates were backfilled.

## Normalize-on-Write

`idea` rebuilds **every** recognized idea line from the canonical form on save — not just the line you edited. So the first mutating command (`done`/`edit`/`reopen`/`rm`) canonicalizes the whole file at once:

- variant bullets (`*`, `+`) → `-`
- leading indentation → stripped
- CRLF → LF
- dateless → dated (today)
- legacy lone backslashes in description text → doubled (`a\b` → `a\\b` on disk; the decoded content is unchanged)

This is a deliberate, accepted trade-off. A single `idea done` on one item can therefore produce a larger git diff if the file had many variant or dateless lines. **Non-mutating** commands (`list`, `show`) never rewrite the file, so pure reads are diff-free. To land the canonicalization churn as its own diff — without a semantic change riding along — use `idea fmt` (next section).

## Explicit Canonicalization & Adoption (`idea fmt`)

`idea fmt` is the explicit, gofmt-style canonicalizer: it rewrites the whole backlog into the canonical form using the same rule set as [normalize-on-write](#normalize-on-write), with no semantic change required to trigger it. Run it once on a legacy file to commit the formatting churn separately; subsequent mutating commands then produce minimal semantic diffs.

- `fmt` is the **only explicit whole-file write verb**. Mutating CRUD commands keep their incidental normalize-on-write; `list`/`show` remain non-mutating.
- `fmt` is **idempotent**: a second run is byte-stable. When the file is already canonical, `fmt` skips the write entirely (not even the mtime changes).

**Automatic adoption of bare checkboxes.** `fmt` additionally brings plain markdown task-list lines under management. A line is an adoption candidate iff it does *not* parse as an idea AND matches the bare-checkbox shape:

```
^\s*[-*+] \[([ xX])\] (.+)$
```

with one precision guard: the captured text — evaluated **after trimming surrounding whitespace**, so extra spaces between the checkbox and a bracket cannot defeat it (`- [ ]  [DEV-1011] x` stays pass-through) — must NOT begin with a `[...]` bracket. Each adopted line receives a fresh unique 4-char ID and today's date; its checked state is preserved (`[x]`/`[X]` adopt as done, `[ ]` as open); its whitespace-trimmed text is treated as real text and escaped on write like any other idea. Worked example (run on 2026-06-12):

```
in:   * [ ] buy milk
      - [X] ship the release
      - [ ] [DEV-1011] external item
out:  - [ ] [k3v9] 2026-06-12: buy milk
      - [x] [p2m4] 2026-06-12: ship the release
      - [ ] [DEV-1011] external item          ← untouched (bracket guard)
```

The bracket guard keeps inert, byte-for-byte: Shape B lines (the `[issue_ids]` slot — see the next section) and bracket-metadata lines such as `- [ ] [TODO] buy milk` or `- [ ] [ab1] text` — external-looking metadata errs toward preservation. Text-less and whitespace-only checkboxes (`- [ ]` with no text) are not adopted. Headers, prose, and blank lines pass through verbatim as always.

**Output contract.** stdout stays empty — success is silence plus exit 0 (the `gofmt -w` precedent). All reporting goes to **stderr**: one `adopted: [id] {escaped text}` line per adopted idea, the dateless-backfill advisory, and a summary count line. A run that changes nothing prints nothing.

**`--check` mode.** `idea fmt --check` writes nothing, prints the same report (what *would* be normalized / adopted / backfilled), and exits **1** when the file is non-canonical, **0** when it is already canonical — one flag serving both the dry-run preview and the scripts/CI gate.

## Shape B Pass-Through (legacy second-bracket lines)

A line with a second bracket immediately following the ID — e.g. an `[{issue_ids}]` slot — is **Shape B** and is treated as **inert pass-through content**:

```
- [ ] [ni3o] [DEV-1011] 2026-02-12: Capture more metrics
```

- Shape B lines are **not** parsed by `idea`, even under the relaxed regex. They do not appear in `idea list`, cannot be addressed by `idea show <id>`, and are unaffected by `done`/`reopen`/`edit`/`rm`.
- Shape B lines are preserved **byte-for-byte** through any `idea` operation that round-trips the file (per Constitution principle I). Normalize-on-write does not touch them.
- The `[{issue_ids}]` slot — and Shape B semantics generally — are owned by **external consumers**. For example, fab-kit's `/fab-new` writes Linear-style issue IDs (e.g. `[DEV-1011]`) into the slot and parses them back independently of `idea`.

This lets `idea` and external tooling share the same backlog file without either tool needing to understand the other's metadata.

## Round-Trip Preservation

Per Constitution principle I (Plain-Text Backlog as Source of Truth), parsing and re-serializing `fab/backlog.md` MUST preserve all non-idea content verbatim. This applies to:

1. **Between lines** — section headers, blank lines, and prose between items pass through unchanged.
2. **Whole lines that look like idea entries but aren't** — any line that does not parse as an idea (including Shape B lines, and lines missing the checkbox/ID anchors) is preserved verbatim.

> **Carve-out (`idea fmt` adoption).** Bare checkbox lines lacking the 4-char `[id]` anchor (e.g. `- [ ] buy milk`) were previously guaranteed non-idea pass-through under rule 2. As of the fmt change, `idea fmt` — and **only** `fmt` — claims them via [automatic adoption](#explicit-canonicalization--adoption-idea-fmt). Every other command still preserves them verbatim, and bracket-led lines (including Shape B) remain inert pass-through everywhere.

The file is meant to be hand-edited; `idea` exists to reduce friction over hand-editing, not to take ownership of the file or its embedded metadata.

## Stability Commitment & Contract Change

- **Output** is a stable, machine-parseable contract: the canonical form above. The `[{ID}]` format (4-char lowercase alphanumeric, unique within the file) and the `{YYYY-MM-DD}` ISO date are part of the public API.
- **Input** is liberal: `idea` accepts the variants listed above. This is a strict *widening* of what previously parsed — no line that was valid before stops parsing.

> **Format-contract change note.** Earlier versions of this spec declared the date a mandatory part of the line and described an idea-managed "Shape A" that *required* the date. As of the resilient-parser change (`260610-wtmn-resilient-backlog-parser`), the date is **optional on input** and **canonical (always present) on output**, and `idea` additionally accepts `*`/`+` bullets, leading whitespace, and CRLF endings on input. This widens the input contract and was a deliberate, documented change to fix silent-failure on dateless backlogs (e.g. the shll.ai backlog). The **output** contract is unchanged: `idea` still emits exactly one canonical, machine-parseable form. Shape B second-bracket lines remain inert pass-through, exactly as before.

> **Format-contract change note.** As of the fmt change (`260612-4m3a-add-fmt-canonicalizer-adoption`), adoption **widens which lines `idea` may rewrite**: bare checkbox lines without the 4-char `[id]` anchor were previously guaranteed non-idea pass-through; `idea fmt` (and only `fmt`) now adopts them as managed canonical idea lines (fresh ID, today's date, checked state preserved). No other command's rewrite scope changed, the canonical output form is unchanged, and Shape B second-bracket lines remain inert byte-for-byte pass-through, exactly as before.

> **Format-contract change note.** As of the multiline-escape change (`260610-49mw-escape-multiline-idea-text`), the `{description}` field is **escaped** in canonical output: backslash persists as `\\` and newline as `\n`, with CRLF/lone CR normalized to LF before escaping (see [Escaped Text in the Description](#escaped-text-in-the-description)). Structurally nothing changed — still exactly one physical line per idea in the same canonical shape — so line-by-line consumers are unaffected; consumers that display raw description text will show escape sequences for multiline ideas (that *is* the canonical form). Descriptions written before this convention decode leniently (unrecognized escapes pass through verbatim) and canonicalize on the next mutating save: lone backslashes double (`a\b` → `a\\b` on disk, decoded content unchanged), and a second save is byte-stable. One rare, accepted consequence: a legacy description containing the literal two-character sequence `\n` (e.g. `C:\new`) is reinterpreted as a real newline on read.

See also [Constitution principle I](../../fab/project/constitution.md) on plain-text backlog preservation.
