package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/shouni/go-gemini-client/gemini"
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
	wav, err := a.GenerateAudio(ctx, recipe)
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

// GenerateFullAudio は Sub, Main, Ending の3セクションを順番に生成し、結合したバイナリを返します。
func (a *LyriaAdapter) GenerateFullAudio(ctx context.Context, recipe *domain.MusicRecipe) ([]byte, error) {
	type sectionSpec struct {
		name     string
		duration int
	}

	specs := []sectionSpec{
		{"Verse", 40},
		{"Chorus", 45},
		{"Outro", 15},
	}

	var fullAudio []byte
	for _, spec := range specs {
		data, err := a.GenerateAudioSection(ctx, recipe, spec.name, spec.duration)
		if err != nil {
			return nil, err
		}

		// TODO::ここでは単純に結合（※音楽的なクロスフェードは別途検討）
		fullAudio = append(fullAudio, data...)
	}

	return fullAudio, nil
}

// GenerateAudioSection は特定のセクション（Sub/Main/Ending）に特化したプロンプトで音声を生成します。
func (a *LyriaAdapter) GenerateAudioSection(ctx context.Context, recipe *domain.MusicRecipe, sectionName string, duration int) ([]byte, error) {
	if recipe == nil {
		return nil, fmt.Errorf("recipe cannot be nil")
	}

	// --- 1. データの抽出 (元のロジックを継承) ---
	var lyricsText string
	if recipe.Lyrics != nil {
		lyricsText = recipe.Lyrics.Lyrics
	}

	var sectionPrompt string
	if len(recipe.Sections) > 0 {
		sectionPrompt = recipe.Sections[0].Prompt
	}

	// --- 2. strings.Builder を使用した明確なプロンプト構築 ---
	var pb strings.Builder
	pb.WriteString(fmt.Sprintf("Current Section: [%s]. Duration: %d seconds.\n", sectionName, duration))

	// セクションごとに歌唱指示を出し分ける
	switch sectionName {
	case "Verse":
		pb.WriteString("Vocal Direction: Focus on singing the [Verse] section of the lyrics. ")
	case "Chorus":
		pb.WriteString("Vocal Direction: This is the peak! Sing the [Chorus] and the Hook intensely. ")
	case "Outro":
		pb.WriteString("Vocal Direction: Sing the [Outro] part as the song fades out into digital echoes. ")
	}

	// 歌詞全体をコンテキストとして渡す
	pb.WriteString(fmt.Sprintf("\nFull Lyrics to reference:\n%s\n", recipe.Lyrics.Lyrics))

	// 歌詞の流し込み
	if lyricsText != "" {
		pb.WriteString(fmt.Sprintf("With the following lyrics:\n\n%s\n\n", lyricsText))
	}

	// 詳細制約の統合
	pb.WriteString(fmt.Sprintf(
		"Additional constraints: Music Detail: %s. Title: '%s', Theme: '%s', Instruments: %s, Tempo: %d BPM.",
		sectionPrompt,
		recipe.Title,
		recipe.Theme,
		strings.Join(recipe.Instruments, ", "),
		recipe.Tempo,
	))

	fullPrompt := pb.String()

	// --- 3. parts の組み立てとオプション設定 ---
	parts := []*genai.Part{
		{Text: fullPrompt},
	}

	// recipe.Lyrics.Seed など、ベースとなるシード値をここで使用します
	seedValue := int64(12345) // 本来は recipe 内に保持している Seed を使用
	// TODO: 本来は recipe 内に保持している Seed を使用
	//if recipe.Lyrics != nil && recipe.Lyrics.Seed != 0 {
	//	seedValue = recipe.Lyrics.Seed
	//}

	opts := gemini.GenerateOptions{
		ResponseMIMEType: "audio/wav",
		Seed:             &seedValue, // ここでポインタを渡してシードを固定！
		SafetySettings: []*genai.SafetySetting{
			{Category: genai.HarmCategoryHarassment, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryHateSpeech, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategorySexuallyExplicit, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockThresholdBlockNone},
		},
	}

	// --- 4. Lyria API 実行 ---
	targetModel := a.defaultLyriaModel
	if recipe.AudioModel != "" {
		targetModel = recipe.AudioModel
	}

	resp, err := a.aiClient.GenerateWithParts(ctx, targetModel, parts, opts)
	if err != nil {
		return nil, fmt.Errorf("lyria generation failed for section %s: %w", sectionName, err)
	}

	if resp == nil || len(resp.Audios) == 0 {
		return nil, fmt.Errorf("no audio data for section %s", sectionName)
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
