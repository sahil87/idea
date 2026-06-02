# Quality Checklist: Add --skip-brew-update flag to update command

**Change**: 260531-t2ov-skip-brew-update-flag
**Generated**: 2026-05-31
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Flag definition: `--skip-brew-update` is a real local cobra bool flag on `update`, default `false`, with usage text mentioning the `brew update` tap-metadata refresh; appears in `idea update --help`.
- [ ] CHK-002 Threaded parameter: `Update` signature is `Update(currentVersion string, skipBrewUpdate bool, out, errOut io.Writer) error` and the cobra `RunE` passes the parsed flag value.
- [ ] CHK-003 Skip guards only brew update: when `skipBrewUpdate == true`, only the `brew update` block is skipped; `brew info`, the up-to-date short-circuit, and `brew upgrade` are unaffected.

## Behavioral Correctness
- [ ] CHK-004 Default unchanged: with the flag absent (`false`), `brew update --quiet` runs exactly as before — no behavioral or output difference from pre-change.
- [ ] CHK-005 Output routing preserved: `brew update`/`brew info` remain captured; `brew upgrade` still inherits `os.Stdin/Stdout/Stderr`; wrapper messages still go to `out`/`errOut`; `errSilent` mapping intact.
- [ ] CHK-006 No convention refactor: no `internal/proc` wrapper or command-runner interface introduced; `os/exec` remains the mechanism; only the minimal `execCommandContext`/`brewInstalled` `var` seam added.

## Scenario Coverage
- [ ] CHK-007 "Skip omits brew update but still upgrades": test asserts skip=true → recorded argv has `info` + `upgrade`, NOT `update`.
- [ ] CHK-008 "Default path runs brew update": test asserts skip=false → recorded argv has `update` + `info` + `upgrade`.
- [ ] CHK-009 Help lists the flag: `--skip-brew-update` is discoverable (verifiable via cobra flag registration / help).

## Edge Cases & Error Handling
- [ ] CHK-010 Up-to-date short-circuit with skip: when skip=true and versions match, `brew upgrade` is NOT invoked and "Already up to date" fires (covered by spec scenario; verify behavior preserved).
- [ ] CHK-011 Single caller compiles: the only `Update` caller (`src/cmd/idea/update.go`) is updated and the module builds.

## Code Quality
- [ ] CHK-012 Pattern consistency: new code follows the file's naming/structure (package-level `var` seam near consts, doc-comment style on `Update`, table-driven test like existing tests).
- [ ] CHK-013 No unnecessary duplication: the helper-process / recorder pattern is the standard stdlib idiom; no duplicated brew-invocation logic.
- [ ] CHK-014 No magic strings: brew subcommand names compared in tests reference clear literals; formula stays the `brewFormula` constant (Constitution: no magic strings, anti-pattern list).
- [ ] CHK-015 Function size: `Update` stays focused; the skip guard does not turn it into a god function.

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`

<!-- Migrated to plan.md on 2026-06-02 — safe to delete. -->
