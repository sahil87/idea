# Intake: Add `idea update` Subcommand

**Change**: 260508-5bw2-add-update-subcommand
**Created**: 2026-05-08
**Status**: Draft

## Origin

> Create an idea update subcommand - that does the same thing as hop update (take reference from ~/code/sahil87/hop/)

One-shot invocation. The user pointed at a sibling repo (`~/code/sahil87/hop/`) as the reference implementation. `hop update` is a self-update command that re-installs the binary via Homebrew when the binary was originally installed via Homebrew, and prints a manual-update hint otherwise. The `idea` binary is shipped via the same release pipeline as `hop` (tag-driven build → GitHub Release → Homebrew tap formula), so a structural port is appropriate.

## Why

1. **Problem**: Users who installed `idea` via `brew install sahil87/tap/idea` have no in-binary mechanism to upgrade. Today they must remember to run `brew update && brew upgrade sahil87/tap/idea` themselves — friction that competes with the same convenience hop already offers.
2. **Consequence if unfixed**: Brew-installed `idea` users run stale versions silently. The release pipeline can ship a fix, but uptake lags because the upgrade path isn't discoverable from the binary itself.
3. **Why this approach**: `hop update` already solved this problem with a small, well-scoped pattern: detect Homebrew install via `/Cellar/` path check, query `brew info --json=v2` for the latest version, run `brew upgrade <fully-qualified-formula>`. Cloning that pattern (rather than inventing a new one) keeps the two CLIs ergonomically symmetric and avoids re-deriving design choices already validated in hop. The fully-qualified formula name (`sahil87/tap/idea`) is required to disambiguate against any same-named core formula, mirroring hop's choice.

## What Changes

### New subcommand: `idea update`

Add a top-level cobra subcommand that self-updates the `idea` binary when it was installed via Homebrew, or prints a manual-update hint otherwise.

**Behavior** (mirrors `hop update` exactly):

