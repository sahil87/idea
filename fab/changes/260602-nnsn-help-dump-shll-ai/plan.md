# Plan: Build-time help-dump → shll.ai command reference

**Change**: 260602-nnsn-help-dump-shll-ai
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md. The JSON shape is a FROZEN cross-repo contract
     (sahil87/shll.ai help/wt.json is the reference truth). -->

### Producer: hidden `help-dump` subcommand

#### R1: Root command factory extraction
The `idea` binary SHALL build its root command via a `newRootCmd() *cobra.Command` factory that constructs the root command and registers ALL subcommands (the existing ones plus `helpDumpCmd()`). `main()` SHALL call `newRootCmd().Execute()` with the existing error handling (`errSilent` sentinel + `os.Exit(1)`). The `var version = "dev"` declaration and `Version: version` wiring MUST remain intact.

- **GIVEN** the refactored `main.go`
- **WHEN** `main()` runs
- **THEN** it builds the command via `newRootCmd()` and executes it with the existing silent-error handling
- **AND** the producer subcommand, when run, walks `cmd.Root()` to see the identical live tree

#### R2: Hidden `help-dump` subcommand exists
The binary SHALL expose a `helpDumpCmd() *cobra.Command` with `Hidden: true`, `Use: "help-dump"`, and a short description indicating it emits the CLI help tree as JSON for build tooling. It MUST be registered in `newRootCmd()`.

- **GIVEN** the registered root command
- **WHEN** the command tree is walked
- **THEN** `help-dump` is present in the tree but `Hidden: true`
- **AND** it never appears in user-facing help nor in its own dump output (excluded by the Hidden filter)

#### R3: Programmatic cobra-tree walk → JSON envelope
`help-dump`'s RunE SHALL walk `cmd.Root()` recursively (NOT regex on `-h`) and marshal a frozen-contract envelope to the command's `OutOrStdout()`. The envelope is `helpDump{Tool, Version, CapturedAt, SchemaVersion, Root}` where each node is `helpNode{Name, Path, Short, Usage, Text, Commands}` in that exact field/JSON order.

- **GIVEN** the running `help-dump` command
- **WHEN** RunE executes
- **THEN** it emits `Tool == "idea"`, `Version == cmd.Root().Version`, `CapturedAt == time.Now().UTC().Format(time.RFC3339)`, `SchemaVersion == 1`, `Root == buildNode(cmd.Root())`
- **AND** the output is produced by `json.MarshalIndent(dump, "", "  ")` followed by a trailing newline, written to `cmd.OutOrStdout()`, returning `nil`

#### R4: Per-node field composition
`buildNode` SHALL populate each node with `Name = cmd.Name()`, `Path = cmd.CommandPath()`, `Short = cmd.Short`, `Usage = cmd.UseLine()`, and `Text = longOrShort(cmd) + "\n\n" + cmd.UsageString()` where `longOrShort` returns `cmd.Long` if non-empty else `cmd.Short`. If BOTH `Long` and `Short` are empty, `Text` MUST be just `cmd.UsageString()` (no leading blank lines).

- **GIVEN** a command node with a non-empty `Long`
- **WHEN** `buildNode` runs
- **THEN** `Text` begins with the `Long` text, then a blank line, then the `Usage:`/`Flags:` blocks from `UsageString()`
- **AND** when `Long` is empty but `Short` is set, `Text` begins with `Short`
- **AND** when both are empty, `Text` is exactly `cmd.UsageString()` with no leading blank lines

#### R5: Recursion filter
During recursion, `buildNode` SHALL skip any child command where `c.Hidden == true` OR `c.Name() == "completion"` OR `c.Name() == "help"`. This also excludes `help-dump` itself (Hidden). The `Commands` slice MUST be initialized to `[]helpNode{}` (never nil) so leaf nodes serialize as `[]`, not `null`.

- **GIVEN** the root command with auto-generated `completion`/`help` and the hidden `help-dump`
- **WHEN** `buildNode(root)` runs
- **THEN** `completion`, `help`, and `help-dump` are absent from `root.commands`
- **AND** a leaf command serializes its `commands` as the JSON array `[]`, never `null`

### CI: dump + validate + PR step

#### R6: Release-pipeline publish step
`.github/workflows/release.yml` SHALL gain ONE new step, ordered AFTER the existing "Create GitHub Release" step (best-effort, last), that produces the dump from the stamped native `linux/amd64` binary, validates it parses as JSON, and opens a PR into `sahil87/shll.ai` touching ONLY `help/idea.json`.

