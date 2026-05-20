---
name: issue-cli
description: ローカルIssueをissue-cliで管理する。タスクをIssue化して残したいとき、現在のIssueを確認するとき、issue/<id>ブランチで作業するとき、.issues/<id>.yamlを直接読み書きするときに使用する。
---

# issue-cli

リポジトリ直下の `.issues/<id>.yaml` でIssueを管理する小さなCLI/TUI。GitHub Issuesではなくローカルファイルで完結する。

## 使用判断

- ユーザが「Issueに残しておいて」「タスク化」「TODOを記録」など永続化を求めたとき → `issue new --title "..."`
- ユーザが「いまどのIssueをやっている?」「現在のIssue」と聞いたとき、または `issue/<id>` ブランチ上にいるとき → `issue` (引数なし) または `issue show <id>`
- 特定IDの中身を確認するとき → `issue show <id>` か `.issues/<id>.yaml` をRead
- 一覧・ステータス更新・削除はTUI主体。非対話で扱う場合は `issue list --format json` / `issue next --format json` か `.issues/*.yaml` を直接走査・編集する

## 使えるコマンド (非対話)

```sh
issue new --title "<タイトル>"                     # status=TODO で新規作成 (IDは max(existing)+1 を採番)
issue show <id>                                    # 詳細をプレーンテキストで標準出力にプリント
issue show <id> --format markdown|yaml|json        # 機械可読フォーマットで1件出力 (存在しないIDは非ゼロ終了)
issue list --format json [--status STATUS] [--all] # 一覧をJSON配列で出力 (フィルタ済み)
issue next --format json                           # 最小IDのTODO Issueを {"issue": ...} で返す (なければ {"issue": null}, 常に exit 0)
issue edit <id> --status <STATUS>                  # ステータスを更新 (大文字小文字不問。done/in-progress/review なども可)
issue                                              # issue/<id> ブランチ上なら詳細を表示、それ以外はTUI起動
```

`issue list` (フォーマット指定なし) や引数なしの `issue new` は対話TUIを起動するため、Claudeからは呼ばない。simple-takt 等のランナーへ流すときは `issue next --format json | simple-takt -w issue-dev` のように `--format json` を使う。

## 保存形式

`.issues/<id>.yaml` (リポジトリルート基準。`git rev-parse --show-toplevel` を解決できない場合はCWD)。

```yaml
id: 1
title: ...
status: TODO        # TODO | In Progress | Reviews | Done のいずれか (この表記でなければパース失敗)
type: Feature       # Bug | Feature | Docs | Refactor のいずれか。省略可 (空のときはキー自体が出力されない)
description: |
  ...
references:
  - https://...
scope:
  - "@path/to/file"
blocked_by:          # int 配列。空でも `blocked_by: []` で出力される
  - 2
  - 3
created_at: 2026-05-17T10:00:00+09:00
updated_at: 2026-05-17T10:00:00+09:00
```

- IDは削除後も再利用されない (常に `max(existing)+1`)
- YAMLを直接編集したときは `updated_at` も合わせて更新する (`issue` CLIから保存した場合は自動で上書きされる)
- フィールドを追加する場合は `internal/model/issue.go` の構造体にyamlタグを付ける必要がある
- `blocked_by` は依存先 Issue ID の配列 (int)。自分自身の ID や 0 以下を含む状態で `Save` するとバリデーションエラーになる。TUIフォーム以外から編集する場合は手書きで `.yaml` を更新する (CLIサブコマンドは未提供)
- `type` は作業の種類分類 (`Bug` / `Feature` / `Docs` / `Refactor`)。Status とは独立した属性で、省略可 (`omitempty`)。指定する場合は表記が完全一致でなければバリデーションエラーになる

## ブランチ規約

- 1 Issue = 1 ブランチ `issue/<id>`
- TUIの `c` キーが `git checkout` (なければ `-b` で作成) を実行する
- Claudeが手動で揃える場合は `git checkout -B issue/<id>`

## 非対話で一覧/編集する

```sh
# 未完了のID/title/status を流す
for f in .issues/*.yaml; do
  grep -E '^(id|title|status):' "$f"
  echo ---
done
```

ステータス変更は `issue edit <id> --status <STATUS>` (非対話) もしくは TUI の `s` キーで行える。複数件を一括で書き換える場合は対象YAMLの `status:` 行 (および `updated_at:`) を直接編集するのが早い。
