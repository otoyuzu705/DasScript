package port

import "github.com/otoyuzu705/DasScript/domain/model"

// ProjectReader は dvc5 ファイルを読み込み model.Project を返す。
type ProjectReader interface {
	Read(path string) (*model.Project, error)
}
