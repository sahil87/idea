---
description: "Source tree layout (cmd/idea + internal/idea), root command factory, backlog path resolution precedence (--system / --main / --file, the XDG system backlog ($XDG_CONFIG_HOME/idea/backlog.md, else ~/.config/idea/backlog.md), and the out-of-git graceful fallback), command aliases vs. the bare-text shorthand, backlog line lifecycle (lenient read / canonical write incl. the escaped-text convention for multiline ideas and the explicit `idea fmt` canonicalizer with bare-checkbox adoption), the RequireSingle resolver query semantics (exact-ID-wins precedence over substring matches, FilterKind-respecting), the TTY/width/color/truncation seam (internal/idea/term.go) + shared printIdeaLines render path, the golang.org/x/term direct dependency, help-dump contract, and version stamping"
---

# CLI Source Structure

`idea` is a single-binary Go repo. The source tree under `src/` follows the convention used by `hop` (and other single-binary Go repos in this account): one repo-level `go.mod`, the cobra entry under `cmd/<bin>/`, and the package logic under `internal/<bin>/`.

## Layout

```
src/
  go.mod                       # module github.com/sahil87/idea
  go.sum
  cmd/
    idea/                      # cobra entry point; one file per subcommand
      main.go                  # newRootCmd() factory + main()
      add.go list.go show.go done.go reopen.go edit.go rm.go prune.go fmt.go resolve.go update.go shell_init.go
      output.go                # printIdeaLines: the single TTY-aware list/prune render path
      help_dump.go             # hidden "help-dump" subcommand (CLI tree → JSON)
      main_test.go fmt_test.go shell_init_test.go help_dump_test.go
  internal/
    idea/                      # package logic (parsing, formatting, ID gen, file I/O, worktree resolution, self-update, $EDITOR round-trip)
      idea.go
      idea_test.go
      editor.go
      editor_test.go
      fmt.go                   # idea fmt: Fmt/FmtResult + bare-checkbox adoption
      fmt_test.go
      term.go                  # TTY/width/color/truncation seam (DisplayListLine)
      term_test.go
      prune_test.go
      update.go
      update_test.go
```

The module path is `github.com/sahil87/idea` (matches the GitHub repo URL). Direct dependencies are `github.com/spf13/cobra` plus `golang.org/x/term` (the **first** non-stdlib/cobra direct dependency, added by `260613-kfcl-tty-aware-output-rendering` for terminal isatty + width detection — stdlib has no width primitive). The constitution's Dependency Discipline principle requires per-change justification rather than a hard cap; `x/term` was justified in that change's intake. It is pinned at `v0.27.0` (with `golang.org/x/sys v0.28.0` indirect) specifically to keep the `go 1.22` directive unchanged — a newer `x/term` would have bumped the directive to 1.25 (Build Reproducibility).

## Why this shape

`idea` is a single binary released independently via Homebrew tap. `hop`'s layout is the proven shape for that pattern in this account, and every piece of release machinery is built around it:

- `scripts/build.sh` runs `go build ... ./cmd/idea` from inside `src/`.
- `.github/workflows/release.yml` cross-compiles `./cmd/idea` per platform and tars the binary into `idea-{os}-{arch}.tar.gz`.
- `.github/formula-template.rb` ships `bin.install "idea"`.
- `justfile` test recipe runs `cd src && go test ./...`.

All of these reference `cmd/idea` (not `cmd/`) and assume a single repo-level `go.mod`. The layout and the release pipeline are coupled by design — see `../release/pipeline.md`.

## Contrast: when to use a different layout

`fab-kit` uses a per-binary layout (`src/go/<bin>/{cmd,internal,go.mod}`) because it ships four binaries from one monorepo with shared kit/templates directories. That layout makes sense when multiple independently-versioned binaries cohabitate one repo. `idea` ships one binary, so the simpler hop layout applies.

Rule of thumb:

- **Single-binary repo, independently released** → hop layout (`src/{go.mod, cmd/<bin>/, internal/<bin>/}`).
- **Multi-binary monorepo, shared assets, one release archive** → fab-kit layout (`src/go/<bin>/{cmd,internal,go.mod}` per binary).

## Constitutional alignment

Two principles in `fab/project/constitution.md` constrain code placement inside this layout:

- **Principle III (Cobra-Idiomatic CLI Surface)** — subcommands are `*cobra.Command` factory functions in `cmd/idea/`; root command exposes the bare-text shorthand (`idea <text>` → `idea add <text>`); persistent flags (`--file`, `--main`, `--system`) are defined on root.
- **Principle IV (Logic Lives in `internal/idea`)** — parsing, formatting, ID generation, file I/O, worktree resolution, and backlog path resolution live in `internal/idea`. `cmd/` files contain only flag wiring, argument validation, and output formatting. The full path-resolution precedence is owned by `idea.ResolveBacklogPath` (see *Backlog path resolution* below); `resolveFile()` in `cmd/idea/resolve.go` is a one-line forwarder of the three persistent-flag values, holding no precedence logic.

The split forces a testable seam — `internal/idea` is unit-tested directly (table-driven, real temp dirs, no mocks) without spawning subprocesses. `cmd/idea/main_test.go` covers the end-to-end CLI by building the binary under test.

