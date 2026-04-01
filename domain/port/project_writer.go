package port

import (
	"context"

	"github.com/otoyuzu705/DasScript/domain/model"
)

// ProjectWriter は model.Project を dvc5 ファイルとして書き出す。
type ProjectWriter interface {
	// Write は project.Path を出力先として書き出す。
	Write(ctx context.Context, project *model.Project) error
	// WriteTo は指定した path に書き出す。
	WriteTo(ctx context.Context, project *model.Project, path string) error
}
