# issue-cli アーキテクチャ

Go モジュール `github.com/FukeKazki/issue-cli`。Bubble Tea TUI + YAML-on-disk ストア + 薄い git ドライバで構成されるローカル Issue 管理 CLI。

## ビルド・テスト

```sh
go build ./cmd/issue-cli
go test ./...
go vet ./...
```

`gitx_test.go` は実際の `git` を使う (`t.Chdir` + temp repo)。`git` がない環境ではスキップされる。

## パッケージ構成

```
cmd/issue-cli/main.go      最小限のサブコマンドパーサ → internal/cli にディスパッチ
internal/model/issue.go    Issue 構造体、Status / Type enum、AdvanceStatus
internal/store/store.go    .issues/<id>.yaml への CRUD (tempfile + Rename で原子書き込み)
internal/gitx/gitx.go      git symbolic-ref / checkout のラッパ、issue/<id> ブランチ規約
internal/cli/              サブコマンドのオーケストレーション (list, new, show, edit, next, metadata, default)
internal/tui/              Bubble Tea モデル群 (list, form, detail_view, detail, status_picker, confirm)
internal/output/output.go  JSON / YAML / Markdown レンダラ (--format 用、TUI 非依存)
```

## データモデル

```go
type Issue struct {
    ID          int               `yaml:"id"`
    Title       string            `yaml:"title"`
    Status      Status            `yaml:"status"`          // TODO | In Progress | Reviews | Done
    Type        Type              `yaml:"type,omitempty"`   // Bug | Feature | Docs | Refactor (省略可)
    Description string            `yaml:"description"`
    References  []string          `yaml:"references"`
    Scope       []string          `yaml:"scope"`
    BlockedBy   []int             `yaml:"blocked_by"`
    Metadata    map[string]string `yaml:"metadata,omitempty"`
    CreatedAt   time.Time         `yaml:"created_at"`
    UpdatedAt   time.Time         `yaml:"updated_at"`
}
```

フィールド追加時は必ず `yaml:` タグを付ける。`Save` が `UpdatedAt` を自動スタンプする。

## Status ライフサイクル

`TODO → In Progress → Reviews → Done`

- `StatusRank` で順序を定義。`AdvanceStatus(target)` は前進のみ (rank が上がる方向)。
- checkout 時の自動遷移 (`advanceOnCheckout`) は `AdvanceStatus` を使う — 前進のみ。
- TUI の `s` キーや `issue-cli edit --status` は任意方向の直接代入 — ユーザー明示操作。
- `ParseStatus` は正規形のみ受理 (ストア検証用)。`ParseStatusFromCLI` は大文字小文字・エイリアス許容 (CLI 入力用)。

## ストア規約

- ファイルパス: `<repo-root>/.issues/<id>.yaml`
- repo root は `git rev-parse --show-toplevel` (失敗時は CWD)。
- `NextID` = `max(既存ID) + 1`。削除後も ID は再利用しない。
- `Save` は `validate` → `EnsureDir` → `tempfile + Rename`。
- `validate`: ID > 0、Title 非空、Status 正規形、Type は空文字 OK・非空なら正規形、BlockedBy に自身を含めない。

## CLI 制御フロー

### List ループ

TUI は副作用を持たない — `ListResult{Action, IssueID}` を返すだけ。`cli.List` ループが `tui.RunForm` / `s.Save` / `s.Delete` 等を呼ぶ。

新しいキーを追加するとき:
1. `tui/list.go` に `ListAction*` 定数を追加
2. `updateBrowsing` で該当キーにセット
3. `cli/list.go` のループで副作用を処理

`Enter` は非破壊 (詳細表示)。`c` が checkout。破壊的操作を `Enter` に割り当てない。

### Default (引数なし)

`issue/<id>` ブランチ上 → そのIssueの詳細表示。それ以外 → `List(nil)`。

### Checkout

checkout はループを抜ける唯一のアクション。成功時に `advanceOnCheckout` が TODO → In Progress に自動遷移。

## 出力パス

| 文脈 | レンダラ |
|------|----------|
| TUI プレビュー / `Default()` / `Show()` (--format なし) | `tui.RenderDetail` (lipgloss スタイル付き plain text) |
| `--format` 付き (`show`, `list`, `next`) | `internal/output` (JSON/YAML/Markdown) |

`RenderDetail` は副作用なし。`internal/output` は `internal/tui` に依存しない。

## git ブランチ規約

- ブランチ名: `issue/<id>` (例: `issue/42`)
- `CurrentIssueID()` で現在のブランチから ID を解析 (不一致時は 0)。
- `CheckoutIssue(id)` はブランチが既存なら `git checkout`、なければ `git checkout -b`。

## 依存ライブラリ

- `github.com/charmbracelet/bubbletea` — TUI フレームワーク
- `github.com/charmbracelet/bubbles` — TUI コンポーネント (textinput 等)
- `github.com/charmbracelet/lipgloss` — ターミナルスタイリング
- `github.com/mattn/go-runewidth` — 全角文字の表示幅計算
- `gopkg.in/yaml.v3` — YAML シリアライズ

## 実装時の注意

- テストやテスト環境はタスクに明示がない限り増やさない。
- TUI の新しい副作用は `ListAction*` パターンを守る。
- `model.Issue` にフィールドを追加するときは `yaml:` タグ必須、`json:` タグも付ける。
- `store.validate` を通過するよう不変条件を守る。
- `.issues/` ディレクトリの gitignore はユーザー選択。CLI は `.gitignore` を操作しない。
