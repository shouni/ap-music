package builder

import (
	"ap-music/internal/domain"
	"ap-music/internal/pipeline"
)

// buildPipeline は、提供された設定と各コンポーネントを使用して新しいパイプラインを初期化して返します。
func buildPipeline(
	collector domain.Collector,
	runner domain.MusicRunner,
	publisher domain.Publisher,
	notifier domain.Notifier,
) (domain.Pipeline, error) {
	return pipeline.MusicPipeline{
		Collector: collector,
		Lyricist:  runner,
		Composer:  runner,
		Generator: runner,
		Publisher: publisher,
		Notifier:  notifier,
	}, nil
}
