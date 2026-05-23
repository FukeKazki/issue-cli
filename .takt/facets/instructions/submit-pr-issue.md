# PR 文面の作成

検証済みの変更を見て、commit message / PR title / PR body を 3 つのファイルに書き出してください。
このステップで git や gh は実行しないでください。実行は次の script step が deterministic に行います。

参照元:

- `{task}`
- `plan.md`, `coder-scope.md`
- 実際の diff (`git diff` / `git diff --staged` で確認可能)

## 出力先 (固定パス)

事前に `mkdir -p .takt/work` を実行してから、以下を書き出してください。

| ファイル | 内容 |
|---|---|
| `.takt/work/commit-msg.txt` | commit メッセージ。1 行目に subject、2 行目空行、3 行目以降に詳細 |
| `.takt/work/pr-title.txt`   | PR タイトル 1 行。末尾に改行なし可 |
| `.takt/work/pr-body.md`     | PR 本文 (Markdown) |

## 内容の規約

- PR 本文には `Local issue: #<id> <title>` または `Refs: .issues/<id>.yaml` を書く。
- ローカル Issue 番号を GitHub Issue として `close #N` で扱わない。
- GitHub に対応する Issue がある場合だけ `close #N` を使う。
- commit メッセージは Conventional Commits 風 (`feat:` / `fix:` / `refactor:` …) を推奨。

## 完了報告

3 ファイルすべて書き終えたら `structured_output` で `files_ready: true` を返す。
1 つでも書けなかった場合は `files_ready: false` を返す (次の step は ABORT する)。
