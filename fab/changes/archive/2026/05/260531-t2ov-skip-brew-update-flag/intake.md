# Intake: Add --skip-brew-update flag to update command

**Change**: 260531-t2ov-skip-brew-update-flag
**Created**: 2026-05-31
**Status**: Draft

## Origin

> Add a boolean --skip-brew-update flag to the `update` command. CONTRACT (cross-toolkit, identical in 6 tools): flag name EXACTLY --skip-brew-update. When set, skip ONLY the internal `brew update` tap-metadata refresh. Everything else unchanged: `brew info` version check, up-to-date short-circuit, `brew upgrade`. Default (absent) = current behavior exactly preserved. THIS REPO (idea): update logic in src/internal/idea/update.go (func Update, the `brew update` call ~L68); wire a real cobra bool flag in cmd/idea/update.go and pass it into Update(). Thread skipBrewUpdate bool through Update(). Preserve the intentional output routing (brew update/info captured, upgrade inherits stdio). Match existing subprocess convention (do NOT refactor). Add a test asserting --skip-brew-update omits `brew update` but still runs `brew upgrade`, following the repo test pattern. Build + run the update package tests before the PR.

One-shot invocation via `/fab-new`. This is one of six identical implementations of a cross-toolkit contract — the flag name, semantics, and default behavior are fixed by that contract and are not open for reinterpretation in this repo. The only repo-specific latitude is *how* to satisfy the "add a test asserting brew update is omitted but brew upgrade still runs" requirement given that the existing `Update()` has no injection seam for observing subprocess invocations.

## Why

1. **Problem**: `idea update` always runs `brew update --quiet` first to refresh tap metadata before checking and upgrading. In automation, CI, or when a caller has *just* run `brew update` (e.g. a wrapper script orchestrating updates across several brew-installed tools), that refresh is redundant and adds latency (a network round-trip against every tap). The cross-toolkit contract introduces `--skip-brew-update` so callers can opt out of the redundant refresh uniformly across all six tools.
2. **Consequence of not doing it**: Each of the six tools keeps paying a redundant `brew update` per invocation in orchestrated/automated contexts, and the toolkit lacks a consistent, predictable flag for suppressing it — callers would have to special-case each tool.
3. **Why this approach**: A single boolean flag threaded through `Update()` is the minimal change that satisfies the contract. It touches exactly the cobra wiring (`cmd/idea/update.go`) and one branch in `Update()` (`src/internal/idea/update.go`), preserves every other behavior byte-for-byte, and keeps the flag name/semantics identical to the other five tools.

## What Changes

### 1. New cobra bool flag in `src/cmd/idea/update.go`

Register a real cobra bool flag named exactly `--skip-brew-update` on the `update` subcommand (local flag, not persistent — it is specific to `update`). Default `false`. Short description matching the contract semantics (skip the internal `brew update` tap-metadata refresh). Read its value in `RunE` and pass it to `idea.Update(...)`.

### 2. Thread `skipBrewUpdate bool` through `Update()` in `src/internal/idea/update.go`

Change the signature of `Update` to accept the new boolean. The natural, convention-matching placement is to add it as a parameter. Proposed:

```go
func Update(currentVersion string, skipBrewUpdate bool, out, errOut io.Writer) error
```

Inside `Update`, guard **only** the `brew update --quiet` block (currently ~L67–L82) behind `if !skipBrewUpdate { ... }`. Everything after it is unchanged:

- `brewLatestVersion()` (`brew info --json=v2`) version check — always runs.
- The `normalizeVersion(latest) == normalizeVersion(currentVersion)` up-to-date short-circuit — always runs.
- `brew upgrade sahil87/tap/idea` (stdio-inheriting) — always runs when not up to date.
- The non-Homebrew-install short-circuit at the top — unchanged.

When `skipBrewUpdate` is true, the `brew update` subprocess is simply never spawned; control flows straight to `brewLatestVersion()`. Default (flag absent → `false`) reproduces the current behavior exactly: `brew update` runs as before.

### 3. Preserve output routing exactly

No change to I/O routing. `brew update` and `brew info` streams stay captured; `brew upgrade` keeps inheriting `os.Stdin/Stdout/Stderr`. Wrapper messages keep going to `out`/`errOut`. The `errSilent` mapping in `RunE` is untouched.

### 4. Test asserting the flag's effect (repo test pattern)

Add a table-driven test in `src/internal/idea/update_test.go` that asserts: with `--skip-brew-update` set, `brew update` is **not** invoked, but `brew upgrade` (and `brew info`) **is**; and with the flag absent, `brew update` **is** invoked.

