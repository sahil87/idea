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
- `{description}` — free-form description text.
- LF line endings; the file ends with a single trailing LF.

This canonical form is the stable, machine-parseable output contract. External tooling that reads `fab/backlog.md` can rely on it.

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

`idea` does not preserve datelessness. When a recognized idea has no date and the file is saved by a mutating command (`done`, `reopen`, `edit`, `rm`), `idea` backfills **today's date** before writing, so every persisted idea line carries a date.

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

This is a deliberate, accepted trade-off. A single `idea done` on one item can therefore produce a larger git diff if the file had many variant or dateless lines. **Non-mutating** commands (`list`, `show`) never rewrite the file, so pure reads are diff-free.

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

The file is meant to be hand-edited; `idea` exists to reduce friction over hand-editing, not to take ownership of the file or its embedded metadata.

## Stability Commitment & Contract Change

- **Output** is a stable, machine-parseable contract: the canonical form above. The `[{ID}]` format (4-char lowercase alphanumeric, unique within the file) and the `{YYYY-MM-DD}` ISO date are part of the public API.
- **Input** is liberal: `idea` accepts the variants listed above. This is a strict *widening* of what previously parsed — no line that was valid before stops parsing.

> **Format-contract change note.** Earlier versions of this spec declared the date a mandatory part of the line and described an idea-managed "Shape A" that *required* the date. As of the resilient-parser change (`260610-wtmn-resilient-backlog-parser`), the date is **optional on input** and **canonical (always present) on output**, and `idea` additionally accepts `*`/`+` bullets, leading whitespace, and CRLF endings on input. This widens the input contract and was a deliberate, documented change to fix silent-failure on dateless backlogs (e.g. the shll.ai backlog). The **output** contract is unchanged: `idea` still emits exactly one canonical, machine-parseable form. Shape B second-bracket lines remain inert pass-through, exactly as before.

See also [Constitution principle I](../../fab/project/constitution.md) on plain-text backlog preservation.
