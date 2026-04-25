package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/shouni/go-gemini-client/gemini"
	"google.golang.org/genai"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

// LyriaAdapter は Lyria API クライアントの実装です。
type LyriaAdapter struct {
	aiClient          gemini.Generator
	promptGen         domain.PromptGenerator
	defaultModel      string // 作詞・作曲(LLM)用デフォルト
	defaultLyriaModel string // 音声生成(Lyria)用デフォルト
}

// NewLyriaAdapter は、指定されたコンテキストと構成を使用して、新しい LyriaAdapter を初期化して返します。
func NewLyriaAdapter(ctx context.Context, cfg *config.Config, promptGen domain.PromptGenerator) (*LyriaAdapter, error) {
	if cfg.GeminiAPIKey == "" {
		return nil, errors.New("GeminiAPIKey is required for LyriaAdapter")
	}
	clientConfig := gemini.Config{
		APIKey: cfg.GeminiAPIKey,
	}

	aiClient, err := gemini.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("Gemini API クライアントの初期化に失敗しました: %w", err)
	}

	if cfg.GeminiModel == "" {
		return nil, fmt.Errorf("GeminiModel is required but not set")
	}

	return &LyriaAdapter{
		aiClient:          aiClient,
		promptGen:         promptGen,
		defaultModel:      cfg.GeminiModel,
		defaultLyriaModel: cfg.LyriaModel,
	}, nil
}

// GenerateLyrics は入力から歌詞のドラフトを生成します。
func (a *LyriaAdapter) GenerateLyrics(ctx context.Context, input string) (domain.LyricsDraft, error) {
	if input == "" {
		return domain.LyricsDraft{}, fmt.Errorf("empty input")
	}

	promptText, err := a.promptGen.GenerateLyrics(input)
	if err != nil {
		return domain.LyricsDraft{}, fmt.Errorf("failed to build lyrics prompt: %w", err)
	}

	resp, err := a.aiClient.GenerateContent(ctx, a.defaultModel, promptText)
	if err != nil {
		return domain.LyricsDraft{}, fmt.Errorf("lyrics generation failed (model: %s): %w", a.defaultModel, err)
	}
	if resp == nil {
		return domain.LyricsDraft{}, fmt.Errorf("lyrics response is nil")
	}

	rawLyrics := strings.TrimSpace(resp.Text)
	if rawLyrics == "" {
		return domain.LyricsDraft{}, fmt.Errorf("AI returned an empty string for the lyrics")
	}

	var lyrics domain.LyricsDraft
	jsonStr := cleanJSONResponse(rawLyrics)
	if err := json.Unmarshal([]byte(jsonStr), &lyrics); err != nil {
		return domain.LyricsDraft{}, fmt.Errorf("failed to unmarshal lyrics json: %w (raw: %s)", err, jsonStr)
	}
	if strings.TrimSpace(lyrics.Lyrics) == "" {
		return domain.LyricsDraft{}, fmt.Errorf("lyrics draft is empty")
	}

	return lyrics, nil
}

// Compose は歌詞案をもとに MusicRecipe を生成します。
func (a *LyriaAdapter) Compose(ctx context.Context, lyrics domain.LyricsDraft) (domain.MusicRecipe, error) {
	promptText, err := a.promptGen.GenerateRecipe(lyrics)
	if err != nil {
		return domain.MusicRecipe{}, fmt.Errorf("failed to build prompt: %w", err)
	}

	resp, err := a.aiClient.GenerateContent(ctx, a.defaultModel, promptText)
	if err != nil {
		return domain.MusicRecipe{}, fmt.Errorf("AI generation failed (model: %s): %w", a.defaultModel, err)
	}

	if resp == nil {
		return domain.MusicRecipe{}, fmt.Errorf("AI response is nil")
	}

	rawRecipe := strings.TrimSpace(resp.Text)
	if rawRecipe == "" {
		return domain.MusicRecipe{}, fmt.Errorf("AI returned an empty string for the recipe")
	}

	jsonStr := cleanJSONResponse(rawRecipe)
	var recipe domain.MusicRecipe
	if err := json.Unmarshal([]byte(jsonStr), &recipe); err != nil {
		return domain.MusicRecipe{}, fmt.Errorf("failed to unmarshal recipe json: %w (raw: %s)", err, jsonStr)
	}

	recipe.Lyrics = &lyrics
	return recipe, nil
}

// GenerateAudio は Lyria 3 モデルを使用して WAV バイナリを生成します。
func (a *LyriaAdapter) GenerateAudio(ctx context.Context, recipe domain.MusicRecipe) ([]byte, error) {
	// 1. データの抽出
	var lyricsText string
	if recipe.Lyrics != nil {
		lyricsText = recipe.Lyrics.Lyrics
	}

	var sectionPrompt string
	if len(recipe.Sections) > 0 {
		sectionPrompt = recipe.Sections[0].Prompt
	}

	// 2. strings.Builder を使用した明確なプロンプト構築
	var promptBuilder strings.Builder
	promptBuilder.WriteString(fmt.Sprintf("Create a %s song.\n\n", recipe.Mood))

	if lyricsText != "" {
		promptBuilder.WriteString(fmt.Sprintf("With the following lyrics:\n\n%s\n\n", lyricsText))
	}

	promptBuilder.WriteString(fmt.Sprintf(
		"Additional constraints: Music Detail: %s. Title: '%s', Theme: '%s', Instruments: %s, Tempo: %d BPM.",
		sectionPrompt,
		recipe.Title,
		recipe.Theme,
		strings.Join(recipe.Instruments, ", "),
		recipe.Tempo,
	))

	fullPrompt := promptBuilder.String()

	// 3. parts の組み立て
	parts := []*genai.Part{
		{Text: fullPrompt},
	}

	// 4. 生成オプションの設定
	opts := gemini.GenerateOptions{
		ResponseMIMEType: "audio/wav",
		SafetySettings: []*genai.SafetySetting{
			{Category: genai.HarmCategoryHarassment, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryHateSpeech, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategorySexuallyExplicit, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockThresholdBlockNone},
		},
	}

	// 5. Lyria API を実行
	targetModel := a.defaultLyriaModel
	if recipe.ComposeModel != "" {
		targetModel = recipe.ComposeModel
	}
	resp, err := a.aiClient.GenerateWithParts(ctx, targetModel, parts, opts)
	if err != nil {
		return nil, fmt.Errorf("lyria generation failed (model: %s): %w", a.defaultLyriaModel, err)
	}

	// 6. 音声データの抽出
	if resp == nil || len(resp.Audios) == 0 {
		return nil, fmt.Errorf("no audio data (WAV) received from Lyria (model: %s)", a.defaultLyriaModel)
	}

	return resp.Audios[0], nil
}

// cleanJSONResponse は LLM が出力しがちな Markdown の装飾を除去します。
func cleanJSONResponse(input string) string {
	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")
	if start == -1 || end == -1 || start > end {
		return input
	}
	return input[start : end+1]
}