The repo's current tests cannot reach the brew code path (`isBrewInstalled()` is false under `go test`, and `Update` shells out to real `brew` with no seam). To observe *which* brew subprocesses run without refactoring the subprocess convention, the leanest approach is a single package-level indirection seam over command construction (e.g. a `var execCommand = exec.CommandContext` that tests stub to record invoked args), plus making `isBrewInstalled` overridable in tests the same way, so the test can drive `Update` down the brew path deterministically and record the `brew update` / `brew info` / `brew upgrade` argv. This keeps `os/exec` as the mechanism (no `internal/proc` wrapper, no interface abstraction) — it is the minimal seam, not a refactor of the convention. The exact seam shape is the one open design decision (see Open Questions / Assumptions).

## Affected Memory

- `cli/update.md`: (modify) Document the `--skip-brew-update` flag: its semantics (skips only the `brew update` tap-metadata refresh), the threaded `skipBrewUpdate bool` parameter on `Update()`, the preserved output routing, and the test seam introduced to assert subprocess invocation.

## Impact

- `src/cmd/idea/update.go` — add cobra bool flag wiring; pass value to `Update`.
- `src/internal/idea/update.go` — `Update` signature gains `skipBrewUpdate bool`; guard the `brew update` block. Possibly a small `var execCommand`/`isBrewInstalled` seam for testability.
- `src/internal/idea/update_test.go` — new table-driven test for flag behavior.
- No new dependencies (stdlib + cobra only — Dependency Discipline preserved).
- No change to the backlog format, worktree resolution, or any other subcommand.
- `Update` has exactly one caller (`cmd/idea/update.go`), so the signature change is fully contained.

## Open Questions

- What is the minimal test seam that lets the test observe `brew update` omission vs. `brew upgrade` invocation **without** refactoring the subprocess convention? Candidates: (a) a package-level `var execCommand = exec.CommandContext` plus a test-overridable `isBrewInstalled`, recording argv via a stub; (b) a `PATH`-shimmed fake `brew` script created in a temp dir (no production code seam at all). Option (b) keeps production code untouched but is heavier and less idiomatic for this repo's table-driven unit style; option (a) is the smaller, more idiomatic seam. Leaning (a).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Flag name is exactly `--skip-brew-update`, boolean, default `false` | Fixed by the cross-toolkit contract; not open for interpretation | S:100 R:90 A:95 D:100 |
| 2 | Certain | Skip guards ONLY the `brew update --quiet` block; `brew info`, up-to-date short-circuit, and `brew upgrade` always run | Explicit in contract; matches the single-branch placement in `Update` | S:100 R:80 A:95 D:100 |
| 3 | Certain | Default (flag absent) preserves current behavior byte-for-byte | Explicit in contract | S:100 R:85 A:95 D:100 |
| 4 | Certain | Output routing unchanged (update/info captured, upgrade inherits stdio); no subprocess-convention refactor | Explicit in contract; matches documented I/O split in `cli/update.md` | S:95 R:75 A:95 D:95 |
| 5 | Confident | Thread the bool as a new parameter on `Update`: `Update(currentVersion string, skipBrewUpdate bool, out, errOut io.Writer)` | Contract says "thread skipBrewUpdate bool through Update()"; param is the idiomatic Go way; `Update` has one caller so the change is contained. Parameter order (bool before writers) groups inputs before output sinks | S:80 R:70 A:80 D:70 |
| 6 | Confident | Flag is a local flag on the `update` command, not a root persistent flag | Constitution III reserves persistent flags for `--file`/`--main`; this flag is update-specific | S:75 R:80 A:90 D:80 |
| 7 | Confident | Test follows the repo's table-driven pattern in `update_test.go` and is run via `cd src && go test ./internal/idea/...` | Constitution V + config stage_directives ("prefer table-driven tests"); justfile uses `cd src && go test` | S:80 R:85 A:90 D:80 |
| 8 | Tentative | Test seam = package-level `var execCommand = exec.CommandContext` + test-overridable `isBrewInstalled`, with a recording stub — chosen over a PATH-shimmed fake `brew` | Contract forbids refactoring the subprocess convention; a single `var` indirection keeps `os/exec` as the mechanism and is the minimal idiomatic seam, but it does touch production code, so other readings exist (PATH shim touches none). Resolvable at spec/apply | S:55 R:55 A:55 D:45 |

8 assumptions (4 certain, 3 confident, 1 tentative, 0 unresolved).
