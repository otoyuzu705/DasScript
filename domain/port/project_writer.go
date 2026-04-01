package port

import "github.com/otoyuzu705/DasScript/domain/model"

// ProjectWriter は model.Project を dvc5 ファイルとして書き出す。
// path が空文字の場合は project.Path を使用する。
type ProjectWriter interface {
	Write(project *model.Project, path string) error
}
