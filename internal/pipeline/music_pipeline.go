package pipeline

import (
	"context"

	"ap-music/internal/domain"
)

// MusicPipeline は Collect -> Compose -> Generate -> Publish -> Notify を統制します。
type MusicPipeline struct {
	Collector domain.Collector
	Composer  domain.Composer
	Generator domain.Generator
	Publisher domain.Publisher
	Notifier  domain.Notifier
}

// Execute はタスクを実行します。
func (p MusicPipeline) Execute(ctx context.Context, task domain.Task) (domain.PublishResult, error) {
	contextText, err := p.Collector.Collect(ctx, task)
	if err != nil {
		return domain.PublishResult{}, err
	}

	recipe, err := p.Composer.Compose(ctx, contextText)
	if err != nil {
		return domain.PublishResult{}, err
	}

	mp3, err := p.Generator.Generate(ctx, recipe)
	if err != nil {
		return domain.PublishResult{}, err
	}

	result, err := p.Publisher.Publish(ctx, task, mp3)
	if err != nil {
		return domain.PublishResult{}, err
	}

	if err := p.Notifier.Notify(ctx, result); err != nil {
		return domain.PublishResult{}, err
	}

	return result, nil
}
