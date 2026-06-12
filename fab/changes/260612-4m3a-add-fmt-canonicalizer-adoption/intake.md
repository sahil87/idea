# Intake: Add `idea fmt` — Explicit Canonicalizer with Automatic Checkbox Adoption

**Change**: 260612-4m3a-add-fmt-canonicalizer-adoption
**Created**: 2026-06-12

## Origin

Initiated via `/fab-new 4m3a` (one-shot, backlog ID input). Backlog item `[4m3a]` (2026-06-12), decoded text:

> Add `idea fmt` — explicit gofmt-style canonicalizer with automatic adoption of bare checkboxes.
>
> PROBLEM: Canonicalization currently happens only as a side-effect of the first mutating command (normalize-on-write rewrites the whole file), so a single `idea done` on a legacy file mixes formatting churn into a semantic diff. Separately, plain `- [ ]` checkbox lines without a 4-char `[id]` anchor are invisible to idea (non-idea pass-through) — there is no path to adopt an existing markdown task list into idea management.
>
> BEHAVIOR:
> - `idea fmt` rewrites the backlog to canonical form: `-` bullet, no indentation, LF endings, date backfill (today, with the existing stderr advisory notice), escape normalization (legacy lone backslashes doubled). Reuses the existing single canonical writer in internal/idea — fmt should be a thin verb over the existing parse/format/save machinery, extended with adoption. No second serialization path.
> - ADOPTION IS AUTOMATIC (decided): lines matching a bare checkbox (`- [ ] text` / `- [x] text`, also `*`/`+` bullets and leading indentation) that LACK an `[id]` anchor get a fresh unique 4-char ID assigned and become managed canonical idea lines (date backfilled to today).
> - Idempotent: second run is byte-stable.
> - Non-idea, non-checkbox content (headers, prose, blank lines) preserved verbatim per Constitution principle I. Shape B second-bracket lines (e.g. `[ni3o] [DEV-1011]`) remain inert byte-for-byte pass-through — fmt must not touch them.
> - Respects --file / --main / IDEAS_FILE resolution like every other command. list/show stay non-mutating; fmt is the only explicit whole-file write verb.
>
> OPEN QUESTIONS (solutioning): false-positive risk on auto-adopt (per-line stderr report at minimum; consider --dry-run); adopt completed `[x]` too? (leaning both); `--check` mode in or out; output contract (counts, stdout/stderr split); update backlog-format.md + overview.md.

The item's "decided" markers are encoded as Certain assumptions; its open questions are resolved below as graded assumptions (one Tentative — see Assumptions #5).

## Why

1. **Diff hygiene (the pain point)**: `SaveFile` regenerates every recognized idea line on any mutating command, so the *first* `idea done` on a legacy file (variant bullets, indentation, dateless lines, legacy backslashes) produces a large mixed diff — formatting churn entangled with the one semantic change. Reviewers cannot tell what actually changed. An explicit `idea fmt` lets the user land the formatting churn as its own commit, after which mutating commands produce minimal semantic diffs.
2. **Adoption gap (the missing capability)**: a pre-existing markdown task list (`- [ ] buy milk`) is invisible to `idea` — the 4-char `[id]` anchor is required for a line to parse. There is today *no command* that brings such lines under management; the user must hand-assign IDs. `fmt` with automatic adoption converts an existing task list into a managed backlog in one command.
3. **Why this approach**: gofmt's model (one explicit canonicalizer verb, one canonical form, idempotent) is the proven pattern, and the canonical writer already exists (`FormatLine`/`SaveFile`). Making `fmt` a thin verb over that machinery — rather than a second serialization path — is both explicitly decided in the backlog item and mandated by Constitution IV (logic in `internal/idea`) and the single-formatter arrangement (`formatLineWith` is the sole home of the format string). Alternatives rejected: an `--adopt` opt-in flag (rejected in the backlog item — adoption is automatic), and canonicalize-on-`list` (violates the non-mutating-reads contract).

## What Changes

### 1. New subcommand: `idea fmt` (`src/cmd/idea/fmt.go`)

```
idea fmt [--check]      # plus inherited persistent flags --file, --main (and IDEAS_FILE)
```

- New `fmtCmd() *cobra.Command` factory registered in `newRootCmd()` (`src/cmd/idea/main.go`), per Constitution III. The cobra file contains only flag wiring and output formatting; all logic lives in `internal/idea` (Constitution IV).
- Enriched `Long` help per the repo convention (terse `Short`, depth in `Long`, inline example) — the help-dump node appears automatically; no schema change.
- **Namespace note**: cobra resolves subcommand names before the root bare-text fallback, so `idea fmt some text` stops routing to the add-shorthand once this command exists. Same namespace trade as the `ls` alias decision; "fmt" plausibly never begins bare idea prose — acceptable.