## Backlog path resolution

Which backlog file a command operates on is decided by `idea.ResolveBacklogPath(systemFlag, mainFlag bool, fileFlag string) (string, error)` in `internal/idea/idea.go`. This is the sole owner of the precedence (Constitution IV); `cmd/idea/resolve.go`'s `resolveFile()` only forwards the three persistent-flag values into it. The out-of-git fallback and the system backlog were added by `260613-2b3m-system-level-backlog`, which made `idea` usable outside any git repository — previously every command shelled out to `git rev-parse` and **failed hard** ("not in a git repository") in any non-repo directory, with `--file`/`IDEAS_FILE` unable to rescue it because their values were still joined to the failed git root.

### Three persistent root selectors

Defined on root in `newRootCmd()` (`cmd/idea/main.go`):

| Flag | Var | Effect |
|------|-----|--------|
| `--file <path>` / `IDEAS_FILE` env | `fileFlag` | Override the backlog file path; rooted at the git root inside a repo, else at `~/.config/idea` (an absolute value is honored verbatim). |
| `--main` | `mainFlag` | Operate on the **main worktree's** backlog. Git-only (errors outside a repo) — unchanged by 2b3m. |
| `--system` | `systemFlag` | Operate on the **system backlog**, from anywhere including inside a repo; skips git entirely. Peer of `--main`, added by 2b3m. |

### Precedence (first match wins)

`ResolveBacklogPath` resolves in this order:

1. **`--system`** → the system backlog (`SystemBacklogPath()`); git is skipped entirely.
2. **`--main`** → the main worktree root (`MainRepoRoot()`), then `--file`/`IDEAS_FILE` rooting applied via `ResolveFilePath`. Git-only — errors with "not in a git repository" outside a repo (unchanged).
3. **Inside a git repo, no `--system`/`--main`** → `WorktreeRoot()` succeeds → `ResolveFilePath(worktreeRoot, fileFlag)`: a `--file`/`IDEAS_FILE` override joined to the worktree root, else the **unchanged default** `{worktree-root}/fab/backlog.md`.
4. **Outside any git repo, no `--system`/`--main`** → `WorktreeRoot()` errors → the **graceful fallback**: a relative `--file`/`IDEAS_FILE` value is joined to `~/.config/idea`, an absolute one is honored verbatim, and with no override the path is the system backlog (`{config-dir}/idea/backlog.md`). Commands no longer fail outside a repo.

**`--system` + `--main` is a hard conflict.** Both select a root; passing both returns `--system and --main are mutually exclusive; pass only one` (non-zero exit via the existing top-level `ERROR:` handler) and resolves no path. The check is the first line of `ResolveBacklogPath`, colocated with the precedence it guards rather than in a separate cobra `PreRunE`.

### System backlog location (XDG)

`SystemBacklogPath() (string, error)` returns `{config-dir}/idea/backlog.md`, where `config-dir` comes from Go stdlib `os.UserConfigDir()` — `$XDG_CONFIG_HOME` when set, else `~/.config` on Unix. So:

- `XDG_CONFIG_HOME=/custom/cfg` → `/custom/cfg/idea/backlog.md`
- unset, `HOME=/home/u` → `/home/u/.config/idea/backlog.md`

This mirrors `hop`'s `~/.config/hop/hop.yaml` convention and stays stdlib-only — **no new dependency** (Dependency Discipline). `os.UserConfigDir` is the only XDG-resolution path; both `SystemBacklogPath` and the out-of-git override-rooting branch use it.

### On-demand config-dir creation

The system config dir (`~/.config/idea/`) is created lazily on the **first mutating write**, not on read. The `os.MkdirAll(filepath.Dir(path), 0755)` lives at the single SaveFile serialization seam — `atomicWriteFile` — so every SaveFile-based mutation (`done`/`reopen`/`edit`/`rm`/`prune --force`/`fmt`) creates a missing dir before writing; `Add` already MkdirAll's its own path independently. Read-only commands (`list`/`show`) on a non-existent system backlog take the existing "no ideas file yet" path — they neither create the dir nor error.

### Format is path-independent

Only path resolution changed. The backlog file format, ID rules, escaping, canonical write, and all CRUD/`fmt`/`list`/`show`/JSON semantics are byte-for-byte identical regardless of which path resolved (Constitution I) — the system backlog is the same canonical Markdown checklist at a different location. The behavior contract for external consumers is in `../../specs/overview.md` ("Worktree Behavior" + resolution precedence).

**Constitution II scope.** Principle II ("worktree-aware by default, all resolution via `git rev-parse`") still governs the in-git case — git resolution remains the default and the only path used when a repo is present and no override is given. 2b3m added a *sanctioned* non-git path (the system backlog) without amending Principle II; the escape hatch is documented in the spec/overview only, a deliberate non-blocking judgment call left at intake.

## Backlog line lifecycle (lenient read, canonical write)

