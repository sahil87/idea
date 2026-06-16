# Plan: System-Level Backlog & Out-of-Git Operation

**Change**: 260613-2b3m-system-level-backlog
**Intake**: `intake.md`

## Requirements

### Path Resolution: Out-of-Git Fallback & System Backlog

#### R1: System backlog path helper
`internal/idea` SHALL expose a `SystemBacklogPath() (string, error)` function that returns the system-level backlog file path. It MUST honor `$XDG_CONFIG_HOME` when set (`$XDG_CONFIG_HOME/idea/backlog.md`) and otherwise fall back to `~/.config/idea/backlog.md`, using Go stdlib `os.UserConfigDir` (no new dependencies).

- **GIVEN** `XDG_CONFIG_HOME` is set to `/custom/cfg`
- **WHEN** `SystemBacklogPath()` is called
- **THEN** it returns `/custom/cfg/idea/backlog.md`
- **AND GIVEN** `XDG_CONFIG_HOME` is unset and `HOME` is `/home/u`
- **WHEN** `SystemBacklogPath()` is called
- **THEN** it returns `/home/u/.config/idea/backlog.md`

#### R2: Resolution precedence
`resolveFile()` (in `cmd/idea`) SHALL resolve the backlog path by this precedence (first match wins): (1) `--system` flag → system backlog, skipping git entirely; (2) `--file <path>` / `IDEAS_FILE` env → joined to the git root when in a repo, else to the system config dir; (3) `--main` → main worktree root (git-only, errors outside git, UNCHANGED); (4) in a git repo with no override → `{worktree-root}/fab/backlog.md` (UNCHANGED default); (5) outside a git repo with no override → system backlog (the NEW graceful fallback).

- **GIVEN** the CWD is not inside any git repository and no flags/env are set
- **WHEN** any backlog command runs
- **THEN** it operates on `SystemBacklogPath()` instead of failing with "not in a git repository"
- **AND GIVEN** the CWD is inside a git repo with no override
- **WHEN** a command runs
- **THEN** it resolves to `{worktree-root}/fab/backlog.md` exactly as before

#### R3: `--system` persistent flag
The root command SHALL define a persistent `--system` boolean flag (peer of `--main`). When set, every command operates on the system backlog regardless of CWD (including inside a git repo).

- **GIVEN** the CWD is inside a git repo
- **WHEN** `idea --system "global todo"` runs
- **THEN** the idea is written to the system backlog, not the repo backlog
- **AND GIVEN** any directory
- **WHEN** `idea --system list` runs
- **THEN** it lists the system backlog

#### R4: `--system` / `--main` conflict
Specifying both `--system` and `--main` SHALL be a user error: the command MUST print a clear error and exit non-zero, performing no backlog operation.

- **GIVEN** both `--system` and `--main` are passed
- **WHEN** any command runs
- **THEN** it exits non-zero with a message naming the conflict and writes nothing to any backlog

#### R5: Out-of-git `--file` / `IDEAS_FILE` rooting
When outside a git repo, a relative `--file`/`IDEAS_FILE` value SHALL be joined to the system config dir (`$XDG_CONFIG_HOME/idea` or `~/.config/idea`); an absolute value SHALL be used as-is. Inside a git repo the rooting is unchanged (joined to the git root).

- **GIVEN** the CWD is outside git and `--file notes.md` is passed
- **WHEN** a command runs
- **THEN** the path resolves to `{config-dir}/idea/notes.md`
- **AND GIVEN** `--file /abs/path.md` is passed (in or out of git)
- **THEN** the path resolves to `/abs/path.md`

#### R6: On-demand config dir creation
The system config dir (`~/.config/idea/`) SHALL be created on demand (`mkdir -p` semantics) on the first mutating write, rather than erroring "no such directory". Read-only commands on a non-existent system backlog behave like read-only commands on any missing file (no error beyond the existing "no ideas file yet" path).

- **GIVEN** `~/.config/idea/` does not exist
- **WHEN** `idea --system "first idea"` runs
- **THEN** the directory is created and the idea is written

#### R7: `--main` stays git-only (unchanged)
`--main` SHALL continue to require a git repo and error with "not in a git repository" when run outside one. Its behavior is entirely unchanged.

- **GIVEN** the CWD is outside any git repo
- **WHEN** `idea --main "x"` runs
- **THEN** it errors with "not in a git repository" and exits non-zero

#### R8: File format and CRUD/fmt semantics unchanged
The backlog file format, ID rules, and all CRUD/`fmt`/`list`/`show` semantics SHALL be identical regardless of which path was resolved. Only path resolution changes.

