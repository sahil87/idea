# Intake: Conform repo to shll.ai README-extraction contract

**Change**: 260608-3ra7-shll-readme-contract
**Created**: 2026-06-08
**Status**: Draft

## Origin

> Task: conform this repo to shll.ai's README-extraction contract.
>
> shll.ai (the toolkit landing page) renders your tool's page by mechanically pulling a slice of your README.md and your docs/site/** tree on a daily schedule — nothing is hand-copied, and you push nothing. Your job is to structure your repo so that pull renders cleanly.
>
> Read the contract and follow its §Producer conformance directive end-to-end: https://github.com/sahil87/shll.ai/blob/main/docs/specs/readme-extraction-contract.md
>
> 1. Find this repo's row in the directive's per-tool table (by repo name) for your slug and reserved page names.
> 2. Do Part 1 — restructure README.md: head order (# H1 → toolkit blockquote → badges → prose), drop GitHub-footer sections below the tail denylist, make all images absolute https://… URLs, render any mermaid to a committed image, and write any link that leaves the site as an absolute URL (relative links 404 on the site).
> 3. Do Part 2 (optional but encouraged) — add a docs/site/**/*.md tree for depth (install guide, deep-dives, etc.), following the four closed-set rules. Use docs/site/install.md / docs/site/workflows.md for those pages.
> 4. Run the directive's Verify checklist before opening the PR.
>
> Ship it as a single PR in this repo. Do not touch shll.ai — it already pulls and renders automatically.

One-shot invocation via `/fab-new ... && /fab-fff`. The contract was fetched and read end-to-end at intake time (raw markdown from the shll.ai repo), and the `idea` row of the per-tool table was located. No prior `/fab-discuss` session — decisions below were derived directly from the contract text plus inspection of the current README.

## Why

1. **Problem**: shll.ai renders the `idea` tool page at `/tools/idea/readme` (and any `/tools/idea/<page>`) by *pulling* a mechanically-deduced slice of this repo's `README.md` and `docs/site/**` tree on a daily schedule. The producer (this repo) is solely responsible for structuring its files so the pull renders cleanly — shll.ai copies verbatim and does not hand-fix anything. Two concrete conformance defects exist today:
   - The README contains **relative links that point outside the rendered set** (`docs/specs/overview.md`, `docs/specs/backlog-format.md`). Per contract §9.1.2, these are NOT rewritten by the site — they render as live **404s** with no warning.
   - There is **no `docs/site/**` tree**, so the tool page is README-only with no depth (no dedicated install or workflows pages at `/tools/idea/install`, `/tools/idea/workflows`).
2. **Consequence if not fixed**: The published `idea` page links visitors to dead 404s, and the page is shallower than peer tools that publish a `docs/site/` tree. Because the pull is daily and verbatim, the broken state ships and stays shipped until the repo is corrected.
3. **Why this approach**: The contract is a *producer conformance* spec — the only correct fix is to restructure our own files to its rules. There is no alternative (we cannot and must not touch shll.ai; it already pulls automatically). The work is deliberately minimal and surgical: the README head order and tail are already compliant, so the change is (a) absolutize the offending links and (b) add the optional-but-encouraged `docs/site/` tree.

## What Changes

### Per-tool table row (idea)

Located in the contract's §Producer conformance directive per-tool table:

| Repo | Slug | `content/` Path | URL Space | Reserved Slugs |
|------|------|-----------------|-----------|----------------|
| idea | `idea` | `content/idea/` | `/tools/idea/` | `overview`, `readme`, `commands` |

- **Slug**: `idea`. Pages render at `/tools/idea/<page>`.
- **Reserved page names** (MUST NOT be used as `docs/site/` filenames): `overview`, `readme`, `commands`.
- `install` and `workflows` are explicitly **NOT reserved** — they are tool-repo-owned and are the contract's recommended page names. We will use exactly those.

### Part 1 — README.md restructuring

Current README head order is **already compliant** and stays as-is:
1. `# idea` (single H1)
2. `> Part of [@sahil87's open source toolkit](https://shll.ai) — see all projects there.` (canonical toolkit blockquote — already byte-for-byte correct)
3. Contiguous badge line (3 shields.io badges)
4. First prose line ("Capture and manage ideas from the command line…")

No changes needed to the head. The following are the **only** README edits:

#### 1. Absolutize external relative links (contract §9.1.2 — the core fix)

Three locations reference repo-internal `docs/specs/` files via *relative* links. These point outside the rendered set and MUST become absolute `https://github.com/sahil87/idea/blob/main/…` URLs:

- **Line 91**: `see [`docs/specs/overview.md`](docs/specs/overview.md) for the full CLI reference and [`docs/specs/backlog-format.md`](docs/specs/backlog-format.md) for the file format contract.`
  → both targets become `https://github.com/sahil87/idea/blob/main/docs/specs/overview.md` and `https://github.com/sahil87/idea/blob/main/docs/specs/backlog-format.md`. Additionally, point the CLI-reference clause at the new in-site page `docs/site/commands` is **not allowed** (`commands` is reserved) — keep it pointing at the absolute spec URL. The "full CLI reference" prose may also gain a relative in-site link to `docs/site/workflows.md` where appropriate (see §workflows wiring below).
- **Line 118**: `any tool that follows [`backlog-format.md`](docs/specs/backlog-format.md) can read or write the file` → absolute URL.
- **Line 124**: `See [`backlog-format.md`](docs/specs/backlog-format.md) for the Shape A vs. Shape B distinction.` → absolute URL.

#### 2. Wire README → docs/site pages (contract §9.1.4, natural syntax)

Add relative links of the form `[text](docs/site/install.md)` / `[text](docs/site/workflows.md)` where they read naturally. The site rewrites `docs/site/<p>.md` → `/tools/idea/<p>` automatically. Candidate insertions:
- In the `## Install` section: a "Full install guide" pointer → `[install guide](docs/site/install.md)`.
- In the `## Integration with fab-kit` / worktree sections: a "deeper walkthrough" pointer → `[workflows](docs/site/workflows.md)`.

These are additive and improve depth; they are NOT a substitute for the absolutized external links.

#### 3. Tail denylist (contract §2)

The pull stops before the first denylisted `##`/`###` heading (Contributing, Development, Building, License, Acknowledgements). The current README has **none** of these sections — the last section is `## Gotchas`, which is kept content. **No deletions required.** (Verified: headings are Why idea?, Install, Shell completion, Quick Start, Command reference, Worktree-aware by default, Integration with fab-kit, Gotchas — all on the keep side.)

#### 4. Images / mermaid / theme fragments (contract §3, §4, §5)

- **Images**: README contains zero images (only shields.io badge links, which are already absolute `https://img.shields.io/…`). Nothing to absolutize.
- **Mermaid**: no mermaid fences present. Nothing to render.
- **Theme fragments**: no `#gh-dark-mode-only` / `#gh-light-mode-only` fragments present. Nothing to remove.

### Part 2 — docs/site/ tree (optional but encouraged)

Create two pages following the four closed-set rules (closure; external links absolute; all images absolute; natural link syntax):

#### `docs/site/install.md` → renders at `/tools/idea/install`

Depth on installation that the README only summarizes:
- Homebrew tap install (`brew install sahil87/tap/idea`) and what the tap provides.
- Manual build from a clean checkout (`just local-install`), prerequisites (Go, `just`), where the binary lands (`~/.local/bin/idea`), `$PATH` note.
- Shell completion setup (`idea shell-init <shell>`) for zsh/bash/fish/powershell, with the `eval "$(idea shell-init zsh)"` rc-file lines.
- Upgrading (`idea update`, self-upgrade via Homebrew; the `--skip-brew-update` behavior if relevant).
- Cross-link to the multi-tool `shll shell-install` (absolute URL, since it leaves the site): `https://github.com/sahil87/shll#shll-shell-install--wire-the-rc-file-recommended`.

#### `docs/site/workflows.md` → renders at `/tools/idea/workflows`

A deep-dive on the two behaviors the README only summarizes:
- **Worktree-aware resolution** — full explanation of default-current-worktree vs `--main`, the `--file` / `IDEAS_FILE` overrides, the resolution table, and *why* the default favors the current worktree.
- **fab-kit integration loop** — capture → triage → start work (`/fab-new <id>`) → close (`idea done <id>`), plus `fab batch new` parallel-queue usage.
- **Backlog format as public contract** — Shape A vs Shape B (extra-bracket pass-through lines), linking to the format spec via **absolute** URL (`https://github.com/sahil87/idea/blob/main/docs/specs/backlog-format.md`) since `docs/specs/` is outside the rendered set.

**Closed-set rule compliance for the tree:**
- Any image referenced (none planned) would be absolute https.
- Any link leaving the rendered set (e.g., to `docs/specs/`, to other repos, to shll.ai pages other than `/tools/idea/*`) is written as an absolute `https://…` URL by us.
- Intra-site links (install ↔ workflows) use natural relative syntax resolving inside `docs/site/` (e.g., `[install guide](install.md)`), with no `..` escapes.
- Neither file is named `overview`, `readme`, or `commands` (reserved). ✓

### Verify checklist (run before PR — contract §Verify)

- [ ] README top is `#` H1 → toolkit blockquote → badges, nothing (frontmatter/HTML/comments) above the H1.
- [ ] First prose line is the intended site intro.
- [ ] All README relative link/image targets either point into `docs/site/` OR are absolute `https://…`.
- [ ] All `docs/site/` intra-tree links resolve inside `docs/site/` (no `..`); all externals absolute.
- [ ] No `#gh-dark-mode-only` / `#gh-light-mode-only` fragments anywhere.
- [ ] No `docs/site/` page named `overview`, `readme`, or `commands`.
- [ ] `install` / `workflows` page names used (allowed).
- [ ] No denylisted footer section accidentally left above a keep section / none present.

## Affected Memory

This is a docs/repo-structure change with no spec-level behavior change to the `idea` CLI itself, so no `docs/memory/` updates are strictly required. Optionally, a reference note could capture the shll.ai contract relationship, but the contract lives in the shll.ai repo and is the authoritative source — duplicating it here would risk drift.

- `release/pipeline.md`: (no change) — the help-dump push wiring was already removed (commit c70ecda); this change does not touch CI.

## Impact

- **Files touched**: `README.md` (link edits only, ~3 lines), `docs/site/install.md` (new), `docs/site/workflows.md` (new).
- **No code changes** — no Go source, no tests, no build. `src/go/idea` and `scripts` untouched.
- **No CI changes** — shll.ai pulls; this repo pushes nothing (per commit c70ecda which removed the push wiring).
- **External dependency**: correctness is validated against the contract at https://github.com/sahil87/shll.ai/blob/main/docs/specs/readme-extraction-contract.md. The actual render is verified by shll.ai's daily pull (out of our control); our job ends at conformance.
- **Risk**: low. Worst case of a mistake is a broken link on the published tool page, self-correcting within ≤24h once fixed (per contract §7 divergence model).

## Open Questions

- None blocking. The contract is unambiguous on every point this change touches, and the per-tool table row for `idea` is explicit. Minor stylistic choice (exactly where to insert the additive README → docs/site links) is left to apply-time judgment and does not affect conformance.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Slug is `idea`; reserved page names are `overview`, `readme`, `commands`; URL space `/tools/idea/`. | Read directly from the contract's per-tool table, idea row. | S:98 R:98 A:98 D:98 |
| 2 | Certain | README head order (H1 → toolkit blockquote → badges → prose) is already compliant and requires no change. | Verified by reading the current README lines 1–7; blockquote is byte-for-byte the canonical text. | S:97 R:97 A:96 D:97 |
| 3 | Certain | No denylisted footer sections exist (Contributing/Development/Building/License/Acknowledgements), so no deletions. | Enumerated all README `##` headings; all are on the keep side of the tail rule. | S:96 R:96 A:95 D:96 |
| 4 | Certain | No images, mermaid, or theme fragments present, so §3/§4/§5 require no edits. | Grepped/read the README — only absolute shields.io badge URLs, no `mermaid` fences, no `#gh-*-mode-only`. | S:97 R:96 A:96 D:96 |
| 5 | Certain | The three relative `docs/specs/*.md` links (README lines 91, 118, 124) MUST become absolute `https://github.com/sahil87/idea/blob/main/docs/specs/*.md` URLs. | Contract §9.1.2: links outside the rendered set are not rewritten and 404; `docs/specs/` is outside the site. Repo confirmed as `sahil87/idea`, default branch `main`. | S:96 R:95 A:95 D:95 |
| 6 | Certain | Use `docs/site/install.md` and `docs/site/workflows.md` as the two depth pages. | Contract explicitly names these as the recommended, non-reserved page names; task instruction reiterates them verbatim. | S:96 R:95 A:96 D:95 |
| 7 | Certain | Within `docs/site/`, link to repo-internal `docs/specs/` via absolute URLs (not relative), because they are outside the rendered set. | Closed-set rule 2 + rule 1 (closure): a relative `../specs/...` would escape `docs/site/` and is forbidden; absolute https is the prescribed form. Directly entailed, not inferred. | S:95 R:95 A:94 D:94 |
| 8 | Certain | Additive README → docs/site links use natural relative syntax `[text](docs/site/<p>.md)`; intra-site links use `[text](<p>.md)`. | Contract §9.1.4 + closed-set rule 4 state this exactly: the site auto-rewrites these mounts; relative-within-tree is required for closure. | S:95 R:94 A:95 D:93 |
| 9 | Confident | No `docs/memory/` updates required — this is a repo-structure/docs change with no `idea` CLI behavior change. | Affected-memory rule: only list when spec-level behavior changes; it does not. A genuine judgment call, hence Confident not Certain. | S:86 R:88 A:86 D:84 |
| 10 | Tentative | Exact placement/wording of the additive README→docs/site pointers is left to apply-time. | Stylistic; does not affect conformance. Apply will insert where it reads naturally (Install and fab-kit/worktree sections). | S:70 R:68 A:72 D:66 |

10 assumptions (8 certain, 1 confident, 1 tentative, 0 unresolved).
