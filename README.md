# issue-cli

ローカル完結型の Issue マネージャ。Issue をリポジトリ内の `.issues/` 配下に
YAML として保存し、組み込みの TUI で閲覧し、id で Git ブランチを切り替えます。

リポジトリ名は `issue-cli` ですが、インストールされるバイナリ名は `issue` です。

## インストール

```sh
go install github.com/FukeKazki/issue-cli/cmd/issue@latest
```

または mise 経由:

```sh
mise use -g go:github.com/FukeKazki/issue-cli/cmd/issue@latest
```

`PATH` 上に `git` が必要です。

## 使い方

```
issue                                  # 現在の issue/<id> ブランチに対応する issue を表示。それ以外では一覧 TUI を開く
issue show <id> [--format markdown|yaml|json]
issue list [--all] [--status=STATUS] [--format json]
issue next [--format json]             # 自動化向け。次に着手すべき TODO issue を返す
issue new [--title TITLE]
issue edit <id> --status STATUS        # CLI から issue のステータスを更新（大文字小文字を区別しない）
issue metadata <id>                    # issue に紐づく自由形式のメタデータを表示
issue metadata set <id> k=v [k=v ...]  # キー/値ペアを issue のメタデータマップにマージ
issue metadata unset <id> k [k ...]    # 指定したキーを削除
issue metadata clear <id>              # メタデータマップを丸ごと削除
```

### `issue`（引数なし）

現在の Git ブランチが `issue/<id>` にマッチする場合、対応する issue の詳細
（タイトル、ステータス、説明、参照、スコープ、タイムスタンプ）を表示します。
それ以外のブランチでは、一覧 TUI を開きます（`issue list` と同じ）。

### `issue list`

オープンな issue（TODO / In Progress / Reviews）を表示する組み込み TUI を開
きます。左ペインが一覧、右ペインが詳細プレビューです。

| キー            | 動作                                                         |
| --------------- | ------------------------------------------------------------ |
| `Enter`         | issue の詳細を表示（読み取り専用。`q`/`Esc`/`Enter` で戻る） |
| `c`             | `issue/<id>` を `git checkout`（または作成）して一覧を抜ける |
| `n`             | 新規 issue を作成（TUI フォーム）                            |
| `e`             | 選択中の issue を編集（TUI フォーム）                        |
| `s`             | ステータスを変更（その後 `1`–`4` または `Enter`）            |
| `d`             | 選択中の issue を削除（確認あり）                            |
| `v`             | 詳細プレビューの表示切り替え                                 |
| `j` / `k`       | カーソル移動（`↓` / `↑` も可）                               |
| `g` / `G`       | 先頭 / 末尾へジャンプ                                        |
| `/`             | フィルタ（大文字小文字を区別しない部分一致）                 |
| `q` / `Esc`     | 終了                                                         |
| `Ctrl-C`        | 終了                                                         |

`Enter` は非破壊操作です。詳細表示を開くだけなので、`c`（checkout）を誤って
押す事故を避けられます。

フィルタ: `--all` で Done も含める、`--status="In Progress"` で 1 つのステー
タスに絞り込み。

### `issue new`

引数なしの場合 → TUI フォームを開く（タイトル / ステータス / 参照 / スコープ）。
`--title "..."` 付きの場合 → そのタイトルでステータス `TODO`、参照・スコープ
なしの issue を作成。

### `issue edit`

非インタラクティブなステータス更新です。変更はどの方向にも許可されます
（前方向のみの自動遷移ルールは `c` checkout のようなイベント駆動の自動遷移
にのみ適用されます）。

```sh
issue edit 13 --status DONE
issue edit #13 --status done
issue edit 13 --status in-progress
```

`--status` に指定できる値（大文字小文字を区別しない）:

| 正規値        | 別名                                                       |
| ------------- | ---------------------------------------------------------- |
| `TODO`        | `todo`                                                     |
| `In Progress` | `in progress`, `in-progress`, `in_progress`, `inprogress`  |
| `Reviews`     | `reviews`, `review`                                        |
| `Done`        | `done`                                                     |

`--status` は必須です。未知の値を渡すと、YAML には触れずに非ゼロで終了しま
す。

### `issue metadata`

issue に対する自由形式のキー/値属性です。CLI はキーや値を解釈しません。自動
化ランナー、スクリプト、人間が、必要な任意のコンテキスト（workflow id、
run id、PR url、計測値など）を CLI 側のスキーマ固定に縛られずに付与できます。