- **GIVEN** the system backlog is the resolved path
- **WHEN** any command operates on it
- **THEN** the line format, escaping, canonical-write, and JSON output are byte-for-byte the same as for a repo backlog

### Non-Goals

- Changing the backlog line format or any CRUD/fmt behavior — only path resolution changes.
- A per-directory (`./fab/backlog.md` in CWD) out-of-git mode — rejected at intake in favor of a single system backlog.
- Auto-creating the system dir on read-only commands — creation is write-triggered only.

### Design Decisions

1. **Out-of-git default is the system backlog**: graceful fallback over erroring or CWD-local. — *Why*: zero-friction capture anywhere; predictable single global list. — *Rejected*: CWD-local `./fab/backlog.md` (scatters fab/ folders); required-flag erroring (defeats the goal).
2. **`--system` is a persistent flag, peer of `--main`**: reachable from inside a repo. — *Why*: mirrors the existing `--main` pattern; no `cd` required. — *Rejected*: implicit-only out-of-git system backlog.
3. **`--system` + `--main` is a hard conflict error**: two explicit root selectors disagreeing fail loudly. — *Why*: no signal for a silent winner. — *Rejected*: silent precedence.
4. **Out-of-git relative `--file`/`IDEAS_FILE` roots at the system config dir**: one consistent non-git anchor. — *Why*: keeps a single anchor coherent with the system-backlog model. — *Rejected*: rooting at CWD.
5. **Resolution centralized in `internal/idea`**: `resolveFile()` in `cmd/` only wires flags. — *Why*: Constitution IV (logic in `internal/idea`). The new `ResolveBacklogPath` helper owns precedence; `cmd/` passes flag values in.

## Tasks

### Phase 1: Core Implementation (internal/idea)

- [x] T001 Add `SystemBacklogPath() (string, error)` to `src/internal/idea/idea.go` using `os.UserConfigDir` (honors `XDG_CONFIG_HOME`, falls back to `~/.config`), returning `{configDir}/idea/backlog.md`. <!-- R1 -->
- [x] T002 Add a centralized resolution helper to `src/internal/idea/idea.go` — `ResolveBacklogPath(systemFlag, mainFlag bool, fileFlag string) (string, error)` — encoding the full precedence (system → file/env rooted at git-root-or-config-dir → main → in-git default → out-of-git system fallback), including the `--system`/`--main` conflict error and the out-of-git `--file`/`IDEAS_FILE` rooting at the config dir. Absolute `--file`/`IDEAS_FILE` values bypass rooting. <!-- R2 R4 R5 R7 -->
- [x] T003 Ensure on-demand config-dir creation on the first mutating write: add `os.MkdirAll(filepath.Dir(path), ...)` to `atomicWriteFile` in `src/internal/idea/idea.go` so SaveFile-based mutations (done/reopen/edit/rm/prune/fmt) create a missing system dir; `Add` already does this. <!-- R6 -->

### Phase 2: Flag Wiring (cmd/idea)

- [x] T004 Register a `--system` persistent bool flag on root in `src/cmd/idea/main.go` (peer of `--main`); update the `--file` flag description to note out-of-git rooting at the config dir. <!-- R3 -->
- [x] T005 Rewrite `resolveFile()` in `src/cmd/idea/resolve.go` to delegate to `idea.ResolveBacklogPath(systemFlag, mainFlag, fileFlag)`, keeping `cmd/` free of resolution logic (Constitution IV). <!-- R2 R3 R4 -->

### Phase 3: Tests

- [x] T006 [P] Add table-driven unit tests in `src/internal/idea/idea_test.go` for `SystemBacklogPath` (XDG set vs. unset via `t.Setenv`) and `ResolveBacklogPath` (all precedence branches incl. conflict error, in-git vs out-of-git file rooting, absolute path bypass). Use `t.TempDir()` + `t.Setenv` for HOME/XDG; gate git-root cases behind a real temp repo or inject the root. <!-- R1 R2 R4 R5 R7 -->
- [x] T007 Add integration tests in `src/cmd/idea/main_test.go`: out-of-git add+list falls back to the system backlog (run binary from a non-git temp dir with HOME/XDG_CONFIG_HOME set via `cmd.Env`); `--system` inside a repo targets the system backlog not the repo backlog; `--system --main` conflict exits non-zero; on-demand dir creation (config dir absent before first `--system add`). <!-- R3 R4 R6 R8 -->

### Phase 4: Docs & Help

- [x] T008 Update per-command `Long` help strings (`add.go`, `list.go`, `show.go`, `done.go`, `reopen.go`, `edit.go`, `rm.go`, `prune.go`, `fmt.go` in `src/cmd/idea/`) to mention `--system` / the system backlog alongside the existing `--main` / `--file` note. <!-- R3 R8 -->
- [x] T009 [P] Update `docs/specs/overview.md` "Worktree Behavior" section to document the resolution precedence, the `--system` flag, the out-of-git system fallback, and the `~/.config/idea/backlog.md` XDG location. <!-- R2 R3 -->

