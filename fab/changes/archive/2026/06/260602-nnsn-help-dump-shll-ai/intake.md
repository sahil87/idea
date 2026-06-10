# Intake: Build-time help-dump → shll.ai command reference

**Change**: 260602-nnsn-help-dump-shll-ai
**Created**: 2026-06-02
**Status**: Draft

## Origin

> Add a build-time 'help-dump' step that emits idea's CLI help tree as help/idea.json and PRs it into sahil87/shll.ai (the shll.ai landing site renders it as an expandable 'Command reference' on the idea tool page). CONTRACT (frozen — copy the reference sample at sahil87/shll.ai path help/wt.json): JSON shape is {tool, version, captured_at (ISO-8601 UTC), schema_version: 1, root: Node} where Node = {name, path (full invocation e.g. 'idea add'), short (one-line desc), usage, text (the RAW -h output byte-for-byte, newlines preserved), commands: Node[] (recursive; empty array = leaf)}. PRODUCER (idea is Cobra/Go, binary 'idea', main at src/cmd/idea): walk the cobra command tree programmatically (rootCmd.Commands() recursively), NOT regex-parsing -h text; per node capture cmd.Name / cmd.CommandPath() / cmd.Short / cmd.UseLine() and cmd.UsageString() (or Long+UsageString) as 'text'. FILTER OUT cobra's auto-generated 'completion' and 'help' subcommands and any cmd.Hidden==true. VERSION: read from the built binary (rootCmd.Version / ldflags) — do NOT hardcode. PUSH: in CI after build, run the dump, write help/idea.json, validate it parses, then open a PR into sahil87/shll.ai using the existing repo secret SHLLAI_TOKEN (contents + pull-request write) with auto-merge enabled (PR, not direct push to main, to avoid the multi-repo push race). This is idea's slice of a 7-tool rollout; the shll.ai site-side consumer (Astro loader + reference UI) is tracked separately in the shll.ai repo.

- **Mode**: One-shot, from backlog item `[nnsn]`.
- **Cross-repo contract**: The JSON shape is **frozen** and shared across a 7-tool rollout. The canonical reference is `sahil87/shll.ai` `help/wt.json`. This change must produce a byte-compatible sibling at `help/idea.json`. The site-side consumer (Astro loader + reference UI) lives in the shll.ai repo and is **out of scope** here.

## Why

1. **Problem**: The shll.ai landing site wants to render an expandable "Command reference" per tool, sourced from machine-generated data rather than hand-maintained docs. Hand-copying `-h` output into the site rots immediately — every flag change or new subcommand silently drifts. `idea` currently exposes its command surface only at runtime via cobra's `-h`; nothing emits it as structured data.
2. **Consequence if not fixed**: `idea`'s tool page either has no command reference or a stale hand-written one. Worse, since this is one slice of a 7-tool rollout with a frozen contract, an `idea`-specific deviation breaks the shared Astro loader for the whole site.
3. **Why this approach**:
   - **Programmatic tree walk over regex-parsing `-h`**: Cobra exposes the command graph as data (`rootCmd.Commands()`, `cmd.CommandPath()`, `cmd.UsageString()`). Walking it is exact and stable; regex-scraping rendered help text is fragile and re-derives what cobra already models. (Aligns with constitution Principle VI — output formats are a stable public contract.)
   - **Producer inside the binary** (a hidden subcommand), not an external Go program: the contract requires reading `rootCmd.Version` (the ldflags-stamped value) and the *live* cobra tree. Both are only available from inside the binary's command construction. A hidden subcommand reuses the exact tree the user sees and the exact stamped version, with zero duplication.
   - **PR with auto-merge, not direct push**: Multiple tools in the 7-tool rollout push to shll.ai concurrently. Direct pushes to `main` race and fail on non-fast-forward. A PR with auto-merge serializes integration through GitHub's merge queue semantics and avoids the race (stated explicitly in the contract).
   - **In CI after build**: The contract says read the version "from the built binary." Running the dump against a freshly built native-runner binary (with the real ldflags version) is the only way to capture the stamped version; running `go run` against `dev` would emit the wrong version.

