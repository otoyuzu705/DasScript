# CLAUDE.md

このファイルは、リポジトリで作業する際に Claude Code (claude.ai/code) へのガイダンスを提供します。

## プロジェクト概要

DasScript は、Starlark スクリプトを使って Daslight5 の `.dvc5` プロジェクトファイル（XML形式の照明制御ファイル）をプログラムから操作するための CLI ツール兼 Go ライブラリです。設計は `Docs/Architecture.md` と `Docs/StarlarkAPI.md` に日本語で詳細に記述されています。現時点では Go のソースコードは未実装です。

## 前提

- 質問、何かを伝える場合は必ず日本語を用いること
- コメントは日本語で書くこと
- 無断でcommit, pushはしないこと
- 仕様を詰める際は質問攻めをして解像度を上げること

## ビルド・テストコマンド

`go.mod` 初期化後（モジュール名は `github.com/otoyuzu705/DasScript`）:

```bash
go build ./cmd/dvc5       # CLI バイナリのビルド
go test ./...             # 全テスト実行
go test ./application/... # 特定パッケージのテスト実行
go vet ./...              # 静的解析
```

利用する Go のバージョンは `go.mod` の `go` ディレクティブに従ってください。

## アーキテクチャ

本プロジェクトは **ヘキサゴナルアーキテクチャ（Ports & Adapters）** を採用しています。依存の方向は厳密に一方向です:

```
cmd/dvc5  →  application/usecase  →  domain/port (インターフェース)
                                           ↑ 実装
                                    infrastructure/{dvc5xml,starlark}
```

**厳守ルール:**
- `domain/*` — 外部依存ゼロ。標準ライブラリのみ使用可。
- `domain/model` — 純粋なデータ構造体。XML タグ・バリデーションロジック禁止。
- `domain/port` — インターフェース定義のみ（`ProjectReader`、`ProjectWriter`、`ScriptRunner`）。
- `application/usecase` — ポートにのみ依存。インフラ具象型への直接依存禁止。
- `infrastructure/*` — 外部ライブラリの import および XML/Starlark の詳細を管理。
- インフラ構造体はすべてコンパイル時インターフェース検証を含めること。

## ディレクトリ構成（予定）

```
dvc5/
├── cmd/dvc5/main.go
├── domain/
│   ├── model/project.go          # Project, Scenes, Bank, Scene 構造体 + センチネルエラー
│   └── port/                     # ProjectReader, ProjectWriter, ScriptRunner インターフェース
├── application/usecase/          # load_project, save_project, rename_bank, rename_scene, run_script
├── infrastructure/
│   ├── dvc5xml/                  # XML アダプター（reader.go, writer.go, mapper.go）
│   └── starlark/                 # Starlark アダプター（runner.go, builtins.go, binding/）
└── internal/xmlutil/
```

## ユースケースのパターン

全ユースケースは**関数**（構造体メソッドではない）で、`XxxInput` / `XxxOutput` 構造体を使います:

```go
func LoadProject(in LoadProjectInput) (LoadProjectOutput, error)
```

ポート（インターフェース）は Input 構造体のフィールドとして注入します。コンストラクタ DI は使いません。

## 主要なドメイン型

```go
type Project struct { Path, Version string; Scenes Scenes }
type Scenes  struct { Banks []Bank }
type Bank    struct { DASUID, Name string; Index int; Scenes []Scene }
type Scene   struct { DASUID, Name string; Index int }
```

センチネルエラー（`domain/model/errors.go`）: `ErrEmptyName`、`ErrNotFound`、`ErrInvalidIndex`。

## インフラ層の注意事項

**dvc5xml アダプター:**
- XML タグ付きのプライベート構造体（`xmlDLMFILE` 等）はドメインモデルと完全に分離し、マッピングは `mapper.go` に集約。
- Phase 1: PATCHS 要素は生バイトとして読み込むだけでデコードしない（Base64+zlib は Phase 2 以降）。
- ラウンドトリップの再現性のため、属性順序を保持すること。

**starlark アダプター:**
- ドメイン型を `starlark.Value` 実装としてラップ: `StarProject`、`StarScenes`、`StarBank`、`StarScene`。
- スクリプトに公開するグローバル関数: `load_project()`、`save_project()`、`log()`。
- Phase 1 で変更可能な操作は Bank/Scene の `.rename()` のみ。その他のフィールドはすべて読み取り専用。

## テスト規約

- ユニットテスト: 同一パッケージ内・テーブル駆動・ポートインターフェースを実装したカスタムモック構造体を使用（モックライブラリ不使用）。
- `dvc5xml` の結合テスト: `testdata/` に実際の `.dvc5` ファイルを配置し、Read → toModel() → fromModel() → marshal → 文字列比較でラウンドトリップ検証。
- `cmd/dvc5` は Phase 1 ではテストなし。

## Phase 1 スコープ

実装対象は **Load**、**Save**、**RenameBank**、**RenameScene**、**RunScript** の5操作のみ。CRUD・タイムライン・PATCHS デコードは Phase 2 以降。

## 外部依存ライブラリ（Phase 1）

| パッケージ | 用途 |
|-----------|------|
| `go.starlark.net/starlark` | Starlark ランタイム |
| `github.com/spf13/cobra` | CLI フレームワーク |
| `encoding/xml`（標準ライブラリ） | XML パース |