## Execution Order

- T001 → T002 (resolution helper uses SystemBacklogPath)
- T002, T003 → T005 (resolveFile delegates to the new helper)
- T004 (flag declaration) → T005, T007 (need the flag var)
- T006 depends on T001, T002; T007 depends on T004, T005, T003
- T008, T009 are independent docs tasks (parallelizable)

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `SystemBacklogPath()` returns `$XDG_CONFIG_HOME/idea/backlog.md` when set and `~/.config/idea/backlog.md` otherwise, via `os.UserConfigDir`.
- [ ] A-002 R2: `resolveFile()`/`ResolveBacklogPath` implements the full first-match-wins precedence (system, file/env, main, in-git default, out-of-git fallback).
- [ ] A-003 R3: A `--system` persistent flag exists on root and forces the system backlog from anywhere, including inside a repo.
- [ ] A-004 R6: The system config dir is created on demand on the first mutating write.
- [ ] A-005 R7: `--main` still requires git and errors "not in a git repository" outside a repo.

### Behavioral Correctness

- [ ] A-006 R2: Inside a git repo with no override, resolution is unchanged (`{worktree-root}/fab/backlog.md`).
- [ ] A-007 R5: Out-of-git relative `--file`/`IDEAS_FILE` roots at the config dir; absolute values pass through unchanged.

### Edge Cases & Error Handling

- [ ] A-008 R4: `--system --main` together exits non-zero with a clear conflict message and writes nothing.
- [ ] A-009 R2: Outside git with no override, commands no longer fail with "not in a git repository" — they use the system backlog.

### Scenario Coverage

- [ ] A-010 R3 R6 R8: Integration tests cover out-of-git fallback, `--system` inside a repo, the conflict error, and on-demand dir creation; format/CRUD on the system path is identical to a repo path.

### Code Quality

- [ ] A-011 Pattern consistency: New code follows surrounding naming and structural patterns; resolution logic lives in `internal/idea`, only flag wiring in `cmd/` (Constitution IV); git resolution still uses `git rev-parse` (Constitution II).
- [ ] A-012 No unnecessary duplication: Reuses `WorktreeRoot`/`MainRepoRoot`/`ResolveFilePath`/`os.UserConfigDir`/`atomicWriteFile` rather than reimplementing; no new dependencies (stdlib + cobra only).
- [ ] A-013 Magic strings: The `idea`/`backlog.md`/`.config` path segments are introduced consistently without scattering unnamed literals where a named helper is clearer.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Precedence centralized in a new `internal/idea.ResolveBacklogPath(systemFlag, mainFlag, fileFlag)` helper; `resolveFile()` in `cmd/` only forwards flag values | Constitution IV mandates logic in `internal/idea`; the existing `resolveFile` already orchestrated, so this is a natural extension. Cheap to reshape if review prefers a different signature | S:70 R:75 A:80 D:70 |
| 2 | Confident | On-demand dir creation is added to `atomicWriteFile` (the single SaveFile serialization point); `Add` already MkdirAll's its own path | Single seam covers all SaveFile-based mutations without scattering MkdirAll across each op; matches the existing `Add` precedent. Reversible | S:65 R:80 A:75 D:75 |
| 3 | Confident | The `--system`/`--main` conflict is detected inside the resolution helper and returned as an error (surfaced via the existing `ERROR:` top-level handler), not via a separate cobra PreRun | Keeps the conflict check colocated with the precedence it guards; reuses the existing non-zero-exit error path. Easy to move to a PreRunE if review prefers earlier rejection | S:60 R:75 A:70 D:65 |
| 4 | Confident | Out-of-git tests drive the built binary with `cmd.Env` setting HOME/XDG_CONFIG_HOME and run from a non-git temp dir; unit tests use `t.Setenv` | Constitution V (real temp dirs, no FS mocks) + the existing `buildBinary`/`setupGitRepo` integration-test pattern; `exec.Command` needs explicit `cmd.Env` to isolate HOME/XDG | S:70 R:80 A:80 D:75 |
| 5 | Confident | Constitution Principle II is NOT amended in this change (spec/overview note only); flagged non-blocking at intake | Intake assumption 10 left this a judgment call and explicitly non-blocking; the change does not violate II (git resolution stays the in-repo default). Trivially revisited later | S:55 R:70 A:60 D:60 |

5 assumptions (0 certain, 5 confident, 0 tentative).
