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
	aiClient   gemini.Generator
	promptGen  domain.PromptGenerator
	model      string
	lyriaModel string
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
		aiClient:   aiClient,
		promptGen:  promptGen,
		model:      cfg.GeminiModel,
		lyriaModel: cfg.LyriaModel,
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

	// TODO: APIクライアントを改修し、ResponseMIMEType: "application/json" を指定してJSON出力を強制する
	resp, err := a.aiClient.GenerateContent(ctx, a.model, promptText)
	if err != nil {
		return domain.LyricsDraft{}, fmt.Errorf("lyrics generation failed (model: %s): %w", a.model, err)
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

// ComposeRecipe は歌詞案をもとに MusicRecipe を生成します。
func (a *LyriaAdapter) ComposeRecipe(ctx context.Context, lyrics domain.LyricsDraft) (domain.MusicRecipe, error) {
	promptText, err := a.promptGen.GenerateRecipe(lyrics)
	if err != nil {
		return domain.MusicRecipe{}, fmt.Errorf("failed to build prompt: %w", err)
	}

	resp, err := a.aiClient.GenerateContent(ctx, a.model, promptText)
	if err != nil {
		return domain.MusicRecipe{}, fmt.Errorf("AI generation failed (model: %s): %w", a.model, err)
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

// Generate は Lyria 3 モデルを使用して、ドキュメントの推奨形式で音楽を生成します。
func (a *LyriaAdapter) Generate(ctx context.Context, recipe domain.MusicRecipe) ([]byte, error) {
	// 1. 歌詞の抽出
	var lyricsText string
	if recipe.Lyrics != nil {
		lyricsText = recipe.Lyrics.Lyrics
	}

	// 2. セクション指示の抽出（音楽的な詳細）
	var sectionPrompt string
	if len(recipe.Sections) > 0 {
		sectionPrompt = recipe.Sections[0].Prompt
	}

	// 3. プロンプトの構築
	// [Verse], [Chorus] タグを含む歌詞を核とし、具体的な音楽的制約を付加します。
	var lyricsSection string
	if lyricsText != "" {
		lyricsSection = fmt.Sprintf(" with the following lyrics:\n\n%s\n\n", lyricsText)
	} else {
		lyricsSection = ".\n\n"
	}

	fullPrompt := fmt.Sprintf(
		"Create a %s song%s"+
			"Additional constraints: Music Detail: %s. Title: '%s', Theme: '%s', Instruments: %s, Tempo: %d BPM.",
		recipe.Mood,
		lyricsSection,
		sectionPrompt,
		recipe.Title,
		recipe.Theme,
		strings.Join(recipe.Instruments, ", "),
		recipe.Tempo,
	)

	// 4. parts の組み立て
	parts := []*genai.Part{
		{Text: fullPrompt},
	}

	// 5. 生成オプションの設定（安全性の緩和を含む）
	opts := gemini.GenerateOptions{
		ResponseMIMEType: "audio/wav",
		SafetySettings: []*genai.SafetySetting{
			{Category: genai.HarmCategoryHarassment, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryHateSpeech, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategorySexuallyExplicit, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockThresholdBlockNone},
		},
	}

	// 6. Lyria API を実行
	resp, err := a.aiClient.GenerateWithParts(ctx, a.lyriaModel, parts, opts)
	if err != nil {
		return nil, fmt.Errorf("lyria generation failed (model: %s): %w", a.lyriaModel, err)
	}

	// 7. 音声データの抽出
	if resp == nil || len(resp.Audios) == 0 {
		return nil, fmt.Errorf("no audio data (WAV) received from Lyria")
	}

	return resp.Audios[0], nil
}

// cleanJSONResponse は LLM が出力しがちな Markdown の装飾を除去します。
func cleanJSONResponse(input string) string {
	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")
	if start != -1 && end != -1 && start < end {
		return input[start : end+1]
	}
	return input
}
