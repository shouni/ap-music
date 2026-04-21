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

// Execute はタスクを実行し、成功・失敗に関わらず通知を試みます。
func (p MusicPipeline) Execute(ctx context.Context, task domain.Task) error {
	// 1. 通知用のメタデータを準備
	notifReq := domain.NotificationRequest{
		SourceURL:      task.RequestURL,
		OutputCategory: "Music Generation",
	}

	// 2. エラートラップ用の変数と defer 処理
	// どのステップで失敗しても、必ず NotifyError を呼び出す設計
	var finalErr error
	defer func() {
		if finalErr != nil {
			// エラーが発生していた場合、Slack に詳細を通知
			// 通知自体のエラーはログに残すが、パイプラインの元エラーを優先する
			if err := p.Notifier.NotifyError(ctx, finalErr, notifReq); err != nil {
				fmt.Printf("critical: failed to send error notification: %v\n", err)
			}
		}
	}()

	// 3. 各フェーズの実行（Early Return パターン）

	// Step A: コンテキスト収集
	contextText, err := p.Collector.Collect(ctx, task)
	if err != nil {
		finalErr = fmt.Errorf("collect phase failed: %w", err)
		return finalErr
	}

	// Step B: レシピ構築（LLM）
	recipe, err := p.Composer.Compose(ctx, contextText)
	if err != nil {
		finalErr = fmt.Errorf("compose (recipe generation) failed: %w", err)
		return finalErr
	}

	// モデルの上書き制御（空文字列による破壊をガード）
	if model := strings.TrimSpace(task.Model); model != "" {
		if recipe.Metadata == nil {
			recipe.Metadata = make(map[string]string)
		}
		// フォームから送られた不完全な名前（lyria-3等）を正規化する余地を残す
		recipe.Metadata["model"] = model
	}

	// Step C: 音楽生成（Lyria 3）
	wav, err := p.Generator.Generate(ctx, recipe)
	if err != nil {
		// ここで BLOCK_REASON: OTHER などの詳細が捕捉される
		finalErr = fmt.Errorf("generate (lyria engine) failed: %w", err)
		return finalErr
	}

	// Step D: 成果物の永続化（GCS保存）
	result, err := p.Publisher.Publish(ctx, task, recipe, wav)
	if err != nil {
		finalErr = fmt.Errorf("publish (storage) failed: %w", err)
		return finalErr
	}

	// Step E: 成功通知（署名付きURL付き）
	if err := p.Notifier.NotifyWithRequest(ctx, result, notifReq); err != nil {
		finalErr = fmt.Errorf("success notification failed: %w", err)
		return finalErr
	}

	return nil
}
