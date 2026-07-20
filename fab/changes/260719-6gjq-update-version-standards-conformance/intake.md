# Intake: Update & Version Standards Conformance

**Change**: 260719-6gjq-update-version-standards-conformance
**Created**: 2026-07-20

## Origin

One-shot `/fab-new` invocation:

> Bring this repo into conformance with the shll toolkit 'update' and 'version' standards (docs/site/standards/update.md and version.md in the shll repo, or https://shll.ai/standards). Audit the update and --version subcommands against every MUST/SHOULD in both standards, fix any gaps found, and add/update tests pinning the fixed behavior. If the audit finds the repo is already fully conformant with no code changes needed, skip /git-pr entirely — do not open an empty PR.

The audit was performed during intake (via `shll standards update` and `shll standards version`) so the gaps are known concretely, not hypothetically. Two gaps were found, so the "skip PR if fully conformant" branch does NOT apply — this change proceeds through the full pipeline.

## Why

The constitution (§ Toolkit Standards) binds this repo to the shll toolkit's published standards. Two of those standards govern surfaces this repo already ships:

1. **`update` standard — brew-handling safety clause (MUST-level) is violated.** `src/internal/idea/update.go` runs all three brew subprocesses under `context.WithTimeout` + `exec.CommandContext` (`brewUpdateTimeout` 30s, `brewInfoTimeout` 30s, `brewUpgradeTimeout` 120s). `exec.CommandContext`'s default cancel function sends `os.Kill` (**SIGKILL**) on deadline. The standard says: "MUST NOT send `SIGKILL` to a package-manager subprocess mid-transaction", "MUST NOT impose a short hard timeout on `brew upgrade`", and its verification checklist requires "No code path sends `SIGKILL` to `brew`". The 120-second hard kill on `brew upgrade` is *literally the incident the standard cites* as its motivating failure mode (observed 2026-07-19: a stalled `api.github.com` call inside `brew upgrade` exceeded a wrapper's 120s hard kill; the SIGKILL landed between `brew unlink` and `brew link` and corrupted the keg — `zsh: permission denied: <tool>`). Left unfixed, any user of `idea update` on a slow network moment can end up with a half-installed binary.

2. **`version` standard — conformance test missing (checklist item).** The `--version` behavior itself is conformant (see audit results below), but the standard's verification checklist says "Keep (or add) a minimal test pinning the above — exit 0, version on line 1, matches the shape — so the contract stays protected." No such test exists (`main_test.go` has no version test). Without it, a future change (e.g. a banner line, a version-check notice) could silently break shll's first-line-only parse.

### Full audit results (what is already conformant — no change needed)

**update standard:**
- ✅ `update` subcommand exists, upgrades in place, works standalone (`src/cmd/idea/update.go`)
- ✅ `idea update --help` contains the literal substring `--skip-brew-update` (flag registration guarantees it) and the flag is honored — skips only the internal `brew update --quiet` (pinned by `TestUpdateSkipBrewUpdate`)
- ✅ Exits 0 on success **including already-up-to-date** (`Already up to date (...)` → `return nil`); non-zero only on genuine failure
- ✅ Self-update gated on brew install via `os.Executable()` → `filepath.EvalSymlinks` → `/Cellar/` substring; non-brew installs get a clear two-line hint and exit 0
- ✅ One-name identity: repo `sahil87/idea`, binary `idea`, formula `sahil87/tap/idea`, roster name `idea`; releases tagged `v{semver}` (release.sh); no rename in flight
- ❌ Brew-handling safety (gap 1 above)

**version standard:**
- ✅ `idea --version` supported via cobra `Version: version` field (`main.go`); exits 0; writes to stdout
- ✅ Output is cobra's stable default: `idea version {version}` — first non-empty line, matches the standard's `<word> version <rest>` prefix rule, and with a release-stamped `vX.Y.Z` it is exactly the RECOMMENDED canonical shape `idea version vX.Y.Z`
- ✅ Purely local — no network I/O, responds far under the 2-second budget
- ✅ Binary name on PATH (`idea`) equals the tool name
- ❌ No test pinning the contract (gap 2 above)

## What Changes

### 1. Remove SIGKILL-bearing deadlines from all brew subprocesses (`src/internal/idea/update.go`)

Delete the three timeout constants (`brewUpdateTimeout`, `brewInfoTimeout`, `brewUpgradeTimeout`) and the `context.WithTimeout` wrappers around all three brew invocations (`brew update --quiet`, `brew info --json=v2`, `brew upgrade`). Each call passes `context.Background()` through the **unchanged** `execCommandContext(ctx, ...)` seam — the seam signature stays as-is so the existing test stubbing pattern keeps working, and the recorded `ctx` becomes the test's assertion surface.

