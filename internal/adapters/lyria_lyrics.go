package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shouni/go-gemini-client/gemini"
	"golang.org/x/sync/singleflight"

	"ap-music/internal/domain"
)

type lyriaLyricist struct {
	aiClient     gemini.ContentGenerator
	promptGen    domain.PromptGenerator
	defaultModel string
	group        singleflight.Group
}

func (g *lyriaLyricist) GenerateLyrics(ctx context.Context, contextText, model string) (*domain.LyricsDraft, error) {
	if contextText == "" {
		return nil, fmt.Errorf("empty input")
	}

	promptText, err := g.promptGen.GenerateLyrics(contextText)
	if err != nil {
		return nil, fmt.Errorf("failed to build lyrics prompt: %w", err)
	}

	targetModel := g.defaultModel
	if model != "" {
		targetModel = model
	}

	key := singleflightKey("lyrics", targetModel, promptText)
	lyrics, err := doSingleflight(ctx, &g.group, key, func(execCtx context.Context) (*domain.LyricsDraft, error) {
		resp, err := g.aiClient.GenerateContent(execCtx, targetModel, promptText)
		if err != nil {
			return nil, fmt.Errorf("lyrics generation failed (model: %s): %w", targetModel, err)
		}
		if resp == nil {
			return nil, fmt.Errorf("lyrics response is nil")
		}

		rawLyrics := strings.TrimSpace(resp.Text)
		if rawLyrics == "" {
			return nil, fmt.Errorf("AI returned an empty string for the lyrics")
		}

		var lyrics domain.LyricsDraft
		jsonStr := cleanJSONResponse(rawLyrics)
		if err := json.Unmarshal([]byte(jsonStr), &lyrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lyrics json: %w (raw: %s)", err, jsonStr)
		}
		if strings.TrimSpace(lyrics.Lyrics) == "" {
			return nil, fmt.Errorf("lyrics draft is empty")
		}

		return &lyrics, nil
	})
	if err != nil {
		return nil, err
	}

	return cloneLyricsDraft(lyrics), nil
}
