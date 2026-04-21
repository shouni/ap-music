package adapters

import (
	"context"
	"encoding/json"
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
	clientConfig := gemini.Config{
		ProjectID:  cfg.ProjectID,
		LocationID: defaultVertexLocationID,
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
	// 1. プロンプトの構築
	// レシピの各セクションやテーマ、テンポを統合して最終的な指示文字列を作成します。
	// TODO::実際には domain.MusicRecipe 側に BuildPrompt() メソッドなどを持たせるとより綺麗です。
	fullPrompt := fmt.Sprintf(
		"%s. Mood: %s, Tempo: %d BPM. Instrumental only, no vocals.",
		recipe.Theme, recipe.Mood, recipe.Tempo,
	)

	// 2. parts の組み立て
	parts := []*genai.Part{
		{Text: fullPrompt},
	}

	// 3. 生成オプションの設定
	opts := gemini.GenerateOptions{
		ResponseMIMEType: "audio/wav",
	}

	// 4. ラッパー経由で Lyria API を実行
	resp, err := a.aiClient.GenerateWithParts(ctx, a.model, parts, opts)
	if err != nil {
		return nil, fmt.Errorf("lyria generation failed: %w", err)
	}

	// 5. 音声データの抽出
	if resp == nil || len(resp.Audios) == 0 {
		return nil, fmt.Errorf("no audio data (WAV) received from Lyria")
	}

	// 生成されたバイナリ（WAV）を返却
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