`internal/idea/idea.go` owns the parse/format/save contract for backlog lines. The governing principle is **lenient on read, canonical on write** (be liberal in what you accept, strict in what you emit). This shape was established by `260610-wtmn-resilient-backlog-parser`, which fixed silent-failure parsing of dateless backlogs (e.g. shll.ai's `- [ ] [id] text` form).

### Parse (lenient)

`ParseLine` matches against `lineRegex`:

```go
^\s*[-*+] \[([ x])\] \[([a-z0-9]{4})\] (?:(\d{4}-\d{2}-\d{2}): )?(.+)$
```

Accepted input variants (all parse to the same `Idea`, modulo `Date`):

| Dimension | Accepted | Notes |
|-----------|----------|-------|
| Date segment | present **or** absent | dateless line yields `Date == ""` |
| Bullet marker | `-`, `*`, `+` | |
| Leading whitespace | any (spaces, tabs) | stripped on canonicalize |
| Line endings | CRLF or LF | `LoadFile` strips a trailing `\r` from each split line before parsing |

The `[ ]`/`[x]` checkbox plus the 4-char `[a-z0-9]{4}` id are the anchors that keep false-positive matching of genuine prose low. The date group is non-capturing, so the date is optional without a second regex.

`ParseLine` is **pure**: it never stamps a date. A dateless line parses with `Date == ""`, so non-mutating reads (`list`, `show`) and `MarshalJSON` faithfully reflect on-disk content — a never-saved dateless idea shows `"date": ""`.

**Shape B precision guard.** Legacy "Shape B" second-bracket lines (`- [ ] [id] [DEV-1011] date: text`) — and, by extension, any captured text beginning with a `[...]` bracket — stay **inert pass-through**. `ParseLine` rejects them via `shapeBPrefixRegex` (`^\[[^\]]*\]` against the captured text group), returning `ok=false` so they are preserved verbatim. The `[issue_ids]` slot is owned by external consumers (fab-kit's `/fab-new`); idea must neither parse nor rewrite these lines. Erring toward preservation is safer than rewriting an external tool's line.

### Escaped text (multiline ideas)

Idea text is **escaped on write, unescaped on read**, so an idea whose text contains newlines still occupies exactly one physical line. Established by `260610-49mw-escape-multiline-idea-text`, which fixed raw multiline pastes silently corrupting the backlog (truncation to the first line, orphaned continuation prose that `rm`/`edit` never touched, and phantom ideas when a pasted line itself matched `lineRegex`). Exactly two escape sequences exist, applied to idea text only — never to non-idea pass-through lines:

| In-memory character | Persisted sequence (2 chars) |
|---------------------|------------------------------|
| `\` backslash (U+005C) | `\\` |
| LF (U+000A) | `\n` |

The helpers are exported from `internal/idea` and built on package-level `strings.Replacer` pairs (`textEscaper`/`textUnescaper`) — stdlib-only, single deterministic left-to-right pass:

- **`EscapeText`** (write direction): CR normalization first (`normalizeCR`: CRLF → LF, then any remaining lone CR → LF), then `\` → `\\` and LF → `\n`. The result contains no raw LF or CR by construction, so a persisted idea line is always one physical line and always matches the single-line `lineRegex`. No raw CR ever reaches the file.
- **`UnescapeText`** (read direction, lenient): `\\` → `\`, `\n` → LF; a backslash followed by any **other** character (e.g. `\b` stays `\b`) and a trailing lone `\` pass through verbatim — unrecognized escapes never error. The Replacer's copy-through-unmatched-bytes behavior implements this leniency exactly.
- **Round-trip law**: `UnescapeText(EscapeText(x)) == x` for any CR-free `x` (including backslash-heavy text like `C:\new`, `a\\b`, trailing `\`); for `x` containing CR it equals `normalizeCR(x)` — CR→LF normalization is the only deliberate loss. `Add` and `Edit` also call `normalizeCR` on incoming text before storing it, so the in-memory `Idea` always equals what round-trips from disk.

**In-memory real, on-disk escaped.** `ParseLine` applies `UnescapeText` to the captured text group — *after* the Shape B precision guard, which evaluates the raw on-disk text. `Idea.Text` therefore always holds the **real** text (raw newlines, raw backslashes) while the file holds the escaped form. Consequences: `MarshalJSON` needed no change (JSON encodes the newlines itself, so `--json` output carries real newlines in `text`), and `Match`/query semantics operate on the real text the user typed.

**Display semantics per command.** `idea list`/`ls` and the `idea prune` dry-run/pre-confirm listing are **TTY-aware** (since `260613-kfcl-tty-aware-output-rendering`): on a terminal they truncate text to the width and color the prefix/checkbox via `DisplayListLine`; when piped or redirected they emit the full canonical `FormatLine` regardless, preserving the line-per-record pipe contract (Constitution VI). Both route through the single `printIdeaLines` helper (see § Shared TTY-aware render path). `--json`, `show`, and all confirmations are unchanged.

| Output | Form |
|--------|------|
| `idea list` / `ls` (incl. `--done`, `--all`, `[id...]`) — **TTY** | one line per idea via `DisplayListLine`: text truncated to width with `…` (unless `--full`), `[id] date:` prefix dimmed, done `[x]` greened (unless `NO_COLOR`) — see `list.md` |
| `idea list` / `ls` — **piped/redirected** | escaped one line per idea via `FormatLine` (no truncation, no ANSI) regardless of `--full` — the line-per-record guarantee for external pipelines |
| `idea show` (plain) | `DisplayLine` — real newlines; continuation lines render below the `- [x] [id] date: ` prefix line |
| `list --json` / `show --json` | real newlines in the `text` field (unchanged `MarshalJSON`); display features never touch JSON |
| confirmations — `Added:` (via `idea.EscapeText` in `cmd/idea/add.go`); `Updated:`/`Done:`/`Removed:`/`Reopened:` (via `FormatLine`) | escaped single line — stdout stays machine-parseable (Constitution VI) |
| `idea prune` (dry run / pre-confirm) — **TTY** | removable done ideas via `DisplayListLine` (truncated/colored unless `--full`) on stdout; a `N done idea(s) would be pruned` count header + a `Prune N done idea(s)? [y/N]` confirm prompt on **stderr** — see `prune.md` |
| `idea prune` (dry run) — **piped/redirected** | escaped one line per removable done idea via `FormatLine` on stdout (pipe-friendly, e.g. `idea prune \| wc -l`); the count header + `Re-run with --force to confirm.` hint go to **stderr**; never prompts |
| `idea prune --force` confirmation | `Pruned N done idea(s).` — count only, no per-line listing (per-line is reserved for the non-force paths; see `prune.md`) |

**Legacy backslash policy (pre-convention files).** Lines written before the escape convention may contain literal backslashes:

- Unrecognized escapes read **verbatim** (`a\b` reads as `a\b`).
- On the next mutating save, in-memory `\` re-serializes as `\\` (on-disk `a\b` → `a\\b`) — content unchanged, only the encoding canonicalizes: a one-time normalize-on-write consistent with the precedent below. A second save is byte-stable (no further churn).
- Accepted consequence: a legacy literal two-character `\n` inside text (e.g. `C:\new`) is reinterpreted on read as a real newline. This is unavoidable under any unescape-on-read scheme (the stored bytes are ambiguous with legacy data) and judged rare; re-saving is stable (the newline re-escapes to `\n`).

### Format / Save (canonical)

The canonical format string lives in **one private formatter**, shared by two exported renderers so line-format knowledge never leaks out of `internal/idea` (Constitution IV; arrangement from `260610-49mw-escape-multiline-idea-text` — the earlier claim that `FormatLine` was untouched after the resilience work no longer holds):

```go
func formatLineWith(i Idea, text string) string {
    return fmt.Sprintf("- [%s] [%s] %s: %s", i.StatusCheck(), i.ID, i.Date, text)
}
```

- `FormatLine(i)` = `formatLineWith(i, EscapeText(i.Text))` — the **persisted/escaped** form; the single source of on-disk output truth. Every write path inherits the one-physical-line guarantee through it: `Add`'s append, `SaveFile`'s rebuild, the confirmations, and `RequireSingle`'s multi-match error listing.
- `DisplayLine(i)` = `formatLineWith(i, i.Text)` — the **real-text** form for human-facing display (plain `idea show`).

Output is always canonical: `- ` bullet, no leading whitespace, date present, single-space delimiters, escaped text, LF line endings (`SaveFile` joins on `\n` and ends the file with a single trailing LF). Because `SaveFile` regenerates **every** recognized idea line from `FormatLine`, the first mutating command (`done`/`reopen`/`edit`/`rm`, or `prune --force` when it removes items) normalizes the whole file at once: variant bullets → `-`, indentation stripped, CRLF → LF, dateless → dated, legacy lone backslashes → doubled (`\\`). This **normalize-on-write** is a deliberate, accepted trade-off — a single `idea done` can produce a large git diff on a file with many variant/dateless lines. To land that churn as its own commit — with no semantic change riding along — use the explicit canonicalizer `idea fmt` (see below). Non-idea lines (headers, blank lines, prose) pass through unchanged (Constitution Principle I).

**Date backfill on save.** The stamping lives in `render(f *File, today string) (string, int)` — the date-stamp + rebuild step extracted from `SaveFile` by `260612-4m3a-add-fmt-canonicalizer-adoption` so `Fmt` can build the canonical content without writing it. `render` stamps the caller-supplied `today` on any idea whose `Date == ""` *before* serializing and returns the backfill count; `SaveFile` is now `render(f, time.Now().Format("2006-01-02"))` + atomic write, still returning `(count, error)`, while `Fmt` passes the same `today` it used for counting and adoption so one run stamps one consistent date even across a midnight boundary (PR-review fix). Stamping at the save seam (not in `ParseLine`) keeps `ParseLine` pure and keeps `MarshalJSON` correct, since the in-memory `Idea` has a date by the time it is marshaled after a save. The write is atomic (temp file + rename) so a crash mid-write cannot leave the source-of-truth backlog partially written.

**Backfill stderr notice (Constitution IV split).** The backfill count flows up to the command layer: the mutating internal ops `Done`, `Reopen`, `Edit`, `Rm` return `(Idea, int, error)`. When count > 0, the `cmd/idea` layer prints `note: stamped today's date on N previously-dateless item(s)` to **stderr** via the `printBackfillNotice` helper (`main.go`, using `cmd.ErrOrStderr()`); it is suppressed entirely at count 0. stdout stays the machine-parseable confirmation only (Constitution Principle VI). `internal/idea` writes nothing to stderr — output-channel policy lives in `cmd/` per Principle IV. The backfill notice was the first idea command output deliberately routed to stderr; `prune`'s dry-run confirm hint (`Re-run with --force to confirm.` — see `prune.md`), `idea edit`'s editor-form no-op note (`note: text unchanged — nothing to do` — see `edit.md`), and `idea fmt`'s entire report (adoption lines + summary counts — see below) followed, making advisory-notes-to-stderr the established channel policy rather than a one-off.

The behavior contract is documented for external consumers in `../../specs/backlog-format.md` and `../../specs/overview.md`.

### Resolver query semantics (exact-ID-wins precedence)

`RequireSingle(query, ideas, filter)` is the single shared seam every single-idea subcommand (`show`/`done`/`reopen`/`edit`/`rm`) routes through to resolve one idea from a query. Its query semantics are **exact-ID-wins** (since `260615-m2qx-exact-id-resolution-precedence`): before any substring matching, `RequireSingle` short-circuits on a case-insensitive exact ID match (`strings.EqualFold(idea.ID, query)`) and returns that idea outright. The short-circuit respects the active `FilterKind` (via `matchesFilter`), so `done <id>` (FilterOpen) does not resolve a *done* idea's exact ID while `reopen <id>` (FilterDone) and the FilterAll commands do. This makes an idea always reachable by its own ID even when that ID string appears as a substring inside another idea's text (the `jznd`/`qg64` cross-reference bug, where `idea edit jznd …` previously errored "Multiple matches" because `qg64`'s body contained the literal `[jznd]`). Soundness rests on Constitution VI — IDs are unique within a backlog, so an exact ID match can never be genuinely ambiguous; the defensive fall-through if two filtered ideas somehow share an exact ID (set the tracked index back to -1 and drop into the substring loop, which then reports them) is unreachable in practice. Only `RequireSingle`'s precedence changed: `Match()`/`FindAll()`/`matchesFilter()`/`FilterKind` are byte-for-byte unchanged, so substring search for non-ID queries — and `list`'s fuzzy lookup via `FindAll` — keeps its prior behavior, including the unchanged "No idea matching" (zero hits) and "Multiple matches" (ambiguous substring) error branches. The fix lives entirely in `internal/idea` with no `cmd/` changes (Constitution IV); fixing the one seam fixed all five subcommands at once.

### Explicit canonicalizer & adoption (`idea fmt`)

`idea fmt` (added by `260612-4m3a-add-fmt-canonicalizer-adoption`) is the explicit, gofmt-style trigger for the canonical write above: it rewrites the whole backlog into canonical form with no semantic change required, so the normalize-on-write churn can land as its own commit. `fmt` is the **only explicit whole-file write verb** — mutating CRUD commands keep their incidental normalize-on-write; `list`/`show` stay non-mutating.

**Internal seams (no second serialization path).** `internal/idea/fmt.go` exports `Fmt(path string, check bool) (FmtResult, error)` with `FmtResult{Adopted []Idea, Normalized, Backfilled int, Changed bool}`. To let `Fmt` reuse the single parse walk and single serialization point, the same change split `LoadFile` into a thin wrapper over `parseContent(content string) *File` and `SaveFile` into atomic-write over `render(f *File, today string) (string, int)` (date-stamp + rebuild, no write; the caller supplies the date). `File.lines` now retains each idea line's raw (post-`\r`-strip) text instead of the former `""` placeholder — required for per-line `Normalized` counting (canonical `FormatLine` output vs. raw line) and invisible to every other caller, since `render`'s rebuild overwrites idea slots from `FormatLine`. Public `LoadFile`/`SaveFile` behavior is unchanged.

**Automatic adoption of bare checkboxes.** `fmt` additionally brings plain markdown task-list lines under management — the only path that does. A line is an adoption candidate iff `ParseLine` rejects it AND it matches `adoptRegex` (`internal/idea/fmt.go`):

```go
^\s*[-*+] \[([ xX])\] (.+)$
```

with two guards evaluated on the **whitespace-trimmed** captured text — trimmed first so extra spaces between the checkbox and a bracket cannot defeat the guard (`- [ ]  [DEV-1011] x` stays inert; tightened by this change's post-review rework): blank text is not adopted, and bracket-led text (`shapeBPrefixRegex` — Shape B remnants, `[DEV-1011]`, `[TODO]`, non-4-char `[ab1]`) is not adopted, erring toward preservation. Each adopted line gets a fresh 4-char ID unique against both the file's parsed IDs and IDs assigned earlier in the same pass (`generateUniqueIDInSet`, an in-memory-set variant mirroring `generateUniqueID`'s retry skeleton — dedup of the two deferred), date = today (counted as *Adopted*, never as *Backfilled* — backfill counts only pre-existing managed dateless lines), checked state preserved (`[x]`/`[X]` adopt as done, canonical `[x]`), and the trimmed text stored as real text (escaped via `FormatLine` on write like every idea). Adopted indented/nested checkboxes flatten to top level — canonical form has no indentation.

**Whole-file verdict.** `FmtResult.Changed` compares the rebuilt content against the original on-disk bytes; it drives both the write (skipped entirely when byte-identical — an idempotent second run touches nothing, not even mtime; a 0-byte file is left untouched, no trailing LF invented) and the `--check` exit code. Accepted edge: a CRLF-only or EOF-newline-only difference rewrites the file / fails `--check` while the per-line counts may read zero — `Changed` is authoritative, the counts are approximate.

**cmd layer & output contract** (`cmd/idea/fmt.go`): `fmtCmd()` with a local `--check` flag and `Args: cobra.NoArgs`. stdout stays empty on every path — success is silence + exit 0, the `gofmt -w` precedent (Constitution VI). stderr composition, in order: one `adopted: [id] {escaped text}` line per adopted idea (file order, via `EscapeText`), then `printBackfillNotice` verbatim, then `fmt: N line(s) normalized, M line(s) adopted` printed only when the file changes (or would change). Zero-activity runs print nothing; `internal/idea` writes nothing to stderr (Constitution IV split). `--check` writes nothing, prints the same report, and exits 1 via the `errSilent` sentinel when non-canonical (0 when clean) — one flag serving both the dry-run preview and the scripts/CI gate. Accepted edge: the `--check` report shows freshly generated would-be IDs that are never persisted — a later real run assigns different ones.

**Namespace claim.** Cobra resolves the `fmt` subcommand name before the root bare-text fallback, so `idea fmt some text` errors instead of adding an idea — the same namespace trade as the `ls` alias decision; "fmt" was judged to plausibly never begin bare idea prose.

**Governance note (Constitution I).** Adoption narrows the round-trip preservation guarantee: bare checkbox lines lacking the `[id]` anchor were previously guaranteed non-idea pass-through, and `fmt` (and only `fmt`) now claims them. The carve-out is documented in `../../specs/backlog-format.md` (Round-Trip Preservation carve-out + format-contract change note) but the constitution text itself is unamended — judged defensible at review because `fmt` is an explicit migration verb the user invokes, not round-trip parsing; a future constitution amendment could codify the carve-out explicitly.

**Uniqueness blind spot (consistent, accepted).** Adopted-ID uniqueness checks parsed ideas only — a 4-char bracket inside an unparseable line is invisible to it, exactly as it is to `Add`'s `checkIDCollision`.

## TTY/width/color/truncation seam (`internal/idea/term.go`)

`internal/idea/term.go` (added by `260613-kfcl-tty-aware-output-rendering`) is the single home of all terminal-aware rendering logic — the Constitution IV seam for the display features. `cmd/` only asks it for a decision (is this a TTY? how wide? use color?) and for a ready-to-print display line; `FormatLine`/`DisplayLine` stay the machine/canonical renderers and are intentionally untouched (display-only change). The exported surface:

| Symbol | Role |
|--------|------|
| `IsTTY(f *os.File) bool` | `term.IsTerminal(int(f.Fd()))`; nil-safe (returns false). The single gate every display change keys on so piped/redirected output stays canonical (Constitution VI). |
| `TermWidth(f *os.File) int` | Resolution order `term.GetSize` → `$COLUMNS` (when it parses to a positive int) → the `defaultTermWidth` const (80). |
| `UseColor(f *os.File) bool` | True only when `f` is a TTY **and** `NO_COLOR` is unset. Uses `os.LookupEnv` presence (not truthiness) per the NO_COLOR spec — any value, including empty, disables color. |
| `DisplayListLine(i Idea, width int, full, color bool) string` | The rune-safe display-line builder (below). Has exactly **one** caller: `printIdeaLines` in `cmd/idea/output.go`. |

Internal helpers: `truncateText(text string, avail int)` (rune-safe `[]rune` clip with the `ellipsis` U+2026 const, multiline-at-first-`\n`), `dimPrefix`/`greenCheck` (ANSI wrappers). ANSI codes are named consts (`ansiReset`/`ansiFaint`/`ansiGreen`), and 80 is the named `defaultTermWidth` const — no magic numbers/strings (code-quality Anti-Patterns).

**Width is a parameter, not read inside the builder**, so tests inject it rather than allocating a real PTY (Constitution V). **Color is applied AFTER truncation** so the width math counts visible runes, never escape bytes. The `- [done] [id] date: ` prefix is never truncated; when colored, the checkbox is its own (green-when-done) span between two dim spans so the id/date stay faint. Full truncation/color contract: `list.md`.

`internal/idea/term_test.go` is table-driven against the seam (Constitution V): `TermWidth` fallback (GetSize-fail via the non-TTY path + `$COLUMNS` set/unset/invalid → 80), `UseColor`/color helpers honoring `NO_COLOR`, and `DisplayListLine` (multibyte rune-safety, prefix-never-truncated, ellipsis presence, multiline-at-first-newline, `full` bypasses truncation, color applied after truncation).

### Shared TTY-aware render path (`cmd/idea/output.go`)

`printIdeaLines(out io.Writer, ideas []idea.Idea, full bool)` in `cmd/idea/output.go` is the **single** TTY-aware render path, shared by `list.go` and `prune.go` (extracted during this change's review rework so the render loop is not duplicated — code-quality Anti-Patterns, A-017). It keys the TTY/width/color decision on `os.Stdout` (the real destination) while writing to `out` (normally `os.Stdout` in production, a buffer under test): on a TTY each idea renders via `idea.DisplayListLine(i, width, full, color)`; when piped it emits `idea.FormatLine(i)` regardless of `full`. The Constitution IV split holds — `output.go` only *picks the mode*; all truncation/color logic lives in `term.go`.

## Command help text (`Short` vs `Long`)

Every subcommand sets an enriched cobra `Long` describing what it does, its key flags, the worktree-vs-`--main` resolution (for backlog-touching commands), and a short example. `Short` stays the terse one-liner used by the `Available Commands` sidebar and the `idea -h` root listing — it is a public, byte-stable string; depth goes in `Long` only. The convention was applied repo-wide by `260602-s73u-enrich-command-long-help` (the 8 backlog/update commands; `main.go` / `shell_init.go` already carried `Long`).

This is the single source for the shll.ai command-reference: the `help-dump` subcommand captures each command's `Long` + `UsageString` as the reference node's `text` (see the `help-dump` subcommand below), so the prose is written once in the binary and never drifts from the site. shll.ai pulls that JSON by running `idea help-dump` on its own schedule — `idea`'s release no longer pushes it (see `../release/pipeline.md`). New subcommands SHOULD carry a `Long` (raw backtick string, short paragraphs, inline example) rather than `Short`-only — there is no CI signal enforcing it.

## Root command factory

`cmd/idea/main.go` builds the root command through a `newRootCmd() *cobra.Command` factory rather than inline inside `main()`. The factory constructs root (with `Version: version`, the bare-text shorthand `RunE`, and the `--file`/`--main`/`--system` persistent flags — see *Backlog path resolution* above for what each selects) and registers every subcommand:

```go
root.AddCommand(
    addCmd(), listCmd(), showCmd(), doneCmd(), reopenCmd(),
    editCmd(), rmCmd(), pruneCmd(), fmtCmd(), updateCmd(), newShellInitCmd(), helpDumpCmd(),
)
```

`main()` is then a four-line wrapper: `newRootCmd().Execute()` with the existing `errSilent` sentinel handling (`errors.Is(err, errSilent)` skips the `ERROR:` line) and `os.Exit(1)` on error.

The factory exists so the live cobra tree can be constructed in two places off the same definition: `main()` for the running binary, and `help_dump.go`'s `buildNode(cmd.Root())`, which serializes that identical tree (see below). It is also the entry point every in-process test uses — `newRootCmd()` with `SetOut(&bytes.Buffer{})` + `SetArgs(...)` exercises the full CLI without building a binary or spawning a subprocess.

## Command aliases and the bare-text shorthand

`list` is the only subcommand with an alias: `Aliases: []string{"ls"}` in the `listCmd()` command literal (`cmd/idea/list.go`), added by `260610-04rt-add-ls-alias`. `idea ls` is identical to `idea list` in every respect — same flags (`--all/-a`, `--done`, `--json`, `--sort`, `--reverse`), same inherited persistent flags (`--file`, `--main`, `--system`), same output. The alias is pure routing; the `list` command's behavior and JSON output are unchanged.

**Routing rule (load-bearing).** Cobra resolves subcommand names **and aliases** before the root `RunE` bare-text fallback fires. Two consequences:

1. Before the alias existed, `idea ls` did not error — it fell through to the bare-text shorthand and silently appended a junk idea with the text "ls". The alias fixed that footgun.
2. Every alias permanently removes a word from the start of bare-text idea capture (`idea <text>` → `idea add <text>`). Adding an alias is therefore a **namespace decision, not a convenience decision** — any word claimed as an alias can never again begin an idea typed bare. The same holds for **subcommand names** themselves: the `prune` verb (added by `260612-drc1-add-prune-subcommand`) claims `prune` from the bare-text namespace — `idea prune the old cache` now routes to the subcommand (and errors under its `cobra.NoArgs`) instead of capturing an idea beginning with "prune"; `fmt` was accepted on the same bar by `260612-4m3a-add-fmt-canonicalizer-adoption` ("fmt" plausibly never begins bare idea prose).

**Rejected aliases** (surveyed and rejected in the 04rt intake discussion; future proposals must clear the same bar): `remove`/`delete` (rm), `upgrade` (update), `cat` (show) — each plausibly starts bare-text idea prose; `undo` (reopen) — implies revert-last-action semantics worth reserving. Scope decision: `ls` is the only alias.

**Guard test.** `TestRouting_LsAliasAndBareShorthand` (`cmd/idea/main_test.go`) is a table-driven subprocess test asserting (a) `ls` / `ls --json` stdout is byte-identical to `list` / `list --json` on a seeded backlog with the backlog file unchanged, and (b) bare text with a non-alias first word (`idea lsx some text`) still routes to the add shorthand. It reuses the existing `buildBinary`/`setupGitRepo`/`writeRepoBacklog`/`runSplit` helpers plus a `readRepoBacklog` helper added by the same change.

**help-dump interaction.** No `aliases` field was added to the help-dump JSON schema — the schema is a frozen cross-repo contract (Constitution VI), and an additive field is deferred until an external consumer needs it. The list node's `text` automatically gains cobra's rendered `Aliases: list, ls` line, since `text` reproduces what `-h` prints (see the next section).

## Hidden `help-dump` subcommand

`cmd/idea/help_dump.go` registers a hidden (`Hidden: true`, `Use: "help-dump"`) subcommand that emits the CLI help tree as JSON to `OutOrStdout()`. It is consumed by shll.ai, which **pulls** it by `brew install`ing `idea` and running `idea help-dump` on its own schedule (see `../release/pipeline.md`); the JSON shape is a **frozen cross-repo contract** shared across a 7-tool rollout, with `sahil87/shll.ai` `help/wt.json` as the reference sample.

**Envelope** (`helpDump`, in field/JSON order): `tool` (literal `"idea"`), `version` (read from `cmd.Root().Version`, the ldflags-stamped value — never hardcoded), `captured_at` (`time.Now().UTC().Format(time.RFC3339)`), `schema_version` (the `helpSchemaVersion` const, `1`), `root` (`buildNode(cmd.Root())`).

**Node** (`helpNode`, in field/JSON order): `name` (`cmd.Name()`), `path` (`cmd.CommandPath()`, e.g. `idea add`), `short` (`cmd.Short`), `usage` (`cmd.UseLine()`), `text`, `commands` (recursive `[]helpNode`, initialized to `[]helpNode{}` so leaves serialize as `[]`, never `null`).

The output is `json.MarshalIndent(dump, "", "  ")` (2-space indent) plus a trailing newline. The subcommand is stdout-only — it has no `--output` flag; the consumer (shll.ai's pull job) owns file placement, redirecting stdout into its own `help/idea.json`.

### Three implementation details that are load-bearing for the contract

1. **`text` composition** — each node's `text` reproduces byte-for-byte what `idea <cmd> -h` prints: `longOrShort(cmd) + "\n\n" + cmd.UsageString()`, where `longOrShort` returns `cmd.Long` if non-empty else `cmd.Short`. `UsageString()` renders only the `Usage:`/`Available Commands:`/`Flags:` blocks (not Long/Short), so the description is concatenated explicitly. If **both** `Long` and `Short` are empty, `text` is just `UsageString()` with no leading blank lines (defensive guard).

2. **Default-flag materialization** — `buildNode` calls `cmd.InitDefaultHelpFlag()` and `cmd.InitDefaultVersionFlag()` **before** reading `UsageString()`. Cobra registers `-h, --help` (and, on a versioned command, `-v, --version`) lazily inside `Execute()`. `help-dump` walks the child commands without executing them, so without these calls the rendered `Flags:` block would omit those flags and diverge from real `-h` output. `InitDefaultHelpFlag()` always adds `-h, --help`; `InitDefaultVersionFlag()` materializes `-v, --version` only when `cmd.Version != ""`, so it adds the flag on the versioned root and is a no-op on subcommands. Both are idempotent. This means root `text` carries both `-h, --help` and `-v, --version`; subcommand `text` carries only `-h, --help`.

3. **Recursion filter** — during the `cmd.Commands()` walk, `buildNode` skips any child where `c.Hidden`, `c.Name() == "completion"`, or `c.Name() == "help"`. The `Hidden` clause also excludes `help-dump` itself from its own output. Filtered subtrees are simply not appended (no placeholder).

`help_dump_test.go` exercises the command in-process via `newRootCmd()` and asserts the envelope, root `name`/`path`, filter exclusions, every real subcommand present with the correct `path`, leaf `commands: []` on the raw JSON bytes, and (the contract lock) that root `text` contains both `-h, --help` and `-v, --version` while a leaf's `text` contains `-h, --help` but not `-v, --version`.

## Version stamping

`cmd/idea/main.go` declares a package-level `var version = "dev"`, with the comment:

```go
// version is the binary version, overridden via -ldflags "-X main.version=..." at build time.
var version = "dev"
```

The root cobra command has `Version: version` set in its struct literal, which gives the binary `--version` support automatically. The pattern mirrors `wt`'s `cmd/wt/main.go`.

This wiring is required because `idea` is released independently:

1. `scripts/build.sh` injects the version via `-ldflags "-X main.version=${VERSION}"` (where `VERSION` comes from `git describe --tags --always`).
2. `.github/workflows/release.yml` does the same per cross-compile target, using the tag name.
3. `.github/formula-template.rb`'s `test do` block runs `#{bin}/idea --version` as the Homebrew install-time check; without the wiring, that test would fail and the formula install would be marked broken.

## Cross-references

- Release pipeline that consumes this layout (build path, version stamping, Homebrew formula); shll.ai pulls the command reference via `idea help-dump` (the release no longer publishes it): `../release/pipeline.md`.
- Self-update subcommand built on top of the Homebrew tap (`update.go` / `internal/idea/update.go`): `update.md`.
- `idea list`/`ls` TTY-aware rendering contract — truncation, `--full`, the `[id...]` filter, and color (`list.go` / `internal/idea/term.go`): `list.md`.
- Bulk-remove subcommand (`prune.go` / `idea.Prune`): dry-run/`--force` contract, the count header + interactive `[y/N]` confirm, output channels, and the deliberate non-archival design: `prune.md`.
- `edit` subcommand two-form contract and the `$VISUAL`/`$EDITOR`/`vi` temp-file round trip (`edit.go` / `internal/idea/editor.go`): `edit.md`.
- Constitution principles III and IV: `fab/project/constitution.md`.
