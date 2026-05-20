#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  tools/issue-takt-run.sh [options]

Options:
  --workflow NAME       simple-takt workflow name (default: issue-dev)
  --limit N            maximum number of issues to run (default: 1)
  --issue ID           run one specific issue instead of selecting from TODO
  --task-format FORMAT issue show format passed to simple-takt: markdown|json|yaml (default: markdown)
  --continue-on-error  record failed runs and continue with the next issue
  --worktree           run each issue in an isolated git worktree
  --worktree-dir DIR   base directory for worktrees (default: ../issue-worktrees)
  --dry-run            print the issues that would run without mutating metadata
  -h, --help           show this help

Environment:
  ISSUE_BIN            issue command to use (default: issue)
  SIMPLE_TAKT_BIN      simple-takt command to use (default: simple-takt)
  ISSUE_TAKT_LOG_DIR   log directory (default: .takt/issue-runner)
EOF
}

workflow="issue-dev"
limit="1"
issue_id=""
task_format="markdown"
dry_run="false"
continue_on_error="false"
use_worktree="false"
worktree_dir="../issue-worktrees"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --workflow)
      workflow="${2:?--workflow requires a value}"
      shift 2
      ;;
    --limit)
      limit="${2:?--limit requires a value}"
      shift 2
      ;;
    --issue)
      issue_id="${2:?--issue requires a value}"
      shift 2
      ;;
    --task-format)
      task_format="${2:?--task-format requires a value}"
      shift 2
      ;;
    --continue-on-error|--contin-on-error)
      continue_on_error="true"
      shift
      ;;
    --worktree)
      use_worktree="true"
      shift
      ;;
    --worktree-dir)
      worktree_dir="${2:?--worktree-dir requires a value}"
      shift 2
      ;;
    --dry-run)
      dry_run="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

case "$limit" in
  ''|*[!0-9]*)
    echo "--limit must be a positive integer" >&2
    exit 2
    ;;
esac
if [ "$limit" -lt 1 ]; then
  echo "--limit must be a positive integer" >&2
  exit 2
fi

case "$task_format" in
  markdown|json|yaml) ;;
  *)
    echo "--task-format must be one of: markdown, json, yaml" >&2
    exit 2
    ;;
esac

takt_bin="${SIMPLE_TAKT_BIN:-simple-takt}"
log_dir="${ISSUE_TAKT_LOG_DIR:-.takt/issue-runner}"
task_file=""

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 127
  fi
}

iso_now() {
  date +"%Y-%m-%dT%H:%M:%S%z"
}

cleanup() {
  if [ -n "$task_file" ]; then
    rm -f "$task_file"
  fi
}
trap cleanup EXIT

issue_bin="${ISSUE_BIN:-issue}"
require_cmd jq
require_cmd "$issue_bin"
if [ "$use_worktree" = "true" ]; then
  require_cmd git
fi
if [ "$dry_run" != "true" ]; then
  require_cmd "$takt_bin"
