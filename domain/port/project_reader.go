package port

import (
	"context"

	"github.com/otoyuzu705/DasScript/domain/model"
)

// ProjectReader は dvc5 ファイルを読み込み model.Project を返す。
type ProjectReader interface {
	Read(ctx context.Context, path string) (*model.Project, error)
}
