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

// Compose は入力から音楽の構成（レシピ）を構築します。
func (a *LyriaAdapter) Compose(ctx context.Context, input string) (domain.MusicRecipe, error) {
	if input == "" {
		return domain.MusicRecipe{}, fmt.Errorf("empty input")
	}

	// 1. プロンプトの組み立て
	promptText, err := a.promptGen.GenerateRecipe("recipe", input)
	if err != nil {
		return domain.MusicRecipe{}, fmt.Errorf("failed to build prompt: %w", err)
	}

	// 2. 構築したプロンプトを実際にAI（Gemini）に投げる
	// TODO: ライブラリ側の引数を修正し、JSONモード(ResponseMIMEType: "application/json")を強制するように変更する
	resp, err := a.aiClient.GenerateContent(ctx, a.model, promptText)
	if err != nil {
		return domain.MusicRecipe{}, fmt.Errorf("AI generation failed (model: %s): %w", a.model, err)
	}

	// 3. レスポンスの存在確認とAIの回答（JSON文字列）を取得
	// nil パニックを防止し、エラー原因を特定しやすくするためのチェック
	if resp == nil {
		return domain.MusicRecipe{}, fmt.Errorf("AI response is nil")
	}

	rawRecipe := strings.TrimSpace(resp.Text)
	if rawRecipe == "" {
		return domain.MusicRecipe{}, fmt.Errorf("AI returned an empty string for the recipe")
	}

	// 4. Markdown の除去（一応残しておくが、JSONモードなら不要な場合も多い）
	jsonStr := cleanJSONResponse(rawRecipe)

	// 5. JSON をデコード
	var recipe domain.MusicRecipe
	if err := json.Unmarshal([]byte(jsonStr), &recipe); err != nil {
		return domain.MusicRecipe{}, fmt.Errorf("failed to unmarshal recipe json: %w (raw: %s)", err, jsonStr)
	}

	return recipe, nil
}

// Generate は Lyria 3 モデルを使用して WAV 形式の音声データを生成します。
func (a *LyriaAdapter) Generate(ctx context.Context, recipe domain.MusicRecipe) ([]byte, error) {
	// 1. プロンプトの高度な構築（優先順位を明確化）
	var sectionPrompt string
	if len(recipe.Sections) > 0 {
		sectionPrompt = recipe.Sections[0].Prompt
	}

	// 詳細な指示（Prompt）を核にし、メタデータを補足として付与する
	fullPrompt := fmt.Sprintf(
		"Music Detail: %s. Title: %s. Theme: %s. Instruments: %s. Mood: %s, Tempo: %d BPM.",
		sectionPrompt,
		recipe.Title,
		recipe.Theme,
		strings.Join(recipe.Instruments, ", "),
		recipe.Mood,
		recipe.Tempo,
	)

	// 2. parts の組み立て
	parts := []*genai.Part{
		{Text: fullPrompt},
	}

	// 3. 生成オプションの設定（セーフティガードの緩和を追加）
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
	resp, err := a.aiClient.GenerateWithParts(ctx, a.lyriaModel, parts, opts)
	if err != nil {
		return nil, fmt.Errorf("lyria generation failed (model: %s): %w", a.lyriaModel, err)
	}

	// 6. 音声データの抽出
	if resp == nil || len(resp.Audios) == 0 {
		return nil, fmt.Errorf("no audio data (WAV) received from Lyria")
	}

	return resp.Audios[0], nil
}

// cleanJSONResponse は LLM が出力しがちな Markdown の装飾を除去します
func cleanJSONResponse(input string) string {
	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")
	if start != -1 && end != -1 && start < end {
		return input[start : end+1]
	}
	return input
}
