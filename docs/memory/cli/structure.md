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

## Hidden `help-dump` subcommand

`cmd/idea/help_dump.go` registers a hidden (`Hidden: true`, `Use: "help-dump"`) subcommand that emits the CLI help tree as JSON to `OutOrStdout()`, for build tooling. It is consumed by the release pipeline's shll.ai command-reference publisher (see `../release/pipeline.md`); the JSON shape is a **frozen cross-repo contract** shared across a 7-tool rollout, with `sahil87/shll.ai` `help/wt.json` as the reference sample.

**Envelope** (`helpDump`, in field/JSON order): `tool` (literal `"idea"`), `version` (read from `cmd.Root().Version`, the ldflags-stamped value — never hardcoded), `captured_at` (`time.Now().UTC().Format(time.RFC3339)`), `schema_version` (the `helpSchemaVersion` const, `1`), `root` (`buildNode(cmd.Root())`).

**Node** (`helpNode`, in field/JSON order): `name` (`cmd.Name()`), `path` (`cmd.CommandPath()`, e.g. `idea add`), `short` (`cmd.Short`), `usage` (`cmd.UseLine()`), `text`, `commands` (recursive `[]helpNode`, initialized to `[]helpNode{}` so leaves serialize as `[]`, never `null`).

The output is `json.MarshalIndent(dump, "", "  ")` (2-space indent) plus a trailing newline. The subcommand is stdout-only — it has no `--output` flag; the CI step owns file placement (redirecting into `help/idea.json`).

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

- Release pipeline that consumes this layout (build path, version stamping, Homebrew formula) and runs `help-dump` to publish the command reference to shll.ai: `../release/pipeline.md`.
- Self-update subcommand built on top of the Homebrew tap (`update.go` / `internal/idea/update.go`): `update.md`.
- Constitution principles III and IV: `fab/project/constitution.md`.