### 2. Canonicalization — explicit trigger over the existing writer

New exported entry point in `internal/idea` (e.g. `Fmt(path string, check bool) (FmtResult, error)` — exact name/shape is a plan decision) that composes the existing machinery: `LoadFile` → adoption pass (below) → `SaveFile`. No second serialization path; every rewrite rule is the existing normalize-on-write set:

- variant bullets (`*`, `+`) → `-`; leading indentation stripped; CRLF → LF
- dateless → today's date, with the **existing** stderr advisory notice mechanism (backfill count flows up; `printBackfillNotice` wording reused)
- legacy lone backslashes in text doubled (`a\b` → `a\\b` on disk; decoded content unchanged)

**Idempotency**: a second run is byte-stable. When the rebuilt content is byte-identical to the file, `fmt` skips the write entirely (no mtime churn, no atomic-rename of identical bytes).

**Mechanism note for plan**: counting "normalized" lines requires comparing each regenerated line against its original raw text, which `LoadFile` currently discards for idea lines (placeholder `""`). The plan must either retain raw lines in `File` or compare whole-file before/after bytes; either is internal-only and must not change `LoadFile`'s public behavior.

### 3. Automatic adoption of bare checkboxes

A line is an **adoption candidate** iff it does *not* parse as an idea (`ParseLine` returns false) AND matches the bare-checkbox shape:

```
^\s*[-*+] \[([ xX])\] (.+)$        # adoption-candidate regex (text group must be non-empty)
```

with one **precision guard**: the text group must NOT begin with a `[...]` bracket. This keeps inert, byte-for-byte:

- Shape B lines — `- [ ] [ni3o] [DEV-1011] 2026-02-12: text` (the `[issue_ids]` slot is owned by external consumers; `ParseLine` already rejects these, and the adoption guard must too)
- bracket-metadata lines — `- [ ] [TODO] buy milk`, `- [ ] [ab1] text` (non-4-char bracket): external-looking metadata, err toward preservation

Each adopted line gets:

