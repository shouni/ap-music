package adapters

import (
	"context"
	"fmt"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

// LyriaAdapter は Lyria API クライアントの雛形です。
type LyriaAdapter struct {
	Model string
}

func NewLyriaAdapter(ctx context.Context, cfg *config.Config) *LyriaAdapter {
	_ = ctx
	return &LyriaAdapter{
		Model: cfg.LyriaModel,
	}
}

// Compose は簡易レシピを返します。
func (a LyriaAdapter) Compose(_ context.Context, input string) (domain.MusicRecipe, error) {
	if input == "" {
		return domain.MusicRecipe{}, fmt.Errorf("empty input")
	}
	return domain.MusicRecipe{
		Title: "Generated Track",
		Theme: "Auto-composed",
		Mood:  "Neutral",
		Tempo: 120,
		Sections: []domain.MusicSection{{
			Name:     "Intro",
			Duration: 15,
			Prompt:   "Build atmosphere",
		}},
	}, nil
}

// Generate はダミーのMP3バイト列を返します。
func (a LyriaAdapter) Generate(_ context.Context, _ domain.MusicRecipe) ([]byte, error) {
	_ = a.Model
	return []byte("FAKE_MP3_DATA"), nil
}
