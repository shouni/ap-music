package prompts

import "fmt"

// BuildRecipePrompt はコンテキストからレシピ生成用プロンプトを構築します。
func BuildRecipePrompt(context string) string {
	return fmt.Sprintf("Create a structured music recipe from the following context:\n\n%s", context)
}