## What Changes

### 1. Producer: hidden `help-dump` subcommand in the `idea` binary

A new cobra subcommand emits the help tree as JSON to stdout. It is **hidden** (`Hidden: true`) so it never appears in the user-facing help or in its own dump output.

**File**: `src/cmd/idea/help_dump.go` (new) — cobra factory `helpDumpCmd()`, registered in `main.go`.

**Refactor prerequisite**: `main.go` currently builds the root command inline inside `main()`. To let the producer walk the *same* tree, extract a `newRootCmd() *cobra.Command` factory that constructs root + registers all subcommands (including `helpDumpCmd()`), and have `main()` call it. The `help-dump` command, when run, walks its own `cmd.Root()` so it sees the identical tree.

**Walk algorithm** (programmatic, recursive — NOT regex on `-h`):

```go
type Node struct {
    Name     string `json:"name"`
    Path     string `json:"path"`
    Short    string `json:"short"`
    Usage    string `json:"usage"`
    Text     string `json:"text"`
    Commands []Node `json:"commands"`
}

func buildNode(cmd *cobra.Command) Node {
    n := Node{
        Name:     cmd.Name(),                 // e.g. "add"
        Path:     cmd.CommandPath(),          // e.g. "idea add"
        Short:    cmd.Short,                  // one-line desc
        Usage:    cmd.UseLine(),              // usage line
        Text:     cmd.UsageString(),          // RAW -h body, newlines preserved
        Commands: []Node{},                   // never nil → JSON "[]" for leaves
    }
    for _, c := range cmd.Commands() {
        if c.Hidden || c.Name() == "completion" || c.Name() == "help" {
            continue
        }
        n.Commands = append(n.Commands, buildNode(c))
    }
    return n
}
```

**Filtering** (applied during recursion, per contract):
- Skip `cmd.Hidden == true` (this also excludes `help-dump` itself).
- Skip cobra's auto-generated `completion` subcommand.
- Skip cobra's auto-generated `help` subcommand.
- Leaves emit `"commands": []` (empty array, never `null`) — initialize the slice, don't leave it nil.

**`text` field**: the RAW `-h` output, byte-for-byte, newlines preserved. **Composition resolved against the frozen `wt.json` reference sample** (fetched and decoded during clarify): every node's `text` is `{description}\n\n{UsageString()}`, where `{description}` is `cmd.Long` if non-empty, else `cmd.Short`. `cmd.UsageString()` renders the `Usage:` / `Available Commands:` / `Flags:` blocks but does **not** include Long/Short — so the producer concatenates explicitly: `text = longOrShort(cmd) + "\n\n" + cmd.UsageString()`. This reproduces exactly what `idea <cmd> -h` prints (confirmed: in `wt.json`, both the root node and leaf nodes like `list` lead with their full Long description, then a blank line, then `Usage:`). `json.Marshal` preserves embedded newlines as `\n` escapes; do not strip or normalize whitespace.

> Helper: `func longOrShort(c *cobra.Command) string { if c.Long != "" { return c.Long }; return c.Short }`. Edge case: if both `Long` and `Short` are empty, emit just `UsageString()` (no leading blank lines) — no `idea` command currently has both empty, but guard defensively.

**Top-level envelope**:

```go
type Dump struct {
    Tool          string `json:"tool"`           // "idea"
    Version       string `json:"version"`         // from rootCmd.Version (ldflags-stamped)
    CapturedAt    string `json:"captured_at"`     // time.Now().UTC().Format(time.RFC3339)
    SchemaVersion int    `json:"schema_version"`  // 1
    Root          Node   `json:"root"`            // buildNode(rootCmd)
}
```

