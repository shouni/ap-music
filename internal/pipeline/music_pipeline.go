package pipeline

import (
	"context"
	"fmt"
	"log/slog"
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

// Execute はタスクを実行し、成功・失敗に関わらず通知を試みます。
func (p MusicPipeline) Execute(ctx context.Context, task domain.Task) (err error) {
	// 1. 通知用のメタデータを準備
	notifReq := domain.NotificationRequest{
		SourceURL:      task.RequestURL,
		OutputCategory: "Music Generation",
	}

	// 2. エラートラップ用の defer 処理
	defer func() {
		if err != nil {
			// エラーが発生していた場合、Slack に詳細を通知
			if notifyErr := p.Notifier.NotifyError(ctx, err, notifReq); notifyErr != nil {
				// 通知自体の失敗は slog で構造化ログとして出力
				slog.ErrorContext(ctx, "failed to send error notification",
					"original_error", err,
					"notification_error", notifyErr,
				)
			}
		}
	}()

	// 3. 各フェーズの実行（Early Return パターン）

	// Step A: コンテキスト収集
	contextText, err := p.Collector.Collect(ctx, task)
	if err != nil {
		return fmt.Errorf("collect phase failed: %w", err)
	}

	// Step B: レシピ構築（LLM）
	recipe, err := p.Composer.Compose(ctx, contextText)
	if err != nil {
		return fmt.Errorf("compose (recipe generation) failed: %w", err)
	}

	// モデルの上書き制御（空文字列による破壊をガード）
	if model := strings.TrimSpace(task.Model); model != "" {
		if recipe.Metadata == nil {
			recipe.Metadata = make(map[string]string)
		}
		recipe.Metadata["model"] = model
	}

	// Step C: 音楽生成
	wav, err := p.Generator.Generate(ctx, recipe)
	if err != nil {
		return fmt.Errorf("generate (lyria engine) failed: %w", err)
	}

	// Step D: 成果物の永続化（GCS保存）
	result, err := p.Publisher.Publish(ctx, task, recipe, wav)
	if err != nil {
		return fmt.Errorf("publish (storage) failed: %w", err)
	}

	// Step E: 成功通知
	if nErr := p.Notifier.NotifyWithRequest(ctx, result, notifReq); nErr != nil {
		slog.WarnContext(ctx, "success notification failed", "error", nErr)
	}

	return nil
}
