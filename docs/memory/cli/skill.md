---
description: "The idea skill subcommand (toolkit skill standard): a visible cobra command printing an embedded, static, byte-identical copy of docs/site/skill.md to stdout (raw markdown, no framing, exit 0), plus the sync+drift-guard mechanism reused from shll — canonical → committed copy src/cmd/idea/skill/skill.md via scripts/sync-skill.sh, the repo's first //go:embed, kept honest by a go-test drift guard and a 150-line budget guard"
type: memory
---

# idea skill Subcommand

**Domain**: cli

## Overview

`idea skill` prints a static, agent-facing usage bundle to stdout. It adopts the shll toolkit's `skill` standard (principle №10): an agent operating an *installed* `idea` binary has no offline usage briefing — `-h` is flag reference, README/`docs/site` needs a checkout or a shll.ai round-trip, and `fab/project` is contributor-scoped. The `skill` bundle is embedded in the binary, so it ships wherever the tool ships and is version-locked to it by construction. `idea` is the first repo in the toolkit to adopt the standard — there was a mechanism precedent in `shll standards`, but no `skill`-bundle content precedent to copy from (260717-3q43-adopt-toolkit-skill-standard).

## Requirements

### Requirement: Static, byte-identical usage bundle to stdout
`idea skill` MUST write the embedded bundle bytes **verbatim** to `cmd.OutOrStdout()` — raw markdown, no rendering, no pager, no added framing. On success stderr MUST be empty and the exit code MUST be `0`. The bundle MUST be **static** — no timestamps, environment lookups, or session state (guaranteed by construction: it reads embedded bytes, does no env lookup). The command MUST take `cobra.NoArgs`, define no flags of its own, and have no `--json` (stdout IS the raw markdown bytes).

#### Scenario: Agent reads the bundle offline
- **GIVEN** an installed `idea` binary with no repo checkout
- **WHEN** `idea skill` runs
- **THEN** stdout is the embedded bundle bytes verbatim, stderr is empty, exit code is 0
- **AND** `idea skill <extra-arg>` errors under `cobra.NoArgs`, and `skill` appears in `idea -h`

### Requirement: Bundle is a bounded usage briefing, not a README clone or flag table
The canonical bundle at `docs/site/skill.md` MUST be raw Markdown, **≤150 lines** (the standard's hard budget, principle №9), in the usage-briefing genre. It MUST cover: when-to-use (and when not); a capabilities map keyed to each user-facing subcommand; composition with fab-kit via the shared backlog line format; the output/exit-code contracts documenting idea's **actual** behavior; and gotchas. (As implemented, the bundle is 97 lines.) It MUST document idea's actual exit-code behavior — the toolkit `0`/`1`/`2` convention: `0` success, `1` operational failure, `2` usage error (260717-xvsj) — never an aspirational or outdated contract, so it never lies to an agent branching on exit codes.

#### Scenario: Bundle grows past budget
- **GIVEN** an edit that pushes `docs/site/skill.md` over 150 lines
- **WHEN** `go test ./...` runs
- **THEN** the budget-guard test fails, reporting the actual line count against the 150 limit

### Requirement: Embedded copy stays byte-identical to canonical (drift guard)
`skill.go` MUST embed the committed copy `src/cmd/idea/skill/skill.md` via `//go:embed skill/skill.md` into an `embed.FS`. That committed copy MUST be byte-identical to the canonical `docs/site/skill.md`. A drift-guard test MUST assert the two are equal on every `go test ./...`; on mismatch it MUST fail naming the drifted file and the fix (`run scripts/sync-skill.sh and commit the refreshed copy`).

#### Scenario: Canonical edited without re-syncing
- **GIVEN** an edit to `docs/site/skill.md` with no re-run of `scripts/sync-skill.sh`
- **WHEN** `go test ./...` runs (locally or in the existing CI PR workflow)
- **THEN** the drift-guard test `TestSkillEmbedMatchesCanonical` fails, naming the drifted file and the remedial command

## Design Decisions

### Reuse shll's sync + drift-guard mechanism verbatim
**Decision**: The canonical bundle lives at `docs/site/skill.md`; `scripts/sync-skill.sh` copies it to the committed embed copy `src/cmd/idea/skill/skill.md`; `skill.go` embeds that copy via `//go:embed`; and `skill_test.go`'s byte-equality drift guard keeps the two honest on every `go test`.
**Why**: The standard names this exact mechanism ("This is the exact mechanism `shll standards` established; reuse it"). The Go module root is `src/` and `docs/site/` sits **above** it, so `//go:embed` cannot reach the canonical file directly — the sync-to-a-committed-copy step bridges that reachability gap. The committed copy means a clean `go build ./...` (which does not run the sync script) compiles.
**Rejected**: Embedding `docs/site/skill.md` directly — impossible, it is outside the module root (the same constraint shll solved this way).
*Introduced by*: `260717-3q43-adopt-toolkit-skill-standard`

### Visible command, not `Hidden: true`
**Decision**: Register `skill` as a first-class **visible** subcommand (contrast the hidden `help-dump`).
**Why**: The standard treats `skill` as a uniform first-class subcommand that agents discover via `idea -h`. `help-dump` is hidden only because it is machine plumbing for shll.ai; `skill` is agent-facing, not plumbing. Because it is visible, its `Short`/`Long` flow into the help-dump JSON automatically (no schema change) — see the help-dump node in [structure](/cli/structure.md).
**Rejected**: Hidden like `help-dump` (a one-line flip if wrong).
*Introduced by*: `260717-3q43-adopt-toolkit-skill-standard`

### Buffer-driven `runSkill(out io.Writer) error` seam
**Decision**: Extract the write logic from the cobra factory into `runSkill(out io.Writer) error`, and name the embed path in a `skillEmbedPath` const rather than repeating the string literal.
**Why**: Mirrors shll's `runStandards` seam and idea's own in-process test style (`help_dump_test.go` drives via `newRootCmd()`). Tests drive the seam with a `bytes.Buffer` — no subprocess needed since the command reads embedded bytes only. The const single-sources the read path (no magic string — code-quality Anti-Patterns). A missing embed file is treated as a build-integrity bug (wrapped error), not user error — the sync step / drift guard should have caught it.
**Rejected**: A subprocess test (unnecessary; the bytes are static).
*Introduced by*: `260717-3q43-adopt-toolkit-skill-standard`

### Enforce the ≤150-line budget in the same test file
**Decision**: `skill_test.go` additionally pins the bundle at ≤150 lines via `TestSkill_LineBudget` (the `maxSkillLines` const).
**Why**: The standard states a hard budget with teeth; enforcing a stated hard rule in the same test file is a cheap, reversible extension. shll's `standards` docs carry no budget, so there was no test precedent — this is an idea-local addition on top of the reused mechanism.
*Introduced by*: `260717-3q43-adopt-toolkit-skill-standard`

## Cross-references

- Registration in `newRootCmd()`'s `root.AddCommand(...)` list, and the repo's first `//go:embed` use: [structure](/cli/structure.md).
- shll.ai pulls `docs/site/**`, so `docs/site/skill.md` renders at `https://shll.ai/idea/skill`: [pipeline](/release/pipeline.md).
- The toolkit-standards conformance article and the `[3q43]`/`[xvsj]` deferrals recorded during the first audit: [structure](/cli/structure.md) § Toolkit-standards conformance.
- Constitution principles III (Cobra-Idiomatic CLI Surface), IV (this command has no business logic — embed + print lives acceptably in `cmd/`, like `help_dump.go`), and Dependency Discipline (`embed` is stdlib): `fab/project/constitution.md`.
