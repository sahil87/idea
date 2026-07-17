# Toolkit Standards Conformance Report

**Audited against:** `shll v0.0.23` (`shll version`, shll row — standards are versioned with the shll release)
**Tool binary audited:** `idea v0.0.15-4-gc7bc701` (freshly built from this branch's `src/` with `-ldflags "-X main.version=..."`)
**Standards enumerated at runtime** (`shll standards`, authoritative): `principles` (foundation), `help-dump` (binary), `readme-extraction` (repo), `skill` (binary+repo). The runtime list and each standard's text matched the intake snapshot exactly.

> Dispositions marked "fixed here (this PR)" refer to changes committed and shipped in this pull request from branch `260717-9uh7-toolkit-standards-conformance`.

---

## principles — gaps found (2 fixed here, 1 deferred; 7 PASS)

Assessed all ten principles against the actual behavior of every subcommand (`add`, `list`/`ls`, `show`, `done`, `reopen`, `edit`, `rm`, `prune`, `fmt`, `update`, `shell-init`, `help-dump`, and the root bare-text shorthand).

| # | Principle | Verdict | Disposition |
|---|-----------|---------|-------------|
| 1 | Non-interactive by default | **Gap → fixed here** | `rm`/`prune` had only `--force` for flag-based consent; the standard names `--yes`/`-y`. Added `--yes`/`-y` as an additive consent alias on both (keeping `--force`). Non-TTY behavior already conformed: `rm` without consent refuses (exit 1, names the flag); `prune` piped is a free dry-run that never hangs and names the consent flag on stderr. Refusal/hint wording updated to lead with the standard's flag: `Use --yes (or --force) to confirm` (review should-fix). **fixed here (this PR)** |
| 2 | stdout is data, stderr is diagnostics | **PASS** | Streams split correctly (data on stdout; count headers, prompts, hints, backfill notices, warnings on stderr). `--json` present and stable on both `list` and `show` (schema `{id,date,status,text}`, Constitution VI). *Note: intake seed #4 claimed `show` lacked `--json` — stale; the running binary emits `show --json` correctly, so no work was needed (runtime is authoritative).* |
| 3 | Help is a published contract | **PASS** (after help-dump fix) | Layered `Long` help repo-wide; hidden `help-dump` walks the live cobra tree (never parses `-h`). See the help-dump section below. |
| 4 | Fail fast with actionable errors | **Gap → deferred** | Errors state what/why/next and non-zero exits are used. But the `0`/`1`/`2` convention is only partially met: usage errors (unknown flag, wrong arg count, unknown subcommand) exit **1**, not **2** — only `shell-init` returns 2 (its own `os.Exit(2)`). A complete fix must tag both the flag-error seam (`SetFlagErrorFunc`) and the arg/command-resolution seam (verified: `SetFlagErrorFunc` does **not** catch arg-count/unknown-command errors), which is structural. A flag-only partial fix would make usage-error classes disagree — worse for an agent than the current uniform 1. **deferred → [xvsj]** |
| 5 | Visible mutation boundaries | **Gap → fixed here** | `rm` had no `--dry-run` (destructive writes MUST support an accurate preview on the real code path). Added `rm --dry-run` routing through the same `RequireSingle` match path the live delete uses (via a new `idea.RmPreview` seam), printing the would-be-deleted line and writing nothing; dry-run wins over consent. `prune` already provides a de-facto dry-run (piped free dry-run + interactive pre-confirm listing) — assessed as satisfying the obligation; no redundant `--dry-run` alias added. **fixed here (this PR)** |
| 6 | Stateless, therefore retry-safe | **PASS** | No state files/cache; worktree + version re-derived per invocation. `idea fmt` is byte-stable on a second run (idempotency reference); mutating commands converge on retry. |
| 7 | Compose, don't reinvent | **PASS** | `update` wraps `brew` (never parses formulas) and probes the callee's advertised flag before use (`--skip-brew-update`); git resolution shells out to `git rev-parse`. No cross-tool internals reached into. |
| 8 | Graceful degradation | **PASS** | `update` degrades to a manual-install hint when not brew-installed (exit 0); color is TTY-gated (`NO_COLOR` honored); list/prune truncation falls back to full canonical lines when piped. |
| 9 | Bounded, high-signal output | **PASS (with note)** | `list`/`prune` truncate to terminal width with a `--full` escape; piped output is canonical one-line-per-record. There is no unbounded surface requiring an explicit cap, and no `--quiet` flag — assessed as not required today (no chatter/progress surface to suppress; stdout is already data-only). Recorded as a judgment, not a gap. |
| 10 | Agent-discoverable documentation (SHOULD) | **PASS for readme-extraction; skill deferred** | README + `docs/site/**` conform to the readme-extraction standard (see below). The `<tool> skill` bundle is not yet adopted — see the skill section. Principle 10 is a SHOULD; the skill bundle is its most forward-leaning obligation and is phased. |

---

## help-dump — PASS after fix (checklist executed verbatim)

Executed the standard's "Verifying conformance" checklist against the freshly built binary:

- [x] `idea help-dump` exits **0**, writes valid JSON to **stdout only**, **stderr empty**.
- [x] Envelope is `{tool, version, schema_version, root}` — **no `captured_at`** (was a confirmed violation; **fixed here**).
- [x] `completion`, `help`, and all hidden commands (including `help-dump` itself) are absent from the tree.
- [x] `version` reflects the built binary (`v0.0.15-4-gc7bc701`, from ldflags — not a literal).
- [x] `schema_version` is the integer `1`.
- [x] A minimal test pins the contract surface (exit 0, valid JSON, `tool`, `schema_version`) — updated to assert `captured_at` is **absent**.

**Gap found & fixed:** the envelope emitted `captured_at` (`time.Now().UTC()`), which the standard explicitly forbids ("the capture timestamp is owned by shll.ai — a tool cannot know its own capture time. The puller stamps it after capture").

- `src/cmd/idea/help_dump.go` — removed the `CapturedAt` struct field and its population; dropped the now-unused `time` import. **fixed here (this PR)**
- `src/cmd/idea/help_dump_test.go` — the test previously *asserted `captured_at` parses as RFC3339* (pinning the violation); inverted to assert the field is **absent** from the raw JSON. **fixed here (this PR)**

The command tree changed this cycle (new `--yes`/`-y`/`--dry-run` flags → changed `-h` text, which the node `text` reproduces), so the checklist was re-run after all fixes and still passes.

---

## readme-extraction — PASS after fix (checklist executed verbatim)

Executed the standard's "Verifying conformance" checklist over `README.md` + `docs/site/` (`install.md`, `workflows.md`):

- [x] README head order: `#` H1 → toolkit blockquote (exact canonical line) → badges → tagline prose. Conforms.
- [x] No relative images anywhere; all links leaving the published set are absolute `https://…` (the `docs/specs/*` links are already absolute GitHub URLs).
- [x] The only relative targets are README → `docs/site/install.md` and README → `docs/site/workflows.md` (the auto-rewritten natural form) — conforms. No relative links inside `docs/site/**`.
- [x] No `#gh-*-mode-only` fragments; no mermaid fences.
- [x] No `docs/site/` page named `overview`, `readme`, or `commands`.
- [x] README cross-links its `docs/site/` pages and the command reference.

**Gap found & fixed:** the command-reference link used `https://shll.ai/tools/idea/commands/`, but the standard specifies `https://shll.ai/<tool>/commands/` = `https://shll.ai/idea/commands/`. Verified live (`curl`): the standard's form is the canonical rendered page (HTTP 200, `<title>Commands | shll</title>`, ~134 KB of real command content); the old form is a redirect stub (HTTP 200 but `<title>Redirecting to: /idea/commands/</title>`, 345 bytes). Corrected the README to the standard's form.

- `README.md` (command-reference section) — `…/tools/idea/commands/` → `…/idea/commands/`. **fixed here (this PR)**

---

## skill — deferred, not yet adopted

`idea` does not implement a `skill` subcommand. Per the standard's own Adoption clause — "Phased, per-repo … **No tool ships `skill` today** … A tool without a `skill` subcommand is not yet in violation — principle №10 is a SHOULD" — and the intake's explicit direction, this is reported as **deferred, not yet adopted**, not fixed here. (Note: probing for the subcommand via bare text mutates the backlog, since the root bare-text shorthand swallows unknown words — the `skill` subcommand is genuinely absent, confirmed by inspecting `src/cmd/idea/main.go`'s registered command set, not by probing.)

Adoption is recorded for future work: **deferred → [3q43]** (add `idea skill` printing a static ≤150-line usage bundle byte-identical to `docs/site/skill.md`, wired via the sync + drift-guard pattern `shll standards` uses; renders at `https://shll.ai/idea/skill` for free).

---

## Deferred-gap backlog entries (this branch's `fab/backlog.md`)

| ID | Gap | Standard |
|----|-----|----------|
| `[3q43]` | Adopt the `skill` standard (`idea skill` bundle + drift-guard) | skill |
| `[xvsj]` | Tree-wide usage-error exit-code `2` convention | principles №4 |

---

## Verification gate

- `go build ./...` — clean.
- `gofmt -l .` (from `src/`) — no output (all files formatted).
- `go vet ./...` — clean.
- `go test ./...` — green (includes the inverted `help_dump_test.go` and new table-driven tests: `TestRmPreview` in `internal/idea`, `TestRmPruneConsent_CLI` in `cmd/idea` — real temp dirs, no mocks, Constitution V).
- Command tree changed → `idea help-dump` checklist re-run: passes (exit 0, valid JSON, no `captured_at`, `schema_version` 1).
