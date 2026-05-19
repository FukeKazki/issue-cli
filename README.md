# issue-cli

Local-only issue manager. Stores issues as YAML under `.issues/` in the repo,
browses them through a built-in TUI, and switches Git branches by id.

The repository is `issue-cli` but the installed binary is named `issue`.

## Install

```sh
go install github.com/FukeKazki/issue-cli/cmd/issue@latest
```

Or via mise:

```sh
mise use -g go:github.com/FukeKazki/issue-cli/cmd/issue@latest
```

Requires `git` on `PATH`.

## Usage

```
issue                                  # show the issue for the current issue/<id> branch, or open the list TUI
issue show <id> [--format markdown|yaml|json]
issue list [--all] [--status=STATUS] [--format json]
issue next [--format json]             # next actionable TODO issue, for automation
issue create [--title TITLE]
issue edit <id> --status STATUS        # update an issue's status from the CLI (case-insensitive)
```

### `issue` (no args)

If the current Git branch matches `issue/<id>`, prints the matching issue's
detail (title, status, description, references, scope, timestamps). On any
other branch, opens the list TUI (same as `issue list`).

### `issue list`

Opens the built-in TUI with open issues (TODO / In Progress / Reviews).
Left pane is the list, right pane shows the detail preview.

| Key             | Action                                                     |
| --------------- | ---------------------------------------------------------- |
| `Enter`         | Show issue detail (read-only; `q`/`Esc`/`Enter` to return) |
| `c`             | `git checkout` (or create) `issue/<id>` and exit the list  |
| `n`             | Create a new issue (TUI form)                              |
| `e`             | Edit the selected issue (TUI form)                         |
| `s`             | Change status (then `1`–`4` or `Enter`)                    |
| `d`             | Delete the selected issue (confirm)                        |
| `v`             | Toggle detail preview                                      |
| `j` / `k`       | Move cursor (also `↓` / `↑`)                               |
| `g` / `G`       | Jump to top / bottom                                       |
| `/`             | Filter (case-insensitive substring)                        |
| `q` / `Esc`     | Quit                                                       |
| `Ctrl-C`        | Quit                                                       |

`Enter` is non-destructive: it opens the detail view so `c` (checkout) can't
be hit by accident.

Filters: `--all` includes Done, `--status="In Progress"` filters by one status.

### `issue create`

Without args → opens TUI form (title / status / references / scope).
With `--title "..."` → creates with that title, status `TODO`, no references/scope.

### `issue edit`

Non-interactive status update. The change is allowed in any direction (the
forward-only rule only applies to event-driven auto-transitions like `c`
checkout).

```sh
issue edit 13 --status DONE
issue edit #13 --status done
issue edit 13 --status in-progress
```

Accepted `--status` values (case-insensitive):

| Canonical     | Aliases                                            |
| ------------- | -------------------------------------------------- |
| `TODO`        | `todo`                                             |
| `In Progress` | `in progress`, `in-progress`, `in_progress`, `inprogress` |
| `Reviews`     | `reviews`, `review`                                |
| `Done`        | `done`                                             |

`--status` is required. Unknown values exit non-zero without touching the YAML.

## Automation / machine-readable output

For piping into runners like
[`simple-takt`](https://github.com/FukeKazki/simple-takt), non-interactive
subcommands emit structured output instead of opening the TUI. The existing
TUI behavior is unchanged — these flags only activate when supplied.

| Command                                          | Output                                                                  |
| ------------------------------------------------ | ----------------------------------------------------------------------- |
| `issue show <id> --format markdown\|yaml\|json`  | one issue; non-zero exit if the id is missing or unknown                |
| `issue list --format json [--status STATUS]`     | JSON array of issues (after the same `--all` / `--status` filter)       |
| `issue next [--format json]`                     | envelope `{"issue": {...}}`, or `{"issue": null}` when no TODO remains  |

`issue next` picks the lowest-id `TODO` issue (deterministic) and always
exits 0 so downstream pipes always receive valid JSON.

Pipe into simple-takt:

```sh
issue next --format json | simple-takt -w issue-dev
```

## Storage

`.issues/<id>.yaml` — one file per issue.

```yaml
id: 1
title: Implement feature X
status: TODO        # TODO | In Progress | Reviews | Done
description: |
  Background, acceptance criteria, or any longer-form notes.
references:
  - https://example.com/spec
scope:
  - "@apps/web/hoge.tsx"
created_at: 2026-05-16T10:30:00+09:00
updated_at: 2026-05-16T10:30:00+09:00
```

IDs are assigned as `max(existing)+1`. Whether `.issues/` is committed to git
is up to you — the CLI doesn't touch `.gitignore`.

Branch switches use `git checkout` directly, so an unclean working tree will
abort with git's own warning.

## Claude Code skill

`skills/issue-cli/SKILL.md` is a Claude Code skill that teaches the agent how
to drive this CLI from a non-interactive context (when to call `issue create
--title`, when to read `.issues/<id>.yaml` directly, branch convention, etc.).

Install it via [apm](https://github.com/yoshinani-dev/apm) by adding the
following to your project's `apm.yml`:

```yaml
dependencies:
  apm: [
    FukeKazki/issue-cli/skills/issue-cli
  ]
```

`apm install` deploys it under `.claude/skills/issue-cli/` (and
`.agents/skills/issue-cli/`) in the consuming project.
