# Plan: Conform repo to shll.ai README-extraction contract

**Change**: 260608-3ra7-shll-readme-contract
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

### Docs: README external-link absolutization (contract §9.1.2)

#### R1: Relative `docs/specs/` links MUST be absolute https URLs
All README links that target `docs/specs/` files (outside the rendered site set) SHALL be written as absolute `https://github.com/sahil87/idea/blob/main/docs/specs/<file>` URLs. The visible link text (the backticked path) SHALL remain unchanged. Three locations are affected: line ~91 (two links: `overview.md`, `backlog-format.md`), line ~118 (`backlog-format.md`), line ~124 (`backlog-format.md`).

- **GIVEN** the README references `docs/specs/overview.md` and `docs/specs/backlog-format.md` via relative link targets
- **WHEN** the change is applied
- **THEN** every such link target is the absolute `https://github.com/sahil87/idea/blob/main/docs/specs/<file>` URL
- **AND** the visible backticked link text is unchanged
- **AND** `grep -nE '\]\(docs/specs/' README.md` returns nothing

### Docs: README → docs/site wiring (contract §9.1.4, natural syntax)

#### R2: README SHALL link into the new docs/site pages using natural relative syntax
The README SHALL gain one additive pointer to `docs/site/install.md` in the `## Install` section and one additive pointer to `docs/site/workflows.md` in the worktree / `## Integration with fab-kit` area. These links use natural relative syntax (`docs/site/<p>.md`) that the site auto-rewrites to `/tools/idea/<p>`. Additions MUST be tasteful (one each) and MUST NOT remove or restructure existing prose.

- **GIVEN** the README has `## Install` and `## Integration with fab-kit` sections
- **WHEN** the change is applied
- **THEN** the Install section contains a `[install guide](docs/site/install.md)` link and the worktree/fab-kit area contains a `[workflows](docs/site/workflows.md)` link
- **AND** no existing prose lines are removed or restructured

### Docs: README head and tail integrity (contract §1, §2, §3-5)

#### R3: README head order, tail, and media MUST remain compliant and unchanged
The README head (line 1 `# idea`, line 3 toolkit blockquote, line 5 badge line, then prose) SHALL remain byte-for-byte unchanged. No denylisted footer section SHALL be introduced. No images, mermaid fences, or `#gh-*-mode-only` theme fragments SHALL exist in the README.

- **GIVEN** the README head is already contract-compliant
- **WHEN** the change is applied
- **THEN** line 1 is `# idea`, line 3 is the toolkit blockquote, line 5 is the badge line, all unchanged
- **AND** no `gh-dark-mode-only` / `gh-light-mode-only` fragments exist in the README

### Docs: docs/site install page (contract Part 2, closed-set rules)

#### R4: `docs/site/install.md` SHALL exist with the prescribed install depth
A new file `docs/site/install.md` SHALL be created, starting with a single `# Install` H1. It SHALL cover: Homebrew tap install and what the tap provides; manual build from a clean checkout (`just local-install`, prereqs Go + `just`, binary lands at `~/.local/bin/idea`, `$PATH` note); shell completion (`idea shell-init <shell>` for zsh/bash/fish/powershell with `eval "$(idea shell-init zsh)"` rc lines); upgrading (`idea update` self-upgrade via Homebrew). It SHALL cross-link to the multi-tool `shll shell-install` via the absolute URL `https://github.com/sahil87/shll#shll-shell-install--wire-the-rc-file-recommended`, and SHALL contain a natural intra-site link to `[workflows](workflows.md)`. It SHALL contain no images. It SHALL NOT be named `overview`, `readme`, or `commands`.

- **GIVEN** there is no docs/site tree today
- **WHEN** the change is applied
- **THEN** `docs/site/install.md` exists, starts with `# Install`, and covers Homebrew, manual build, shell completion, and upgrading
- **AND** it cross-links to `shll shell-install` as an absolute https URL and links to `workflows.md` via natural relative syntax
- **AND** it contains no images and no relative link escapes `docs/site/`

### Docs: docs/site workflows page (contract Part 2, closed-set rules)

