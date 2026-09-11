# Intake: HexoKit Banner Sweep

**Change**: 260911-jmkb-hexokit-banner-sweep
**Created**: 2026-09-11

## Origin

One-shot `/fab-new` invocation, fully specified by the cross-repo plan doc. Raw input:

> Per fab/plans/sahil/26-09-10-hexokit-rebrand.md row C7 (hexokit-banner-sweep), applied to the idea repo, gated on C1 (shll change ttoa, PR shll#98 up, review-pr done): apply the readme-extraction standard's revised mandated blockquote to this repo's README (-> "Part of [HexoKit](https://hexokit.com) -- see all projects there"); flip any PRESENT-TENSE "run-kit" PRODUCT mentions (prose referring to the dashboard product) to HexoKit. The `rk`/`run-kit` SUBSTRATE (binary name, verbs, options) is UNTOUCHED. Repo links stay as-is until R2. Read the plan doc's Decision log (D1-D14) and row C7 first; run `shll standards` and read `readme-extraction` before editing.

**Plan context read at intake time.** The plan lives in the run-kit repo at `fab/plans/sahil/26-09-10-hexokit-rebrand.md`. The binding decisions for this row:

- **D1** (Confirmed): HexoKit names the dashboard product; the family is "the HexoKit toolkit"; companions (fab-kit, wt, **idea**, tu, hop, `shll`) keep their names.
- **D2** (Confirmed): the `rk` binary and every `RK_*` / `@rk_*` / `rk-*` identifier are untouched — substrate tier, never renamed.
- **D11** (Proposed, treated as binding for scope): historical text (`fab/changes/`, `fab/plans/`, `docs/memory/` narrative, git history) is not renamed; only live surfaces change.
- **D14** (Confirmed): the standards are not renamed. C1 edits the `readme-extraction` mandated blockquote to `> Part of [HexoKit](https://hexokit.com) — see all projects there.`; the remaining `shll.ai` / "shll toolkit" mentions in the standards wait for **X4** (Phase 2, after shll.ai stops being the consuming site at X2).
- **Row C7** scope (Phase 1, gated on C1, size S per repo): "README banner via the standard; present-tense 'run-kit' product mentions → HexoKit (`rk` verbs untouched) … Repo links stay `sahil87/run-kit` until R2."

**Gate verified at intake time (2026-09-11).** C1 is shll fab change `ttoa`, PR [shll#98](https://github.com/sahil87/shll/pull/98) (`260911-ttoa-hexokit-banner-and-policy`) — state OPEN, draft, pipeline through review-pr done per the invoking prompt. Its diff to `docs/site/standards/readme-extraction.md` § README structure rule 1 replaces the fenced canonical line:

```diff
-> Part of the [shll toolkit](https://shll.ai) — see all projects there.
+> Part of [HexoKit](https://hexokit.com) — see all projects there.
```

The **installed** `shll` binary (`shll standards readme-extraction`) still serves the pre-C1 line — C1 is not yet merged/released. The revised line is therefore taken **from the PR diff, not from the binary**. Note the character: the standard uses an em-dash (`—`, U+2014), not the `--` the prompt typed as ASCII shorthand.

**Scope grounding performed at intake time.** A repo-wide grep (`run-kit|runkit|run kit|shll\.ai|hexokit`, case-insensitive, excluding `.git/`, `fab/`, `.agents/`, `.claude/`) enumerated every live-surface occurrence. Result: **zero** `run-kit` / `runkit` / `RunKit` occurrences anywhere in this repo's live surfaces (README, `docs/site/**`, `docs/specs/**`, `src/**`, `scripts/**`) — consistent with the plan's own evidence line ("idea references run-kit 0×"). The only brand-tier hit is the README blockquote on line 3. The complete edit list is in § What Changes.

## Why

1. **Problem.** The toolkit's README blockquote is a byte-exact cross-repo contract — the `readme-extraction` standard says "this exact line in all seven repos" — and C1 (shll#98) revises that line to name HexoKit. As of C1, this repo's line 3 (`> Part of the [shll toolkit](https://shll.ai) — see all projects there.`) is the superseded wording. The blockquote is the first line under the H1 on the repo page and the most visible cross-repo brand surface (D14 rationale); leaving it reintroduces the two-brand split the rebrand exists to remove.
2. **Consequence if unfixed.** The constitution's § Toolkit Standards article binds this repo to revised standards "without further amendment", so once C1 ships the repo is out of conformance on its public README. Phase 2's X1 (site cutover prep) depends on C3 and C7 being merged, so an un-swept satellite blocks the cutover order.
3. **Approach.** Pure markdown conformance edit, no behavior change, following the exact precedent of `260718-92gj-shll-toolkit-rename-conformance` (the previous blockquote flip in this repo). Phase-1 scope discipline is deliberate: only the banner changes now; `shll.ai` URLs and "shll toolkit" family-name prose stay until shll.ai stops being the consuming site (X2/X4), so the README never points a reader at a domain whose install endpoint is not yet product-first.

## What Changes

### 1. README.md — the toolkit blockquote (line 3)

One line edit, byte-exact:

- Before: `> Part of the [shll toolkit](https://shll.ai) — see all projects there.`
- After: `> Part of [HexoKit](https://hexokit.com) — see all projects there.`

Note the wording drops the article ("Part of the …" → "Part of …") and the link text is `HexoKit`, not "HexoKit toolkit". Em-dash preserved. Trailing period preserved.

The mandated head order is already in place and MUST be preserved unchanged: line 1 `# idea` H1 → line 3 blockquote → line 5 the contiguous three-badge run (`Latest release` / `Downloads` / `Stars`, all pointed at `sahil87/idea`) → line 7 tagline prose. Nothing above the H1; no frontmatter or HTML introduced.

### 2. Present-tense "run-kit" product mentions → HexoKit — verified no-op

The C7 row's second clause. Grep confirms zero occurrences of `run-kit`, `runkit`, `RunKit`, or "Run Kit" in `README.md`, `docs/site/**`, `docs/specs/**`, `src/**`, `scripts/**`, or `fab/project/**`. This repo never refers to the dashboard product in prose (its only sibling references are to fab-kit and `shll`). **No edits**, and therefore no `rk`-substrate judgment calls arise in this repo. Recorded so the apply agent does not go hunting.

### 3. Everything deliberately left untouched (Phase 1 scope)

| Surface | Occurrence | Why it stays |
|---------|------------|--------------|
| `README.md` line 12, 18; `docs/site/install.md` lines 10, 17 | `curl -fsSL https://shll.ai/install \| sh …` | shll.ai remains the live install endpoint until X2 (redirect stub) and the `install-composition` Policy B location flip is C1's standards-side change; the satellite install text is re-pointed when hexokit.com is the announced site |
| `README.md` line 15; `docs/site/install.md` lines 3, 14 | "the entire shll toolkit", "[shll toolkit](https://shll.ai)", "[shll installer](https://shll.ai)" | Family-name prose. D14/C1 explicitly leave "shll toolkit" mentions for X4; the same ordering applies to satellites — flipping them while the link target is still shll.ai would be internally inconsistent |
| `README.md` line 54; `docs/site/install.md` "Have other shll tools?" | `shll` named as the toolkit manager | D4: `shll` the command stays; `shll shell-install` is a real verb |
| `README.md` line 100 | `[full command reference](https://shll.ai/idea/commands/) on shll.ai` | shll.ai is still the consuming/rendering site until X2; the D7 redirect (`shll.ai/<tool>/* → hexokit.com/<tool>/*`) covers it afterwards |
| Badge URLs, `github.com/sahil87/idea/...` links, `raw.githubusercontent.com` | GitHub identifiers | Identifiers name the owner/repo, not the brand; repo links stay until R2 (and `idea`'s repo is not being renamed at all) |
| `fab/project/constitution.md` § Toolkit Standards ("part of the shll toolkit … `shll standards` … https://shll.ai") | Governance prose | D14: `shll standards` stays the command and canonical-source reference; toolkit-name wording waits for the X4-era pass. No amendment; `Last Amended` untouched |
| `src/cmd/idea/{help_dump.go,help_dump_test.go,skill.go}` comments | "shll.ai's puller", "renders at https://shll.ai/idea/skill" | Code comments describing the current consumer, which is still shll.ai; correctness fix belongs with X2 |
| `docs/site/skill.md` and its embedded copy `src/cmd/idea/skill/skill.md` | no brand mention | No edit → no `scripts/sync-skill.sh` re-run; the drift-guard test passes by construction |
| `fab/changes/**`, `fab/backlog.md`, `docs/memory/**` narrative | historical text | D11. Memory's *present-truth* line about the blockquote is corrected at hydrate (§ Affected Memory), not during apply |

### 4. Verification

- `head -7 README.md` shows H1 → blank → blockquote (new line, byte-exact) → blank → badge line → blank → tagline.
- `grep -c 'Part of \[HexoKit\](https://hexokit.com) — see all projects there\.' README.md` → 1; `grep -c 'shll toolkit](https://shll.ai) — see' README.md` → 0.
- `go test ./...` stays green (nothing in `src/` changes; run as the standing backstop, per the 92gj precedent).
- Conformance note for the PR body: the README's blockquote matches the `readme-extraction` standard **as revised in shll#98 (C1)**; the installed `shll` binary reports the pre-C1 line until shll releases C1 — this is the intended C1 → C7 ordering, not a defect.

## Affected Memory

- `cli/structure`: (modify) § Toolkit-standards conformance, final paragraph — currently states the README blockquote "is byte-identical to the readme-extraction standard's canonical line `> Part of the [shll toolkit](https://shll.ai) — see all projects there.`" and that toolkit-name prose uses "shll toolkit" (260718-92gj). Update the present-truth: the blockquote is now `> Part of [HexoKit](https://hexokit.com) — see all projects there.` per the standard as revised by shll#98 (HexoKit rebrand, plan row C7); note that "shll toolkit" family-name prose and `shll.ai` URLs are deliberately retained until the site cutover (X2/X4), and that the repo has zero run-kit product mentions so the C7 sweep's second clause was a verified no-op. Keep the 92gj attribution as history.
- `release/pipeline`: (modify, optional) § "shll.ai tool page: README + docs/site/** are also pulled and rendered" — if hydrate judges it useful, one sentence noting the blockquote text is brand-owned by the readme-extraction standard and changed under C7; the consumer extractor matches any leading blockquote (`BLOCKQUOTE_RE`), so the change is free on the pipeline side. No other content in this file is affected.

## Impact

- **Files**: `README.md` (1 line). Hydrate additionally touches `docs/memory/cli/structure.md` (and optionally `docs/memory/release/pipeline.md`). Markdown only.
- **Code/tests**: none. `go test ./...` runs as the standing backstop and must stay green.
- **External**: shll.ai's daily README pull renders the new blockquote at `/idea/readme` on the next pull after merge (the extractor matches any leading blockquote, so nothing on the pipeline side changes). hexokit.com's `/idea/readme` (S3 roster, same source repo) likewise. No release/tag required — the pull reads the repo, not the binary.
- **Cross-repo**: this is one of the C7 satellites (fab-kit, wt, idea, tu, hop, sahil87 profile). Merging it is a precondition for X1. After merge, update the plan doc's row C7 / Status line in the run-kit repo per its pickup protocol step 4 (operator action, outside this change's repo).
- **Ship**: normal flow — one fab change → PR against `main`.

## Open Questions

None — the replacement line is verified from the C1 PR diff, the sweep's second clause is a grep-verified no-op, the exclusion set is enumerated, and the phase ordering is fixed by the plan.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Replace README line 3 with `> Part of [HexoKit](https://hexokit.com) — see all projects there.` byte-identically (em-dash, no article, trailing period), preserving the H1 → blockquote → badges head order | Line read verbatim from shll#98's diff to `docs/site/standards/readme-extraction.md` at intake time; the prompt's `--` is ASCII shorthand for the standard's `—` | S:95 R:95 A:100 D:95 |
| 2 | Certain | The "present-tense run-kit product mentions → HexoKit" clause is a verified no-op in this repo — no edits, no substrate judgment calls | Repo-wide grep at intake time found zero `run-kit`/`runkit`/`RunKit` occurrences on any live surface; matches the plan's own "idea references run-kit 0×" | S:90 R:100 A:100 D:100 |
| 3 | Certain | Proceed although C1 (shll#98) is an open draft, not merged/released; take the revised line from the PR, not the installed binary | The prompt states the gate as "PR up, review-pr done" and the plan's order is C1 → (C3 ∥ C7); the installed `shll standards` lagging is the expected two-site-window state, flagged in the PR body | S:90 R:90 A:90 D:90 |
| 4 | Confident | Leave every `shll.ai` URL, the "shll toolkit"/"shll installer" family-name prose (README line 15, `docs/site/install.md` lines 3/14), the "Have other shll tools?" hint, and the constitution's article untouched | C7 scope names only the banner and run-kit product mentions; D14/C1 explicitly defer "shll toolkit" mentions to X4 because shll.ai is still the consuming site until X2. Trivially reversible if the operator wants the family-name prose flipped in this pass | S:75 R:95 A:75 D:65 |
| 5 | Certain | Memory present-truth (`cli/structure` § Toolkit-standards conformance) is corrected at hydrate, not during apply; `release/pipeline` touch is optional | Same split the 92gj precedent used; hydrate owns memory, and the paragraph in question is the one place memory states the blockquote text | S:70 R:95 A:85 D:80 |
| 6 | Certain | No `docs/site/skill.md` edit, no `scripts/sync-skill.sh` re-run, no source or test change; `go test ./...` as backstop only | Grep shows no brand mention in `docs/site/skill.md` or `src/**` user-visible strings; drift guard passes by construction | S:85 R:95 A:95 D:95 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
