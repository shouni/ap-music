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

const (
	defaultVertexLocationID = "global"
)

// LyriaAdapter は Lyria API クライアントの実装です。
type LyriaAdapter struct {
	aiClient  gemini.Generator
	promptGen domain.PromptGenerator
	model     string
}

// NewLyriaAdapter initializes and returns a new LyriaAdapter using the provided context and configuration.
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

	return &LyriaAdapter{
		aiClient:  aiClient,
		promptGen: promptGen,
		model:     cfg.LyriaModel,
	}, nil
}

// Compose は入力から音楽の構成（レシピ）を構築します。
func (a *LyriaAdapter) Compose(ctx context.Context, input string) (domain.MusicRecipe, error) {
	if input == "" {
		return domain.MusicRecipe{}, fmt.Errorf("empty input")
	}

	// 1. PromptAdapter を使って LLM から JSON 文字列（Markdown 含む）を取得
	rawRecipe, err := a.promptGen.GenerateRecipe("recipe", input)
	if err != nil {
		return domain.MusicRecipe{}, fmt.Errorf("failed to generate recipe: %w", err)
	}

	// 2. Markdown のコードブロックを除去して純粋な JSON を抽出
	jsonStr := cleanJSONResponse(rawRecipe)

	// 3. JSON を domain.MusicRecipe 構造体にデコード
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
			{Category: genai.HarmCategoryCivicIntegrity, Threshold: genai.HarmBlockThresholdBlockNone},
		},
	}

	// 4. モデル選択の堅牢化
	model := a.model
	if recipe.Metadata != nil {
		if selected := strings.TrimSpace(recipe.Metadata["model"]); selected != "" {
			model = selected
		}
	}

	// 5. ラッパー経由で Lyria API を実行
	resp, err := a.aiClient.GenerateWithParts(ctx, model, parts, opts)
	if err != nil {
		return nil, fmt.Errorf("lyria generation failed (model: %s): %w", model, err)
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
