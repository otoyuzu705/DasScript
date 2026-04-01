# Daslight5 DVC ファイルフォーマット仕様書

> 本ドキュメントは `all.dvc` (Daslight5 ビルド 24.1205.158.90) の実データを逆解析して作成したものです。
> 公式仕様ではなく、観測ベースの記述です。

---

## 目次

1. [ファイル概要](#1-ファイル概要)
2. [エンコードデータの共通形式](#2-エンコードデータの共通形式)
3. [DLMFILE (ルート要素)](#3-dlmfile-ルート要素)
4. [CONFIGURATION](#4-configuration)
5. [PATCHS (パッチデータ)](#5-patchs-パッチデータ)
6. [FIXTUREGROUPS](#6-fixturegroups)
7. [SCENES](#7-scenes)
8. [SHORTCUTS](#8-shortcuts)
9. [TOUCH](#9-touch)
10. [DEVICES](#10-devices)
- [付録A: 要素間のクロスリファレンス（参照関係）](#付録a-要素間のクロスリファレンス参照関係)
- [付録B: 共通データ型](#付録b-共通データ型)

---

## 1. ファイル概要

`.dvc` ファイルは Daslight5 照明制御ソフトウェアのプロジェクトファイルであり、**XML形式**で記述されている。

一部の大きなデータ（パッチ情報、フィクスチャデータ等）は **Base64 + zlib圧縮** でインライン埋め込みされている。

### 全体のツリー構造

```
DLMFILE
├── CONFIGURATION          ... ビュー設定、UIレイアウト
├── PATCHS                 ... フィクスチャ定義（エンコード済み）
├── FIXTUREGROUPS          ... フィクスチャのグループ定義
│   └── FIXTUREGROUP*
├── SCENES                 ... シーン・バンク・エフェクト定義
│   ├── BANK*
│   │   └── SCENE*
│   └── DAS_SELECTIONS
├── SHORTCUTS              ... ショートカット／MIDIマッピング
│   └── SHORTCUT*
├── TOUCH                  ... タッチUIページ定義
│   └── TOUCHPAGE*
└── DEVICES                ... 外部デバイス設定
```

---

## 2. エンコードデータの共通形式

DVC ファイル内では複数のエンコード形式が使用されている。

### 2.1 PATCHS DATA 形式 (Base64 + 4バイトヘッダー + zlib)

| オフセット | サイズ | 内容 |
|-----------|--------|------|
| 0 | 4 bytes | ヘッダー（展開後データサイズ、uint32 Big-Endian） |
| 4 | 残り全て | zlib 圧縮データ |

**デコード手順:**

```
Base64文字列 → Base64デコード → 先頭4バイトをスキップ → zlib解凍 → UTF-8 XML文字列
```

**Python デコード例:**

```python
import base64
import zlib

def decode_daslight5_patch(data):
    decoded_b64 = base64.b64decode(data)
    header = decoded_b64[:4]           # 4バイトヘッダー（展開後サイズ）
    compressed_data = decoded_b64[4:]
    decompressed = zlib.decompress(compressed_data)
    xml_str = decompressed.decode('utf-8')
    return xml_str
```

> 参照: https://gist.github.com/otoyuzu705/fb6e296eb3007a0964ddd0ccec5b412c

### 2.2 FIXTUREDATA DATA 形式 (Base64 + zlib直接)

SCENE 内の `FIXTUREDATA` 要素の `DATA` 属性は、4バイトヘッダー**なし**で直接 zlib 圧縮データが Base64 エンコードされている。

```
Base64文字列 → Base64デコード → zlib解凍 → バイナリデータ（チャンネル値等）
```

### 2.3 Hex + zlib 形式

`CONFIGURATION` の `TOUCH_DOCK_MANAGER` や、PATCHS 内部の `OUTMODE` / `FLAG` 属性で使用。

```
16進数文字列 → バイト列に変換 → 先頭4バイトヘッダー → 残りをzlib解凍
```

`TOUCH_DOCK_MANAGER` の展開結果は Qt の `QtAdvancedDockingSystem` レイアウト XML となる。

---

## 3. DLMFILE (ルート要素)

```xml
<DLMFILE TYPE="Daslight" VERSION="5" DASBUILD="24.1205.158.90" VERSIONFILE="2">
```

| 属性 | 説明 | 例 |
|------|------|-----|
| `TYPE` | アプリケーション種別 | `"Daslight"` |
| `VERSION` | メジャーバージョン | `"5"` |
| `DASBUILD` | ビルドバージョン | `"24.1205.158.90"` |
| `VERSIONFILE` | ファイルフォーマットバージョン | `"2"` |

---

## 4. CONFIGURATION

ビューポート設定とUI状態を保持する。

```xml
<CONFIGURATION VIEWZOOM="1" VIEWPOSX="-75" VIEWPOSY="2"
    TOUCH_DOCK_MANAGER="(hex+zlib)" TOUCH_ZOOMS="1,1"/>
```

| 属性 | 説明 |
|------|------|
| `VIEWZOOM` | 2Dビューのズームレベル |
| `VIEWPOSX` | 2Dビューの X スクロール位置 |
| `VIEWPOSY` | 2Dビューの Y スクロール位置 |
| `TOUCH_DOCK_MANAGER` | UIドッキングレイアウト（Hex+4バイトヘッダー+zlib → Qt XML） |
| `TOUCH_ZOOMS` | タッチページのズーム値（カンマ区切り） |

---

## 5. PATCHS (パッチデータ)

フィクスチャのライブラリ定義とパッチ配置情報を含む、**エンコードされた大規模XMLデータ**。

```xml
<PATCHS DATA="(Base64エンコード文字列)"/>
```

### 5.1 デコード後の内部XML構造

```
PATCH  (NBFIXTURE="5")
└── FIXTURES*            ... ライブラリ+フィクスチャのペア（繰り返し）
    ├── SSLLIBRARY       ... フィクスチャライブラリ定義
    │   ├── SSLPROPERTIES    ... 物理的プロパティ
    │   └── SSLMODES         ... チャンネルモード定義
    │       └── SSLMODE*
    │           └── SSLCHANNEL*
    │               └── SSLPRESETS
    │                   └── SSLPRESET*
    └── FIXTURE*         ... パッチされたフィクスチャインスタンス
        └── BEAM*        ... ビーム位置
```

### 5.2 PATCH

```xml
<PATCH NBFIXTURE="5">
```

| 属性 | 説明 |
|------|------|
| `NBFIXTURE` | パッチされたフィクスチャの総数 |

### 5.3 SSLLIBRARY (フィクスチャライブラリ)

```xml
<SSLLIBRARY SSLFIXUID="495fd2a0-..." SSLNAME="_imported/ETC_LinearLight.ssl2">
```

| 属性 | 説明 |
|------|------|
| `SSLFIXUID` | ライブラリの一意識別子 |
| `SSLNAME` | SSL2 ライブラリファイルのパス |

### 5.4 SSLPROPERTIES (フィクスチャ物理プロパティ)

| 属性 | 説明 | 例 |
|------|------|-----|
| `SSLFIXTFAMILY` | フィクスチャファミリー種別 | `"0"` |
| `SSLFIXTTYPE` | フィクスチャタイプ | `"3"` |
| `SSLSIZEX/Y/Z` | 物理サイズ (cm) | `"50"`, `"15"`, `"11"` |
| `SSLWEIGHT` | 重量 (g) | `"5000"` |
| `SSLBEAMOPENING` | ビーム角度 (最小) | `"16"` |
| `SSLBEAMOPENING2` | ビーム角度 (最大) | `"60"` |
| `SSLAMPLIPAN` | パン可動範囲 (度) | `"360"` |
| `SSLAMPLITILT` | チルト可動範囲 (度) | `"220"` |
| `SSLTIMEPAN` | パン移動時間 (ms) | `"1000"` |
| `SSLTIMETILT` | チルト移動時間 (ms) | `"1000"` |
| `SSLPANCENTER` | パン中央 DMX 値 | `"127"` |
| `SSLTILTCENTER` | チルト中央 DMX 値 | `"127"` |
| `SSLLAMPPOWER` | ランプ出力種別 | `"3"` |
| `SSLLAMPTEMP` | 色温度 (K) | `"7000"` |
| `SSLLAMPTYPE` | ランプ種別 | `"led"` |
| `SSLBEAMTYPE` | ビームタイプ | `"0"` |
| `SSL3DOBJECTFILE` | 3Dモデルファイル | `"Heads/Head.x1"` |
| `SSLCOLOR` | デフォルトカラー | `"255"` |
| `SSLINVPAN` | パン反転フラグ | `"0"` / `"1"` |
| `SSLINVTILT` | チルト反転フラグ | `"0"` / `"1"` |
| `SSLCREATOR` | ライブラリ作成者 | メールアドレス等 |

### 5.5 SSLMODES / SSLMODE (チャンネルモード)

```xml
<SSLMODES SSLNBMODE="1">
  <SSLMODE SSLMODEINDEX="0" SSLNBCHANNEL="5">
```

| 属性 | 説明 |
|------|------|
| `SSLNBMODE` | モード数 |
| `SSLMODEINDEX` | モードのインデックス番号 |
| `SSLNBCHANNEL` | チャンネル数 |

### 5.6 SSLCHANNEL (チャンネル定義)

```xml
<SSLCHANNEL SSLCHANNELTYPE="7" SSLCHANNELNAME="Dimmer"
    SSLCHANNELTYPEINDEX="0" SSLCHANNELMSB="0" SSLCHANNELLSB="0" ...>
```

| 属性 | 説明 |
|------|------|
| `SSLCHANNELTYPE` | チャンネル種別コード（下表参照） |
| `SSLCHANNELNAME` | チャンネル名 |
| `SSLCHANNELTYPEINDEX` | 同一タイプ内のインデックス |
| `SSLCHANNELMSB` | 16bit MSB チャンネルフラグ |
| `SSLCHANNELLSB` | 16bit LSB チャンネルフラグ |
| `SSLCHANNEL2PRESETS` | プリセット表示モード |

**チャンネルタイプコード (観測値):**

| コード | チャンネル種別 |
|--------|---------------|
| `0` | その他 / 汎用 |
| `1` | Pan (X軸) |
| `2` | Tilt (Y軸) |
| `5` | カラーホイール |
| `7` | ディマー |
| `8` | ゴボホイール |
| `15` | シャッター / ストロボ |
| `18` | Pan/Tilt スピード |
| `24` | スモーク |
| `25` | Red (RGB) |
| `26` | Green (RGB) |
| `27` | Blue (RGB) |

### 5.7 SSLPRESET (プリセット定義)

```xml
<SSLPRESET SSLPRESETTYPE="4" SSLPRESETNAME="Dimmer"
    SSLPRESETDMXSTART="0" SSLPRESETDMXEND="255" SSLPRESETDMXDEFAULT="255"
    SSLPRESETSHOWDIMMER="1" SSLPRESETDEFAULTPRESET="1" .../>
```

| 属性 | 説明 |
|------|------|
| `SSLPRESETTYPE` | プリセット種別コード |
| `SSLPRESETNAME` | プリセット名 |
| `SSLPRESETICON` | アイコンパス |
| `SSLPRESETDMXSTART` | DMX 開始値 (0-255) |
| `SSLPRESETDMXEND` | DMX 終了値 (0-255) |
| `SSLPRESETDMXDEFAULT` | DMX デフォルト値 |
| `SSLPRESETSHOWDIMMER` | ディマーバー表示フラグ |
| `SSLPRESETDEFAULTPRESET` | デフォルトプリセットフラグ |
| `SSLPRESETCOLOR` | カラー値 (10進整数) |
| `SSLPRESETPARAMMIN` | パラメータ最小値 |
| `SSLPRESETPARAMMAX` | パラメータ最大値 |

### 5.8 FIXTURE (パッチ済みフィクスチャインスタンス)

```xml
<FIXTURE TYPE="0" DASUID="9fad3b38-..." NAME="etc_linearlight"
    INDEX="2" SHAPE="1" SIZE="30" POSX="430" POSY="160"
    ANGLE="0" ADDRESS="27" UNIVERS="1" OUTMODE="(hex)" FLAG="(hex)">
  <BEAM INDEX="0" POSX="430" POSY="160"/>
</FIXTURE>
```

| 属性 | 説明 |
|------|------|
| `TYPE` | フィクスチャタイプ |
| `DASUID` | 一意識別子 (UUID) |
| `NAME` | フィクスチャ名 |
| `INDEX` | パッチインデックス (1始まり) |
| `SHAPE` | 2Dビュー形状 |
| `SIZE` | 2Dビュー表示サイズ |
| `POSX` / `POSY` | 2Dビュー座標 |
| `ANGLE` | 回転角度 |
| `ADDRESS` | DMX 開始アドレス (1始まり) |
| `UNIVERS` | DMX ユニバース番号 (1始まり) |
| `OUTMODE` | 出力モード設定 (hex+zlib) |
| `FLAG` | フラグデータ (hex+zlib) |
| `IPAN` | パン反転 (オプション) |
| `MINTILT` / `MAXTILT` | チルト制限範囲 (オプション) |
| `MINDIMMER` / `MAXDIMMER` | ディマー制限範囲 (オプション) |

**子要素:**

```xml
<BEAM INDEX="0" POSX="430" POSY="160"/>
```

| 属性 | 説明 |
|------|------|
| `INDEX` | ビーム番号 |
| `POSX` / `POSY` | ビーム位置座標 |

---

## 6. FIXTUREGROUPS

フィクスチャのグルーピングとビーム選択を管理する。

```xml
<FIXTUREGROUPS>
  <FIXTUREGROUP DASUID="..." NAME="All"
      TOGGLE_BLACKOUT="0" TOGGLE_STROBE="0" TOGGLE_FULL="0"
      TOGGLE_EXCLUSIVE_FULL="0" TOGGLE_FLASH="0" TOGGLE_EXCLUSIVE_FLASH="0">
    <FIXTURES>
      <FIXTURE DASUID="(fixture-uuid)"/>
    </FIXTURES>
    <BEAMSELECTION>
      <BEAMS DASUID="..." NAME="1aa" NB="3">
        <BEAM FIXTURE="(fixture-uuid)" BEAMID="0" IDSELECTION="1"/>
      </BEAMS>
    </BEAMSELECTION>
  </FIXTUREGROUP>
</FIXTUREGROUPS>
```

### 6.1 FIXTUREGROUP

| 属性 | 説明 |
|------|------|
| `DASUID` | グループの一意識別子 |
| `NAME` | グループ名 |
| `TOGGLE_BLACKOUT` | ブラックアウトトグル状態 |
| `TOGGLE_STROBE` | ストロボトグル状態 |
| `TOGGLE_FULL` | フル出力トグル状態 |
| `TOGGLE_EXCLUSIVE_FULL` | 排他フル出力トグル状態 |
| `TOGGLE_FLASH` | フラッシュトグル状態 |
| `TOGGLE_EXCLUSIVE_FLASH` | 排他フラッシュトグル状態 |

### 6.2 BEAMSELECTION / BEAMS

| 属性 | 説明 |
|------|------|
| `DASUID` | ビーム選択セットの UUID |
| `NAME` | ビーム選択名 |
| `NB` | ビーム数 |

### 6.3 BEAM (グループ内)

| 属性 | 説明 |
|------|------|
| `FIXTURE` | 対象フィクスチャの DASUID |
| `BEAMID` | ビームID |
| `IDSELECTION` | 選択順序番号 |

---

## 7. SCENES

シーン、バンク、エフェクトのメイン制御データを格納する。

```
SCENES
├── BANK*
│   └── SCENE*
│       ├── FIXTUREDATAS
│       │   └── FIXTUREDATA*    ... 静的チャンネル値
│       └── RACKS
│           └── RACK*           ... エフェクト・タイムライン等
└── DAS_SELECTIONS              ... 現在の選択状態
```

### 7.1 BANK

```xml
<BANK DASUID="..." NAME="Bank 1" COLOR="#ffffcf08"
    COLLAPSED="0" PAUSED="0" HIDDEN="0" NBSCENE="2">
```

| 属性 | 説明 |
|------|------|
| `DASUID` | バンクの UUID |
| `NAME` | バンク名 |
| `COLOR` | 表示カラー (`#AARRGGBB` 形式) |
| `COLLAPSED` | 折りたたみ状態 |
| `PAUSED` | 一時停止状態 |
| `HIDDEN` | 非表示フラグ |
| `NBSCENE` | バンク内のシーン数 |

### 7.2 SCENE

```xml
<SCENE DASUID="..." NAME="New Scene" COLOR="#ffffcf08"
    ENABLE="1" VISIBLE="1" LOOP_MODE="0" LOOP="0"
    PLAY_DIRECTION="1" PLAY_MODE="0" FLASH="0" BOUNCE_MODE="0"
    PLAY_TRIGGER="0" PLAY_DIVISION="8" DIMMER="1" D_FEATURE="1"
    SPEED="0.5" PHASE="0" SIZE="1" AUTO_RELEASE_END="1"
    JUMP_MODE="-1" RELEASE_MODE="2" RELEASE_PROTECT_MODE="0"
    FADE_IN="0" FADE_OUT="0" ATTRIBUTEVALUE_MODE="0"
    LTPPRIORITY="2" PAUSE="0" SAIMAGEPATH="">
```

| 属性 | 説明 | 値の例 |
|------|------|--------|
| `DASUID` | シーンの UUID | |
| `NAME` | シーン名 | `"New Scene"` |
| `COLOR` | 表示カラー | `"#ffffcf08"` |
| `ENABLE` | 有効/無効 | `0` / `1` |
| `VISIBLE` | 表示/非表示 | `0` / `1` |
| `LOOP_MODE` | ループモード | `0` |
| `LOOP` | ループ回数 | `0` = 無限 |
| `PLAY_DIRECTION` | 再生方向 | `1` = 順方向 |
| `PLAY_MODE` | 再生モード | `0` |
| `FLASH` | フラッシュモード | `0` / `1` |
| `BOUNCE_MODE` | バウンスモード | `0` |
| `PLAY_TRIGGER` | トリガーモード | `0` |
| `PLAY_DIVISION` | BPM分割数 | `8` |
| `DIMMER` | ディマー値 | `0.0` ～ `1.0` |
| `D_FEATURE` | ディマー機能有効 | `0` / `1` |
| `SPEED` | 再生速度 | `0.5` |
| `PHASE` | フェーズオフセット | `0` |
| `SIZE` | サイズ | `1` |
| `AUTO_RELEASE_END` | 終了時自動リリース | `0` / `1` |
| `JUMP_MODE` | ジャンプモード | `-1` = なし |
| `RELEASE_MODE` | リリースモード | `2` |
| `RELEASE_PROTECT_MODE` | リリース保護モード | `0` |
| `FADE_IN` | フェードイン時間 | `0` |
| `FADE_OUT` | フェードアウト時間 | `0` |
| `ATTRIBUTEVALUE_MODE` | アトリビュート値モード | `0` |
| `LTPPRIORITY` | LTP 優先度 | `2` |
| `PAUSE` | 一時停止状態 | `0` |
| `SAIMAGEPATH` | シーンイメージパス | `""` |

### 7.3 FIXTUREDATAS / FIXTUREDATA

シーンの静的チャンネル値を保持する。

```xml
<FIXTUREDATAS NB="3">
  <FIXTUREDATA FIXTURE="(fixture-uuid)" DATA="(Base64+zlib)"/>
</FIXTUREDATAS>
```

| 属性 | 説明 |
|------|------|
| `NB` | フィクスチャデータ数 |
| `FIXTURE` | 対象フィクスチャの DASUID |
| `DATA` | チャンネル値データ（Base64 + zlib、ヘッダーなし） |

### 7.4 RACK (エフェクト/タイムライン コンテナ)

`RACK` は TYPE によって子要素の構造が異なる。

```xml
<RACK TYPE="(type-id)" EXPAND_RACK="1" DURATION="160" ...>
```

| 属性 | 説明 |
|------|------|
| `TYPE` | ラックの種別（下表参照） |
| `EXPAND_RACK` | UI 展開状態 |
| `DURATION` | 継続時間 (ms) ※タイムラインのみ |
| `DISPLAY_MODE` | 表示モード ※タイムラインのみ |
| `GRID_BPM` | BPM グリッド値 ※タイムラインのみ |
| `GRID_OFFSET` | グリッドオフセット (ms) ※タイムラインのみ |

**RACK TYPE 一覧 (観測値):**

| TYPE | 種別 | 子要素 |
|------|------|--------|
| `1` | タイムライン | `TIMELINES` |
| `2` | カラーエフェクト | `EFFECT` + `BEAMS` |
| `3` | プリセットエフェクト | `EFFECT` + `PRESETS` + `BEAMS` |
| `4` | ポジションエフェクト | `EFFECT` + `BEAMS` |
| `5` | 2Dカラーマッピング | `EFFECT` + `MAPPING` + `BEAMS` |
| `6` | 2Dプリセットマッピング | `EFFECT` + `MAPPING` + `PRESETS` + `BEAMS` |
| `7` | カラー+プリセット | `EFFECT` + `PRESETS` + `BEAMS` |
| `8` | プリセット+プリセット | `EFFECT` + `PRESETS` + `BEAMS` |
| `9` | ステップシーケンス | `STEPS` |

### 7.5 EFFECT

```xml
<EFFECT TYPE="2" ID="130" DURATION="5000">
  <PARAMS NB="6">
    <PARAM TYPE="4" ID="1">
      <COLORS NB="8">
        <COLOR VAL="1/0/0/0/0/0/0/0/1/1/1/1/0/0/0/0/0/1"/>
      </COLORS>
    </PARAM>
    <PARAM TYPE="5" ID="1">
      <POINTS NB="4">
        <POINT X="0.25" Y="0.5"/>
      </POINTS>
    </PARAM>
    <PARAM TYPE="2" ID="2" VAL="0"/>
  </PARAMS>
</EFFECT>
```

| 属性 | 説明 |
|------|------|
| `TYPE` | エフェクトタイプ |
| `ID` | エフェクト ID |
| `DURATION` | エフェクト時間 (ms) |

**PARAM TYPE 一覧 (観測値):**

| TYPE | 内容 | 値の格納方法 |
|------|------|-------------|
| `0` | 整数パラメータ | `VAL` 属性 |
| `1` | 浮動小数パラメータ | `VAL` 属性 |
| `2` | 浮動小数パラメータ | `VAL` 属性 |
| `4` | カラーリスト | 子要素 `COLORS > COLOR` |
| `5` | ポイントリスト（パス） | 子要素 `POINTS > POINT` |
| `6` | 浮動小数パラメータ | `VAL` 属性 |

**COLOR VAL 形式:**

スラッシュ区切りの 18 個の浮動小数値:
```
R/G/B/0/0/0/0/0/1/1/1/1/0/0/0/0/0/1
```
先頭 3 値が RGB (0.0～1.0)。

### 7.6 TIMELINES (タイムライン)

RACK TYPE=1 の場合の子要素。

```xml
<TIMELINES>
  <TIMELINE DASUID="..." NAME="0" INDEX="0"
      DASTLLOCKED="0" DASTLMUTED="0" DASTLFOLDED="1">
    <BLOCKS>
      <BLOCK TYPE="1" DASUID="..." NAME="New Scene"
          START="999" END="2249" POSITION="0"
          FADEIN="0" FADEOUT="0" SPEED="1.6"
          ALLOWLOOP="1" CONFORM_TO_TEMPO="1"
          SCENEUUID="(scene-uuid)">
        <DASTLSUBLINES DASTLNB="0"/>
      </BLOCK>
    </BLOCKS>
  </TIMELINE>
</TIMELINES>
```

#### TIMELINE

| 属性 | 説明 |
|------|------|
| `DASUID` | タイムラインの UUID |
| `NAME` | タイムライン名 |
| `INDEX` | タイムラインインデックス |
| `DASTLLOCKED` | ロック状態 |
| `DASTLMUTED` | ミュート状態 |
| `DASTLFOLDED` | 折りたたみ状態 |

#### BLOCK

| 属性 | 説明 |
|------|------|
| `TYPE` | ブロックタイプ |
| `DASUID` | ブロックの UUID |
| `NAME` | ブロック名 |
| `START` | 開始位置 (ms) |
| `END` | 終了位置 (ms) |
| `POSITION` | ポジション |
| `FADEIN` | フェードイン時間 (ms) |
| `FADEOUT` | フェードアウト時間 (ms) |
| `SPEED` | 再生速度倍率 |
| `ALLOWLOOP` | ループ許可フラグ |
| `CONFORM_TO_TEMPO` | テンポ同期フラグ |
| `SCENEUUID` | 参照先シーンの UUID |

### 7.7 STEPS (ステップシーケンス)

RACK TYPE=9 の場合の子要素。

```xml
<STEPS NB="2">
  <STEP WAITTIME="25" FADETIME="0">
    <FIXTUREDATAS NB="3">
      <FIXTUREDATA FIXTURE="(uuid)" DATA="(Base64+zlib)"/>
    </FIXTUREDATAS>
  </STEP>
</STEPS>
```

| 属性 | 説明 |
|------|------|
| `NB` | ステップ数 |
| `WAITTIME` | ウェイト時間 (×100ms = 2.5秒) |
| `FADETIME` | フェード時間 |

### 7.8 MAPPING (2Dマッピング)

```xml
<MAPPING NAME="Rectangle" DASUID="..." TYPE="0"
    X="340" Y="150" SX="130" SY="50" ANGLE="0" LOCKED="0"/>
```

| 属性 | 説明 |
|------|------|
| `NAME` | マッピング形状名 |
| `TYPE` | マッピングタイプ |
| `X` / `Y` | 位置座標 |
| `SX` / `SY` | サイズ |
| `ANGLE` | 回転角度 |
| `LOCKED` | ロック状態 |

### 7.9 PRESETS (エフェクトプリセット参照)

```xml
<PRESETS>
  <PRESET SSLFIXTURE="" SSLCHANNEL="-1" SSLPRESET="4" MIN="0" MAX="1">
    <BEAMS>
      <BEAM FIXTURE="(uuid)" BEAMID="0"/>
    </BEAMS>
  </PRESET>
</PRESETS>
```

| 属性 | 説明 |
|------|------|
| `SSLFIXTURE` | 対象フィクスチャ (空=全体) |
| `SSLCHANNEL` | 対象チャンネル (-1=自動) |
| `SSLPRESET` | プリセットタイプ (例: 4=Dimmer) |
| `MIN` / `MAX` | 値の範囲 |

### 7.10 DAS_SELECTIONS

```xml
<DAS_SELECTIONS SELECTED_LIVE_BANK="0">
  <SELECTED_LIVE_SCENES DASUID="(scene-uuid)"/>
</DAS_SELECTIONS>
```

| 属性 | 説明 |
|------|------|
| `SELECTED_LIVE_BANK` | 現在選択されているバンクのインデックス |
| `DASUID` | 現在選択されているシーンの UUID |

---

## 8. SHORTCUTS

外部デバイス（MIDI、タッチUI、OSC等）からのイベントマッピング。
タッチUI のボタン/フェーダー操作や、OSC メッセージ受信時のアクションを定義する。

```xml
<SHORTCUTS>
  <!-- タッチUI マッピング例 -->
  <SHORTCUT TYPE="2">
    <EVENT DATA="(touchcontrol-uuid):(channel)"/>
    <ACTION TYPE="107" TARGET="(scene-uuid)" TARGETINDEX="2001"/>
    <SETTINGS SMODE="0" CMODE="0" TMODE="0"
        MIN="0" MAX="1" INC="0.001" LOOP="0" FLASH="1" INV="0"/>
  </SHORTCUT>
  <!-- OSC マッピング例 -->
  <SHORTCUT TYPE="2">
    <EVENT DATA="(osc-control-uuid):(channel)"/>
    <ACTION TYPE="107" TARGET="(scene-uuid)" TARGETINDEX="9000"/>
    <SETTINGS SMODE="0" CMODE="0" TMODE="0"
        MIN="0" MAX="1" INC="0.001" LOOP="0" FLASH="1" INV="0"/>
  </SHORTCUT>
</SHORTCUTS>
```

### 8.1 SHORTCUT

| 属性 | 説明 |
|------|------|
| `TYPE` | ショートカットタイプ |

### 8.2 EVENT

| 属性 | 説明 |
|------|------|
| `DATA` | `(コントロールUUID):(チャンネル番号)` 形式 |

EVENT DATA の UUID 部分は、**マッピングソースの種別**によって参照先が異なる:

| ソース | UUID の参照先 |
|--------|-------------|
| タッチUI | `TOUCHCONTROL.DASUID`（TOUCH セクション内） |
| OSC | OSC 仮想コントロールの UUID（TOUCH セクションには存在しない） |

チャンネル番号は通常 `0` だが、フェーダー等の連続値コントロールでは `1` が使用される場合がある。

### 8.3 ACTION

| 属性 | 説明 |
|------|------|
| `TYPE` | アクションタイプコード（下表参照） |
| `TARGET` | ターゲット要素の DASUID |
| `TARGETINDEX` | ターゲット位置インデックス（下記参照） |

**ACTION TYPE (観測値):**

| TYPE | アクション |
|------|-----------|
| `107` | シーンの再生/停止 |
| `231` | グループディマー |
| `232` | グループ HUE コントロール |
| `235` | グループブラックアウト |

**TARGETINDEX エンコーディング:**

TARGETINDEX は以下の計算式でバンク内のシーン位置をエンコードする（タッチUI・OSC 共通）:

```
TARGETINDEX = bank_index × 1000 + scene_index_within_bank
```

> ⚠️ **重要:** TARGETINDEX はマッピング作成時のバンク/シーン位置に基づいて決定され、
> その後バンクの追加・削除・並び替えを行っても **値は更新されない**。
> したがって、実際のシーン参照には常に `TARGET`（DASUID）が使用される。
> TARGETINDEX は作成時点のスナップショットであり、現在の位置を保証しない。

**エンコード例:**

| TARGETINDEX | 計算 | 意味 |
|-------------|------|------|
| `0` | 0×1000+0 | Bank 0 の 1番目のシーン、またはグループ操作 |
| `4` | 0×1000+4 | Bank 0 の 5番目のシーン |
| `2001` | 2×1000+1 | Bank 2 の 2番目のシーン |
| `3000` | 3×1000+0 | Bank 3 の 1番目のシーン |
| `36000` | 36×1000+0 | Bank 36 の 1番目のシーン |

**バンク移動による不一致の実例:**

```
作成時: Bank 9 [test] → Step1 (scene 0) → TARGETINDEX = 9×1000+0 = 9000
現　在: Bank 8 [キャリブレーション] → Step1 (scene 0)
  ※ Bank 7 と 8 の間にバンクが削除され、Bank 9 → Bank 8 にシフトしたが
    TARGETINDEX=9000 は変更されない。TARGET (DASUID) で正しいシーンを特定する。
```

### 8.4 SETTINGS

| 属性 | 説明 |
|------|------|
| `SMODE` | スライダーモード |
| `CMODE` | コントロールモード |
| `TMODE` | トリガーモード |
| `MIN` / `MAX` | 値の範囲 |
| `INC` | 増分値 |
| `LOOP` | ループフラグ |
| `FLASH` | フラッシュフラグ |
| `INV` | 反転フラグ |

### 8.5 OSC マッピング

Daslight5 の OSC マッピング機能を使用すると、SHORTCUTS セクションに SHORTCUT エントリが追加される。
構造はタッチUI マッピングと同一（`TYPE="2"`）だが、以下の点が異なる:

| 項目 | タッチUI | OSC |
|------|---------|-----|
| EVENT DATA の UUID | `TOUCHCONTROL.DASUID` を参照 | OSC 固有の仮想コントロール UUID |
| TOUCH セクションとの連携 | `TOUCHCONTROL` 要素が存在 | 対応する TOUCH 要素なし |
| TARGETINDEX | 同一の計算式（`bank_index × 1000 + scene_index`） | 同一の計算式 |

> **注:** TARGETINDEX のエンコーディングはタッチUI・OSC で共通。
> 以前 7000/9000 番台が「OSC 専用レンジ」と思われたが、実データの検証により
> 単にバンク番号が大きいシーン（Bank 7, Bank 9 等）への通常のマッピングであることが判明。

**OSC マッピングの実例:**

```xml
<!-- OSC → Bank 0, Scene 4 "気まぐれメルシィ新" の再生/停止 -->
<SHORTCUT TYPE="2">
    <EVENT DATA="(osc-virtual-uuid):0"/>
    <ACTION TYPE="107" TARGET="(scene-dasuid)" TARGETINDEX="4"/>
    <SETTINGS SMODE="0" CMODE="0" TMODE="0" MIN="0" MAX="1" INC="0.001" LOOP="0" FLASH="1" INV="0"/>
</SHORTCUT>

<!-- OSC → Bank 36, Scene 0 "千本桜" の再生/停止 -->
<SHORTCUT TYPE="2">
    <EVENT DATA="(osc-virtual-uuid):0"/>
    <ACTION TYPE="107" TARGET="(scene-dasuid)" TARGETINDEX="36000"/>
    <SETTINGS SMODE="0" CMODE="0" TMODE="0" MIN="0" MAX="1" INC="0.001" LOOP="0" FLASH="1" INV="0"/>
</SHORTCUT>
```

> **注:** OSC マッピングの EVENT DATA UUID は TOUCH セクション内のどの TOUCHCONTROL とも紐付かない。
> OSC エンジンが内部的にこの UUID を管理し、受信した OSC メッセージと対応付ける。

---

## 9. TOUCH

タッチUI のページとコントロール定義。

```xml
<TOUCH>
  <TOUCHPAGE DASUID="..." NAME="Page 1" TOUCHCONTROL="6">
    <TOUCHCONTROL DASUID="..." NAME="Label" TYPE="0"
        GRIDWIDTH="2" GRIDHEIGHT="1" GRIDPOSX="0" GRIDPOSY="0"
        ICONPATH="" PRESETICON="0"/>
  </TOUCHPAGE>
</TOUCH>
```

### 9.1 TOUCHPAGE

| 属性 | 説明 |
|------|------|
| `DASUID` | ページの UUID |
| `NAME` | ページ名 |
| `TOUCHCONTROL` | ページ内のコントロール数 |

### 9.2 TOUCHCONTROL

| 属性 | 説明 |
|------|------|
| `DASUID` | コントロールの UUID（SHORTCUT EVENT と紐付く） |
| `NAME` | コントロール表示名 |
| `TYPE` | コントロール種別（下表参照） |
| `GRIDWIDTH` / `GRIDHEIGHT` | グリッドサイズ |
| `GRIDPOSX` / `GRIDPOSY` | グリッド位置 |
| `CONTROLCOLOR` | コントロールカラー (オプション) |
| `ICONPATH` | アイコンパス |
| `PRESETICON` | プリセットアイコン ID |
| `FLASHMODE` | フラッシュモード (オプション、ボタンのみ) |

**TOUCHCONTROL TYPE (観測値):**

| TYPE | 種別 |
|------|------|
| `0` | ラベル |
| `2` | ボタン |
| `3` | フェーダー（縦スライダー） |
| `4` | XYパッド |

---

## 10. DEVICES

外部デバイスの接続設定。サンプルファイルでは空。

```xml
<DEVICES/>
```

---

## 付録A: 要素間のクロスリファレンス（参照関係）

DVC ファイル内の要素は **DASUID (UUID)** を主キーとして相互参照している。
以下に全参照チェーンを図解する。

### 全体リファレンスマップ

```
PATCHS DATA (エンコード済み)
  └── FIXTURE.DASUID ─────────────────────────┐  ← フィクスチャの定義元（マスター）
                                               │
FIXTUREGROUPS                                  │
  └── FIXTUREGROUP.DASUID ──────────────┐      │
        └── FIXTURE.DASUID ─────────────┼──────┤  ← PATCHS内FIXTUREのDASUID で参照
        └── BEAMSELECTION               │      │
              └── BEAM.FIXTURE ─────────┼──────┤  ← 同上
                                        │      │
SCENES                                  │      │
  └── BANK                              │      │
        └── SCENE.DASUID ──────────┐    │      │
              │                    │    │      │
              ├── FIXTUREDATAS     │    │      │
              │     └── FIXTUREDATA.FIXTURE ───┤  ← PATCHS内FIXTUREのDASUID で参照
              │                    │    │      │
              └── RACKS            │    │      │
                    ├── RACK (タイムライン)     │
                    │     └── BLOCK.SCENEUUID ─┤  ← SCENE.DASUID で参照
                    │                    │     │
                    ├── RACK (エフェクト)  │     │
                    │     └── BEAM.FIXTURE ────┤  ← PATCHS内FIXTUREのDASUID で参照
                    │                    │     │
                    └── RACK (ステップ)   │     │
                          └── STEP       │     │
                                └── FIXTUREDATA.FIXTURE ─┘
                                         │
SHORTCUTS                                │
  └── SHORTCUT                           │
        ├── EVENT.DATA ──────────────────┼──→ TOUCHCONTROL.DASUID（タッチUI時）
        │                                │    または OSC 仮想コントロール UUID（OSC時）
        └── ACTION.TARGET ───────────────┼──→ SCENE.DASUID または FIXTUREGROUP.DASUID
                                         │
TOUCH                                    │
  └── TOUCHPAGE                          │
        └── TOUCHCONTROL.DASUID ─────────┘  ← SHORTCUTのEVENTから参照される
                                              （OSCマッピングの場合は対応要素なし）
                                              
DAS_SELECTIONS
  └── SELECTED_LIVE_SCENES.DASUID ──────────→ SCENE.DASUID
```

### 参照チェーン詳細

#### ① フィクスチャ参照: `PATCHS FIXTURE.DASUID`

**定義元:** PATCHS DATA 内の `<FIXTURE DASUID="...">` がフィクスチャのマスター定義。

**参照元（すべて `FIXTURE` 属性または `DASUID` 属性で参照）:**

| 参照元要素 | 参照属性 | 参照先 |
|-----------|---------|--------|
| `FIXTUREGROUP > FIXTURES > FIXTURE` | `DASUID` | → PATCHS `FIXTURE.DASUID` |
| `FIXTUREGROUP > BEAMSELECTION > BEAM` | `FIXTURE` | → PATCHS `FIXTURE.DASUID` |
| `SCENE > FIXTUREDATAS > FIXTUREDATA` | `FIXTURE` | → PATCHS `FIXTURE.DASUID` |
| `STEP > FIXTUREDATAS > FIXTUREDATA` | `FIXTURE` | → PATCHS `FIXTURE.DASUID` |
| `RACK > BEAMS > BEAM` | `FIXTURE` | → PATCHS `FIXTURE.DASUID` |
| `RACK > PRESETS > PRESET > BEAMS > BEAM` | `FIXTURE` | → PATCHS `FIXTURE.DASUID` |

**実例:**
```
PATCHS: FIXTURE DASUID="5d7c2645-d53b-4e8a-bfed-01dbb3034e0c" NAME="lpc007" ADDRESS="1"
                    ↑
FIXTUREGROUP "All": FIXTURE DASUID="5d7c2645-d53b-4e8a-bfed-01dbb3034e0c"
                    ↑
SCENE "New Scene":  FIXTUREDATA FIXTURE="5d7c2645-d53b-4e8a-bfed-01dbb3034e0c"
                    ↑
RACK BEAMS:         BEAM FIXTURE="5d7c2645-d53b-4e8a-bfed-01dbb3034e0c" BEAMID="0"
```

#### ② シーン参照: `SCENE.DASUID`

**定義元:** `<SCENE DASUID="...">` がシーンのマスター定義。

| 参照元要素 | 参照属性 | 参照先 |
|-----------|---------|--------|
| `TIMELINE > BLOCK` | `SCENEUUID` | → `SCENE.DASUID` |
| `SHORTCUT > ACTION` (TYPE=107) | `TARGET` | → `SCENE.DASUID` |
| `DAS_SELECTIONS > SELECTED_LIVE_SCENES` | `DASUID` | → `SCENE.DASUID` |

**実例（スーパーシーン/タイムライン → シーン）:**
```
SCENE "New Scene" DASUID="28281441-..."  ← タイムラインを含むスーパーシーン
  └── RACK TYPE="1" (タイムライン)
        └── TIMELINE
              └── BLOCK SCENEUUID="921b7d59-..."  ─→ SCENE "New Scene" (Bank 2)
              └── BLOCK SCENEUUID="410a6ca7-..."  ─→ SCENE "New Scene" (Bank 2)
              └── BLOCK SCENEUUID="901d1098-..."  ─→ SCENE "aaa"      (Bank 3)
```

> **注:** タイムライン BLOCK は `SCENEUUID` 属性で別のシーンを参照する。
> 参照先シーンは別のバンクに存在していても参照可能。

#### ③ フィクスチャグループ参照: `FIXTUREGROUP.DASUID`

**定義元:** `<FIXTUREGROUP DASUID="...">` がグループのマスター定義。

| 参照元要素 | 参照属性 | 参照先 |
|-----------|---------|--------|
| `SHORTCUT > ACTION` (TYPE=231,232,235) | `TARGET` | → `FIXTUREGROUP.DASUID` |

**実例:**
```
TOUCHCONTROL "Dimmer"   → SHORTCUT ACTION TYPE=231 TARGET="751fd06f-..." → FIXTUREGROUP "All"
TOUCHCONTROL "Blackout" → SHORTCUT ACTION TYPE=235 TARGET="751fd06f-..." → FIXTUREGROUP "All"
TOUCHCONTROL "HUE"      → SHORTCUT ACTION TYPE=232 TARGET="751fd06f-..." → FIXTUREGROUP "All"
```

#### ④ タッチコントロール / OSC コントロール参照

**タッチUI の場合:**

**定義元:** `<TOUCHCONTROL DASUID="...">` がコントロールのマスター定義。

| 参照元要素 | 参照属性 | 参照先 |
|-----------|---------|--------|
| `SHORTCUT > EVENT` | `DATA` (UUID部分) | → `TOUCHCONTROL.DASUID` |

**EVENT DATA 形式:** `"(TOUCHCONTROL DASUID):(チャンネル番号)"`

**実例:**
```
TOUCHCONTROL "frfwas" DASUID="58da7698-..."
                ↑
SHORTCUT EVENT DATA="58da7698-70f7-4619-af38-aae2f2950a04:0"
         ACTION TYPE=107 TARGET="081378df-..."  → SCENE "frfwas"
         TARGETINDEX=2001  ← バンク位置準拠
```

**OSC マッピングの場合:**

EVENT DATA の UUID は OSC エンジン内部で管理される仮想コントロール UUID であり、
TOUCH セクション内には対応する要素が存在しない。

| 参照元要素 | 参照属性 | 参照先 |
|-----------|---------|--------|
| `SHORTCUT > EVENT` | `DATA` (UUID部分) | → OSC 仮想コントロール UUID（TOUCH 外） |

**実例:**
```
(TOUCH セクションに対応要素なし)

SHORTCUT EVENT DATA="cf7e2578-885e-41c3-8112-f1a0867a1d23:0"
         ACTION TYPE=107 TARGET="fd88edc5-..."  → SCENE
         TARGETINDEX=9000  ← Bank 9 の通常レンジ（バンク番号由来）
```

### 参照フロー図（ユーザー操作の流れ）

```
[タッチUI操作]
     │
     ▼
TOUCHCONTROL.DASUID ──(EVENT.DATA)──→ SHORTCUT
     │                                    │
     │                              ACTION.TARGET
     │                                    │
     ▼                                    ▼
  (表示用)                    SCENE.DASUID  or  FIXTUREGROUP.DASUID
                                    │
                              ┌─────┴──────┐
                              ▼            ▼
                    FIXTUREDATA      RACK > BLOCK
                    .FIXTURE          .SCENEUUID
                        │                 │
                        ▼                 ▼
                  PATCHS FIXTURE     他の SCENE
                  .DASUID            .DASUID
                        │
                        ▼
                  DMX ADDRESS / UNIVERS
                  (実際の出力先)
```

### 参照に使われる属性名の一覧

| 属性名 | 所在要素 | 参照先の定義元 |
|--------|---------|---------------|
| `DASUID` | FIXTUREGROUP>FIXTURE | PATCHS内 FIXTURE |
| `FIXTURE` | FIXTUREDATA, BEAM | PATCHS内 FIXTURE.DASUID |
| `SCENEUUID` | BLOCK | SCENE.DASUID |
| `TARGET` | ACTION | SCENE.DASUID または FIXTUREGROUP.DASUID |
| `DATA` (UUID部分) | EVENT | TOUCHCONTROL.DASUID |
| `DASUID` | SELECTED_LIVE_SCENES | SCENE.DASUID |

---

## 付録B: 共通データ型

### DASUID
UUID v4 形式の一意識別子。例: `"751fd06f-13b9-4734-bd08-2736f1f80b90"`

ファイル内のほぼ全ての要素に付与され、要素間の参照キーとして使用される（上記 付録A 参照）。

### COLOR 属性
`#AARRGGBB` 形式の 32bit カラー値。例: `"#ffffcf08"` (α=FF, R=FF, G=CF, B=08)
