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

**Help-dump → shll.ai command reference.** The final step (ordered last, after the GitHub Release and Homebrew tap update) publishes `idea`'s CLI help tree to the shll.ai landing site, which renders it as an expandable "Command reference" on the tool page. It is **best-effort** — placed last so a failure here cannot abort the release. The step:

1. Runs `dist/idea-linux-amd64/idea help-dump > help/idea.json` — the native `linux/amd64` runner binary, which carries the real ldflags version stamp from the Cross-compile step (a `go run`/`dev` build would emit `version: dev`). The dump is the hidden `help-dump` subcommand in the binary; its JSON contract is documented in `../cli/structure.md`.
2. Validates the file parses as JSON locally (`python3 -c "import json; json.load(...)"`), failing the step early if malformed.
3. Clones `sahil87/shll.ai` over HTTPS using `SHLLAI_TOKEN`, copies in **only** `help/idea.json`, and — when it differs from the committed version — commits on a per-version branch `help-dump/idea-${version}` and opens a PR via `gh pr create`.

The step deliberately does **NOT** call `gh pr merge`. shll.ai owns merging through its own `.github/workflows/help-automerge.yml`, which enables auto-merge only when three guards pass: **actor** (PR author is the trusted identity `sahil87`), **content** (every changed file is under `help/`), and **schema** (the JSON passes shll.ai's `validate-help.mjs` Zod contract). idea's side only opens the PR. (shll.ai's repo `allow_auto_merge` is `false` at the native-feature level, so a `gh pr merge --auto` from here would fail — the receiving workflow does the merge.)

Three constraints in the step exist to satisfy those guards:

- **Authorship** — the commit is configured as `sahil87`/`sahil@noon.design`, not `github-actions[bot]`. The shll.ai actor guard (`TRUSTED_AUTHOR: sahil87`) refuses to auto-merge any other author. This is why `SHLLAI_TOKEN` must be a `sahil87` PAT, not a generic bot token — the PAT both authenticates the clone/push and attributes `gh` authorship to `sahil87`.
- **Content** — `git add` / commit stage **only** `help/idea.json`. A mixed diff trips the content guard and is left for manual review.
- **No-op guard** — when `help/idea.json` is unchanged (no flag/command churn since the last release), `git diff --quiet` short-circuits the step to `exit 0` with no PR, avoiding empty-PR spam.

**Why PR, not direct push.** This is one slice of a 7-tool rollout whose tools all write to shll.ai. Direct pushes to `main` race on non-fast-forward; routing through a PR + the receiving auto-merge workflow serializes integration. The site-side consumer (Astro loader + reference UI, Zod schema, `help-automerge.yml`) lives in the shll.ai repo and is out of scope for this repo.

## Secrets

`HOMEBREW_TAP_TOKEN` must have `contents: write` permission on `sahil87/homebrew-tap`. The same token already powers the `hop` and `fab-kit` releases — there is one shared token across the maintainer's single-binary Go releases, set per-repo.

`SHLLAI_TOKEN` is a **`sahil87` Personal Access Token** with `contents` + `pull-requests` write on `sahil87/shll.ai`, used by the help-dump step. Two things make it specifically a PAT (not the in-repo `GITHUB_TOKEN` or a bot token): (1) it must authenticate against a *different* repo (shll.ai), and (2) its `sahil87` identity is load-bearing — the step commits as `sahil87` and `gh pr create` attributes the PR to that identity, which is required to pass shll.ai's `help-automerge.yml` actor guard (`TRUSTED_AUTHOR: sahil87`). A `github-actions[bot]`-authored PR would not auto-merge. It is exported as `GH_TOKEN` so both `gh` and the clone URL authenticate against shll.ai.

The release workflow itself also declares `permissions: contents: write` for the in-repo GitHub Release creation step. The shll.ai write does not use `GITHUB_TOKEN`, so no in-repo permission change was needed to add the help-dump step.

## Local install path (no release needed)

`./scripts/install.sh` builds the binary via `./scripts/build.sh` (which derives `VERSION` from `git describe --tags --always`) and copies it to `~/.local/bin/idea`. This is the supported way to run a development build without going through the tag/CI cycle. The justfile exposes this as `just local-install`.

## File index

- `scripts/release.sh` — tag cutter (local).
- `scripts/build.sh` — local cross-compile-free build with `git describe`-derived version stamp.
- `scripts/install.sh` — `build.sh` + copy to `~/.local/bin/idea`.
- `.github/workflows/release.yml` — CI release pipeline (cross-compile, GitHub Release, Homebrew tap update, help-dump PR to shll.ai).
- `.github/formula-template.rb` — Homebrew formula with five sed placeholders.
- `justfile` — wrapper recipes (`build`, `local-install`, `test`, `release`).

## Cross-references

- Source layout assumed by the build path (`./cmd/idea`) and version-stamp wiring (`-X main.version=...`): see `../cli/structure.md`.
- The hidden `help-dump` subcommand the help-dump step invokes, and the frozen JSON contract it emits: `../cli/structure.md`.
