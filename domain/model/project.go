package model

// Project は dvc5 プロジェクトファイル全体を表す。
type Project struct {
	Path    string // 元ファイルのパス。save時の出力先デフォルト値として使用する
	Version string // DASBUILD属性値（例: "24.1205.158.90"）
	Scenes  Scenes
}
