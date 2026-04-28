package adapters

import (
	"context"
	"errors"
	"fmt"

	"github.com/shouni/go-gemini-client/gemini"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"
	"google.golang.org/genai"

	"ap-music/internal/domain"
)

// lyriaAudioGenerator は MusicRecipe を Lyria に渡し、音声バイナリを生成します。
type lyriaAudioGenerator struct {
	aiClient          gemini.Generator
	promptBuilder     lyriaAudioPromptBuilder
	defaultLyriaModel string
	limiter           *rate.Limiter
	group             singleflight.Group
}

// GenerateAudio は MusicRecipe 全体を 1 回の Lyria 呼び出しで音声化します。
func (g *lyriaAudioGenerator) GenerateAudio(ctx context.Context, recipe *domain.MusicRecipe) ([]byte, error) {
	if recipe == nil {
		return nil, fmt.Errorf("recipe cannot be nil")
	}

	targetModel := g.defaultLyriaModel
	if recipe.AudioModel != "" {
		targetModel = recipe.AudioModel
	}

	promptText := g.promptBuilder.BuildFullSong(recipe)
	responseMIMEType := ""
	key := singleflightKey("audio-full", targetModel, promptText, singleflightSeedKey(recipe.AIModels.Seed), responseMIMEType)
	audio, err := doSingleflight(ctx, &g.group, key, func() ([]byte, error) {
		if err := g.limiter.Wait(ctx); err != nil {
			return nil, err
		}

		resp, err := g.aiClient.GenerateWithParts(
			ctx,
			targetModel,
			[]*genai.Part{{Text: promptText}},
			buildAudioGenerateOptions(recipe.AIModels.Seed, responseMIMEType),
		)
		if err != nil {
			return nil, fmt.Errorf("lyria generation failed (model: %s): %w", targetModel, err)
		}
		if resp == nil || len(resp.Audios) == 0 {
			return nil, fmt.Errorf("no audio data received from Lyria")
		}

		return resp.Audios[0], nil
	})
	if err != nil {
		return nil, err
	}

	return cloneBytes(audio), nil
}

// GenerateFullAudio は MusicRecipe の各セクションを個別に生成し、1 つの WAV に結合します。
func (g *lyriaAudioGenerator) GenerateFullAudio(ctx context.Context, recipe *domain.MusicRecipe) ([]byte, error) {
	if recipe == nil || len(recipe.Sections) == 0 {
		return nil, errors.New("recipe sections are empty")
	}

	wavParts := make([][]byte, len(recipe.Sections))
	group, groupCtx := errgroup.WithContext(ctx)

	for i, sec := range recipe.Sections {
		group.Go(func() error {
			data, err := g.generateAudioSection(groupCtx, recipe, sec)
			if err != nil {
				return fmt.Errorf("section '%s' generation failed: %w", sec.Name, err)
			}
			wavParts[i] = data
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	combinedWav, err := CombineWavData(wavParts)
	if err != nil {
		return nil, fmt.Errorf("failed to combine wav sections: %w", err)
	}

	return combinedWav, nil
}

// generateAudioSection は指定された 1 セクションを Lyria で音声化します。
func (g *lyriaAudioGenerator) generateAudioSection(ctx context.Context, recipe *domain.MusicRecipe, sec domain.MusicSection) ([]byte, error) {
	if recipe == nil {
		return nil, errors.New("recipe is nil")
	}
	if sec.Prompt == "" {
		return nil, fmt.Errorf("section '%s' prompt is empty", sec.Name)
	}

	targetModel := g.defaultLyriaModel
	if recipe.AudioModel != "" {
		targetModel = recipe.AudioModel
	}

	promptText := g.promptBuilder.BuildSection(recipe, sec)
	responseMIMEType := "audio/wav"
	key := singleflightKey("audio-section", targetModel, promptText, singleflightSeedKey(recipe.AIModels.Seed), responseMIMEType)
	audio, err := doSingleflight(ctx, &g.group, key, func() ([]byte, error) {
		if err := g.limiter.Wait(ctx); err != nil {
			return nil, err
		}

		resp, err := g.aiClient.GenerateWithParts(
			ctx,
			targetModel,
			[]*genai.Part{{Text: promptText}},
			buildAudioGenerateOptions(recipe.AIModels.Seed, responseMIMEType),
		)
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.Audios) == 0 {
			return nil, fmt.Errorf("no audio from Lyria for %s", sec.Name)
		}

		return resp.Audios[0], nil
	})
	if err != nil {
		return nil, err
	}

	return cloneBytes(audio), nil
}

// buildAudioGenerateOptions は Lyria 音声生成で共通して使う生成オプションを組み立てます。
func buildAudioGenerateOptions(seed *int64, responseMIMEType string) gemini.GenerateOptions {
	return gemini.GenerateOptions{
		ResponseMIMEType: responseMIMEType,
		Seed:             seed,
		SafetySettings: []*genai.SafetySetting{
			{Category: genai.HarmCategoryHarassment, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryHateSpeech, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategorySexuallyExplicit, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockThresholdBlockNone},
		},
	}
}
