package adapters

import (
	"context"
	"fmt"

	"ap-music/internal/config"
	"ap-music/internal/domain"

	"github.com/shouni/go-gemini-client/gemini"
	"google.golang.org/genai"
)

// LyriaAdapter は Lyria API クライアントの実装です。
type LyriaAdapter struct {
	aiClient gemini.Generator
	model    string
}

func NewLyriaAdapter(ctx context.Context, cfg *config.Config, aiClient gemini.Generator) *LyriaAdapter {
	return &LyriaAdapter{
		aiClient: aiClient,
		model:    cfg.LyriaModel,
	}
}

// Compose は入力から音楽の構成（レシピ）を構築します。
// ※ ここでは Gemini 1.5 等を使ってプロンプトを構造化するロジックを想定
func (a *LyriaAdapter) Compose(ctx context.Context, input string) (domain.MusicRecipe, error) {
	if input == "" {
		return domain.MusicRecipe{}, fmt.Errorf("empty input")
	}

	// 実際にはここで別の LLM を使い、入力を [Verse][Chorus] 等の
	// Lyria が解釈しやすいプロンプト形式に変換する処理を挟むのが理想的です。
	// 今回は簡易的に入力をそのままテーマとして扱います。
	return domain.MusicRecipe{
		Title: "Generated Track",
		Theme: input,
		Mood:  "Dynamic",
		Tempo: 120,
		Sections: []domain.MusicSection{{
			Name:     "Full Track",
			Duration: 30, // Lyria 3 Clip なら 30秒固定
			Prompt:   input,
		}},
	}, nil
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
	if len(resp.Audios) == 0 {
		return nil, fmt.Errorf("no audio data (WAV) received from Lyria")
	}

	// 生成されたバイナリ（WAV）を返却
	return resp.Audios[0], nil
}
