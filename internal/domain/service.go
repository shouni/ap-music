package domain

import "context"

// Pipeline は、デコードされたペイロードを受け取って実際の処理を行うインターフェースです。
type Pipeline interface {
	Execute(ctx context.Context, payload Task) (err error)
}

// Collector は入力コンテキスト収集を行います。
type Collector interface {
	Collect(ctx context.Context, task Task) (string, error)
}

// Lyricist は歌詞生成を担う役割です。
type Lyricist interface {
	GenerateLyrics(ctx context.Context, input string) (LyricsDraft, error)
}

// Composer は楽曲の設計（レシピ構築）を担う役割です。
type Composer interface {
	Compose(ctx context.Context, lyrics LyricsDraft) (MusicRecipe, error)
}

// AudioGenerator は MusicRecipe から音声バイナリを生成します。
type AudioGenerator interface {
	GenerateAudio(ctx context.Context, recipe MusicRecipe) ([]byte, error)
}

// MusicRunner は音楽生成のプロセスを統合したインターフェースです。
type MusicRunner interface {
	Lyricist
	Composer
	AudioGenerator
}
