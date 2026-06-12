---
description: "`idea edit` two-form contract: inline replacement (two-arg form) vs. $EDITOR round-trip (`edit <query>`) — editor resolution chain, temp-file mechanics, edge/exit semantics, and the fake-editor test seam"
---

# `idea edit` Subcommand

`idea edit` has two forms, switched purely by the presence or absence of the text argument (`Args: cobra.RangeArgs(1, 2)` — no mode flag). The cobra wrapper lives at `src/cmd/idea/edit.go` and stays wiring-only; the editor plumbing lives at `src/internal/idea/editor.go` (Constitution Principle IV seam). The editor form was added by `260612-w4p7-edit-via-editor`; the pattern precedent is `git commit` / `kubectl edit`.

## The two forms

| Form | Behavior |
|------|----------|
| `idea edit <query> "text"` | Inline replacement — the quick one-liner and scripting path. Never launches an editor (tripwire-tested) and is byte-for-byte the pre-change behavior: same validation, same output. |
| `idea edit <query>` | Opens the resolved editor on a temp file containing the idea's **decoded** text (real newlines, real backslashes — `Idea.Text`); on clean exit the buffer is post-processed and persisted via the existing `idea.Edit`. |

**Resolve before launch.** The one-arg form resolves the match via the existing `idea.Show` (LoadFile + `RequireSingle`, `FilterAll`) **before** launching the editor — an ambiguous or unmatched query is refused with the match list and the editor never opens. This reuses `idea.Edit`'s exact match semantics (substring, case-insensitive, ID or text) with zero new resolver code.

## Editor resolution chain

`ResolveEditor() string` returns `$VISUAL` if non-empty, else `$EDITOR` if non-empty, else `vi` (the `defaultEditor` constant — the conventional Unix fallback; git uses the same chain). The returned value is a command *string* that may carry arguments (e.g. `code --wait`); the launch goes through the shell so multi-word values work. Stdlib `os/exec` only — no new dependencies (Dependency Discipline).

## Temp-file round trip (`EditInEditor`)

`EditInEditor(text string) (edited string, unchanged bool, err error)` owns the mechanics:

1. **Temp file**: `os.CreateTemp("", editorTempPattern)` where `editorTempPattern = "idea-edit-*.md"` (the `.md` extension buys editor syntax highlighting). Seeded with the raw decoded text only — no comment header. Removed via `defer os.Remove`.
2. **Launch**: `sh -c '<editor> "$1"' sh <tmpPath>` (via `exec.Command`) with inherited `os.Stdin`/`os.Stdout`/`os.Stderr`. The shell makes multi-word `$EDITOR` values work; passing the path as a positional parameter sidesteps quoting.
3. **No TTY gate**: a TTY-requiring editor (vi) fails fast on its own with a non-zero exit in non-interactive contexts — satisfying "clear error, never hang" — while a script `$EDITOR` stays viable headlessly, which the test strategy depends on.
4. **Exit 0** → read the file back, apply the existing `normalizeCR`, strip **exactly one** trailing LF, return the result as `edited`. Editors conventionally append a final newline, so stripping one means a one-line idea round-trips without gaining a `\n`; deliberate extra trailing blank lines beyond that final LF are kept as content.
5. **Unchanged detection** → `unchanged` is true when *either* the stripped buffer *or* the pre-strip (CR-normalized only) buffer equals the seeded text. The stripped comparison makes an editor-appended final LF a no-op for ordinary text; the pre-strip comparison makes an untouched buffer a no-op even when the text itself ends in an LF (which the strip would otherwise eat). A deliberate edit of `a\n` to `a` still registers as changed — neither form of the buffer matches the original.
6. **Non-zero exit / launch failure** → error `editor (<name>) failed: … — idea unchanged`; the caller persists nothing.

## Edge semantics (one-arg form)

