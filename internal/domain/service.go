package domain

import "context"

// Pipeline は、デコードされたペイロードを受け取って実際の処理を行うインターフェースです。
type Pipeline interface {
	// Execute は、指定されたコンテキストに基づいて GenerateTaskPayload を処理し、問題が発生した場合はエラーを返します。
	Execute(ctx context.Context, payload Task) (err error)
}

// Collector は入力コンテキスト収集を行います。
type Collector interface {
	Collect(ctx context.Context, task Task) (string, error)
}

// Composer はコンテキストから MusicRecipe を生成します。
type Composer interface {
	Compose(ctx context.Context, input string) (MusicRecipe, error)
}

// Generator は MusicRecipe から音楽バイナリを生成します。
type Generator interface {
	Generate(ctx context.Context, recipe MusicRecipe) ([]byte, error)
}
