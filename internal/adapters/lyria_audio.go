package adapters

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/shouni/go-gemini-client/gemini"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
	"google.golang.org/genai"

	"ap-music/internal/domain"
)

type lyriaAudioGenerator struct {
	aiClient          gemini.Generator
	defaultLyriaModel string
	limiter           *rate.Limiter
	promptBuilder     lyriaAudioPromptBuilder
}

type lyriaAudioPromptBuilder struct{}

func (g *lyriaAudioGenerator) GenerateAudio(ctx context.Context, recipe *domain.MusicRecipe) ([]byte, error) {
	if recipe == nil {
		return nil, fmt.Errorf("recipe cannot be nil")
	}

	targetModel := g.defaultLyriaModel
	if recipe.AudioModel != "" {
		targetModel = recipe.AudioModel
	}

	if err := g.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	resp, err := g.aiClient.GenerateWithParts(ctx, targetModel, []*genai.Part{{Text: g.promptBuilder.BuildFullSong(recipe)}}, buildAudioGenerateOptions(recipe.AIModels.Seed, ""))
	if err != nil {
		return nil, fmt.Errorf("lyria generation failed (model: %s): %w", targetModel, err)
	}
	if resp == nil || len(resp.Audios) == 0 {
		return nil, fmt.Errorf("no audio data received from Lyria")
	}

	return resp.Audios[0], nil
}

func (g *lyriaAudioGenerator) GenerateFullAudio(ctx context.Context, recipe *domain.MusicRecipe) ([]byte, error) {
	if recipe == nil || len(recipe.Sections) == 0 {
		return nil, errors.New("recipe sections are empty")
	}

	wavParts := make([][]byte, len(recipe.Sections))
	group, groupCtx := errgroup.WithContext(ctx)

	for i, sec := range recipe.Sections {
		i, sec := i, sec
		group.Go(func() error {
			if err := g.limiter.Wait(groupCtx); err != nil {
				return err
			}

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

	resp, err := g.aiClient.GenerateWithParts(
		ctx,
		targetModel,
		[]*genai.Part{{Text: g.promptBuilder.BuildSection(recipe, sec)}},
		buildAudioGenerateOptions(recipe.AIModels.Seed, "audio/wav"),
	)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Audios) == 0 {
		return nil, fmt.Errorf("no audio from Lyria for %s", sec.Name)
	}

	return resp.Audios[0], nil
}

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

func (lyriaAudioPromptBuilder) BuildFullSong(recipe *domain.MusicRecipe) string {
	var pb strings.Builder
	pb.WriteString(fmt.Sprintf("Task: Generate a full high-fidelity song titled '%s'.\n", recipe.Title))
	pb.WriteString(fmt.Sprintf("Style & Mood: %s\n", recipe.Mood))
	pb.WriteString(fmt.Sprintf("Tempo: %d BPM. Instruments: %s.\n\n", recipe.Tempo, strings.Join(recipe.Instruments, ", ")))

	if recipe.Lyrics != nil && recipe.Lyrics.Lyrics != "" {
		pb.WriteString("Lyrics (Perform with clear Japanese vocals and passionate enunciation):\n")
		pb.WriteString(recipe.Lyrics.Lyrics)
		pb.WriteString("\n\n")
	}

	if len(recipe.Sections) > 0 {
		pb.WriteString("Detailed Song Structure & Multi-Stage Vocal Directions:\n")
		for _, sec := range recipe.Sections {
			var direction string
			switch sec.Name {
			case "Verse":
				direction = "Vocal Strategy: Focus on singing the [Verse] section. Start narratively and build tension for the next phase."
			case "Chorus":
				direction = "Vocal Strategy: Max energy! Perform the [Chorus] and Hook with high-octane passion. Keep the heat throughout this long climax."
			case "Outro":
				direction = "Vocal Strategy: Emotional digital fade-out for the [Outro]. Let the Japanese vocals dissolve into a cybernetic echo."
			default:
				direction = fmt.Sprintf("Vocal Strategy: Adapt your energy to sustain this %d-second section with consistent Japanese vocal quality.", sec.Duration)
			}
			pb.WriteString(fmt.Sprintf("- [%s] (%d sec): %s %s\n", sec.Name, sec.Duration, direction, sec.Prompt))
		}
		pb.WriteString("\n")
	}

	pb.WriteString("[Final Executive Constraints]\n")
	pb.WriteString("- Total Duration: Exactly 180 seconds. Do not end early.\n")
	pb.WriteString("- Seamless Flow: Ensure each long section evolves naturally into the next without any energy drops.\n")
	pb.WriteString("- Zero Silence: Maintain a continuous, high-fidelity sonic wall. No gaps or unintentional pauses.\n")
	pb.WriteString("- Vocal Purity: Clear, passionate Japanese vocals throughout. Absolute priority on lyrical clarity.")

	return pb.String()
}

func (lyriaAudioPromptBuilder) BuildSection(recipe *domain.MusicRecipe, sec domain.MusicSection) string {
	var lyricsText string
	if recipe.Lyrics != nil {
		lyricsText = recipe.Lyrics.Lyrics
	}

	var pb strings.Builder
	pb.WriteString(fmt.Sprintf("Current Section: [%s]. Duration: %d seconds.\n", sec.Name, sec.Duration))

	switch sec.Name {
	case "Verse":
		pb.WriteString("Vocal Direction: Focus on singing the [Verse] section. Build tension for the next phase. ")
	case "Chorus":
		pb.WriteString("Vocal Direction: Max energy! Perform the [Chorus] and Hook with high-octane passion. ")
	case "Outro":
		pb.WriteString("Vocal Direction: Emotional digital fade-out for the [Outro]. Leave a cybernetic echo. ")
	default:
		pb.WriteString(fmt.Sprintf("Vocal Direction: Perform the [%s] section with clear Japanese vocals and appropriate energy for the track. ", sec.Name))
	}

	pb.WriteString("Clear Japanese vocals with passionate enunciation. No silence.")

	if lyricsText != "" {
		pb.WriteString(fmt.Sprintf("\nFull Lyrics to reference:\n%s\n", lyricsText))
	}

	pb.WriteString(fmt.Sprintf(
		"\n[Audio Generation Constraints]\n- Title: '%s'\n- Instruments: %s\n- Tempo: %d BPM\n- Music Detail: %s",
		recipe.Title,
		strings.Join(recipe.Instruments, ", "),
		recipe.Tempo,
		sec.Prompt,
	))

	return pb.String()
}
