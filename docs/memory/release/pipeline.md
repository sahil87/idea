# Release Pipeline

`idea` is released independently via a tag-driven pipeline: a local script cuts and pushes a semver tag, GitHub Actions takes over from the tag push to cross-compile, publish a GitHub Release, and update the Homebrew tap formula.

## Flow

### 1. Local: `scripts/release.sh <patch|minor|major>`

The release script is the only step a maintainer runs by hand. It:

- Validates the working tree is clean (refuses to release with uncommitted changes).
- Validates the current HEAD is on a named branch (refuses detached HEAD).
- Reads the latest tag via `git describe --tags --abbrev=0` (defaulting to `v0.0.0` if none exists).
- Computes the next semver from the bump type argument (`patch`, `minor`, or `major`).
- Creates the new tag locally and pushes it to `origin`.

The script does NOT modify any tracked files — no VERSION file, no commit. The git tag itself is the version source of truth. This is what makes the pipeline tag-driven: pushing the tag is the entire trigger.

`just release [bump]` is the wrapper recipe; it forwards to `scripts/release.sh` with `patch` as the default bump.

### 2. CI: `.github/workflows/release.yml`

Triggered by `tags: ["v*"]`. The workflow extracts the version from the tag (`refs/tags/vX.Y.Z` → `X.Y.Z`), then runs the following stages on `ubuntu-latest`:

**Cross-compile matrix.** A single shell loop builds for four targets:

- `darwin/arm64`
- `darwin/amd64`
- `linux/arm64`
- `linux/amd64`

For each target, it runs `CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -ldflags "-X main.version=$tag" -o dist/idea-$os-$arch/idea ./cmd/idea` (working directory: `src/`), then tars the result into `dist/idea-{os}-{arch}.tar.gz`. Build path `./cmd/idea` matches the source layout described in `cli/structure.md`. Version stamping flows through `-ldflags` into `main.version` (declared in `cmd/idea/main.go`).

**GitHub Release.** Uses `softprops/action-gh-release` (pinned by SHA) to create the release with all four `.tar.gz` artifacts attached and auto-generated release notes. For minor releases (patch component is `0`), a "release notes base tag" detection step finds the earliest tag of the previous minor version (`vMAJOR.MINOR-1.*`) and passes it as `previous_tag` so the auto-generated notes span the whole minor cycle, not just the patch range.

**Homebrew tap update.** Final step reads the four tarball SHA256s, clones `sahil87/homebrew-tap` (over HTTPS using `HOMEBREW_TAP_TOKEN`), and runs `sed` over `.github/formula-template.rb` to substitute five placeholders:

- `VERSION_PLACEHOLDER` → version string (no `v` prefix)
- `SHA_DARWIN_ARM64`, `SHA_DARWIN_AMD64`, `SHA_LINUX_ARM64`, `SHA_LINUX_AMD64` → tarball SHA256s

The substituted formula is written to `/tmp/homebrew-tap/Formula/idea.rb`, committed as `idea v${version}` (author: `github-actions[bot]`), and pushed.

## Secrets

`HOMEBREW_TAP_TOKEN` is the only repo secret the pipeline needs. It must have `contents: write` permission on `sahil87/homebrew-tap`. The same token already powers the `hop` and `fab-kit` releases — there is one shared token across the maintainer's single-binary Go releases, set per-repo.

The release workflow itself also declares `permissions: contents: write` for the in-repo GitHub Release creation step.

## Local install path (no release needed)

`./scripts/install.sh` builds the binary via `./scripts/build.sh` (which derives `VERSION` from `git describe --tags --always`) and copies it to `~/.local/bin/idea`. This is the supported way to run a development build without going through the tag/CI cycle. The justfile exposes this as `just local-install`.

## File index

- `scripts/release.sh` — tag cutter (local).
- `scripts/build.sh` — local cross-compile-free build with `git describe`-derived version stamp.
- `scripts/install.sh` — `build.sh` + copy to `~/.local/bin/idea`.
- `.github/workflows/release.yml` — CI release pipeline (cross-compile, GitHub Release, Homebrew tap update).
- `.github/formula-template.rb` — Homebrew formula with five sed placeholders.
- `justfile` — wrapper recipes (`build`, `local-install`, `test`, `release`).

## Cross-references

- Source layout assumed by the build path (`./cmd/idea`) and version-stamp wiring (`-X main.version=...`): see `cli/structure.md`.