fi

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
case "$log_dir" in
  /*) ;;
  *) log_dir="${repo_root}/${log_dir}" ;;
esac
case "$worktree_dir" in
  /*) ;;
  *) worktree_dir="${repo_root}/${worktree_dir}" ;;
esac

select_issue_ids() {
  if [ -n "$issue_id" ]; then
    printf '%s\n' "$issue_id"
    return
  fi

  "$issue_bin" list --status TODO --format json |
    jq -r --argjson limit "$limit" '
      [
        .[]
        | select((((.metadata // {})["workflow-status"] // "") as $s | ($s != "running" and $s != "queued")))
        | .id
      ][: $limit][]
    '
}

sync_takt_config() {
  target_dir="$1"

  if [ ! -d "${repo_root}/.takt" ]; then
    return
  fi

  mkdir -p "${target_dir}/.takt"
  for entry in config.yaml workflows facets; do
    if [ -e "${repo_root}/.takt/${entry}" ]; then
      rm -rf "${target_dir}/.takt/${entry}"
      cp -R "${repo_root}/.takt/${entry}" "${target_dir}/.takt/${entry}"
    fi
  done
}

ensure_worktree() {
  id="$1"
  branch="issue/${id}"
  path="${worktree_dir}/issue-${id}"

  if [ -e "$path" ]; then
    if [ ! -d "$path/.git" ] && ! git -C "$path" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
      echo "worktree path exists but is not a git worktree: $path" >&2
      return 1
    fi
    current_branch="$(git -C "$path" branch --show-current)"
    if [ "$current_branch" != "$branch" ]; then
      echo "worktree path is on $current_branch, expected $branch: $path" >&2
      return 1
    fi
  else
    mkdir -p "$worktree_dir"
    if git show-ref --verify --quiet "refs/heads/${branch}"; then
      git worktree add "$path" "$branch" >&2
    else
      git worktree add -b "$branch" "$path" >&2
    fi
  fi

  if [ -n "$(git -C "$path" status --porcelain)" ]; then
    echo "worktree is dirty; refusing to run: $path" >&2
    return 1
  fi

  sync_takt_config "$path"
  printf '%s\n' "$path"
}

ids="$(select_issue_ids)"
if [ -z "$ids" ]; then
  echo "no runnable TODO issues"
  exit 0
fi

if [ "$dry_run" != "true" ]; then
  mkdir -p "$log_dir"
fi

failed_count=0
succeeded_count=0
last_exit_code=0
summary_rows=()

add_summary_row() {
  status="$1"
  id="$2"
  exit_code="$3"
  log_file="$4"
  worktree_path="$5"

  summary_rows+=("${status}"$'\t'"${id}"$'\t'"${exit_code}"$'\t'"${log_file}"$'\t'"${worktree_path}")
}

print_summary() {
  if [ "${#summary_rows[@]}" -eq 0 ]; then
    return
  fi

  echo
  echo "Summary:"
  echo "- succeeded: ${succeeded_count}"
  echo "- failed: ${failed_count}"
  echo
  echo "Details:"

  for row in "${summary_rows[@]}"; do
    IFS=$'\t' read -r status id exit_code row_log_file row_worktree_path <<<"$row"
    if [ "$exit_code" = "0" ]; then
      echo "- #${id}: ${status}"
    else
      echo "- #${id}: ${status} (exit-code=${exit_code})"
    fi
    echo "  log: ${row_log_file}"
    if [ -n "$row_worktree_path" ]; then
      echo "  worktree: ${row_worktree_path}"
    fi
  done
}

while IFS= read -r id; do
  [ -n "$id" ] || continue

  run_id="$(date +%Y%m%d-%H%M%S)-issue-${id}"
  started_at="$(iso_now)"
  log_file="${log_dir}/${run_id}.log"
  run_dir="$repo_root"
  worktree_path=""

  if [ "$dry_run" = "true" ]; then
    title="$("$issue_bin" show "$id" --format json | jq -r '.title')"
    if [ "$use_worktree" = "true" ]; then
      printf 'would run #%s: %s (workflow=%s, task-format=%s, worktree=%s)\n' "$id" "$title" "$workflow" "$task_format" "${worktree_dir}/issue-${id}"
    else
      printf 'would run #%s: %s (workflow=%s, task-format=%s)\n' "$id" "$title" "$workflow" "$task_format"
    fi
    continue
  fi

  if [ "$use_worktree" = "true" ]; then
    if worktree_path="$(ensure_worktree "$id")"; then
      run_dir="$worktree_path"
    else
      exit_code="$?"
      finished_at="$(iso_now)"
      "$issue_bin" metadata set "$id" \
        workflow="$workflow" \
        workflow-status=failed \
        run-id="$run_id" \
        started-at="$started_at" \
        finished-at="$finished_at" \
        exit-code="$exit_code" \
        log-file="$log_file" >/dev/null
      echo "issue #${id}: worktree setup failed (exit-code=${exit_code})" >&2
      failed_count=$((failed_count + 1))
      last_exit_code="$exit_code"
      add_summary_row failed "$id" "$exit_code" "$log_file" "$worktree_path"
      if [ "$continue_on_error" != "true" ]; then
        break
      fi
      echo "continuing after failure because --continue-on-error is set" >&2
      continue
    fi
  fi

  echo "running issue #${id} with workflow ${workflow} (run-id=${run_id})"
  metadata_args=(
    metadata set "$id"
    workflow="$workflow"
    workflow-status=running
    run-id="$run_id"
    started-at="$started_at"
    log-file="$log_file"
  )
  if [ -n "$worktree_path" ]; then
    metadata_args+=(worktree-path="$worktree_path")
  fi
  "$issue_bin" "${metadata_args[@]}" >/dev/null

  task_file="$(mktemp)"

  "$issue_bin" show "$id" --format "$task_format" >"$task_file"

  if (cd "$run_dir" && "$takt_bin" -w "$workflow" <"$task_file") > >(tee "$log_file") 2> >(tee -a "$log_file" >&2); then
    finished_at="$(iso_now)"
    "$issue_bin" metadata set "$id" \
      workflow-status=success \
      finished-at="$finished_at" \
      exit-code=0 \
      log-file="$log_file" >/dev/null
    echo "issue #${id}: workflow succeeded"
    succeeded_count=$((succeeded_count + 1))
    add_summary_row success "$id" 0 "$log_file" "$worktree_path"
  else
    exit_code="$?"
    finished_at="$(iso_now)"
    "$issue_bin" metadata set "$id" \
      workflow-status=failed \
      finished-at="$finished_at" \
      exit-code="$exit_code" \
      log-file="$log_file" >/dev/null
    echo "issue #${id}: workflow failed (exit-code=${exit_code})" >&2
    failed_count=$((failed_count + 1))
    last_exit_code="$exit_code"
    add_summary_row failed "$id" "$exit_code" "$log_file" "$worktree_path"
    if [ "$continue_on_error" != "true" ]; then
      rm -f "$task_file"
      task_file=""
      break
    fi
    echo "continuing after failure because --continue-on-error is set" >&2
  fi

  rm -f "$task_file"
  task_file=""
done <<<"$ids"

print_summary

if [ "$failed_count" -gt 0 ]; then
  echo "completed with ${failed_count} failed issue(s)" >&2
  exit "$last_exit_code"
fi
