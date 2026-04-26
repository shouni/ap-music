package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shouni/go-gemini-client/gemini"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
	"google.golang.org/genai"

	"ap-music/assets"
	"ap-music/internal/config"
	"ap-music/internal/domain"
)

// LyriaAdapter は Lyria API クライアントの実装です。
type LyriaAdapter struct {
	aiClient          gemini.Generator
	promptGen         domain.PromptGenerator
	defaultModel      string // 作詞・作曲(LLM)用デフォルト
	defaultLyriaModel string // 音声生成(Lyria)用デフォルト
	limiter           *rate.Limiter
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

	limiter := rate.NewLimiter(rate.Every(10*time.Second), 1)

	return &LyriaAdapter{
		aiClient:          aiClient,
		promptGen:         promptGen,
		defaultModel:      cfg.GeminiModel,
		defaultLyriaModel: cfg.LyriaModel,
		limiter:           limiter,
	}, nil
}

// Run は音楽生成のコアプロセス（作詞〜音声生成）を一括で行います。
func (a *LyriaAdapter) Run(ctx context.Context, task domain.Task, contextText string) (*domain.MusicRecipe, []byte, error) {
	// Step 1: 作詞
	lyrics, err := a.GenerateLyrics(ctx, contextText, task.AIModels.TextModel)
	if err != nil {
		return nil, nil, err
	}

	// Step 2: 作曲 (レシピ)
	recipe, err := a.Compose(ctx, lyrics, task.AIModels.TextModel, task.AIModels.ComposeMode)
	if err != nil {
		return nil, nil, err
	}
	recipe.AIModels = task.AIModels

	// Step 3: 音声生成
	wav, err := a.GenerateFullAudio(ctx, recipe)
	if err != nil {
		return nil, nil, err
	}

	return recipe, wav, nil
}

// GenerateLyrics は入力から歌詞のドラフトを生成します。
func (a *LyriaAdapter) GenerateLyrics(ctx context.Context, contextText, model string) (*domain.LyricsDraft, error) {
	if contextText == "" {
		return nil, fmt.Errorf("empty input")
	}

	promptText, err := a.promptGen.GenerateLyrics(contextText)
	if err != nil {
		return nil, fmt.Errorf("failed to build lyrics prompt: %w", err)
	}

	targetModel := a.defaultModel
	if model != "" {
		targetModel = model
	}
	resp, err := a.aiClient.GenerateContent(ctx, targetModel, promptText)
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
}

// Compose は歌詞案をもとに MusicRecipe を生成します。
func (a *LyriaAdapter) Compose(ctx context.Context, lyrics *domain.LyricsDraft, model, mode string) (*domain.MusicRecipe, error) {
	if lyrics == nil {
		return nil, fmt.Errorf("lyrics cannot be nil")
	}
	// 1. プロンプトモードの決定。空の場合は assets のデフォルトを使用
	targetMode := mode
	if targetMode == "" {
		targetMode = assets.ModeCompose
	}

	// 2. 指定されたモードでプロンプトを構築
	promptText, err := a.promptGen.GenerateRecipe(targetMode, lyrics)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt (mode: %s): %w", targetMode, err)
	}

	// 3. モデルの決定
	targetModel := a.defaultModel
	if model != "" {
		targetModel = model
	}

	// 4. LLM 実行
	resp, err := a.aiClient.GenerateContent(ctx, targetModel, promptText)
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

	// 5. JSON クリーニングとパース
	jsonStr := cleanJSONResponse(rawRecipe)
	var recipe domain.MusicRecipe
	if err := json.Unmarshal([]byte(jsonStr), &recipe); err != nil {
		return nil, fmt.Errorf("failed to unmarshal recipe json: %w (raw: %s)", err, jsonStr)
	}

	// 6. 後続の処理のために情報を保持
	recipe.Lyrics = lyrics
	return &recipe, nil
}

// GenerateAudio は Lyria 3 モデルを使用して WAV バイナリを生成します。
func (a *LyriaAdapter) GenerateAudio(ctx context.Context, recipe *domain.MusicRecipe) ([]byte, error) {
	if recipe == nil {
		return nil, fmt.Errorf("recipe cannot be nil")
	}
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
	var pb strings.Builder
	pb.WriteString(fmt.Sprintf("Create a %s song.\n\n", recipe.Mood))

	if lyricsText != "" {
		pb.WriteString(fmt.Sprintf("With the following lyrics:\n\n%s\n\n", lyricsText))
	}

	pb.WriteString(fmt.Sprintf(
		"Additional constraints: Music Detail: %s. Title: '%s', Theme: '%s', Instruments: %s, Tempo: %d BPM.",
		sectionPrompt,
		recipe.Title,
		recipe.Theme,
		strings.Join(recipe.Instruments, ", "),
		recipe.Tempo,
	))

	fullPrompt := pb.String()

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
	if recipe.AudioModel != "" {
		targetModel = recipe.AudioModel
	}
	resp, err := a.aiClient.GenerateWithParts(ctx, targetModel, parts, opts)
	if err != nil {
		return nil, fmt.Errorf("lyria generation failed (model: %s): %w", targetModel, err)
	}

	// 6. 音声データの抽出
	if resp == nil || len(resp.Audios) == 0 {
		return nil, fmt.Errorf("no audio data (WAV) received from Lyria (model: %s)", targetModel)
	}

	return resp.Audios[0], nil
}

