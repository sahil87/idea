# CLI Source Structure

`idea` is a single-binary Go repo. The source tree under `src/` follows the convention used by `hop` (and other single-binary Go repos in this account): one repo-level `go.mod`, the cobra entry under `cmd/<bin>/`, and the package logic under `internal/<bin>/`.

## Layout

```
src/
  go.mod                       # module github.com/sahil87/idea
  go.sum
  cmd/
    idea/                      # cobra entry point; one file per subcommand
      main.go
      add.go list.go show.go done.go reopen.go edit.go rm.go resolve.go
      main_test.go
  internal/
    idea/                      # package logic (parsing, formatting, ID gen, file I/O, worktree resolution)
      idea.go
      idea_test.go
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

- Release pipeline that consumes this layout (build path, version stamping, Homebrew formula): `../release/pipeline.md`.
- Constitution principles III and IV: `fab/project/constitution.md`.
