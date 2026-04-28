package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shouni/go-gemini-client/gemini"
	"golang.org/x/sync/singleflight"

	"ap-music/assets"
	"ap-music/internal/domain"
)

type lyriaComposer struct {
	aiClient     gemini.ContentGenerator
	promptGen    domain.PromptGenerator
	defaultModel string
	group        singleflight.Group
}

func (g *lyriaComposer) Compose(ctx context.Context, lyrics *domain.LyricsDraft, model, mode string) (*domain.MusicRecipe, error) {
	if lyrics == nil {
		return nil, fmt.Errorf("lyrics cannot be nil")
	}

	targetMode := mode
	if targetMode == "" {
		targetMode = assets.ModeCompose
	}

	promptText, err := g.promptGen.GenerateRecipe(targetMode, lyrics)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt (mode: %s): %w", targetMode, err)
	}

	targetModel := g.defaultModel
	if model != "" {
		targetModel = model
	}

	key := singleflightKey("compose", targetModel, promptText)
	recipe, err := doSingleflight(ctx, &g.group, key, func() (*domain.MusicRecipe, error) {
		resp, err := g.aiClient.GenerateContent(ctx, targetModel, promptText)
		if err != nil {
			return nil, fmt.Errorf("AI generation failed (model: %s): %w", targetModel, err)
		}
		if resp == nil {
			return nil, fmt.Errorf("AI response is nil")
		}

		rawRecipe := strings.TrimSpace(resp.Text)
		if rawRecipe == "" {
			return nil, fmt.Errorf("AI returned an empty string for the recipe")
		}

		jsonStr := cleanJSONResponse(rawRecipe)
		var recipe domain.MusicRecipe
		if err := json.Unmarshal([]byte(jsonStr), &recipe); err != nil {
			return nil, fmt.Errorf("failed to unmarshal recipe json: %w (raw: %s)", err, jsonStr)
		}

		recipe.Lyrics = lyrics
		return &recipe, nil
	})
	if err != nil {
		return nil, err
	}

	return cloneMusicRecipe(recipe), nil
}