- **GIVEN** a release run with the cross-compiled stamped binary at `dist/idea-linux-amd64/idea`
- **WHEN** the new step runs
- **THEN** it writes `help/idea.json` via `dist/idea-linux-amd64/idea help-dump`, validates with `python3 -c "import json; json.load(...)"`, clones shll.ai with `SHLLAI_TOKEN`, stages only `help/idea.json`, and (when changed) commits as author `sahil87`/`sahil@noon.design` on branch `help-dump/idea-${version}` and runs `gh pr create`
- **AND** it does NOT call `gh pr merge` (shll.ai's `help-automerge.yml` owns merging)
- **AND** when `help/idea.json` is unchanged (`git diff --quiet`) it exits 0 without opening a PR

### Tests

#### R7: In-process contract test
`src/cmd/idea/help_dump_test.go` SHALL execute the command in-process via `newRootCmd()` with stdout captured into a buffer and args `["help-dump"]`, then unmarshal and assert the full contract.

- **GIVEN** a `newRootCmd()` with `SetOut(&bytes.Buffer{})` and args `["help-dump"]`
- **WHEN** `Execute()` runs and the captured JSON is unmarshaled
- **THEN** `tool == "idea"`, `schema_version == 1`, `version` is non-empty, `captured_at` parses as RFC3339, `root.name == "idea"`, `root.path == "idea"`
- **AND** `completion`, `help`, and `help-dump` are ABSENT from `root.commands`
- **AND** each real subcommand (`add`, `list`, `show`, `done`, `reopen`, `edit`, `rm`, `update`, `shell-init`) is present with correct `path` (e.g. `idea add`)
- **AND** a leaf node serializes `commands` as `[]` (asserted on raw JSON), and representative `text`/`usage`/`short` are non-empty

### Non-Goals
- The shll.ai site-side consumer (Astro loader + reference UI, Zod schema, `help-automerge.yml`) — owned by the shll.ai repo, out of scope here.
- A `--output` file flag on the subcommand — output is stdout-only; CI owns file placement.
- Any change to `internal/idea` — this is cmd-layer cobra-tree serialization, no backlog logic.
- A push-to-`main` dump variant — release-only (the version stamp only exists in the release pipeline).

### Design Decisions
1. **Producer is a hidden in-binary subcommand**: reuses the exact live cobra tree and the ldflags-stamped `Version` — *Why*: both are only available inside the binary's command construction — *Rejected*: an external Go program (would duplicate the tree and could not read the stamped version).
2. **Programmatic tree walk, not regex on `-h`**: cobra exposes the graph as data — *Why*: exact and stable — *Rejected*: regex-scraping rendered help (fragile, re-derives what cobra models).
3. **PR with no `gh pr merge`, not direct push**: serializes multi-repo writes and defers merging to shll.ai's receiving workflow — *Why*: avoids the multi-repo push race and the actor/content/schema guards live on the receiving side — *Rejected*: direct push to `main` (races on non-fast-forward); `gh pr merge --auto` from idea (would fail; `allow_auto_merge` is false at native level).

## Tasks

### Phase 1: Core Implementation

- [x] T001 Extract `newRootCmd() *cobra.Command` factory in `src/cmd/idea/main.go`: move inline root construction + persistent-flag registration + `AddCommand(...)` into the factory (adding `helpDumpCmd()` to the registration list); have `main()` call `newRootCmd().Execute()` with the existing `errSilent`/`os.Exit(1)` handling; keep `var version` and `Version: version` intact <!-- R1 -->
- [x] T002 Create `src/cmd/idea/help_dump.go`: define `helpNode`/`helpDump` structs, `longOrShort` helper, `buildNode` recursive walk with the Hidden/completion/help filter and `[]helpNode{}` init, and `helpDumpCmd() *cobra.Command` (`Hidden: true`, `Use: "help-dump"`) whose RunE marshals the envelope with `json.MarshalIndent(dump, "", "  ")` + trailing newline to `OutOrStdout()` <!-- R2 R3 R4 R5 --> <!-- reworked: A-008 — buildNode now calls InitDefaultHelpFlag()/InitDefaultVersionFlag() before UsageString() so every node's text Flags block carries -h, --help (and root carries -v, --version), matching the frozen wt.json -h output -->

### Phase 2: Integration

- [x] T003 Add the "Dump help tree and PR to shll.ai" step to `.github/workflows/release.yml` after the "Create GitHub Release" step, verbatim per intake What Changes §2 (env `GH_TOKEN: ${{ secrets.SHLLAI_TOKEN }}`; produce + validate + clone + commit-only-`help/idea.json` + `gh pr create`; no-op guard; NO `gh pr merge`) <!-- R6 -->

### Phase 3: Tests

- [x] T004 Create `src/cmd/idea/help_dump_test.go`: table-driven where natural; execute `newRootCmd()` in-process with `SetOut(&bytes.Buffer{})` + args `["help-dump"]`; assert envelope fields, root name/path, filter exclusions, every real subcommand present with correct `path`, leaf `commands: []` on raw JSON, and non-empty representative `text`/`usage`/`short` <!-- R7 --> <!-- reworked: A-006 — added TestHelpDump_TextContainsDefaultFlags locking the contract: root.text contains both -h, --help and -v, --version; a leaf's text contains -h, --help (and not -v, --version) -->

## Execution Order

- T001 blocks T002 (the factory must exist before `helpDumpCmd()` is registered) and T004 (the test calls `newRootCmd()`).
- T002 blocks T004 (the test exercises the command).
- T003 is independent of the Go code and may run in parallel.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `newRootCmd()` exists in `main.go`, registers all existing subcommands plus `helpDumpCmd()`, and `main()` calls `newRootCmd().Execute()` with the existing `errSilent`/`os.Exit(1)` handling; `var version`/`Version: version` intact
- [x] A-002 R2: `helpDumpCmd()` exists with `Hidden: true` and `Use: "help-dump"`, registered in `newRootCmd()`
- [x] A-003 R3: running `help-dump` emits `tool == "idea"`, `version == cmd.Root().Version`, `captured_at` as RFC3339 UTC, `schema_version == 1`, and `root` built from `buildNode(cmd.Root())`, via `json.MarshalIndent(dump, "", "  ")` + trailing newline to `OutOrStdout()`
- [x] A-004 R4: each node's `text` is `longOrShort(cmd) + "\n\n" + UsageString()` (Long preferred, else Short, else bare `UsageString()`); `name`/`path`/`short`/`usage` map to `Name()`/`CommandPath()`/`Short`/`UseLine()`
- [x] A-005 R6: `release.yml` has one new step after "Create GitHub Release" that dumps from `dist/idea-linux-amd64/idea`, validates JSON, clones shll.ai with `SHLLAI_TOKEN`, commits only `help/idea.json` as `sahil87`, and `gh pr create`s (no `gh pr merge`)
- [x] A-006 R7: `help_dump_test.go` executes via `newRootCmd()` and asserts the envelope, root, filters, subcommand presence/paths, leaf `[]`, and non-empty representative fields

### Behavioral Correctness

- [x] A-007 R5: `completion`, `help`, and `help-dump` are absent from `root.commands`; leaf nodes serialize `commands` as `[]` not `null` (verified on raw JSON)
- [x] A-008 R3: the produced JSON structure matches the frozen `help/wt.json` reference (key names, ordering, 2-space indentation, `text` composition)

### Edge Cases & Error Handling

- [x] A-009 R4: a command with both `Long` and `Short` empty emits `text` equal to `UsageString()` with no leading blank lines (defensive guard)
- [x] A-010 R6: when `help/idea.json` is unchanged, the CI step exits 0 without opening a PR (no-op guard)

### Code Quality

- [x] A-011 Pattern consistency: `help_dump.go` follows the `*cobra.Command` factory pattern and file structure of sibling subcommand files; `gofmt`-clean
- [x] A-012 No unnecessary duplication: reuses `cmd.Root()`/cobra accessors and the existing error-handling seam; no reimplementation of tree traversal
- [x] A-013 Readability: `buildNode`/`longOrShort`/RunE are small and focused (no god functions); no magic strings beyond the contract literals (`"idea"`, `schema_version 1`)
- [x] A-014 Dependency discipline: no new module dependencies — only stdlib (`encoding/json`, `time`, `bytes`) and the already-present `cobra`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- Constitution Principle IV (logic in `internal/idea`) is intentionally not triggered: this is cobra-tree serialization in the cmd layer, not backlog logic (per intake Impact section).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Struct types named `helpNode`/`helpDump` (not `Node`/`Dump`), unexported in `package main` | Intake "exact contract" block names them `helpNode`/`helpDump`; constitution says keep types in-package unless cross-package test needs them — in-package test reaches them directly | S:95 R:90 A:90 D:90 |
| 2 | Certain | The shell-init subcommand registers as `shell-init` (`newShellInitCmd()`); test asserts `path == "idea shell-init"` | Confirmed by reading `shell_init.go` (`Use: "shell-init <shell>"`); `cmd.Name()` is `shell-init` | S:98 R:95 A:95 D:95 |
| 3 | Certain | 2-space `MarshalIndent` and `text = description\n\n + UsageString()` composition | Verified against the live frozen `help/wt.json` (root + leaves lead with full Long/Short, blank line, then `Usage:`; 2-space indent) | S:98 R:80 A:95 D:95 |
| 4 | Certain | Trailing newline after the marshaled JSON (matches `wt.json` which is newline-terminated and is conventional for stdout JSON) | Explicit in intake (§1 "then a trailing newline"); also matches reference file | S:95 R:90 A:90 D:90 |

4 assumptions (4 certain, 0 confident, 0 tentative).
