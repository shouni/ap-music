package builder

import (
	"ap-music/internal/config"
	"ap-music/internal/domain"
	"ap-music/internal/pipeline"
)

// buildPipeline は、提供された設定と各コンポーネントを使用して新しいパイプラインを初期化して返します。
func buildPipeline(cfg *config.Config, workflows domain.Workflows, slack domain.Notifier) (domain.Pipeline, error) {
	p, err := pipeline.NewMangaPipeline(cfg, workflows, slack)
	if err != nil {
		return nil, err
	}

	return p, nil
}
