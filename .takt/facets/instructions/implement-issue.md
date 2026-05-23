# 実装

以下のタスクと `plan.md` に従って、必要な変更だけを実装してください。
Report Directory のレポートと実際のファイル内容を優先し、他のレポートディレクトリは参照しないでください。

### タスク
{task}

守ること:

- タスクの description / scope から外れる変更は避ける。必要な場合は理由を `coder-scope.md` に書く。
- タスクに明示がない限り、新しいテストやテスト環境は増やさない。
- `model.Issue` にフィールドを追加するときは `yaml:` タグを付ける。
- TUI の新しい副作用は `ListAction*` を介して `cli/list.go` 側で扱う。

検証は次の `verify` ステップが `go vet ./... && go test ./...` を実行します。
実装中は必要に応じて部分テストや `go build ./cmd/issue-cli` だけ実行してください。

## 完了判定 (structured_output)

次ステップへの routing は `structured_output` (JSON) で判定します。
`coder-scope.md` の文面ではなく、必ず次の bool 3 つを返してください。

| フィールド | true にすべき条件 |
|---|---|
| `completed` | 実装できて検証ステップに進める |
| `blocked` | 進行不能な理由があり ABORT したい |
| `needs_user_input` | ユーザー入力を待って同じ step に戻りたい |

組み合わせ例:

- 成功: `completed=true, blocked=false, needs_user_input=false` → verify へ
- 失敗: `completed=false, blocked=true, needs_user_input=false` → ABORT
- 要確認: `completed=false, blocked=false, needs_user_input=true` → implement を再ループ
- 継続: `completed=false, blocked=false, needs_user_input=false` → implement を再ループ

## `coder-scope.md` (人間用の記録)

`coder-scope.md` には、少なくとも次を含めてください。routing はもうここを読みません。

- タスクの要約
- 変更したファイル
- 影響範囲と scope 外変更の理由
- 実施した確認
