# dvc5 Go ホスト実装 アーキテクチャ仕様書

| 項目 | 内容 |
|---|---|
| 文書バージョン | 1.0.2 |
| 対象フェーズ | Phase 1（読み込み・保存・リネーム） |
| 対象実装言語 | Go 1.22 以上 |
| 対象読者 | 本プロジェクトの実装者 |
| 最終更新 | 2026-04 |

---

## 目次

1. [アーキテクチャ選定](#1-アーキテクチャ選定)
2. [依存関係ルール](#2-依存関係ルール)
3. [ディレクトリ構成](#3-ディレクトリ構成)
4. [各層の仕様](#4-各層の仕様)
   - 4.1 [domain/model](#41-domainmodel)
   - 4.2 [domain/port](#42-domainport)
   - 4.3 [application/usecase](#43-applicationusecase)
   - 4.4 [infrastructure/dvc5xml](#44-infrastructuredvc5xml)
   - 4.5 [infrastructure/starlark](#45-infrastructurestarlark)
   - 4.6 [cmd/dvc5](#46-cmddvc5)
5. [ライブラリとしての利用](#5-ライブラリとしての利用)
6. [テスト方針](#6-テスト方針)
7. [採用ライブラリ](#7-採用ライブラリ)
8. [Phase 2 以降の拡張ポイント](#8-phase-2-以降の拡張ポイント)
9. [変更履歴](#9-変更履歴)

---

## 1. アーキテクチャ選定

### 採用: Hexagonal Architecture（ポート & アダプター）

本ツールには以下の特性がある。

- 永続化層（DB）を持たない
- HTTP インターフェースを持たない（Phase 1）
- 差し替えが必要な外部依存は「XML パーサー」と「Starlark ランタイム」の 2 つのみ
- CLI ツールとしての利用と、Go ライブラリとしての利用を両立する必要がある

クリーンアーキテクチャは UI・DB・外部サービスを複数抱える大規模アプリ向けであり、
本ツールには過剰な抽象層を生む。

Hexagonal Architecture は「ポート（interface）」と「アダプター（実装）」という
2 概念で差し替え可能な境界を表現でき、Go の interface 機構と直接対応する。
上記特性に対して最も適合するアーキテクチャと判断した。

---

## 2. 依存関係ルール

### 依存の方向

```
cmd/dvc5
    │
    ▼
application/usecase
    │
    ▼
domain/port  (interface 定義)
    ▲
    │  implements
infrastructure/*  (dvc5xml / starlark)
```

`domain/model` はすべての層から参照されるが、`domain/model` 自身は他の層を参照しない。

### 禁止事項

以下の依存を作ってはならない。違反はコードレビューで差し戻す。

| 禁止される依存方向 | 理由 |
|---|---|
| `domain/*` → `infrastructure/*` | domain は外部実装を知らない |
| `domain/*` → `application/*` | domain は上位層を知らない |
| `domain/*` → 標準ライブラリ以外のサードパーティ | 外部依存ゼロを保つ |
| `application/usecase` → `infrastructure/*` | usecase は port interface 経由でのみ依存する |
| `application/usecase` → `cmd/*` | 上位層への依存禁止 |

### コンパイル時アサーション

各アダプターは以下のアサーションを必ずファイル先頭に記述し、
interface の実装漏れをコンパイル時に検出する。

```go
var _ port.ProjectReader = (*Reader)(nil)
var _ port.ProjectWriter = (*Writer)(nil)
var _ port.ScriptRunner  = (*Runner)(nil)
```

---

## 3. ディレクトリ構成

```
dvc5/
├── cmd/
│   └── dvc5/
│       └── main.go               # CLI エントリポイント
│
├── domain/
│   ├── model/
│   │   ├── project.go            # Project struct
│   │   ├── scene.go              # Scenes / Bank / Scene struct
│   │   └── errors.go             # ドメインエラー定数
│   └── port/
│       ├── project_reader.go     # ProjectReader interface
│       ├── project_writer.go     # ProjectWriter interface
│       └── script_runner.go      # ScriptRunner interface
│
├── application/
│   └── usecase/
│       ├── load_project.go
│       ├── save_project.go
│       ├── rename_bank.go
│       ├── rename_scene.go
│       └── run_script.go
│
├── infrastructure/
│   ├── dvc5xml/
│   │   ├── reader.go             # ProjectReader 実装
│   │   ├── writer.go             # ProjectWriter 実装
│   │   └── mapper.go             # XML struct ↔ domain/model 変換
│   └── starlark/
│       ├── runner.go             # ScriptRunner 実装
│       ├── builtins.go           # Starlark ビルトイン関数の登録
│       └── binding/
│           ├── project.go        # StarProject 型（starlark.Value 実装）
│           ├── scenes.go         # StarScenes / StarBank / StarScene 型
│           └── util.go           # starlark.Value 変換ユーティリティ
│
└── internal/
    └── xmlutil/
        └── xmlutil.go            # XML 属性読み書きユーティリティ（非公開）
```

### 命名規約

| 対象 | 規約 | 例 |
|---|---|---|
| パッケージ名 | 小文字、単語は詰める | `dvc5xml`, `usecase` |
| ファイル名 | スネークケース | `rename_bank.go` |
| 公開型 | アッパーキャメルケース | `ProjectReader`, `StarProject` |
| 非公開型 | ローワーキャメルケース | `xmlDLMFILE`, `xmlBank` |
| インターフェース名 | 動作を表す名詞 + er | `ProjectReader`, `ScriptRunner` |

---

## 4. 各層の仕様

### 4.1 `domain/model`

#### 目的

dvc5 プロジェクトのビジネスオブジェクトを表現する。
外部依存ゼロの純粋な Go struct のみで構成し、
「ファイルフォーマットの詳細」ではなく「照明プロジェクトとして意味のある構造」を表現する。

#### 構造体仕様

```go
// domain/model/project.go
package model

// Project は dvc5 プロジェクトファイル全体を表す。
type Project struct {
    Path    string // 元ファイルのパス。save 時の出力先デフォルト値として使用する
    Version string // DASBUILD 属性値（例: "24.1205.158.90"）
    Scenes  Scenes
}
```

```go
// domain/model/scene.go
package model

// Scenes は SCENES 要素全体を表す。
type Scenes struct {
    Banks []Bank
}

// Bank は BANK 要素を表す。
// Index は Scenes.Banks 内での 0 始まり位置であり、
// XML への書き出し時に NBSCENE 属性の更新に使用する。
// Scenes は Bank 内のシーンリストであり、順序が XML の出現順と一致する。
type Bank struct {
    DASUID string
    Name   string
    Index  int     // 読み取り専用。usecase 層が更新する
    Scenes []Scene // Bank 内のシーンリスト。NBSCENE 属性は len(Scenes) から算出する
}

// Scene は SCENE 要素を表す。
// Phase 1 では Name と DASUID のみを扱う。
type Scene struct {
    DASUID string
    Name   string
    Index  int    // 読み取り専用
}
```

#### エラー定数仕様

```go
// domain/model/errors.go
package model

import "errors"

var (
    ErrEmptyName    = errors.New("name must not be empty")
    ErrNotFound     = errors.New("not found")
    ErrInvalidIndex = errors.New("index out of range")
)
```

`errors.Is` で比較可能な sentinel error として定義する。
エラーメッセージのラップは `fmt.Errorf("...: %w", model.ErrNotFound)` を使用する。

#### 規約

- `domain/model` 内でのバリデーションは行わない。バリデーションは `application/usecase` が担う
- Phase 2 以降に追加されるフィールド（`Speed`, `FadeIn` 等）もこのパッケージに追加する
- XML タグ（`` `xml:"..."` ``）は記述しない。XML 変換は `infrastructure/dvc5xml` が担う

---

### 4.2 `domain/port`

#### 目的

`application/usecase` と `infrastructure/*` の境界を interface として定義する。
実装を持たず、interface 宣言のみで構成する。

#### interface 仕様

```go
// domain/port/project_reader.go
package port

import (
    "context"
    "github.com/yourname/dvc5/domain/model"
)

// ProjectReader は dvc5 ファイルを読み込み model.Project を返す。
type ProjectReader interface {
    Read(ctx context.Context, path string) (*model.Project, error)
}
```

```go
// domain/port/project_writer.go
package port

import (
    "context"
    "github.com/yourname/dvc5/domain/model"
)

// ProjectWriter は model.Project を dvc5 ファイルとして書き出す。
type ProjectWriter interface {
    // Write は project.Path を出力先として書き出す。
    Write(ctx context.Context, project *model.Project) error
    // WriteTo は指定した path に書き出す。
    WriteTo(ctx context.Context, project *model.Project, path string) error
}
```

```go
// domain/port/script_runner.go
package port

import (
    "context"
    "github.com/yourname/dvc5/domain/model"
)

// ScriptRunner は Starlark スクリプトを実行し、変更後の Project を返す。
// スクリプト内で save_project が呼ばれた場合は Runner 側で処理する。
type ScriptRunner interface {
    Run(ctx context.Context, scriptPath string, project *model.Project) (*model.Project, error)
}
```

#### 規約

- Phase 2 で機能を追加する場合は、既存 interface へのメソッド追加ではなく、
  新しい interface を追加することを原則とする（既存アダプターへの影響を最小化するため）
- ただし論理的に同一の責務であれば既存 interface への追加を許容する

---

### 4.3 `application/usecase`

#### 目的

ポートを通じてドメインオブジェクトを操作するビジネスフローを実装する。
HTTP・CLI・ライブラリのいずれの呼び出し元にも依存しない。

#### 実装規約

- ユースケースは **関数** として定義する（struct + メソッドは使わない）
- 入力は `XxxInput` struct、出力は `XxxOutput` struct と `error` の 2 値返しに統一する
- ポートの interface は Input struct のフィールドとして受け取る（コンストラクタ DI は使わない）

#### Phase 1 ユースケース一覧

| 関数名 | Input | Output | 説明 |
|---|---|---|---|
| `LoadProject` | `path string`, `Reader port.ProjectReader` | `*model.Project` | ファイルを読み込む |
| `SaveProject` | `*model.Project`, `path string`, `Writer port.ProjectWriter` | — | ファイルを保存する |
| `RenameBank` | `*model.Project`, `oldName string`, `newName string` | `*model.Project` | バンクをリネームする |
| `RenameScene` | `*model.Project`, `bankName string`, `oldName string`, `newName string` | `*model.Project` | シーンをリネームする |
| `RunScript` | `scriptPath string`, `projectPath string`, 各 port | — | スクリプトを実行する |

#### 実装仕様（RenameBank を例示）

```go
// application/usecase/rename_bank.go
package usecase

import "github.com/yourname/dvc5/domain/model"

type RenameBankInput struct {
    Project *model.Project
    OldName string
    NewName string
}

type RenameBankOutput struct {
    Project *model.Project
}

// RenameBank は指定したバンクをリネームする。
// OldName に該当するバンクが存在しない場合は model.ErrNotFound を返す。
// NewName が空文字の場合は model.ErrEmptyName を返す。
func RenameBank(in RenameBankInput) (RenameBankOutput, error) {
    if in.NewName == "" {
        return RenameBankOutput{}, model.ErrEmptyName
    }
    for i := range in.Project.Scenes.Banks {
        if in.Project.Scenes.Banks[i].Name == in.OldName {
            in.Project.Scenes.Banks[i].Name = in.NewName
            return RenameBankOutput{Project: in.Project}, nil
        }
    }
    return RenameBankOutput{}, fmt.Errorf("bank %q: %w", in.OldName, model.ErrNotFound)
}
```

#### 実装仕様（RenameScene）

`Bank.Scenes []Scene` を走査するため、`Bank` が `Scenes` フィールドを持つことが前提となる。

```go
// application/usecase/rename_scene.go
package usecase

import "github.com/yourname/dvc5/domain/model"

type RenameSceneInput struct {
    Project  *model.Project
    BankName string
    OldName  string
    NewName  string
}

type RenameSceneOutput struct {
    Project *model.Project
}

// RenameScene は指定したバンク内のシーンをリネームする。
// BankName に該当するバンクが存在しない場合は model.ErrNotFound を返す。
// OldName に該当するシーンが存在しない場合は model.ErrNotFound を返す。
// NewName が空文字の場合は model.ErrEmptyName を返す。
func RenameScene(in RenameSceneInput) (RenameSceneOutput, error) {
    if in.NewName == "" {
        return RenameSceneOutput{}, model.ErrEmptyName
    }
    for i := range in.Project.Scenes.Banks {
        bank := &in.Project.Scenes.Banks[i]
        if bank.Name != in.BankName {
            continue
        }
        for j := range bank.Scenes {
            if bank.Scenes[j].Name == in.OldName {
                bank.Scenes[j].Name = in.NewName
                return RenameSceneOutput{Project: in.Project}, nil
            }
        }
        return RenameSceneOutput{}, fmt.Errorf("scene %q in bank %q: %w", in.OldName, in.BankName, model.ErrNotFound)
    }
    return RenameSceneOutput{}, fmt.Errorf("bank %q: %w", in.BankName, model.ErrNotFound)
}
```

#### RunScript の仕様

```go
// application/usecase/run_script.go
package usecase

import "github.com/yourname/dvc5/domain/port"

type RunScriptInput struct {
    ScriptPath  string
    ProjectPath string             // 空文字の場合はスクリプト内で load_project を明示する
    Reader      port.ProjectReader
    Writer      port.ProjectWriter
    Runner      port.ScriptRunner
}

// RunScript は Starlark スクリプトを実行する。
// ProjectPath が指定された場合は実行前に自動で読み込み、
// スクリプト終了後に自動で保存する。
// save_project がスクリプト内で明示的に呼ばれた場合は二重保存しない。
func RunScript(in RunScriptInput) error {
    var proj *model.Project
    if in.ProjectPath != "" {
        var err error
        proj, err = in.Reader.Read(in.ProjectPath)
        if err != nil {
            return fmt.Errorf("read project: %w", err)
        }
    }
    result, err := in.Runner.Run(in.ScriptPath, proj)
    if err != nil {
        return fmt.Errorf("run script: %w", err)
    }
    if result != nil && in.ProjectPath != "" {
        if err := in.Writer.Write(result, in.ProjectPath); err != nil {
            return fmt.Errorf("save project: %w", err)
        }
    }
    return nil
}
```

---

### 4.4 `infrastructure/dvc5xml`

#### 目的

`domain/port.ProjectReader` と `domain/port.ProjectWriter` を実装する。
XML の構造的な詳細（属性名・エンコード形式）をすべてこの層に閉じ込める。

#### ファイル責務

| ファイル | 責務 |
|---|---|
| `reader.go` | ファイル I/O とデコードの制御。`mapper.go` を呼ぶ |
| `writer.go` | ファイル I/O とエンコードの制御。`mapper.go` を呼ぶ |
| `mapper.go` | XML struct ↔ `domain/model` の変換ロジック |

#### XML struct 定義規約（mapper.go）

XML タグ付き struct は `xml` プレフィックスを付けた非公開型として定義する。
`domain/model` の型とは完全に分離する。

```go
// infrastructure/dvc5xml/mapper.go
package dvc5xml

import (
    "encoding/xml"
    "github.com/yourname/dvc5/domain/model"
)

type xmlDLMFILE struct {
    XMLName  xml.Name  `xml:"DLMFILE"`
    Type     string    `xml:"TYPE,attr"`
    Version  string    `xml:"VERSION,attr"`
    DASBuild string    `xml:"DASBUILD,attr"`
    Scenes   xmlScenes `xml:"SCENES"`
}

type xmlScenes struct {
    Banks []xmlBank `xml:"BANK"`
}

type xmlBank struct {
    DASUID  string     `xml:"DASUID,attr"`
    Name    string     `xml:"NAME,attr"`
    NBScene int        `xml:"NBSCENE,attr"`
    Scenes  []xmlScene `xml:"SCENE"`
}

type xmlScene struct {
    DASUID string `xml:"DASUID,attr"`
    Name   string `xml:"NAME,attr"`
}

// toModel は XML struct を domain/model に変換する。
func toModel(x *xmlDLMFILE, path string) *model.Project { ... }

// fromModel は domain/model を XML struct に変換する。
func fromModel(p *model.Project) *xmlDLMFILE { ... }
```

#### 規約

- Phase 1 では `<PATCHS>` 要素は読み飛ばし、書き出し時は元の DATA 属性値をそのまま保持する
- 属性の順序は読み込んだ順序を維持する（`encoding/xml` の仕様に従う）
- Phase 2 以降で PATCHS の操作が必要になった場合、`mapper.go` に codec を追加する

---

### 4.5 `infrastructure/starlark`

#### 目的

`domain/port.ScriptRunner` を実装する。
`go.starlark.net/starlark` を使い、Starlark スクリプトに対して
`domain/model` オブジェクトを操作できる実行環境を提供する。

#### ファイル責務

| ファイル | 責務 |
|---|---|
| `runner.go` | スクリプト実行の制御。Thread の生成と ExecFile の呼び出し |
| `builtins.go` | Starlark グローバル関数（`load_project`, `save_project`, `log`）の実装 |
| `binding/project.go` | `StarProject` 型（`starlark.Value` 実装） |
| `binding/scenes.go` | `StarScenes`, `StarBank`, `StarScene` 型 |
| `binding/util.go` | `starlark.Value` ↔ Go 基本型の変換ユーティリティ |

#### Runner 仕様

```go
// infrastructure/starlark/runner.go
package starlark

import (
    "go.starlark.net/starlark"
    "github.com/yourname/dvc5/domain/model"
    "github.com/yourname/dvc5/domain/port"
)

type Runner struct{}

var _ port.ScriptRunner = (*Runner)(nil)

func (r *Runner) Run(scriptPath string, proj *model.Project) (*model.Project, error) {
    thread := &starlark.Thread{Name: "dvc5"}
    globals := r.predeclared(proj)
    if _, err := starlark.ExecFile(thread, scriptPath, nil, globals); err != nil {
        return nil, err
    }
    // save_project が呼ばれた場合は globals の状態から変更後 Project を取得する
    return extractProject(globals), nil
}

func (r *Runner) predeclared(proj *model.Project) starlark.StringDict {
    return starlark.StringDict{
        "load_project": starlark.NewBuiltin("load_project", builtinLoadProject),
        "save_project": starlark.NewBuiltin("save_project", builtinSaveProject),
        "log":          starlark.NewBuiltin("log", builtinLog),
    }
}
```

#### Starlark 型ラッパーの実装規約（binding/）

`starlark.Value` を実装するすべての型は以下のメソッドを実装しなければならない。

| メソッド | 仕様 |
|---|---|
| `String() string` | `"TypeName"` の形式で返す |
| `Type() string` | Starlark 上の型名を返す（例: `"Project"`） |
| `Freeze()` | 何もしない（Starlark の freeze 機構は使用しない） |
| `Truth() starlark.Bool` | 常に `starlark.True` を返す |
| `Hash() (uint32, error)` | `fmt.Errorf("unhashable type: %s", p.Type())` を返す |
| `Attr(name string)` | フィールド名に応じて `starlark.Value` を返す |
| `AttrNames() []string` | 公開フィールド名の一覧を返す |

```go
// infrastructure/starlark/binding/project.go
package binding

import (
    "fmt"
    "go.starlark.net/starlark"
    "github.com/yourname/dvc5/domain/model"
)

// StarProject は Starlark スクリプトから見える Project オブジェクト。
type StarProject struct {
    inner *model.Project
}

func (p *StarProject) String() string        { return "Project" }
func (p *StarProject) Type() string          { return "Project" }
func (p *StarProject) Freeze()               {}
func (p *StarProject) Truth() starlark.Bool  { return starlark.True }
func (p *StarProject) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: Project") }

func (p *StarProject) Attr(name string) (starlark.Value, error) {
    switch name {
    case "path":
        return starlark.String(p.inner.Path), nil
    case "version":
        return starlark.String(p.inner.Version), nil
    case "scenes":
        return &StarScenes{inner: &p.inner.Scenes}, nil
    }
    return nil, nil // nil, nil = 属性が存在しない
}

func (p *StarProject) AttrNames() []string {
    return []string{"path", "version", "scenes"}
}

// Unwrap は内部の model.Project を返す。runner.go から利用する。
func (p *StarProject) Unwrap() *model.Project { return p.inner }
```

#### ビルトイン関数の仕様（builtins.go）

| 関数名 | シグネチャ（Starlark） | 説明 |
|---|---|---|
| `load_project` | `load_project(path: str) -> Project` | dvc5 ファイルを読み込む |
| `save_project` | `save_project(project: Project, path: str = None)` | プロジェクトを保存する |
| `log` | `log(msg: str)` | 標準出力にログを出力する |

---

### 4.6 `cmd/dvc5`

#### 目的

アダプターを組み立て（DI）、ユースケースを呼び出す CLI エントリポイント。
ビジネスロジックを一切持たない。

#### 実装仕様

```go
// cmd/dvc5/main.go
package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/yourname/dvc5/application/usecase"
    "github.com/yourname/dvc5/infrastructure/dvc5xml"
    starlarkinfra "github.com/yourname/dvc5/infrastructure/starlark"
)

func main() {
    if err := newRootCmd().Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func newRootCmd() *cobra.Command {
    root := &cobra.Command{
        Use:   "dvc5",
        Short: "Daslight5 dvc5 ファイル編集ツール",
    }
    root.AddCommand(newRunCmd())
    return root
}

func newRunCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "run <script.star> [project.dvc5]",
        Short: "Starlark スクリプトを実行する",
        Args:  cobra.RangeArgs(1, 2),
        RunE: func(cmd *cobra.Command, args []string) error {
            projectPath := ""
            if len(args) > 1 {
                projectPath = args[1]
            }
            return usecase.RunScript(usecase.RunScriptInput{
                ScriptPath:  args[0],
                ProjectPath: projectPath,
                Reader:      &dvc5xml.Reader{},
                Writer:      &dvc5xml.Writer{},
                Runner:      &starlarkinfra.Runner{},
            })
        },
    }
}
```

#### 規約

- `cmd/dvc5/main.go` には `main()` と `newXxxCmd()` のみを置く
- フラグのパースは Cobra に委譲し、手動の `os.Args` 解析は行わない
- エラーは `RunE` から `return err` で返し、Cobra に終了処理を委ねる

---

## 5. ライブラリとしての利用

`cmd/` を使わず、`application/usecase` と `infrastructure/` を直接 import することで
Go プログラムからライブラリとして利用できる。

```go
import (
    "github.com/yourname/dvc5/application/usecase"
    "github.com/yourname/dvc5/infrastructure/dvc5xml"
)

// 読み込み
loadOut, err := usecase.LoadProject(usecase.LoadProjectInput{
    Path:   "show.dvc5",
    Reader: &dvc5xml.Reader{},
})

// リネーム
renameOut, err := usecase.RenameBank(usecase.RenameBankInput{
    Project: loadOut.Project,
    OldName: "Bank 1",
    NewName: "本番",
})

// 保存
err = usecase.SaveProject(usecase.SaveProjectInput{
    Project: renameOut.Project,
    Path:    "show_edited.dvc5",
    Writer:  &dvc5xml.Writer{},
})
```

公開 API は `application/usecase` の関数群と各 `Input` / `Output` struct。
`infrastructure/*` は差し替え可能な実装として公開するが、
呼び出し元は port interface を通じて利用することを推奨する。

---

## 6. テスト方針

### テスト対象と優先度

| 対象 | テスト種別 | 優先度 |
|---|---|---|
| `application/usecase` | ユニットテスト | 必須 |
| `domain/model` | ユニットテスト | 必須 |
| `infrastructure/dvc5xml` | 統合テスト（testdata ファイル使用） | 推奨 |
| `infrastructure/starlark` | 統合テスト（.star ファイル使用） | 推奨 |
| `cmd/dvc5` | なし | — |

### ユニットテスト規約

- テストファイルは対象ファイルと同じパッケージに置く（`_test.go`）
- テーブル駆動テストを基本とする
- `infrastructure/*` の mock は `domain/port` の interface を実装した
  テスト用 struct として定義する（モックライブラリは使わない）

```go
// application/usecase/rename_bank_test.go
package usecase_test

import (
    "errors"
    "testing"
    "github.com/yourname/dvc5/domain/model"
)

func TestRenameBank(t *testing.T) {
    tests := []struct {
        name    string
        in      RenameBankInput
        want    string
        wantErr error
    }{
        {
            name: "正常系: リネーム成功",
            in: RenameBankInput{
                Project: &model.Project{Scenes: model.Scenes{
                    Banks: []model.Bank{{Name: "Draft", DASUID: "abc"}},
                }},
                OldName: "Draft",
                NewName: "本番",
            },
            want: "本番",
        },
        {
            name:    "異常系: 空文字",
            in:      RenameBankInput{NewName: ""},
            wantErr: model.ErrEmptyName,
        },
        {
            name: "異常系: 存在しないバンク",
            in: RenameBankInput{
                Project: &model.Project{},
                OldName: "存在しない",
                NewName: "新名前",
            },
            wantErr: model.ErrNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            out, err := RenameBank(tt.in)
            if tt.wantErr != nil {
                if !errors.Is(err, tt.wantErr) {
                    t.Errorf("want %v, got %v", tt.wantErr, err)
                }
                return
            }
            if err != nil {
                t.Fatal(err)
            }
            if got := out.Project.Scenes.Banks[0].Name; got != tt.want {
                t.Errorf("want %q, got %q", tt.want, got)
            }
        })
    }
}
```

### 統合テスト規約

`infrastructure/dvc5xml` のテストは `testdata/` ディレクトリに実際の `.dvc5` ファイルを
配置し、以下のフローで検証する。

```
Read(testdata/sample.dvc5) → toModel() → fromModel() → marshal → 文字列比較
```

---

## 7. 採用ライブラリ

| 用途 | ライブラリ | バージョン | 採用理由 |
|---|---|---|---|
| XML パース | `encoding/xml`（標準） | — | 外部依存なし。dvc5 の XML 構造は標準で対応可能 |
| Starlark | `go.starlark.net/starlark` | v0.0.0 最新 | 公式 Go 実装。メンテナンス継続中 |
| CLI | `github.com/spf13/cobra` | v1.x | Go CLI のデファクトスタンダード |
| テスト | `testing`（標準） | — | Phase 1 の規模ではサードパーティ不要 |

---

## 8. Phase 2 以降の拡張ポイント

Phase 2 以降の機能追加は、以下の場所のみを変更することで対応できる。
既存のテスト済みコードへの変更は最小限に抑えられる。

| 追加機能 | 変更箇所 | 備考 |
|---|---|---|
| バンク追加/削除 | `domain/model/scene.go` + `usecase/` に関数追加 | port 変更なし |
| シーンパラメータ変更 | `domain/model/scene.go` にフィールド追加 + `mapper.go` | port 変更なし |
| PATCHS デコード/操作 | `domain/model/patch.go` 新規 + `mapper.go` に codec 追加 | port に `PatchDecoder` を追加 |
| タイムライン操作 | `domain/model/timeline.go` 新規 + `binding/` に型追加 | port 変更なし |
| HTTP API 化 | `infrastructure/http/` を新規追加 | 他層の変更なし |
| Starlark ランタイム差し替え | `infrastructure/starlark/` を丸ごと差し替え | port interface が同じなら他層に影響なし |

---

## 9. 変更履歴

| バージョン | 日付 | 変更内容 |
|---|---|---|
| 1.0.2 | 2026-04 | 全ポートに `context.Context` を追加。`ProjectWriter` を `Write` / `WriteTo` に分割 |
| 1.0.1 | 2025-01 | `Bank` struct に `Scenes []Scene` のコメントを明記、`RenameScene` 実装仕様を追加 |
| 1.0.0 | 2025-01 | 初版作成（Phase 1 対象） |