#### R5: `docs/site/workflows.md` SHALL exist with the prescribed workflow deep-dives
A new file `docs/site/workflows.md` SHALL be created, starting with a single `# Workflows` H1. It SHALL cover: (a) worktree-aware resolution deep-dive — default current-worktree vs `--main`, `--file`/`IDEAS_FILE` overrides, the resolution table, and why the default favors the current worktree; (b) the fab-kit integration loop — capture → triage (`idea list`) → start work (`/fab-new <id>`) → close (`idea done <id>`), plus `fab batch new` parallel queue; (c) backlog format as public contract — Shape A vs Shape B pass-through lines, linking to the format spec via the absolute URL `https://github.com/sahil87/idea/blob/main/docs/specs/backlog-format.md`. It SHALL contain a natural intra-site link to `[install guide](install.md)`. It SHALL contain no images. It SHALL NOT be named `overview`, `readme`, or `commands`.

- **GIVEN** there is no docs/site tree today
- **WHEN** the change is applied
- **THEN** `docs/site/workflows.md` exists, starts with `# Workflows`, and covers worktree resolution, the fab-kit loop, and Shape A/B backlog format
- **AND** the backlog-format spec link is an absolute https URL and the install link uses natural relative syntax
- **AND** it contains no images and no relative link escapes `docs/site/`

### Docs: closed-set closure verification (contract Verify checklist)

#### R6: All docs/site relative links MUST resolve inside docs/site/ and the Verify checklist MUST pass
Every relative link inside `docs/site/` SHALL resolve inside `docs/site/` (no `..` escapes); any link leaving the rendered set SHALL be absolute https. The contract's Verify greps MUST pass.

- **GIVEN** the docs/site pages are created
- **WHEN** the Verify greps run
- **THEN** `grep -nE '\]\(\.\./' docs/site/*.md` returns nothing
- **AND** `grep -rn 'gh-dark-mode-only\|gh-light-mode-only' README.md docs/site/` returns nothing
- **AND** no docs/site file is named `overview`, `readme`, or `commands`

### Non-Goals

- No Go source, test, build, or CI changes — `src/`, `scripts/`, and `.github/` are untouched.
- No `docs/specs/` content edits — only how the README/site link to them.
- No restructuring of existing README prose beyond the two additive site links and the three link-target absolutizations.

### Design Decisions

1. **Absolutize rather than relocate `docs/specs/` content**: Keep specs where they are; only rewrite README link targets to absolute URLs — *Why*: `docs/specs/` is deliberately outside the rendered site set; the contract's prescribed fix for outside-set links is absolute https — *Rejected*: moving spec content into `docs/site/` (would duplicate/relocate the authoritative spec and risk drift).
2. **Use `install`/`workflows` as the two depth pages**: *Why*: contract names them as the recommended, non-reserved page names; `overview`/`readme`/`commands` are reserved — *Rejected*: any reserved name.
3. **Ground site-page content in the existing README + specs**, expanding without inventing flags — *Why*: constitution treats output/format as a public contract; specs are source of truth — *Rejected*: inventing new install paths or flags.

## Tasks

### Phase 2: Core Implementation

