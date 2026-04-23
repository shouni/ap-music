package adapters

import (
	"fmt"

	"github.com/shouni/go-prompt-kit/prompts"

	"ap-music/assets"
	"ap-music/internal/domain"
)

// lyricsPromptData は歌詞プロンプトのテンプレートに渡すデータ構造です。
type lyricsPromptData struct {
	InputText string
}

// recipePromptData はレシピプロンプトのテンプレートに渡すデータ構造です。
type recipePromptData struct {
	Lyrics domain.LyricsDraft
}

// promptBuilder は、フォーマット済みのプロンプトを作成するためのインターフェース
type promptBuilder interface {
	Build(mode string, data any) (string, error)
}

// PromptAdapter は、さまざまなモードとデータに基づいてプロンプトを生成する役割を担います。
type PromptAdapter struct {
	builder promptBuilder
}

// NewPromptAdapter は動的に読み込んだテンプレートを使用して Builder を構築します。
func NewPromptAdapter() (*PromptAdapter, error) {
	// 1. テンプレートの読み込み
	recipeTemplates, err := assets.LoadPrompts()
	if err != nil {
		return nil, fmt.Errorf("レシピテンプレートの読み込みに失敗: %w", err)
	}

	// 2. ビルダーの構築
	recipe, err := prompts.NewBuilder(recipeTemplates)
	if err != nil {
		return nil, fmt.Errorf("レシピビルダーの構築に失敗: %w", err)
	}

	return &PromptAdapter{
		builder: recipe,
	}, nil
}

// GenerateLyrics は歌詞生成用プロンプトを返します。
func (pa *PromptAdapter) GenerateLyrics(content string) (string, error) {
	data := lyricsPromptData{
		InputText: content,
	}
	prompt, err := pa.builder.Build(assets.ModeLyrics, data)
	if err != nil {
		return "", fmt.Errorf("歌詞テンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}

// GenerateRecipe はレシピ生成用プロンプトを返します。
func (pa *PromptAdapter) GenerateRecipe(lyrics domain.LyricsDraft) (string, error) {
	data := recipePromptData{
		Lyrics: lyrics,
	}
	prompt, err := pa.builder.Build(assets.ModeMusic, data)
	if err != nil {
		return "", fmt.Errorf("レシピテンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}
