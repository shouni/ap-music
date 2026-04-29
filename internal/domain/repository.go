package domain

import "context"

// Publisher は生成物の保存先を抽象化します。
type Publisher interface {
	Publish(ctx context.Context, task Task, recipe *MusicRecipe, audioData []byte) (*PublishResult, error)
}

// StorageCleaner は、書き込み途中で残った成果物を削除するための抽象です。
type StorageCleaner interface {
	Delete(ctx context.Context, uri string) error
}

// Notifier は完了通知を抽象化します。
type Notifier interface {
	Notify(ctx context.Context, result *PublishResult) error
	NotifyWithRequest(ctx context.Context, result *PublishResult, req NotificationRequest) error
	NotifyError(ctx context.Context, errDetail error, req NotificationRequest) error
}

// TaskQueue は非同期キューを抽象化します。
type TaskQueue interface {
	Enqueue(ctx context.Context, task Task) error
}

// PromptGenerator は、AIプロンプトを生成するインターフェースです。
type PromptGenerator interface {
	GenerateLyrics(content string) (string, error)
	GenerateRecipe(mode string, lyrics *LyricsDraft) (string, error)
}

// MusicRepository は、生成された楽曲の成果物（レシピや履歴）を管理するための定義です
// 永続化および取得操作を定義するインターフェースなのだ。
type MusicRepository interface {
	// ListHistory は、指定されたユーザーに関連付けられた楽曲生成履歴の一覧を取得します。
	ListHistory(ctx context.Context, userID string) ([]MusicHistory, error)
	// GetRecipe は、指定されたジョブIDに対応する詳細な楽曲設計図（MusicRecipe）を取得します。
	GetRecipe(ctx context.Context, jobID string) (*MusicRecipe, error)
}