- [x] T001 Absolutize the three `docs/specs/` relative link targets in `README.md` (line ~91 two links, line ~118, line ~124) to `https://github.com/sahil87/idea/blob/main/docs/specs/<file>`, keeping visible backticked text unchanged <!-- R1 -->
- [x] T002 [P] Create `docs/site/install.md` (H1 `# Install`) covering Homebrew tap, manual `just local-install` build, shell completion, and upgrading; cross-link `shll shell-install` as absolute https; intra-site link to `[workflows](workflows.md)` <!-- R4 -->
- [x] T003 [P] Create `docs/site/workflows.md` (H1 `# Workflows`) covering worktree-aware resolution, the fab-kit integration loop, and Shape A/B backlog format; link the format spec as absolute https; intra-site link to `[install guide](install.md)` <!-- R5 -->
- [x] T004 Add additive README → docs/site links: `[install guide](docs/site/install.md)` in `## Install`, `[workflows](docs/site/workflows.md)` in the worktree/`## Integration with fab-kit` area, without removing or restructuring existing prose <!-- R2 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Verify README head/tail/media integrity: line 1 `# idea`, line 3 toolkit blockquote, line 5 badges unchanged; no denylisted footer section; no theme fragments <!-- R3 -->
- [x] T006 Run the contract Verify greps: `\]\(docs/specs/` in README.md returns nothing; `\]\(\.\./` in docs/site/*.md returns nothing; `gh-dark-mode-only`/`gh-light-mode-only` in README.md + docs/site/ returns nothing; no docs/site file named overview/readme/commands; optional `go build ./...` sanity no-op <!-- R6 -->

## Execution Order

- T001 and T004 both edit README.md — run T001 before T004 to avoid overlapping edits.
- T002 and T003 are independent new files (`[P]`), but each references the other's filename for intra-site links; create both before running T006 closure verification.
- T005 and T006 are verification gates — run after T001–T004.

## Acceptance

### Functional Completeness

- [x] A-001 R1: All three README `docs/specs/` link targets are absolute `https://github.com/sahil87/idea/blob/main/docs/specs/<file>` URLs with unchanged visible text
- [x] A-002 R2: README `## Install` contains `[install guide](docs/site/install.md)` and the worktree/fab-kit area contains `[workflows](docs/site/workflows.md)`, with no existing prose removed
- [x] A-003 R3: README line 1 is `# idea`, line 3 is the toolkit blockquote, line 5 is the badge line — all unchanged; no denylisted footer section added
- [x] A-004 R4: `docs/site/install.md` exists, starts with `# Install`, and covers Homebrew tap, manual build, shell completion, and upgrading
- [x] A-005 R5: `docs/site/workflows.md` exists, starts with `# Workflows`, and covers worktree resolution, the fab-kit loop, and Shape A/B backlog format

### Behavioral Correctness

- [x] A-006 R4: `docs/site/install.md` cross-links `shll shell-install` via the exact absolute URL and links `[workflows](workflows.md)` via natural relative syntax
- [x] A-007 R5: `docs/site/workflows.md` links the backlog-format spec via the absolute URL and links `[install guide](install.md)` via natural relative syntax

### Scenario Coverage

- [x] A-008 R1: `grep -nE '\]\(docs/specs/' README.md` returns nothing
- [x] A-009 R6: `grep -nE '\]\(\.\./' docs/site/*.md` returns nothing
- [x] A-010 R6: `grep -rn 'gh-dark-mode-only\|gh-light-mode-only' README.md docs/site/` returns nothing

### Edge Cases & Error Handling

- [x] A-011 R6: No `docs/site/` file is named `overview`, `readme`, or `commands`; `install` and `workflows` are used
- [x] A-012 R4: `docs/site/install.md` contains no images; all external links are absolute https and no relative link escapes `docs/site/`
- [x] A-013 R5: `docs/site/workflows.md` contains no images; all external links are absolute https and no relative link escapes `docs/site/`

### Code Quality

- [x] A-014 Pattern consistency: New docs pages follow the README's voice/structure and Markdown conventions
- [x] A-015 No unnecessary duplication: Site pages expand on (not verbatim-copy) README/spec content; the authoritative specs are linked, not duplicated

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Absolute URL base is `https://github.com/sahil87/idea/blob/main/` for repo-internal files. | Repo is `sahil87/idea`, default branch `main`, stated in intake assumption 5 and task brief. | S:97 R:96 A:96 D:96 |
| 2 | Certain | Use `just local-install` (not `./scripts/install.sh`) as the documented manual-build path on the install page. | Verified the justfile: `local-install` recipe runs `./scripts/install.sh`; the README and intake both use the `just` form as the public-facing path. overview.md's raw script path is the internal implementation. | S:92 R:90 A:90 D:88 |
| 3 | Certain | shell-init advertises zsh/bash/fish/powershell; document all four per intake. | Verified `src/cmd/idea/shell_init.go`: switch handles zsh, bash, fish, powershell; Long help states "Supported shells: zsh, bash, fish, powershell." | S:95 R:93 A:94 D:93 |
| 4 | Confident | Place the README `[workflows]` pointer at the end of the `## Integration with fab-kit` section (after the public-contract paragraph), and the `[install guide]` pointer at the end of `## Install`. | Intake assumption 10 leaves exact placement to apply-time; these spots read most naturally and keep additions tasteful. | S:80 R:82 A:80 D:76 |
| 5 | Confident | Do not document `--skip-brew-update` prominently on the install page (mention update self-upgrade only). | Intake says "the `--skip-brew-update` behavior if relevant"; it is a niche performance flag, so a brief mention suffices rather than a dedicated subsection. | S:78 R:84 A:80 D:75 |

5 assumptions (3 certain, 2 confident, 0 tentative).