1. Detect whether the running binary lives under `/Cellar/` (the canonical Homebrew install signature). If not, print `idea {version} was not installed via Homebrew.` followed by `Update manually, or reinstall with: brew install sahil87/tap/idea`, then exit 0.
2. If brew-installed:
   - Print `Current version: {version}` and `Checking for updates...`.
   - Run `brew update --quiet` (30s timeout).
   - Run `brew info --json=v2 sahil87/tap/idea` (30s timeout) and parse `formulae[0].versions.stable`.
   - Compare against the binary's `version` (strip a single leading `v` from each side before equality check — same as hop).
   - If equal: print `Already up to date ({version}).` and exit 0.
   - If different: print `Updating {current} → v{latest}...`, run `brew upgrade sahil87/tap/idea` (foreground, 120s timeout, child stdio inherited so brew's progress stream is visible), then print `Updated to v{latest}.` on success.
3. If `brew` is missing on `PATH` at any step: write `idea update: brew not found on PATH.` to stderr and exit non-zero (silenced by the cobra wrapper so it isn't double-printed).

**Constants** (literal copies from hop, name-substituted):

```go
const brewFormula = "sahil87/tap/idea"
const (
    brewUpdateTimeout  = 30 * time.Second
    brewInfoTimeout    = 30 * time.Second
    brewUpgradeTimeout = 120 * time.Second
)
```

### Code layout

Following Constitution Principles III/IV (cobra-idiomatic, logic in `internal/`):

- **`src/cmd/idea/update.go`** — cobra factory `updateCmd()` that calls into the internal package. Mirrors hop's `cmd/hop/update.go` but uses `os/exec` error mapping inline (idea has no `internal/proc` package — see "Impact" below).
- **`src/internal/idea/update.go`** — `func Update(currentVersion string, out, errOut io.Writer) error` housing the actual logic (brew detection, version query, upgrade orchestration). Mirrors hop's `internal/update/update.go` 1:1 in structure, with subprocess calls via stdlib `os/exec` rather than hop's `internal/proc` wrapper.
- **`src/internal/idea/update_test.go`** — table-driven tests for `normalizeVersion` (mirrors hop's test cases), a smoke test that `isBrewInstalled` returns a bool without panicking, and a `TestUpdateNonBrewInstall` test that asserts the non-brew code path prints the expected hint and returns nil. Skipped automatically when the test binary happens to be brew-installed.

### Wiring

In `src/cmd/idea/main.go`, add `updateCmd()` to `root.AddCommand(...)` alongside the existing seven subcommands.

### Help/version text

`Use: "update"`, `Short: "self-update the idea binary via Homebrew"`, `Args: cobra.NoArgs` (matches hop). The `version` package-level variable already exists in `main.go` and is `-ldflags`-stamped at build time — pass it directly to the internal package.

## Affected Memory

- `cli/structure.md`: (modify) Add the new `update` subcommand to the cobra subcommand inventory and add a one-line description of its self-update behavior.
- `cli/update.md`: (new) New memory file documenting the self-update flow — Homebrew detection logic, the `sahil87/tap/idea` formula reference, the version-comparison rule (strip single leading `v`), and the three timeout constants. Cross-reference `release/pipeline.md` for the source of truth on how the formula is published.

## Impact

**Code areas**:
- `src/cmd/idea/main.go` — append `updateCmd()` to the `AddCommand` list.
- `src/cmd/idea/update.go` — new file (cobra factory + error mapping).
- `src/internal/idea/update.go` — new file (logic).
- `src/internal/idea/update_test.go` — new file (tests).

**APIs**:
- New public CLI surface: `idea update`. No flag changes on root. No changes to existing subcommands.
- New exported function in `internal/idea`: `Update(currentVersion string, out, errOut io.Writer) error`. (Existing public API in `internal/idea` is untouched.)

**Dependencies**:
- No new module dependencies. Uses stdlib (`context`, `encoding/json`, `errors`, `fmt`, `io`, `os`, `os/exec`, `path/filepath`, `strings`, `time`) plus existing `cobra`. Honours Constitution "Dependency Discipline".

**Subprocess routing**:
- Idea has no `internal/proc` indirection — it already uses `os/exec` directly elsewhere (`internal/idea/idea.go` line 9). The implementation will use `exec.CommandContext` directly. The `brew upgrade` step needs streaming stdio for tty-aware progress; we'll attach `cmd.Stdin/Stdout/Stderr = os.Stdin/Stdout/Stderr` for that one call, matching hop's `proc.RunForeground` semantics.

**Out of scope** (explicit non-goals):
- A full version-bump check via semver comparison (hop uses simple string equality after `v`-stripping; we mirror that).
- Auto-update on every invocation, scheduled checks, telemetry — not in scope.
- Non-Homebrew install channels (apt, pacman, manual `scripts/install.sh`) — for these the existing "not installed via Homebrew" hint is sufficient.
- Rollback / pinning — Homebrew handles that natively.

## Open Questions

(None — all design decisions transfer directly from `hop update`.)

## Clarifications

### Session 2026-05-08 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 8 | Confirmed | — |
| 9 | Confirmed | — |
| 10 | Confirmed | — |
| 11 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Formula reference is `sahil87/tap/idea` (fully qualified) | Confirmed by `docs/specs/overview.md` and `README.md` install instructions; the same fully-qualified pattern hop uses for the same disambiguation reason | S:95 R:95 A:95 D:95 |
| 2 | Certain | Logic lives in `internal/idea/update.go`, cobra wrapper in `cmd/idea/update.go` | Constitution Principle IV mandates logic in `internal/idea` with thin cobra wrappers in `cmd/` | S:95 R:90 A:95 D:95 |
| 3 | Certain | No new module dependencies — stdlib + cobra only | Constitution Dependency Discipline; hop's update implementation already uses only stdlib (its `internal/proc` is itself a stdlib `os/exec` wrapper) | S:95 R:95 A:95 D:95 |
| 4 | Certain | Subcommand wiring goes in `main.go`'s existing `root.AddCommand(...)` block | That's the established pattern for the seven existing subcommands | S:95 R:95 A:95 D:95 |
| 5 | Certain | Use `os/exec` directly rather than introducing an `internal/proc` package | Clarified — user confirmed | S:95 R:75 A:80 D:80 |
| 6 | Certain | Version comparison strips a single leading `v` and uses string equality (no semver parsing) | Clarified — user confirmed | S:95 R:80 A:85 D:85 |
| 7 | Certain | Timeout constants copied verbatim from hop (30s update, 30s info, 120s upgrade) | Clarified — user confirmed | S:95 R:80 A:85 D:85 |
| 8 | Certain | Use `cobra.NoArgs` and short string `"self-update the idea binary via Homebrew"` | Clarified — user confirmed | S:95 R:90 A:85 D:90 |
| 9 | Certain | New memory file `cli/update.md` is created (rather than appending to `cli/structure.md`) | Clarified — user confirmed | S:95 R:80 A:80 D:75 |
| 10 | Certain | Cobra `RunE` maps a `brew not found` error to a silent-exit sentinel | Clarified — user confirmed | S:95 R:75 A:80 D:80 |
| 11 | Certain | Tests mirror hop's update tests (normalizeVersion table, isBrewInstalled smoke, non-brew code path assertion) | Clarified — user confirmed | S:95 R:85 A:85 D:80 |

11 assumptions (11 certain, 0 confident, 0 tentative, 0 unresolved).
