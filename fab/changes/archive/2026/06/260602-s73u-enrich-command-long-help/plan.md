# Plan: Enrich Cobra `Long` Help for idea's Commands

**Change**: 260602-s73u-enrich-command-long-help
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

> All requirements are additive help-text prose. No `RunE`, flag wiring, or
> `internal/idea` logic changes. Each `Long` is a backtick raw Go string literal
> placed immediately after `Short:` in the command's `*cobra.Command` literal,
> matching the prose style already in `main.go` / `shell_init.go` (short
> paragraphs, blank-line separation, an inline indented example). Cobra appends
> the `Usage:` / flag block automatically below `Long`, so the prose MUST NOT
> restate it.

### CLI Help: Backlog-Touching Commands

#### R1: `add` carries an enriched `Long`
`add.go` SHALL gain a `Long` describing that it appends a new idea (4-char ID +
today's date) to the current worktree's backlog, that `--id`/`--date` override
the generated values, a reference to worktree-vs-`--main`/`--file`/`IDEAS_FILE`
resolution, and a short example.

- **GIVEN** the built binary
- **WHEN** a user runs `idea add -h`
- **THEN** the enriched `Long` prose renders above the auto-generated `Usage:` block
- **AND** `Short` ("Add a new idea to the backlog") is byte-unchanged

#### R2: `list` carries an enriched `Long`
`list.go` SHALL gain a `Long` describing that it prints ideas from the current
worktree's backlog (open by default), documenting `--all/-a`, `--done`,
`--json`, `--sort`, `--reverse`, a worktree-vs-`--main` reference, and an example.

- **GIVEN** the built binary
- **WHEN** a user runs `idea list -h`
- **THEN** the enriched `Long` renders above `Usage:` and `Short` is unchanged

#### R3: `done` carries an enriched `Long`
`done.go` SHALL gain a `Long` describing that it marks a matching **open** idea
done, documenting `<query>` semantics (ID or case-insensitive substring;
ambiguous matches refused with the match list), a worktree-vs-`--main`
reference, and an example.

- **GIVEN** the built binary
- **WHEN** a user runs `idea done -h`
- **THEN** the enriched `Long` renders above `Usage:` and `Short` is unchanged

#### R4: `reopen` carries an enriched `Long`
`reopen.go` SHALL gain a `Long` describing that it reopens a matching **done**
idea, with the same `<query>` semantics, a worktree-vs-`--main` reference, and
an example.

- **GIVEN** the built binary
- **WHEN** a user runs `idea reopen -h`
- **THEN** the enriched `Long` renders above `Usage:` and `Short` is unchanged

#### R5: `edit` carries an enriched `Long`
`edit.go` SHALL gain a `Long` describing that it replaces an idea's text,
documenting `<query>` semantics, the `--id`/`--date` flags (change the matched
idea's ID/date), a worktree-vs-`--main` reference, and an example.

- **GIVEN** the built binary
- **WHEN** a user runs `idea edit -h`
- **THEN** the enriched `Long` renders above `Usage:` and `Short` is unchanged

#### R6: `rm` carries an enriched `Long`
`rm.go` SHALL gain a `Long` describing that it deletes a matching idea,
documenting `<query>` semantics, that `--force` is required to confirm the
deletion, a worktree-vs-`--main` reference, and an example.

- **GIVEN** the built binary
- **WHEN** a user runs `idea rm -h`
- **THEN** the enriched `Long` renders above `Usage:` and `Short` is unchanged

#### R7: `show` carries an enriched `Long`
`show.go` SHALL gain a `Long` describing that it prints a single matching idea,
documenting `<query>` semantics, the `--json` flag, a worktree-vs-`--main`
reference, and an example.

- **GIVEN** the built binary
- **WHEN** a user runs `idea show -h`
- **THEN** the enriched `Long` renders above `Usage:` and `Short` is unchanged

### CLI Help: Non-Backlog Command

#### R8: `update` carries an enriched `Long`
`update.go` SHALL gain a `Long` describing that it self-updates the `idea`
binary via Homebrew, documenting `--skip-brew-update`, with NO worktree note
(it does not touch the backlog), and an example.

- **GIVEN** the built binary
- **WHEN** a user runs `idea update -h`
- **THEN** the enriched `Long` renders above `Usage:` and `Short` is unchanged

### Non-Goals

- Editing `main.go` / `shell_init.go` — they already carry `Long`.
- Any `RunE`, flag-wiring, or `internal/idea` logic change.
- A regression test asserting `Long` presence — explicitly out per intake
  clarification (help text is not behavior).
- The fab-kit mirror — separate repo, follow-up only.

### Design Decisions

1. **Worktree-vs-`--main` is referenced, not re-derived**: each backlog-touching
   command's `Long` carries one consistent short sentence pointing at the
   root-defined `--main` / `--file` / `IDEAS_FILE` behavior — *Why*: the
   persistent flags are documented on root; duplicating the full resolution
   algorithm 7x invites drift — *Rejected*: a verbatim wall of resolution text
   per command.
2. **Filter nuance documented for `done`/`reopen`**: `done` matches only open
   ideas (`FilterOpen`), `reopen` only done ideas (`FilterDone`); `show`/`edit`/
   `rm` match across all (`FilterAll`) — *Why*: verified in `idea.go`; it is
   user-facing behavior a `-h` reader needs — *Rejected*: a generic "matches any
   idea" line that would be wrong for `done`/`reopen`.

## Tasks

### Phase 2: Core Implementation

- [x] T001 [P] Add `Long` to `src/cmd/idea/add.go` (behavior, `--id`/`--date`, worktree ref, example). <!-- R1 -->
- [x] T002 [P] Add `Long` to `src/cmd/idea/list.go` (behavior, `--all`/`--done`/`--json`/`--sort`/`--reverse`, worktree ref, example). <!-- R2 -->
- [x] T003 [P] Add `Long` to `src/cmd/idea/done.go` (marks open idea done, query semantics, worktree ref, example). <!-- R3 -->
- [x] T004 [P] Add `Long` to `src/cmd/idea/reopen.go` (reopens done idea, query semantics, worktree ref, example). <!-- R4 -->
- [x] T005 [P] Add `Long` to `src/cmd/idea/edit.go` (replaces text, query semantics, `--id`/`--date`, worktree ref, example). <!-- R5 -->
- [x] T006 [P] Add `Long` to `src/cmd/idea/rm.go` (deletes idea, query semantics, `--force` required, worktree ref, example). <!-- R6 -->
- [x] T007 [P] Add `Long` to `src/cmd/idea/show.go` (prints single idea, query semantics, `--json`, worktree ref, example). <!-- R7 -->
- [x] T008 [P] Add `Long` to `src/cmd/idea/update.go` (self-update via Homebrew, `--skip-brew-update`, no worktree note, example). <!-- R8 -->

### Phase 4: Polish

- [x] T009 From `src/`, run `gofmt -l ./cmd/idea` (no output = formatted) and `go build ./...` (must succeed). <!-- R1 R2 R3 R4 R5 R6 R7 R8 -->
- [x] T010 Spot-check rendered help: `go run ./cmd/idea {add,list,rm,show,update} -h` — confirm `Long` renders above `Usage:`. <!-- R1 R2 R6 R7 R8 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `add.go` has a `Long` covering behavior, `--id`/`--date`, worktree ref, and an example; `Short` unchanged.
- [x] A-002 R2: `list.go` has a `Long` covering behavior, all five flags, worktree ref, and an example; `Short` unchanged.
- [x] A-003 R3: `done.go` has a `Long` covering open-idea matching, query semantics, worktree ref, and an example; `Short` unchanged.
- [x] A-004 R4: `reopen.go` has a `Long` covering done-idea matching, query semantics, worktree ref, and an example; `Short` unchanged.
- [x] A-005 R5: `edit.go` has a `Long` covering text replacement, query semantics, `--id`/`--date`, worktree ref, and an example; `Short` unchanged.
- [x] A-006 R6: `rm.go` has a `Long` covering deletion, query semantics, `--force` requirement, worktree ref, and an example; `Short` unchanged.
- [x] A-007 R7: `show.go` has a `Long` covering single-idea print, query semantics, `--json`, worktree ref, and an example; `Short` unchanged.
- [x] A-008 R8: `update.go` has a `Long` covering Homebrew self-update, `--skip-brew-update`, no worktree note, and an example; `Short` unchanged.

### Behavioral Correctness

- [x] A-009 R1: No `RunE`, flag-registration, or `internal/idea` change in any of the 8 files — diff is `Long:`-field-only (plus gofmt).

### Scenario Coverage

- [x] A-010 R1: `go run ./cmd/idea add -h` (and `list`/`rm`/`show`/`update`) renders the enriched `Long` above the auto-generated `Usage:` block.

### Edge Cases & Error Handling

- [x] A-011 R3: `done`/`reopen` `Long` correctly states the open-vs-done filter nuance (not a generic "matches any idea").
- [x] A-012 R6: `rm` `Long` states `--force` is required to actually delete.

### Code Quality

- [x] A-013 Pattern consistency: `Long` literals are backtick raw strings in the established `main.go`/`shell_init.go` prose style; do not restate cobra's `Usage:`/flag block.
- [x] A-014 No unnecessary duplication: worktree-vs-`--main` resolution is referenced once per command, not duplicated as a full algorithm.
- [x] A-015 Build: `gofmt -l ./cmd/idea` reports nothing and `go build ./...` succeeds from `src/`.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Source lives at `src/cmd/idea/` (config.yaml `source_paths` lists a stale `src/go/idea`). | Intake and actual `find` of the tree both confirm `src/cmd/idea/`; the binary is built from there. | S:95 R:90 A:95 D:95 |
| 2 | Confident | `done`/`reopen` `Long` states the open-vs-done filter nuance; `show`/`edit`/`rm` say "open or done". | Verified in `idea.go`: `Done`→FilterOpen, `Reopen`→FilterDone, others→FilterAll. Accurate `-h` prose requires it. | S:85 R:80 A:90 D:80 |
| 3 | Confident | `Long` placed immediately after `Short:` inside each struct literal, before `Args:`. | Mirrors `main.go`/`shell_init.go` field ordering; one obvious placement. | S:80 R:85 A:90 D:85 |
| 4 | Certain | No test added; diff is help-text prose only. | Intake clarification 2026-06-02 resolved this explicitly. | S:95 R:75 A:75 D:60 |

4 assumptions (2 certain, 2 confident, 0 tentative).
