# Intake: Toolkit Standards Conformance

**Change**: 260717-9uh7-toolkit-standards-conformance
**Created**: 2026-07-18

## Origin

One-shot `/fab-new` invocation with a prescriptive task brief (verbatim):

> Task: Bring this repo and its tool into conformance with the sahil87 toolkit standards.
>
> Precondition: `shll standards` runs on this machine (if the subcommand is missing, run `shll update`; if it still fails, stop and report -- do not proceed from memory or the website). This repo's constitution carries the Toolkit Standards article; this task is the conformance work it mandates.
>
> 1. Enumerate at runtime: run `shll standards`, then `shll standards <name>` for every listed entry. The list is authoritative -- do not assume which standards exist or what they require.
> 2. Audit this repo against each standard. For mechanical contracts (machine help output, README/docs-site structure), execute the standard's own verification checklist verbatim. For the principles, assess each numbered principle against the tool's actual behavior -- prompts and TTY handling, stdout/stderr separation, --json/--dry-run/--yes coverage, exit codes and error wording, idempotency, output volume.
> 3. Fix what is proportionate here: all mechanical-contract violations, and principle gaps that are small and additive (a missing flag, a misrouted stream, an unhelpful error). Larger gaps that would restructure the tool are NOT for this change -- record each as a draft change or issue per this repo's convention and reference it.
> 4. Deliverable: one fab change whose PR body contains a conformance report -- one section per standard with PASS or the gaps found, each gap dispositioned as fixed here (with the commit) or deferred to <ref>. Include the shll version audited against (`shll version`'s shll row), since standards are versioned with the shll release. Tests green; if the command tree changed, re-verify the machine-help contract afterward.
>
> Note on the "skill" standard specifically: if this repo has not yet implemented a `<tool> skill` subcommand, that is a known, deferred gap (per the toolkit's phased per-repo adoption -- no seven-repo flag-day) -- report it as "deferred, not yet adopted" rather than treating it as an in-scope fix for this change.

**Intake-time verifications performed** (all in this conversation):

- Precondition PASSES: `shll standards` runs and lists 4 entries — `principles` (foundation), `help-dump` (binary), `readme-extraction` (repo), `skill` (binary+repo). `shll version` reports `shll v0.0.23`.
- The Toolkit Standards article was NOT in this worktree's constitution at invocation time — it merged to `origin/main` as PR #32 (`260717-vlr1-constitution-toolkit-standards`, commit c7bc701, constitution v1.1.0). The `std2` placeholder branch was **fast-forwarded to origin/main** before this change was created, so the mandate is now in-tree at `fab/project/constitution.md` § Toolkit Standards.
- Reconnaissance findings against the current tree/binary are recorded in **What Changes** below as audit seeds — the apply-stage audit re-enumerates and re-verifies everything at runtime; the seeds are leads, not conclusions.
- Caution learned live: `idea skill` does NOT print a skill bundle — the bare-text root shorthand swallows unknown words, so probing it **added an idea to the backlog** (reverted). The `skill` subcommand is genuinely absent → deferred per the task note.

## Why

1. **Pain point**: The constitution (v1.1.0, § Toolkit Standards) now binds this repo to the toolkit's published standards, and the standards are versioned with the shll release — but the repo has never been audited against them. At least one mechanical-contract violation is already confirmed (`help-dump` emits `captured_at`, which the standard explicitly forbids), so the repo is out of conformance today, not hypothetically.
2. **Consequence of inaction**: shll.ai consumes this repo's `help-dump` output and README/`docs/site` tree mechanically. Contract drift means broken or stale public pages, schema friction for the puller, and a constitution article that is dead letter — every future CLI/docs change would build on an unaudited baseline.
3. **Approach**: One conformance change that (a) enumerates the standards at runtime (the list is authoritative — no memory, no website), (b) executes each mechanical contract's own verification checklist verbatim and assesses each principle against actual tool behavior, (c) fixes mechanical violations and small additive principle gaps in-place, (d) defers restructuring-sized gaps to backlog entries with references, and (e) ships a per-standard conformance report in the PR body pinned to the audited shll version. Alternative rejected: fixing only the known `captured_at` bug without the full audit — rejected because the constitution mandates conformance with the *set* of standards, and the audit is the deliverable the task defines.

## What Changes

### 1. Runtime enumeration + audit procedure (apply-stage step one)

At apply entry, re-run the enumeration — it is authoritative and versioned with the shll release:

```sh
shll version           # pin the shll row for the report (intake-time: shll v0.0.23)
shll standards         # authoritative list (intake-time: principles, help-dump, readme-extraction, skill)
shll standards <name>  # full text of each listed standard — re-read all of them
```

If the list or any standard's text differs from the intake-time snapshot below, **the runtime text wins**. Audit the repo against each standard: for the mechanical contracts (`help-dump`, `readme-extraction`), execute the standard's own "Verifying conformance" checklist verbatim; for `principles`, assess each of the ten numbered principles against actual behavior (prompts/TTY handling, stdout/stderr split, `--json`/`--dry-run`/`--yes` coverage, exit codes and error wording, idempotency, output volume). For `skill`, see § 5.

### 2. help-dump contract — confirmed violation, fix here

The standard's rule with teeth: **"Do not emit `captured_at`. The capture timestamp is owned by shll.ai — a tool cannot know its own capture time. The puller stamps it after capture."**

Current source violates it:

- `src/cmd/idea/help_dump.go:29` — `CapturedAt string \`json:"captured_at"\`` in the envelope struct
- `src/cmd/idea/help_dump.go:109` — `CapturedAt: time.Now().UTC().Format(time.RFC3339)`
- `src/cmd/idea/help_dump_test.go:46-47` — test *asserts the field parses as RFC3339* (pins the violation)

Fix: remove the field from the envelope struct and its population; update the test to assert `captured_at` is **absent** from the emitted JSON. Envelope stays `{tool, version, schema_version, root}` with `schema_version: 1`. Then execute the standard's full verification checklist: exits 0, valid JSON to stdout only, stderr empty; `completion`/`help`/hidden commands absent from the tree; `version` reflects the built binary (ldflags, not a literal); the minimal pinning test stays green. Re-run this checklist again at the end if any other in-scope fix touches the command tree (added flags change `-h` text → change `text` fields).

### 3. readme-extraction contract — execute checklist; one candidate gap

Execute the standard's checklist verbatim over `README.md` + `docs/site/` (`install.md`, `workflows.md` — no reserved names `overview`/`readme`/`commands` in use). Intake-time recon: head order (H1 → toolkit blockquote → badges → tagline prose) conforms; the only relative links are README → `docs/site/*.md` (the auto-rewritten form — conforms); no images, no mermaid, no `#gh-*-mode-only` fragments; deep links to `docs/specs/*` are already absolute GitHub URLs.

**Candidate gap**: README line ~100 links the command reference as `https://shll.ai/tools/idea/commands/`, but the standard specifies the absolute URL form `https://shll.ai/<tool>/commands/` (i.e., `https://shll.ai/idea/commands/`). At apply, verify which URL is live (e.g., `curl -sIL` both; the standard's form is presumptively correct since standards are versioned with the installed shll) and fix the README to the standard's form. If verification contradicts the standard's form, record the finding in the report instead of editing.

### 4. principles — per-principle assessment; small additive fixes in scope

Assess all ten principles (№1 non-interactive, №2 stdout=data/stderr=diagnostics + `--json` stability, №3 help contract, №4 fail fast + exit codes 0/1/2, №5 visible mutation boundaries + `--dry-run`, №6 stateless/idempotent, №7 compose don't reinvent, №8 graceful degradation, №9 bounded output, №10 agent-discoverable docs) against every subcommand: `add`, `list/ls`, `show`, `done`, `reopen`, `edit`, `rm`, `prune`, `fmt`, `update`, `shell-init`, `help-dump`, plus the root bare-text shorthand.

Intake-time seeds (verify each at apply; fix the ones that hold as small additive gaps):

- **Consent-flag naming (№1/№5)**: `rm` requires `--force` to confirm; `prune` uses TTY prompt `[y/N]` / `--force`. Principle №1 says flag-satisfiable consent named as `--yes`/`-y`. Additive fix: add `--yes`/`-y` as an alias for the existing consent semantics on `rm` and `prune`, keeping `--force` (public CLI surface is a contract — no removals/renames).
- **`--dry-run` on destructive writes (№5)**: `rm` has no `--dry-run`; principle №5 requires destructive writes to support an accurate preview sharing the real code path. Additive fix: add `--dry-run` to `rm` (resolve the match via the same matcher, print what would be deleted, write nothing). `prune` already has a de-facto dry-run (piped mode + `--check`-like listing) — assess whether an explicit `--dry-run` alias is warranted or whether existing behavior satisfies the obligation; record the judgment either way.
- **Non-TTY prompt behavior (№1)**: №1 requires a command that would prompt to *refuse naming the flag* when stdin is not a TTY, never hang. `prune` piped is documented (docs/memory/cli/prune.md) as a deliberate free-dry-run + stderr hint, exit without deleting — it never hangs and names the flag. Assess whether this satisfies №1's contract (refusal-shaped: it declines to mutate and names `--force`) or is a gap; this is an audit judgment to record, not a pre-committed fix.
- **`--json` coverage (№2)**: `list` has `--json` (stable `{id,date,status,text}` per constitution VI). `show` emits a single record with no `--json`; assess whether `show --json` is a proportionate additive fix or a deferred gap.
- **Exit codes (№4)**: verify the 0/1/2 convention (0 success, 1 operational failure, 2 usage error) across subcommands and that errors state what failed, why, and what to do next. Fix only unhelpful error wording or wrong codes where the fix is local; defer anything structural.
- **№6/№8/№9**: `fmt` is byte-stable on second run (idempotency reference); output truncation is TTY-gated with `--full` escape; `update` degrades with a non-brew fallback hint. Verify and record; no gaps expected.

Any principle gap that would restructure the tool (e.g., a redesign of the query matcher, changing the bare-text shorthand's collision behavior with future subcommand names) is **out of scope**: record it as a deferred backlog entry (§ 6) and reference it in the report.

### 5. skill standard — deferred, not yet adopted (do NOT implement)

`idea skill` does not exist (the bare-text shorthand swallows the word — probing mutates the backlog; don't probe without `--file` isolation or cleanup). Per the task note and the standard's own adoption section ("No tool ships `skill` today ... a tool without a `skill` subcommand is not yet in violation"), the report's skill section reads **"deferred, not yet adopted"** with a pointer to the phased per-repo adoption. No implementation work in this change. Add a backlog entry for future adoption (§ 6) and reference it as the deferral target.

### 6. Deferred-gap convention: backlog entries in this branch

Larger gaps and the skill adoption are recorded as entries in **this worktree's** `fab/backlog.md` via `idea add "<text>"` (run from the repo root of this worktree — the default target is the current worktree's backlog, which is exactly this branch's committed file, so the entries ride this PR). Reference each entry's 4-char ID in the report's disposition column (`deferred → [id]`). This is the repo's demonstrated convention for future work (cf. existing entries `e3rk`, `ykwp`).

### 7. Conformance report — artifact + PR body

Author the report at `fab/changes/260717-9uh7-toolkit-standards-conformance/conformance-report.md` during apply, then the ship stage includes it verbatim in the PR body. Structure:

```markdown
# Toolkit Standards Conformance Report
Audited against: shll v0.0.23 (`shll version`, shll row; standards are versioned with the shll release)

## principles   — {PASS | gaps}
| # | Principle | Verdict | Disposition |
...one row per numbered principle; each gap: fixed here (commit <sha>) | deferred → [backlog-id]
## help-dump    — {PASS after fix | ...}   (checklist executed verbatim; captured_at removal noted with commit)
## readme-extraction — {...}               (checklist executed verbatim)
## skill        — deferred, not yet adopted (phased per-repo adoption; → [backlog-id])
```

### 8. Verification gate

- `go test ./...` green (fixes include updated `help_dump_test.go`; new flags get table-driven tests per constitution V — real temp dirs, no mocks).
- `go build` clean; if the command tree changed (new flags → changed `-h` text), re-run the help-dump verification checklist afterward (§ 2).
- `gofmt`/`go vet` clean (CI gate parity per docs/memory/ci).

## Affected Memory

- `cli/structure`: (modify) help-dump contract note — envelope no longer carries `captured_at`; any new persistent/consent flags on the command surface
- `cli/prune`: (modify) consent-flag surface if `--yes`/`-y` alias lands; №1 audit judgment on the piped free-dry-run
- `release/pipeline`: (modify) shll.ai pull relationship — `captured_at` is stamped by the puller, not emitted by the binary
- `cli/rm`: (new) — only if `rm` gains `--dry-run`/`--yes` and the audit judges a per-subcommand note warranted (rm currently has no memory file; otherwise fold into `cli/structure`)

## Impact

- `src/cmd/idea/help_dump.go`, `src/cmd/idea/help_dump_test.go` — captured_at removal (confirmed)
- `src/cmd/idea/rm.go`, `src/cmd/idea/prune.go` (+ tests) — consent-flag aliases, `rm --dry-run` (audit-confirmed candidates)
- `src/cmd/idea/show.go` — possible `--json` (audit judgment)
- `README.md` — command-reference URL form (verify-then-fix)
- `fab/backlog.md` — deferred-gap entries (additive lines via `idea add`)
- `fab/changes/260717-9uh7-toolkit-standards-conformance/conformance-report.md` — new artifact
- No dependency changes; no restructuring. Public CLI surface only grows (aliases/flags), never breaks (constitution VI: output formats are public contract).

## Open Questions

- None. The brief is prescriptive; the two apply-time verifications (live commands-URL check in § 3, prune non-TTY judgment in § 4) are recorded as Confident assumptions with their verification steps rather than questions.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The audit target set is whatever `shll standards` lists at apply time (intake snapshot: principles, help-dump, readme-extraction, skill @ shll v0.0.23); runtime text wins over this intake | Task states the list is authoritative — do not assume; verified running at intake | S:95 R:90 A:95 D:95 |
| 2 | Certain | Change is based on origin/main c7bc701 (constitution v1.1.0 with the Toolkit Standards article); the stale `std2` placeholder branch was fast-forwarded before creation | Verified in conversation: PR #32 merged 2026-07-17; ff was clean (no unique commits) | S:90 R:85 A:95 D:90 |
| 3 | Certain | `skill` standard is reported "deferred, not yet adopted" with a backlog ref — no implementation | Task note is explicit; the standard's own Adoption section says no tool ships it yet | S:95 R:90 A:95 D:95 |
| 4 | Certain | Remove `captured_at` from the help-dump envelope and update the test that currently asserts it | The standard forbids emitting it ("rule with teeth"); task mandates fixing all mechanical-contract violations; source locations confirmed | S:90 R:80 A:90 D:90 |
| 5 | Confident | Add `--yes`/`-y` as consent alias on `rm` and `prune`, keeping `--force` (no renames/removals) | Principle №1 names `--yes`/`-y`; task lists "a missing flag" as in-scope; alias is additive so the public surface only grows | S:70 R:75 A:75 D:65 |
| 6 | Confident | Add `--dry-run` to `idea rm` sharing the live match path | Principle №5 MUST for destructive writes; small additive; prune's existing piped/`--check`-style preview assessed separately | S:75 R:75 A:75 D:70 |
| 7 | Confident | README commands URL corrected to the standard's `https://shll.ai/idea/commands/` after a live-URL check; on contradiction, record instead of edit | Standard specifies `/<tool>/commands/`; current README uses `/tools/idea/`; standards version matches installed shll | S:65 R:85 A:70 D:70 |
| 8 | Confident | Deferred gaps recorded as `idea add` entries in this worktree's committed `fab/backlog.md`, IDs referenced in the report | Repo's demonstrated convention (e3rk, ykwp); entries ride this PR; task says "draft change or issue per this repo's convention" | S:60 R:80 A:75 D:65 |
| 9 | Confident | Conformance report authored as a change-folder artifact (`conformance-report.md`); ship copies it into the PR body | Task requires the PR body carry it; artifact-first keeps it reviewable in-repo and survives PR-body edits | S:70 R:85 A:80 D:70 |
| 10 | Certain | Report pins the shll row from `shll version` re-read at apply (intake-time: `shll v0.0.23`) | Task explicit — standards are versioned with the shll release | S:95 R:90 A:90 D:95 |
| 11 | Confident | `prune`'s piped free-dry-run (never hangs, names `--force` on stderr, exits without deleting) is assessed at apply as an audit judgment against №1's refusal contract — documented design, not a pre-committed fix | Behavior is deliberate and documented (docs/memory/cli/prune.md); №1's intent (no hang, flag named) is arguably met; judgment recorded either way | S:55 R:70 A:60 D:55 |

11 assumptions (5 certain, 6 confident, 0 tentative, 0 unresolved).
