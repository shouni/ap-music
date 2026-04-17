package domain

import "context"

// Publisher は生成物の保存先を抽象化します。
type Publisher interface {
	Publish(ctx context.Context, task Task, mp3 []byte) (PublishResult, error)
}

// Notifier は完了通知を抽象化します。
type Notifier interface {
	Notify(ctx context.Context, result PublishResult) error
}

// TaskQueue は非同期キューを抽象化します。
type TaskQueue interface {
	Enqueue(ctx context.Context, task Task) error
}