```sh
issue metadata 13                                                    # 表示
issue metadata set 13 workflow=issue-dev run-id=20260520-abc         # マージ
issue metadata set 13 result=success pr-url=https://github.com/.../42
issue metadata unset 13 error                                        # キーを削除
issue metadata clear 13                                              # すべて削除
```

`set` は既存マップへマージします（同じキーの値は上書き）。`unset` は指定し
たキーを削除し、マップが空になった場合は `metadata:` ブロック自体を YAML か
ら落とすので、残骸が残りません。値は文字列で、CLI はタイムスタンプや数値の
パースは行いません。フォーマットの責任は呼び出し側にあります。すべてのサブ
コマンドは `--format json|yaml|markdown` を受け付け、プレーンテキストのサマ
リの代わりに更新後の issue を出力できます。

## 自動化 / 機械可読出力

[`simple-takt`](https://github.com/FukeKazki/simple-takt) のようなランナー
にパイプで渡すために、非インタラクティブなサブコマンドは TUI を開く代わり
に構造化された出力を返します。既存の TUI 動作は変わりません。これらのフラ
グは指定された場合のみ有効になります。

| コマンド                                         | 出力                                                                            |
| ------------------------------------------------ | ------------------------------------------------------------------------------- |
| `issue show <id> --format markdown\|yaml\|json`  | issue を 1 件出力。id が無い／不明なら非ゼロで終了                              |
| `issue list --format json [--status STATUS]`     | issue の JSON 配列（`--all` / `--status` フィルタ適用後）                       |
| `issue next [--format json]`                     | エンベロープ `{"issue": {...}}`。TODO が残っていない場合は `{"issue": null}`   |
| `issue metadata <subcmd> <id> ... --format json` | 更新後の issue（mutation 適用後）を 1 件の JSON オブジェクトとして出力          |

`issue next` は最小 id の `TODO` issue を選びます（決定的）。常に 0 で終了
するので、下流のパイプは常に有効な JSON を受け取れます。

simple-takt にパイプする例:

```sh
issue next --format json | simple-takt -w issue-dev
```

## ストレージ

`.issues/<id>.yaml` — issue 1 件につきファイル 1 つ。

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
blocked_by:                    # other issue ids this one depends on (int array)
  - 2
  - 3
metadata:                      # optional; free-form string key/value pairs
  workflow: issue-dev          # any keys callers choose — the CLI does not interpret them
  run-id: 20260520-abc
  pr-url: https://github.com/owner/repo/pull/42
created_at: 2026-05-16T10:30:00+09:00
updated_at: 2026-05-16T10:30:00+09:00
```

TUI フォーム（`issue new` または一覧での `e`）には **BLOCKED BY** フィール
ドがあります。1 行につき issue id を 1 つ入力します。入力中は既存 issue の
オートコンプリートポップアップが表示されます（id の前方一致またはタイトル
の部分一致でフィルタされます）。`tab` / `enter` で選択した id を挿入できま
す。自己参照および 0 以下の id は保存時に拒否されます。

メタデータが無い issue は `metadata:` ブロックを書き出しません。`issue
metadata set` が遅延書き込みし、`unset` で最後のキーが消えた時点で再び落と
されます。

ID は `max(existing)+1` で採番されます。`.issues/` を git にコミットするか
は利用者の判断に任されます。CLI は `.gitignore` を触りません。

ブランチ切り替えは `git checkout` を直接呼ぶため、作業ツリーがクリーンでな
い場合は git 自身の警告とともに中断されます。

## Claude Code スキル

`skills/issue-cli/SKILL.md` は Claude Code 用のスキルで、エージェントに非イ
ンタラクティブな文脈からこの CLI を扱う方法（`issue new --title` をいつ呼ぶ
か、`.issues/<id>.yaml` を直接読むのはどんなときか、ブランチ命名規約など）
を教えます。

[apm](https://github.com/yoshinani-dev/apm) 経由でインストールするには、プ
ロジェクトの `apm.yml` に次を追記してください:

```yaml
dependencies:
  apm: [
    FukeKazki/issue-cli/skills/issue-cli
  ]
```

`apm install` を実行すると、利用先プロジェクトの `.claude/skills/issue-cli/`
（および `.agents/skills/issue-cli/`）配下に展開されます。
