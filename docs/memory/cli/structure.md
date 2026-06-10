---
description: "Source tree layout (cmd/idea + internal/idea), root command factory, command aliases vs. the bare-text shorthand, backlog line lifecycle (lenient read / canonical write incl. the escaped-text convention for multiline ideas), help-dump contract, and version stamping"
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
      add.go list.go show.go done.go reopen.go edit.go rm.go resolve.go update.go shell_init.go
      help_dump.go             # hidden "help-dump" subcommand (CLI tree → JSON)
      main_test.go shell_init_test.go help_dump_test.go
  internal/
    idea/                      # package logic (parsing, formatting, ID gen, file I/O, worktree resolution, self-update)
      idea.go
      idea_test.go
      update.go
      update_test.go
```

The module path is `github.com/sahil87/idea` (matches the GitHub repo URL). Direct dependencies are limited to `github.com/spf13/cobra` plus the standard library, per the constitution's dependency-discipline principle.

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

- **Principle III (Cobra-Idiomatic CLI Surface)** — subcommands are `*cobra.Command` factory functions in `cmd/idea/`; root command exposes the bare-text shorthand (`idea <text>` → `idea add <text>`); persistent flags (`--file`, `--main`) are defined on root.
- **Principle IV (Logic Lives in `internal/idea`)** — parsing, formatting, ID generation, file I/O, and worktree resolution live in `internal/idea`. `cmd/` files contain only flag wiring, argument validation, and output formatting.

The split forces a testable seam — `internal/idea` is unit-tested directly (table-driven, real temp dirs, no mocks) without spawning subprocesses. `cmd/idea/main_test.go` covers the end-to-end CLI by building the binary under test.

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

**Display semantics per command:**

| Output | Form |
|--------|------|
| `idea list` (incl. `--done`, `--all`) | escaped one line per idea via `FormatLine` — the line-per-record guarantee for external pipelines |
| `idea show` (plain) | `DisplayLine` — real newlines; continuation lines render below the `- [x] [id] date: ` prefix line |
| `list --json` / `show --json` | real newlines in the `text` field (unchanged `MarshalJSON`) |
| confirmations — `Added:` (via `idea.EscapeText` in `cmd/idea/add.go`); `Updated:`/`Done:`/`Removed:`/`Reopened:` (via `FormatLine`) | escaped single line — stdout stays machine-parseable (Constitution VI) |

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

Output is always canonical: `- ` bullet, no leading whitespace, date present, single-space delimiters, escaped text, LF line endings (`SaveFile` joins on `\n` and ends the file with a single trailing LF). Because `SaveFile` regenerates **every** recognized idea line from `FormatLine`, the first mutating command (`done`/`reopen`/`edit`/`rm`) normalizes the whole file at once: variant bullets → `-`, indentation stripped, CRLF → LF, dateless → dated, legacy lone backslashes → doubled (`\\`). This **normalize-on-write** is a deliberate, accepted trade-off — a single `idea done` can produce a large git diff on a file with many variant/dateless lines. Non-idea lines (headers, blank lines, prose) pass through unchanged (Constitution Principle I).

**Date backfill on save.** `SaveFile` stamps `time.Now().Format("2006-01-02")` on any idea whose `Date == ""` *before* serializing, and returns `(count, error)` — the count of backfilled dates. Stamping at the save seam (not in `ParseLine`) keeps `ParseLine` pure and keeps `MarshalJSON` correct, since the in-memory `Idea` has a date by the time it is marshaled after a save. The write is atomic (temp file + rename) so a crash mid-write cannot leave the source-of-truth backlog partially written.

**Backfill stderr notice (Constitution IV split).** The backfill count flows up to the command layer: the mutating internal ops `Done`, `Reopen`, `Edit`, `Rm` return `(Idea, int, error)`. When count > 0, the `cmd/idea` layer prints `note: stamped today's date on N previously-dateless item(s)` to **stderr** via the `printBackfillNotice` helper (`main.go`, using `cmd.ErrOrStderr()`); it is suppressed entirely at count 0. stdout stays the machine-parseable confirmation only (Constitution Principle VI). `internal/idea` writes nothing to stderr — output-channel policy lives in `cmd/` per Principle IV. This backfill notice is the first idea command output deliberately routed to stderr rather than stdout.

The behavior contract is documented for external consumers in `../../specs/backlog-format.md` and `../../specs/overview.md`.

## Command help text (`Short` vs `Long`)

Every subcommand sets an enriched cobra `Long` describing what it does, its key flags, the worktree-vs-`--main` resolution (for backlog-touching commands), and a short example. `Short` stays the terse one-liner used by the `Available Commands` sidebar and the `idea -h` root listing — it is a public, byte-stable string; depth goes in `Long` only. The convention was applied repo-wide by `260602-s73u-enrich-command-long-help` (the 8 backlog/update commands; `main.go` / `shell_init.go` already carried `Long`).

This is the single source for the shll.ai command-reference: the `help-dump` subcommand captures each command's `Long` + `UsageString` as the reference node's `text` (see the `help-dump` subcommand below), so the prose is written once in the binary and never drifts from the site. shll.ai pulls that JSON by running `idea help-dump` on its own schedule — `idea`'s release no longer pushes it (see `../release/pipeline.md`). New subcommands SHOULD carry a `Long` (raw backtick string, short paragraphs, inline example) rather than `Short`-only — there is no CI signal enforcing it.

## Root command factory

`cmd/idea/main.go` builds the root command through a `newRootCmd() *cobra.Command` factory rather than inline inside `main()`. The factory constructs root (with `Version: version`, the bare-text shorthand `RunE`, and the `--file`/`--main` persistent flags) and registers every subcommand:

```go
root.AddCommand(
    addCmd(), listCmd(), showCmd(), doneCmd(), reopenCmd(),
    editCmd(), rmCmd(), updateCmd(), newShellInitCmd(), helpDumpCmd(),
)
```

`main()` is then a four-line wrapper: `newRootCmd().Execute()` with the existing `errSilent` sentinel handling (`errors.Is(err, errSilent)` skips the `ERROR:` line) and `os.Exit(1)` on error.

The factory exists so the live cobra tree can be constructed in two places off the same definition: `main()` for the running binary, and `help_dump.go`'s `buildNode(cmd.Root())`, which serializes that identical tree (see below). It is also the entry point every in-process test uses — `newRootCmd()` with `SetOut(&bytes.Buffer{})` + `SetArgs(...)` exercises the full CLI without building a binary or spawning a subprocess.

## Command aliases and the bare-text shorthand

`list` is the only subcommand with an alias: `Aliases: []string{"ls"}` in the `listCmd()` command literal (`cmd/idea/list.go`), added by `260610-04rt-add-ls-alias`. `idea ls` is identical to `idea list` in every respect — same flags (`--all/-a`, `--done`, `--json`, `--sort`, `--reverse`), same inherited persistent flags (`--file`, `--main`), same output. The alias is pure routing; the `list` command's behavior and JSON output are unchanged.

**Routing rule (load-bearing).** Cobra resolves subcommand names **and aliases** before the root `RunE` bare-text fallback fires. Two consequences:

1. Before the alias existed, `idea ls` did not error — it fell through to the bare-text shorthand and silently appended a junk idea with the text "ls". The alias fixed that footgun.
2. Every alias permanently removes a word from the start of bare-text idea capture (`idea <text>` → `idea add <text>`). Adding an alias is therefore a **namespace decision, not a convenience decision** — any word claimed as an alias can never again begin an idea typed bare.

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
- Constitution principles III and IV: `fab/project/constitution.md`.
