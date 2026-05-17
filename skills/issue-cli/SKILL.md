---
name: issue-cli
description: ローカルIssueをissue-cliで管理する。タスクをIssue化して残したいとき、現在のIssueを確認するとき、issue/<id>ブランチで作業するとき、.issues/<id>.yamlを直接読み書きするときに使用する。
---

# issue-cli

リポジトリ直下の `.issues/<id>.yaml` でIssueを管理する小さなCLI/TUI。GitHub Issuesではなくローカルファイルで完結する。

## 使用判断

- ユーザが「Issueに残しておいて」「タスク化」「TODOを記録」など永続化を求めたとき → `issue create --title "..."`
- ユーザが「いまどのIssueをやっている?」「現在のIssue」と聞いたとき、または `issue/<id>` ブランチ上にいるとき → `issue` (引数なし) または `issue _show <id>`
- 特定IDの中身を確認するとき → `issue _show <id>` か `.issues/<id>.yaml` をRead
- 一覧・ステータス更新・削除はTUI主体。非対話で扱う場合は `.issues/*.yaml` を直接走査・編集する

## 使えるコマンド (非対話)

```sh
issue create --title "<タイトル>"   # status=TODO で新規作成 (IDは max(existing)+1 を採番)
issue _show <id>                    # 詳細を標準出力にプリント
issue                               # issue/<id> ブランチ上なら詳細を表示、それ以外はTUI起動
```

`issue list` や引数なしの `issue create` は対話TUIを起動するため、Claudeからは呼ばない。

## 保存形式

`.issues/<id>.yaml` (リポジトリルート基準。`git rev-parse --show-toplevel` を解決できない場合はCWD)。

```yaml
id: 1
title: ...
status: TODO        # TODO | In Progress | Reviews | Done のいずれか (この表記でなければパース失敗)
description: |
  ...
references:
  - https://...
scope:
  - "@path/to/file"
created_at: 2026-05-17T10:00:00+09:00
updated_at: 2026-05-17T10:00:00+09:00
```

- IDは削除後も再利用されない (常に `max(existing)+1`)
- YAMLを直接編集したときは `updated_at` も合わせて更新する (`issue` CLIから保存した場合は自動で上書きされる)
- フィールドを追加する場合は `internal/model/issue.go` の構造体にyamlタグを付ける必要がある

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

ステータス変更は対象YAMLの `status:` 行 (および `updated_at:`) を書き換える。CLI経由の更新はTUI (`s` キー) のみなので、Claudeから一括変更するときはファイル編集が早い。
