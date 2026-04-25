package pipeline

import (
	"ap-music/internal/domain"
	"context"
	"fmt"
	"log/slog"
)

// MusicPipeline は Collect -> Lyrics -> Compose -> GenerateAudio -> Publish -> Notify を統制します。
type MusicPipeline struct {
	Collector domain.Collector
	Lyricist  domain.Lyricist
	Composer  domain.Composer
	Generator domain.AudioGenerator
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
			if notifyErr := p.Notifier.NotifyError(ctx, err, notifReq); notifyErr != nil {
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

	// Step B-1: 作詞フェーズ
	lyricsDraft, err := p.Lyricist.GenerateLyrics(ctx, contextText, task.AIModels.TextModel)
	if err != nil {
		return fmt.Errorf("lyrics generation failed: %w", err)
	}

	// Step B-2: 作曲（レシピ構築）フェーズ
	recipe, err := p.Composer.Compose(ctx, lyricsDraft, task.AIModels.TextModel)
	if err != nil {
		return fmt.Errorf("compose phase failed: %w", err)
	}

	// Compose 時にデフォルト値が設定される可能性があるため、
	// task に含まれるユーザー指定のモデル情報で上書き（または補完）します。
	recipe.AIModels = task.AIModels

	// Step C: 音楽生成（音声バイナリの生成）
	wav, err := p.Generator.GenerateAudio(ctx, recipe)
	if err != nil {
		return fmt.Errorf("audio generation (lyria engine) failed: %w", err)
	}

	// Step D: 成果物の永続化
	result, err := p.Publisher.Publish(ctx, task, recipe, wav)
	if err != nil {
		return fmt.Errorf("publish phase failed: %w", err)
	}

	// Step E: 成功通知
	if nErr := p.Notifier.NotifyWithRequest(ctx, result, notifReq); nErr != nil {
		slog.WarnContext(ctx, "success notification failed", "error", nErr)
	}

	return nil
}
