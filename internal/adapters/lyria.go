package adapters

import (
	"context"
	"errors"

	"github.com/shouni/go-gemini-client/gemini"
	"github.com/shouni/go-gemini-client/lyria"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

// LyriaAdapter は domain 型と pkg/lyria 型の変換を担う境界アダプターです。
type LyriaAdapter struct {
	core *lyria.Workflow
}

// LyriaAdapterOption configures the Lyria adapter boundary.
type LyriaAdapterOption = lyria.Option

// NewLyriaAdapter は既存の adapters API を維持し、Lyria 実装を pkg/lyria へ委譲します。
func NewLyriaAdapter(cfg *config.Config, aiClient gemini.Generator, promptGen domain.PromptGenerator, adapterOptions ...LyriaAdapterOption) (*LyriaAdapter, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	options := []lyria.Option{
		lyria.WithGeminiModel(cfg.GeminiModel),
		lyria.WithLyriaModel(cfg.LyriaModel),
		lyria.WithRateInterval(cfg.RateInterval),
		lyria.WithMaxConcurrency(cfg.MaxConcurrency),
		lyria.WithAudioPromptBuilder(NewDefaultLyriaAudioPromptBuilder()),
	}
	options = append(options, adapterOptions...)

	core, err := lyria.New(aiClient, lyriaPromptGenerator{inner: promptGen}, options...)
	if err != nil {
		return nil, err
	}

	return &LyriaAdapter{core: core}, nil
}

func (a *LyriaAdapter) Run(ctx context.Context, task domain.Task, input *domain.CollectedContent) (*domain.MusicRecipe, []byte, error) {
	recipe, audio, err := a.core.Run(ctx, toLyriaAIModels(task.AIModels), toLyriaCollectedContent(input))
	if err != nil {
		return nil, nil, err
	}
	return toDomainMusicRecipe(recipe), audio, nil
}

func (a *LyriaAdapter) GenerateLyrics(ctx context.Context, ai domain.AIModels, input *domain.CollectedContent) (*domain.LyricsDraft, error) {
	lyrics, err := a.core.GenerateLyrics(ctx, toLyriaAIModels(ai), toLyriaCollectedContent(input))
	if err != nil {
		return nil, err
	}
	return toDomainLyricsDraft(lyrics), nil
}

func (a *LyriaAdapter) Compose(ctx context.Context, ai domain.AIModels, lyrics *domain.LyricsDraft) (*domain.MusicRecipe, error) {
	recipe, err := a.core.Compose(ctx, toLyriaAIModels(ai), toLyriaLyricsDraft(lyrics))
	if err != nil {
		return nil, err
	}
	return toDomainMusicRecipe(recipe), nil
}

func (a *LyriaAdapter) GenerateAudio(ctx context.Context, recipe *domain.MusicRecipe, images []domain.ImagePayload) ([]byte, error) {
	return a.core.GenerateAudio(ctx, toLyriaMusicRecipe(recipe), toLyriaImagePayloads(images))
}

func (a *LyriaAdapter) GenerateFullAudio(ctx context.Context, recipe *domain.MusicRecipe, images []domain.ImagePayload) ([]byte, error) {
	return a.core.GenerateFullAudio(ctx, toLyriaMusicRecipe(recipe), toLyriaImagePayloads(images))
}

type lyriaPromptGenerator struct {
	inner domain.PromptGenerator
}

func (g lyriaPromptGenerator) GenerateLyrics(mode string, input string) (string, error) {
	return g.inner.GenerateLyrics(mode, input)
}

func (g lyriaPromptGenerator) GenerateRecipe(mode string, lyrics *lyria.LyricsDraft) (string, error) {
	return g.inner.GenerateRecipe(mode, toDomainLyricsDraft(lyrics))
}

func toLyriaAIModels(ai domain.AIModels) lyria.AIModels {
	return lyria.AIModels{
		TextModel:   ai.TextModel,
		AudioModel:  ai.AudioModel,
		LyricsMode:  ai.LyricsMode,
		ComposeMode: ai.ComposeMode,
		Seed:        cloneInt64Ptr(ai.Seed),
	}
}

func toDomainAIModels(ai lyria.AIModels) domain.AIModels {
	return domain.AIModels{
		TextModel:   ai.TextModel,
		AudioModel:  ai.AudioModel,
		LyricsMode:  ai.LyricsMode,
		ComposeMode: ai.ComposeMode,
		Seed:        cloneInt64Ptr(ai.Seed),
	}
}

func toLyriaCollectedContent(input *domain.CollectedContent) *lyria.CollectedContent {
	if input == nil {
		return nil
	}
	return &lyria.CollectedContent{
		Prompt: input.Prompt,
		Images: toLyriaImagePayloads(input.Images),
	}
}

func toLyriaImagePayloads(images []domain.ImagePayload) []lyria.ImagePayload {
	if images == nil {
		return nil
	}
	out := make([]lyria.ImagePayload, len(images))
	for i, image := range images {
		out[i] = lyria.ImagePayload{
			Data:     append([]byte(nil), image.Data...),
			MIMEType: image.MIMEType,
		}
	}
	return out
}

func toLyriaLyricsDraft(src *domain.LyricsDraft) *lyria.LyricsDraft {
	if src == nil {
		return nil
	}
	return &lyria.LyricsDraft{
		Title:     src.Title,
		Theme:     src.Theme,
		Hook:      src.Hook,
		Lyrics:    src.Lyrics,
		Keywords:  append([]string(nil), src.Keywords...),
		Mood:      src.Mood,
		Narrative: src.Narrative,
	}
}

func toDomainLyricsDraft(src *lyria.LyricsDraft) *domain.LyricsDraft {
	if src == nil {
		return nil
	}
	return &domain.LyricsDraft{
		Title:     src.Title,
		Theme:     src.Theme,
		Hook:      src.Hook,
		Lyrics:    src.Lyrics,
		Keywords:  append([]string(nil), src.Keywords...),
		Mood:      src.Mood,
		Narrative: src.Narrative,
	}
}

func toLyriaMusicRecipe(src *domain.MusicRecipe) *lyria.MusicRecipe {
	if src == nil {
		return nil
	}
	out := &lyria.MusicRecipe{
		Title:       src.Title,
		Theme:       src.Theme,
		Mood:        src.Mood,
		Tempo:       src.Tempo,
		Instruments: append([]string(nil), src.Instruments...),
		Sections:    toLyriaMusicSections(src.Sections),
		Lyrics:      toLyriaLyricsDraft(src.Lyrics),
		AIModels:    toLyriaAIModels(src.AIModels),
	}
	return out
}

func toDomainMusicRecipe(src *lyria.MusicRecipe) *domain.MusicRecipe {
	if src == nil {
		return nil
	}
	return &domain.MusicRecipe{
		Title:       src.Title,
		Theme:       src.Theme,
		Mood:        src.Mood,
		Tempo:       src.Tempo,
		Instruments: append([]string(nil), src.Instruments...),
		Sections:    toDomainMusicSections(src.Sections),
		Lyrics:      toDomainLyricsDraft(src.Lyrics),
		AIModels:    toDomainAIModels(src.AIModels),
	}
}

func toLyriaMusicSections(sections []domain.MusicSection) []lyria.MusicSection {
	if sections == nil {
		return nil
	}
	out := make([]lyria.MusicSection, len(sections))
	for i, section := range sections {
		out[i] = lyria.MusicSection{
			Name:     section.Name,
			Duration: section.Duration,
			Prompt:   section.Prompt,
		}
	}
	return out
}

func toDomainMusicSections(sections []lyria.MusicSection) []domain.MusicSection {
	if sections == nil {
		return nil
	}
	out := make([]domain.MusicSection, len(sections))
	for i, section := range sections {
		out[i] = domain.MusicSection{
			Name:     section.Name,
			Duration: section.Duration,
			Prompt:   section.Prompt,
		}
	}
	return out
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	return new(*value)
}
