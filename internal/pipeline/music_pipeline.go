package pipeline

import (
	"context"
	"fmt"
	"log/slog"

	"ap-music/internal/domain"
)

// MusicPipeline は外部システムとの連携（Collect/Publish/Notify）と、
// コア生成ロジック（MusicGenerator）を統制します。
type MusicPipeline struct {
	Collector      domain.Collector
	MusicGenerator domain.MusicRunner
	AudioGenerator domain.AudioGenerator
	Publisher      domain.Publisher
	Notifier       domain.Notifier
}

// Execute はタスクを実行し、成功・失敗に関わらず通知を試みます。
func (p MusicPipeline) Execute(ctx context.Context, task domain.Task) (err error) {
	// 1. 通知用のメタデータを準備
	notifReq := domain.NotificationRequest{
		Command:   string(task.NormalizedCommand()),
		SourceURL: task.RequestURL,
		Mode:      notificationMode(task),
		Seed:      task.AIModels.Seed,
	}

	// 2. エラートラップ用の defer 処理
	defer func() {
		if err != nil {
			if notifyErr := p.Notifier.NotifyError(ctx, err, notifReq); notifyErr != nil {
				slog.ErrorContext(ctx, "failed to send error notification",
					"original_error", err,
					"notification_error", notifyErr,
				)
			}
		}
	}()

	switch task.NormalizedCommand() {
	case domain.TaskCommandCompose:
		return p.executeCompose(ctx, task, notifReq)
	case domain.TaskCommandGenerateFromRecipe:
		return p.executeGenerateFromRecipe(ctx, task, notifReq)
	default:
		return fmt.Errorf("unsupported command: %s", task.Command)
	}
}

func notificationMode(task domain.Task) string {
	if task.NormalizedCommand() != domain.TaskCommandCompose {
		return ""
	}
	return task.AIModels.ComposeMode
}

func (p MusicPipeline) executeCompose(ctx context.Context, task domain.Task, notifReq domain.NotificationRequest) error {
	// 3. 各フェーズの実行

	// Step A: コンテキスト収集（外部情報の取得）
	contextText, err := p.Collector.Collect(ctx, task)
	if err != nil {
		return fmt.Errorf("collect phase failed: %w", err)
	}

	// Step B: コア生成プロセス（AIによる作詞・作曲・音声生成を一括実行）
	// 生成の順序や中間データの扱いは MusicGenerator が隠蔽する
	recipe, wav, err := p.MusicGenerator.Run(ctx, task, contextText)
	if err != nil {
		return fmt.Errorf("music generation failed: %w", err)
	}

	// Step C: 成果物の永続化
	result, err := p.Publisher.Publish(ctx, task, recipe, wav)
	if err != nil {
		return fmt.Errorf("publish phase failed: %w", err)
	}

	// Step D: 成功通知
	if nErr := p.Notifier.NotifyWithRequest(ctx, result, notifReq); nErr != nil {
		slog.WarnContext(ctx, "success notification failed", "error", nErr)
	}

	return nil
}

func (p MusicPipeline) executeGenerateFromRecipe(ctx context.Context, task domain.Task, notifReq domain.NotificationRequest) error {
	if task.Recipe == nil {
		return fmt.Errorf("recipe is required for command %q", domain.TaskCommandGenerateFromRecipe)
	}
	recipe := task.Recipe
	if err := recipe.ValidateForGeneration(); err != nil {
		return fmt.Errorf("recipe validation failed: %w", err)
	}
	applyTaskOverridesToRecipe(task, recipe)

	wav, err := p.AudioGenerator.GenerateAudio(ctx, recipe, nil)
	if err != nil {
		return fmt.Errorf("audio generation failed: %w", err)
	}

	result, err := p.Publisher.Publish(ctx, task, recipe, wav)
	if err != nil {
		return fmt.Errorf("publish phase failed: %w", err)
	}

	if nErr := p.Notifier.NotifyWithRequest(ctx, result, notifReq); nErr != nil {
		slog.WarnContext(ctx, "success notification failed", "error", nErr)
	}

	return nil
}

func applyTaskOverridesToRecipe(task domain.Task, recipe *domain.MusicRecipe) {
	if task.AIModels.AudioModel != "" {
		recipe.AIModels.AudioModel = task.AIModels.AudioModel
	}
	if task.AIModels.Seed != nil {
		recipe.AIModels.Seed = task.AIModels.Seed
	}
}
