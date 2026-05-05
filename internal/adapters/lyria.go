package adapters

import (
	"context"
	"errors"

	"github.com/shouni/go-gemini-client/gemini"
	"golang.org/x/time/rate"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

// LyriaAdapter は、歌詞生成・作曲・音声生成を束ねるファサードです。
type LyriaAdapter struct {
	lyricist domain.Lyricist
	composer domain.Composer
	audio    domain.AudioGenerator
}

// NewLyriaAdapter は、指定されたコンテキストと構成を使用して、新しい LyriaAdapter を初期化して返します。
func NewLyriaAdapter(cfg *config.Config, aiClient gemini.Generator, promptGen domain.PromptGenerator) (*LyriaAdapter, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if aiClient == nil {
		return nil, errors.New("aiClient is required")
	}
	if promptGen == nil {
		return nil, errors.New("promptGen is required")
	}
	if cfg.GeminiModel == "" {
		return nil, errors.New("GeminiModel is required but not set")
	}
	if cfg.LyriaModel == "" {
		return nil, errors.New("LyriaModel is required but not set")
	}

	limiter := rate.NewLimiter(rate.Every(cfg.RateInterval), 1)

	return &LyriaAdapter{
		lyricist: &lyriaLyricist{
			aiClient:     aiClient,
			promptGen:    promptGen,
			defaultModel: cfg.GeminiModel,
		},
		composer: &lyriaComposer{
			aiClient:     aiClient,
			promptGen:    promptGen,
			defaultModel: cfg.GeminiModel,
		},
		audio: &lyriaAudioGenerator{
			aiClient:          aiClient,
			promptBuilder:     lyriaAudioPromptBuilder{},
			limiter:           limiter,
			maxConcurrency:    cfg.MaxConcurrency,
			defaultLyriaModel: cfg.LyriaModel,
		},
	}, nil
}

// Run は音楽生成のコアプロセス（作詞〜音声生成）を一括で行います。
func (a *LyriaAdapter) Run(ctx context.Context, task domain.Task, input *domain.CollectedContent) (*domain.MusicRecipe, []byte, error) {
	lyrics, err := a.GenerateLyrics(ctx, input, task.AIModels.TextModel, task.AIModels.ComposeMode)
	if err != nil {
		return nil, nil, err
	}

	recipe, err := a.Compose(ctx, lyrics, task.AIModels.TextModel, task.AIModels.ComposeMode)
	if err != nil {
		return nil, nil, err
	}

	recipe.AIModels = task.AIModels
	recipe.Lyrics = lyrics

	wav, err := a.GenerateAudio(ctx, recipe, input.Images)
	if err != nil {
		return nil, nil, err
	}

	return recipe, wav, nil
}

func (a *LyriaAdapter) GenerateLyrics(ctx context.Context, input *domain.CollectedContent, model, mode string) (*domain.LyricsDraft, error) {
	return a.lyricist.GenerateLyrics(ctx, input, model, mode)
}

func (a *LyriaAdapter) Compose(ctx context.Context, lyrics *domain.LyricsDraft, model, mode string) (*domain.MusicRecipe, error) {
	return a.composer.Compose(ctx, lyrics, model, mode)
}

func (a *LyriaAdapter) GenerateAudio(ctx context.Context, recipe *domain.MusicRecipe, images []domain.ImagePayload) ([]byte, error) {
	return a.audio.GenerateAudio(ctx, recipe, images)
}

func (a *LyriaAdapter) GenerateFullAudio(ctx context.Context, recipe *domain.MusicRecipe, images []domain.ImagePayload) ([]byte, error) {
	return a.audio.GenerateFullAudio(ctx, recipe, images)
}
