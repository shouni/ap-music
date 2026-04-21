package builder

import (
	"ap-music/internal/domain"
	"ap-music/internal/pipeline"
)

// buildPipeline は、提供された設定と各コンポーネントを使用して新しいパイプラインを初期化して返します。
func buildPipeline(
	collector domain.Collector,
	composer domain.Composer,
	generator domain.Generator,
	publisher domain.Publisher,
	notifier domain.Notifier,
) (domain.Pipeline, error) {
	return pipeline.MusicPipeline{
		Collector: collector,
		Composer:  composer,
		Generator: generator,
		Publisher: publisher,
		Notifier:  notifier,
	}, nil
}
