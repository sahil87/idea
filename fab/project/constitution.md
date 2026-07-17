# idea Constitution

## Core Principles

### I. Plain-Text Backlog as Source of Truth
The backlog file (default `fab/backlog.md`, overridable via `--file` or `IDEAS_FILE`) MUST remain a human-readable Markdown checklist. The line format `- [ ] [id] YYYY-MM-DD: text` SHALL be stable across versions — external scripts and human edits are first-class consumers. Round-trip parsing MUST preserve non-idea lines verbatim (headers, blank lines, prose between items).

**Rationale**: The tool exists to reduce friction over hand-editing a markdown file. Any drift in format breaks that contract.

### II. Worktree-Aware by Default, Main-Worktree Opt-In
All commands SHALL operate on the **current worktree's** backlog by default. The `--main` flag opts into the main worktree's backlog. Resolution MUST use `git rev-parse --path-format=absolute --git-common-dir` (parent dir) for the main repo and `git rev-parse --show-toplevel` for the current worktree — never environment variables or heuristics.

**Rationale**: Idea capture during exploratory work happens in linked worktrees; cross-worktree pollution is a footgun.

### III. Cobra-Idiomatic CLI Surface
Subcommands MUST be implemented as `*cobra.Command` factory functions (e.g., `addCmd()`) so they are independently testable and composable. The root command SHALL expose the bare-text shorthand (`idea <text>` → `idea add <text>`) but SHALL NOT add behavior beyond delegation. Persistent flags (`--file`, `--main`) MUST be defined on root and inherited.

**Rationale**: Keeps subcommand files thin, enables `RunE` testing, and prevents accidental coupling.

### IV. Logic Lives in `internal/idea`, Not in `cmd/`
Parsing, formatting, ID generation, file I/O, and worktree resolution MUST live in the `internal/idea` package. The `cmd/` package SHALL contain only flag wiring, argument validation, and output formatting. No business logic in cobra `RunE` bodies beyond a few lines of orchestration.

**Rationale**: Forces a testable seam — `internal/idea` is unit-tested without spawning subprocesses or mocking cobra.

### V. Table-Driven Tests, No Mocks for Filesystem
Tests SHALL use table-driven patterns (`tests := []struct { name string; ... }{ ... }`) for case enumeration. Filesystem behavior MUST be tested against real temp directories (`t.TempDir()`), not mocked interfaces. Git-dependent behavior MAY be tested via `exec.Command` against a real repo created in the test setup.

**Rationale**: The tool is small and I/O-heavy; mocks would test the mocks, not the behavior.

### VI. Stable IDs, Stable Output
Idea IDs MUST be exactly 4 lowercase alphanumeric chars (`[a-z0-9]{4}`) and MUST remain unique within a single backlog file. JSON output (when added) SHALL include `id`, `date`, `status`, `text` fields in that order with explicit `status: "open"|"done"` (never the boolean `done` field). Output formats are part of the public contract.

**Rationale**: External tooling (shell pipelines, fab integration) depends on stable schemas.

## Additional Constraints

### Test Integrity
Tests MUST conform to the implementation spec — never the other way around. When tests fail, the fix SHALL either (a) update the tests to match the spec, or (b) update the implementation to match the spec. Modifying implementation code solely to accommodate test fixtures or test infrastructure is prohibited. Specs are the source of truth; tests verify conformance to specs.

### Build Reproducibility
Builds MUST be reproducible via `go build` from a clean checkout with no environment-specific flags beyond `CGO_ENABLED=0`, `GOOS`, and `GOARCH`. Version stamping (when added) SHALL use `-ldflags '-X main.version=...'` only — no build-time codegen, no embedded timestamps.

### Dependency Discipline
Direct dependencies SHOULD be limited to the standard library plus `github.com/spf13/cobra`. New dependencies require justification in the change spec — "convenience" is not justification.

### Toolkit Standards
This tool is part of the sahil87 toolkit and MUST conform to the toolkit's published standards. The standards are enumerated by running `shll standards` — each entry names what it governs; read one with `shll standards <name>`. Before changing the CLI surface, help output, README.md, or docs/site/, the change MUST be checked against the standards governing that surface. If shll is unavailable, the canonical sources are the sahil87/shll repository's docs/site/standards/ tree (rendered on https://shll.ai). Standards added or revised there bind this repo without further amendment to this constitution.

## Governance

**Version**: 1.1.0 | **Ratified**: 2026-05-03 | **Last Amended**: 2026-07-18
