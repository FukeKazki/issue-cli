# Automation Runner 方針

## 背景

issue-cli は local-first なタスク管理を担い、simple-takt は AI ワークフローランナーを担う。両者をつなぐ最初の仕組みとして、重い常駐プロセスや専用キュー基盤ではなく、shell script で薄く接続する。

この方針は `docs/product/vision.md` の次の原則に合わせる。

- Local-first: ローカル環境だけで完結する。
- AI-readable: issue の内容を Agent が読みやすい形で渡す。
- Simple over Enterprise: 個人開発向けに、まずは単純な仕組みを優先する。
- Markdown-native: デフォルトでは `issue show --format markdown` を task input として扱う。

## 責務分担

- issue-cli: issue の source of truth。title、description、scope、Issue status、metadata を管理する。
- simple-takt: 1つの issue を受け取り、計画、実装、検証、PR 作成などの workflow を実行する。
- runner script: 実行対象の issue を選び、simple-takt に渡し、workflow の実行状態を issue metadata に記録する。

## Issue status と workflow status は分ける

`status` は Issue 自体の状態を表す。workflow の claim / running / failed / success とは別軸として扱う。

Issue status の例:

- `TODO`: タスクとして未完了。
- `In Progress`: 人間または Agent がタスク自体を作業中として扱う状態。
- `Reviews`: PR や成果物の確認待ち。
- `Done`: 完了。

workflow status の例:

- `queued`
- `running`
- `success`
- `failed`
- `blocked`
- `aborted`

runner script は Issue status を claim の代わりに使わない。workflow の状態は `metadata.workflow-status` に記録する。

## 現在の script

現在の実装は `tools/issue-takt-run.sh`。

主な挙動:

- `issue list --status TODO --format json` から実行候補を選ぶ。
- `metadata.workflow-status` が `running` または `queued` の issue は選ばない。
- `issue show <id> --format markdown` を simple-takt に渡す。
- 実行開始時に metadata へ `workflow-status=running`、`run-id`、`started-at`、`log-file` などを記録する。
- simple-takt 実行時に `ISSUE_ID`、`ISSUE_RUN_ID`、`ISSUE_WORKFLOW`、`ISSUE_BIN`、`ISSUE_REPO_ROOT`、`ISSUE_RUN_DIR`、`ISSUE_WORKTREE_PATH`、`ISSUE_TASK_FORMAT`、`ISSUE_LOG_FILE` を環境変数として渡す。
- 成功時は `workflow-status=success`、`finished-at`、`exit-code=0` を記録する。
- 失敗時は `workflow-status=failed`、`finished-at`、`exit-code` を記録する。
- デフォルトでは失敗時に終了し、`--continue-on-error` 指定時は次の issue に進む。
- `--until-empty` 指定時は、実行後に TODO 候補を再取得し、実行可能な TODO がなくなるまで直列で続ける。
- `--worktree` 指定時は issue ごとの git worktree を用意し、その中で simple-takt を実行する。
- 実行後に succeeded / failed 件数、issue ID、exit code、log path、worktree path を summary として出力する。
- Issue status は変更しない。

`issue` コマンドはデフォルトで PATH 上の `issue` を使う。別の binary を使いたい場合は `ISSUE_BIN=/path/to/issue` で明示する。

例:

```sh
tools/issue-takt-run.sh --dry-run --limit 1
tools/issue-takt-run.sh --workflow issue-dev --limit 1
tools/issue-takt-run.sh --limit 3 --continue-on-error
tools/issue-takt-run.sh --until-empty --continue-on-error
tools/issue-takt-run.sh --limit 2 --worktree
tools/issue-takt-run.sh --issue 21
```

## 現在あえて入れていないこと

### `--limit` 省略時の挙動

`--limit` を指定しない場合は 1件だけ実行する。TODO がなくなるまで自動実行する挙動にはしていない。

理由:

- うっかり全 TODO を流す事故を避ける。
- まずは小さく実行して、metadata と simple-takt の接続を確認しやすくする。

TODO がなくなるまで実行したい場合は、明示的に `--until-empty` を指定する。`--until-empty` では `--limit` 省略時の上限を外す。`--limit N` を併用した場合は「TODO がなくなるまで、または N 件に達するまで」実行する。

同じ runner 起動内では、成功・失敗にかかわらず一度試した issue は再選択しない。これにより、workflow が Issue status を変更しなかった場合や `--continue-on-error` で失敗を記録して進む場合でも、同じ TODO を即座に繰り返し実行しない。

### 並列実行

現在はすべて直列実行する。`--limit N` を指定しても、1件の simple-takt 実行が終わってから次の issue に進む。

理由:

- atomic な claim がない。
- metadata 更新やログ集約が干渉しやすい。

並列化する場合は、`--worktree` を前提に issue ごとの log directory 分離や summary 出力を整える。

### エラー時の継続

デフォルトでは、途中で simple-takt が失敗したら対象 issue に失敗 metadata を記録して script 全体を終了する。

夜間バッチのように失敗を記録しながら流し続けたい場合は、`--continue-on-error` を指定する。

想定する運用:

- デフォルト: 失敗したら止まる。
- `--continue-on-error`: 失敗を記録して次へ進む。
- `--until-empty --continue-on-error`: TODO がなくなるまで、失敗も記録しながら走る。

`--continue-on-error` を指定した場合でも、失敗した issue が1件以上あれば script の最終 exit code は非ゼロにする。これにより、cron や CI から「完走はしたが失敗を含む」ことを検知できる。

### branch / worktree の切り替え

デフォルトでは現在の working tree で simple-takt を起動する。`--worktree` を指定した場合だけ、runner が issue ごとの worktree を作成または再利用する。

方針:

- simple-takt には worktree 管理を期待しない。
- worktree は workflow の中身ではなく実行環境の隔離なので、runner tool が管理する。
- branch 名は `issue/<id>`、worktree path はデフォルトで `../issue-worktrees/issue-<id>` とする。
- 既存 worktree が dirty な場合は実行しない。
- worktree 側には `.takt/config.yaml`、`.takt/workflows`、`.takt/facets` を同期する。
- 実行 metadata には `worktree-path` を記録する。

例:

```sh
tools/issue-takt-run.sh --worktree --limit 1
tools/issue-takt-run.sh --worktree --worktree-dir ../issue-worktrees --limit 2
```

内部的には次のような worktree を使う。

```sh
git worktree add "../issue-worktrees/issue-$id" -b "issue/$id"
```