| Situation | Behavior | Exit |
|-----------|----------|------|
| Editor exits non-zero / fails to launch | No change to the backlog; error via the `RunE` error path, message states the idea is unchanged | non-zero |
| Buffer unchanged (`EditInEditor`'s `unchanged` verdict — stripped *or* pre-strip buffer equals current `Idea.Text`) and no `--id`/`--date` | No-op: **no file rewrite at all** (so no normalize-on-write side effect, no date backfill), `note: text unchanged — nothing to do` on **stderr** | 0 |
| Buffer emptied | Refused: cmd-layer pre-check errors with `editor buffer is empty — idea unchanged (use "idea rm" to delete it)`; `idea.Edit`'s existing `text is required` guard is the safety net (git's empty-commit-message abort precedent) | non-zero |
| `--id`/`--date` passed with the one-arg form | Editor still opens; flags apply at save via the existing `idea.Edit` params. The unchanged-text no-op is **suppressed** when either flag is present, so a metadata-only change still lands — and when the text is unchanged, the save passes the original `Idea.Text` verbatim (not the post-processed buffer), so a metadata-only change never mutates text | per outcome |

The unchanged-buffer no-op is proven by tests against a backlog containing non-canonical (dateless, star-bullet) lines: the file stays byte-identical, demonstrating that skipping the rewrite really does skip the whole-file normalize-on-write.

**Trailing-LF edge (closed by post-review fix).** An untouched session is now *always* a no-op, even for text that ends in an LF: unchanged-detection compares the pre-strip buffer as well as the stripped one, so the strip-one rule can no longer defeat the equality check and silently rewrite the idea minus its trailing LF. The remaining nuance is that an **edited** LF-terminated text still loses its trailing LF — the strip-one rule applies to changed buffers — and a metadata-only save (`--id`/`--date` with unchanged text) preserves the original text verbatim, trailing LF included.

## Output channels and persistence

Success output is unchanged: `Updated: <FormatLine(i)>` (escaped single line) on **stdout**; the backfill notice on stderr when applicable. All no-op/abort messaging goes to **stderr** only. Together with the backfill notice (see `structure.md`), this establishes advisory-notes-to-stderr as the channel policy: stdout carries only machine-parseable confirmations (Constitution VI), and channel policy lives in `cmd/`, never `internal/idea` (Constitution IV).

Persistence goes through the existing `idea.Edit(path, query, newText, newID, newDate)` canonical writer wholesale — whole-file normalize-on-write, date backfill with stderr notice, atomic temp+rename save, non-idea lines preserved verbatim. The editor form added no new write path.

## Help text

`Use` is `edit <query> [new-text]`; `Long` documents both forms with examples (including the bare `idea edit a7k2` editor-form example); `Short` stays byte-stable per the repo convention. The help-dump JSON schema is unchanged — the edit node's `text` updates automatically since it reproduces `-h` output.

## Tests (fake-editor seam)

Per Constitution V: table-driven, real temp dirs, no mocks. The editor is faked with `EDITOR=<small shell script>`.

- **`src/internal/idea/editor_test.go`** — `TestResolveEditor` (VISUAL wins / EDITOR fallback / `vi` default, via `t.Setenv`) and `TestEditInEditor` (rewrite + trailing-LF strip, exactly-one-LF semantics, untouched buffer, editor-appended-LF unchanged, untouched LF-terminated text unchanged via pre-strip, LF-terminated-to-bare deliberate change, CRLF normalization, multi-word editor value, non-zero exit, temp-file removal/naming).
- **`src/cmd/idea/main_test.go`** — `TestEdit_EditorForm`, twelve subprocess cases: editor rewrite, multiline round-trip with `$SIDE` decoded-buffer capture (the script records the buffer it received to a side file, proving the editor sees real newlines), trailing-LF no-op, abort, unchanged-buffer no-op with byte-identical non-canonical file, untouched LF-terminated text as byte-identical no-op, `--date` on unchanged LF-terminated text preserving text verbatim, emptied buffer, VISUAL-over-EDITOR, two-arg tripwire regression, `--date` no-op suppression, ambiguous query with tripwire.
- **Helpers**: `writeEditorScript` (executable fake-editor scripts in `t.TempDir()`), `editorEnv` (inherited env with any host `VISUAL`/`EDITOR` **scrubbed** before per-case overrides — a developer/CI `VISUAL` would otherwise shadow the fake `EDITOR`, and resolution order is the feature under test), `runSplitEnv` (`runSplit` with an explicit environment).

## Cross-references

- Source-tree placement, backlog line lifecycle (escaped text, normalize-on-write, backfill stderr notice), and the stdout/stderr channel split: `structure.md`.
- Command-table rows for both forms: `../../specs/overview.md`.
- Constitution Principles III–VI: `fab/project/constitution.md`.
- Originating change: `260612-w4p7-edit-via-editor`.