- a **fresh unique 4-char ID** — unique against both the IDs already in the file and IDs assigned earlier in the same fmt pass (the existing `checkIDCollision` reads from disk only; in-memory uniqueness within one run is a plan-level extension)
- **date = today** (counted as *adopted*, not double-counted as *date-backfilled* — backfill counts only pre-existing managed dateless lines)
- **checked state preserved**: `[x]` and `[X]` adopt as done (`[x]` canonical), `[ ]` as open
- text treated as **real text** (CR-normalized, escaped on write via `FormatLine` like every other write — a literal `\` doubles on disk, consistent with the legacy backslash policy)

Worked example (run on 2026-06-12):

```
in:   * [ ] buy milk
      - [X] ship the release
      - [ ] [DEV-1011] external item
out:  - [ ] [k3v9] 2026-06-12: buy milk
      - [x] [p2m4] 2026-06-12: ship the release
      - [ ] [DEV-1011] external item          ← untouched (bracket guard)
```

Headers, prose, blank lines, and empty checkboxes (`- [ ]` with no text — regex requires non-empty text) pass through verbatim per Constitution I.

### 4. `--check` mode — unified preview + CI gate

`idea fmt --check`: writes nothing, prints the same report (what *would* be normalized / adopted / backfilled), and exits non-zero when the file is non-canonical (any normalization, adoption, or backfill would occur), zero when already canonical. One flag serves both needs raised in the backlog item: the dry-run preview before destructive-ish adoption *and* the scripts/CI gate.
<!-- clarified: single --check flag confirmed by user (2026-06-12 clarify session) — no separate --dry-run; writes nothing, prints the would-be report to stderr, exits 1 when non-canonical, 0 when clean -->

### 5. Output contract

- **stdout**: empty. Success is silence + exit 0 (`gofmt -w` precedent); stdout stays machine-parseable per Constitution VI.
- **stderr**: all human-facing reporting —
  - per-line adoption report, one line per adopted idea (e.g. `adopted: [k3v9] buy milk`) — mitigates the false-positive risk by making every claimed line visible
  - summary counts: lines normalized / adopted / dates backfilled (exact wording is a plan decision; zero-activity runs print nothing)
  - the existing dateless-backfill advisory (`note: stamped today's date on N previously-dateless item(s)`) is retained/subsumed by the counts — plan decides composition, but the stderr channel and advisory tone are fixed
- `internal/idea` writes nothing to stderr — counts flow up in the result value and `cmd/idea` prints, per the established Constitution IV split.

### 6. File resolution & command surface

`fmt` inherits `--file` / `--main` persistent flags and `IDEAS_FILE` resolution like every command. `list`/`show` remain non-mutating; **fmt becomes the only explicit whole-file write verb** (mutating CRUD commands keep their incidental normalize-on-write).

### 7. Spec updates (contract change)

- `docs/specs/backlog-format.md`: new format-contract change note — adoption **widens which lines `idea` may rewrite**: bare checkbox lines were previously guaranteed non-idea pass-through; after this change, `idea fmt` (and only `fmt`) claims them. Round-Trip Preservation section gets the carve-out; Shape B guarantees unchanged.
- `docs/specs/overview.md`: command table row for `idea fmt`; parse/format section gains a sentence on the explicit canonicalizer.

## Affected Memory

- `cli/structure`: (modify) backlog line lifecycle gains the explicit `fmt` verb (canonicalize + adopt), the adoption-candidate regex and bracket precision guard, the `--check` contract, and the fmt subcommand note (namespace claim, stderr report convention)

## Impact

- `src/internal/idea/` — adoption + fmt logic (new `fmt.go` or extension of `idea.go`; keep files small per config directive), possible internal-only `File`/`LoadFile` extension for change detection; table-driven tests with `t.TempDir()` (Constitution V)
- `src/cmd/idea/fmt.go` (new) + `src/cmd/idea/main.go` (one-line registration); e2e coverage in `cmd/idea` tests for routing, `--check` exit codes, stderr/stdout split
- `docs/specs/backlog-format.md`, `docs/specs/overview.md` — contract documentation
- No new dependencies (stdlib + cobra only). help-dump JSON schema unchanged (new node appears via the existing walk). Release/CI machinery untouched.

## Open Questions

None — the backlog item's open questions are resolved as graded assumptions below (#4, #5, #6, #7). The former Tentative (#5, `--check` unification) was confirmed by the user in the 2026-06-12 clarify session.

## Clarifications

### Session 2026-06-12

| Q | Answer |
|---|--------|
| `--check` mode shape: single unified flag, separate `--dry-run`/`--check`, or no flag at all? | Single `--check` flag — writes nothing, prints the would-be per-line report + counts to stderr, exits 1 when the file is non-canonical, 0 when clean. No separate `--dry-run`. |

### Session 2026-06-12 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 4 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 8 | Confirmed | — |
| 9 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `fmt` is a thin verb over the existing `LoadFile`/`SaveFile`/`FormatLine` machinery; no second serialization path | Explicitly decided in backlog item; mandated by Constitution IV and the single-formatter arrangement | S:95 R:80 A:95 D:95 |
| 2 | Certain | Adoption is automatic — no opt-in flag | Explicitly marked "(decided)" in the backlog item | S:100 R:70 A:90 D:95 |
| 3 | Certain | Date backfill stamps today via the existing `SaveFile` seam with the existing stderr advisory mechanism | Explicit in item; mechanism already exists and is documented | S:95 R:85 A:95 D:95 |
| 4 | Certain | Completed bare checkboxes (`[x]`/`[X]`) are adopted too, preserving checked state | Clarified — user confirmed | S:95 R:80 A:75 D:80 |
| 5 | Certain | Single `--check` flag unifies dry-run preview and CI gate (write nothing, print report, exit 1 if non-canonical); no separate `--dry-run` | Clarified — user confirmed | S:95 R:85 A:60 D:40 |
| 6 | Certain | Output channels: per-line adoption report + summary counts to stderr; stdout empty (silence + exit code = success) | Clarified — user confirmed | S:95 R:80 A:85 D:70 |
| 7 | Certain | Adoption precision guard: candidates whose text begins with any `[...]` bracket are skipped (stay verbatim pass-through) | Clarified — user confirmed | S:95 R:70 A:85 D:75 |
| 8 | Certain | Adoption candidates accept `*`/`+` bullets, leading indentation (explicit in item) and uppercase `[X]` (lenient-read posture); text-less checkboxes are not adopted | Clarified — user confirmed | S:95 R:85 A:80 D:70 |
| 9 | Certain | `fmt` skips the write when the rebuilt content is byte-identical (idempotent run touches nothing, not even mtime) | Clarified — user confirmed | S:95 R:90 A:85 D:80 |

9 assumptions (9 certain, 0 confident, 0 tentative, 0 unresolved).
