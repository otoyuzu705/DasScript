package model

// Scenes は SCENES 要素全体を表す。
type Scenes struct {
	Banks []Bank
}

// Bank は BANK 要素を表す。
// Index は Scenes.Banks 内での0始まり位置であり、バンクの並び順管理に使用する。
// Scenes は Bank 内のシーンリストであり、順序が XML の出現順と一致する。
type Bank struct {
	DASUID string
	Name   string
	Index  int     // 読み取り専用。usecase層が更新する
	Scenes []Scene // Bank内のシーンリスト。NBSCENE属性はlen(Scenes)から算出する
}

// Scene は SCENE 要素を表す。
// Phase 1 では Name と DASUID のみを扱う。
type Scene struct {
	DASUID string
	Name   string
	Index  int // 読み取り専用
}
