package adapters

import (
	"encoding/json"
	"fmt"

	"github.com/shouni/go-prompt-kit/prompts"

	"ap-music/assets"
	"ap-music/internal/domain"
)

// lyricsPromptData は歌詞プロンプトのテンプレートに渡すデータ構造です。
type lyricsPromptData struct {
	InputText    string
	OutputSchema string
}

// recipePromptData はレシピプロンプトのテンプレートに渡すデータ構造です。
type recipePromptData struct {
	Lyrics *domain.LyricsDraft
}

// promptBuilder は、フォーマット済みのプロンプトを作成するためのインターフェース
type promptBuilder interface {
	Build(mode string, data any) (string, error)
}

// PromptAdapter は、さまざまなモードとデータに基づいてプロンプトを生成する役割を担います。
type PromptAdapter struct {
	lyrics  promptBuilder
	compose promptBuilder
}

// NewPromptAdapter は動的に読み込んだテンプレートを使用して Builder を構築します。
func NewPromptAdapter() (*PromptAdapter, error) {
	// 1. テンプレートの読み込み
	lyricsTemplates, err := assets.LoadLyricsFiles()
	if err != nil {
		return nil, fmt.Errorf("歌詞テンプレートの読み込みに失敗: %w", err)
	}

	// 2. ビルダーの構築
	lyrics, err := prompts.NewBuilder(lyricsTemplates)
	if err != nil {
		return nil, fmt.Errorf("歌詞ビルダーの構築に失敗: %w", err)
	}

	// 3. テンプレートの読み込み
	composeTemplates, err := assets.LoadComposeFiles()
	if err != nil {
		return nil, fmt.Errorf("作曲テンプレートの読み込みに失敗: %w", err)
	}

	// 4. ビルダーの構築
	compose, err := prompts.NewBuilder(composeTemplates)
	if err != nil {
		return nil, fmt.Errorf("作曲ビルダーの構築に失敗: %w", err)
	}

	return &PromptAdapter{
		lyrics:  lyrics,
		compose: compose,
	}, nil
}

// GenerateLyrics は歌詞生成用プロンプトを返します。
func (pa *PromptAdapter) GenerateLyrics(content string) (string, error) {
	draft := domain.LyricsDraft{
		Title:     "楽曲のタイトル",
		Theme:     "世界観の核",
		Hook:      "印象的なフレーズ",
		Lyrics:    "[Verse]\n...\n[Chorus]\n...",
		Keywords:  []string{"キーワード1", "キーワード2"},
		Mood:      "雰囲気",
		Narrative: "背景物語",
	}

	schemaBytes, _ := json.MarshalIndent(draft, "", "  ")
	outputSchema := string(schemaBytes)

	data := lyricsPromptData{
		InputText:    content,
		OutputSchema: outputSchema,
	}
	prompt, err := pa.lyrics.Build(assets.ModeLyrics, data)
	if err != nil {
		return "", fmt.Errorf("歌詞テンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}

// GenerateRecipe はレシピ生成用プロンプトを返します。
func (pa *PromptAdapter) GenerateRecipe(mode string, lyrics *domain.LyricsDraft) (string, error) {
	data := recipePromptData{
		Lyrics: lyrics,
	}
	prompt, err := pa.compose.Build(mode, data)
	if err != nil {
		return "", fmt.Errorf("レシピテンプレートの実行に失敗: %w", err)
	}
	return prompt, nil
}
