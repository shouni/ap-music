package pipeline

import (
	"context"
	"fmt"
	"strings"

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
func (p MusicPipeline) Execute(ctx context.Context, task domain.Task) error {
	contextText, err := p.Collector.Collect(ctx, task)
	if err != nil {
		return err
	}

	recipe, err := p.Composer.Compose(ctx, contextText)
	if err != nil {
		return err
	}
	if model := strings.TrimSpace(task.Model); model != "" {
		if recipe.Metadata == nil {
			recipe.Metadata = make(map[string]string)
		}
		recipe.Metadata["model"] = model
	}

	wav, err := p.Generator.Generate(ctx, recipe)
	if err != nil {
		return err
	}

	result, err := p.Publisher.Publish(ctx, task, recipe, wav)
	if err != nil {
		return err
	}

	if err := p.Notifier.Notify(ctx, result); err != nil {
		return fmt.Errorf("notify failed: %w", err)
	}

	return nil
}
