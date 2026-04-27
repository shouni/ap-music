package adapters

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shouni/go-gemini-client/gemini"
	"golang.org/x/time/rate"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

// LyriaAdapter は、歌詞生成・作曲・音声生成を束ねるファサードです。
type LyriaAdapter struct {
	aiClient          gemini.Generator
	promptGen         domain.PromptGenerator
	defaultModel      string
	defaultLyriaModel string
	limiter           *rate.Limiter

	lyricist domain.Lyricist
	composer domain.Composer
	audio    *lyriaAudioGenerator
}

// NewLyriaAdapter は、指定されたコンテキストと構成を使用して、新しい LyriaAdapter を初期化して返します。
func NewLyriaAdapter(ctx context.Context, cfg *config.Config, promptGen domain.PromptGenerator) (*LyriaAdapter, error) {
	if cfg.GeminiAPIKey == "" {
		return nil, errors.New("GeminiAPIKey is required for LyriaAdapter")
	}

	aiClient, err := gemini.NewClient(ctx, gemini.Config{APIKey: cfg.GeminiAPIKey})
	if err != nil {
		return nil, fmt.Errorf("Gemini API クライアントの初期化に失敗しました: %w", err)
	}
	if cfg.GeminiModel == "" {
		return nil, fmt.Errorf("GeminiModel is required but not set")
	}

	adapter := &LyriaAdapter{
		aiClient:          aiClient,
		promptGen:         promptGen,
		defaultModel:      cfg.GeminiModel,
		defaultLyriaModel: cfg.LyriaModel,
		limiter:           rate.NewLimiter(rate.Every(10*time.Second), 1),
	}
	adapter.initComponents()

	return adapter, nil
}

// Run は音楽生成のコアプロセス（作詞〜音声生成）を一括で行います。
func (a *LyriaAdapter) Run(ctx context.Context, task domain.Task, contextText string) (*domain.MusicRecipe, []byte, error) {
	lyrics, err := a.GenerateLyrics(ctx, contextText, task.AIModels.TextModel)
	if err != nil {
		return nil, nil, err
	}

	recipe, err := a.Compose(ctx, lyrics, task.AIModels.TextModel, task.AIModels.ComposeMode)
	if err != nil {
		return nil, nil, err
	}
	recipe.AIModels = task.AIModels

	wav, err := a.GenerateAudio(ctx, recipe)
	if err != nil {
		return nil, nil, err
	}

	return recipe, wav, nil
}

func (a *LyriaAdapter) GenerateLyrics(ctx context.Context, contextText, model string) (*domain.LyricsDraft, error) {
	return a.lyricist.GenerateLyrics(ctx, contextText, model)
}

func (a *LyriaAdapter) Compose(ctx context.Context, lyrics *domain.LyricsDraft, model, mode string) (*domain.MusicRecipe, error) {
	return a.composer.Compose(ctx, lyrics, model, mode)
}

func (a *LyriaAdapter) GenerateAudio(ctx context.Context, recipe *domain.MusicRecipe) ([]byte, error) {
	return a.audio.GenerateAudio(ctx, recipe)
}

func (a *LyriaAdapter) GenerateFullAudio(ctx context.Context, recipe *domain.MusicRecipe) ([]byte, error) {
	return a.audio.GenerateFullAudio(ctx, recipe)
}

func (a *LyriaAdapter) initComponents() {
	a.lyricist = &lyriaLyricist{
		aiClient:     a.aiClient,
		promptGen:    a.promptGen,
		defaultModel: a.defaultModel,
	}
	a.composer = &lyriaComposer{
		aiClient:     a.aiClient,
		promptGen:    a.promptGen,
		defaultModel: a.defaultModel,
	}
	a.audio = &lyriaAudioGenerator{
		aiClient:          a.aiClient,
		defaultLyriaModel: a.defaultLyriaModel,
		limiter:           a.limiter,
		promptBuilder:     lyriaAudioPromptBuilder{},
	}
}