Chosen approach: **no bound at all**, rather than a generous SIGTERM+grace bound, because:
- Go's graceful pattern (`cmd.Cancel` = SIGTERM + `cmd.WaitDelay`) still sends SIGKILL when the grace elapses — the checklist's "No code path sends SIGKILL to brew" is only strictly satisfiable by having no kill path
- The standard itself points away from timeouts ("…rather than reaching for a timeout at all")
- Ctrl-C remains the user's escape hatch: SIGINT goes to the foreground process group, brew traps it and unwinds cleanly
- `brew upgrade` already inherits the tty (`os.Stdin/Stdout/Stderr`), so a slow upgrade is visible, not a silent hang; `brew update`/`brew info` run after the wrapper prints `Checking for updates...`, so the user knows what is running

`exec.CommandContext` with `context.Background()` behaves identically to `exec.Command` (the cancel path is never armed). No behavior change on the success path; wrapper messages, I/O routing, error mapping (`errSilent` / `exec.ErrNotFound`) are all untouched.

### 2. Pin the no-deadline contract in tests (`src/internal/idea/update_test.go`)

Extend the existing `execCommandContext` recorder (used by `TestUpdateSkipBrewUpdate`) to also capture the `ctx` passed at each call site, and assert `ctx.Deadline()` reports **no deadline** for every recorded brew invocation (`update`, `info`, `upgrade`). This pins the MUST NOT clause: reintroducing a `context.WithTimeout` fails the test.

### 3. Add the version-shape conformance test (`src/cmd/idea/main_test.go`)

A CLI-level test following the file's existing pattern (build root via `newRootCmd()`, execute with args, capture out/err):

- Execute the root command with `--version`
- Assert: no error (exit 0 path), output on **stdout** (stderr empty), and the **first non-empty line** matches `^idea version \S+$` — the `<word> version <rest>` prefix shape shll's `versionPrefixRE` parses (the dev build emits `idea version dev`; release builds emit `idea version vX.Y.Z`, the RECOMMENDED canonical shape — both satisfy the prefix rule)
- Assert nothing precedes the version line (no banner) — i.e. the version line IS line 1

### Out of scope

- `HOMEBREW_NO_GITHUB_API=1` — the standard suggests it only for tools that *must* bound the call; with no bound, it's unnecessary
- Consumer-side behavior (`shll update` / `shll version` internals), naming/release alignment, help-dump — audited conformant, no change
- `docs/memory/cli/update.md`'s "Timeout constants" section becomes stale — updated at hydrate, not here

## Affected Memory

- `cli/update`: (modify) Replace the "Timeout constants" section with the no-deadline brew-safety contract (why no `context.WithTimeout` may wrap brew subprocesses, per the toolkit update standard); note the ctx-deadline test assertion in the test-seam section
- `cli/structure`: (modify) Minor — note the version-shape conformance test alongside the existing toolkit-standards conformance notes

## Impact

- `src/internal/idea/update.go` — constants deleted, three call sites switch to `context.Background()`, doc comments updated
- `src/internal/idea/update_test.go` — recorder extended with ctx capture + no-deadline assertions
- `src/cmd/idea/main_test.go` — new version-shape test
- No CLI surface change, no help-text change (timeouts appear nowhere in help), so no help-dump or README/docs-site impact
- No new dependencies (stdlib only — Dependency Discipline holds)

## Open Questions

*(none — the standards are explicit and the audit resolved the input concretely)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Fix brew-safety by removing deadlines entirely (`context.Background()`), not by a generous SIGTERM+grace bound | Standard's checklist ("No code path sends SIGKILL to brew") is only strictly met with no kill path — Go's `WaitDelay` still SIGKILLs after grace; standard itself suggests avoiding timeouts; Ctrl-C remains the escape hatch. Easily revisited if a bound is later wanted. <!-- assumed: no-bound over SIGTERM+grace — strictest conformance reading --> | S:80 R:85 A:80 D:65 |
| 2 | Certain | Keep the `execCommandContext(ctx, ...)` seam signature unchanged; tests assert conformance via the recorded ctx's absent deadline | Minimal-diff path; existing test seam is designed exactly for this; memory doc documents the seam as deliberate | S:75 R:90 A:95 D:90 |
| 3 | Certain | Version test lives in `src/cmd/idea/main_test.go` following its existing root-command test pattern, pinning exit-0 / stdout / first-line `^idea version \S+$` | Standard's checklist names exactly these assertions; `main_test.go` is where CLI-level contract tests live (exit codes, output contracts) | S:75 R:95 A:90 D:85 |
| 4 | Certain | All other MUST/SHOULD items in both standards need no code change (audited conformant: `--skip-brew-update` advertise+honor, exit codes, `/Cellar/` gating, one-name identity, v-tags, first-line shape, 2s/no-network, PATH name) | Verified directly against source, help output, existing tests, and release memory during intake | S:85 R:80 A:90 D:85 |
| 5 | Confident | `brew update` and `brew info` also lose their deadlines, not just `brew upgrade` | Checklist clause "No code path sends SIGKILL to brew" is unqualified; `brew update` mutates tap state (a transaction); uniform treatment is simpler than a per-call split | S:75 R:85 A:80 D:70 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
