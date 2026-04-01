package port

import (
	"context"

	"github.com/otoyuzu705/DasScript/domain/model"
)

// ScriptRunner は Starlark スクリプトを実行し、変更後の Project を返す。
// スクリプト内で save_project が呼ばれた場合は Runner 側で処理する。
type ScriptRunner interface {
	Run(ctx context.Context, scriptPath string, project *model.Project) (*model.Project, error)
}