- `tool`: literal `"idea"`.
- `version`: read from `cmd.Root().Version` (the value cobra's `--version` prints), i.e. the ldflags-stamped `main.version`. **Never hardcode.** When run from a release binary this is the tag (e.g. `v0.2.1`); from a `dev` build it is `dev` (acceptable for local testing, but CI always runs the stamped binary).
- `captured_at`: ISO-8601 UTC, RFC3339 (`2026-06-02T14:03:21Z`).
- `schema_version`: integer `1`.
- `root`: the recursively-built tree starting at the root command. The root node's own `commands` filtering excludes `completion`, `help`, and `help-dump`.

**Output**: marshal with `json.MarshalIndent(dump, "", "  ")` (2-space indent — confirm against `wt.json` indentation), write to **stdout**, exit 0. The CI step redirects stdout to `help/idea.json`. Keeping it stdout-only (no `--output` file flag) keeps the subcommand pure and testable; the CI step owns file placement.

**Contract-fidelity step**: Before implementing, fetch `sahil87/shll.ai` `help/wt.json` and diff the produced `idea.json` structure against it field-for-field (key names, ordering, indentation, `text` composition, leaf `commands: []`). The reference sample is the frozen truth; the producer conforms to it.

### 2. CI: dump + validate + PR step in `release.yml`

Add a new step to `.github/workflows/release.yml`, **after** the `Cross-compile` step (so a stamped binary exists) and after the GitHub Release is created (the help dump is a downstream, best-effort publish, not a release blocker — order it last so a failure here doesn't abort the release).

**Binary selection**: Use the native runner binary. The runner is `ubuntu-latest` (linux/amd64), so run `dist/idea-linux-amd64/idea help-dump`. This binary carries the real ldflags version stamp from the Cross-compile step.

**Producer obligation (resolved against shll.ai contract PR #12, branch `260602-xiis-help-collection-contract`)**: The producer's only job is to **open a PR** that touches `help/idea.json` and nothing else. shll.ai owns merging via its own receiving-side workflow `.github/workflows/help-automerge.yml`, which enables auto-merge only when **all three guards** pass:
1. **Actor guard** — PR author must be the trusted identity `sahil87` (the workflow's `TRUSTED_AUTHOR`). ⇒ `SHLLAI_TOKEN` must be a `sahil87` PAT, and the PR must be created with that token (so `gh` attributes authorship to `sahil87`, not `github-actions[bot]`).
2. **Content guard** — every changed file must be under `help/`. ⇒ the PR diff is **exactly** `help/idea.json`, nothing else.
3. **Schema-validation gate** — the JSON must pass shll.ai's `validate-help.mjs` (a Zod `Node`/envelope contract). ⇒ field names, recursion, and envelope must match the contract exactly.

⇒ **The idea-side CI step does NOT call `gh pr merge`.** It opens the PR and stops; the receiving workflow merges. (Repo settings confirm `allow_auto_merge: false` at the native-feature level — auto-merge is the workflow's job, not a `gh pr merge --auto` call from here. Calling it from idea's side would fail and duplicate the mechanism.)

```yaml
- name: Dump help tree and PR to shll.ai
  env:
    GH_TOKEN: ${{ secrets.SHLLAI_TOKEN }}   # sahil87 PAT: contents + pull-requests write on shll.ai
  run: |
    version="${{ steps.version.outputs.version }}"

    # 1. Produce the dump from the stamped native binary
    mkdir -p help
    dist/idea-linux-amd64/idea help-dump > help/idea.json

    # 2. Validate it parses as JSON locally (fail the step early if malformed;
    #    the authoritative schema check is shll.ai's validate-help.mjs gate).
    python3 -c "import json; json.load(open('help/idea.json'))" \
      || { echo "help/idea.json is not valid JSON"; exit 1; }

    # 3. Clone shll.ai, stage ONLY help/idea.json (content guard), open a PR.
    git clone "https://x-access-token:${GH_TOKEN}@github.com/sahil87/shll.ai.git" /tmp/shll-ai
    cp help/idea.json /tmp/shll-ai/help/idea.json
    cd /tmp/shll-ai
    if git diff --quiet -- help/idea.json; then
      echo "No change to help/idea.json — skipping PR"
      exit 0
    fi
    branch="help-dump/idea-${version}"
    # Author as sahil87 to satisfy the actor guard (TRUSTED_AUTHOR).
    git config user.name "sahil87"
    git config user.email "sahil@noon.design"
    git checkout -b "$branch"
    git add help/idea.json                      # only this file — content guard
    git commit -m "idea: update command reference (v${version})"
    git push -u origin "$branch"
    gh pr create --repo sahil87/shll.ai --base main --head "$branch" \
      --title "idea: command reference v${version}" \
      --body "Automated help-dump from sahil87/idea v${version}. Auto-merges via help-automerge.yml once the three guards (actor/content/schema) pass."
    # NO `gh pr merge` here — shll.ai's help-automerge.yml owns merging.
```

Notes / decisions baked into the step:
- **`SHLLAI_TOKEN`** is the existing repo secret — a **`sahil87` PAT** with `contents` + `pull-requests` write on `sahil87/shll.ai` (required so authorship passes the actor guard). Exported as `GH_TOKEN` so `gh` and the clone URL both authenticate against shll.ai, not the current repo's `GITHUB_TOKEN`.
- **PR not direct push**: serializes the multi-repo write to avoid the push race named in the contract; the receiving workflow + the existing push-to-`main` deploy then ship it.
- **No `gh pr merge`** on idea's side — merging is shll.ai's `help-automerge.yml` responsibility (gated by the three guards).
- **Content guard compliance**: stage and commit **only** `help/idea.json`. A mixed diff would be left for manual review by the receiving workflow.
- **Schema-gate compliance**: the produced JSON must satisfy shll.ai's Zod `Node`/envelope schema (field names + recursion + envelope). Diffing against `help/wt.json` during apply is the cheapest way to verify before the gate sees it.
- **No-op guard**: if `help/idea.json` is unchanged (no flag/command churn since last release), skip the PR entirely to avoid empty-PR spam.
- **Branch-per-version** (`help-dump/idea-${version}`) keeps PRs isolated and re-runnable.

### 3. Tests

Per constitution Principle V (table-driven, real I/O, no mocks):
- **`src/cmd/idea/help_dump_test.go`**: build/execute the `help-dump` command in-process (capture stdout via `cmd.SetOut`), unmarshal the JSON, and assert:
  - Envelope fields present: `tool == "idea"`, `schema_version == 1`, non-empty `version`, `captured_at` parses as RFC3339.
  - `root.name == "idea"`, `root.path == "idea"`.
  - `completion`, `help`, and `help-dump` are **absent** from `root.commands`.
  - Every known user subcommand (`add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`, `update`, plus the shell-init command) **is present** with correct `path` (e.g. `idea add`).
  - Leaf nodes serialize `commands` as `[]` (not `null`) — assert on the raw JSON bytes.
  - `text`/`usage`/`short` are non-empty for representative nodes.
- Keep the JSON struct types unexported in the `main` package unless a test in another package needs them; the in-package test can reach them directly.

## Affected Memory

- `cli/structure.md`: (modify) Document the new hidden `help-dump` subcommand, the `newRootCmd()` factory extraction, and the JSON contract it emits.
- `release/pipeline.md`: (modify) Document the new CI step (dump → validate → PR to shll.ai), the `SHLLAI_TOKEN` secret, and the auto-merge/PR-not-push rationale. Add `SHLLAI_TOKEN` to the Secrets section.

## Impact

- **Source**: `src/cmd/idea/main.go` (extract `newRootCmd()`); `src/cmd/idea/help_dump.go` (new); `src/cmd/idea/help_dump_test.go` (new). No changes to `internal/idea` — this is a presentation-layer concern over the existing command graph (consistent with Principle IV: logic stays in `internal`, but here there is no *backlog* logic, only cobra-tree serialization, which is cmd-layer-appropriate).
- **CI**: `.github/workflows/release.yml` — one new step. The job already declares `permissions: contents: write` for the in-repo release; the shll.ai write uses `SHLLAI_TOKEN`, not `GITHUB_TOKEN`, so no in-repo permission change is needed.
- **Dependencies**: none added. `encoding/json` and `time` are stdlib (constitution dependency discipline satisfied — no new module deps).
- **Cross-repo (out of scope, dependencies — verified during clarify against shll.ai contract PR #12)**:
  - `SHLLAI_TOKEN` must be available to *this* repo's Actions as a **`sahil87` PAT** with `contents` + `pull-requests` write on shll.ai. The `sahil87` authorship is load-bearing: shll.ai's `help-automerge.yml` actor guard (`TRUSTED_AUTHOR: sahil87`) refuses to auto-merge PRs from any other author. A `github-actions[bot]`-authored PR would be left for manual review.
  - Merging is **owned by shll.ai**, not idea: `.github/workflows/help-automerge.yml` enables auto-merge for inbound `help/**` PRs once three guards pass (actor = `sahil87`, content = `help/**`-only diff, schema = `validate-help.mjs`). idea's CI must NOT call `gh pr merge`. (shll.ai repo `allow_auto_merge` is `false` at the native-feature level — the workflow does the merge.)
  - The produced JSON must conform to shll.ai's Zod schema (`sites/astro-starlight-terminal1/src/lib/schemas.ts`) — the recursive `Node` + envelope. The `help/wt.json` reference sample is the frozen template; if it changes, this producer must follow.
- **Build reproducibility**: unaffected — no new ldflags, no codegen. The dump runs *after* build, consuming the already-built binary (constitution Build Reproducibility satisfied).

## Open Questions

- ~~`text` field composition~~ — **RESOLVED** during clarify by decoding the frozen `help/wt.json`: `text = longOrShort(cmd) + "\n\n" + cmd.UsageString()`. See What Changes §1.
- ~~Auto-merge mechanism / merge strategy~~ — **RESOLVED** during clarify against shll.ai contract PR #12: shll.ai's `help-automerge.yml` owns merging via three guards; idea's CI only opens the PR (no `gh pr merge`). See What Changes §2.
- `config.yaml` `source_paths` lists `src/go/idea` but the actual tree is `src/cmd/idea` + `src/internal/idea` (fab-kit-style vs. hop-style layout). This is a pre-existing config drift, **not** caused by this change. Flagging for a separate cleanup; out of scope here.
- Should the dump also run on every push to `main` (keeping shll.ai fresh between releases), or only on release tags? Contract framing is release-context ("in CI after build"), and the version stamp only exists in the release pipeline; defaulting to release-only. (See Assumption 8.) A push-to-`main` variant would emit `version: dev` unless reworked — another reason to keep it release-only.

## Clarifications

### Session 2026-06-02

Resolved by fetching authoritative cross-repo sources (the frozen `help/wt.json` reference sample and shll.ai contract PR #12 / branch `260602-xiis-help-collection-contract`) rather than asking — these are the contract's own source of truth.

| # | Action | Detail |
|---|--------|--------|
| 12 | Resolved | `text = longOrShort(cmd) + "\n\n" + UsageString()` — decoded from `wt.json` (root + leaves lead with full Long/Short, then `Usage:`). Tentative → Certain. |
| 13 | Resolved | idea's CI opens a PR only; shll.ai's `help-automerge.yml` owns merging (three guards). No merge-strategy decision on idea's side. Tentative → Certain. |
| 5 | Refined | PR-not-push confirmed; the "auto-merge" is the receiving workflow's job, not a `gh pr merge` call. Repo `allow_auto_merge` is `false` at native level. |
| — | New (14) | PR must be authored as `sahil87` to pass the actor guard (`TRUSTED_AUTHOR`); `SHLLAI_TOKEN` is a sahil87 PAT. Corrects the original `github-actions[bot]` author. |
| — | New (15) | CI must commit ONLY `help/idea.json` (content guard) and the JSON must pass shll.ai's Zod `validate-help.mjs` (schema gate). |

### Session 2026-06-02 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 8 | Confirmed | — |
| 9 | Confirmed | — |
| 10 | Confirmed | — |
| 11 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | JSON envelope = `{tool, version, captured_at, schema_version:1, root:Node}`; Node = `{name, path, short, usage, text, commands[]}` | Frozen contract, copied verbatim from `help/wt.json` reference sample | S:98 R:70 A:95 D:98 |
| 2 | Certain | Walk cobra tree programmatically via `Commands()`/`CommandPath()`/`UsageString()`, not regex on `-h` | Explicitly mandated by contract | S:98 R:75 A:95 D:98 |
| 3 | Certain | Filter out `completion`, `help`, and any `Hidden==true` command | Explicitly mandated by contract | S:98 R:80 A:95 D:98 |
| 4 | Certain | Version read from `rootCmd.Version` (ldflags-stamped), never hardcoded | Explicitly mandated; wiring already exists in `main.go` (`var version`) | S:98 R:80 A:98 D:98 |
| 5 | Certain | Open a PR into `sahil87/shll.ai` via `SHLLAI_TOKEN` (sahil87 PAT); merging owned by shll.ai's `help-automerge.yml`, NOT a `gh pr merge` call from idea | Clarified — verified against shll.ai contract PR #12: receiving-side workflow auto-merges on three guards; idea only opens the PR. Avoids multi-repo push race | S:95 R:65 A:95 D:95 |
| 6 | Certain | Producer is a **hidden `help-dump` subcommand** inside the `idea` binary (not an external program) | Clarified — user confirmed | S:95 R:75 A:80 D:78 |
| 7 | Certain | Extract `newRootCmd()` factory from `main()` so the producer walks the same tree | Clarified — user confirmed | S:95 R:75 A:85 D:75 |
| 8 | Certain | CI step runs on release tags only (in `release.yml`), after Cross-compile, ordered last/best-effort | Clarified — user confirmed | S:95 R:75 A:75 D:70 |
| 9 | Certain | Use the native `linux/amd64` runner binary for the dump | Clarified — user confirmed | S:95 R:85 A:85 D:80 |
| 10 | Certain | Dump emits to stdout; CI redirects to `help/idea.json` (no in-binary `--output` flag) | Clarified — user confirmed | S:95 R:85 A:80 D:75 |
| 11 | Certain | No-op guard: skip the PR when `help/idea.json` is unchanged | Clarified — user confirmed | S:95 R:90 A:80 D:75 |
| 12 | Certain | `text = longOrShort(cmd) + "\n\n" + cmd.UsageString()` (Long if non-empty, else Short) | Clarified — decoded frozen `help/wt.json`: every node (root + leaves) leads with full Long/Short, blank line, then `Usage:`. Matches byte-for-byte `-h` output per contract | S:95 R:70 A:90 D:90 |
| 13 | Certain | idea's CI does NOT pick a merge strategy — it only opens the PR; shll.ai's `help-automerge.yml` merges | Clarified — contract PR #12 assigns merging to the receiving workflow. No merge-strategy decision exists on idea's side | S:95 R:80 A:90 D:90 |
| 14 | Certain | PR must be authored as `sahil87` (commit identity `sahil87`/`sahil@noon.design`), not `github-actions[bot]` | Clarified — shll.ai actor guard `TRUSTED_AUTHOR: sahil87` refuses to auto-merge other authors; `SHLLAI_TOKEN` is a sahil87 PAT | S:95 R:70 A:90 D:90 |
| 15 | Certain | CI step stages/commits ONLY `help/idea.json` (content guard) and JSON must pass shll.ai Zod schema | Clarified — contract PR #12 content guard rejects mixed diffs; `validate-help.mjs` gates the schema | S:95 R:75 A:90 D:90 |

15 assumptions (15 certain, 0 confident, 0 tentative, 0 unresolved).
