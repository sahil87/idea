---
description: "Tag-driven release pipeline: release.sh cuts the semver tag; GitHub Actions cross-compiles, publishes the GitHub Release, and updates the Homebrew tap; shll.ai pulls the help-dump JSON on its own schedule"
---

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

The substituted formula is written to `/tmp/homebrew-tap/Formula/idea.rb`, committed as `idea v${version}` (author: `github-actions[bot]`), and pushed. This is the **final** step of the release workflow.

### shll.ai command reference: pull model (idea no longer pushes)

`idea`'s release **does not publish** its CLI help tree anywhere. The shll.ai landing site renders the command reference by **pulling** `idea`'s help on its own schedule: a scheduled job in `sahil87/shll.ai` `brew install`s `idea`, runs `idea help-dump`, and commits the captured JSON itself (also on-demand via `workflow_dispatch`). The previous release-side push (clone shll.ai, diff `help/idea.json`, open an auto-merged PR via a `sahil87` PAT) was removed once shll.ai's pull workflow went live — it raced/duplicated the pull and kept an unnecessary cross-repo write path and credential alive.

The producer — the hidden `help-dump` subcommand and the frozen JSON contract it emits — is **unchanged** and remains the single contract surface shll.ai pulls. See `../cli/structure.md`.

### shll.ai tool page: README + `docs/site/**` are also pulled and rendered

Beyond the `help-dump` JSON, shll.ai also **pulls and renders this repo's `README.md` and `docs/site/**` tree daily** to build the `idea` tool page (README → `/tools/idea/readme`; `docs/site/<page>.md` → `/tools/idea/<page>`, e.g. `install.md` → `/tools/idea/install`, `workflows.md` → `/tools/idea/workflows`). The pull is mechanical and verbatim — nothing is pushed or hand-fixed — so **this repo's file structure is the only thing that controls whether the page renders cleanly**. Editing-time gotchas a contributor will trip over:

- **External links in `README.md` and `docs/site/**` MUST be absolute `https://…`.** Relative links that leave the rendered set (e.g. to `docs/specs/`) are not rewritten and render as live 404s. Intra-`docs/site/` links use natural relative syntax (`[x](workflows.md)`) with no `..` escapes.
- **Reserved page slugs MUST NOT be used as `docs/site/` filenames**: `overview`, `readme`, `commands` (the `idea` row of the contract's per-tool table). `install`/`workflows` are allowed.
- **No mermaid fences and no `#gh-*-mode-only` theme fragments** in the rendered set (render as broken); any image must be an absolute `https://…` URL.

The authoritative rules live in shll.ai's README-extraction contract — do not duplicate them here, follow the source: <https://github.com/sahil87/shll.ai/blob/main/docs/specs/readme-extraction-contract.md> (per-tool table + §Producer conformance directive). Established by change `260608-3ra7`.

## Secrets

`HOMEBREW_TAP_TOKEN` must have `contents: write` permission on `sahil87/homebrew-tap`. The same token already powers the `hop` and `fab-kit` releases — there is one shared token across the maintainer's single-binary Go releases, set per-repo.

The release workflow declares `permissions: contents: write` for the in-repo GitHub Release creation step.

## Local install path (no release needed)

`./scripts/install.sh` builds the binary via `./scripts/build.sh` (which derives `VERSION` from `git describe --tags --always`) and copies it to `~/.local/bin/idea`. This is the supported way to run a development build without going through the tag/CI cycle. The justfile exposes this as `just local-install`.

## File index

- `scripts/release.sh` — tag cutter (local).
- `scripts/build.sh` — local cross-compile-free build with `git describe`-derived version stamp.
- `scripts/install.sh` — `build.sh` + copy to `~/.local/bin/idea`.
- `.github/workflows/release.yml` — CI release pipeline (cross-compile, GitHub Release, Homebrew tap update).
- `.github/formula-template.rb` — Homebrew formula with five sed placeholders.
- `justfile` — wrapper recipes (`build`, `local-install`, `test`, `release`).

## Design Decisions

- **260603-wtjc — retired the release-side help-dump push to shll.ai.** The release workflow used to walk the Cobra tree, write `help/idea.json`, and open an auto-merged PR into `sahil87/shll.ai` via a `sahil87` PAT (`SHLLAI_TOKEN`). That step was removed once shll.ai went to a pull model: shll.ai now `brew install`s `idea`, runs `idea help-dump`, and commits the JSON on its own schedule, so the push raced/duplicated the pull and kept an unnecessary cross-repo write path and credential alive. *Transport only* — the `help-dump` command and its JSON contract are unchanged (`schema_version` still `1`). **Follow-up (manual, not done by the PR):** delete the now-unused repo secret — `gh secret delete SHLLAI_TOKEN --repo sahil87/idea` — no workflow references it.
- **260608-3ra7 — conformed the repo to shll.ai's README-extraction contract.** Absolutized the README's external `docs/specs/` links (relative links 404 on the rendered page) and added a `docs/site/**` tree (`install.md`, `workflows.md`) that shll.ai pulls and renders at `/tools/idea/install` and `/tools/idea/workflows`. *Repo structure only* — no Go code, tests, CLI behavior, or CI changed. The durable editing constraints this imposes (absolute external links, no reserved page slugs, no mermaid/theme-fragments) are captured above under "shll.ai tool page: README + `docs/site/**` are also pulled and rendered"; the authoritative rules stay in the shll.ai contract, not duplicated here.

## Cross-references

- Pre-merge CI (the other GitHub Actions workflow — `ci.yml` runs gofmt/vet/test on PRs and push-to-`main`, distinct from this tag-driven release pipeline): `../ci/pipeline.md`.
- Source layout assumed by the build path (`./cmd/idea`) and version-stamp wiring (`-X main.version=...`): see `../cli/structure.md`.
- The hidden `help-dump` subcommand that shll.ai pulls, and the frozen JSON contract it emits: `../cli/structure.md`.
- shll.ai's README-extraction contract (the source of truth for how `README.md` + `docs/site/**` must be structured to render cleanly): <https://github.com/sahil87/shll.ai/blob/main/docs/specs/readme-extraction-contract.md>.
