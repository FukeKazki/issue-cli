# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build / test

The Go module is `github.com/FukeKazki/issue-cli`; the binary (under `cmd/issue/`) is `issue`.

```sh
go build ./cmd/issue              # build (output: ./issue, gitignored)
go install ./cmd/issue            # install as `issue` on $PATH
go test ./...                     # all tests
go test ./internal/store -run TestSaveAndLoadRoundTrip  # single test
go vet ./...
```

`gitx_test.go` shells out to a real `git` (and `t.Chdir`s into a temp repo) — it will be skipped if `git` is missing.

## Architecture

A small Bubble Tea TUI wrapped around a YAML-on-disk store and a thin `git` driver.

- **`cmd/issue/main.go`** — minimal arg parser, dispatches to `internal/cli`. Subcommands: `list`/`ls`, `create`/`new`, `sync`, `_show` (internal), `help`. No flag library at the top level; subcommands use `flag.NewFlagSet`.
- **`internal/model`** — `Issue` struct + `Status` enum (`TODO` / `In Progress` / `Reviews` / `Done`). `OpenStatuses()` excludes `Done`. `ParseStatus` is the only string→`Status` path; use it instead of casting. `StatusRank` orders the enum and `(*Issue).AdvanceStatus(target)` is the forward-only setter used by auto-transitions — prefer it over assigning `iss.Status` directly when the change is driven by an event (checkout / sync) rather than explicit user choice.
- **`internal/store`** — one YAML file per issue at `<repo-root>/.issues/<id>.yaml`. Repo root comes from `git rev-parse --show-toplevel` (falls back to CWD). `Save` writes via tempfile + `os.Rename` for atomicity and always stamps `UpdatedAt`. `NextID` is `max(existing)+1` — IDs are never reused even after delete.
- **`internal/gitx`** — wraps `git symbolic-ref` and `git checkout`. `CurrentIssueID()` parses an `issue/<n>` branch back into an int (returns `0` when the branch does not match). `CheckoutIssue` creates the branch with `-b` if missing, otherwise reuses it.
- **`internal/cli`** — orchestration. `Default()` is the no-arg entry: on an `issue/<id>` branch it prints `RenderDetail`; otherwise it falls through to `List(nil)`. `List` is a loop — the TUI returns a `ListResult{Action, IssueID}` and the loop dispatches (show / checkout / edit / create / status / delete), then re-runs the TUI with `lastID` preserved as the cursor anchor. `Checkout` is the only action that exits the loop instead of looping back, and on a successful checkout `advanceOnCheckout` auto-bumps a TODO issue to In Progress (forward-only). `Sync` is a separate subcommand that calls `gh pr list` for `issue/<id>` and advances status accordingly (open→Reviews, merged→Done); `prStatusForBranch` is a package var so tests can stub the `gh` dependency.
- **`internal/tui`** — Bubble Tea models. Each interaction (list, form, detail view, status picker, confirm) is its own `tea.Program` invoked via a `RunX` function that returns a result struct or `ErrCanceled`. `RenderDetail` returns a plain string and is used by both the TUI preview pane and the non-TUI `Default()` / `Show()` paths — keep it side-effect-free.

### Control-flow shape worth knowing

The list TUI does not perform mutations itself: it returns a sentinel action and the `cli.List` loop calls the relevant `tui.RunForm` / `tui.RunConfirm` / `s.Save` / `s.Delete`. New keys should follow this split — add a `ListAction*`, set it from `updateBrowsing`, and handle the side-effect in `cli/list.go`.

`Enter` is intentionally non-destructive (opens detail). `c` is checkout. Don't wire mutating actions to `Enter`.

## Storage convention

Issues live under `<repo-root>/.issues/`. The CLI does not touch `.gitignore`; whether the directory is committed is the user's choice (this repo gitignores it). When adding fields to `model.Issue`, also add a YAML tag — `Save` round-trips through `gopkg.in/yaml.v3`.

## APM / skills

`apm.yml` + `apm.lock.yaml` describe Anthropic Package Manager dependencies; `apm_modules/` is the gitignored apm download target. `.claude/skills/` and `.agents/skills/` hold apm-deployed skill copies (currently just `submit-pr`).

This repo also publishes its own skill: `skills/issue-cli/SKILL.md`. External projects install it via apm with `FukeKazki/issue-cli/skills/issue-cli` in their `apm.yml`.
