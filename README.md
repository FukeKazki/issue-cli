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
issue list [--all] [--status=STATUS]
issue create [--title TITLE]
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
