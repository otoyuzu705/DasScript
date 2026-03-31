# dvc5 Starlark スクリプト API 仕様書

| 項目 | 内容 |
|---|---|
| 文書バージョン | 1.0.0 |
| 対象フェーズ | Phase 1（読み込み・保存・リネーム） |
| 対象ファイル形式 | Daslight5 `.dvc5` (XML) |
| スクリプト言語 | [Starlark](https://github.com/bazelbuild/starlark)（Python サブセット） |
| ホスト実装言語 | Go（`go.starlark.net`） |
| 実行方法 | `dvc5 run <script.star> [project.dvc5]` |
| 対象読者 | dvc5 スクリプトの作成者・ホスト実装者 |
| 最終更新 | 2025-01 |

---

## 目次

1. [設計方針](#1-設計方針)
2. [トップレベル関数](#2-トップレベル関数)
3. [型仕様](#3-型仕様)
   - 3.1 [Project](#31-project-型)
   - 3.2 [Scenes](#32-scenes-型)
   - 3.3 [Bank](#33-bank-型)
   - 3.4 [Scene](#34-scene-型)
4. [エラー仕様](#4-エラー仕様)
5. [スクリプト例](#5-スクリプト例)
6. [Phase 2 以降（実装予定）](#6-phase-2-以降実装予定)
7. [ホスト実装者向け仕様](#7-ホスト実装者向け仕様)
8. [変更履歴](#8-変更履歴)

---

## 1. 設計方針

| 方針 | 内容 |
|---|---|
| オブジェクト指向ラッパー | XML 要素をそのまま露出せず、照明プロジェクトとして意味のある型でラップする |
| 読み取り専用フィールド | Phase 1 ではすべてのフィールドを読み取り専用とし、変更は `rename()` メソッド経由のみとする |
| None 返しと fail() の使い分け | 「存在しない」は `None` を返す。「不正な操作」は `fail()` で即時停止する |
| PATCHS の遅延デコード | PATCHS（Base64+zlib）は Phase 1 では読み飛ばし、変更しない |
| 参照整合性 | DASUID の孤立参照検出はホスト側が担う。スクリプト側は UUID を直接操作しない |

---

## 2. トップレベル関数

スクリプト内でそのまま呼び出せるグローバル関数。

### `load_project(path: str) -> Project`

`.dvc5` ファイルを読み込み `Project` オブジェクトを返す。

**引数**

| 名前 | 型 | 必須 | 説明 |
|---|---|---|---|
| `path` | `str` | ✓ | 読み込む `.dvc5` ファイルのパス |

**戻り値:** `Project`

**エラー:** ファイルが存在しない・読み込めない場合は `fail()` で停止する。

```python
p = load_project("show.dvc5")
```

---

### `save_project(project: Project, path: str = None)`

`Project` オブジェクトを `.dvc5` ファイルとして保存する。

**引数**

| 名前 | 型 | 必須 | 説明 |
|---|---|---|---|
| `project` | `Project` | ✓ | 保存するプロジェクト |
| `path` | `str` | — | 保存先パス。省略時は `project.path`（読み込み元）に上書き保存する |

**戻り値:** `None`

**エラー:** 書き込みに失敗した場合は `fail()` で停止する。

```python
save_project(p, "show_edited.dvc5")  # 別名で保存
save_project(p)                       # 上書き保存
```

---

### `log(msg: str)`

ホスト側の標準出力にメッセージを出力する。

**引数**

| 名前 | 型 | 必須 | 説明 |
|---|---|---|---|
| `msg` | `str` | ✓ | 出力するメッセージ |

**戻り値:** `None`

```python
log("処理開始")
log("バンク数: " + str(len(p.scenes.banks())))
```

---

## 3. 型仕様

### 3.1 `Project` 型

`load_project()` の戻り値。dvc5 プロジェクトファイル全体を表す。

#### フィールド

| フィールド | 型 | アクセス | 説明 |
|---|---|---|---|
| `.path` | `str` | 読み取り専用 | 読み込み元ファイルパス |
| `.version` | `str` | 読み取り専用 | `DASBUILD` 属性値（例: `"24.1205.158.90"`） |
| `.scenes` | `Scenes` | 読み取り専用 | SCENES セクションのラッパー |

#### 使用例

```python
p = load_project("show.dvc5")
log(p.version)  # => "24.1205.158.90"
log(p.path)     # => "show.dvc5"
```

---

### 3.2 `Scenes` 型

`Project.scenes` から取得する。SCENES セクション全体を表す。

#### メソッド

| メソッド | 戻り値 | 説明 |
|---|---|---|
| `.banks()` | `list[Bank]` | 全バンクを index 順で返す |
| `.bank(name: str)` | `Bank \| None` | バンク名で検索する。存在しない場合は `None` |
| `.bank_at(idx: int)` | `Bank \| None` | 0始まりインデックスで取得する。範囲外は `None` |

#### 使用例

```python
banks = p.scenes.banks()          # 全バンク
b = p.scenes.bank("本番")         # 名前検索
b = p.scenes.bank_at(0)           # 先頭バンク
```

---

### 3.3 `Bank` 型

`Scenes` のメソッドから取得する。`<BANK>` 要素を表す。
内部にシーンのリストを保持しており、`bank.scenes()` 等のメソッドで操作する。

#### フィールド

| フィールド | 型 | アクセス | 説明 |
|---|---|---|---|
| `.dasuid` | `str` | 読み取り専用 | UUID（`DASUID` 属性値） |
| `.name` | `str` | 読み取り専用 | バンク名 |
| `.index` | `int` | 読み取り専用 | SCENES 内の 0始まり位置 |

#### メソッド

##### `bank.rename(new_name: str)`

バンク名を変更する。

| 引数 | 型 | 必須 | 説明 |
|---|---|---|---|
| `new_name` | `str` | ✓ | 新しいバンク名 |

**戻り値:** `None`

**エラー:** `new_name` が空文字の場合は `fail()` で停止する。

```python
b = p.scenes.bank("Draft")
b.rename("本番")
```

---

##### `bank.scenes() -> list[Scene]`

バンク内の全シーンを `index` 順で返す。シーンが存在しない場合は空リストを返す。

```python
for scene in bank.scenes():
    log(scene.name)
```

---

##### `bank.scene(name: str) -> Scene | None`

シーン名で検索する。存在しない場合は `None` を返す。

| 引数 | 型 | 必須 | 説明 |
|---|---|---|---|
| `name` | `str` | ✓ | 検索するシーン名 |

```python
s = bank.scene("イントロ")
if s == None:
    fail("シーンが見つかりません")
```

---

##### `bank.scene_at(idx: int) -> Scene | None`

0始まりインデックスでシーンを取得する。範囲外は `None` を返す。

| 引数 | 型 | 必須 | 説明 |
|---|---|---|---|
| `idx` | `int` | ✓ | 0始まりのインデックス |

---

### 3.4 `Scene` 型

`Bank` のメソッドから取得する。`<SCENE>` 要素を表す。

#### フィールド

| フィールド | 型 | アクセス | 説明 |
|---|---|---|---|
| `.dasuid` | `str` | 読み取り専用 | UUID（`DASUID` 属性値） |
| `.name` | `str` | 読み取り専用 | シーン名 |
| `.index` | `int` | 読み取り専用 | Bank 内の 0始まり位置 |

#### メソッド

##### `scene.rename(new_name: str)`

シーン名を変更する。

| 引数 | 型 | 必須 | 説明 |
|---|---|---|---|
| `new_name` | `str` | ✓ | 新しいシーン名 |

**エラー:** `new_name` が空文字の場合は `fail()` で停止する。

```python
s = bank.scene("New Scene")
s.rename("イントロ")
```

---

## 4. エラー仕様

### None を返す操作

以下の操作は「対象が存在しない」場合に `None` を返す。`fail()` では停止しない。
スクリプト側で `None` チェックを行い、必要に応じて `fail()` を呼ぶこと。

| 操作 | `None` を返す条件 |
|---|---|
| `Scenes.bank(name)` | 指定した名前のバンクが存在しない |
| `Scenes.bank_at(idx)` | インデックスが範囲外 |
| `Bank.scene(name)` | 指定した名前のシーンが存在しない |
| `Bank.scene_at(idx)` | インデックスが範囲外 |

### fail() で停止する操作

以下の操作は不正な引数を受け取った場合にスクリプトを即時停止する。

| 操作 | 停止する条件 |
|---|---|
| `load_project(path)` | ファイルが存在しない、または読み込みに失敗した |
| `save_project(project, path)` | 書き込みに失敗した |
| `Bank.rename(new_name)` | `new_name` が空文字 |
| `Scene.rename(new_name)` | `new_name` が空文字 |

### None チェックのパターン

```python
bank = p.scenes.bank("本番")
if bank == None:
    fail("バンク '本番' が見つかりません")
```

---

## 5. スクリプト例

### 例1: バンク・シーンの一覧をログ出力

```python
p = load_project("show.dvc5")

for bank in p.scenes.banks():
    log("[" + str(bank.index) + "] " + bank.name + " (" + bank.dasuid + ")")
    for scene in bank.scenes():
        log("    [" + str(scene.index) + "] " + scene.name)
```

### 例2: バンク名の一括リネーム

```python
p = load_project("show.dvc5")

renames = {
    "Bank 1": "オープニング",
    "Bank 2": "本編",
    "Bank 3": "エンディング",
}

for bank in p.scenes.banks():
    new_name = renames.get(bank.name)
    if new_name != None:
        log(bank.name + " -> " + new_name)
        bank.rename(new_name)

save_project(p)
```

### 例3: 特定バンク内のシーンをリネーム

```python
p = load_project("show.dvc5")

bank = p.scenes.bank("本編")
if bank == None:
    fail("バンク '本編' が見つかりません")

scene = bank.scene("New Scene")
if scene == None:
    fail("シーン 'New Scene' が見つかりません")

scene.rename("サビ")
save_project(p)
```

### 例4: 全バンクの全シーンを "New Scene" → 連番でリネーム

```python
p = load_project("show.dvc5")

for bank in p.scenes.banks():
    counter = 1
    for scene in bank.scenes():
        if scene.name == "New Scene":
            scene.rename("Scene " + str(counter))
            counter = counter + 1

save_project(p)
```

---

## 6. Phase 2 以降（実装予定）

以下は Phase 1 には含まれない。将来の実装方針として記録する。

### バンク・シーンの追加/削除

`Scenes` に `add_bank()` / `remove_bank()`、`Bank` に `add_scene()` / `remove_scene()` を追加予定。
削除時は `SHORTCUT` などクロスリファレンスの孤立参照をホスト側が検出し、警告を出す。

### シーンパラメータの変更

`Scene` に `speed` / `fade_in` / `fade_out` / `loop_mode` / `dimmer` などのフィールドを追加予定。
フィールドへの代入で直接変更できるようにする。

### PATCHS（フィクスチャ定義）の操作

`Project.patchs` を通じてフィクスチャの DMX アドレス・ユニバース変更などを行う予定。
PATCHS は Base64+zlib エンコードされているため、明示的な `decode()` / `encode()` 呼び出しが必要になる。

### FIXTUREDATA（静的チャンネル値）の読み書き

シーン内の各フィクスチャのチャンネル値を `get_channel_value()` / `set_channel_value()` で操作予定。
Dimmer を上げる・RGB を指定するといった操作が対象。

### タイムライン操作

`Scene` に紐づく `Rack`（TYPE=1）に対し、`Block` の追加・削除・時間変更・一括シフトを行う予定。
BPM グリッドベースの ms 変換ヘルパー（`Rack.beat_to_ms()` / `beats_range()`）も合わせて実装する。

### ステップシーケンス操作

`Rack`（TYPE=9）に紐づく `Steps` / `Step` の追加・削除・並び替え・チャンネル値の読み書きを行う予定。

### フィクスチャグループの操作

`Project.fixture_groups` を通じて、グループへのフィクスチャ追加・削除・グループ自体の作成・削除を行う予定。

---

## 7. ホスト実装者向け仕様

スクリプト作成者ではなく、ホスト（Go）側を実装する開発者向けの仕様。

### Starlark 型ラッパーの公開フィールド対応表

各型の `Attr()` メソッドが返すべきフィールドの一覧。

| Starlark 型 | フィールド名 | 返す値 |
|---|---|---|
| `Project` | `path` | `starlark.String(project.Path)` |
| `Project` | `version` | `starlark.String(project.Version)` |
| `Project` | `scenes` | `*StarScenes` |
| `Bank` | `dasuid` | `starlark.String(bank.DASUID)` |
| `Bank` | `name` | `starlark.String(bank.Name)` |
| `Bank` | `index` | `starlark.MakeInt(bank.Index)` |
| `Scene` | `dasuid` | `starlark.String(scene.DASUID)` |
| `Scene` | `name` | `starlark.String(scene.Name)` |
| `Scene` | `index` | `starlark.MakeInt(scene.Index)` |

### Starlark 型ラッパーのメソッド対応表

`Attr()` で返す呼び出し可能オブジェクト（`*starlark.Builtin`）の一覧。
メソッドは `starlark.NewBuiltin` でラップし、`Attr()` から返す。

| Starlark 型 | メソッド名 | シグネチャ | Go 実装関数 |
|---|---|---|---|
| `Scenes` | `banks` | `() -> list[StarBank]` | `builtinScenesGetBanks` |
| `Scenes` | `bank` | `(name: str) -> StarBank\|None` | `builtinScenesGetBank` |
| `Scenes` | `bank_at` | `(idx: int) -> StarBank\|None` | `builtinScenesGetBankAt` |
| `Bank` | `rename` | `(new_name: str)` | `builtinBankRename` |
| `Bank` | `scenes` | `() -> list[StarScene]` | `builtinBankGetScenes` |
| `Bank` | `scene` | `(name: str) -> StarScene\|None` | `builtinBankGetScene` |
| `Bank` | `scene_at` | `(idx: int) -> StarScene\|None` | `builtinBankGetSceneAt` |
| `Scene` | `rename` | `(new_name: str)` | `builtinSceneRename` |

`Bank.scenes()` は `inner.Scenes []model.Scene` をそのまま `starlark.List` に変換して返す。
空リストの場合は `starlark.NewList(nil)` を返す（`None` は返さない）。

### ビルトイン関数の登録

Phase 1 で登録するビルトイン関数の一覧。

| 関数名 | Go 実装関数 | 備考 |
|---|---|---|
| `load_project` | `builtinLoadProject` | `dvc5xml.Reader` を内部で使用 |
| `save_project` | `builtinSaveProject` | `dvc5xml.Writer` を内部で使用 |
| `log` | `builtinLog` | `fmt.Fprintln(os.Stdout, ...)` |

将来 `new_uuid()` を追加する想定で、実装上のプレースホルダとして `builtins.go` にコメントを残しておくこと。

### PATCHS の扱い（Phase 1）

Phase 1 では PATCHS（`<PATCHS DATA="..."/>` 要素）を変更しない。
`dvc5xml.Reader` は PATCHS の `DATA` 属性値を `string` のまま保持し、
`dvc5xml.Writer` は受け取った値をそのまま書き出す。デコード・エンコードは行わない。

### 属性順序の保持

`encoding/xml` は属性を struct フィールドの定義順で出力する。
読み込んだ属性の順序を維持するため、`xmlDLMFILE` 等の struct フィールド順序は
実際の dvc5 ファイルの属性出現順に合わせること。

---

## 8. 変更履歴

| バージョン | 日付 | 変更内容 |
|---|---|---|
| 1.0.1 | 2025-01 | `Bank` がシーンリストを保持することを明記、`bank.scenes()` 等の引数・戻り値を補完、ホスト実装者向けメソッド対応表を追加 |
| 1.0.0 | 2025-01 | 初版作成（Phase 1 対象） |