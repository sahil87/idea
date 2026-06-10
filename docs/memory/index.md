# Memory Index

<!-- This index is maintained by /fab-continue (hydrate) when changes are completed. -->
<!-- Each domain gets a row linking to its memory files. -->

| Domain | Description | Memory Files |
|--------|-------------|------|
| ci | Pre-merge CI workflow (`ci.yml`): gofmt/vet/test on PRs and push-to-`main`, plus the `ci-gate` stable required status check — distinct from the tag-driven release pipeline | [ci/pipeline.md](ci/pipeline.md) |
| release | Tag-driven release pipeline (release.sh + GitHub Actions + Homebrew tap) and the shll.ai pull relationship (help-dump JSON + README/`docs/site/**` rendering) | [release/pipeline.md](release/pipeline.md) |
| cli | CLI source structure (cmd/idea + internal/idea + version wiring), the backlog line lifecycle (lenient-read / canonical-write parse-format-save contract), and per-subcommand notes | [cli/structure.md](cli/structure.md), [cli/update.md](cli/update.md) |
