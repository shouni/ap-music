package domain

import "context"

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
