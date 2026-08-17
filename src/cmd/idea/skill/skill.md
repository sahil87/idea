# idea — agent skill bundle

`idea` is a worktree-aware backlog CLI: it manages a plain-Markdown checklist of ideas
(`fab/backlog.md`) per git worktree. Capture, list, and triage backlog items without leaving
the terminal or hand-editing Markdown. This bundle is the usage briefing for an agent *using*
the tool — for the full flag reference run `idea <cmd> -h`; for the command tree, `idea help-dump`.

## When to use

- You want to jot a backlog idea mid-task without breaking flow: `idea "the thing"`.
- You need to list, show, complete, edit, or remove ideas in the current repo/worktree.
- You are handing IDs to fab-kit: `idea`'s 4-char IDs are what `/fab-new <id>` consumes.

**Not the right reach**: it is a flat backlog list, not a task runner, issue tracker, or
scheduler. There are no priorities, assignees, labels, due dates, or dependencies — just
`open`/`done` checklist lines. Reach for a real tracker when you need those.

## Capabilities map

One line per capability, keyed to the subcommand. Every backlog command honors the target
selectors below and the persistent `-f/--file` override.

| Command | Does |
|---------|------|
| `idea add <text>` | Append a new idea (generated 4-char ID + today's date). `--id`/`--date` override. |
| `idea <text>` | Bare-text shorthand for `add` — any first word that is not a subcommand/alias. |
| `idea list` / `ls` | List ideas (open by default; `--all/-a`, `--done`, `[id...]` filter, `--sort`, `--reverse`, `--full`, `--json`, `--stale N[d]` for open ideas older than N days). |
| `idea show <query>` | Show one matching idea (`--json` for the record). |
| `idea done <query>` | Mark a matching open idea done. |
| `idea reopen <query>` | Re-open a matching done idea. |
| `idea edit <query> [new-text]` | Replace text inline, or open `$VISUAL`/`$EDITOR`/`vi` when text is omitted. `--id`/`--date` too. |
| `idea rm <query>` | Delete one idea. Needs consent: `--yes/-y` (or `--force`). `--dry-run` previews, writes nothing. |
| `idea prune` | Bulk-remove all done ideas. Bare run is a dry run; `--yes/-y`/`--force` confirms. |
| `idea promote <query>` | Move an idea to the main worktree's backlog (ID/date/status preserved; refuses on ID collision). `--main`/`--system` are rejected — promote picks its own roots. |
| `idea fmt` | Rewrite the backlog into canonical form (and adopt bare `- [ ]` checkboxes). `--check` gates without writing. |
| `idea update` | Self-update the binary via Homebrew. |

`<query>` matches by 4-char ID or a case-insensitive substring of the text.

## Targets (which backlog a command operates on)

- **default** — the *current worktree's* backlog (`{worktree-root}/fab/backlog.md`).
- **`-m, --main`** — the main worktree's shared backlog (git-only).
- **`-s, --system`** — `~/.config/idea/backlog.md` (cross-repo; also the automatic fallback
  outside any git repo). `~/.config/idea` is used on every platform; `$XDG_CONFIG_HOME` is ignored.
- **`-f, --file <path>` / `IDEAS_FILE`** — override the path within the selected root (ignored under `--system`).

`--main` and `--system` are mutually exclusive.

## Composition with fab-kit

The backlog file is shared vocabulary. The line format is the stable cross-tool contract:

```
- [ ] [id] YYYY-MM-DD: text
```

fab-kit's `/fab-new <id>` consumes an idea's 4-char ID to start a change; the checkbox flips to
`[x]` as the item is worked. `idea` owns capture and triage; fab-kit owns turning an idea into a
change. Keep IDs stable — external scripts and `/fab-new` key on them.

## Output & exit-code contracts

- **stdout is data.** Confirmations and list/show output go to stdout as canonical, machine-parseable
  lines. **Advisory notices go to stderr** (e.g. `note: stamped today's date on N item(s)`,
  `warning: no idea with ID ...`, prune's confirm prompt/hint) so stdout stays clean for pipes.
- **Piped `list` output is canonical and untruncated.** On a TTY, `list`/`ls` truncate long text to
  the width and add color; when piped or redirected they emit full canonical lines regardless of
  `--full`, with no ANSI. Rely on the piped form in scripts.
- **`--json`** exists on `list` and `show` **only**. Schema per record: `{id, date, status, text}`
  with `status: "open"|"done"` (never a boolean). `list --json` is an array; an empty/absent backlog
  yields `[]`.
- **Exit codes follow the toolkit convention: `0` success, `1` operational failure, `2` usage
  error.** Malformed invocations exit `2` — unknown flags, wrong argument counts, a missing or
  unsupported `shell-init` shell, and the `--system`+`--main` conflict. Well-formed invocations
  that fail exit `1` — consent refusals (`rm`/`prune` without `--yes`), no-match or ambiguous
  queries, `fmt --check` on a non-canonical file, and I/O failures. Branching on `2` to detect
  usage errors is supported.

## Gotchas

- **Worktree default.** With no target flag inside a repo, you operate on the *current worktree's*
  backlog — not the main one. Use `-m` for the shared main backlog. This is by design (idea capture
  during exploratory work should not pollute across worktrees).
- **Exact-ID precedence.** When a 4-char ID also appears as substring text inside another idea, a
  query equal to that exact ID resolves to the ID's owner (not the ambiguity error). Passing the
  exact ID is the documented way to disambiguate.
- **Ambiguous queries are refused.** `show`/`done`/`reopen`/`edit`/`rm` refuse a query that matches
  more than one idea and list the matches — be more specific or use the exact ID.
- **Destructive writes need consent.** `rm` and `prune` refuse without `--yes/-y` (or `--force`).
  `rm --dry-run` previews via the real match path and wins over any consent flag.
- **Multiline text is escaped on disk.** An idea whose text contains newlines is stored as one
  physical line with `\n` escapes; `--json` and `show` render the real newlines back.
- **Canonical-write churn.** Any mutating command rewrites the whole file into canonical form
  (bullets normalized, dates backfilled). To land that churn as its own commit with no semantic
  change, run `idea fmt` explicitly. `idea fmt --check` exits 1 when the file is non-canonical.
- **Namespace.** Subcommand names and the `ls` alias are reserved — `idea prune ...` routes to the
  subcommand, never to bare-text capture. `idea add "..."` is the unambiguous capture path.
