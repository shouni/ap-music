package adapters

import (
	"fmt"

	"github.com/shouni/go-prompt-kit/prompts"

	"ap-music/assets"
)

// recipeData はレシピプロンプトのテンプレートに渡すデータ構造です。
type recipeData struct {
	InputText string
}

// promptBuilder は、フォーマット済みのプロンプトを作成するためのインターフェース
type promptBuilder interface {
	Build(mode string, data any) (string, error)
}

// PromptAdapter は、さまざまなモードとデータに基づいてプロンプトを生成する役割を担います。
type PromptAdapter struct {
	recipeBuilder promptBuilder
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
		recipeBuilder: recipe,
	}, nil
}

// GenerateRecipe はレシピのMarkdownを生成します。
func (pa *PromptAdapter) GenerateRecipe(mode, content string) (string, error) {
	data := recipeData{
		InputText: content,
	}
	prompt, err := pa.recipeBuilder.Build(mode, data)
	if err != nil {
		// 52行目: 修正
		return "", fmt.Errorf("レシピテンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}