// GenerateFullAudio は recipe.Sections に基づいて並行して音声を生成し、最終的に結合します。
func (a *LyriaAdapter) GenerateFullAudio(ctx context.Context, recipe *domain.MusicRecipe) ([]byte, error) {
	if recipe == nil || len(recipe.Sections) == 0 {
		return nil, errors.New("recipe sections are empty")
	}

	wavParts := make([][]byte, len(recipe.Sections))
	g, gCtx := errgroup.WithContext(ctx)

	for i, sec := range recipe.Sections {
		g.Go(func() error {
			if err := a.limiter.Wait(gCtx); err != nil {
				return err
			}

			data, err := a.generateAudioSection(gCtx, recipe, sec)
			if err != nil {
				return fmt.Errorf("section '%s' generation failed: %w", sec.Name, err)
			}
			wavParts[i] = data
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	combinedWav, err := CombineWavData(wavParts)
	if err != nil {
		return nil, fmt.Errorf("failed to combine wav sections: %w", err)
	}

	return combinedWav, nil
}

// GenerateAudioSection は特定セクションのプロンプトを構築して生成を実行します。
func (a *LyriaAdapter) generateAudioSection(ctx context.Context, recipe *domain.MusicRecipe, sec domain.MusicSection) ([]byte, error) {
	if recipe == nil {
		return nil, errors.New("recipe is nil")
	}

	// 1. セクションの存在確認とプロンプトのバリデーション
	sectionName := sec.Name
	duration := sec.Duration
	sectionPrompt := sec.Prompt

	if sectionPrompt == "" {
		return nil, fmt.Errorf("section '%s' prompt is empty", sectionName)
	}

	// 2. データの抽出
	var lyricsText string
	if recipe.Lyrics != nil {
		lyricsText = recipe.Lyrics.Lyrics
	}

	// 3. プロンプト構築（詳細指示を文末に配置して強調）
	var pb strings.Builder
	pb.WriteString(fmt.Sprintf("Current Section: [%s]. Duration: %d seconds.\n", sectionName, duration))

	switch sectionName {
	case "Verse":
		pb.WriteString("Vocal Direction: Focus on singing the [Verse] section. Build tension. ")
	case "Chorus":
		pb.WriteString("Vocal Direction: High energy! Sing the [Chorus] and Hook powerfully. ")
	case "Outro":
		pb.WriteString("Vocal Direction: Emotional fade-out with [Outro] lyrics. ")
	}

	if lyricsText != "" {
		pb.WriteString(fmt.Sprintf("\nFull Lyrics to reference:\n%s\n", lyricsText))
	}

	pb.WriteString(fmt.Sprintf(
		"\n[Audio Generation Constraints]\n- Title: '%s'\n- Instruments: %s\n- Tempo: %d BPM\n- Music Detail: %s",
		recipe.Title,
		strings.Join(recipe.Instruments, ", "),
		recipe.Tempo,
		sectionPrompt,
	))

	// 4. API オプションの設定
	opts := gemini.GenerateOptions{
		ResponseMIMEType: "audio/wav",
		Seed:             recipe.AIModels.Seed,
		SafetySettings: []*genai.SafetySetting{
			{Category: genai.HarmCategoryHarassment, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryHateSpeech, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategorySexuallyExplicit, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockThresholdBlockNone},
		},
	}

	targetModel := a.defaultLyriaModel
	if recipe.AudioModel != "" {
		targetModel = recipe.AudioModel
	}

	// 5. Lyria API 実行
	resp, err := a.aiClient.GenerateWithParts(ctx, targetModel, []*genai.Part{{Text: pb.String()}}, opts)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Audios) == 0 {
		return nil, fmt.Errorf("no audio from Lyria for %s", sectionName)
	}

	return resp.Audios[0], nil
}

// cleanJSONResponse は LLM が出力しがちな Markdown の装飾を除去します。
func cleanJSONResponse(input string) string {
	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")
	if start == -1 || end == -1 || start > end {
		// インデックスが不正な場合はそのまま返し、後続の Unmarshal でエラーをハンドリングさせる
		return input
	}
	return input[start : end+1]
}
